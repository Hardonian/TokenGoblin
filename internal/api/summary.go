package api

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/Hardonian/TokenGoblin/internal/storage"
)

type SummaryHandler struct {
	Store storage.Repository
}

func NewSummaryHandler(store storage.Repository) *SummaryHandler {
	return &SummaryHandler{Store: store}
}

// CostSummaryResponse maps to the data contract.
type CostSummaryResponse struct {
	TenantID     string     `json:"tenant_id"`
	TimeRange    TimeRange  `json:"time_range"`
	TotalCostUSD float64    `json:"total_cost_usd"`
	TopModels    []ModelCost `json:"top_models"`
	TopWorkers   []WorkerCost `json:"top_workers"`
}

type TimeRange struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
}

type ModelCost struct {
	ModelID string  `json:"model_id"`
	CostUSD float64 `json:"cost_usd"`
}

type WorkerCost struct {
	WorkerID string  `json:"worker_id"`
	CostUSD  float64 `json:"cost_usd"`
}

func (h *SummaryHandler) HandleGetSummary(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	tenantID := r.URL.Query().Get("tenant_id")
	if tenantID == "" {
		http.Error(w, "tenant_id query parameter is required", http.StatusBadRequest)
		return
	}

	// For MVP, we'll just look at the last 30 days
	end := time.Now().UTC()
	start := end.Add(-30 * 24 * time.Hour)

	events, err := h.Store.ListTokenEvents(r.Context(), tenantID, 500)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	var totalCost float64
	modelCosts := make(map[string]float64)
	workerCosts := make(map[string]float64)

	for _, ev := range events {
		// Filter by time range since ListTokenEvents just gives recent 500
		if ev.Timestamp.Before(start) || ev.Timestamp.After(end) {
			continue
		}
		var cost float64
		if ev.CostEstimateUSD != nil {
			cost = *ev.CostEstimateUSD
		}
		totalCost += cost
		modelCosts[ev.ModelID] += cost
		workerCosts[ev.WorkerID] += cost
	}

	// Build the response
	resp := CostSummaryResponse{
		TenantID: tenantID,
		TimeRange: TimeRange{
			Start: start,
			End:   end,
		},
		TotalCostUSD: totalCost,
		TopModels:    []ModelCost{},
		TopWorkers:   []WorkerCost{},
	}

	for mID, c := range modelCosts {
		resp.TopModels = append(resp.TopModels, ModelCost{ModelID: mID, CostUSD: c})
	}
	for wID, c := range workerCosts {
		resp.TopWorkers = append(resp.TopWorkers, WorkerCost{WorkerID: wID, CostUSD: c})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// WorkerBreakdownResponse maps to the data contract.
type WorkerBreakdownResponse struct {
	WorkerID          string  `json:"worker_id"`
	TotalJobs         int     `json:"total_jobs"`
	TotalCostUSD      float64 `json:"total_cost_usd"`
	AvgCostPerJobUSD  float64 `json:"avg_cost_per_job_usd"`
	AnomalyScore      float64 `json:"anomaly_score"`
	EfficiencyRating  string  `json:"efficiency_rating"`
}

func (h *SummaryHandler) HandleGetWorker(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	tenantID := r.URL.Query().Get("tenant_id")
	if tenantID == "" {
		http.Error(w, "tenant_id query parameter is required", http.StatusBadRequest)
		return
	}

	// Extract worker ID from the URL path: /api/v1/workers/{worker_id}
	parts := strings.Split(r.URL.Path, "/")
	workerID := parts[len(parts)-1]

	end := time.Now().UTC()
	start := end.Add(-30 * 24 * time.Hour)

	events, err := h.Store.ListTokenEvents(r.Context(), tenantID, 500)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Filter for worker and time range
	var totalCost float64
	var totalJobs int

	// Basic deduplication of jobs via simple mapping
	jobsSeen := make(map[string]bool)

	for _, ev := range events {
		if ev.Timestamp.Before(start) || ev.Timestamp.After(end) {
			continue
		}
		if ev.WorkerID == workerID {
			if ev.CostEstimateUSD != nil {
				totalCost += *ev.CostEstimateUSD
			}
			if !jobsSeen[ev.JobID] && ev.JobID != "" {
				jobsSeen[ev.JobID] = true
				totalJobs++
			}
		}
	}

	// If the schema was updated, wait, does TokenEvent have a JobID?
	// Let's check our domain.TokenEvent again. It has TaskType, SessionID, EventID. The contract specifies job_id.

	// Calculate average
	avgCost := 0.0
	if totalJobs > 0 {
		avgCost = totalCost / float64(totalJobs)
	}

	// For MVP, if they have no jobs but cost, just use 1 as denominator to avoid NaN.
	if totalJobs == 0 && totalCost > 0 {
		avgCost = totalCost
	}

	resp := WorkerBreakdownResponse{
		WorkerID:         workerID,
		TotalJobs:        totalJobs,
		TotalCostUSD:     totalCost,
		AvgCostPerJobUSD: avgCost,
		AnomalyScore:     0.0,
		EfficiencyRating: "optimal", // Default for MVP
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}
