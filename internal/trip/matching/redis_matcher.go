package matching

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/example/ridellite/internal/trip/domain"
)

// RedisMatcherConfig configures the Redis backed matcher behaviour.
type RedisMatcherConfig struct {
	CandidateLimit int
	SearchRadiusKM float64
	MaxAttempts    int
	Backoff        time.Duration
	ReservationTTL time.Duration
}

// RedisMatcher implements domain.MatchingEngine using Redis data structures.
type RedisMatcher struct {
	geo          GeoIndex
	reservations ReservationStore
	cfg          RedisMatcherConfig
}

// NewRedisMatcher builds a matcher from redis-backed collaborators.
func NewRedisMatcher(geo GeoIndex, reservations ReservationStore, cfg RedisMatcherConfig) (*RedisMatcher, error) {
	if geo == nil {
		return nil, errors.New("geo index is required")
	}
	if reservations == nil {
		return nil, errors.New("reservation store is required")
	}
	if cfg.CandidateLimit <= 0 {
		cfg.CandidateLimit = 5
	}
	if cfg.SearchRadiusKM <= 0 {
		cfg.SearchRadiusKM = 5
	}
	if cfg.MaxAttempts <= 0 {
		cfg.MaxAttempts = 3
	}
	if cfg.Backoff <= 0 {
		cfg.Backoff = 100 * time.Millisecond
	}
	if cfg.ReservationTTL <= 0 {
		cfg.ReservationTTL = 10 * time.Second
	}
	return &RedisMatcher{geo: geo, reservations: reservations, cfg: cfg}, nil
}

// ReserveDriver finds the closest reservable driver within the configured search radius.
func (r *RedisMatcher) ReserveDriver(ctx context.Context, trip domain.Trip) (*uuid.UUID, error) {
	var lastErr error
	for attempt := 0; attempt < r.cfg.MaxAttempts; attempt++ {
		candidates, err := r.geo.Nearby(ctx, trip.VehicleType, trip.Pickup, r.cfg.SearchRadiusKM, r.cfg.CandidateLimit)
		if err != nil {
			return nil, fmt.Errorf("geo nearby: %w", err)
		}
		for _, driverID := range candidates {
			reserved, err := r.reservations.TryReserve(ctx, driverID, trip.ID, r.cfg.ReservationTTL)
			if err != nil {
				lastErr = err
				continue
			}
			if reserved {
				return &driverID, nil
			}
		}
		if attempt < r.cfg.MaxAttempts-1 {
			backoff := r.cfg.Backoff << attempt
			select {
			case <-time.After(backoff):
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		}
	}
	if lastErr != nil {
		return nil, lastErr
	}
	return nil, ErrNoDriver
}

// ReleaseDriver releases the driver reservation.
func (r *RedisMatcher) ReleaseDriver(ctx context.Context, driverID uuid.UUID) error {
	return r.reservations.Release(ctx, driverID)
}
