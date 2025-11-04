package service

import (
	"context"
	"math"
	"time"

	"github.com/google/uuid"

	"github.com/example/ridellite/internal/trip/domain"
)

// Repository exposes read methods for location snapshots.
type Repository interface {
	Snapshot(ctx context.Context, driverID uuid.UUID) (domain.LocationSnapshot, bool)
	All() []domain.LocationSnapshot
}

// Service calculates ETAs using haversine distance and average speeds.
type Service struct {
	repo Repository
}

// New creates an ETA service.
func New(repo Repository) *Service {
	return &Service{repo: repo}
}

// EstimateDriverETA returns the fastest driver estimate from available snapshots.
func (s *Service) EstimateDriverETA(ctx context.Context, pickup domain.GeoPoint) (time.Duration, *uuid.UUID) {
	const avgSpeed = 30.0 // km/h
	const meterPerSecond = avgSpeed * 1000.0 / 3600.0
	snapshots := s.repo.All()
	var bestDuration time.Duration
	var bestDriver *uuid.UUID
	for _, snap := range snapshots {
		dist := haversine(snap.Point, pickup)
		sec := dist / meterPerSecond
		duration := time.Duration(sec) * time.Second
		if bestDriver == nil || duration < bestDuration {
			snapshotDriver := snap.DriverID
			bestDriver = &snapshotDriver
			bestDuration = duration
		}
	}
	return bestDuration, bestDriver
}

// EstimateTripETA approximates total trip time using distance and average speed.
func (s *Service) EstimateTripETA(_ context.Context, pickup, dropoff domain.GeoPoint) time.Duration {
	const avgSpeed = 35.0 // km/h
	const meterPerSecond = avgSpeed * 1000.0 / 3600.0
	dist := haversine(pickup, dropoff)
	sec := dist / meterPerSecond
	return time.Duration(sec) * time.Second
}

func haversine(a, b domain.GeoPoint) float64 {
	const earthRadius = 6371000.0
	lat1 := toRadians(a.Lat)
	lat2 := toRadians(b.Lat)
	dlat := toRadians(b.Lat - a.Lat)
	dlon := toRadians(b.Lng - a.Lng)

	sinDlat := math.Sin(dlat / 2)
	sinDlon := math.Sin(dlon / 2)
	aa := sinDlat*sinDlat + math.Cos(lat1)*math.Cos(lat2)*sinDlon*sinDlon
	c := 2 * math.Atan2(math.Sqrt(aa), math.Sqrt(1-aa))
	return earthRadius * c
}

func toRadians(deg float64) float64 {
	return deg * math.Pi / 180.0
}
