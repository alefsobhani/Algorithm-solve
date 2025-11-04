package location

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/example/ridellite/internal/trip/domain"
)

// StreamObserver stores latest driver location snapshots.
type StreamObserver struct {
	mu        sync.RWMutex
	snapshots map[uuid.UUID]domain.LocationSnapshot
}

// NewStreamObserver constructs the observer.
func NewStreamObserver() *StreamObserver {
	return &StreamObserver{snapshots: make(map[uuid.UUID]domain.LocationSnapshot)}
}

// Update stores snapshot data.
func (o *StreamObserver) Update(_ context.Context, driverID uuid.UUID, point domain.GeoPoint, speed, accuracy float64) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.snapshots[driverID] = domain.LocationSnapshot{
		DriverID: driverID,
		Point:    point,
		Speed:    speed,
		Accuracy: accuracy,
		Updated:  time.Now().UTC(),
	}
}

// Snapshot returns the stored snapshot.
func (o *StreamObserver) Snapshot(_ context.Context, driverID uuid.UUID) (domain.LocationSnapshot, bool) {
	o.mu.RLock()
	defer o.mu.RUnlock()
	snap, ok := o.snapshots[driverID]
	return snap, ok
}

// All returns all snapshots.
func (o *StreamObserver) All() []domain.LocationSnapshot {
	o.mu.RLock()
	defer o.mu.RUnlock()
	res := make([]domain.LocationSnapshot, 0, len(o.snapshots))
	for _, snap := range o.snapshots {
		res = append(res, snap)
	}
	return res
}
