package domain

import "time"

type Issue struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Field   string `json:"field,omitempty"`
}

type Tenant struct {
	TenantID  string    `json:"tenant_id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Worker struct {
	TenantID  string    `json:"tenant_id"`
	WorkerID  string    `json:"worker_id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Job struct {
	TenantID     string    `json:"tenant_id"`
	JobID        string    `json:"job_id"`
	WorkerID     string    `json:"worker_id"`
	Name         string    `json:"name"`
	TaskCategory string    `json:"task_category"`
	Status       string    `json:"status"`
	StartedAt    time.Time `json:"started_at,omitempty"`
	EndedAt      time.Time `json:"ended_at,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type CostSnapshot struct {
	SnapshotID      string    `json:"snapshot_id"`
	TenantID        string    `json:"tenant_id"`
	EventID         string    `json:"event_id"`
	Provider        string    `json:"provider"`
	ModelID         string    `json:"model_id"`
	InputTokens     int       `json:"input_tokens"`
	OutputTokens    int       `json:"output_tokens"`
	CachedTokens    int       `json:"cached_tokens"`
	CostEstimateUSD *float64  `json:"cost_estimate_usd,omitempty"`
	Currency        string    `json:"currency"`
	IsDegraded      bool      `json:"is_degraded"`
	DegradedCode    string    `json:"degraded_code,omitempty"`
	CreatedAt       time.Time `json:"created_at"`
}

type IngestionResult struct {
	Event        TokenEvent      `json:"event"`
	CostSnapshot CostSnapshot    `json:"cost_snapshot"`
	Anomalies    []AnomalySignal `json:"anomalies"`
	Warnings     []Issue         `json:"warnings,omitempty"`
	Degraded     []Issue         `json:"degraded,omitempty"`
}

type WorkerBreakdown struct {
	WorkerID                        string   `json:"worker_id"`
	WorkerName                      string   `json:"worker_name"`
	EventCount                      int      `json:"event_count"`
	OutputCount                     int      `json:"output_count"`
	FailedOutputCount               int      `json:"failed_output_count"`
	TotalTokens                     int      `json:"total_tokens"`
	TotalCostUSD                    float64  `json:"total_cost_usd"`
	UnknownCostEventCount           int      `json:"unknown_cost_event_count"`
	AvgLatencyMs                    *float64 `json:"avg_latency_ms,omitempty"`
	AnomalyCount                    int      `json:"anomaly_count"`
	CostPerAcceptedOutputWithReview *float64 `json:"cost_per_accepted_output_with_review,omitempty"`
}

type CategoryBreakdown struct {
	TaskCategory string   `json:"task_category"`
	EventCount   int      `json:"event_count"`
	OutputCount  int      `json:"output_count"`
	TotalCostUSD float64  `json:"total_cost_usd"`
	AvgLatencyMs *float64 `json:"avg_latency_ms,omitempty"`
}

type CostDriver struct {
	Type         string  `json:"type"`
	Key          string  `json:"key"`
	Label        string  `json:"label"`
	TotalCostUSD float64 `json:"total_cost_usd"`
	EventCount    int    `json:"event_count"`
}

type ProductivitySummary struct {
	SummaryID                       string              `json:"summary_id"`
	TenantID                        string              `json:"tenant_id"`
	PeriodStart                     *time.Time          `json:"period_start,omitempty"`
	PeriodEnd                       *time.Time          `json:"period_end,omitempty"`
	GeneratedAt                     time.Time           `json:"generated_at"`
	TotalCostUSD                    float64             `json:"total_cost_usd"`
	KnownCostEventCount             int                 `json:"known_cost_event_count"`
	UnknownCostEventCount           int                 `json:"unknown_cost_event_count"`
	TotalEvents                     int                 `json:"total_events"`
	OutputCount                     int                 `json:"output_count"`
	AvgLatencyMs                    *float64            `json:"avg_latency_ms,omitempty"`
	AnomalyCount                    int                 `json:"anomaly_count"`
	CostPerAcceptedOutputWithReview *float64            `json:"cost_per_accepted_output_with_review,omitempty"`
	CostByWorker                    []WorkerBreakdown   `json:"cost_by_worker"`
	CostByCategory                  []CategoryBreakdown `json:"cost_by_category"`
	TopCostDrivers                  []CostDriver        `json:"top_cost_drivers"`
	Degraded                        []Issue             `json:"degraded,omitempty"`
}
