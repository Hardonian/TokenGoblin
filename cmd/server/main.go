package main

import (
	"log"
	"net/http"

	"github.com/Hardonian/TokenGoblin/internal/api"
	"github.com/Hardonian/TokenGoblin/internal/ingestion"
	"github.com/Hardonian/TokenGoblin/internal/storage"
)

func main() {
	// Initialize in-memory storage for MVP
	store := storage.NewMemoryStore()
	store.SeedPricing()

	// Initialize the ingestion engine
	engine := ingestion.NewEngine(store, store)

	// Initialize the HTTP router
	mux := api.NewRouter(engine, store)

	log.Println("TokenGoblin Ingestion Gateway starting on :8080...")
	if err := http.ListenAndServe(":8080", mux); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}
