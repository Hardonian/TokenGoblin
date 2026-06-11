package domain

import "time"

// ═══════════════════════════════════════════════════════════
// Layer 6 — Governance Engine Types
// ═══════════════════════════════════════════════════════════

// PolicyType defines the kind of governance policy.
type PolicyType string

const (
	PolicyBudgetLimit      PolicyType = "budget_limit"
	PolicyModelAllowlist   PolicyType = "model_allowlist"
	PolicyModelBlocklist   PolicyType = "model_blocklist"
	PolicyPIIBlock         PolicyType = "pii_block"
	PolicyRateLimit        PolicyType = "rate_limit"
	PolicyApprovalRequired PolicyType = "approval_required"
	PolicyCostCeiling      PolicyType = "cost_ceiling"
)

// GovernancePolicy defines a tenant-scoped governance rule.
type GovernancePolicy struct {
	PolicyID   string     `json:"policy_id"`
	TenantID   string     `json:"tenant_id"`
	Name       string     `json:"name"`
	Type       PolicyType `json:"type"`
	ConfigJSON string     `json:"config_json"`
	IsActive   bool       `json:"is_active"`
	CreatedBy  string     `json:"created_by,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
}

// BudgetLimitConfig is the JSON config for budget_limit policies.
type BudgetLimitConfig struct {
	LimitUSD          float64 `json:"limit_usd"`
	Period            string  `json:"period"`
	ScopeType         string  `json:"scope_type"`
	ScopeID           string  `json:"scope_id,omitempty"`
	AlertThresholdPct float64 `json:"alert_threshold_pct"`
	EnforceHardLimit  bool    `json:"enforce_hard_limit"`
}

// ModelAllowlistConfig is the JSON config for model_allowlist policies.
type ModelAllowlistConfig struct {
	AllowedModels []string `json:"allowed_models"`
	ScopeType     string   `json:"scope_type"`
	ScopeID       string   `json:"scope_id,omitempty"`
}

// PolicyViolation records a governance policy breach.
type PolicyViolation struct {
	ViolationID    string     `json:"violation_id"`
	TenantID       string     `json:"tenant_id"`
	PolicyID       string     `json:"policy_id"`
	EventID        string     `json:"event_id,omitempty"`
	WorkerID       string     `json:"worker_id,omitempty"`
	ViolationType  string     `json:"violation_type"`
	Severity       Severity   `json:"severity"`
	Description    string     `json:"description"`
	MetadataJSON   string     `json:"metadata_json,omitempty"`
	DetectedAt     time.Time  `json:"detected_at"`
	ResolvedAt     *time.Time `json:"resolved_at,omitempty"`
	ResolvedBy     string     `json:"resolved_by,omitempty"`
	ResolutionNote string     `json:"resolution_note,omitempty"`
}

// ComplianceReport summarizes governance status for a tenant.
type ComplianceReport struct {
	TenantID           string    `json:"tenant_id"`
	GeneratedAt        time.Time `json:"generated_at"`
	ActivePolicies     int       `json:"active_policies"`
	TotalViolations30d int       `json:"total_violations_30d"`
	OpenViolations     int       `json:"open_violations"`
	ResolvedViolations int       `json:"resolved_violations"`
	PIIDetections30d   int       `json:"pii_detections_30d"`
	ShadowAIEvents30d  int       `json:"shadow_ai_events_30d"`
	ComplianceScore    int       `json:"compliance_score"`
}
