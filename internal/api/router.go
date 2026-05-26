package api

import (
	"net/http"

	"github.com/Hardonian/TokenGoblin/internal/ingestion"
	"github.com/Hardonian/TokenGoblin/internal/storage"
)

// NewRouter creates a new HTTP multiplexer with all routes registered.
func NewRouter(service ingestion.Service, store storage.Repository) *http.ServeMux {
	mux := http.NewServeMux()
	
	ingestHandler := NewIngestionHandler(service)
	summaryHandler := NewSummaryHandler(store)

	// Register routes
	mux.HandleFunc("/api/v1/ingest", ingestHandler.HandleTokenEvent)
	mux.HandleFunc("/api/v1/completions", ingestHandler.HandleTaskCompletion)
	
	mux.HandleFunc("/api/v1/summary", summaryHandler.HandleGetSummary)
	mux.HandleFunc("/api/v1/workers/", summaryHandler.HandleGetWorker) // Trailing slash allows path matching

	return mux
}
