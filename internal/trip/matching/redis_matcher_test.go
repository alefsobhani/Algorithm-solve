package matching

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
	rediscontainer "github.com/testcontainers/testcontainers-go/modules/redis"
	"github.com/testcontainers/testcontainers-go/wait"
	"go.uber.org/zap"

	"github.com/example/ridellite/internal/trip/domain"
)

func TestRedisReservationStoreContention(t *testing.T) {
	ctx := context.Background()
	client := startRedis(t, ctx)
	store := NewRedisReservationStore(client, "")
	driverID := uuid.New()
	tripA := uuid.New()
	tripB := uuid.New()

	reserved, err := store.TryReserve(ctx, driverID, tripA, 2*time.Second)
	require.NoError(t, err)
	require.True(t, reserved)

	done := make(chan bool)
	go func() {
		ok, err := store.TryReserve(ctx, driverID, tripB, 2*time.Second)
		require.NoError(t, err)
		done <- ok
	}()

	select {
	case ok := <-done:
		require.False(t, ok, "concurrent reservation should fail")
	case <-time.After(2 * time.Second):
		t.Fatal("reservation attempt timed out")
	}

	time.Sleep(2100 * time.Millisecond)

	reuse, err := store.TryReserve(ctx, driverID, tripB, 2*time.Second)
	require.NoError(t, err)
	require.True(t, reuse, "reservation should succeed after TTL expiry")
}

func TestRedisMatcherSkipsBusyDriver(t *testing.T) {
	ctx := context.Background()
	client := startRedis(t, ctx)
	geo := NewRedisGeoIndex(client, "")
	store := NewRedisReservationStore(client, "")
	logger := zap.NewNop()

	driverBusy := uuid.New()
	driverFree := uuid.New()
	require.NoError(t, geo.UpsertLocation(ctx, driverBusy, domain.GeoPoint{Lat: 37.7749, Lng: -122.4194}))
	require.NoError(t, geo.UpsertLocation(ctx, driverFree, domain.GeoPoint{Lat: 37.7750, Lng: -122.4195}))

	busyTrip := uuid.New()
	reserved, err := store.TryReserve(ctx, driverBusy, busyTrip, 5*time.Second)
	require.NoError(t, err)
	require.True(t, reserved)

	matcher := NewRedisMatcher(geo, store, logger, RedisMatcherConfig{
		RadiusKM:    1,
		TopK:        5,
		ReserveTTL:  5 * time.Second,
		MaxAttempts: 3,
		Backoff:     10 * time.Millisecond,
	})

	trip := domain.Trip{ID: uuid.New(), Pickup: domain.GeoPoint{Lat: 37.7749, Lng: -122.4194}}
	selected, err := matcher.ReserveDriver(ctx, trip)
	require.NoError(t, err)
	require.NotNil(t, selected)
	require.Equal(t, driverFree, *selected)
}

func startRedis(t *testing.T, ctx context.Context) *redis.Client {
	container, err := rediscontainer.Run(ctx, "redis:7", rediscontainer.WithWaitStrategy(wait.ForLog("Ready to accept connections")))
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, container.Terminate(ctx))
	})
	endpoint, err := container.ConnectionString(ctx)
	require.NoError(t, err)
	addr := strings.TrimPrefix(endpoint, "redis://")
	client := redis.NewClient(&redis.Options{Addr: addr})
	require.NoError(t, client.Ping(ctx).Err())
	t.Cleanup(func() {
		_ = client.Close()
	})
	return client
}
