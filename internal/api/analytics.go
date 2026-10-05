package api

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/Hardonian/TokenGoblin/internal/storage/clickhouse"
)

// AnalyticsStore is the read-only analytics surface (the ClickHouse mirror).
// Kept as a small interface so handlers can be tested with fakes and so the
// API layer depends on query shapes, not on a database driver.
type AnalyticsStore interface {
	GetCostByTenant(ctx context.Context, tenantID string, start, end time.Time) (clickhouse.CostSummary, error)
	GetCostByModel(ctx context.Context, tenantID string, start, end time.Time) ([]clickhouse.ModelCostSummary, error)
	GetCostByFeature(ctx context.Context, tenantID string, start, end time.Time) ([]clickhouse.FeatureCostSummary, error)
	GetZombieAgents(ctx context.Context, tenantID string, threshold float64, window time.Duration) ([]clickhouse.ZombieAgentRecord, error)
	GetAnomalies(ctx context.Context, tenantID string, start, end time.Time) ([]clickhouse.AnomalyRecord, error)
}

// AnalyticsHandler serves cost-intelligence endpoints from the mirror. The
// tenant ALWAYS comes from the authenticated request context — never from a
// query parameter — so cross-tenant reads are structurally impossible here.
type AnalyticsHandler struct {
	Analytics AnalyticsStore
}

func NewAnalyticsHandler(analytics AnalyticsStore) *AnalyticsHandler {
	return &AnalyticsHandler{Analytics: analytics}
}

// windowFromRequest resolves (start, end) with a default 30-day window.
func windowFromRequest(r *http.Request) (time.Time, time.Time) {
	end := time.Now().UTC()
	start := end.Add(-30 * 24 * time.Hour)
	if raw := r.URL.Query().Get("start"); raw != "" {
		if parsed, err := time.Parse(time.RFC3339, raw); err == nil {
			start = parsed.UTC()
		}
	}
	if raw := r.URL.Query().Get("end"); raw != "" {
		if parsed, err := time.Parse(time.RFC3339, raw); err == nil {
			end = parsed.UTC()
		}
	}
	return start, end
}

func (h *AnalyticsHandler) unavailable(w http.ResponseWriter) {
	writeJSON(w, http.StatusServiceUnavailable, Envelope{
		OK:     false,
		Status: "error",
		Error: issue("analytics_unavailable",
			"Analytics are served from the ClickHouse telemetry mirror, which is not configured on this deployment. Set TG_CLICKHOUSE_ADDR to enable it."),
	})
}

func (h *AnalyticsHandler) queryError(w http.ResponseWriter) {
	writeJSON(w, http.StatusServiceUnavailable, Envelope{
		OK:     false,
		Status: "error",
		Error:  issue("analytics_query_failed", "The analytics store could not answer this query."),
	})
}

// HandleCostSummary — GET /v1/analytics/cost
func (h *AnalyticsHandler) HandleCostSummary(w http.ResponseWriter, r *http.Request) {
	if h.Analytics == nil {
		h.unavailable(w)
		return
	}
	tenantID := getTenantID(r)
	start, end := windowFromRequest(r)
	summary, err := h.Analytics.GetCostByTenant(r.Context(), tenantID, start, end)
	if err != nil {
		h.queryError(w)
		return
	}
	writeJSON(w, http.StatusOK, Envelope{OK: true, Status: "success", Data: summary})
}

// HandleCostByModel — GET /v1/analytics/cost/by-model
func (h *AnalyticsHandler) HandleCostByModel(w http.ResponseWriter, r *http.Request) {
	if h.Analytics == nil {
		h.unavailable(w)
		return
	}
	tenantID := getTenantID(r)
	start, end := windowFromRequest(r)
	models, err := h.Analytics.GetCostByModel(r.Context(), tenantID, start, end)
	if err != nil {
		h.queryError(w)
		return
	}
	if models == nil {
		models = []clickhouse.ModelCostSummary{}
	}
	writeJSON(w, http.StatusOK, Envelope{OK: true, Status: "success", Data: models})
}

// HandleCostByFeature — GET /v1/analytics/cost/by-feature
func (h *AnalyticsHandler) HandleCostByFeature(w http.ResponseWriter, r *http.Request) {
	if h.Analytics == nil {
		h.unavailable(w)
		return
	}
	tenantID := getTenantID(r)
	start, end := windowFromRequest(r)
	features, err := h.Analytics.GetCostByFeature(r.Context(), tenantID, start, end)
	if err != nil {
		h.queryError(w)
		return
	}
	if features == nil {
		features = []clickhouse.FeatureCostSummary{}
	}
	writeJSON(w, http.StatusOK, Envelope{OK: true, Status: "success", Data: features})
}

// HandleZombieAgents — GET /v1/analytics/zombie-agents?threshold=0.2&window_hours=720
func (h *AnalyticsHandler) HandleZombieAgents(w http.ResponseWriter, r *http.Request) {
	if h.Analytics == nil {
		h.unavailable(w)
		return
	}
	tenantID := getTenantID(r)
	threshold := 0.2
	if raw := r.URL.Query().Get("threshold"); raw != "" {
		if parsed, err := strconv.ParseFloat(raw, 64); err == nil && parsed >= 0 && parsed <= 1 {
			threshold = parsed
		}
	}
	window := 30 * 24 * time.Hour
	if raw := r.URL.Query().Get("window_hours"); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil && parsed > 0 {
			window = time.Duration(parsed) * time.Hour
		}
	}
	agents, err := h.Analytics.GetZombieAgents(r.Context(), tenantID, threshold, window)
	if err != nil {
		h.queryError(w)
		return
	}
	if agents == nil {
		agents = []clickhouse.ZombieAgentRecord{}
	}
	writeJSON(w, http.StatusOK, Envelope{OK: true, Status: "success", Data: agents})
}

// HandleAnomalies — GET /v1/analytics/anomalies (mirrored anomaly signals)
func (h *AnalyticsHandler) HandleAnomalies(w http.ResponseWriter, r *http.Request) {
	if h.Analytics == nil {
		h.unavailable(w)
		return
	}
	tenantID := getTenantID(r)
	start, end := windowFromRequest(r)
	anomalies, err := h.Analytics.GetAnomalies(r.Context(), tenantID, start, end)
	if err != nil {
		h.queryError(w)
		return
	}
	if anomalies == nil {
		anomalies = []clickhouse.AnomalyRecord{}
	}
	writeJSON(w, http.StatusOK, Envelope{OK: true, Status: "success", Data: anomalies})
}