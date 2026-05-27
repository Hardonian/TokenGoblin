package ingestion

import (
	"context"
	"log/slog"
	"time"

	"github.com/Hardonian/TokenGoblin/internal/storage"
)

type RetentionWorker struct {
	repo          storage.Repository
	logger        *slog.Logger
	retentionDays int
}

func NewRetentionWorker(repo storage.Repository, logger *slog.Logger, retentionDays int) *RetentionWorker {
	if logger == nil {
		logger = slog.Default()
	}
	return &RetentionWorker{
		repo:          repo,
		logger:        logger,
		retentionDays: retentionDays,
	}
}

func (w *RetentionWorker) Start(ctx context.Context) {
	if w.retentionDays <= 0 {
		w.logger.Info("Data retention worker disabled (retentionDays <= 0)")
		return
	}

	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()

	// Run once on startup
	w.cleanup(ctx)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.cleanup(ctx)
		}
	}
}

func (w *RetentionWorker) cleanup(ctx context.Context) {
	w.logger.Info("Starting data retention cleanup", "retention_days", w.retentionDays)
	deleted, err := w.repo.DeleteOldEvents(ctx, w.retentionDays)
	if err != nil {
		w.logger.Error("Failed to delete old events", "error", err)
		return
	}
	w.logger.Info("Data retention cleanup completed", "deleted_events", deleted)
}
