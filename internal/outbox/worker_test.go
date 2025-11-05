package outbox

import (
	"context"
	"database/sql"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go/modules/nats"
	postgrescontainer "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
	"go.uber.org/zap"

	_ "github.com/jackc/pgx/v5/stdlib"
)

func TestWorkerPublishesOutboxEntries(t *testing.T) {
	ctx := context.Background()
	pg := startPostgres(t, ctx)
	conn := openDB(t, ctx, pg)
	prepareOutboxTable(t, ctx, conn)
	insertOutbox(t, ctx, conn, "trip.events", []byte(`{"id":1}`))

	natsContainer := startNATS(t, ctx)
	natsURL, err := natsContainer.ConnectionString(ctx)
	require.NoError(t, err)
	nc, err := nats.Connect(natsURL)
	require.NoError(t, err)
	t.Cleanup(func() { _ = nc.Drain() })

	msgCh := make(chan *nats.Msg, 1)
	_, err = nc.Subscribe("trip.events", func(msg *nats.Msg) {
		msgCh <- msg
	})
	require.NoError(t, err)

	worker := NewWorker(conn, nc, zap.NewNop(), WorkerConfig{PollInterval: 100 * time.Millisecond, BatchSize: 10, RetryMax: 5})
	ctxWorker, cancel := context.WithCancel(ctx)
	defer cancel()
	go func() {
		_ = worker.Run(ctxWorker)
	}()

	select {
	case <-time.After(10 * time.Second):
		t.Fatal("expected outbox message")
	case msg := <-msgCh:
		require.Equal(t, []byte(`{"id":1}`), msg.Data)
	}

	assertPublished(t, ctx, conn, 1)
	cancel()
}

func TestWorkerRetriesOnFailure(t *testing.T) {
	ctx := context.Background()
	pg := startPostgres(t, ctx)
	conn := openDB(t, ctx, pg)
	prepareOutboxTable(t, ctx, conn)
	insertOutbox(t, ctx, conn, "trip.events", []byte(`{"retry":true}`))

	natsContainer := startNATS(t, ctx)
	natsURL, err := natsContainer.ConnectionString(ctx)
	require.NoError(t, err)
	nc, err := nats.Connect(natsURL)
	require.NoError(t, err)
	t.Cleanup(func() { _ = nc.Drain() })

	msgCh := make(chan *nats.Msg, 1)
	_, err = nc.Subscribe("trip.events", func(msg *nats.Msg) {
		msgCh <- msg
	})
	require.NoError(t, err)

	worker := NewWorker(conn, nc, zap.NewNop(), WorkerConfig{PollInterval: 100 * time.Millisecond, BatchSize: 5, RetryMax: 5})
	worker.publisher = &flakyPublisher{base: nc, failFor: 3}

	ctxWorker, cancel := context.WithCancel(ctx)
	defer cancel()
	go func() {
		_ = worker.Run(ctxWorker)
	}()

	select {
	case <-time.After(15 * time.Second):
		t.Fatal("expected retry publish")
	case msg := <-msgCh:
		require.Equal(t, []byte(`{"retry":true}`), msg.Data)
	}

	assertPublished(t, ctx, conn, 1)
	cancel()
}

type flakyPublisher struct {
	base    *nats.Conn
	failFor int32
}

func (f *flakyPublisher) PublishMsg(msg *nats.Msg) error {
	if atomic.LoadInt32(&f.failFor) > 0 {
		atomic.AddInt32(&f.failFor, -1)
		return errors.New("simulated nats outage")
	}
	return f.base.PublishMsg(msg)
}

func startPostgres(t *testing.T, ctx context.Context) *postgrescontainer.PostgresContainer {
	pg, err := postgrescontainer.Run(ctx, "postgres:16", postgrescontainer.WithDatabase("ridellite"), postgrescontainer.WithUsername("postgres"), postgrescontainer.WithPassword("postgres"), postgrescontainer.WithWaitStrategy(wait.ForLog("database system is ready to accept connections")))
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, pg.Terminate(ctx))
	})
	return pg
}

func openDB(t *testing.T, ctx context.Context, pg *postgrescontainer.PostgresContainer) *sql.DB {
	dsn, err := pg.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)
	db, err := sql.Open("pgx", dsn)
	require.NoError(t, err)
	require.NoError(t, db.PingContext(ctx))
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func prepareOutboxTable(t *testing.T, ctx context.Context, db *sql.DB) {
	ddl := `CREATE TABLE IF NOT EXISTS outbox (
id SERIAL PRIMARY KEY,
topic TEXT,
payload BYTEA,
published BOOLEAN DEFAULT FALSE,
created_at TIMESTAMPTZ DEFAULT now()
)`
	_, err := db.ExecContext(ctx, ddl)
	require.NoError(t, err)
}

func insertOutbox(t *testing.T, ctx context.Context, db *sql.DB, topic string, payload []byte) {
	_, err := db.ExecContext(ctx, `INSERT INTO outbox (topic, payload, published) VALUES ($1, $2, false)`, topic, payload)
	require.NoError(t, err)
}

func assertPublished(t *testing.T, ctx context.Context, db *sql.DB, id int64) {
	var published bool
	row := db.QueryRowContext(ctx, `SELECT published FROM outbox WHERE id = $1`, id)
	require.NoError(t, row.Scan(&published))
	require.True(t, published)
}

func startNATS(t *testing.T, ctx context.Context) *nats.Container {
	container, err := nats.Run(ctx, "nats:2", nats.WithJetStream(true))
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, container.Terminate(ctx))
	})
	return container
}
