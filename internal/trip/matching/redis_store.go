package matching

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

const defaultReservationPrefix = "reserve:driver:"

// RedisReservationStore coordinates driver reservations by relying on Redis
// SETNX semantics. A TTL is attached to every reservation to avoid stale locks.
type RedisReservationStore struct {
	client     redis.Cmdable
	keyPrefix  string
	clockSkew  time.Duration
	bufferTime time.Duration
}

// NewRedisReservationStore constructs the reservation helper.
func NewRedisReservationStore(client redis.Cmdable, prefix string) *RedisReservationStore {
	if prefix == "" {
		prefix = defaultReservationPrefix
	}
	return &RedisReservationStore{client: client, keyPrefix: prefix}
}

// TryReserve attempts to acquire a reservation using SET NX EX.
func (r *RedisReservationStore) TryReserve(ctx context.Context, driverID, tripID uuid.UUID, ttl time.Duration) (bool, error) {
	if ttl <= 0 {
		ttl = 10 * time.Second
	}
	key := r.keyPrefix + driverID.String()
	ok, err := r.client.SetNX(ctx, key, tripID.String(), ttl).Result()
	if err != nil {
		return false, fmt.Errorf("redis setnx: %w", err)
	}
	return ok, nil
}

// Release removes the reservation key.
func (r *RedisReservationStore) Release(ctx context.Context, driverID uuid.UUID) error {
	key := r.keyPrefix + driverID.String()
	if err := r.client.Del(ctx, key).Err(); err != nil {
		return fmt.Errorf("redis del: %w", err)
	}
	return nil
}
