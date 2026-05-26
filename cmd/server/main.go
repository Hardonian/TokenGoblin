package main

import (
	"context"
	"log"
	"net/http"

	"github.com/Hardonian/TokenGoblin/internal/api"
	"github.com/Hardonian/TokenGoblin/internal/ingestion"
	"github.com/Hardonian/TokenGoblin/internal/storage"
)

func main() {
	// Initialize SQLite storage for MVP
	store, err := storage.OpenSQLite(context.Background(), ":memory:")
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer store.Close()

	// Initialize the ingestion engine
	engine := ingestion.NewEngine(store)

	// Initialize the HTTP router
	mux := api.NewRouter(engine, store)

	log.Println("TokenGoblin Ingestion Gateway starting on :8080...")
	if err := http.ListenAndServe(":8080", mux); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}
