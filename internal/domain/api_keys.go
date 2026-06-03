package domain

import "time"

type APIKey struct {
	KeyID      string     `json:"key_id"`
	TenantID   string     `json:"tenant_id"`
	Name       string     `json:"name"`
	Role       string     `json:"role"`
	KeyHash    string     `json:"-"`
	CreatedAt  time.Time  `json:"created_at"`
	LastUsedAt *time.Time `json:"last_used_at,omitempty"`
	IsRevoked  bool       `json:"is_revoked"`
}
