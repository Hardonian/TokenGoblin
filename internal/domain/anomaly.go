package domain

import "time"

type Severity string

const (
	SeverityLow      Severity = "LOW"
	SeverityMed      Severity = "MED"
	SeverityHigh     Severity = "HIGH"
	SeverityCritical Severity = "CRITICAL"
)

// AnomalyEvent represents an automatically detected inefficiency or waste signature.
type AnomalyEvent struct {
	AnomalyID            string   `json:"anomaly_id"`
	TenantID             string   `json:"tenant_id"`
	WorkerID             string   `json:"worker_id"`
	DetectedAt           time.Time `json:"detected_at"`
	Severity             Severity `json:"severity"`
	Signature            string   `json:"signature"`
	Description          string   `json:"description"`
	WastedTokensEstimate int      `json:"wasted_tokens_estimate"`
}
