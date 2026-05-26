package domain

import "time"

type OutputStatus string

const (
	OutputAccepted  OutputStatus = "accepted"
	OutputSucceeded OutputStatus = "succeeded"
	OutputFailed    OutputStatus = "failed"
	OutputRejected  OutputStatus = "rejected"
	OutputPending   OutputStatus = "pending"
)

// ExternalEstimate records caller-provided cost as untrusted reference data.
// The service never uses it as the internal cost estimate.
type ExternalEstimate struct {
	CostUSD  float64 `json:"cost_usd"`
	Currency string  `json:"currency"`
}

// TokenEvent represents a single atomic usage of an AI model.
type TokenEvent struct {
	EventID          string            `json:"event_id"`
	Timestamp        time.Time         `json:"timestamp"`
	CreatedAt        time.Time         `json:"created_at,omitempty"`
	TenantID         string            `json:"tenant_id,omitempty"`
	WorkerID         string            `json:"worker_id"`
	WorkerName       string            `json:"worker_name"`
	JobID            string            `json:"job_id,omitempty"`
	SessionID        string            `json:"session_id,omitempty"`
	RunID            string            `json:"run_id,omitempty"`
	ModelID          string            `json:"model_id"`
	Provider         string            `json:"provider"`
	PromptTokens     int               `json:"prompt_tokens"`
	CompletionTokens int               `json:"completion_tokens"`
	CachedTokens     int               `json:"cached_tokens"`
	InputTokens      int               `json:"input_tokens"`
	OutputTokens     int               `json:"output_tokens"`
	TotalTokens      int               `json:"total_tokens"`
	CostEstimateUSD  *float64          `json:"cost_estimate_usd,omitempty"`
	CostCurrency     string            `json:"cost_currency,omitempty"`
	CostIsDegraded   bool              `json:"cost_is_degraded"`
	CostDegradedCode string            `json:"cost_degraded_code,omitempty"`
	ExternalEstimate *ExternalEstimate `json:"external_estimate,omitempty"`
	LatencyMs        int               `json:"latency_ms"`
	TaskType         string            `json:"task_type,omitempty"`
	TaskCategory     string            `json:"task_category,omitempty"`
	OutputStatus     OutputStatus      `json:"output_status"`
	ReviewScore      *float64          `json:"review_score,omitempty"`
	Tags             map[string]string `json:"tags,omitempty"`
}
