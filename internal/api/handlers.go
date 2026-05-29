package api

import (
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/Hardonian/TokenGoblin/internal/domain"
	"github.com/Hardonian/TokenGoblin/internal/ingestion"
	"github.com/Hardonian/TokenGoblin/internal/storage"
)

type IngestionHandler struct {
	Service ingestion.Service
}

type Envelope struct {
	OK       bool           `json:"ok"`
	Status   string         `json:"status"`
	Data     interface{}    `json:"data,omitempty"`
	Warnings []domain.Issue `json:"warnings,omitempty"`
	Degraded []domain.Issue `json:"degraded,omitempty"`
	Error    *domain.Issue  `json:"error,omitempty"`
}

func NewIngestionHandler(service ingestion.Service) *IngestionHandler {
	return &IngestionHandler{Service: service}
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
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", "attachment;filename=token_goblin_export.csv")
	w.WriteHeader(http.StatusOK)

	writer := csv.NewWriter(w)
	writer.Write([]string{"event_id", "timestamp", "worker_id", "job_id", "provider", "model_id", "total_tokens", "cost_usd", "task_category", "output_status"})
	
	for _, event := range events {
		costStr := ""
		if event.CostEstimateUSD != nil {
			costStr = fmt.Sprintf("%.6f", *event.CostEstimateUSD)
		}
		writer.Write([]string{
			event.EventID,
			event.Timestamp.Format(time.RFC3339),
			event.WorkerID,
			event.JobID,
			event.Provider,
			event.ModelID,
			fmt.Sprintf("%d", event.TotalTokens),
			costStr,
			event.TaskCategory,
			string(event.OutputStatus),
		})
	}
	writer.Flush()
}

func tenantFromRequest(w http.ResponseWriter, r *http.Request) (string, bool) {
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
    w.WriteHeader(http.StatusNotImplemented)
}

func (h *IngestionHandler) HandleExportTenant(w http.ResponseWriter, r *http.Request) {
    w.WriteHeader(http.StatusNotImplemented)
}
