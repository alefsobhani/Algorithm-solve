package matching

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/example/ridellite/internal/trip/domain"
)

// ErrNoDriver indicates there is no driver available for reservation.
var ErrNoDriver = errors.New("no driver available")

// CandidateSource provides available drivers.
type CandidateSource interface {
	NearestDrivers(ctx context.Context, pickup domain.GeoPoint, vehicleType string, limit int) ([]uuid.UUID, error)
}

// ReservationStore ensures a driver is reserved exactly once.
type ReservationStore interface {
	TryReserve(ctx context.Context, driverID uuid.UUID, tripID uuid.UUID, ttl time.Duration) (bool, error)
	Release(ctx context.Context, driverID uuid.UUID) error
}

// SimpleMatcher implements MatchingEngine using provided dependencies.
type SimpleMatcher struct {
	source CandidateSource
	store  ReservationStore
	limit  int
	ttl    time.Duration
}

// NewSimpleMatcher constructs the matcher.
func NewSimpleMatcher(source CandidateSource, store ReservationStore, limit int) *SimpleMatcher {
	if limit <= 0 {
		limit = 3
	}
	return &SimpleMatcher{source: source, store: store, limit: limit, ttl: 10 * time.Second}
}

// ReserveDriver selects the first reservable driver.
func (m *SimpleMatcher) ReserveDriver(ctx context.Context, trip domain.Trip) (*uuid.UUID, error) {
	candidates, err := m.source.NearestDrivers(ctx, trip.Pickup, trip.VehicleType, m.limit)
	if err != nil {
		return nil, err
	}
	for _, driverID := range candidates {
		reserved, err := m.store.TryReserve(ctx, driverID, trip.ID, m.ttl)
		if err != nil {
			return nil, err
		}
		if reserved {
			return &driverID, nil
		}
	}
	return nil, ErrNoDriver
}

// ReleaseDriver frees any existing reservation for the driver.
func (m *SimpleMatcher) ReleaseDriver(ctx context.Context, driverID uuid.UUID) error {
	if m.store == nil {
		return nil
	}
	return m.store.Release(ctx, driverID)
}

// MemorySource is a trivial in-memory candidate implementation.
type MemorySource struct {
	mu              sync.RWMutex
	drivers         map[string]uuid.UUID
	driverByVehicle map[uuid.UUID]string
}

// NewMemorySource constructs MemorySource.
func NewMemorySource() *MemorySource {
	return &MemorySource{drivers: make(map[string]uuid.UUID), driverByVehicle: make(map[uuid.UUID]string)}
}

// UpsertDriver stores driver coordinate hashed cell.
func (m *MemorySource) UpsertDriver(_ context.Context, driverID uuid.UUID, vehicleType string, cell string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.drivers[cell] = driverID
	m.driverByVehicle[driverID] = vehicleType
}

// NearestDrivers returns deterministic order for tests.
func (m *MemorySource) NearestDrivers(_ context.Context, _ domain.GeoPoint, vehicleType string, limit int) ([]uuid.UUID, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var ids []uuid.UUID
	for _, driverID := range m.drivers {
		if vt, ok := m.driverByVehicle[driverID]; ok && vt == vehicleType {
			ids = append(ids, driverID)
		}
	}
	if len(ids) > limit {
		ids = ids[:limit]
	}
	return ids, nil
}

// MemoryReservationStore ensures exclusive reservation.
type MemoryReservationStore struct {
	mu       sync.Mutex
	reserved map[uuid.UUID]uuid.UUID
}

// NewMemoryReservationStore constructs MemoryReservationStore.
func NewMemoryReservationStore() *MemoryReservationStore {
	return &MemoryReservationStore{reserved: make(map[uuid.UUID]uuid.UUID)}
}

// TryReserve attempts to reserve a driver for trip.
func (m *MemoryReservationStore) TryReserve(_ context.Context, driverID uuid.UUID, tripID uuid.UUID, _ time.Duration) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, exists := m.reserved[driverID]; exists {
		return false, nil
	}
	m.reserved[driverID] = tripID
	return true, nil
}

// Release removes a driver reservation.
func (m *MemoryReservationStore) Release(_ context.Context, driverID uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.reserved, driverID)
	return nil
}
