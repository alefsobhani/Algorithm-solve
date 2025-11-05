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

// SimpleMatcher implements MatchingEngine using provided dependencies.
type SimpleMatcher struct {
	index GeoIndex
	store ReservationStore
	limit int
}

// NewSimpleMatcher constructs the matcher.
func NewSimpleMatcher(index GeoIndex, store ReservationStore, limit int) *SimpleMatcher {
	if limit <= 0 {
		limit = 3
	}
	return &SimpleMatcher{index: index, store: store, limit: limit}
}

// ReserveDriver selects the first reservable driver.
func (m *SimpleMatcher) ReserveDriver(ctx context.Context, trip domain.Trip) (*uuid.UUID, error) {
	candidates, err := m.index.Nearby(ctx, trip.Pickup, 0, m.limit)
	if err != nil {
		return nil, err
	}
	for _, driverID := range candidates {
		reserved, err := m.store.TryReserve(ctx, driverID, trip.ID, time.Minute)
		if err != nil {
			return nil, err
		}
		if reserved {
			return &driverID, nil
		}
	}
	return nil, ErrNoDriver
}

// MemorySource is a trivial in-memory candidate implementation.
type MemorySource struct {
	mu              sync.RWMutex
	drivers         []uuid.UUID
	driverByVehicle map[uuid.UUID]string
}

// NewMemorySource constructs MemorySource.
func NewMemorySource() *MemorySource {
	return &MemorySource{driverByVehicle: make(map[uuid.UUID]string)}
}

// UpsertDriver stores driver coordinate hashed cell.
func (m *MemorySource) UpsertDriver(_ context.Context, driverID uuid.UUID, vehicleType string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, exists := m.driverByVehicle[driverID]; !exists {
		m.drivers = append(m.drivers, driverID)
	}
	m.driverByVehicle[driverID] = vehicleType
}

// Nearby returns deterministic order for tests irrespective of coordinates.
func (m *MemorySource) Nearby(_ context.Context, _ domain.GeoPoint, _ float64, limit int) ([]uuid.UUID, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if limit <= 0 || limit > len(m.drivers) {
		limit = len(m.drivers)
	}
	ids := make([]uuid.UUID, 0, limit)
	for _, driverID := range m.drivers {
		ids = append(ids, driverID)
		if len(ids) == limit {
			break
		}
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

// Release removes the reservation.
func (m *MemoryReservationStore) Release(_ context.Context, driverID uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.reserved, driverID)
	return nil
}
