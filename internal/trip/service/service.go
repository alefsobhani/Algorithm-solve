package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"github.com/example/ridellite/internal/trip/domain"
)

type Service struct {
	repo       domain.Repository
	events     domain.EventPublisher
	matcher    domain.MatchingEngine
	clock      domain.Clock
	idempotent domain.IdempotencyRepository
}

func New(repo domain.Repository, events domain.EventPublisher, matcher domain.MatchingEngine, clock domain.Clock, idem domain.IdempotencyRepository) *Service {
	return &Service{repo: repo, events: events, matcher: matcher, clock: clock, idempotent: idem}
}

type CreateTripRequest struct {
	RiderID     uuid.UUID
	Pickup      domain.GeoPoint
	Dropoff     domain.GeoPoint
	VehicleType string
}

type CreateTripResponse struct {
	TripID uuid.UUID         `json:"trip_id"`
	Status domain.TripStatus `json:"status"`
}

func (s *Service) CreateTrip(ctx context.Context, key string, req CreateTripRequest) (CreateTripResponse, error) {
	if key != "" && s.idempotent != nil {
		if cached, ok, err := s.idempotent.GetResponse(ctx, key); err != nil {
			return CreateTripResponse{}, fmt.Errorf("get idempotent response: %w", err)
		} else if ok {
			return decodeCreateTripResponse(cached)
		}
	}

	now := s.clock.Now()
	trip := domain.Trip{
		ID:          uuid.New(),
		RiderID:     req.RiderID,
		Pickup:      req.Pickup,
		Dropoff:     req.Dropoff,
		VehicleType: req.VehicleType,
		Status:      domain.StatusRequested,
		RequestedAt: now,
		Version:     1,
	}

	created, err := s.repo.CreateTrip(ctx, trip)
	if err != nil {
		return CreateTripResponse{}, fmt.Errorf("create trip: %w", err)
	}

	s.recordEvent(ctx, domain.TripEvent{ // TripRequested
		TripID: created.ID,
		Type:   domain.EventTripRequested,
		Payload: map[string]any{
			"rider_id": created.RiderID.String(),
		},
	})

	if s.matcher != nil {
		if driverID, matchErr := s.matcher.ReserveDriver(ctx, created); matchErr == nil && driverID != nil {
			created.DriverID = driverID
			created.Status = domain.StatusDriverAssigned
			updated, updateErr := s.repo.UpdateTrip(ctx, created)
			if updateErr != nil {
				_ = s.matcher.ReleaseDriver(ctx, *driverID)
				return CreateTripResponse{}, fmt.Errorf("assign driver: %w", updateErr)
			}
			created = updated
			s.recordEvent(ctx, domain.TripEvent{
				TripID: created.ID,
				Type:   domain.EventDriverAssigned,
				Payload: map[string]any{
					"driver_id": driverID.String(),
				},
			})
		}
	}

	resp := CreateTripResponse{TripID: created.ID, Status: created.Status}
	if key != "" && s.idempotent != nil {
		if err := s.idempotent.PutResponse(ctx, key, encodeCreateTripResponse(resp)); err != nil {
			return CreateTripResponse{}, fmt.Errorf("store idempotent response: %w", err)
		}
	}
	return resp, nil
}

func (s *Service) GetTrip(ctx context.Context, id uuid.UUID) (domain.Trip, error) {
	return s.repo.GetTripByID(ctx, id)
}

func (s *Service) AcceptTrip(ctx context.Context, tripID, driverID uuid.UUID) (domain.Trip, error) {
	trip, err := s.repo.GetTripByID(ctx, tripID)
	if err != nil {
		return domain.Trip{}, err
	}

	if trip.DriverID == nil || *trip.DriverID != driverID {
		return domain.Trip{}, domain.ErrDriverMismatch
	}

	if trip.Status == domain.StatusDriverAccepted {
		return trip, nil
	}

	if !trip.Status.CanTransitionTo(domain.StatusDriverAccepted) {
		return domain.Trip{}, domain.ErrInvalidTransition
	}

	now := s.clock.Now()
	trip.Status = domain.StatusDriverAccepted
	trip.AcceptedAt = &now

	updated, err := s.repo.UpdateTrip(ctx, trip)
	if err != nil {
		return domain.Trip{}, err
	}

	s.recordEvent(ctx, domain.TripEvent{
		TripID: updated.ID,
		Type:   domain.EventDriverAccepted,
		Payload: map[string]any{
			"driver_id": driverID.String(),
		},
	})

	return updated, nil
}

func (s *Service) CancelTrip(ctx context.Context, tripID uuid.UUID, reason domain.CancellationReason) (domain.Trip, error) {
	trip, err := s.repo.GetTripByID(ctx, tripID)
	if err != nil {
		return domain.Trip{}, err
	}

	newStatus := reason.CancellationStatus()
	if trip.Status == newStatus {
		return trip, nil
	}

	if !trip.Status.CanTransitionTo(newStatus) {
		return domain.Trip{}, domain.ErrInvalidTransition
	}

	now := s.clock.Now()
	trip.Status = newStatus
	trip.CancelledAt = &now
	trip.CancelledBy = &reason

	updated, err := s.repo.UpdateTrip(ctx, trip)
	if err != nil {
		return domain.Trip{}, err
	}

	s.recordEvent(ctx, domain.TripEvent{
		TripID: updated.ID,
		Type:   domain.EventTripCancelled,
		Payload: map[string]any{
			"reason": string(reason),
		},
	})

	if trip.DriverID != nil && s.matcher != nil {
		_ = s.matcher.ReleaseDriver(ctx, *trip.DriverID)
	}

	return updated, nil
}

func (s *Service) StartTrip(ctx context.Context, tripID uuid.UUID) (domain.Trip, error) {
	trip, err := s.repo.GetTripByID(ctx, tripID)
	if err != nil {
		return domain.Trip{}, err
	}

	if !trip.Status.CanTransitionTo(domain.StatusInProgress) {
		return domain.Trip{}, domain.ErrInvalidTransition
	}

	now := s.clock.Now()
	trip.Status = domain.StatusInProgress
	trip.StartedAt = &now

	updated, err := s.repo.UpdateTrip(ctx, trip)
	if err != nil {
		return domain.Trip{}, err
	}

	s.recordEvent(ctx, domain.TripEvent{TripID: updated.ID, Type: domain.EventTripStarted})

	return updated, nil
}

func (s *Service) CompleteTrip(ctx context.Context, tripID uuid.UUID, priceCents int64) (domain.Trip, error) {
	trip, err := s.repo.GetTripByID(ctx, tripID)
	if err != nil {
		return domain.Trip{}, err
	}

	if trip.Status == domain.StatusCompleted {
		return trip, nil
	}

	if !trip.Status.CanTransitionTo(domain.StatusCompleted) {
		return domain.Trip{}, domain.ErrInvalidTransition
	}

	now := s.clock.Now()
	trip.Status = domain.StatusCompleted
	trip.FinishedAt = &now
	trip.PriceCents = priceCents

	updated, err := s.repo.UpdateTrip(ctx, trip)
	if err != nil {
		return domain.Trip{}, err
	}

	s.recordEvent(ctx, domain.TripEvent{
		TripID: updated.ID,
		Type:   domain.EventTripFinished,
		Payload: map[string]any{
			"price_cents": priceCents,
		},
	})

	if trip.DriverID != nil && s.matcher != nil {
		_ = s.matcher.ReleaseDriver(ctx, *trip.DriverID)
	}

	return updated, nil
}

func encodeCreateTripResponse(resp CreateTripResponse) []byte {
	payload, err := json.Marshal(resp)
	if err != nil {
		return nil
	}
	return payload
}

func decodeCreateTripResponse(b []byte) (CreateTripResponse, error) {
	var resp CreateTripResponse
	if len(b) == 0 {
		return resp, errors.New("empty idempotent payload")
	}
	if err := json.Unmarshal(b, &resp); err != nil {
		return CreateTripResponse{}, err
	}
	if resp.TripID == uuid.Nil {
		return CreateTripResponse{}, errors.New("missing trip id")
	}
	return resp, nil
}

func (s *Service) recordEvent(ctx context.Context, event domain.TripEvent) {
	if event.CreatedAt.IsZero() {
		event.CreatedAt = s.clock.Now()
	}
	if s.repo != nil {
		_ = s.repo.CreateTripEvent(ctx, event)
	}
	if s.events != nil {
		_ = s.events.Publish(ctx, event)
	}
}
