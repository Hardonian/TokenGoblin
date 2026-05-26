package domain

import "time"

type Severity string

const (
	SeverityInfo     Severity = "info"
	SeverityLow      Severity = "low"
	SeverityMed      Severity = "medium"
	SeverityHigh     Severity = "high"
	SeverityCritical Severity = "critical"
)

type AnomalyType string

const (
	AnomalySpendSpike                AnomalyType = "spend_spike"
	AnomalyTokenSpike                AnomalyType = "token_spike"
	AnomalyLatencySpike              AnomalyType = "latency_spike"
	AnomalyUnknownModelPricing       AnomalyType = "unknown_model_pricing"
	AnomalyRepeatedFailedOutputs     AnomalyType = "repeated_failed_outputs"
	AnomalyHighCostPerAcceptedOutput AnomalyType = "high_cost_per_accepted_output"
)

// AnomalySignal represents a deterministic rule hit.
type AnomalySignal struct {
	AnomalyID      string                 `json:"anomaly_id"`
	TenantID       string                 `json:"tenant_id"`
	EventID        string                 `json:"event_id,omitempty"`
	WorkerID       string                 `json:"worker_id,omitempty"`
	DetectedAt     time.Time              `json:"detected_at"`
	Severity       Severity               `json:"severity"`
	Type           AnomalyType            `json:"type"`
	Description    string                 `json:"description"`
	ObservedValue  *float64               `json:"observed_value,omitempty"`
	ThresholdValue *float64               `json:"threshold_value,omitempty"`
	Details        map[string]interface{} `json:"details,omitempty"`
}
