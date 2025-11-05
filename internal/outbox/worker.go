package outbox

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

var (
	outboxPublishTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "outbox_publish_total",
		Help: "Total number of successfully published outbox messages.",
	})
	outboxFailTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "outbox_fail_total",
		Help: "Total number of outbox publish failures after exhausting retries.",
	})
	outboxLagSeconds = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "outbox_lag_seconds",
		Help: "Age of the oldest processed outbox event in seconds.",
	})
)

// WorkerConfig defines tunables for the dispatcher worker.
type WorkerConfig struct {
	PollInterval time.Duration
	BatchSize    int
	RetryMax     int
}

// Worker loads unpublished events from the database and publishes them.
type natsPublisher interface {
	PublishMsg(msg *nats.Msg) error
}

type Worker struct {
	db        *sql.DB
	publisher natsPublisher
	logger    *zap.Logger
	cfg       WorkerConfig
	tracer    trace.Tracer
}

// NewWorker constructs a dispatcher worker.
func NewWorker(db *sql.DB, conn *nats.Conn, logger *zap.Logger, cfg WorkerConfig) *Worker {
	if cfg.PollInterval <= 0 {
		cfg.PollInterval = 200 * time.Millisecond
	}
	if cfg.BatchSize <= 0 {
		cfg.BatchSize = 100
	}
	if cfg.RetryMax <= 0 {
		cfg.RetryMax = 3
	}
	if logger == nil {
		logger = zap.NewNop()
	}
	return &Worker{
		db:        db,
		publisher: conn,
		logger:    logger,
		cfg:       cfg,
		tracer:    otel.Tracer("trip.outbox.worker"),
	}
}

// Run starts the polling loop until the context is cancelled.
func (w *Worker) Run(ctx context.Context) error {
	if w.db == nil || w.publisher == nil {
		return errors.New("outbox worker requires database and NATS connection")
	}
	ticker := time.NewTicker(w.cfg.PollInterval)
	defer ticker.Stop()
	for {
		if err := w.processOnce(ctx); err != nil && !errors.Is(err, context.Canceled) {
			w.logger.Error("outbox batch failed", zap.Error(err))
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}

type record struct {
	ID        int64
	Topic     string
	Payload   []byte
	CreatedAt time.Time
}

func (w *Worker) processOnce(ctx context.Context) error {
	ctx, span := w.tracer.Start(ctx, "outbox.batch")
	defer span.End()
	records, tx, err := w.loadPending(ctx)
	if err != nil {
		return err
	}
	if len(records) == 0 {
		return tx.Commit()
	}
	ids := make([]int64, 0, len(records))
	maxLag := 0.0
	for _, rec := range records {
		if err := w.publishWithRetry(ctx, rec); err != nil {
			_ = tx.Rollback()
			return err
		}
		ids = append(ids, rec.ID)
		outboxPublishTotal.Inc()
		lag := time.Since(rec.CreatedAt).Seconds()
		if lag > maxLag {
			maxLag = lag
		}
	}
	outboxLagSeconds.Set(maxLag)
	if err := w.markPublished(ctx, tx, ids); err != nil {
		_ = tx.Rollback()
		return err
	}
	return tx.Commit()
}

func (w *Worker) loadPending(ctx context.Context) ([]record, *sql.Tx, error) {
	tx, err := w.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return nil, nil, fmt.Errorf("begin tx: %w", err)
	}
	rows, err := tx.QueryContext(ctx, `SELECT id, topic, payload, created_at FROM outbox WHERE published = false ORDER BY id LIMIT $1 FOR UPDATE SKIP LOCKED`, w.cfg.BatchSize)
	if err != nil {
		_ = tx.Rollback()
		return nil, nil, fmt.Errorf("select outbox: %w", err)
	}
	defer rows.Close()
	var records []record
	for rows.Next() {
		var rec record
		if err := rows.Scan(&rec.ID, &rec.Topic, &rec.Payload, &rec.CreatedAt); err != nil {
			_ = rows.Close()
			_ = tx.Rollback()
			return nil, nil, fmt.Errorf("scan outbox: %w", err)
		}
		records = append(records, rec)
	}
	if err := rows.Err(); err != nil {
		_ = tx.Rollback()
		return nil, nil, fmt.Errorf("iterate outbox: %w", err)
	}
	return records, tx, nil
}

func (w *Worker) markPublished(ctx context.Context, tx *sql.Tx, ids []int64) error {
	if len(ids) == 0 {
		return nil
	}
	placeholders := make([]string, len(ids))
	args := make([]any, len(ids))
	for i, id := range ids {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
		args[i] = id
	}
	query := fmt.Sprintf("UPDATE outbox SET published = true WHERE id IN (%s)", strings.Join(placeholders, ","))
	if _, err := tx.ExecContext(ctx, query, args...); err != nil {
		return fmt.Errorf("mark published: %w", err)
	}
	return nil
}

func (w *Worker) publishWithRetry(ctx context.Context, rec record) error {
	ctx, span := w.tracer.Start(ctx, "outbox.publish")
	defer span.End()
	if rec.Topic == "" {
		return errors.New("outbox record missing topic")
	}
	msg := nats.NewMsg(rec.Topic)
	msg.Data = rec.Payload
	if sc := span.SpanContext(); sc.IsValid() {
		msg.Header.Set("traceparent", fmt.Sprintf("00-%s-%s-01", sc.TraceID(), sc.SpanID()))
	}
	var attempt int
	for {
		attempt++
		err := w.publisher.PublishMsg(msg)
		if err == nil {
			return nil
		}
		w.logger.Warn("publish failed", zap.Error(err), zap.Int("attempt", attempt), zap.Int64("outbox_id", rec.ID))
		if attempt >= w.cfg.RetryMax {
			outboxFailTotal.Inc()
			return fmt.Errorf("publish outbox %d: %w", rec.ID, err)
		}
		backoff := time.Duration(attempt*attempt) * 100 * time.Millisecond
		select {
		case <-time.After(backoff):
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}
