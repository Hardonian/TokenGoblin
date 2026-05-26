package domain

import "time"

// WorkerSnapshot represents an aggregated view of a worker's efficiency over time.
type WorkerSnapshot struct {
	WorkerID           string    `json:"worker_id"`
	PeriodStart        time.Time `json:"period_start"`
	PeriodEnd          time.Time `json:"period_end"`
	TotalTokens        int       `json:"total_tokens"`
	TotalCost          float64   `json:"total_cost"`
	TasksCompleted     int       `json:"tasks_completed"`
	EfficiencyScore    float64   `json:"efficiency_score"`
	DegradedStateFlags []string  `json:"degraded_state_flags"`
}
