package domain

import "time"

// ═══════════════════════════════════════════════════════════
// Agent Management — Agentic Future Types
// ═══════════════════════════════════════════════════════════

// AgentStatus represents the lifecycle state of an agent.
type AgentStatus string

const (
	AgentStatusActive    AgentStatus = "active"
	AgentStatusPaused    AgentStatus = "paused"
	AgentStatusRetired   AgentStatus = "retired"
	AgentStatusDegraded  AgentStatus = "degraded"
)

// AgentFramework identifies the agent's orchestration framework.
type AgentFramework string

const (
	AgentFrameworkCrewAI         AgentFramework = "crewai"
	AgentFrameworkAutoGen        AgentFramework = "autogen"
	AgentFrameworkLangGraph      AgentFramework = "langgraph"
	AgentFrameworkSemanticKernel AgentFramework = "semantic_kernel"
	AgentFrameworkCustom         AgentFramework = "custom"
)

// Agent represents a registered AI agent with metadata.
type Agent struct {
	AgentID          string         `json:"agent_id"`
	TenantID         string         `json:"tenant_id"`
	Name             string         `json:"name"`
	Description      string         `json:"description,omitempty"`
	OwnerID          string         `json:"owner_id,omitempty"`
	AgentType        WorkerType     `json:"agent_type"`
	Framework        AgentFramework `json:"framework,omitempty"`
	Status           AgentStatus    `json:"status"`
	BudgetUSD        *float64       `json:"budget_usd,omitempty"`
	BudgetPeriod     string         `json:"budget_period,omitempty"`
	SLALatencyMs     *int           `json:"sla_latency_ms,omitempty"`
	SLASuccessRate   *float64       `json:"sla_success_rate,omitempty"`
	CreatedAt        time.Time      `json:"created_at"`
	UpdatedAt        time.Time      `json:"updated_at"`
	RetiredAt        *time.Time     `json:"retired_at,omitempty"`
	RetirementReason string         `json:"retirement_reason,omitempty"`
}

// EfficiencyGrade classifies agent performance.
type EfficiencyGrade string

const (
	GradeA EfficiencyGrade = "A"
	GradeB EfficiencyGrade = "B"
	GradeC EfficiencyGrade = "C"
	GradeD EfficiencyGrade = "D"
	GradeF EfficiencyGrade = "F"
)

// AgentRecommendation classifies what action to take on an agent.
type AgentRecommendation string

const (
	AgentRecContinue AgentRecommendation = "continue"
	AgentRecOptimize AgentRecommendation = "optimize"
	AgentRecRetire   AgentRecommendation = "retire"
)

// AgentPerformanceReview is a periodic assessment of an agent's value.
type AgentPerformanceReview struct {
	ReviewID        string              `json:"review_id"`
	AgentID         string              `json:"agent_id"`
	TenantID        string              `json:"tenant_id"`
	PeriodStart     time.Time           `json:"period_start"`
	PeriodEnd       time.Time           `json:"period_end"`
	EventCount      int                 `json:"event_count"`
	TotalCostUSD    float64             `json:"total_cost_usd"`
	AcceptanceRate  float64             `json:"acceptance_rate"`
	AvgLatencyMs    float64             `json:"avg_latency_ms"`
	CostPerOutcome  float64             `json:"cost_per_outcome"`
	SLAViolations   int                 `json:"sla_violations"`
	EfficiencyGrade EfficiencyGrade     `json:"efficiency_grade"`
	Recommendation  AgentRecommendation `json:"recommendation"`
	GeneratedAt     time.Time           `json:"generated_at"`
}

// ExecutiveScorecard is the AI maturity dashboard for leadership.
type ExecutiveScorecard struct {
	TenantID           string    `json:"tenant_id"`
	GeneratedAt        time.Time `json:"generated_at"`
	AIMaturityScore    int       `json:"ai_maturity_score"`
	AIEffectivenessIdx float64   `json:"ai_effectiveness_index"`
	AIAdoptionVelocity float64   `json:"ai_adoption_velocity"`
	AIROIIndex         float64   `json:"ai_roi_index"`
	TotalSpend30d      float64   `json:"total_spend_30d"`
	TotalEvents30d     int       `json:"total_events_30d"`
	ActiveWorkers      int       `json:"active_workers"`
	ActiveAgents       int       `json:"active_agents"`
	AvgAcceptanceRate  float64   `json:"avg_acceptance_rate"`
	CostPerOutcome     float64   `json:"cost_per_outcome"`
	WasteDetected30d   float64   `json:"waste_detected_30d"`
	SavingsRealized30d float64   `json:"savings_realized_30d"`
	TopRecommendations []string  `json:"top_recommendations"`
}
