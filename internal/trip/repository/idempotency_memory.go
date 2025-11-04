package repository

import (
	"context"
	"sync"
)

// MemoryIdempotencyRepo stores responses keyed by idempotency key.
type MemoryIdempotencyRepo struct {
	mu        sync.RWMutex
	responses map[string][]byte
}

// NewMemoryIdempotencyRepo constructs repository.
func NewMemoryIdempotencyRepo() *MemoryIdempotencyRepo {
	return &MemoryIdempotencyRepo{responses: make(map[string][]byte)}
}

// GetResponse retrieves cached response.
func (m *MemoryIdempotencyRepo) GetResponse(_ context.Context, key string) ([]byte, bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	value, ok := m.responses[key]
	return append([]byte(nil), value...), ok, nil
}

// PutResponse stores response payload.
func (m *MemoryIdempotencyRepo) PutResponse(_ context.Context, key string, payload []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.responses[key] = append([]byte(nil), payload...)
	return nil
}
