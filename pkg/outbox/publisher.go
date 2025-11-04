package outbox

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/nats-io/nats.go"
	"go.opentelemetry.io/otel/trace"

	"github.com/example/ridellite/internal/trip/domain"
)

// Publisher writes trip events to a NATS subject.
type Publisher struct {
	conn    *nats.Conn
	subject string
}

// NewPublisher builds a Publisher using the provided NATS connection.
func NewPublisher(conn *nats.Conn, subject string) *Publisher {
	return &Publisher{conn: conn, subject: subject}
}

// Publish satisfies domain.EventPublisher.
func (p *Publisher) Publish(ctx context.Context, event domain.TripEvent) error {
	if p == nil || p.conn == nil {
		return nil
	}

	payload, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal event: %w", err)
	}

	return p.conn.PublishMsg(&nats.Msg{Subject: p.subject, Data: payload, Header: map[string][]string{
		"x-trace-id":   {traceIDFromContext(ctx)},
		"x-event-type": {string(event.Type)},
	}})
}

func traceIDFromContext(ctx context.Context) string {
	span := trace.SpanFromContext(ctx)
	if span == nil {
		return ""
	}
	sc := span.SpanContext()
	if !sc.IsValid() {
		return ""
	}
	return sc.TraceID().String()
}

// Worker pulls events from the database outbox and publishes them to NATS.
type Worker struct {
	Loader    func(ctx context.Context) ([]domain.TripEvent, error)
	Marker    func(ctx context.Context, events []domain.TripEvent) error
	Publisher domain.EventPublisher
}

// Run executes the worker loop once.
func (w *Worker) Run(ctx context.Context) error {
	events, err := w.Loader(ctx)
	if err != nil {
		return fmt.Errorf("load events: %w", err)
	}
	if len(events) == 0 {
		return nil
	}
	for _, evt := range events {
		if err := w.Publisher.Publish(ctx, evt); err != nil {
			return err
		}
	}
	if err := w.Marker(ctx, events); err != nil {
		return fmt.Errorf("mark published: %w", err)
	}
	return nil
}
