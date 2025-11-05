package matching

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"

	"github.com/example/ridellite/internal/trip/domain"
)

const defaultGeoKey = "driver:locs"

// RedisGeoIndex persists driver locations in Redis using the GEO* family of
// commands.
type RedisGeoIndex struct {
	client redis.GeoCmdable
	key    string
}

// NewRedisGeoIndex constructs a GEO-backed index. The provided client must be
// safe for concurrent use (go-redis clients satisfy this requirement).
func NewRedisGeoIndex(client redis.GeoCmdable, key string) *RedisGeoIndex {
	if key == "" {
		key = defaultGeoKey
	}
	return &RedisGeoIndex{client: client, key: key}
}

// Nearby queries Redis for the closest drivers using GEOSEARCH.
func (g *RedisGeoIndex) Nearby(ctx context.Context, p domain.GeoPoint, radiusKM float64, k int) ([]uuid.UUID, error) {
	if k <= 0 {
		return nil, nil
	}
	if radiusKM <= 0 {
		radiusKM = 5 // sensible default radius in kilometres
	}
	locations, err := g.client.GeoSearchLocation(ctx, g.key, &redis.GeoSearchQuery{
		Longitude:  p.Lng,
		Latitude:   p.Lat,
		Radius:     radiusKM,
		RadiusUnit: "km",
		Count:      k,
		Sort:       "ASC",
		WithDist:   true,
	}).Result()
	if err != nil {
		return nil, fmt.Errorf("redis geosearch: %w", err)
	}
	results := make([]uuid.UUID, 0, len(locations))
	for _, loc := range locations {
		id, err := uuid.Parse(loc.Name)
		if err != nil {
			continue
		}
		results = append(results, id)
	}
	return results, nil
}

// UpsertLocation stores or updates a driver's position inside the GEO index.
func (g *RedisGeoIndex) UpsertLocation(ctx context.Context, driverID uuid.UUID, p domain.GeoPoint) error {
	return g.client.GeoAdd(ctx, g.key, &redis.GeoLocation{
		Longitude: p.Lng,
		Latitude:  p.Lat,
		Name:      driverID.String(),
	}).Err()
}
