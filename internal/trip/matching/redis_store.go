package matching

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

const reserveKeyPrefix = "reserve:driver:"

// RedisReservationStore persists driver reservations with TTL.
type RedisReservationStore struct {
	client *redis.Client
}

// NewRedisReservationStore constructs a Redis backed reservation store.
func NewRedisReservationStore(client *redis.Client) *RedisReservationStore {
	return &RedisReservationStore{client: client}
}

// TryReserve attempts to lock the driver for the provided trip id.
func (r *RedisReservationStore) TryReserve(ctx context.Context, driverID uuid.UUID, tripID uuid.UUID, ttl time.Duration) (bool, error) {
	if r == nil || r.client == nil {
		return false, fmt.Errorf("redis reservation store not configured")
	}
	key := reserveKeyPrefix + driverID.String()
	ok, err := r.client.SetNX(ctx, key, tripID.String(), ttl).Result()
	if err != nil {
		return false, fmt.Errorf("redis setnx: %w", err)
	}
	return ok, nil
}

// Release removes the reservation lock.
func (r *RedisReservationStore) Release(ctx context.Context, driverID uuid.UUID) error {
	if r == nil || r.client == nil {
		return nil
	}
	key := reserveKeyPrefix + driverID.String()
	if err := r.client.Del(ctx, key).Err(); err != nil {
		return fmt.Errorf("redis del: %w", err)
	}
	return nil
}
