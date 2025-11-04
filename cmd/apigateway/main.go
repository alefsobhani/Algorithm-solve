package main

import (
	"context"
	"io"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"go.uber.org/zap"

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

	r := chi.NewRouter()
	r.Use(middleware.RequestID, middleware.RealIP, middleware.Logger, middleware.Recoverer)
	r.Mount("/observability", observability.MetricsRouter())
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
