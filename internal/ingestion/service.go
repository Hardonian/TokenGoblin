package ingestion

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/Hardonian/TokenGoblin/internal/anomaly"
	"github.com/Hardonian/TokenGoblin/internal/cost"
	"github.com/Hardonian/TokenGoblin/internal/domain"
	"github.com/Hardonian/TokenGoblin/internal/moat"
	"github.com/Hardonian/TokenGoblin/internal/productivity"
	"github.com/Hardonian/TokenGoblin/internal/storage"
)

var tenantIDPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_-]{1,79}$`)

type Service interface {
	IngestTokenEvent(ctx context.Context, tenantID string, event domain.TokenEvent) (domain.IngestionResult, error)
	IngestTokenEventBatch(ctx context.Context, tenantID string, events []domain.TokenEvent) ([]domain.IngestionResult, error)
	Overview(ctx context.Context, tenantID string) (domain.ProductivitySummary, error)
	Workers(ctx context.Context, tenantID string) ([]domain.WorkerBreakdown, error)
	Anomalies(ctx context.Context, tenantID string, limit int) ([]domain.AnomalySignal, error)
	RecentEvents(ctx context.Context, tenantID string, limit int) ([]domain.TokenEvent, error)
	RecentEventsBefore(ctx context.Context, tenantID string, before time.Time, limit int) ([]domain.TokenEvent, error)
	Recommendations(ctx context.Context, tenantID string) ([]domain.RoutingRecommendation, error)
}

type ExecutionService struct {
	repo       storage.Repository
	pricing    cost.Registry
	thresholds anomaly.Thresholds
	now        func() time.Time
	eventQueue chan domain.TokenEvent
}

func NewService(repo storage.Repository, pricing cost.Registry) *ExecutionService {
	return &ExecutionService{
		repo:       repo,
		pricing:    pricing,
		thresholds: anomaly.DefaultThresholds(),
		now:        func() time.Time { return time.Now().UTC() },
		eventQueue: make(chan domain.TokenEvent, 10000),
	}
}

// StartWorker starts a background goroutine to process ingested events.
func (s *ExecutionService) StartWorker(ctx context.Context) {
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case event := <-s.eventQueue:
				s.processEvent(ctx, event)
			}
		}
	}()
}

func (s *ExecutionService) processEvent(ctx context.Context, normalized domain.TokenEvent) {
	// Retry loop for DB locks/unavailability
	for retries := 0; retries < 3; retries++ {
		err := s.tryProcessEvent(ctx, normalized)
		if err == nil {
			return // Success
		}
		if errors.Is(err, storage.ErrUnavailable) {
			time.Sleep(time.Millisecond * 100 * time.Duration(retries+1))
			continue
		}
		// Some other error, log or ignore
		break
	}
}

func (s *ExecutionService) tryProcessEvent(ctx context.Context, normalized domain.TokenEvent) error {
	// Calculate cost synchronously since it is purely in-memory
	costResult := s.pricing.Calculate(normalized)
	normalized.CostEstimateUSD = costResult.CostEstimateUSD
	normalized.CostCurrency = costResult.Currency
	normalized.CostIsDegraded = costResult.Status == cost.StatusDegraded
	normalized.CostDegradedCode = costResult.DegradedCode

	prior, err := s.repo.ListTokenEventsBefore(ctx, normalized.TenantID, normalized.Timestamp, 100)
	if err != nil && !errors.Is(err, storage.ErrUnavailable) {
		return err
	}
	if errors.Is(err, storage.ErrUnavailable) {
		return err
	}

	snapshot := domain.CostSnapshot{
		SnapshotID:      normalized.EventID + ":cost",
		TenantID:        normalized.TenantID,
		EventID:         normalized.EventID,
		Provider:        normalized.Provider,
		ModelID:         normalized.ModelID,
		InputTokens:     normalized.InputTokens,
		OutputTokens:    normalized.OutputTokens,
		CachedTokens:    normalized.CachedTokens,
		CostEstimateUSD: normalized.CostEstimateUSD,
		Currency:        normalized.CostCurrency,
		IsDegraded:      normalized.CostIsDegraded,
		DegradedCode:    normalized.CostDegradedCode,
		CreatedAt:       normalized.CreatedAt,
	}

	signals := anomaly.Detect(normalized, prior, s.now(), s.thresholds)

	if err := s.repo.SaveTokenEvent(ctx, normalized); err != nil {
		return err
	}
	if err := s.repo.SaveCostSnapshot(ctx, snapshot); err != nil {
		return err
	}
	for _, signal := range signals {
		if err := s.repo.SaveAnomalySignal(ctx, signal); err != nil {
			return err
		}
	}
	return nil
}

func (s *ExecutionService) WithClock(clock func() time.Time) *ExecutionService {
	if clock != nil {
		s.now = clock
	}
	return s
}

func (s *ExecutionService) IngestTokenEvent(ctx context.Context, tenantID string, event domain.TokenEvent) (domain.IngestionResult, error) {
	normalized, warnings, err := s.normalize(tenantID, event)
	if err != nil {
		return domain.IngestionResult{}, err
	}

	costResult := s.pricing.Calculate(normalized)
	normalized.CostEstimateUSD = costResult.CostEstimateUSD
	normalized.CostCurrency = costResult.Currency
	normalized.CostIsDegraded = costResult.Status == cost.StatusDegraded
	normalized.CostDegradedCode = costResult.DegradedCode

	select {
	case s.eventQueue <- normalized:
		// Buffered successfully
	default:
		// Queue is full, return error
		return domain.IngestionResult{}, errors.New("ingestion buffer full")
	}

	result := domain.IngestionResult{
		Event:    normalized,
		Warnings: append(warnings, s.pricing.Diagnostics()...),
	}
	if normalized.CostIsDegraded {
		result.Degraded = append(result.Degraded, domain.Issue{
			Code:    normalized.CostDegradedCode,
			Message: "Internal cost estimate is unavailable for this provider/model.",
		})
	}
	result.Degraded = append(result.Degraded, domain.Issue{
		Code:    "buffered",
		Message: "Event buffered for asynchronous processing.",
	})

	return result, nil
}

func (s *ExecutionService) IngestTokenEventBatch(ctx context.Context, tenantID string, events []domain.TokenEvent) ([]domain.IngestionResult, error) {
	if err := validateTenantID(tenantID); err != nil {
		return nil, err
	}
	
	results := make([]domain.IngestionResult, 0, len(events))
	
	// A simple approach is just calling the single item method for each item
	// In a real enterprise app with massive batches, you would want bulk DB ops and bulk queuing
	for _, event := range events {
		res, err := s.IngestTokenEvent(ctx, tenantID, event)
		if err != nil {
			var validationErr ValidationError
			if errors.As(err, &validationErr) {
				// We can return a result with error status instead of failing the whole batch
				res.Event = event
				res.Degraded = append(res.Degraded, validationErr.Issues...)
				res.Warnings = append(res.Warnings, domain.Issue{Code: "validation_failed", Message: "Event failed validation."})
			} else {
				// For structural errors (queue full), we might fail the whole batch
				return nil, err
			}
		}
		results = append(results, res)
	}
	return results, nil
}

func (s *ExecutionService) Overview(ctx context.Context, tenantID string) (domain.ProductivitySummary, error) {
	if err := validateTenantID(tenantID); err != nil {
		return domain.ProductivitySummary{}, err
	}
	events, err := s.repo.ListTokenEvents(ctx, tenantID, 500)
	if err != nil {
		return domain.ProductivitySummary{}, err
	}
	signals, err := s.repo.ListAnomalySignals(ctx, tenantID, 500)
	if err != nil {
		return domain.ProductivitySummary{}, err
	}
	summary := productivity.BuildSummary(tenantID, events, signals, s.now())
	if len(events) > 0 {
		_ = s.repo.SaveProductivitySummary(ctx, summary)
	}
	return summary, nil
}

func (s *ExecutionService) Workers(ctx context.Context, tenantID string) ([]domain.WorkerBreakdown, error) {
	summary, err := s.Overview(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	return summary.CostByWorker, nil
}

func (s *ExecutionService) Anomalies(ctx context.Context, tenantID string, limit int) ([]domain.AnomalySignal, error) {
	if err := validateTenantID(tenantID); err != nil {
		return nil, err
	}
	return s.repo.ListAnomalySignals(ctx, tenantID, limit)
}

func (s *ExecutionService) RecentEvents(ctx context.Context, tenantID string, limit int) ([]domain.TokenEvent, error) {
	if err := validateTenantID(tenantID); err != nil {
		return nil, err
	}
	return s.repo.ListTokenEvents(ctx, tenantID, limit)
}

func (s *ExecutionService) RecentEventsBefore(ctx context.Context, tenantID string, before time.Time, limit int) ([]domain.TokenEvent, error) {
	if err := validateTenantID(tenantID); err != nil {
		return nil, err
	}
	return s.repo.ListTokenEventsBefore(ctx, tenantID, before, limit)
}

func (s *ExecutionService) Recommendations(ctx context.Context, tenantID string) ([]domain.RoutingRecommendation, error) {
	if err := validateTenantID(tenantID); err != nil {
		return nil, err
	}
	// Fetch up to 1000 recent events to base recommendations on
	events, err := s.repo.ListTokenEvents(ctx, tenantID, 1000)
	if err != nil {
		return nil, err
	}
	
	// Import statement will be added automatically by goimports or we can add it later if needed.
	// Wait, I need to make sure "github.com/Hardonian/TokenGoblin/internal/moat" is imported in service.go
	return moat.RecommendRoutes(events), nil
}

func (s *ExecutionService) normalize(tenantID string, event domain.TokenEvent) (domain.TokenEvent, []domain.Issue, error) {
	if err := validateTenantID(tenantID); err != nil {
		return domain.TokenEvent{}, nil, err
	}
	if event.TenantID != "" && event.TenantID != tenantID {
		return domain.TokenEvent{}, nil, TenantMismatchError{TenantID: event.TenantID}
	}

	var issues []domain.Issue
	now := s.now()
	event.TenantID = tenantID
	event.Provider = strings.ToLower(strings.TrimSpace(event.Provider))
	event.ModelID = strings.TrimSpace(event.ModelID)
	event.WorkerID = strings.TrimSpace(event.WorkerID)
	event.WorkerName = strings.TrimSpace(event.WorkerName)
	event.TaskCategory = strings.TrimSpace(event.TaskCategory)
	event.TaskType = strings.TrimSpace(event.TaskType)

	if event.EventID == "" {
		event.EventID = "evt_" + randomHex(12)
	}
	if event.Timestamp.IsZero() {
		event.Timestamp = now
	}
	event.Timestamp = event.Timestamp.UTC()
	if event.CreatedAt.IsZero() {
		event.CreatedAt = now
	}
	event.CreatedAt = event.CreatedAt.UTC()
	if event.WorkerName == "" {
		event.WorkerName = event.WorkerID
	}
	if event.TaskCategory == "" {
		event.TaskCategory = event.TaskType
	}
	if event.TaskCategory == "" {
		event.TaskCategory = "uncategorized"
	}
	event.TaskType = event.TaskCategory
	if event.OutputStatus == "" {
		event.OutputStatus = domain.OutputSucceeded
	}
	if event.CostCurrency == "" {
		event.CostCurrency = "USD"
	}
	if event.InputTokens == 0 {
		event.InputTokens = event.PromptTokens
	}
	if event.OutputTokens == 0 {
		event.OutputTokens = event.CompletionTokens
	}
	event.TotalTokens = event.InputTokens + event.OutputTokens

	if event.CostEstimateUSD != nil {
		issues = append(issues, domain.Issue{
			Code:    "ignored_client_cost",
			Message: "Client-supplied cost_estimate_usd was ignored; internal pricing is authoritative.",
			Field:   "cost_estimate_usd",
		})
		event.CostEstimateUSD = nil
	}
	if event.ExternalEstimate != nil {
		event.ExternalEstimate.Currency = strings.ToUpper(strings.TrimSpace(event.ExternalEstimate.Currency))
		if event.ExternalEstimate.Currency == "" {
			event.ExternalEstimate.Currency = "USD"
		}
	}

	if validationIssues := validateEvent(event); len(validationIssues) > 0 {
		return domain.TokenEvent{}, issues, ValidationError{Issues: validationIssues}
	}

	return event, issues, nil
}

func validateEvent(event domain.TokenEvent) []domain.Issue {
	var issues []domain.Issue
	required := map[string]string{
		"event_id":  event.EventID,
		"worker_id": event.WorkerID,
		"provider":  event.Provider,
		"model_id":  event.ModelID,
	}
	for field, value := range required {
		if strings.TrimSpace(value) == "" {
			issues = append(issues, domain.Issue{Code: "required", Message: field + " is required.", Field: field})
		}
	}
	if event.InputTokens < 0 || event.OutputTokens < 0 || event.PromptTokens < 0 || event.CompletionTokens < 0 || event.CachedTokens < 0 {
		issues = append(issues, domain.Issue{Code: "invalid_tokens", Message: "Token counts must be non-negative.", Field: "tokens"})
	}
	if event.CachedTokens > event.InputTokens {
		issues = append(issues, domain.Issue{Code: "invalid_cached_tokens", Message: "cached_tokens cannot exceed input_tokens.", Field: "cached_tokens"})
	}
	if event.LatencyMs < 0 {
		issues = append(issues, domain.Issue{Code: "invalid_latency", Message: "latency_ms must be non-negative.", Field: "latency_ms"})
	}
	if event.ReviewScore != nil && (*event.ReviewScore < 0 || *event.ReviewScore > 100) {
		issues = append(issues, domain.Issue{Code: "invalid_review_score", Message: "review_score must be between 0 and 100.", Field: "review_score"})
	}
	if event.ExternalEstimate != nil && event.ExternalEstimate.CostUSD < 0 {
		issues = append(issues, domain.Issue{Code: "invalid_external_estimate", Message: "external_estimate.cost_usd must be non-negative.", Field: "external_estimate.cost_usd"})
	}
	if !validOutputStatus(event.OutputStatus) {
		issues = append(issues, domain.Issue{Code: "invalid_output_status", Message: "output_status is not supported.", Field: "output_status"})
	}
	return issues
}

func validOutputStatus(status domain.OutputStatus) bool {
	switch status {
	case domain.OutputAccepted, domain.OutputSucceeded, domain.OutputFailed, domain.OutputRejected, domain.OutputPending:
		return true
	default:
		return false
	}
}

func validateTenantID(tenantID string) error {
	if strings.TrimSpace(tenantID) == "" {
		return TenantMissingError{}
	}
	if !tenantIDPattern.MatchString(tenantID) {
		return ValidationError{Issues: []domain.Issue{{
			Code:    "tenant_invalid",
			Message: "Invalid tenant context.",
			Field:   "x-tenant-id",
		}}}
	}
	return nil
}

type ValidationError struct {
	Issues []domain.Issue
}

func (e ValidationError) Error() string {
	return "validation failed"
}

type TenantMissingError struct{}

func (e TenantMissingError) Error() string {
	return "missing tenant context"
}

type TenantMismatchError struct {
	TenantID string
}

func (e TenantMismatchError) Error() string {
	return "payload tenant does not match request tenant context"
}

func randomHex(bytes int) string {
	buffer := make([]byte, bytes)
	if _, err := rand.Read(buffer); err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(buffer)
}
