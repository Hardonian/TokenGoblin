package domain

import "time"

// ═══════════════════════════════════════════════════════════
// Layer 4 — Intelligence Engine Types
// ═══════════════════════════════════════════════════════════

// WorkerType distinguishes human-initiated from agent-driven work.
type WorkerType string

const (
	WorkerTypeHuman    WorkerType = "human"
	WorkerTypeAgent    WorkerType = "agent"
	WorkerTypePipeline WorkerType = "pipeline"
	WorkerTypeTool     WorkerType = "tool"
)

// PromptFingerprint tracks unique prompts for dedup and waste analysis.
type PromptFingerprint struct {
	FingerprintID         string    `json:"fingerprint_id"`
	TenantID              string    `json:"tenant_id"`
	PromptHash            string    `json:"prompt_hash"`
	FirstSeenAt           time.Time `json:"first_seen_at"`
	LastSeenAt            time.Time `json:"last_seen_at"`
	OccurrenceCount       int       `json:"occurrence_count"`
	AvgCostUSD            float64   `json:"avg_cost_usd"`
	AvgOutputTokens       int       `json:"avg_output_tokens"`
	AvgAcceptanceRate     float64   `json:"avg_acceptance_rate"`
	CanonicalTaskCategory string    `json:"canonical_task_category,omitempty"`
	IsWasteful            bool      `json:"is_wasteful"`
	WasteReason           string    `json:"waste_reason,omitempty"`
	WasteScore            float64   `json:"waste_score"`
	TotalCostUSD          float64   `json:"total_cost_usd"`
	UniqueWorkers         int       `json:"unique_workers"`
}

// WorkflowTrace tracks a multi-step agent interaction from start to finish.
type WorkflowTrace struct {
	TraceID        string     `json:"trace_id"`
	TenantID       string     `json:"tenant_id"`
	RootJobID      string     `json:"root_job_id"`
	WorkerID       string     `json:"worker_id"`
	StartedAt      time.Time  `json:"started_at"`
	EndedAt        *time.Time `json:"ended_at,omitempty"`
	TotalEvents    int        `json:"total_events"`
	TotalCostUSD   float64    `json:"total_cost_usd"`
	TotalTokens    int        `json:"total_tokens"`
	FinalOutcome   string     `json:"final_outcome,omitempty"`
	TotalLatencyMs int        `json:"total_latency_ms"`
	StepCount      int        `json:"step_count"`
}

// WorkflowStep represents a single step within a workflow trace.
type WorkflowStep struct {
	StepID       string   `json:"step_id"`
	TraceID      string   `json:"trace_id"`
	TenantID     string   `json:"tenant_id"`
	EventID      string   `json:"event_id,omitempty"`
	StepOrder    int      `json:"step_order"`
	StepType     string   `json:"step_type,omitempty"`
	ModelID      string   `json:"model_id,omitempty"`
	CostUSD      *float64 `json:"cost_usd,omitempty"`
	Tokens       int      `json:"tokens"`
	LatencyMs    int      `json:"latency_ms"`
	OutputStatus string   `json:"output_status,omitempty"`
}

// WasteReport summarizes waste detection results for a tenant.
type WasteReport struct {
	TenantID         string              `json:"tenant_id"`
	GeneratedAt      time.Time           `json:"generated_at"`
	TotalWasteUSD    float64             `json:"total_waste_usd"`
	WastefulPrompts  []PromptFingerprint `json:"wasteful_prompts"`
	DuplicatePrompts []DuplicateCluster  `json:"duplicate_prompts"`
	ZombieAgents     []ZombieAgent       `json:"zombie_agents"`
	CostLeaks        []CostLeak          `json:"cost_leaks"`
}

// DuplicateCluster groups prompts used identically across workers/teams.
type DuplicateCluster struct {
	PromptHash       string   `json:"prompt_hash"`
	OccurrenceCount  int      `json:"occurrence_count"`
	UniqueWorkers    int      `json:"unique_workers"`
	TotalCostUSD     float64  `json:"total_cost_usd"`
	RedundantCostUSD float64  `json:"redundant_cost_usd"`
	WorkerIDs        []string `json:"worker_ids"`
	TaskCategory     string   `json:"task_category,omitempty"`
}

// ZombieAgent identifies an agent running with low business value.
type ZombieAgent struct {
	WorkerID       string  `json:"worker_id"`
	WorkerName     string  `json:"worker_name"`
	EventCount7d   int     `json:"event_count_7d"`
	TotalCost7dUSD float64 `json:"total_cost_7d_usd"`
	AcceptanceRate float64 `json:"acceptance_rate"`
	OutcomeCount   int     `json:"outcome_count"`
	ZombieScore    float64 `json:"zombie_score"`
	Reason         string  `json:"reason"`
}

// CostLeak identifies a pattern of silent, invisible spending.
type CostLeak struct {
	Type        string  `json:"type"`
	Description string  `json:"description"`
	CostUSD     float64 `json:"cost_usd"`
	EventCount  int     `json:"event_count"`
	Severity    string  `json:"severity"`
	WorkerID    string  `json:"worker_id,omitempty"`
	ModelID     string  `json:"model_id,omitempty"`
}

// CostLeakType constants
const (
	CostLeakRetryStorm      = "retry_storm"
	CostLeakContextPadding  = "context_padding"
	CostLeakCacheMiss       = "cache_miss"
	CostLeakOffHours        = "off_hours_spending"
	CostLeakDuplicatePrompt = "duplicate_prompt"
)

// HallucinationCell represents a single cell in the hallucination heatmap.
type HallucinationCell struct {
	ModelID      string  `json:"model_id"`
	TaskCategory string  `json:"task_category"`
	HourOfDay    int     `json:"hour_of_day,omitempty"`
	WorkerID     string  `json:"worker_id,omitempty"`
	FailureCount int     `json:"failure_count"`
	TotalCount   int     `json:"total_count"`
	FailureRate  float64 `json:"failure_rate"`
	TotalCostUSD float64 `json:"total_cost_usd"`
}
