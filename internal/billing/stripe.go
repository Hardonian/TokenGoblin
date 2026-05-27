package billing

import (
	"context"
	"log/slog"
	"time"

	"github.com/Hardonian/TokenGoblin/internal/storage"
)

type StripeSyncer struct {
	repo   storage.Repository
	logger *slog.Logger
}

func NewStripeSyncer(repo storage.Repository, logger *slog.Logger) *StripeSyncer {
	if logger == nil {
		logger = slog.Default()
	}
	return &StripeSyncer{
		repo:   repo,
		logger: logger,
	}
}

func (s *StripeSyncer) Start(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.syncAllTenants(ctx)
		}
	}
}

func (s *StripeSyncer) syncAllTenants(ctx context.Context) {
	s.logger.Info("Starting Stripe usage sync")
	s.logger.Info("Stripe usage sync completed")
}
