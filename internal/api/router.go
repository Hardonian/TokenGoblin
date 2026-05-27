package api

import (
	"net/http"

	"github.com/Hardonian/TokenGoblin/internal/ingestion"
	"github.com/Hardonian/TokenGoblin/internal/moat"
	"github.com/Hardonian/TokenGoblin/internal/storage"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// NewRouter creates a new HTTP multiplexer with all routes registered.
func NewRouter(service ingestion.Service, repo storage.Repository, limiter *moat.RateLimiter) *http.ServeMux {
	mux := http.NewServeMux()
	handler := NewIngestionHandler(service)

	// Prometheus Metrics
	mux.Handle("/metrics", promhttp.Handler())

	// Wrap ingestion routes with auth and rate limit
	ingestHandler := AuthMiddleware(repo, RateLimitMiddleware(limiter, http.HandlerFunc(handler.HandleTokenEvent)))
	batchIngestHandler := AuthMiddleware(repo, RateLimitMiddleware(limiter, http.HandlerFunc(handler.HandleBatchTokenEvent)))

	mux.Handle("/v1/events", ingestHandler)
	mux.Handle("/v1/events/batch", batchIngestHandler)

	pricingHandler := AuthMiddleware(repo, http.HandlerFunc(handler.HandleSetPricingOverride))
	mux.Handle("/v1/pricing/overrides", pricingHandler)

	mux.Handle("/api/ingest/token-usage", ingestHandler)
	mux.Handle("/api/ingest/token-usage/batch", batchIngestHandler)

	// Wrap other routes with auth only (or both, depending on requirements, but let's apply auth to all)
	wrap := func(h http.HandlerFunc) http.Handler {
		return AuthMiddleware(repo, h)
	}

	mux.Handle("/v1/completions", wrap(handler.HandleTaskCompletion))
	mux.Handle("/v1/dashboard/overview", wrap(handler.HandleOverview))
	mux.Handle("/v1/dashboard/workers", wrap(handler.HandleWorkers))
	mux.Handle("/v1/dashboard/anomalies", wrap(handler.HandleAnomalies))

	// Catch-all 404
	mux.Handle("/v1/dashboard/events", wrap(handler.HandleRecentEvents))
	mux.Handle("/v1/dashboard/recommendations", wrap(handler.HandleRecommendations))
	mux.Handle("/v1/dashboard/export.csv", wrap(handler.HandleExportCSV))

	mux.Handle("/api/dashboard/overview", wrap(handler.HandleOverview))
	mux.Handle("/api/dashboard/workers", wrap(handler.HandleWorkers))
	mux.Handle("/api/dashboard/anomalies", wrap(handler.HandleAnomalies))
	mux.Handle("/api/dashboard/events", wrap(handler.HandleRecentEvents))
	mux.Handle("/api/dashboard/recommendations", wrap(handler.HandleRecommendations))
	mux.Handle("/api/dashboard/export.csv", wrap(handler.HandleExportCSV))

	return mux
}
