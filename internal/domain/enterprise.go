package domain

import "time"

const (
	RoleOwner   = "owner"
	RoleAdmin   = "admin"
	RoleAnalyst = "analyst"
	RoleIngest  = "ingest"
	RoleViewer  = "viewer"
)

type TenantMember struct {
	TenantID  string    `json:"tenant_id"`
	SubjectID string    `json:"subject_id"`
	Email     string    `json:"email,omitempty"`
	Role      string    `json:"role"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type AuditEvent struct {
	EventID   string                 `json:"event_id"`
	TenantID  string                 `json:"tenant_id"`
	Type      string                 `json:"type"`
	Actor     string                 `json:"actor"`
	Resource  string                 `json:"resource,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
}

type RecommendationState struct {
	TenantID         string    `json:"tenant_id"`
	RecommendationID string    `json:"recommendation_id"`
	Status           string    `json:"status"`
	Actor            string    `json:"actor,omitempty"`
	Note             string    `json:"note,omitempty"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

type RecommendationStateUpdate struct {
	Status string `json:"status"`
	Note   string `json:"note,omitempty"`
}
