package domain

import "time"

type TaskStatus string

const (
	StatusSuccess  TaskStatus = "SUCCESS"
	StatusFailure  TaskStatus = "FAILURE"
	StatusDegraded TaskStatus = "DEGRADED"
	StatusAborted  TaskStatus = "ABORTED"
)

// TaskCompletion represents the operational outcome of an AI workflow.
type TaskCompletion struct {
	CompletionID       string            `json:"completion_id"`
	Timestamp          time.Time         `json:"timestamp"`
	TenantID           string            `json:"tenant_id"`
	WorkerID           string            `json:"worker_id"`
	SessionID          string            `json:"session_id"`
	TaskType           string            `json:"task_type"`
	Status             TaskStatus        `json:"status"`
	DurationMs         int               `json:"duration_ms"`
	TotalTokensUsed    int               `json:"total_tokens_used"`
	TotalCost          *float64          `json:"total_cost,omitempty"`
	OutputQualityScore *float64          `json:"output_quality_score,omitempty"`
	ErrorCode          string            `json:"error_code"`
	Tags               map[string]string `json:"tags"`
}
