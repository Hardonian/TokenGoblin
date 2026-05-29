package domain

import "time"

type AnalysisIssue struct {
	Code     string   `json:"code"`
	Severity Severity `json:"severity"`
	Message  string   `json:"message"`
	Evidence string   `json:"evidence,omitempty"`
}

type AnalysisEvidence struct {
	Type   string  `json:"type"`
	Label  string  `json:"label"`
	Value  string  `json:"value,omitempty"`
	Metric float64 `json:"metric,omitempty"`
}

type OutputAnalysis struct {
	AnalysisID      string             `json:"analysis_id"`
	TenantID        string             `json:"tenant_id"`
	EventID         string             `json:"event_id"`
	WorkerID        string             `json:"worker_id"`
	AnalyzedAt      time.Time          `json:"analyzed_at"`
	EfficiencyScore int                `json:"efficiency_score"`
	GoblinScore     int                `json:"goblin_score"`
	Issues          []AnalysisIssue    `json:"issues"`
	Recommendations []string           `json:"recommendations"`
	Evidence        []AnalysisEvidence `json:"evidence"`
	Degraded        []Issue            `json:"degraded,omitempty"`
}

type WorkerReview struct {
	Worker                 WorkerBreakdown `json:"worker"`
	LatestOutput           *TokenEvent     `json:"latest_output,omitempty"`
	LatestAnalysis         *OutputAnalysis `json:"latest_analysis,omitempty"`
	WasteSignals           []AnalysisIssue `json:"waste_signals"`
	RecommendedConstraints []string        `json:"recommended_constraints"`
	Degraded               []Issue         `json:"degraded,omitempty"`
}
