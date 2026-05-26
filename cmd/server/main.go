package main

import (
	"context"
	"log"
	"net/http"

	"github.com/Hardonian/TokenGoblin/internal/api"
	"github.com/Hardonian/TokenGoblin/internal/domain"
	"github.com/Hardonian/TokenGoblin/internal/ingestion"
)

// mockService is a temporary stub so the server can build and run.
type mockService struct{}

func (m *mockService) ProcessTokenEvent(ctx context.Context, event domain.TokenEvent) error { return nil }
func (m *mockService) ProcessTaskCompletion(ctx context.Context, completion domain.TaskCompletion) error { return nil }

func main() {
	// TODO: Initialize real dependencies (config, logger, storage, service)
	// For now, Codex will wire this up to the real database implementation.

	// Temporary mock service just to satisfy the router interface
	service := &mockService{}

	// Initialize the HTTP router
	mux := api.NewRouter(service)

	log.Println("TokenGoblin Ingestion Gateway starting on :8080...")
	if err := http.ListenAndServe(":8080", mux); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}
