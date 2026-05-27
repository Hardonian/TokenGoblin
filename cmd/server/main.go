package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"

	"github.com/Hardonian/TokenGoblin/internal/api"
	"github.com/Hardonian/TokenGoblin/internal/cost"
	"github.com/Hardonian/TokenGoblin/internal/ingestion"
	"github.com/Hardonian/TokenGoblin/internal/storage"
)

func main() {
	// 1. Structured JSON Logging
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	ctx := context.Background()

	// 2. Storage Initialization (Postgres or SQLite fallback)
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

	service := ingestion.NewService(repo, cost.LoadRegistry(cost.ConfigFromEnv()))
	service.StartWorker(ctx)
	mux := api.NewRouter(service)

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
