package storage

import (
	"context"
	"time"

	"github.com/Hardonian/TokenGoblin/internal/domain"
)

// EventRepository defines how to persist operational events.
type EventRepository interface {
	SaveTokenEvent(ctx context.Context, event domain.TokenEvent) error
	SaveTaskCompletion(ctx context.Context, completion domain.TaskCompletion) error
}

// PricingRepository handles versioned model pricing.
type PricingRepository interface {
	GetPricePoint(ctx context.Context, modelID string, at time.Time) (domain.PricePoint, error)
}

// SnapshotRepository handles worker aggregations.
type SnapshotRepository interface {
	SaveWorkerSnapshot(ctx context.Context, snapshot domain.WorkerSnapshot) error
}
