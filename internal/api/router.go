package api

import (
	"context"
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
	billingHandler := NewBillingHandler(repo)

	// Health & Readiness (no auth required)
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	mux.HandleFunc("/readyz", func(w http.ResponseWriter, r *http.Request) {
		// Check database connectivity
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()
		if err := repo.Ping(ctx); err != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = w.Write([]byte("not ready: " + err.Error()))
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ready"))
	})

	// Prometheus Metrics
	mux.Handle("/metrics", promhttp.Handler())

	// Wrap ingestion routes with auth and rate limit
	ingestHandler := AuthMiddleware(repo, RateLimitMiddleware(limiter, http.HandlerFunc(handler.HandleTokenEvent)))
	batchIngestHandler := AuthMiddleware(repo, RateLimitMiddleware(limiter, http.HandlerFunc(handler.HandleBatchTokenEvent)))

	mux.Handle("/v1/events", ingestHandler)
	mux.Handle("/v1/events/batch", batchIngestHandler)

	pricingHandler := AuthMiddleware(repo, http.HandlerFunc(handler.HandleSetPricingOverride))
	pricingReadHandler := AuthMiddleware(repo, http.HandlerFunc(handler.HandleGetPricing))
	seedHandler := AuthMiddleware(repo, RequireRole("admin")(http.HandlerFunc(handler.HandleSeedDemoData)))
	resetHandler := AuthMiddleware(repo, RequireRole("admin")(http.HandlerFunc(handler.HandleResetTenantData)))
	stripeWebhookHandler := http.HandlerFunc(handler.HandleStripeWebhook)
	verifiedStripeHandler := http.HandlerFunc(handler.HandleVerifiedStripeEvent)
	mux.Handle("/v1/pricing/overrides", pricingHandler)
	mux.Handle("/v1/pricing", pricingReadHandler)
	mux.Handle("/api/pricing", pricingReadHandler)
	mux.Handle("/v1/dashboard/seed", seedHandler)
	mux.Handle("/api/dashboard/seed", seedHandler)
	mux.Handle("/v1/dashboard/reset", resetHandler)
	mux.Handle("/api/dashboard/reset", resetHandler)
	mux.Handle("/api/v1/webhooks/stripe", stripeWebhookHandler)
	mux.Handle("/internal/billing/stripe-event", verifiedStripeHandler)

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
	mux.Handle("/v1/tenant/members", wrapAdmin(handler.HandleTenantMembers))

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
	mux.Handle("/api/tenant/members", wrapAdmin(handler.HandleTenantMembers))

	// Billing routes
	mux.Handle("/api/billing/checkout", wrapAdmin(http.HandlerFunc(billingHandler.HandleCreateCheckout)))
	mux.Handle("/api/billing/portal", wrapAdmin(http.HandlerFunc(billingHandler.HandleCreatePortal)))
	mux.Handle("/api/billing/status", wrap(billingHandler.HandleBillingStatus))

	// Public tenant registration (no auth)
	mux.Handle("/api/tenant/register", IPRateLimitMiddleware(limiter, http.HandlerFunc(billingHandler.HandleRegisterTenant)))

	// API Key Routes (requires auth)
	mux.Handle("/api/tenant/login", AuthMiddleware(repo, http.HandlerFunc(billingHandler.HandleTenantLogin)))
	mux.Handle("/api/tenant/webhook", wrapAdmin(http.HandlerFunc(billingHandler.HandleUpdateWebhook)))
	mux.Handle("/api/tenant/keys", wrapAdmin(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			billingHandler.HandleListAPIKeys(w, r)
		case http.MethodPost:
			billingHandler.HandleGenerateAPIKey(w, r)
		case http.MethodDelete:
			billingHandler.HandleRevokeAPIKey(w, r)
		default:
			writeMethodError(w)
		}
	})))

	// ═══════════════════════════════════════════════════
	// V2 API — Intelligence, Forecasting, Executive
	// ═══════════════════════════════════════════════════
	v2 := NewV2Handler(repo)

	// Intelligence Engine endpoints
	mux.Handle("/v2/intelligence/waste", wrap(v2.HandleWasteReport))
	mux.Handle("/v2/intelligence/prompt-graveyard", AuthMiddleware(repo, http.HandlerFunc(v2.HandlePromptGraveyard)))
	mux.Handle("/v2/intelligence/zombie-agents", AuthMiddleware(repo, http.HandlerFunc(v2.HandleZombieAgents)))
	mux.Handle("/v2/intelligence/duplicates", AuthMiddleware(repo, http.HandlerFunc(v2.HandleDuplicates)))
	mux.Handle("/v2/intelligence/cost-leaks", AuthMiddleware(repo, http.HandlerFunc(v2.HandleCostLeaks)))
	mux.Handle("/v2/intelligence/refine", AuthMiddleware(repo, http.HandlerFunc(v2.HandleRefinePrompt)))
	mux.Handle("/v2/intelligence/hallucination-map", wrap(v2.HandleHallucinationMap))
	mux.Handle("/v2/intelligence/insights", wrap(v2.HandleScholarInsights))
	mux.Handle("/api/admin/scholar/train", wrapAdmin(v2.HandleScholarTrain))

	// Forecasting Engine endpoints
	mux.Handle("/v2/forecasts/spend", wrap(v2.HandleSpendForecast))

	// Executive endpoints
	mux.Handle("/v2/executive/scorecard", wrap(v2.HandleExecutiveScorecard))

	// Analytics endpoints (v2)
	mux.Handle("/v2/analytics/models", wrap(v2.HandleModelComparison))

	handlerWithMiddleware := TimeoutMiddleware(15*time.Second, CORSMiddleware(LoggingMiddleware(RecoverMiddleware(mux))))
	return handlerWithMiddleware
}
