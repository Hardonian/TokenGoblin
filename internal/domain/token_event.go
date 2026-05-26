package domain

import "time"

// TokenEvent represents a single atomic usage of an AI model.
type TokenEvent struct {
	EventID          string            `json:"event_id"`
	Timestamp        time.Time         `json:"timestamp"`
	TenantID         string            `json:"tenant_id"`
	WorkerID         string            `json:"worker_id"`
	SessionID        string            `json:"session_id"`
	ModelID          string            `json:"model_id"`
	Provider         string            `json:"provider"`
	PromptTokens     int               `json:"prompt_tokens"`
	CompletionTokens int               `json:"completion_tokens"`
	TotalTokens      int               `json:"total_tokens"`
	TotalCost        float64           `json:"total_cost"`
	LatencyMs        int               `json:"latency_ms"`
	TaskType         string            `json:"task_type"`
	Tags             map[string]string `json:"tags"`
}
