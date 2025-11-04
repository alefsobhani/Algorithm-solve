package matching_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"

	"github.com/example/ridellite/internal/trip/domain"
	"github.com/example/ridellite/internal/trip/matching"
)

func TestRedisMatcherChoosesAvailableDriver(t *testing.T) {
	client, cleanup := newRedisClient(t)
	defer cleanup()

	ctx := context.Background()
	geo := matching.NewRedisGeoIndex(client, "driver:locs")
	store := matching.NewRedisReservationStore(client)
	matcher, err := matching.NewRedisMatcher(geo, store, matching.RedisMatcherConfig{
		CandidateLimit: 5,
		SearchRadiusKM: 10,
		MaxAttempts:    3,
		Backoff:        10 * time.Millisecond,
		ReservationTTL: time.Second,
	})
	require.NoError(t, err)

	driver1 := uuid.New()
	driver2 := uuid.New()
	// driver1 closer
	_, err = client.GeoAdd(ctx, "driver:locs:sedan", &redis.GeoLocation{Name: driver1.String(), Longitude: 51.4000, Latitude: 35.7000})
	require.NoError(t, err)
	_, err = client.GeoAdd(ctx, "driver:locs:sedan", &redis.GeoLocation{Name: driver2.String(), Longitude: 51.4050, Latitude: 35.7050})
	require.NoError(t, err)

	// reserve driver1 to simulate busy
	reserved, err := store.TryReserve(ctx, driver1, uuid.New(), time.Second)
	require.NoError(t, err)
	require.True(t, reserved)

	trip := domain.Trip{
		ID:          uuid.New(),
		Pickup:      domain.GeoPoint{Lat: 35.7001, Lng: 51.4001},
		VehicleType: "sedan",
	}

	selected, err := matcher.ReserveDriver(ctx, trip)
	require.NoError(t, err)
	require.NotNil(t, selected)
	require.Equal(t, driver2, *selected)
}

func TestRedisMatcherReleaseDriver(t *testing.T) {
	client, cleanup := newRedisClient(t)
	defer cleanup()

	ctx := context.Background()
	geo := matching.NewRedisGeoIndex(client, "driver:locs")
	store := matching.NewRedisReservationStore(client)
	matcher, err := matching.NewRedisMatcher(geo, store, matching.RedisMatcherConfig{})
	require.NoError(t, err)

	driverID := uuid.New()
	_, err = client.GeoAdd(ctx, "driver:locs:sedan", &redis.GeoLocation{Name: driverID.String(), Longitude: 0, Latitude: 0})
	require.NoError(t, err)

	trip := domain.Trip{ID: uuid.New(), Pickup: domain.GeoPoint{Lat: 0, Lng: 0}, VehicleType: "sedan"}
	selected, err := matcher.ReserveDriver(ctx, trip)
	require.NoError(t, err)
	require.NotNil(t, selected)

	key := "reserve:driver:" + driverID.String()
	exists, err := client.Exists(ctx, key).Result()
	require.NoError(t, err)
	require.Equal(t, int64(1), exists)

	require.NoError(t, matcher.ReleaseDriver(ctx, driverID))
	exists, err = client.Exists(ctx, key).Result()
	require.NoError(t, err)
	require.Equal(t, int64(0), exists)
}
