package matching

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/example/ridellite/internal/trip/domain"
)

// GeoIndex exposes the ability to fetch nearby driver identifiers based on a
// pickup coordinate. Implementations are expected to return drivers sorted by
// proximity (closest first) and respect the provided limit.
type GeoIndex interface {
	Nearby(ctx context.Context, p domain.GeoPoint, radiusKM float64, k int) ([]uuid.UUID, error)
}

// ReservationStore coordinates exclusive driver reservations across the fleet.
// The TTL controls how long the reservation should be considered valid in the
// underlying datastore.
type ReservationStore interface {
	TryReserve(ctx context.Context, driverID, tripID uuid.UUID, ttl time.Duration) (bool, error)
	Release(ctx context.Context, driverID uuid.UUID) error
}
