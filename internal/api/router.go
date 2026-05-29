package api

import (
	"net/http"
	"time"

	"github.com/Hardonian/TokenGoblin/internal/ingestion"
	"github.com/Hardonian/TokenGoblin/internal/moat"
	"github.com/Hardonian/TokenGoblin/internal/storage"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// NewRouter creates a new HTTP multiplexer with all routes registered.
func NewRouter(service ingestion.Service, repo storage.Repository, limiter *moat.RateLimiter) http.Handler {
	mux := http.NewServeMux()
	handler := NewIngestionHandler(service, repo)

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
	wrapAdmin := func(h http.HandlerFunc) http.Handler {
		return AuthMiddleware(repo, RequireRole("admin")(h))
	}
	wrapAnalyst := func(h http.HandlerFunc) http.Handler {
		return AuthMiddleware(repo, RequireRole("admin", "analyst")(h))
	}

	mux.Handle("/v1/completions", wrap(handler.HandleTaskCompletion))
	mux.Handle("/v1/dashboard/overview", wrap(handler.HandleOverview))
	mux.Handle("/v1/dashboard/workers", wrap(handler.HandleWorkers))
	mux.Handle("/v1/dashboard/workers/", wrap(handler.HandleWorkerReview))
	mux.Handle("/v1/dashboard/anomalies", wrap(handler.HandleAnomalies))

	// Catch-all 404
	mux.Handle("/v1/dashboard/events", wrap(handler.HandleRecentEvents))
	mux.Handle("/v1/dashboard/output-analysis", wrap(handler.HandleOutputAnalyses))
	mux.Handle("/v1/dashboard/recommendations", wrap(handler.HandleRecommendations))
	mux.Handle("/v1/dashboard/recommendations/", wrapAnalyst(handler.HandleRecommendationState))
	mux.Handle("/v1/dashboard/export.csv", wrap(handler.HandleExportCSV))
	mux.Handle("/v1/dashboard/report.md", wrap(handler.HandleReportMarkdown))
	mux.Handle("/v1/audit/events", wrap(handler.HandleAuditEvents))
	mux.Handle("/v1/tenant/members", wrap(handler.HandleTenantMembers))

	mux.Handle("/api/dashboard/overview", wrap(handler.HandleOverview))
	mux.Handle("/api/dashboard/workers", wrap(handler.HandleWorkers))
	mux.Handle("/api/dashboard/workers/", wrap(handler.HandleWorkerReview))
	mux.Handle("/api/dashboard/anomalies", wrap(handler.HandleAnomalies))
	mux.Handle("/api/dashboard/events", wrap(handler.HandleRecentEvents))
	mux.Handle("/api/dashboard/output-analysis", wrap(handler.HandleOutputAnalyses))
	mux.Handle("/api/dashboard/recommendations", wrap(handler.HandleRecommendations))
	mux.Handle("/api/dashboard/recommendations/", wrapAnalyst(handler.HandleRecommendationState))
	mux.Handle("/api/dashboard/export.csv", wrap(handler.HandleExportCSV))
	mux.Handle("/api/dashboard/report.md", wrap(handler.HandleReportMarkdown))
	mux.Handle("/api/audit/events", wrap(handler.HandleAuditEvents))
	mux.Handle("/api/tenant/members", wrap(handler.HandleTenantMembers))

	handlerWithMiddleware := TimeoutMiddleware(15*time.Second, CORSMiddleware(LoggingMiddleware(RecoverMiddleware(mux))))
	return handlerWithMiddleware
}
