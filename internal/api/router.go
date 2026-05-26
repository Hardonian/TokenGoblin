package api

import (
	"net/http"

	"github.com/Hardonian/TokenGoblin/internal/ingestion"
)

// NewRouter creates a new HTTP multiplexer with all routes registered.
func NewRouter(service ingestion.Service) *http.ServeMux {
	mux := http.NewServeMux()
	
	handler := NewIngestionHandler(service)

	// Register routes
	mux.HandleFunc("/v1/events", handler.HandleTokenEvent)
	mux.HandleFunc("/v1/completions", handler.HandleTaskCompletion)

	return mux
}
