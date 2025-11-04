package domain

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
)

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

var ErrInvalidTransition = errors.New("invalid trip state transition")
var ErrDriverMismatch = errors.New("driver not assigned to trip")

type CancellationReason string

const (
	CancellationReasonRider  CancellationReason = "rider"
	CancellationReasonDriver CancellationReason = "driver"
)

func (r CancellationReason) CancellationStatus() TripStatus {
	switch r {
	case CancellationReasonDriver:
		return StatusCancelledDriver
	case CancellationReasonRider:
		return StatusCancelledRider
	default:
		return StatusCancelledRider
	}
}

var allowedTransitions = map[TripStatus][]TripStatus{
	StatusRequested:      {StatusDriverAssigned, StatusCancelledRider},
	StatusDriverAssigned: {StatusDriverAccepted, StatusPickupEnRoute, StatusCancelledRider, StatusCancelledDriver},
	StatusDriverAccepted: {StatusPickupEnRoute, StatusInProgress, StatusCancelledRider, StatusCancelledDriver},
	StatusPickupEnRoute:  {StatusInProgress, StatusCancelledRider, StatusCancelledDriver},
	StatusInProgress:     {StatusCompleted, StatusCancelledDriver},
}

func (s TripStatus) CanTransitionTo(next TripStatus) bool {
	if s == next {
		return true
	}
	allowed := allowedTransitions[s]
	for _, candidate := range allowed {
		if candidate == next {
			return true
		}
	}
	return false
}

type GeoPoint struct {
	Lat float64 `json:"lat"`
	Lng float64 `json:"lng"`
}

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
	CancelledBy *CancellationReason
	PriceCents  int64
	Version     int64
}

type TripEventType string

const (
	EventTripRequested  TripEventType = "TripRequested"
	EventDriverAssigned TripEventType = "DriverAssigned"
	EventDriverAccepted TripEventType = "DriverAccepted"
	EventTripStarted    TripEventType = "TripStarted"
	EventTripFinished   TripEventType = "TripFinished"
	EventTripCancelled  TripEventType = "TripCancelled"
)

type TripEvent struct {
	ID        int64
	TripID    uuid.UUID
	Type      TripEventType
	Payload   map[string]any
	CreatedAt time.Time
}

type Repository interface {
	CreateTrip(ctx context.Context, trip Trip) (Trip, error)
	GetTripByID(ctx context.Context, id uuid.UUID) (Trip, error)
	UpdateTrip(ctx context.Context, trip Trip) (Trip, error)
	CreateTripEvent(ctx context.Context, event TripEvent) error
}

type IdempotencyRepository interface {
	GetResponse(ctx context.Context, key string) ([]byte, bool, error)
	PutResponse(ctx context.Context, key string, payload []byte) error
}

type LocationSnapshot struct {
	DriverID uuid.UUID
	Point    GeoPoint
	Speed    float64
	Accuracy float64
	Updated  time.Time
}

type MatchingEngine interface {
	ReserveDriver(ctx context.Context, trip Trip) (*uuid.UUID, error)
	ReleaseDriver(ctx context.Context, driverID uuid.UUID) error
}

type EventPublisher interface {
	Publish(ctx context.Context, event TripEvent) error
}

type Clock interface {
	Now() time.Time
}

type SystemClock struct{}

func (SystemClock) Now() time.Time { return time.Now().UTC() }
