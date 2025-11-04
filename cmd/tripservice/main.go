package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/nats-io/nats.go"
	"go.uber.org/zap"

	"github.com/example/ridellite/internal/trip/domain"
	"github.com/example/ridellite/internal/trip/handler"
	"github.com/example/ridellite/internal/trip/matching"
	"github.com/example/ridellite/internal/trip/repository"
	tripservice "github.com/example/ridellite/internal/trip/service"
	"github.com/example/ridellite/pkg/observability"
	"github.com/example/ridellite/pkg/outbox"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	logger := observability.SetupLogger("trip-service")
	defer logger.Sync() //nolint:errcheck

	shutdown, err := observability.SetupTracer(ctx, "trip-service")
	if err != nil {
		logger.Warn("tracer setup failed", zap.Error(err))
	} else {
		defer shutdown(context.Background())
	}

	repo := repository.NewMemoryRepository()
	idem := repository.NewMemoryIdempotencyRepo()
	locationSource := matching.NewMemorySource()
	reservation := matching.NewMemoryReservationStore()
	matcher := matching.NewSimpleMatcher(locationSource, reservation, 5)

	var natsConn *nats.Conn
	if url := os.Getenv("NATS_URL"); url != "" {
		if conn, err := nats.Connect(url); err == nil {
			natsConn = conn
			defer conn.Drain()
		} else {
			logger.Warn("nats connection failed", zap.Error(err))
		}
	}
	publisher := outbox.NewPublisher(natsConn, "trip.events")

	svc := tripservice.New(repo, publisher, matcher, domain.SystemClock{}, idem)
	tripHTTP := handler.NewHTTP(svc)

	r := chi.NewRouter()
	r.Mount("/", tripHTTP.Router())
	r.Mount("/observability", observability.MetricsRouter())

	srv := &http.Server{
		Addr:              ":8080",
		Handler:           r,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		logger.Info("trip service listening", zap.String("addr", srv.Addr))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("http server", zap.Error(err))
		}
	}()

	<-ctx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = srv.Shutdown(shutdownCtx)
}
