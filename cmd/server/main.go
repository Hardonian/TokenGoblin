package main

import (
	"context"
	"log"
	"net/http"
	"os"

	"github.com/Hardonian/TokenGoblin/internal/api"
	"github.com/Hardonian/TokenGoblin/internal/cost"
	"github.com/Hardonian/TokenGoblin/internal/ingestion"
	"github.com/Hardonian/TokenGoblin/internal/storage"
)

func main() {
	ctx := context.Background()
	repo, err := storage.OpenFromEnv(ctx)
	if err != nil {
		log.Printf("storage unavailable at startup; serving degraded routes: %v", err)
		repo = storage.NewUnavailableRepository(err)
	}
	defer repo.Close()

	service := ingestion.NewService(repo, cost.LoadRegistry(cost.ConfigFromEnv()))
	service.StartWorker(ctx)
	mux := api.NewRouter(service)

	addr := os.Getenv("TG_ADDR")
	if addr == "" {
		addr = ":8080"
	}

	log.Printf("TokenGoblin execution layer listening on %s", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("server failed to start: %v", err)
	}
}
