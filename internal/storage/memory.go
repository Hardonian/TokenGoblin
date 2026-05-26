package storage

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/Hardonian/TokenGoblin/internal/domain"
)

// MemoryStore implements the storage interfaces in-memory.
type MemoryStore struct {
	mu          sync.RWMutex
	events      []domain.TokenEvent
	completions []domain.TaskCompletion
	pricing     map[string]domain.PricePoint
	snapshots   map[string]domain.WorkerSnapshot
}

var _ EventRepository = (*MemoryStore)(nil)
var _ PricingRepository = (*MemoryStore)(nil)
var _ SnapshotRepository = (*MemoryStore)(nil)

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		events:      make([]domain.TokenEvent, 0),
		completions: make([]domain.TaskCompletion, 0),
		pricing:     make(map[string]domain.PricePoint),
		snapshots:   make(map[string]domain.WorkerSnapshot),
	}
}

// SeedPricing adds standard pricing for testing.
func (s *MemoryStore) SeedPricing() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.pricing["gemini-1.5-pro"] = domain.PricePoint{
		ModelID:             "gemini-1.5-pro",
		PromptCostPer1k:     0.0035,
		CompletionCostPer1k: 0.0105,
	}
	s.pricing["claude-3-opus"] = domain.PricePoint{
		ModelID:             "claude-3-opus",
		PromptCostPer1k:     0.015,
		CompletionCostPer1k: 0.075,
	}
}

func (s *MemoryStore) SaveTokenEvent(ctx context.Context, event domain.TokenEvent) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.events = append(s.events, event)
	return nil
}

func (s *MemoryStore) SaveTaskCompletion(ctx context.Context, completion domain.TaskCompletion) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.completions = append(s.completions, completion)
	return nil
}

func (s *MemoryStore) GetTokenEvents(ctx context.Context, tenantID string, start, end time.Time) ([]domain.TokenEvent, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []domain.TokenEvent
	for _, ev := range s.events {
		if ev.TenantID == tenantID {
			// Include events that don't have a timestamp check for now, or match it
			if (ev.Timestamp.After(start) || ev.Timestamp.Equal(start)) && ev.Timestamp.Before(end) {
				result = append(result, ev)
			}
		}
	}
	return result, nil
}

func (s *MemoryStore) GetWorkerCompletions(ctx context.Context, tenantID, workerID string) ([]domain.TaskCompletion, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []domain.TaskCompletion
	for _, comp := range s.completions {
		if comp.TenantID == tenantID && comp.WorkerID == workerID {
			result = append(result, comp)
		}
	}
	return result, nil
}

func (s *MemoryStore) GetPricePoint(ctx context.Context, modelID string, at time.Time) (domain.PricePoint, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if p, ok := s.pricing[modelID]; ok {
		return p, nil
	}
	return domain.PricePoint{}, errors.New("pricing not found")
}

func (s *MemoryStore) SaveWorkerSnapshot(ctx context.Context, snapshot domain.WorkerSnapshot) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	key := snapshot.WorkerID + "_" + snapshot.PeriodStart.String()
	s.snapshots[key] = snapshot
	return nil
}
