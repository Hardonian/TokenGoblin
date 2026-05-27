package audit

import (
	"context"
	"log/slog"
	"time"

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
	return nil
}
