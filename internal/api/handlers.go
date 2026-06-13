package api

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/csv"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/stripe/stripe-go/v76"
	"github.com/stripe/stripe-go/v76/webhook"

	"github.com/Hardonian/TokenGoblin/internal/billing"
	"github.com/Hardonian/TokenGoblin/internal/demo"
	"github.com/Hardonian/TokenGoblin/internal/domain"
	"github.com/Hardonian/TokenGoblin/internal/ingestion"
	"github.com/Hardonian/TokenGoblin/internal/storage"
)

type IngestionHandler struct {
	Service ingestion.Service
	Repo    storage.Repository
}

type Envelope struct {
	OK       bool           `json:"ok"`
	Status   string         `json:"status"`
	Data     interface{}    `json:"data,omitempty"`
	Warnings []domain.Issue `json:"warnings,omitempty"`
	Degraded []domain.Issue `json:"degraded,omitempty"`
	Error    *domain.Issue  `json:"error,omitempty"`
}

func NewIngestionHandler(service ingestion.Service, repo storage.Repository) *IngestionHandler {
	return &IngestionHandler{Service: service, Repo: repo}
}

func (h *IngestionHandler) HandleTokenEvent(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, Envelope{
			OK:     false,
			Status: "error",
			Error:  issue("method_not_allowed", "Use POST for token usage ingestion."),
		})
		return
	}

	tenantID, ok := tenantFromRequest(w, r)
	if !ok {
		return
	}

	var event domain.TokenEvent
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&event); err != nil {
		writeJSON(w, http.StatusBadRequest, Envelope{
			OK:     false,
			Status: "error",
			Error:  issue("invalid_json", "Request body must be a valid token usage event JSON object."),
			Degraded: []domain.Issue{{
				Code:    "invalid_json",
				Message: "JSON decoding failed.",
				Field:   "body",
			}},
		})
		return
	}

	result, err := h.Service.IngestTokenEvent(r.Context(), tenantID, event)
	if err != nil {
		var quotaErr ingestion.QuotaExceededError
		if errors.As(err, &quotaErr) {
			writeJSON(w, http.StatusPaymentRequired, Envelope{
				OK:     false,
				Status: "error",
				Error:  issue("quota_exceeded", err.Error()),
				Degraded: []domain.Issue{{
					Code:    "quota_exceeded",
					Message: err.Error(),
				}},
			})
			return
		}
		writeServiceError(w, err, false)
		return
	}

	status := "success"
	if len(result.Degraded) > 0 {
		status = "degraded"
	}
	writeJSON(w, http.StatusAccepted, Envelope{
		OK:       true,
		Status:   status,
		Data:     result,
		Warnings: result.Warnings,
		Degraded: result.Degraded,
	})
}

func (h *IngestionHandler) HandleBatchTokenEvent(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeMethodError(w)
		return
	}

	tenantID, ok := tenantFromRequest(w, r)
	if !ok {
		return
	}

	var events []domain.TokenEvent
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&events); err != nil {
		writeJSON(w, http.StatusBadRequest, Envelope{
			OK:     false,
			Status: "error",
			Error:  issue("invalid_json", "Request body must be a valid JSON array of token usage events."),
			Degraded: []domain.Issue{{
				Code:    "invalid_json",
				Message: "JSON decoding failed.",
				Field:   "body",
			}},
		})
		return
	}

	results, err := h.Service.IngestTokenEventBatch(r.Context(), tenantID, events)
	if err != nil {
		var quotaErr ingestion.QuotaExceededError
		if errors.As(err, &quotaErr) {
			writeJSON(w, http.StatusPaymentRequired, Envelope{
				OK:     false,
				Status: "error",
				Error:  issue("quota_exceeded", err.Error()),
				Degraded: []domain.Issue{{
					Code:    "quota_exceeded",
					Message: err.Error(),
				}},
			})
			return
		}
		writeServiceError(w, err, false)
		return
	}

	hasDegraded := false
	for _, res := range results {
		if len(res.Degraded) > 0 {
			hasDegraded = true
			break
		}
	}

	status := "success"
	if hasDegraded {
		status = "degraded"
	}

	writeJSON(w, http.StatusAccepted, Envelope{
		OK:     true,
		Status: status,
		Data:   results,
	})
}

func (h *IngestionHandler) HandleSetPricingOverride(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeMethodError(w)
		return
	}

	tenantID, ok := tenantFromRequest(w, r)
	if !ok {
		return
	}

	var override domain.PricePoint
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&override); err != nil {
		writeJSON(w, http.StatusBadRequest, Envelope{
			OK:     false,
			Status: "error",
			Error:  issue("invalid_json", "Request body must be a valid PricePoint JSON."),
			Degraded: []domain.Issue{{
				Code:    "invalid_json",
				Message: "JSON decoding failed.",
				Field:   "body",
			}},
		})
		return
	}

	if override.Provider == "" || override.ModelID == "" {
		writeJSON(w, http.StatusBadRequest, Envelope{
			OK:     false,
			Status: "error",
			Error:  issue("invalid_request", "Provider and ModelID are required."),
		})
		return
	}

	if err := h.Service.SetPricingOverride(r.Context(), tenantID, override); err != nil {
		writeServiceError(w, err, false)
		return
	}
	h.audit(r, tenantID, "pricing.override_set", "pricing:"+override.Provider+":"+override.ModelID, map[string]interface{}{
		"provider": override.Provider,
		"model_id": override.ModelID,
	})

	writeJSON(w, http.StatusOK, Envelope{
		OK:     true,
		Status: "success",
		Data:   override,
	})
}

func (h *IngestionHandler) HandleTaskCompletion(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusGone, Envelope{
		OK:     false,
		Status: "degraded",
		Error:  issue("route_replaced", "Task completion ingestion is replaced by token usage event ingestion in this MVP."),
		Degraded: []domain.Issue{{
			Code:    "route_replaced",
			Message: "Use /api/ingest/token-usage with output_status and review_score fields.",
		}},
	})
}

func (h *IngestionHandler) HandleOverview(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeMethodError(w)
		return
	}
	tenantID, ok := tenantFromRequest(w, r)
	if !ok {
		return
	}
	summary, err := h.Service.Overview(r.Context(), tenantID)
	if err != nil {
		fallback := domain.ProductivitySummary{
			TenantID:       tenantID,
			GeneratedAt:    time.Now().UTC(),
			CostByWorker:   []domain.WorkerBreakdown{},
			CostByCategory: []domain.CategoryBreakdown{},
			TopCostDrivers: []domain.CostDriver{},
		}
		if writeDashboardError(w, err, fallback) {
			return
		}
		return
	}
	status := "success"
	if len(summary.Degraded) > 0 {
		status = "degraded"
	}
	writeJSON(w, http.StatusOK, Envelope{OK: true, Status: status, Data: summary, Degraded: summary.Degraded})
}

func (h *IngestionHandler) HandleWorkers(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeMethodError(w)
		return
	}
	tenantID, ok := tenantFromRequest(w, r)
	if !ok {
		return
	}
	workers, err := h.Service.Workers(r.Context(), tenantID)
	if err != nil {
		if writeDashboardError(w, err, []domain.WorkerBreakdown{}) {
			return
		}
		return
	}
	status := "success"
	degraded := []domain.Issue(nil)
	if len(workers) == 0 {
		status = "degraded"
		degraded = append(degraded, domain.Issue{Code: "no_data", Message: "No workers exist for this tenant."})
	}
	writeJSON(w, http.StatusOK, Envelope{OK: true, Status: status, Data: workers, Degraded: degraded})
}

func (h *IngestionHandler) HandleWorkerReview(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeMethodError(w)
		return
	}
	tenantID, ok := tenantFromRequest(w, r)
	if !ok {
		return
	}
	workerID := workerIDFromPath(r.URL.Path)
	review, err := h.Service.WorkerReview(r.Context(), tenantID, workerID)
	if err != nil {
		if writeDashboardError(w, err, domain.WorkerReview{}) {
			return
		}
		return
	}
	status := "success"
	if len(review.Degraded) > 0 {
		status = "degraded"
	}
	writeJSON(w, http.StatusOK, Envelope{OK: true, Status: status, Data: review, Degraded: review.Degraded})
}

func (h *IngestionHandler) HandleAnomalies(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeMethodError(w)
		return
	}
	tenantID, ok := tenantFromRequest(w, r)
	if !ok {
		return
	}
	signals, err := h.Service.Anomalies(r.Context(), tenantID, limitFromRequest(r))
	if err != nil {
		if writeDashboardError(w, err, []domain.AnomalySignal{}) {
			return
		}
		return
	}
	status := "success"
	degraded := []domain.Issue(nil)
	if len(signals) == 0 {
		status = "degraded"
		degraded = append(degraded, domain.Issue{Code: "no_data", Message: "No anomaly signals exist for this tenant."})
	}
	writeJSON(w, http.StatusOK, Envelope{OK: true, Status: status, Data: signals, Degraded: degraded})
}

func (h *IngestionHandler) HandleRecentEvents(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeMethodError(w)
		return
	}
	tenantID, ok := tenantFromRequest(w, r)
	if !ok {
		return
	}
	var events []domain.TokenEvent
	var err error
	if beforeStr := r.URL.Query().Get("before"); beforeStr != "" {
		if before, parseErr := time.Parse(time.RFC3339Nano, beforeStr); parseErr == nil {
			events, err = h.Service.RecentEventsBefore(r.Context(), tenantID, before, limitFromRequest(r))
		} else {
			events, err = h.Service.RecentEvents(r.Context(), tenantID, limitFromRequest(r))
		}
	} else {
		events, err = h.Service.RecentEvents(r.Context(), tenantID, limitFromRequest(r))
	}

	if err != nil {
		if writeDashboardError(w, err, []domain.TokenEvent{}) {
			return
		}
		return
	}
	status := "success"
	degraded := []domain.Issue(nil)
	if len(events) == 0 {
		status = "degraded"
		degraded = append(degraded, domain.Issue{Code: "no_data", Message: "No usage events exist for this tenant."})
	}
	writeJSON(w, http.StatusOK, Envelope{OK: true, Status: status, Data: events, Degraded: degraded})
}

func (h *IngestionHandler) HandleOutputAnalyses(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeMethodError(w)
		return
	}
	tenantID, ok := tenantFromRequest(w, r)
	if !ok {
		return
	}
	analyses, err := h.Service.OutputAnalyses(r.Context(), tenantID, limitFromRequest(r))
	if err != nil {
		if writeDashboardError(w, err, []domain.OutputAnalysis{}) {
			return
		}
		return
	}
	status := "success"
	degraded := []domain.Issue(nil)
	if len(analyses) == 0 {
		status = "degraded"
		degraded = append(degraded, domain.Issue{Code: "no_data", Message: "No output analyses exist for this tenant."})
	}
	writeJSON(w, http.StatusOK, Envelope{OK: true, Status: status, Data: analyses, Degraded: degraded})
}

func (h *IngestionHandler) HandleRecommendations(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeMethodError(w)
		return
	}
	tenantID, ok := tenantFromRequest(w, r)
	if !ok {
		return
	}
	recs, err := h.Service.Recommendations(r.Context(), tenantID)
	if err != nil {
		if writeDashboardError(w, err, []domain.RoutingRecommendation{}) {
			return
		}
		return
	}
	status := "success"
	degraded := []domain.Issue(nil)
	if len(recs) == 0 {
		status = "degraded"
		degraded = append(degraded, domain.Issue{Code: "no_data", Message: "No routing recommendations available."})
	}
	writeJSON(w, http.StatusOK, Envelope{OK: true, Status: status, Data: recs, Degraded: degraded})
}

func (h *IngestionHandler) HandleRecommendationState(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeMethodError(w)
		return
	}
	tenantID, ok := tenantFromRequest(w, r)
	if !ok {
		return
	}
	recommendationID := recommendationIDFromPath(r.URL.Path)
	var update domain.RecommendationStateUpdate
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&update); err != nil {
		writeJSON(w, http.StatusBadRequest, Envelope{
			OK:       false,
			Status:   "error",
			Error:    issue("invalid_json", "Request body must be a valid recommendation state update."),
			Degraded: []domain.Issue{{Code: "invalid_json", Message: "JSON decoding failed.", Field: "body"}},
		})
		return
	}
	state, err := h.Service.SetRecommendationState(r.Context(), tenantID, recommendationID, getActor(r), update)
	if err != nil {
		writeServiceError(w, err, false)
		return
	}
	h.audit(r, tenantID, "recommendation.state_changed", "recommendation:"+recommendationID, map[string]interface{}{
		"recommendation_id": recommendationID,
		"status":            state.Status,
	})
	writeJSON(w, http.StatusOK, Envelope{OK: true, Status: "success", Data: state})
}

func (h *IngestionHandler) HandleExportCSV(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeMethodError(w)
		return
	}
	tenantID, ok := tenantFromRequest(w, r)
	if !ok {
		return
	}
	events, err := h.Service.RecentEvents(r.Context(), tenantID, 10000)
	if err != nil {
		w.Header().Set("Content-Type", "text/csv")
		w.WriteHeader(http.StatusOK)
		writer := csv.NewWriter(w)
		_ = writer.Write([]string{"status", "code", "message"})
		_ = writer.Write([]string{"degraded", "database_unavailable", "Storage is unavailable; export contains no tenant records."})
		writer.Flush()
		return
	}

	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", "attachment;filename=token_goblin_export.csv")
	w.WriteHeader(http.StatusOK)

	writer := csv.NewWriter(w)
	if err := writer.Write([]string{"event_id", "timestamp", "worker_id", "job_id", "provider", "model_id", "total_tokens", "cost_usd", "task_category", "output_status"}); err != nil {
		return
	}

	for _, event := range events {
		costStr := ""
		if event.CostEstimateUSD != nil {
			costStr = fmt.Sprintf("%.6f", *event.CostEstimateUSD)
		}
		if err := writer.Write([]string{
			sanitizeCSVField(event.EventID),
			event.Timestamp.Format(time.RFC3339),
			sanitizeCSVField(event.WorkerID),
			sanitizeCSVField(event.JobID),
			sanitizeCSVField(event.Provider),
			sanitizeCSVField(event.ModelID),
			fmt.Sprintf("%d", event.TotalTokens),
			costStr,
			sanitizeCSVField(event.TaskCategory),
			sanitizeCSVField(string(event.OutputStatus)),
		}); err != nil {
			return
		}
	}
	writer.Flush()
	h.audit(r, tenantID, "export.csv", "tenant_export", map[string]interface{}{"format": "csv"})
}

func (h *IngestionHandler) HandleReportMarkdown(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeMethodError(w)
		return
	}
	tenantID, ok := tenantFromRequest(w, r)
	if !ok {
		return
	}
	summary, err := h.Service.Overview(r.Context(), tenantID)
	if err != nil {
		if errors.Is(err, storage.ErrUnavailable) {
			w.Header().Set("Content-Type", "text/markdown; charset=utf-8")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("# TokenGoblin Review Report\n\nStatus: degraded\n\nStorage is unavailable; no tenant records were exported.\n"))
			return
		}
		writeServiceError(w, err, true)
		return
	}
	recs, _ := h.Service.Recommendations(r.Context(), tenantID)
	analyses, _ := h.Service.OutputAnalyses(r.Context(), tenantID, 10)

	var b strings.Builder
	b.WriteString("# TokenGoblin Review Report\n\n")
	b.WriteString(fmt.Sprintf("- Tenant: `%s`\n", tenantID))
	b.WriteString(fmt.Sprintf("- Generated: `%s`\n", summary.GeneratedAt.Format(time.RFC3339)))
	b.WriteString(fmt.Sprintf("- Total tokens: `%d`\n", totalTokens(summary.CostByWorker)))
	b.WriteString(fmt.Sprintf("- Estimated cost: `$%.4f USD`\n", summary.TotalCostUSD))
	b.WriteString(fmt.Sprintf("- Outputs: `%d`\n", summary.OutputCount))
	b.WriteString(fmt.Sprintf("- Unknown-cost events: `%d`\n\n", summary.UnknownCostEventCount))

	b.WriteString("## Waste Signals\n\n")
	if len(summary.TopCostDrivers) == 0 && len(analyses) == 0 {
		b.WriteString("No usage evidence exists for this tenant yet.\n\n")
	} else {
		for _, driver := range summary.TopCostDrivers {
			b.WriteString("- ")
			b.WriteString(driver.Type)
			b.WriteString(" `")
			b.WriteString(driver.Key)
			b.WriteString("`: estimated `$")
			b.WriteString(strconv.FormatFloat(driver.TotalCostUSD, 'f', 4, 64))
			b.WriteString("` across `")
			b.WriteString(strconv.Itoa(driver.EventCount))
			b.WriteString("` events\n")
		}
		for _, item := range analyses {
			for _, issue := range item.Issues {
				b.WriteString("- `")
				b.WriteString(issue.Code)
				b.WriteString("` on event `")
				b.WriteString(item.EventID)
				b.WriteString("`: ")
				b.WriteString(issue.Message)
				b.WriteString("\n")
			}
		}
		b.WriteString("\n")
	}

	b.WriteString("## Top Recommendations\n\n")
	if len(recs) == 0 {
		b.WriteString("No routing recommendations are available from the current evidence set.\n")
	} else {
		for _, rec := range recs {
			b.WriteString(fmt.Sprintf("- %s Estimated savings: `$%.2f`. Evidence count: `%d`.\n", rec.Reason, rec.EstimatedSavingsUSD, rec.EvidenceCount))
		}
	}

	w.Header().Set("Content-Type", "text/markdown; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(b.String()))
	h.audit(r, tenantID, "export.markdown", "tenant_report", map[string]interface{}{"format": "markdown"})
}

func (h *IngestionHandler) HandleGetPricing(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeMethodError(w)
		return
	}
	tenantID, ok := tenantFromRequest(w, r)
	if !ok {
		return
	}
	pricing, err := h.Service.GetActivePricing(r.Context(), tenantID)
	if err != nil {
		writeServiceError(w, err, true)
		return
	}
	writeJSON(w, http.StatusOK, Envelope{
		OK:     true,
		Status: "success",
		Data:   pricing,
	})
}

func (h *IngestionHandler) HandleResetTenantData(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete && r.Method != http.MethodPost {
		writeMethodError(w)
		return
	}
	tenantID, ok := tenantFromRequest(w, r)
	if !ok {
		return
	}
	if err := h.Service.DeleteTenantData(r.Context(), tenantID); err != nil {
		writeServiceError(w, err, false)
		return
	}
	h.audit(r, tenantID, "tenant.reset", "tenant:"+tenantID, nil)
	writeJSON(w, http.StatusOK, Envelope{
		OK:     true,
		Status: "success",
		Data:   "Tenant data cleared successfully.",
	})
}

func (h *IngestionHandler) HandleSeedDemoData(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeMethodError(w)
		return
	}
	tenantID, ok := tenantFromRequest(w, r)
	if !ok {
		return
	}

	if err := demo.Seed(r.Context(), h.Repo, h.Service, tenantID); err != nil {
		writeServiceError(w, err, false)
		return
	}
	h.audit(r, tenantID, "tenant.seed_demo", "tenant:"+tenantID, nil)

	writeJSON(w, http.StatusOK, Envelope{
		OK:     true,
		Status: "success",
		Data:   "Demo data seeded successfully.",
	})
}

func (h *IngestionHandler) HandleAuditEvents(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeMethodError(w)
		return
	}
	tenantID, ok := tenantFromRequest(w, r)
	if !ok {
		return
	}
	events, err := h.Service.AuditEvents(r.Context(), tenantID, limitFromRequest(r))
	if err != nil {
		if writeDashboardError(w, err, []domain.AuditEvent{}) {
			return
		}
		return
	}
	status := "success"
	degraded := []domain.Issue(nil)
	if len(events) == 0 {
		status = "degraded"
		degraded = append(degraded, domain.Issue{Code: "no_data", Message: "No audit events exist for this tenant."})
	}
	writeJSON(w, http.StatusOK, Envelope{OK: true, Status: status, Data: events, Degraded: degraded})
}

func (h *IngestionHandler) HandleTenantMembers(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := tenantFromRequest(w, r)
	if !ok {
		return
	}
	switch r.Method {
	case http.MethodGet:
		members, err := h.Service.TenantMembers(r.Context(), tenantID)
		if err != nil {
			if writeDashboardError(w, err, []domain.TenantMember{}) {
				return
			}
			return
		}
		status := "success"
		degraded := []domain.Issue(nil)
		if len(members) == 0 {
			status = "degraded"
			degraded = append(degraded, domain.Issue{Code: "no_members", Message: "No explicit tenant members have been configured."})
		}
		writeJSON(w, http.StatusOK, Envelope{OK: true, Status: status, Data: members, Degraded: degraded})
	case http.MethodPost:
		if getRole(r) != "owner" && getRole(r) != "admin" {
			writeJSON(w, http.StatusForbidden, Envelope{OK: false, Status: "error", Error: issue("forbidden", "Only admin or owner roles can change tenant members.")})
			return
		}
		var member domain.TenantMember
		decoder := json.NewDecoder(r.Body)
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&member); err != nil {
			writeJSON(w, http.StatusBadRequest, Envelope{
				OK:       false,
				Status:   "error",
				Error:    issue("invalid_json", "Request body must be a valid tenant member."),
				Degraded: []domain.Issue{{Code: "invalid_json", Message: "JSON decoding failed.", Field: "body"}},
			})
			return
		}
		saved, err := h.Service.UpsertTenantMember(r.Context(), tenantID, member)
		if err != nil {
			writeServiceError(w, err, false)
			return
		}
		h.audit(r, tenantID, "tenant_member.upserted", "member:"+saved.SubjectID, map[string]interface{}{"role": saved.Role})
		writeJSON(w, http.StatusOK, Envelope{OK: true, Status: "success", Data: saved})
	default:
		writeMethodError(w)
	}
}

func (h *IngestionHandler) HandleVerifiedStripeEvent(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeMethodError(w)
		return
	}
	if !validInternalBearer(r) {
		writeJSON(w, http.StatusUnauthorized, Envelope{
			OK:     false,
			Status: "error",
			Error:  issue("internal_auth_invalid", "Internal billing route requires a valid service bearer token."),
		})
		return
	}

	var event billing.VerifiedStripeEvent
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&event); err != nil {
		writeJSON(w, http.StatusBadRequest, Envelope{
			OK:     false,
			Status: "error",
			Error:  issue("invalid_json", "Request body must be a verified Stripe lifecycle event JSON object."),
		})
		return
	}

	result, err := billing.ProcessVerifiedStripeEvent(r.Context(), h.Repo, event, time.Now().UTC())
	if err != nil {
		var validationErr billing.StripeValidationError
		if errors.As(err, &validationErr) {
			writeJSON(w, http.StatusBadRequest, Envelope{
				OK:     false,
				Status: "error",
				Error:  issue("invalid_stripe_event", validationErr.Error()),
			})
			return
		}
		var conflictErr billing.StripeConflictError
		if errors.As(err, &conflictErr) {
			writeJSON(w, http.StatusConflict, Envelope{
				OK:     false,
				Status: "error",
				Error:  issue("stripe_owner_conflict", conflictErr.Error()),
			})
			return
		}
		writeJSON(w, http.StatusInternalServerError, Envelope{
			OK:     false,
			Status: "error",
			Error:  issue("billing_update_failed", "Verified Stripe event could not be applied."),
		})
		return
	}

	status := "ignored"
	if result.Applied {
		status = "success"
	}
	writeJSON(w, http.StatusOK, Envelope{OK: true, Status: status, Data: result})
}

func (h *IngestionHandler) HandleStripeWebhook(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeMethodError(w)
		return
	}

	const MaxBodyBytes = int64(65536)
	r.Body = http.MaxBytesReader(w, r.Body, MaxBodyBytes)
	payload, err := io.ReadAll(r.Body)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, Envelope{OK: false, Status: "error", Error: issue("read_error", err.Error())})
		return
	}

	secret := os.Getenv("STRIPE_WEBHOOK_SECRET")
	if secret == "" {
		writeJSON(w, http.StatusInternalServerError, Envelope{OK: false, Status: "error", Error: issue("internal_error", "STRIPE_WEBHOOK_SECRET is not configured")})
		return
	}

	event, err := webhook.ConstructEvent(payload, r.Header.Get("Stripe-Signature"), secret)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, Envelope{OK: false, Status: "error", Error: issue("invalid_signature", err.Error())})
		return
	}

	verifiedEvent := billing.VerifiedStripeEvent{
		EventID:   event.ID,
		EventType: string(event.Type),
	}

	if strings.HasPrefix(string(event.Type), "customer.subscription.") {
		var sub stripe.Subscription
		if err := json.Unmarshal(event.Data.Raw, &sub); err != nil {
			writeJSON(w, http.StatusInternalServerError, Envelope{OK: false, Status: "error", Error: issue("internal_error", "failed to parse subscription data")})
			return
		}
		verifiedEvent.SubscriptionID = sub.ID
		if sub.Customer != nil {
			verifiedEvent.CustomerID = sub.Customer.ID
		}
		verifiedEvent.SubscriptionStatus = string(sub.Status)
		verifiedEvent.Metadata = sub.Metadata
	} else if string(event.Type) == "checkout.session.completed" {
		var session stripe.CheckoutSession
		if err := json.Unmarshal(event.Data.Raw, &session); err != nil {
			writeJSON(w, http.StatusInternalServerError, Envelope{OK: false, Status: "error", Error: issue("internal_error", "failed to parse checkout session data")})
			return
		}
		if session.Customer != nil {
			verifiedEvent.CustomerID = session.Customer.ID
		}
		if session.Subscription != nil {
			verifiedEvent.SubscriptionID = session.Subscription.ID
		}
		verifiedEvent.TenantID = session.ClientReferenceID
		verifiedEvent.Metadata = session.Metadata
	}

	_, err = billing.ProcessVerifiedStripeEvent(r.Context(), h.Repo, verifiedEvent, time.Now().UTC())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, Envelope{OK: false, Status: "error", Error: issue("processing_error", err.Error())})
		return
	}

	w.WriteHeader(http.StatusOK)
}

func tenantFromRequest(w http.ResponseWriter, r *http.Request) (string, bool) {
	if val := r.Context().Value(tenantIDKey); val != nil {
		if tenantID, ok := val.(string); ok && tenantID != "" {
			return tenantID, true
		}
	}
	tenantID := strings.TrimSpace(r.Header.Get("x-tenant-id"))
	if tenantID == "" {
		writeJSON(w, http.StatusUnauthorized, Envelope{
			OK:     false,
			Status: "error",
			Error:  issue("tenant_missing", "Missing required x-tenant-id header."),
		})
		return "", false
	}
	return tenantID, true
}

func workerIDFromPath(path string) string {
	for _, prefix := range []string{"/api/dashboard/workers/", "/v1/dashboard/workers/"} {
		if strings.HasPrefix(path, prefix) {
			return strings.TrimSpace(strings.TrimPrefix(path, prefix))
		}
	}
	return ""
}

func recommendationIDFromPath(path string) string {
	for _, prefix := range []string{"/api/dashboard/recommendations/", "/v1/dashboard/recommendations/"} {
		if strings.HasPrefix(path, prefix) {
			rest := strings.Trim(strings.TrimPrefix(path, prefix), "/")
			raw := strings.TrimSuffix(rest, "/status")
			decoded, err := url.PathUnescape(raw)
			if err == nil {
				return decoded
			}
			return raw
		}
	}
	return ""
}

func (h *IngestionHandler) audit(r *http.Request, tenantID, eventType, resource string, metadata map[string]interface{}) {
	_ = h.Repo.SaveAuditEvent(r.Context(), domain.AuditEvent{
		EventID:   "aud_" + randomHex(12),
		TenantID:  tenantID,
		Type:      eventType,
		Actor:     getActor(r),
		Resource:  resource,
		Metadata:  metadata,
		Timestamp: time.Now().UTC(),
	})
}

func validInternalBearer(r *http.Request) bool {
	expected := strings.TrimSpace(os.Getenv("TG_INTERNAL_WEBHOOK_SECRET"))
	if expected == "" {
		return false
	}
	header := strings.TrimSpace(r.Header.Get("Authorization"))
	if !strings.HasPrefix(header, "Bearer ") {
		return false
	}
	token := strings.TrimSpace(strings.TrimPrefix(header, "Bearer "))
	if token == "" || len(token) != len(expected) {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(token), []byte(expected)) == 1
}

func randomHex(bytes int) string {
	buffer := make([]byte, bytes)
	if _, err := rand.Read(buffer); err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(buffer)
}

func totalTokens(workers []domain.WorkerBreakdown) int {
	total := 0
	for _, worker := range workers {
		total += worker.TotalTokens
	}
	return total
}

func writeMethodError(w http.ResponseWriter) {
	writeJSON(w, http.StatusMethodNotAllowed, Envelope{
		OK:     false,
		Status: "error",
		Error:  issue("method_not_allowed", "Method is not allowed for this route."),
	})
}

func writeServiceError(w http.ResponseWriter, err error, readRoute bool) {
	var validationErr ingestion.ValidationError
	if errors.As(err, &validationErr) {
		writeJSON(w, http.StatusBadRequest, Envelope{
			OK:       false,
			Status:   "error",
			Error:    issue("validation_error", "Request validation failed."),
			Degraded: validationErr.Issues,
		})
		return
	}
	var missingTenant ingestion.TenantMissingError
	if errors.As(err, &missingTenant) {
		writeJSON(w, http.StatusUnauthorized, Envelope{
			OK:     false,
			Status: "error",
			Error:  issue("tenant_missing", "Missing required x-tenant-id header."),
		})
		return
	}
	var tenantMismatch ingestion.TenantMismatchError
	if errors.As(err, &tenantMismatch) {
		writeJSON(w, http.StatusForbidden, Envelope{
			OK:     false,
			Status: "error",
			Error:  issue("tenant_mismatch", "Payload tenant_id does not match request tenant context."),
		})
		return
	}
	if errors.Is(err, storage.ErrUnavailable) {
		degraded := []domain.Issue{{
			Code:    "database_unavailable",
			Message: "Storage is unavailable; returning a degraded response.",
		}}
		statusCode := http.StatusServiceUnavailable
		if readRoute {
			statusCode = http.StatusOK
		}
		writeJSON(w, statusCode, Envelope{
			OK:       readRoute,
			Status:   "degraded",
			Data:     emptyReadData(readRoute),
			Degraded: degraded,
			Error:    issue("database_unavailable", "Storage is unavailable."),
		})
		return
	}

	writeJSON(w, http.StatusServiceUnavailable, Envelope{
		OK:     false,
		Status: "degraded",
		Error:  issue("service_unavailable", "Request could not be completed."),
	})
}

func writeDashboardError(w http.ResponseWriter, err error, fallback interface{}) bool {
	if errors.Is(err, storage.ErrUnavailable) {
		degraded := []domain.Issue{{
			Code:    "database_unavailable",
			Message: "Storage is unavailable; returning a degraded response.",
		}}
		writeJSON(w, http.StatusOK, Envelope{
			OK:       true,
			Status:   "degraded",
			Data:     fallback,
			Degraded: degraded,
			Error:    issue("database_unavailable", "Storage is unavailable."),
		})
		return true
	}
	writeServiceError(w, err, false)
	return true
}

func emptyReadData(readRoute bool) interface{} {
	if readRoute {
		return []interface{}{}
	}
	return nil
}

func writeJSON(w http.ResponseWriter, statusCode int, body Envelope) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(body)
}

func issue(code string, message string) *domain.Issue {
	return &domain.Issue{Code: code, Message: message}
}

func limitFromRequest(r *http.Request) int {
	raw := r.URL.Query().Get("limit")
	if raw == "" {
		return 100
	}
	limit, err := strconv.Atoi(raw)
	if err != nil || limit <= 0 {
		return 100
	}
	if limit > 500 {
		return 500
	}
	return limit
}

func (h *IngestionHandler) HandleDeleteTenant(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost && r.Method != http.MethodDelete {
		writeMethodError(w)
		return
	}
	tenantID, ok := tenantFromRequest(w, r)
	if !ok {
		return
	}
	if err := h.Service.DeleteTenantData(r.Context(), tenantID); err != nil {
		writeServiceError(w, err, false)
		return
	}
	h.audit(r, tenantID, "tenant.deleted", "tenant:"+tenantID, nil)
	writeJSON(w, http.StatusOK, Envelope{
		OK:     true,
		Status: "success",
		Data:   "Tenant and all associated data deleted successfully.",
	})
}

func (h *IngestionHandler) HandleExportTenant(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeMethodError(w)
		return
	}
	tenantID, ok := tenantFromRequest(w, r)
	if !ok {
		return
	}

	// Gather all tenant data for export
	export := map[string]interface{}{
		"tenant_id":   tenantID,
		"exported_at": time.Now().UTC().Format(time.RFC3339),
	}

	// Get overview/summary
	summary, err := h.Service.Overview(r.Context(), tenantID)
	if err == nil {
		export["overview"] = summary
	} else {
		export["overview_error"] = err.Error()
	}

	// Get recent events
	events, err := h.Service.RecentEvents(r.Context(), tenantID, 10000)
	if err == nil {
		export["events"] = events
	}

	// Get workers
	workers, err := h.Service.Workers(r.Context(), tenantID)
	if err == nil {
		export["workers"] = workers
	}

	// Get anomalies
	anomalies, err := h.Service.Anomalies(r.Context(), tenantID, 100)
	if err == nil {
		export["anomalies"] = anomalies
	}

	// Get audit events
	auditEvents, err := h.Service.AuditEvents(r.Context(), tenantID, 1000)
	if err == nil {
		export["audit_events"] = auditEvents
	}

	// Get tenant members
	members, err := h.Service.TenantMembers(r.Context(), tenantID)
	if err == nil {
		export["tenant_members"] = members
	}

	// Get pricing
	pricing, err := h.Service.GetActivePricing(r.Context(), tenantID)
	if err == nil {
		export["pricing"] = pricing
	}

	// Get output analyses
	analyses, err := h.Service.OutputAnalyses(r.Context(), tenantID, 100)
	if err == nil {
		export["output_analyses"] = analyses
	}

	// Get recommendations
	recs, err := h.Service.Recommendations(r.Context(), tenantID)
	if err == nil {
		export["recommendations"] = recs
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Disposition", "attachment; filename=tenant_export_"+tenantID+".json")
	w.WriteHeader(http.StatusOK)

	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	_ = encoder.Encode(export)

	h.audit(r, tenantID, "export.tenant_json", "tenant:"+tenantID, map[string]interface{}{"format": "json"})
}

func sanitizeCSVField(s string) string {
	if s == "" {
		return s
	}
	switch s[0] {
	case '=', '+', '-', '@':
		return "'" + s
	}
	return s
}
