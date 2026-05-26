package ingestion

import (
	"context"
	"log"

	"github.com/Hardonian/TokenGoblin/internal/domain"
	"github.com/Hardonian/TokenGoblin/internal/storage"
)

type Engine struct {
	eventsStore  storage.EventRepository
	pricingStore storage.PricingRepository
}

// Verify Engine implements Service
var _ Service = (*Engine)(nil)

func NewEngine(events storage.EventRepository, pricing storage.PricingRepository) *Engine {
	return &Engine{
		eventsStore:  events,
		pricingStore: pricing,
	}
}

func (e *Engine) ProcessTokenEvent(ctx context.Context, event domain.TokenEvent) error {
	// 1. Get pricing for the model
	price, err := e.pricingStore.GetPricePoint(ctx, event.ModelID, event.Timestamp)
	if err != nil {
		log.Printf("Warning: Could not find pricing for model %s. Cost will be 0.", event.ModelID)
	} else {
		// Calculate precise cost based on per-1k tokens
		promptCost := float64(event.PromptTokens) * (price.PromptCostPer1k / 1000.0)
		completionCost := float64(event.CompletionTokens) * (price.CompletionCostPer1k / 1000.0)
		event.TotalCost = promptCost + completionCost
	}

	event.TotalTokens = event.PromptTokens + event.CompletionTokens

	// 2. Anomaly Check (Basic: Flag if TotalTokens > 50,000)
	if event.TotalTokens > 50000 {
		log.Printf("ANOMALY DETECTED: Tenant %s, Worker %s used %d tokens in a single event.", event.TenantID, event.WorkerID, event.TotalTokens)
		// Future: Fire domain.AnomalyEvent to an event bus or AnomalyRepository
	}

	// 3. Persist Event
	return e.eventsStore.SaveTokenEvent(ctx, event)
}

func (e *Engine) ProcessTaskCompletion(ctx context.Context, completion domain.TaskCompletion) error {
	return e.eventsStore.SaveTaskCompletion(ctx, completion)
}
