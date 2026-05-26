package domain

import "time"

// WorkerSnapshot represents an aggregated view of a worker's efficiency over time.
type WorkerSnapshot struct {
	TenantID           string    `json:"tenant_id"`
	WorkerID           string    `json:"worker_id"`
	WorkerName         string    `json:"worker_name"`
	PeriodStart        time.Time `json:"period_start"`
	PeriodEnd          time.Time `json:"period_end"`
	TotalTokens        int       `json:"total_tokens"`
	TotalCost          float64   `json:"total_cost"`
	TasksCompleted     int       `json:"tasks_completed"`
	CostPerOutput      *float64  `json:"cost_per_output,omitempty"`
	AvgLatencyMs       *float64  `json:"avg_latency_ms,omitempty"`
	DegradedStateFlags []string  `json:"degraded_state_flags"`
}
