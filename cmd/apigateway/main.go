package main

import (
	"context"
	"io"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	ratelimitmw "github.com/example/ridellite/internal/http/middleware"
	"github.com/example/ridellite/pkg/observability"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	logger := observability.SetupLogger("api-gateway")
	defer logger.Sync() //nolint:errcheck

	shutdown, err := observability.SetupTracer(ctx, "api-gateway")
	if err != nil {
		logger.Warn("tracer setup failed", zap.Error(err))
	} else {
		defer shutdown(context.Background())
	}

	tripURL := getenv("TRIP_SERVICE_URL", "http://localhost:8080")
	etaURL := getenv("ETA_SERVICE_URL", "http://localhost:8081")

	redisClient := newRedisClient(ctx, logger)
	defer func() {
		if redisClient != nil {
			_ = redisClient.Close()
		}
	}()

	limiter := ratelimitmw.NewRateLimiter(redisClient, ratelimitmw.RateConfig{
		Rate:  parseFloatEnv("RATE_READ_RPS", 50),
		Burst: parseFloatEnv("RATE_READ_BURST", 100),
	}, ratelimitmw.RateConfig{
		Rate:  parseFloatEnv("RATE_WRITE_RPS", 10),
		Burst: parseFloatEnv("RATE_WRITE_BURST", 20),
	})

	r := chi.NewRouter()
	r.Use(chimiddleware.RequestID, chimiddleware.RealIP, chimiddleware.Logger, chimiddleware.Recoverer)
	if limiter != nil {
		r.Use(limiter.Middleware)
	}
	r.Mount("/observability", observability.MetricsRouter())
	r.Get("/docs", swaggerHandler)
	r.Get("/docs/", swaggerHandler)
	r.Get("/docs/index.html", swaggerHandler)
	r.Get("/docs/openapi.yaml", openAPIHandler)
	r.Mount("/v1/trips", http.StripPrefix("/v1/trips", http.HandlerFunc(proxy(tripURL+"/v1/trips"))))
	r.Handle("/v1/eta", proxy(etaURL+"/v1/eta"))

	srv := &http.Server{Addr: ":8088", Handler: r, ReadHeaderTimeout: 5 * time.Second}
	go func() {
		logger.Info("api gateway listening", zap.String("addr", srv.Addr))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("http server", zap.Error(err))
		}
	}()

	<-ctx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = srv.Shutdown(shutdownCtx)
}

func proxy(target string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		req, err := http.NewRequestWithContext(r.Context(), r.Method, target+r.URL.Path, r.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		req.Header = r.Header.Clone()
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		defer resp.Body.Close()
		copyHeader(w.Header(), resp.Header)
		w.WriteHeader(resp.StatusCode)
		_, _ = io.Copy(w, resp.Body)
	}
}

func copyHeader(dst, src http.Header) {
	for k, v := range src {
		vv := make([]string, len(v))
		copy(vv, v)
		dst[k] = vv
	}
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
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

func newRedisClient(ctx context.Context, logger *zap.Logger) *redis.Client {
	addr := os.Getenv("REDIS_ADDR")
	if addr == "" {
		return nil
	}
	client := redis.NewClient(&redis.Options{Addr: addr})
	if err := client.Ping(ctx).Err(); err != nil {
		logger.Warn("redis ping failed", zap.Error(err))
		_ = client.Close()
		return nil
	}
	return client
}
