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

	"github.com/Hardonian/TokenGoblin/internal/analysis"
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
	OutputAnalyses(ctx context.Context, tenantID string, limit int) ([]domain.OutputAnalysis, error)
	WorkerReview(ctx context.Context, tenantID, workerID string) (domain.WorkerReview, error)
	Recommendations(ctx context.Context, tenantID string) ([]domain.RoutingRecommendation, error)
	SetRecommendationState(ctx context.Context, tenantID, recommendationID, actor string, update domain.RecommendationStateUpdate) (domain.RecommendationState, error)
	AuditEvents(ctx context.Context, tenantID string, limit int) ([]domain.AuditEvent, error)
	TenantMembers(ctx context.Context, tenantID string) ([]domain.TenantMember, error)
	UpsertTenantMember(ctx context.Context, tenantID string, member domain.TenantMember) (domain.TenantMember, error)
	SetPricingOverride(ctx context.Context, tenantID string, point domain.PricePoint) error
	GetActivePricing(ctx context.Context, tenantID string) ([]domain.PricePoint, error)
	DeleteTenantData(ctx context.Context, tenantID string) error
}

type ExecutionService struct {
	repo       storage.Repository
	pricing    cost.Registry
	eventQueue chan domain.TokenEvent
	now        func() time.Time
	thresholds anomaly.Thresholds
	alerter    moat.Alerter
}

func NewService(repo storage.Repository, registry cost.Registry) *ExecutionService {
	return &ExecutionService{
		repo:       repo,
		pricing:    registry,
		eventQueue: make(chan domain.TokenEvent, 1000),
		now:        time.Now,
		thresholds: anomaly.DefaultThresholds(),
		alerter:    moat.NewWebhookAlerter(),
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
	if normalized.CostEstimateUSD == nil && !normalized.CostIsDegraded && normalized.CostDegradedCode == "" {
		normalized = applyCostResult(normalized, s.pricing.Calculate(ctx, normalized, s.repo))
	}

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

	now := s.now()
	signals := anomaly.Detect(normalized, prior, now, s.thresholds)

	if err := s.repo.SaveTokenEvent(ctx, normalized); err != nil {
		return err
	}
	outputAnalysis := analysis.Analyze(normalized, now)
	if err := s.repo.SaveOutputAnalysis(ctx, outputAnalysis); err != nil {
		return err
	}
	if err := s.repo.SaveCostSnapshot(ctx, snapshot); err != nil {
		return err
	}
	if len(signals) > 0 {
		for _, signal := range signals {
			if err := s.repo.SaveAnomalySignal(ctx, signal); err != nil {
				return err
			}
		}

		// Run alerter asynchronously
		go func() {
			if err := s.alerter.Alert(context.Background(), normalized.TenantID, signals); err != nil {
				// log error in a real app
			}
		}()
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
	tenant, err := s.repo.GetTenant(ctx, tenantID)
	if err != nil {
		return domain.IngestionResult{}, err
	}
	if tenant != nil {
		usage, err := s.repo.GetTenantCurrentMonthCost(ctx, tenantID)
		if err == nil && usage >= tenant.UsageLimitUSD {
			return domain.IngestionResult{}, QuotaExceededError{Limit: tenant.UsageLimitUSD, Usage: usage}
		}
	}

	normalized, warnings, err := s.normalize(tenantID, event)
	if err != nil {
		return domain.IngestionResult{}, err
	}

	costResult := s.pricing.Calculate(ctx, normalized, s.repo)
	normalized = applyCostResult(normalized, costResult)

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
		normalized, warnings, err := s.normalize(tenantID, event)
		if err != nil {
			var validationErr ValidationError
			if errors.As(err, &validationErr) {
				// We can return a result with error status instead of failing the whole batch
				res := domain.IngestionResult{
					Event:    event,
					Warnings: append(warnings, domain.Issue{Code: "validation_failed", Message: "Event failed validation."}),
					Degraded: validationErr.Issues,
				}
				results = append(results, res)
				continue
			} else {
				// For structural errors (queue full), we might fail the whole batch
				return nil, err
			}
		}

		costResult := s.pricing.Calculate(ctx, normalized, s.repo)
		normalized.CostEstimateUSD = costResult.CostEstimateUSD
		normalized.CostCurrency = costResult.Currency
		normalized.CostIsDegraded = costResult.Status == cost.StatusDegraded
		normalized.CostDegradedCode = costResult.DegradedCode

		select {
		case s.eventQueue <- normalized:
			// Buffered successfully
		default:
			// Queue is full, return error
			return nil, errors.New("ingestion buffer full")
		}

		res := domain.IngestionResult{
			Event:    normalized,
			Warnings: append(warnings, s.pricing.Diagnostics()...),
		}
		if normalized.CostIsDegraded {
			res.Degraded = append(res.Degraded, domain.Issue{
				Code:    normalized.CostDegradedCode,
				Message: "Internal cost estimate is unavailable for this provider/model.",
			})
		}
		res.Degraded = append(res.Degraded, domain.Issue{
			Code:    "buffered",
			Message: "Event buffered for asynchronous processing.",
		})

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

func (s *ExecutionService) OutputAnalyses(ctx context.Context, tenantID string, limit int) ([]domain.OutputAnalysis, error) {
	if err := validateTenantID(tenantID); err != nil {
		return nil, err
	}
	return s.repo.ListOutputAnalyses(ctx, tenantID, limit)
}

func (s *ExecutionService) WorkerReview(ctx context.Context, tenantID, workerID string) (domain.WorkerReview, error) {
	if err := validateTenantID(tenantID); err != nil {
		return domain.WorkerReview{}, err
	}
	workerID = strings.TrimSpace(workerID)
	if workerID == "" {
		return domain.WorkerReview{}, ValidationError{Issues: []domain.Issue{{
			Code:    "required",
			Message: "worker_id is required.",
			Field:   "worker_id",
		}}}
	}
	workers, err := s.Workers(ctx, tenantID)
	if err != nil {
		return domain.WorkerReview{}, err
	}
	review := domain.WorkerReview{
		WasteSignals:           []domain.AnalysisIssue{},
		RecommendedConstraints: []string{},
	}
	found := false
	for _, worker := range workers {
		if worker.WorkerID == workerID {
			review.Worker = worker
			found = true
			break
		}
	}
	if !found {
		review.Worker = domain.WorkerBreakdown{WorkerID: workerID, WorkerName: workerID}
		review.Degraded = append(review.Degraded, domain.Issue{Code: "no_data", Message: "No usage exists for this worker."})
		return review, nil
	}

	events, err := s.repo.ListTokenEvents(ctx, tenantID, 500)
	if err != nil {
		return domain.WorkerReview{}, err
	}
	for i := range events {
		if events[i].WorkerID == workerID {
			review.LatestOutput = &events[i]
			break
		}
	}
	analyses, err := s.repo.ListOutputAnalysesByWorker(ctx, tenantID, workerID, 25)
	if err != nil {
		return domain.WorkerReview{}, err
	}
	if len(analyses) > 0 {
		review.LatestAnalysis = &analyses[0]
		for _, item := range analyses {
			review.WasteSignals = append(review.WasteSignals, item.Issues...)
			review.RecommendedConstraints = append(review.RecommendedConstraints, item.Recommendations...)
		}
		review.RecommendedConstraints = dedupeStrings(review.RecommendedConstraints)
		if len(review.WasteSignals) > 8 {
			review.WasteSignals = review.WasteSignals[:8]
		}
	} else {
		review.Degraded = append(review.Degraded, domain.Issue{Code: "analysis_pending", Message: "No persisted output analysis exists for this worker yet."})
	}
	return review, nil
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

func (s *ExecutionService) SetPricingOverride(ctx context.Context, tenantID string, point domain.PricePoint) error {
	if err := validateTenantID(tenantID); err != nil {
		return err
	}
	if err := s.ensureTenant(ctx, tenantID); err != nil {
		return err
	}
	return s.repo.SetPricingOverride(ctx, tenantID, point)
}

func (s *ExecutionService) GetActivePricing(ctx context.Context, tenantID string) ([]domain.PricePoint, error) {
	if err := validateTenantID(tenantID); err != nil {
		return nil, err
	}

	overrides, err := s.repo.ListPricingOverrides(ctx, tenantID)
	if err != nil && !errors.Is(err, storage.ErrUnavailable) {
		return nil, err
	}

	overrideMap := make(map[string]domain.PricePoint)
	for _, o := range overrides {
		key := strings.ToLower(o.Provider) + ":" + strings.ToLower(o.ModelID)
		overrideMap[key] = o
	}

	var active []domain.PricePoint
	for _, dp := range cost.DefaultPrices() {
		key := strings.ToLower(dp.Provider) + ":" + strings.ToLower(dp.ModelID)
		if over, ok := overrideMap[key]; ok {
			active = append(active, over)
			delete(overrideMap, key)
		} else {
			active = append(active, dp)
		}
	}

	for _, over := range overrideMap {
		active = append(active, over)
	}

	return active, nil
}

func (s *ExecutionService) DeleteTenantData(ctx context.Context, tenantID string) error {
	if err := validateTenantID(tenantID); err != nil {
		return err
	}
	return s.repo.DeleteTenantData(ctx, tenantID)
}

func (s *ExecutionService) SetRecommendationState(ctx context.Context, tenantID, recommendationID, actor string, update domain.RecommendationStateUpdate) (domain.RecommendationState, error) {
	if err := validateTenantID(tenantID); err != nil {
		return domain.RecommendationState{}, err
	}
	recommendationID = strings.TrimSpace(recommendationID)
	if recommendationID == "" {
		return domain.RecommendationState{}, ValidationError{Issues: []domain.Issue{{Code: "required", Message: "recommendation_id is required.", Field: "recommendation_id"}}}
	}
	status := strings.ToLower(strings.TrimSpace(update.Status))
	if !validRecommendationStatus(status) {
		return domain.RecommendationState{}, ValidationError{Issues: []domain.Issue{{Code: "invalid_status", Message: "recommendation status is not supported.", Field: "status"}}}
	}
	states, err := s.repo.ListRecommendationStates(ctx, tenantID)
	if err != nil {
		return domain.RecommendationState{}, err
	}
	now := s.now().UTC()
	state := domain.RecommendationState{
		TenantID:         tenantID,
		RecommendationID: recommendationID,
		Status:           status,
		Actor:            strings.TrimSpace(actor),
		Note:             strings.TrimSpace(update.Note),
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	for _, existing := range states {
		if existing.RecommendationID == recommendationID {
			if !existing.CreatedAt.IsZero() {
				state.CreatedAt = existing.CreatedAt
			}
			break
		}
	}
	if err := s.repo.SetRecommendationState(ctx, state); err != nil {
		return domain.RecommendationState{}, err
	}
	return state, nil
}

func (s *ExecutionService) AuditEvents(ctx context.Context, tenantID string, limit int) ([]domain.AuditEvent, error) {
	if err := validateTenantID(tenantID); err != nil {
		return nil, err
	}
	return s.repo.ListAuditEvents(ctx, tenantID, limit)
}

func (s *ExecutionService) TenantMembers(ctx context.Context, tenantID string) ([]domain.TenantMember, error) {
	if err := validateTenantID(tenantID); err != nil {
		return nil, err
	}
	return s.repo.ListTenantMembers(ctx, tenantID)
}

func (s *ExecutionService) UpsertTenantMember(ctx context.Context, tenantID string, member domain.TenantMember) (domain.TenantMember, error) {
	if err := validateTenantID(tenantID); err != nil {
		return domain.TenantMember{}, err
	}
	member.SubjectID = strings.TrimSpace(member.SubjectID)
	if member.SubjectID == "" {
		return domain.TenantMember{}, ValidationError{Issues: []domain.Issue{{Code: "required", Message: "subject_id is required.", Field: "subject_id"}}}
	}
	if err := s.ensureTenant(ctx, tenantID); err != nil {
		return domain.TenantMember{}, err
	}
	now := s.now().UTC()
	member.TenantID = tenantID
	member.Email = strings.TrimSpace(member.Email)
	member.Role = normalizeRole(member.Role)
	if member.CreatedAt.IsZero() {
		member.CreatedAt = now
	}
	member.UpdatedAt = now
	if err := s.repo.UpsertTenantMember(ctx, member); err != nil {
		return domain.TenantMember{}, err
	}
	return member, nil
}

func (s *ExecutionService) ensureTenant(ctx context.Context, tenantID string) error {
	existing, err := s.repo.GetTenant(ctx, tenantID)
	if err != nil {
		return err
	}
	if existing != nil {
		return nil
	}

	now := s.now().UTC()
	return s.repo.UpsertTenant(ctx, domain.Tenant{
		TenantID:      tenantID,
		Name:          tenantID,
		Tier:          "free",
		UsageLimitUSD: 10,
		CreatedAt:     now,
		UpdatedAt:     now,
	})
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
	event.PromptExcerpt = truncate(strings.TrimSpace(event.PromptExcerpt), 4000)
	event.OutputExcerpt = truncate(strings.TrimSpace(event.OutputExcerpt), 4000)
	event.PromptReference = truncate(strings.TrimSpace(event.PromptReference), 512)
	event.OutputReference = truncate(strings.TrimSpace(event.OutputReference), 512)

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
	if strings.TrimSpace(event.EventID) == "" {
		issues = append(issues, requiredIssue("event_id"))
	}
	if strings.TrimSpace(event.WorkerID) == "" {
		issues = append(issues, requiredIssue("worker_id"))
	}
	if strings.TrimSpace(event.Provider) == "" {
		issues = append(issues, requiredIssue("provider"))
	}
	if strings.TrimSpace(event.ModelID) == "" {
		issues = append(issues, requiredIssue("model_id"))
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

func applyCostResult(event domain.TokenEvent, result domain.CostResult) domain.TokenEvent {
	event.CostEstimateUSD = result.CostEstimateUSD
	event.CostCurrency = result.Currency
	event.CostIsDegraded = result.Status == cost.StatusDegraded
	event.CostDegradedCode = result.DegradedCode
	return event
}

func requiredIssue(field string) domain.Issue {
	return domain.Issue{Code: "required", Message: field + " is required.", Field: field}
}

func validOutputStatus(status domain.OutputStatus) bool {
	switch status {
	case domain.OutputAccepted, domain.OutputSucceeded, domain.OutputFailed, domain.OutputRejected, domain.OutputPending:
		return true
	default:
		return false
	}
}

func validRecommendationStatus(status string) bool {
	switch status {
	case "open", "accepted", "rejected", "implemented":
		return true
	default:
		return false
	}
}

func normalizeRole(role string) string {
	switch strings.ToLower(strings.TrimSpace(role)) {
	case domain.RoleOwner:
		return domain.RoleOwner
	case domain.RoleAdmin:
		return domain.RoleAdmin
	case domain.RoleAnalyst:
		return domain.RoleAnalyst
	case domain.RoleIngest:
		return domain.RoleIngest
	case domain.RoleViewer:
		return domain.RoleViewer
	default:
		return domain.RoleViewer
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

type QuotaExceededError struct {
	Limit float64
	Usage float64
}

func (e QuotaExceededError) Error() string {
	return fmt.Sprintf("tenant quota exceeded: current usage $%.2f is greater than or equal to limit $%.2f", e.Usage, e.Limit)
}

func randomHex(bytes int) string {
	buffer := make([]byte, bytes)
	if _, err := rand.Read(buffer); err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(buffer)
}

func truncate(value string, max int) string {
	if len(value) <= max {
		return value
	}
	return value[:max]
}

func dedupeStrings(values []string) []string {
	seen := map[string]bool{}
	var result []string
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		result = append(result, value)
	}
	return result
}
