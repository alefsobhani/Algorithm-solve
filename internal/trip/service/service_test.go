package service_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/example/ridellite/internal/trip/domain"
	"github.com/example/ridellite/internal/trip/matching"
	"github.com/example/ridellite/internal/trip/repository"
	"github.com/example/ridellite/internal/trip/service"
)

type stubPublisher struct{ events []domain.TripEvent }

type stubClock struct{ t time.Time }

type stubMatcher struct{ id *uuid.UUID }

func (s *stubPublisher) Publish(_ context.Context, event domain.TripEvent) error {
	s.events = append(s.events, event)
	return nil
}

func (s stubClock) Now() time.Time { return s.t }

func (s *stubMatcher) ReserveDriver(context.Context, domain.Trip) (*uuid.UUID, error) {
	return s.id, nil
}

func TestCreateTripAssignsDriverAndPublishesEvents(t *testing.T) {
	repo := repository.NewMemoryRepository()
	idem := repository.NewMemoryIdempotencyRepo()
	driverID := uuid.New()
	matcher := &stubMatcher{id: &driverID}
	publisher := &stubPublisher{}
	clock := stubClock{t: time.Unix(0, 0).UTC()}

	svc := service.New(repo, publisher, matcher, clock, idem)
	riderID := uuid.New()
	resp, err := svc.CreateTrip(context.Background(), "key-1", service.CreateTripRequest{
		RiderID:     riderID,
		Pickup:      domain.GeoPoint{Lat: 35.7, Lng: 51.4},
		Dropoff:     domain.GeoPoint{Lat: 35.75, Lng: 51.5},
		VehicleType: "sedan",
	})
	require.NoError(t, err)
	require.Equal(t, domain.StatusDriverAssigned, resp.Status)

	// idempotent re-call returns cached response
	cached, err := svc.CreateTrip(context.Background(), "key-1", service.CreateTripRequest{RiderID: riderID})
	require.NoError(t, err)
	require.Equal(t, resp.TripID, cached.TripID)

	require.Len(t, publisher.events, 2)
	require.Equal(t, domain.EventDriverAssigned, publisher.events[0].Type)
	require.Equal(t, domain.EventTripRequested, publisher.events[1].Type)
}

func TestCancelTripByRider(t *testing.T) {
	repo := repository.NewMemoryRepository()
	publisher := &stubPublisher{}
	clock := stubClock{t: time.Unix(0, 0).UTC()}
	svc := service.New(repo, publisher, nil, clock, repository.NewMemoryIdempotencyRepo())
	riderID := uuid.New()
	trip, err := repo.CreateTrip(context.Background(), domain.Trip{
		ID:          uuid.New(),
		RiderID:     riderID,
		Pickup:      domain.GeoPoint{Lat: 35.7, Lng: 51.4},
		Dropoff:     domain.GeoPoint{Lat: 35.75, Lng: 51.5},
		VehicleType: "sedan",
		Status:      domain.StatusDriverAssigned,
		RequestedAt: clock.Now(),
	})
	require.NoError(t, err)

	updated, err := svc.CancelTrip(context.Background(), trip.ID, domain.StatusCancelledRider)
	require.NoError(t, err)
	require.Equal(t, domain.StatusCancelledRider, updated.Status)
}

func TestMatcherNoDriver(t *testing.T) {
	matcher := matching.NewSimpleMatcher(matching.NewMemorySource(), matching.NewMemoryReservationStore(), 3)
	_, err := matcher.ReserveDriver(context.Background(), domain.Trip{Pickup: domain.GeoPoint{}, VehicleType: "sedan"})
	require.Error(t, err)
}
