package matching_test

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"

	"github.com/example/ridellite/internal/trip/matching"
)

func newRedisClient(t *testing.T) (*redis.Client, func()) {
	t.Helper()
	mr, err := miniredis.Run()
	require.NoError(t, err)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	cleanup := func() {
		_ = client.Close()
		mr.Close()
	}
	return client, cleanup
}

func TestRedisReservationStoreReserveAndRelease(t *testing.T) {
	client, cleanup := newRedisClient(t)
	defer cleanup()

	store := matching.NewRedisReservationStore(client)
	ctx := context.Background()
	driverID := uuid.New()
	tripID := uuid.New()

	reserved, err := store.TryReserve(ctx, driverID, tripID, time.Second)
	require.NoError(t, err)
	require.True(t, reserved)

	reserved, err = store.TryReserve(ctx, driverID, uuid.New(), time.Second)
	require.NoError(t, err)
	require.False(t, reserved)

	require.NoError(t, store.Release(ctx, driverID))

	reserved, err = store.TryReserve(ctx, driverID, uuid.New(), time.Second)
	require.NoError(t, err)
	require.True(t, reserved)
}

func TestRedisReservationStoreTTLExpiry(t *testing.T) {
	client, cleanup := newRedisClient(t)
	defer cleanup()

	store := matching.NewRedisReservationStore(client)
	ctx := context.Background()
	driverID := uuid.New()

	reserved, err := store.TryReserve(ctx, driverID, uuid.New(), 100*time.Millisecond)
	require.NoError(t, err)
	require.True(t, reserved)

	time.Sleep(120 * time.Millisecond)

	reserved, err = store.TryReserve(ctx, driverID, uuid.New(), time.Second)
	require.NoError(t, err)
	require.True(t, reserved)
}
