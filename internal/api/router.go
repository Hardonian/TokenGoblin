package api

import (
	"net/http"

	"github.com/Hardonian/TokenGoblin/internal/ingestion"
)

// NewRouter creates a new HTTP multiplexer with all routes registered.
func NewRouter(service ingestion.Service) *http.ServeMux {
	mux := http.NewServeMux()
	handler := NewIngestionHandler(service)

	mux.HandleFunc("/v1/events", handler.HandleTokenEvent)
	mux.HandleFunc("/v1/completions", handler.HandleTaskCompletion)
	mux.HandleFunc("/v1/dashboard/overview", handler.HandleOverview)
	mux.HandleFunc("/v1/dashboard/workers", handler.HandleWorkers)
	mux.HandleFunc("/v1/dashboard/anomalies", handler.HandleAnomalies)
	mux.HandleFunc("/v1/dashboard/events", handler.HandleRecentEvents)
	mux.HandleFunc("/v1/dashboard/recommendations", handler.HandleRecommendations)

	mux.HandleFunc("/api/ingest/token-usage", handler.HandleTokenEvent)
	mux.HandleFunc("/api/dashboard/overview", handler.HandleOverview)
	mux.HandleFunc("/api/dashboard/workers", handler.HandleWorkers)
	mux.HandleFunc("/api/dashboard/anomalies", handler.HandleAnomalies)
	mux.HandleFunc("/api/dashboard/events", handler.HandleRecentEvents)
	mux.HandleFunc("/api/dashboard/recommendations", handler.HandleRecommendations)

	return mux
}
