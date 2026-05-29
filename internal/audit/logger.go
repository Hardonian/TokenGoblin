package audit

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"time"

	"github.com/Hardonian/TokenGoblin/internal/domain"
	"github.com/Hardonian/TokenGoblin/internal/storage"
)

type EventType string

const (
	EventTenantDeleted  EventType = "tenant.deleted"
	EventPricingChanged EventType = "tenant.pricing_changed"
	EventAPIKeyCreated  EventType = "apikey.created"
	EventAPIKeyRevoked  EventType = "apikey.revoked"
	EventDataExported   EventType = "tenant.data_exported"
)

type Event struct {
	ID        string                 `json:"id"`
	TenantID  string                 `json:"tenant_id"`
	Type      EventType              `json:"type"`
	Actor     string                 `json:"actor"`
	Metadata  map[string]interface{} `json:"metadata"`
	Timestamp time.Time              `json:"timestamp"`
}

type Logger struct {
	repo   storage.Repository
	logger *slog.Logger
}

func NewLogger(repo storage.Repository, logger *slog.Logger) *Logger {
	if logger == nil {
		logger = slog.Default()
	}
	return &Logger{
		repo:   repo,
		logger: logger,
	}
}

func (l *Logger) LogEvent(ctx context.Context, event Event) error {
	// Stub implementation: log to stdout for now.
	// In a real implementation, we would write this to an audit_logs table or a stream like Kafka/Kinesis.
	l.logger.Info("AUDIT EVENT",
		"tenant_id", event.TenantID,
		"type", event.Type,
		"actor", event.Actor,
		"metadata", event.Metadata,
	)

	return l.repo.SaveAuditEvent(ctx, domainAuditEvent(event))
}

func domainAuditEvent(event Event) domain.AuditEvent {
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now().UTC()
	}
	if event.ID == "" {
		event.ID = "aud_" + randomHex(12)
	}
	return domain.AuditEvent{
		EventID:   event.ID,
		TenantID:  event.TenantID,
		Type:      string(event.Type),
		Actor:     event.Actor,
		Metadata:  event.Metadata,
		Timestamp: event.Timestamp,
	}
}

func randomHex(bytes int) string {
	buffer := make([]byte, bytes)
	if _, err := rand.Read(buffer); err != nil {
		return hex.EncodeToString([]byte(time.Now().UTC().Format(time.RFC3339Nano)))
	}
	return hex.EncodeToString(buffer)
}
