package matching

import (
	"context"
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"

	"github.com/example/ridellite/internal/trip/domain"
)

// ErrNoCandidate signals that no driver could be reserved.
var ErrNoCandidate = errors.New("no candidate driver available")

// RedisMatcherConfig defines tunable parameters for the matcher.
type RedisMatcherConfig struct {
	RadiusKM    float64
	TopK        int
	ReserveTTL  time.Duration
	MaxAttempts int
	Backoff     time.Duration
}

// RedisMatcher implements the MatchingEngine using Redis backed primitives.
type RedisMatcher struct {
	geo    GeoIndex
	store  ReservationStore
	logger *zap.Logger
	config RedisMatcherConfig
	tracer trace.Tracer
}

// NewRedisMatcher wires the matcher with the required collaborators.
func NewRedisMatcher(geo GeoIndex, store ReservationStore, logger *zap.Logger, cfg RedisMatcherConfig) *RedisMatcher {
	if cfg.RadiusKM <= 0 {
		cfg.RadiusKM = 5
	}
	if cfg.TopK <= 0 {
		cfg.TopK = 5
	}
	if cfg.ReserveTTL <= 0 {
		cfg.ReserveTTL = 10 * time.Second
	}
	if cfg.MaxAttempts <= 0 {
		cfg.MaxAttempts = 5
	}
	if cfg.Backoff <= 0 {
		cfg.Backoff = 50 * time.Millisecond
	}
	if logger == nil {
		logger = zap.NewNop()
	}
	return &RedisMatcher{
		geo:    geo,
		store:  store,
		logger: logger,
		config: cfg,
		tracer: otel.Tracer("trip.matching.redis"),
	}
}

// ReserveDriver implements domain.MatchingEngine.
func (m *RedisMatcher) ReserveDriver(ctx context.Context, trip domain.Trip) (*uuid.UUID, error) {
	ctx, span := m.tracer.Start(ctx, "redis_matcher.reserve")
	defer span.End()
	start := time.Now()
	resultLabel := "failure"
	logFields := m.logFields(ctx, trip)
	m.logger.Info("matching started", logFields...)

	var lastErr error
	for attempt := 1; attempt <= m.config.MaxAttempts; attempt++ {
		candidates, err := m.geo.Nearby(ctx, trip.Pickup, m.config.RadiusKM, m.config.TopK)
		if err != nil {
			lastErr = fmt.Errorf("fetch candidates: %w", err)
			break
		}
		m.logger.Debug("matching candidates", append(logFields, zap.Int("attempt", attempt), zap.Int("candidate_count", len(candidates)))...)
		for _, driverID := range candidates {
			reserved, err := m.store.TryReserve(ctx, driverID, trip.ID, m.config.ReserveTTL)
			if err != nil {
				lastErr = fmt.Errorf("reserve driver %s: %w", driverID, err)
				m.logger.Warn("reservation failed", append(logFields, zap.Error(err), zap.String("driver_id", driverID.String()))...)
				continue
			}
			label := "contended"
			if reserved {
				label = "success"
			}
			assignmentAttempts.WithLabelValues(label).Inc()
			if reserved {
				resultLabel = "success"
				matchingDuration.WithLabelValues(resultLabel).Observe(time.Since(start).Seconds())
				m.logger.Info("driver reserved", append(logFields, zap.String("driver_id", driverID.String()), zap.Int("attempt", attempt))...)
				return &driverID, nil
			}
		}
		if ctx.Err() != nil {
			lastErr = ctx.Err()
			break
		}
		sleep := m.backoffForAttempt(attempt)
		m.logger.Debug("matcher backoff", append(logFields, zap.Duration("sleep", sleep))...)
		select {
		case <-time.After(sleep):
		case <-ctx.Done():
			lastErr = ctx.Err()
			matchingDuration.WithLabelValues(resultLabel).Observe(time.Since(start).Seconds())
			return nil, lastErr
		}
	}
	matchingDuration.WithLabelValues(resultLabel).Observe(time.Since(start).Seconds())
	if lastErr != nil {
		return nil, lastErr
	}
	return nil, ErrNoCandidate
}

func (m *RedisMatcher) backoffForAttempt(attempt int) time.Duration {
	if attempt <= 0 {
		return m.config.Backoff
	}
	factor := math.Pow(2, float64(attempt-1))
	return time.Duration(float64(m.config.Backoff) * factor)
}

func (m *RedisMatcher) logFields(ctx context.Context, trip domain.Trip) []zap.Field {
	fields := []zap.Field{
		zap.String("trip_id", trip.ID.String()),
	}
	if reqID := middleware.GetReqID(ctx); reqID != "" {
		fields = append(fields, zap.String("request_id", reqID))
	}
	if span := trace.SpanFromContext(ctx); span != nil {
		sc := span.SpanContext()
		if sc.IsValid() {
			fields = append(fields, zap.String("trace_id", sc.TraceID().String()))
		}
	}
	return fields
}
