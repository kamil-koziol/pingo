package main

import (
	"context"
	"database/sql"
	_ "embed"
	"errors"
	"flag"
	"fmt"
	"log"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	pb "github.com/kamil-koziol/pingo/gen/go/pingo/v1"
	"github.com/kamil-koziol/pingo/internal/alerting"
	"github.com/kamil-koziol/pingo/internal/configuration"
	"github.com/kamil-koziol/pingo/internal/contextx"
	"github.com/kamil-koziol/pingo/internal/db"
	"github.com/kamil-koziol/pingo/internal/handler"
	"github.com/kamil-koziol/pingo/internal/middleware"
	"github.com/kamil-koziol/pingo/internal/monitoring"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	_ "modernc.org/sqlite"
)

//go:embed schema.sql
var ddl string

const (
	DEFAULT_CONFIG_PATH = "config.yml"
	DEFAULT_DB_PATH     = "pingo.db"
)

func createDB(ctx context.Context, dbPath string) (*sql.DB, error) {
	conn, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("unable to open db: %w", err)
	}

	// create tables
	if _, err := conn.ExecContext(ctx, ddl); err != nil {
		log.Fatalf("unable to migrate db: %v", err)
	}

	if _, err := conn.ExecContext(ctx, "PRAGMA journal_mode=WAL;"); err != nil {
		return nil, fmt.Errorf("unable to pragma: %w", err)
	}

	if _, err := conn.ExecContext(ctx, "PRAGMA synchronous=NORMAL;"); err != nil {
		return nil, fmt.Errorf("unable to pragma: %w", err)
	}

	if _, err := conn.ExecContext(ctx, "PRAGMA busy_timeout=5000;"); err != nil {
		return nil, fmt.Errorf("unable to pragma: %w", err)
	}

	return conn, nil
}

func main() {
	err := run()
	if err != nil {
		slog.Error("unable to start", "err", err)
		os.Exit(1)
	}
}

func run() error {
	configPtr := flag.String("config", DEFAULT_CONFIG_PATH, "Path to config file")
	dbPtr := flag.String("db", DEFAULT_DB_PATH, "Path to db")
	flag.Parse()

	ctx := context.Background()

	f, err := os.Open(*configPtr)
	if err != nil {
		return fmt.Errorf("unable to read config: %v", err)
	}

	config, err := configuration.Parse(f)
	if err != nil {
		log.Fatalf("unable to parse config: %v", err)
	}

	var alerter alerting.Alerter

	alerters := make([]alerting.Alerter, len(config.Alerts))
	for i, alert := range config.Alerts {
		alerter, err := alert.Build()
		if err != nil {
			return fmt.Errorf("unable to build alerter: %w", err)
		}
		alerters[i] = alerter
	}

	alerter = alerting.NewMultiAlerter(alerters...)

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)
	alerter = alerting.NewLoggingAlerter(logger, alerter)

	conn, err := createDB(ctx, *dbPtr)
	if err != nil {
		log.Fatalf("unable to create db: %v", err)
	}

	q := db.New(conn)

	// Ensure services
	for _, check := range config.Checks {
		service, err := q.UpsertService(ctx, db.UpsertServiceParams{
			Name:            check.Name,
			Url:             check.URL.String(),
			IntervalSeconds: int64(check.Interval.Seconds()),
			ExpectedStatus:  int64(check.ExpectedStatus),
		})
		if err != nil {
			log.Fatalf("unable to upsert service: %v", err)
		}

		m := monitoring.NewMonitor(service, q, alerter)
		monitorCtx := contextx.WithLogger(ctx, logger)
		go m.Run(monitorCtx)
	}

	grpcAddr := fmt.Sprintf(":%d", config.API.GRPC.Port)
	httpAddr := fmt.Sprintf(":%d", config.API.HTTP.Port)

	var grpcServer *grpc.Server
	var httpServer *http.Server

	// ---- gRPC server ----
	if config.API.GRPC.Enabled {
		lis, err := net.Listen("tcp", grpcAddr)
		if err != nil {
			return fmt.Errorf("failed to listen on gRPC port %s: %w", grpcAddr, err)
		}

		serviceHandler := handler.NewServiceHandler(conn, q)
		pingHandler := handler.NewPingHandler(conn, q)

		grpcServer = grpc.NewServer(
			grpc.UnaryInterceptor(middleware.LoggingInterceptor(logger)),
		)
		pb.RegisterServiceServiceServer(grpcServer, serviceHandler)
		pb.RegisterPingServiceServer(grpcServer, pingHandler)

		go func() {
			logger.Info("gRPC listening on", "addr", grpcAddr)
			if err := grpcServer.Serve(lis); err != nil && !errors.Is(err, grpc.ErrServerStopped) {
				logger.Error("gRPC server error", "err", err)
			}
		}()
	}

	// ---- HTTP JSON Gateway ----
	if config.API.HTTP.Enabled {
		mux := runtime.NewServeMux()
		opts := []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}
		grpcHP := "localhost" + grpcAddr

		if err := pb.RegisterServiceServiceHandlerFromEndpoint(ctx, mux, grpcHP, opts); err != nil {
			return fmt.Errorf("failed to register service handler: %w", err)
		}

		if err := pb.RegisterPingServiceHandlerFromEndpoint(ctx, mux, grpcHP, opts); err != nil {
			return fmt.Errorf("failed to register ping handler: %w", err)
		}

		httpServer = &http.Server{
			Addr:    httpAddr,
			Handler: mux,
		}

		go func() {
			logger.Info("HTTP JSON listening on", "addr", httpAddr)
			if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
				logger.Error("HTTP gateway server error", "err", err)
			}
		}()
	}

	// Block until shutdown received
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down servers...")

	// Graceful shutdown
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if httpServer != nil {
		if err := httpServer.Shutdown(shutdownCtx); err != nil {
			logger.Error("HTTP gateway shutdown error", "err", err)
		}
	}

	if grpcServer != nil {
		grpcServer.GracefulStop()
	}

	logger.Info("Server stopped cleanly")
	return nil
}
