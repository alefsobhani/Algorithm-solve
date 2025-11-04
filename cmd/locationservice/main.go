package main

import (
	"context"
	"net"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
	"google.golang.org/grpc"

	"github.com/example/ridellite/internal/eta/handler"
	etasvc "github.com/example/ridellite/internal/eta/service"
	"github.com/example/ridellite/internal/location"
	"github.com/example/ridellite/pkg/observability"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	logger := observability.SetupLogger("location-service")
	defer logger.Sync() //nolint:errcheck

	shutdown, err := observability.SetupTracer(ctx, "location-service")
	if err != nil {
		logger.Warn("tracer setup failed", zap.Error(err))
	} else {
		defer shutdown(context.Background())
	}

	observer := location.NewStreamObserver()
	etaSvc := etasvc.New(observer)

	go runREST(logger, etaSvc)
	go runGRPC(logger, observer)

	<-ctx.Done()
	logger.Info("shutdown signal received")
}

func runREST(logger *zap.Logger, etaSvc *etasvc.Service) {
	r := chi.NewRouter()
	r.Mount("/", handler.New(etaSvc).Router())
	r.Mount("/observability", observability.MetricsRouter())

	srv := &http.Server{Addr: ":8081", Handler: r, ReadHeaderTimeout: 5 * time.Second}
	logger.Info("eta REST listening", zap.String("addr", srv.Addr))
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		logger.Fatal("eta rest server", zap.Error(err))
	}
}

func runGRPC(logger *zap.Logger, observer *location.StreamObserver) {
	lis, err := net.Listen("tcp", ":9090")
	if err != nil {
		logger.Fatal("listen grpc", zap.Error(err))
	}

	srv := grpc.NewServer()
	RegisterLocationServer(srv, NewServer(observer))
	logger.Info("location grpc listening", zap.String("addr", lis.Addr().String()))
	if err := srv.Serve(lis); err != nil {
		logger.Fatal("grpc serve", zap.Error(err))
	}
}
