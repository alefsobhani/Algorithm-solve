package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"github.com/example/ridellite/internal/trip/domain"
)

// Service coordinates trip operations between handlers and repositories.
type Service struct {
	repo       domain.Repository
	events     domain.EventPublisher
	matcher    domain.MatchingEngine
	clock      domain.Clock
	idempotent domain.IdempotencyRepository
}

// New constructs a Service with the required collaborators.
func New(repo domain.Repository, events domain.EventPublisher, matcher domain.MatchingEngine, clock domain.Clock, idem domain.IdempotencyRepository) *Service {
	return &Service{repo: repo, events: events, matcher: matcher, clock: clock, idempotent: idem}
}

// CreateTripRequest contains the request payload for creating a trip.
type CreateTripRequest struct {
	RiderID     uuid.UUID
	Pickup      domain.GeoPoint
	Dropoff     domain.GeoPoint
	VehicleType string
}

// CreateTripResponse returns the created trip identifier and status.
type CreateTripResponse struct {
	TripID uuid.UUID         `json:"trip_id"`
	Status domain.TripStatus `json:"status"`
}

// CreateTrip handles a new trip creation request ensuring idempotency and driver matching.
func (s *Service) CreateTrip(ctx context.Context, key string, req CreateTripRequest) (CreateTripResponse, error) {
	if key != "" && s.idempotent != nil {
		if cached, ok, err := s.idempotent.GetResponse(ctx, key); err == nil && ok {
			return decodeCreateTripResponse(cached)
		}
	}

	trip := domain.Trip{
		ID:          uuid.New(),
		RiderID:     req.RiderID,
		Pickup:      req.Pickup,
		Dropoff:     req.Dropoff,
		VehicleType: req.VehicleType,
		Status:      domain.StatusRequested,
		RequestedAt: s.clock.Now(),
		Version:     1,
	}

	created, err := s.repo.CreateTrip(ctx, trip)
	if err != nil {
		return CreateTripResponse{}, fmt.Errorf("create trip: %w", err)
	}

	if s.matcher != nil {
		if driverID, err := s.matcher.ReserveDriver(ctx, created); err == nil && driverID != nil {
			created.DriverID = driverID
			created.Status = domain.StatusDriverAssigned
			if _, err = s.repo.UpdateTrip(ctx, created); err != nil {
				return CreateTripResponse{}, fmt.Errorf("assign driver: %w", err)
			}
			_ = s.events.Publish(ctx, domain.TripEvent{
				TripID:  created.ID,
				Type:    domain.EventDriverAssigned,
				Payload: map[string]any{"driver_id": driverID.String()},
			})
		}
	}

	event := domain.TripEvent{
		TripID:  created.ID,
		Type:    domain.EventTripRequested,
		Payload: map[string]any{"rider_id": created.RiderID.String()},
	}
	_ = s.events.Publish(ctx, event)

	resp := CreateTripResponse{TripID: created.ID, Status: created.Status}
	if key != "" && s.idempotent != nil {
		_ = s.idempotent.PutResponse(ctx, key, encodeCreateTripResponse(resp))
	}

	return resp, nil
}

// GetTrip retrieves a trip by identifier.
func (s *Service) GetTrip(ctx context.Context, id uuid.UUID) (domain.Trip, error) {
	return s.repo.GetTripByID(ctx, id)
}

// AcceptTrip allows a driver to accept an assigned trip.
func (s *Service) AcceptTrip(ctx context.Context, tripID, driverID uuid.UUID) (domain.Trip, error) {
	trip, err := s.repo.GetTripByID(ctx, tripID)
	if err != nil {
		return domain.Trip{}, err
	}

	if trip.DriverID == nil || *trip.DriverID != driverID {
		return domain.Trip{}, errors.New("driver not assigned to trip")
	}

	switch trip.Status {
	case domain.StatusDriverAssigned:
		now := s.clock.Now()
		trip.Status = domain.StatusDriverAccepted
		trip.AcceptedAt = &now
	default:
		return domain.Trip{}, domain.ErrInvalidTransition
	}

	updated, err := s.repo.UpdateTrip(ctx, trip)
	if err != nil {
		return domain.Trip{}, err
	}

	_ = s.events.Publish(ctx, domain.TripEvent{
		TripID:  updated.ID,
		Type:    domain.EventDriverAccepted,
		Payload: map[string]any{"driver_id": driverID.String()},
	})

	return updated, nil
}

// CancelTrip handles rider initiated cancellations prior to start.
func (s *Service) CancelTrip(ctx context.Context, tripID uuid.UUID, actor domain.TripStatus) (domain.Trip, error) {
	trip, err := s.repo.GetTripByID(ctx, tripID)
	if err != nil {
		return domain.Trip{}, err
	}

	switch trip.Status {
	case domain.StatusRequested, domain.StatusDriverAssigned, domain.StatusDriverAccepted, domain.StatusPickupEnRoute:
		now := s.clock.Now()
		trip.Status = actor
		trip.CancelledAt = &now
		trip.CancelledBy = &actor
	default:
		return domain.Trip{}, domain.ErrInvalidTransition
	}

	updated, err := s.repo.UpdateTrip(ctx, trip)
	if err != nil {
		return domain.Trip{}, err
	}

	_ = s.events.Publish(ctx, domain.TripEvent{
		TripID:  updated.ID,
		Type:    domain.EventTripCancelled,
		Payload: map[string]any{"status": string(actor)},
	})

	return updated, nil
}

// StartTrip transitions to IN_PROGRESS when the driver begins the ride.
func (s *Service) StartTrip(ctx context.Context, tripID uuid.UUID) (domain.Trip, error) {
	trip, err := s.repo.GetTripByID(ctx, tripID)
	if err != nil {
		return domain.Trip{}, err
	}

	if trip.Status != domain.StatusPickupEnRoute && trip.Status != domain.StatusDriverAccepted {
		return domain.Trip{}, domain.ErrInvalidTransition
	}

	now := s.clock.Now()
	trip.Status = domain.StatusInProgress
	trip.StartedAt = &now

	updated, err := s.repo.UpdateTrip(ctx, trip)
	if err != nil {
		return domain.Trip{}, err
	}

	_ = s.events.Publish(ctx, domain.TripEvent{
		TripID: updated.ID,
		Type:   domain.EventTripStarted,
	})

	return updated, nil
}

// CompleteTrip marks the trip as completed.
func (s *Service) CompleteTrip(ctx context.Context, tripID uuid.UUID, priceCents int64) (domain.Trip, error) {
	trip, err := s.repo.GetTripByID(ctx, tripID)
	if err != nil {
		return domain.Trip{}, err
	}

	if trip.Status != domain.StatusInProgress {
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

	_ = s.events.Publish(ctx, domain.TripEvent{
		TripID:  updated.ID,
		Type:    domain.EventTripFinished,
		Payload: map[string]any{"price_cents": priceCents},
	})

	return updated, nil
}

func encodeCreateTripResponse(resp CreateTripResponse) []byte {
	return []byte(resp.TripID.String() + ":" + string(resp.Status))
}

func decodeCreateTripResponse(b []byte) (CreateTripResponse, error) {
	parts := string(b)
	var resp CreateTripResponse
	if len(parts) == 0 {
		return resp, errors.New("empty payload")
	}
	// simple split without strings to avoid allocations.
	var idStr string
	var statusStr string
	for i := range parts {
		if parts[i] == ':' {
			idStr = parts[:i]
			if i+1 < len(parts) {
				statusStr = parts[i+1:]
			}
			break
		}
	}
	id, err := uuid.Parse(idStr)
	if err != nil {
		return resp, err
	}
	resp.TripID = id
	resp.Status = domain.TripStatus(statusStr)
	return resp, nil
}
