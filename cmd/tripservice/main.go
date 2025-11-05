package main

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/nats-io/nats.go"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	_ "github.com/jackc/pgx/v5/stdlib"

	outboxworker "github.com/example/ridellite/internal/outbox"
	"github.com/example/ridellite/internal/trip/domain"
	"github.com/example/ridellite/internal/trip/handler"
	"github.com/example/ridellite/internal/trip/matching"
	"github.com/example/ridellite/internal/trip/repository"
	tripservice "github.com/example/ridellite/internal/trip/service"
	"github.com/example/ridellite/pkg/observability"
	outboxpkg "github.com/example/ridellite/pkg/outbox"
)

type appConfig struct {
	HTTPAddr        string
	PostgresDSN     string
	RedisAddr       string
	NATSURL         string
	MatchRadiusKM   float64
	MatchTopK       int
	ReserveTTL      time.Duration
	MatchMaxAttempt int
	MatchBackoff    time.Duration
	OutboxPoll      time.Duration
	OutboxBatch     int
	OutboxRetry     int
}

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

	cfg := loadConfig()

	var db *sql.DB
	if cfg.PostgresDSN != "" {
		db, err = sql.Open("pgx", cfg.PostgresDSN)
		if err != nil {
			logger.Fatal("postgres connect", zap.Error(err))
		}
		db.SetMaxOpenConns(10)
		db.SetConnMaxLifetime(5 * time.Minute)
		if err := db.PingContext(ctx); err != nil {
			logger.Fatal("postgres ping", zap.Error(err))
		}
		defer db.Close()
	}

	var redisClient *redis.Client
	if cfg.RedisAddr != "" {
		redisClient = redis.NewClient(&redis.Options{Addr: cfg.RedisAddr})
		if err := redisClient.Ping(ctx).Err(); err != nil {
			logger.Fatal("redis ping", zap.Error(err))
		}
		defer redisClient.Close()
	}

	var natsConn *nats.Conn
	if cfg.NATSURL != "" {
		if conn, err := nats.Connect(cfg.NATSURL, nats.Name("tripservice")); err == nil {
			natsConn = conn
			defer conn.Drain()
		} else {
			logger.Warn("nats connection failed", zap.Error(err))
		}
	}

	matcher := buildMatcher(redisClient, logger, cfg)

	repo := repository.NewMemoryRepository()
	idem := repository.NewMemoryIdempotencyRepo()
	publisher := outboxpkg.NewPublisher(natsConn, "trip.events")

	svc := tripservice.New(repo, publisher, matcher, domain.SystemClock{}, idem)
	tripHTTP := handler.NewHTTP(svc)

	r := chi.NewRouter()
	r.Mount("/", tripHTTP.Router())
	r.Mount("/observability", observability.MetricsRouter())

	srv := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           r,
		ReadHeaderTimeout: 5 * time.Second,
	}

	if db != nil && natsConn != nil {
		worker := outboxworker.NewWorker(db, natsConn, logger.Named("outbox"), outboxworker.WorkerConfig{
			PollInterval: cfg.OutboxPoll,
			BatchSize:    cfg.OutboxBatch,
			RetryMax:     cfg.OutboxRetry,
		})
		go func() {
			if err := worker.Run(ctx); err != nil && !errors.Is(err, context.Canceled) {
				logger.Error("outbox worker stopped", zap.Error(err))
			}
		}()
	} else {
		logger.Warn("outbox worker disabled", zap.Bool("db", db != nil), zap.Bool("nats", natsConn != nil))
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

func buildMatcher(redisClient *redis.Client, logger *zap.Logger, cfg appConfig) domain.MatchingEngine {
	if redisClient == nil {
		return matching.NewSimpleMatcher(matching.NewMemorySource(), matching.NewMemoryReservationStore(), cfg.MatchTopK)
	}
	geo := matching.NewRedisGeoIndex(redisClient, "")
	store := matching.NewRedisReservationStore(redisClient, "")
	return matching.NewRedisMatcher(geo, store, logger.Named("matcher"), matching.RedisMatcherConfig{
		RadiusKM:    cfg.MatchRadiusKM,
		TopK:        cfg.MatchTopK,
		ReserveTTL:  cfg.ReserveTTL,
		MaxAttempts: cfg.MatchMaxAttempt,
		Backoff:     cfg.MatchBackoff,
	})
}

func loadConfig() appConfig {
	return appConfig{
		HTTPAddr:        getenv("HTTP_ADDR", ":8080"),
		PostgresDSN:     firstNonEmpty(os.Getenv("POSTGRES_DSN"), os.Getenv("DATABASE_URL")),
		RedisAddr:       os.Getenv("REDIS_ADDR"),
		NATSURL:         os.Getenv("NATS_URL"),
		MatchRadiusKM:   parseFloatEnv("MATCH_RADIUS_KM", 5),
		MatchTopK:       parseIntEnv("MATCH_TOPK", 5),
		ReserveTTL:      time.Duration(parseIntEnv("RESERVE_TTL_SEC", 10)) * time.Second,
		MatchMaxAttempt: parseIntEnv("MATCH_MAX_ATTEMPTS", 5),
		MatchBackoff:    time.Duration(parseIntEnv("MATCH_BACKOFF_MS", 50)) * time.Millisecond,
		OutboxPoll:      time.Duration(parseIntEnv("OUTBOX_POLL_MS", 200)) * time.Millisecond,
		OutboxBatch:     parseIntEnv("OUTBOX_BATCH", 100),
		OutboxRetry:     parseIntEnv("OUTBOX_RETRY_MAX", 3),
	}
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}

func parseIntEnv(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil {
			return parsed
		}
	}
	return fallback
}

func parseFloatEnv(key string, fallback float64) float64 {
	if v := os.Getenv(key); v != "" {
		if parsed, err := strconv.ParseFloat(v, 64); err == nil {
			return parsed
		}
	}
	return fallback
}
