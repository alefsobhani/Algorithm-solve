package domain

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
)

// TripStatus represents the lifecycle states of a trip.
type TripStatus string

const (
	StatusRequested       TripStatus = "REQUESTED"
	StatusDriverAssigned  TripStatus = "DRIVER_ASSIGNED"
	StatusDriverAccepted  TripStatus = "DRIVER_ACCEPTED"
	StatusPickupEnRoute   TripStatus = "PICKUP_EN_ROUTE"
	StatusInProgress      TripStatus = "IN_PROGRESS"
	StatusCompleted       TripStatus = "COMPLETED"
	StatusCancelledRider  TripStatus = "CANCELLED_BY_RIDER"
	StatusCancelledDriver TripStatus = "CANCELLED_BY_DRIVER"
)

// ErrInvalidTransition is returned when a state transition is not allowed.
var ErrInvalidTransition = errors.New("invalid trip state transition")

// GeoPoint captures latitude and longitude using WGS84.
type GeoPoint struct {
	Lat float64 `json:"lat"`
	Lng float64 `json:"lng"`
}

// Trip aggregates the data required to manage a ride lifecycle.
type Trip struct {
	ID          uuid.UUID
	RiderID     uuid.UUID
	DriverID    *uuid.UUID
	Pickup      GeoPoint
	Dropoff     GeoPoint
	VehicleType string

	Status      TripStatus
	RequestedAt time.Time
	AcceptedAt  *time.Time
	StartedAt   *time.Time
	FinishedAt  *time.Time
	CancelledAt *time.Time
	CancelledBy *TripStatus
	PriceCents  int64
	Version     int64
}

// TripEventType enumerates domain events published by the service.
type TripEventType string

const (
	EventTripRequested  TripEventType = "TripRequested"
	EventDriverAssigned TripEventType = "DriverAssigned"
	EventDriverAccepted TripEventType = "DriverAccepted"
	EventTripStarted    TripEventType = "TripStarted"
	EventTripFinished   TripEventType = "TripFinished"
	EventTripCancelled  TripEventType = "TripCancelled"
)

// TripEvent captures a domain event for the outbox pattern.
type TripEvent struct {
	ID        int64
	TripID    uuid.UUID
	Type      TripEventType
	Payload   map[string]any
	CreatedAt time.Time
}

// Repository describes persistence operations required by the service.
type Repository interface {
	CreateTrip(ctx context.Context, trip Trip) (Trip, error)
	GetTripByID(ctx context.Context, id uuid.UUID) (Trip, error)
	UpdateTrip(ctx context.Context, trip Trip) (Trip, error)
	CreateTripEvent(ctx context.Context, event TripEvent) error
}

// IdempotencyRepository stores request/response pairs for idempotent APIs.
type IdempotencyRepository interface {
	GetResponse(ctx context.Context, key string) ([]byte, bool, error)
	PutResponse(ctx context.Context, key string, payload []byte) error
}

// LocationSnapshot is cached to Redis for ETA/matching decisions.
type LocationSnapshot struct {
	DriverID uuid.UUID
	Point    GeoPoint
	Speed    float64
	Accuracy float64
	Updated  time.Time
}

// MatchingEngine selects a driver for a trip request.
type MatchingEngine interface {
	ReserveDriver(ctx context.Context, trip Trip) (*uuid.UUID, error)
}

// EventPublisher publishes domain events via the outbox worker.
type EventPublisher interface {
	Publish(ctx context.Context, event TripEvent) error
}

// Clock allows deterministic tests by mocking time.
type Clock interface {
	Now() time.Time
}

// SystemClock is the production implementation of Clock.
type SystemClock struct{}

func (SystemClock) Now() time.Time { return time.Now().UTC() }
