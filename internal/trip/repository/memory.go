package repository

import (
	"context"
	"errors"
	"sync"

	"github.com/google/uuid"

	"github.com/example/ridellite/internal/trip/domain"
)

// ErrNotFound indicates missing entities.
var ErrNotFound = errors.New("trip not found")

// MemoryRepository provides an in-memory implementation suitable for tests and local demos.
type MemoryRepository struct {
	mu     sync.RWMutex
	trips  map[uuid.UUID]domain.Trip
	events []domain.TripEvent
}

// NewMemoryRepository constructs an empty memory repository.
func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{trips: make(map[uuid.UUID]domain.Trip)}
}

// CreateTrip stores the trip and returns it.
func (m *MemoryRepository) CreateTrip(_ context.Context, trip domain.Trip) (domain.Trip, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.trips[trip.ID] = trip
	return trip, nil
}

// GetTripByID retrieves a trip.
func (m *MemoryRepository) GetTripByID(_ context.Context, id uuid.UUID) (domain.Trip, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	trip, ok := m.trips[id]
	if !ok {
		return domain.Trip{}, ErrNotFound
	}
	return trip, nil
}

// UpdateTrip replaces the stored trip, performing optimistic locking on version.
func (m *MemoryRepository) UpdateTrip(_ context.Context, trip domain.Trip) (domain.Trip, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	existing, ok := m.trips[trip.ID]
	if !ok {
		return domain.Trip{}, ErrNotFound
	}
	trip.Version = existing.Version + 1
	m.trips[trip.ID] = trip
	return trip, nil
}

// CreateTripEvent appends events to an in-memory buffer.
func (m *MemoryRepository) CreateTripEvent(_ context.Context, event domain.TripEvent) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.events = append(m.events, event)
	return nil
}

// Events returns stored events (for tests).
func (m *MemoryRepository) Events() []domain.TripEvent {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return append([]domain.TripEvent(nil), m.events...)
}
