package api

import (
	"net/http"

	"github.com/Hardonian/TokenGoblin/internal/domain"
	"github.com/Hardonian/TokenGoblin/internal/forecast"
	"github.com/Hardonian/TokenGoblin/internal/intelligence"
	"github.com/Hardonian/TokenGoblin/internal/storage"
)

// V2Handler serves the v2 API endpoints for intelligence, forecasting,
// and executive features.
type V2Handler struct {
	repo       storage.Repository
	intel      *intelligence.Engine
	forecaster *forecast.Engine
}

// NewV2Handler creates a handler for v2 API routes.
func NewV2Handler(repo storage.Repository) *V2Handler {
	return &V2Handler{
		repo:       repo,
		intel:      intelligence.NewEngine(),
		forecaster: forecast.NewEngine(),
	}
}

// ═══════════════════════════════════════════════════════════
// Intelligence Endpoints
// ═══════════════════════════════════════════════════════════

// HandleWasteReport returns a comprehensive waste analysis for the tenant.
// GET /v2/intelligence/waste
func (h *V2Handler) HandleWasteReport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeMethodError(w)
		return
	}

	tenantID, ok := tenantFromRequest(w, r)
	if !ok {
		return
	}

	events, err := h.repo.ListTokenEvents(r.Context(), tenantID, 10000)
	if err != nil {
		if writeDashboardError(w, err, domain.WasteReport{}) {
			return
		}
		return
	}

	report := h.intel.GenerateWasteReport(tenantID, events)
	writeJSON(w, http.StatusOK, Envelope{OK: true, Status: "success", Data: report})
}

// HandleRefinePrompt accepts a raw prompt and returns a minified, refined version.
// POST /v2/intelligence/refine
func (h *V2Handler) HandleRefinePrompt(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeMethodError(w)
		return
	}

	_, ok := tenantFromRequest(w, r)
	if !ok {
		return
	}

	var req struct {
		Prompt string `json:"prompt"`
	}
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, Envelope{
			OK:     false,
			Status: "error",
			Error:  issue("invalid_json", "Request body must include 'prompt' field."),
		})
		return
	}

	if req.Prompt == "" {
		writeJSON(w, http.StatusBadRequest, Envelope{
			OK:     false,
			Status: "error",
			Error:  issue("invalid_request", "prompt is required."),
		})
		return
	}

	refined := intelligence.RefinePrompt(req.Prompt)

	writeJSON(w, http.StatusOK, Envelope{
		OK:     true,
		Status: "success",
		Data: map[string]interface{}{
			"original_length": len(req.Prompt),
			"refined_length":  len(refined),
			"refined_prompt":  refined,
			"savings_percent": float64(len(req.Prompt)-len(refined)) / float64(len(req.Prompt)) * 100,
		},
	})
}

// HandlePromptGraveyard returns prompts consuming cost with zero value.
// GET /v2/intelligence/prompt-graveyard
func (h *V2Handler) HandlePromptGraveyard(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeMethodError(w)
		return
	}

	tenantID, ok := tenantFromRequest(w, r)
	if !ok {
		return
	}

	events, err := h.repo.ListTokenEvents(r.Context(), tenantID, 10000)
	if err != nil {
		if writeDashboardError(w, err, []domain.PromptFingerprint{}) {
			return
		}
		return
	}

	fingerprints := h.intel.BuildFingerprints(tenantID, events)
	graveyard := h.intel.FindGraveyardPrompts(fingerprints)

	type graveyardResult struct {
		GraveyardPrompts []domain.PromptFingerprint `json:"graveyard_prompts"`
		TotalWasteUSD    float64                    `json:"total_waste_usd"`
		Count            int                        `json:"count"`
	}

	totalWaste := 0.0
	for _, fp := range graveyard {
		totalWaste += fp.TotalCostUSD
	}
	if graveyard == nil {
		graveyard = []domain.PromptFingerprint{}
	}

	writeJSON(w, http.StatusOK, Envelope{OK: true, Status: "success", Data: graveyardResult{
		GraveyardPrompts: graveyard,
		TotalWasteUSD:    totalWaste,
		Count:            len(graveyard),
	}})
}

// HandleZombieAgents returns agents with high activity but low value.
// GET /v2/intelligence/zombie-agents
func (h *V2Handler) HandleZombieAgents(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeMethodError(w)
		return
	}

	tenantID, ok := tenantFromRequest(w, r)
	if !ok {
		return
	}

	events, err := h.repo.ListTokenEvents(r.Context(), tenantID, 10000)
	if err != nil {
		if writeDashboardError(w, err, []domain.ZombieAgent{}) {
			return
		}
		return
	}

	zombies := h.intel.DetectZombieAgents(events)
	if zombies == nil {
		zombies = []domain.ZombieAgent{}
	}

	type zombieResult struct {
		ZombieAgents []domain.ZombieAgent `json:"zombie_agents"`
		Count        int                  `json:"count"`
	}

	writeJSON(w, http.StatusOK, Envelope{OK: true, Status: "success", Data: zombieResult{
		ZombieAgents: zombies,
		Count:        len(zombies),
	}})
}

// HandleDuplicates returns prompts used identically across multiple workers.
// GET /v2/intelligence/duplicates
func (h *V2Handler) HandleDuplicates(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeMethodError(w)
		return
	}

	tenantID, ok := tenantFromRequest(w, r)
	if !ok {
		return
	}

	events, err := h.repo.ListTokenEvents(r.Context(), tenantID, 10000)
	if err != nil {
		if writeDashboardError(w, err, []domain.DuplicateCluster{}) {
			return
		}
		return
	}

	fingerprints := h.intel.BuildFingerprints(tenantID, events)
	duplicates := h.intel.FindDuplicates(fingerprints)

	totalRedundantCost := 0.0
	for _, d := range duplicates {
		totalRedundantCost += d.RedundantCostUSD
	}
	if duplicates == nil {
		duplicates = []domain.DuplicateCluster{}
	}

	type dupResult struct {
		DuplicateClusters  []domain.DuplicateCluster `json:"duplicate_clusters"`
		TotalRedundantCost float64                   `json:"total_redundant_cost"`
		Count              int                       `json:"count"`
	}

	writeJSON(w, http.StatusOK, Envelope{OK: true, Status: "success", Data: dupResult{
		DuplicateClusters:  duplicates,
		TotalRedundantCost: totalRedundantCost,
		Count:              len(duplicates),
	}})
}

// HandleCostLeaks returns patterns of silent invisible spending.
// GET /v2/intelligence/cost-leaks
func (h *V2Handler) HandleCostLeaks(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeMethodError(w)
		return
	}

	tenantID, ok := tenantFromRequest(w, r)
	if !ok {
		return
	}

	events, err := h.repo.ListTokenEvents(r.Context(), tenantID, 10000)
	if err != nil {
		if writeDashboardError(w, err, []domain.CostLeak{}) {
			return
		}
		return
	}

	leaks := h.intel.DetectCostLeaks(events)
	totalLeakCost := 0.0
	for _, l := range leaks {
		totalLeakCost += l.CostUSD
	}
	if leaks == nil {
		leaks = []domain.CostLeak{}
	}

	type leakResult struct {
		CostLeaks     []domain.CostLeak `json:"cost_leaks"`
		TotalLeakCost float64           `json:"total_leak_cost"`
		Count         int               `json:"count"`
	}

	writeJSON(w, http.StatusOK, Envelope{OK: true, Status: "success", Data: leakResult{
		CostLeaks:     leaks,
		TotalLeakCost: totalLeakCost,
		Count:         len(leaks),
	}})
}

// HandleHallucinationMap returns failure clusters by model and category.
// GET /v2/intelligence/hallucination-map
func (h *V2Handler) HandleHallucinationMap(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeMethodError(w)
		return
	}

	tenantID, ok := tenantFromRequest(w, r)
	if !ok {
		return
	}

	events, err := h.repo.ListTokenEvents(r.Context(), tenantID, 10000)
	if err != nil {
		if writeDashboardError(w, err, []domain.HallucinationCell{}) {
			return
		}
		return
	}

	heatmap := h.intel.BuildHallucinationHeatmap(events)
	if heatmap == nil {
		heatmap = []domain.HallucinationCell{}
	}

	type heatmapResult struct {
		HeatmapCells []domain.HallucinationCell `json:"heatmap_cells"`
		Count        int                        `json:"count"`
	}

	writeJSON(w, http.StatusOK, Envelope{OK: true, Status: "success", Data: heatmapResult{
		HeatmapCells: heatmap,
		Count:        len(heatmap),
	}})
}

// ═══════════════════════════════════════════════════════════
// Forecasting Endpoints
// ═══════════════════════════════════════════════════════════

// HandleSpendForecast returns a monthly spend forecast with confidence intervals.
// GET /v2/forecasts/spend
func (h *V2Handler) HandleSpendForecast(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeMethodError(w)
		return
	}

	tenantID, ok := tenantFromRequest(w, r)
	if !ok {
		return
	}

	events, err := h.repo.ListTokenEvents(r.Context(), tenantID, 50000)
	if err != nil {
		if writeDashboardError(w, err, domain.SpendForecast{}) {
			return
		}
		return
	}

	spendForecast := h.forecaster.ForecastMonthlySpend(tenantID, events)
	writeJSON(w, http.StatusOK, Envelope{OK: true, Status: "success", Data: spendForecast})
}

// HandleExecutiveScorecard returns an AI scorecard for leadership.
// GET /v2/executive/scorecard
func (h *V2Handler) HandleExecutiveScorecard(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeMethodError(w)
		return
	}

	tenantID, ok := tenantFromRequest(w, r)
	if !ok {
		return
	}

	events, err := h.repo.ListTokenEvents(r.Context(), tenantID, 50000)
	if err != nil {
		if writeDashboardError(w, err, domain.ExecutiveScorecard{}) {
			return
		}
		return
	}

	// Get waste data for scorecard
	report := h.intel.GenerateWasteReport(tenantID, events)
	scorecard := h.forecaster.GenerateScorecard(tenantID, events, report.TotalWasteUSD)

	writeJSON(w, http.StatusOK, Envelope{OK: true, Status: "success", Data: scorecard})
}

// HandleModelComparison returns side-by-side model cost/quality/latency analysis.
// GET /v2/analytics/models
func (h *V2Handler) HandleModelComparison(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeMethodError(w)
		return
	}

	tenantID, ok := tenantFromRequest(w, r)
	if !ok {
		return
	}

	events, err := h.repo.ListTokenEvents(r.Context(), tenantID, 10000)
	if err != nil {
		if writeDashboardError(w, err, []modelStats{}) {
			return
		}
		return
	}

	models := make(map[string]*modelStats)
	for i := range events {
		ev := &events[i]
		key := ev.Provider + ":" + ev.ModelID
		ms, ok := models[key]
		if !ok {
			ms = &modelStats{ModelID: ev.ModelID, Provider: ev.Provider}
			models[key] = ms
		}
		ms.EventCount++
		if ev.CostEstimateUSD != nil {
			ms.TotalCostUSD += *ev.CostEstimateUSD
		}
		ms.TotalTokens += ev.TotalTokens
		if ev.OutputStatus == domain.OutputAccepted || ev.OutputStatus == domain.OutputSucceeded {
			ms.AcceptedCount++
		}
		ms.AvgLatencyMs += float64(ev.LatencyMs)
	}

	var result []modelStats
	for _, ms := range models {
		if ms.EventCount > 0 {
			ms.AvgCostPerCall = ms.TotalCostUSD / float64(ms.EventCount)
			ms.AcceptanceRate = float64(ms.AcceptedCount) / float64(ms.EventCount)
			ms.AvgLatencyMs = ms.AvgLatencyMs / float64(ms.EventCount)
		}
		if ms.AcceptedCount > 0 {
			ms.CostPerOutcome = ms.TotalCostUSD / float64(ms.AcceptedCount)
		}
		result = append(result, *ms)
	}
	if result == nil {
		result = []modelStats{}
	}

	type modelResult struct {
		Models []modelStats `json:"models"`
		Count  int          `json:"count"`
	}

	writeJSON(w, http.StatusOK, Envelope{OK: true, Status: "success", Data: modelResult{
		Models: result,
		Count:  len(result),
	}})
}

type modelStats struct {
	ModelID        string  `json:"model_id"`
	Provider       string  `json:"provider"`
	EventCount     int     `json:"event_count"`
	TotalCostUSD   float64 `json:"total_cost_usd"`
	AvgCostPerCall float64 `json:"avg_cost_per_call"`
	TotalTokens    int     `json:"total_tokens"`
	AcceptedCount  int     `json:"accepted_count"`
	AcceptanceRate float64 `json:"acceptance_rate"`
	AvgLatencyMs   float64 `json:"avg_latency_ms"`
	CostPerOutcome float64 `json:"cost_per_outcome"`
}
