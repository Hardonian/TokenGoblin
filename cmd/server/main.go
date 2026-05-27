package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"

	"github.com/Hardonian/TokenGoblin/internal/api"
	"github.com/Hardonian/TokenGoblin/internal/billing"
	"github.com/Hardonian/TokenGoblin/internal/cost"
	"github.com/Hardonian/TokenGoblin/internal/ingestion"
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

	ctx := context.Background()

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
	defer repo.Close()

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

	// Start Billing Syncer
	stripeSyncer := billing.NewStripeSyncer(repo, logger)
	go stripeSyncer.Start(ctx)

	rateLimiter := moat.NewRateLimiter(redisClient)
	mux := api.NewRouter(ingestionService, repo, rateLimiter)

	addr := os.Getenv("TG_ADDR")
	if addr == "" {
		addr = ":8080"
	}

	slog.Info("TokenGoblin execution layer starting", "addr", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		slog.Error("server failed to start", "error", err)
		os.Exit(1)
	}
}
