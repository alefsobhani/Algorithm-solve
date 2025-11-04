package matching

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"

	"github.com/example/ridellite/internal/trip/domain"
)

// GeoIndex abstracts querying spatially indexed drivers.
type GeoIndex interface {
	Nearby(ctx context.Context, vehicleType string, point domain.GeoPoint, radiusKM float64, limit int) ([]uuid.UUID, error)
}

var errInvalidGeoResult = errors.New("invalid geo search result")

// RedisGeoIndex implements GeoIndex using Redis GEO commands.
type RedisGeoIndex struct {
	client *redis.Client
	key    string
}

// NewRedisGeoIndex constructs a Redis-backed geo index.
func NewRedisGeoIndex(client *redis.Client, key string) *RedisGeoIndex {
	if key == "" {
		key = "driver:locs"
	}
	return &RedisGeoIndex{client: client, key: key}
}

// Nearby returns up to limit driver ids sorted by distance to the pickup point.
func (r *RedisGeoIndex) Nearby(ctx context.Context, vehicleType string, point domain.GeoPoint, radiusKM float64, limit int) ([]uuid.UUID, error) {
	if r == nil || r.client == nil {
		return nil, errors.New("redis geo index not configured")
	}

	key := r.key
	if vehicleType != "" {
		key = fmt.Sprintf("%s:%s", key, vehicleType)
	}

	query := &redis.GeoSearchLocationQuery{
		GeoSearchQuery: redis.GeoSearchQuery{
			Key:        key,
			Longitude:  point.Lng,
			Latitude:   point.Lat,
			Radius:     radiusKM,
			RadiusUnit: "km",
			Sort:       "ASC",
			Count:      int64(limit),
			WithDist:   true,
		},
	}

	results, err := r.client.GeoSearchLocation(ctx, query).Result()
	if err != nil {
		return nil, fmt.Errorf("redis geosearch: %w", err)
	}

	ids := make([]uuid.UUID, 0, len(results))
	for _, res := range results {
		id, err := uuid.Parse(res.Name)
		if err != nil {
			return nil, fmt.Errorf("%w: %s", errInvalidGeoResult, res.Name)
		}
		ids = append(ids, id)
	}

	return ids, nil
}
