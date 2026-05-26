package ingestion

import (
	"context"

	"github.com/Hardonian/TokenGoblin/internal/domain"
)

// Service defines the interface for handling incoming events and completions.
type Service interface {
	// ProcessTokenEvent validates and ingests a single token usage event.
	ProcessTokenEvent(ctx context.Context, event domain.TokenEvent) error
	
	// ProcessTaskCompletion validates and ingests a task completion event.
	ProcessTaskCompletion(ctx context.Context, completion domain.TaskCompletion) error
}
