package ingestion

import (
	"context"
	"log"

	"github.com/Hardonian/TokenGoblin/internal/domain"
	"github.com/Hardonian/TokenGoblin/internal/storage"
)

type Engine struct {
	store   storage.Repository
	pricing map[string]domain.PricePoint
}

// Verify Engine implements Service
var _ Service = (*Engine)(nil)

func NewEngine(store storage.Repository) *Engine {
	// Seed static pricing for MVP
	pricing := make(map[string]domain.PricePoint)
	pricing["gemini-1.5-pro"] = domain.PricePoint{
		InputCostPerMillion:  3.50,
		OutputCostPerMillion: 10.50,
	}
	pricing["claude-3-opus"] = domain.PricePoint{
		InputCostPerMillion:  15.00,
		OutputCostPerMillion: 75.00,
	}
	return &Engine{
		store:   store,
		pricing: pricing,
	}
}

func (e *Engine) ProcessTokenEvent(ctx context.Context, event domain.TokenEvent) error {
	// 1. Get pricing for the model
	price, ok := e.pricing[event.ModelID]
	if !ok {
		log.Printf("Warning: Could not find pricing for model %s. Cost will be 0.", event.ModelID)
	} else {
		// Calculate precise cost based on per-1M tokens
		promptCost := float64(event.PromptTokens) * (price.InputCostPerMillion / 1000000.0)
		completionCost := float64(event.CompletionTokens) * (price.OutputCostPerMillion / 1000000.0)
		totalCost := promptCost + completionCost
		event.CostEstimateUSD = &totalCost
	}

	event.TotalTokens = event.PromptTokens + event.CompletionTokens

	// 2. Anomaly Check (Basic: Flag if TotalTokens > 50,000)
	if event.TotalTokens > 50000 {
		log.Printf("ANOMALY DETECTED: Tenant %s, Worker %s used %d tokens in a single event.", event.TenantID, event.WorkerID, event.TotalTokens)
		// Future: Fire domain.AnomalyEvent to an event bus or AnomalyRepository
	}

	// 3. Persist Event
	return e.store.SaveTokenEvent(ctx, event)
}

func (e *Engine) ProcessTaskCompletion(ctx context.Context, completion domain.TaskCompletion) error {
	// Not implemented in the new SQLite storage layer directly. Let's return nil.
	return nil
}
