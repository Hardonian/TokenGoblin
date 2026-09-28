package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Hardonian/TokenGoblin/internal/api"
	"github.com/Hardonian/TokenGoblin/internal/config"
	"github.com/Hardonian/TokenGoblin/internal/cost"
	"github.com/Hardonian/TokenGoblin/internal/ingestion"
	"github.com/Hardonian/TokenGoblin/internal/intelligence"
	"github.com/Hardonian/TokenGoblin/internal/moat"
	"github.com/Hardonian/TokenGoblin/internal/storage"
	"github.com/redis/go-redis/v9"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
)

func initTracer() *sdktrace.TracerProvider {
	exporter, err := stdouttrace.New(stdouttrace.WithPrettyPrint())
	if err != nil {
		slog.Error("failed to initialize stdouttrace exporter", "error", err)
		return nil
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceNameKey.String("token-goblin"),
		)),
	)
	otel.SetTracerProvider(tp)
	return tp
}

func main() {
	// 1. Structured JSON Logging
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	// 2. OpenTelemetry Tracing Setup
	tp := initTracer()
	if tp != nil {
		defer func() {
			if err := tp.Shutdown(context.Background()); err != nil {
				slog.Error("failed to shutdown tracer provider", "error", err)
			}
		}()
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := config.ValidateServerEnv(); err != nil {
		slog.Error("unsafe production configuration", "error", err)
		os.Exit(1)
	}

	if !config.IsProduction() {
		slog.Warn("DEPRECATION NOTICE: Demo tenant mode (x-tenant-id header without API key) is active because TG_ENV is not 'production'. This mode is deprecated and will be removed in v1.0. Please migrate to using API keys.")
	}

	// 3. Storage Initialization (Postgres or SQLite fallback)
	var repo storage.Repository
	var err error
	if dsn := os.Getenv("TG_DB_DSN"); dsn != "" {
		repo, err = storage.OpenPostgres(ctx, dsn)
		if err != nil {
			slog.Error("postgres unavailable at startup; serving degraded routes", "error", err)
			repo = storage.NewUnavailableRepository(err)
		} else {
			slog.Info("connected to postgres database")
		}
	} else {
		repo, err = storage.OpenFromEnv(ctx)
		if err != nil {
			slog.Error("sqlite unavailable at startup; serving degraded routes", "error", err)
			repo = storage.NewUnavailableRepository(err)
		} else {
			slog.Info("connected to sqlite database")
		}
	}
	defer func() { _ = repo.Close() }()

	var redisClient *redis.Client
	if redisAddr := os.Getenv("TG_REDIS_ADDR"); redisAddr != "" {
		redisClient = redis.NewClient(&redis.Options{Addr: redisAddr})
		if err := redisClient.Ping(ctx).Err(); err != nil {
			slog.Error("redis unavailable", "error", err)
			redisClient = nil
		} else {
			slog.Info("connected to redis")
		}
	}

	registry := cost.LoadRegistry(ctx, cost.ConfigFromEnv())
	ingestionService := ingestion.NewService(repo, registry)
	ingestionService.StartWorker(ctx)

	// Start Retention Worker (30 days retention default for MVP)
	retentionWorker := ingestion.NewRetentionWorker(repo, logger, 30)
	go retentionWorker.Start(ctx)

	// Start Intelligence Workers
	if dataLakeDir := os.Getenv("TG_DATALAKE_DIR"); dataLakeDir != "" {
		dataLakeExporter := intelligence.NewDataLakeExporter(repo, dataLakeDir)
		dataLakeExporter.Start()
		defer dataLakeExporter.Stop()
	}

	rateLimiter := moat.NewRateLimiter(redisClient)
	mux := api.NewRouter(ingestionService, repo, rateLimiter)

	addr := os.Getenv("TG_ADDR")
	if addr == "" {
		port := os.Getenv("PORT")
		if port == "" {
			port = "8080"
		}
		addr = ":" + port
	}

	slog.Info("TokenGoblin execution layer starting", "addr", addr)

	server := &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       120 * time.Second,
		MaxHeaderBytes:    api.MaxHeaderBytes,
	}

	serverErrors := make(chan error, 1)
	go func() {
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErrors <- err
		}
	}()

	select {
	case err := <-serverErrors:
		slog.Error("server failed", "error", err)
		return
	case <-ctx.Done():
	}

	slog.Info("shutting down gracefully...")

	ctxShutdown, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if err := server.Shutdown(ctxShutdown); err != nil {
		slog.Error("server shutdown failed", "error", err)
		_ = server.Close()
	}

	slog.Info("server exited gracefully")
}
