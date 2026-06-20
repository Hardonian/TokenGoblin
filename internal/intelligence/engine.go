package intelligence

import (
	"crypto/sha256"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/Hardonian/TokenGoblin/internal/domain"
)

// Engine implements Layer 4 intelligence analysis: waste detection,
// prompt fingerprinting, zombie agent detection, duplication finding,
// cost leak detection, and hallucination heatmapping.
type Engine struct{}

// NewEngine creates a new intelligence engine instance.
func NewEngine() *Engine {
	return &Engine{}
}

// ═══════════════════════════════════════════════════════════
// Prompt Fingerprinting
// ═══════════════════════════════════════════════════════════

// HashPrompt produces a stable SHA-256 hash of a normalized prompt.
func HashPrompt(prompt string) string {
	normalized := strings.TrimSpace(strings.ToLower(prompt))
	h := sha256.Sum256([]byte(normalized))
	return fmt.Sprintf("%x", h)
}

type promptAccumulator struct {
	hash          string
	firstSeen     time.Time
	lastSeen      time.Time
	count         int
	totalCost     float64
	totalOutput   int
	acceptedCount int
	workerSet     map[string]struct{}
	taskCategory  string
}

func mapAccumulatorToFingerprint(tenantID string, acc *promptAccumulator) domain.PromptFingerprint {
	acceptanceRate := 0.0
	if acc.count > 0 {
		acceptanceRate = float64(acc.acceptedCount) / float64(acc.count)
	}
	avgCost := 0.0
	if acc.count > 0 {
		avgCost = acc.totalCost / float64(acc.count)
	}
	avgOutput := 0
	if acc.count > 0 {
		avgOutput = acc.totalOutput / acc.count
	}

	// Waste score: higher = more wasteful
	// (1 - acceptance_rate) × avg_cost × occurrence_count
	wasteScore := (1.0 - acceptanceRate) * avgCost * float64(acc.count)
	isWasteful := wasteScore > 1.0 && acceptanceRate < 0.2 && acc.count >= 3

	wasteReason := ""
	if isWasteful {
		if acceptanceRate == 0 {
			wasteReason = "zero_acceptance"
		} else if acceptanceRate < 0.1 {
			wasteReason = "near_zero_acceptance"
		} else {
			wasteReason = "low_acceptance_high_cost"
		}
	}

	return domain.PromptFingerprint{
		FingerprintID:         fmt.Sprintf("fp_%s", acc.hash[:16]),
		TenantID:              tenantID,
		PromptHash:            acc.hash,
		FirstSeenAt:           acc.firstSeen,
		LastSeenAt:            acc.lastSeen,
		OccurrenceCount:       acc.count,
		AvgCostUSD:            avgCost,
		AvgOutputTokens:       avgOutput,
		AvgAcceptanceRate:     acceptanceRate,
		CanonicalTaskCategory: acc.taskCategory,
		IsWasteful:            isWasteful,
		WasteReason:           wasteReason,
		WasteScore:            wasteScore,
		TotalCostUSD:          acc.totalCost,
		UniqueWorkers:         len(acc.workerSet),
	}
}

// BuildFingerprints aggregates events into prompt fingerprints.
func (e *Engine) BuildFingerprints(tenantID string, events []domain.TokenEvent) []domain.PromptFingerprint {
	buckets := make(map[string]*promptAccumulator)

	for i := range events {
		ev := &events[i]
		if ev.PromptExcerpt == "" {
			continue
		}
		hash := HashPrompt(ev.PromptExcerpt)
		acc, ok := buckets[hash]
		if !ok {
			acc = &promptAccumulator{
				hash:      hash,
				firstSeen: ev.Timestamp,
				lastSeen:  ev.Timestamp,
				workerSet: make(map[string]struct{}),
			}
			buckets[hash] = acc
		}

		acc.count++
		if ev.Timestamp.Before(acc.firstSeen) {
			acc.firstSeen = ev.Timestamp
		}
		if ev.Timestamp.After(acc.lastSeen) {
			acc.lastSeen = ev.Timestamp
		}
		if ev.CostEstimateUSD != nil {
			acc.totalCost += *ev.CostEstimateUSD
		}
		acc.totalOutput += ev.OutputTokens
		if ev.OutputStatus == domain.OutputAccepted || ev.OutputStatus == domain.OutputSucceeded {
			acc.acceptedCount++
		}
		acc.workerSet[ev.WorkerID] = struct{}{}
		if ev.TaskCategory != "" {
			acc.taskCategory = ev.TaskCategory
		}
	}

	fingerprints := make([]domain.PromptFingerprint, 0, len(buckets))
	for _, acc := range buckets {
		fingerprints = append(fingerprints, mapAccumulatorToFingerprint(tenantID, acc))
	}

	// Sort by waste score descending
	sort.Slice(fingerprints, func(i, j int) bool {
		return fingerprints[i].WasteScore > fingerprints[j].WasteScore
	})

	return fingerprints
}

// ═══════════════════════════════════════════════════════════
// Prompt Graveyard
// ═══════════════════════════════════════════════════════════

// FindGraveyardPrompts returns prompts with high cost and zero accepted outputs.
func (e *Engine) FindGraveyardPrompts(fingerprints []domain.PromptFingerprint) []domain.PromptFingerprint {
	var graveyard []domain.PromptFingerprint
	for _, fp := range fingerprints {
		if fp.OccurrenceCount >= 3 && fp.AvgAcceptanceRate == 0 && fp.TotalCostUSD > 1.0 {
			graveyard = append(graveyard, fp)
		}
	}
	return graveyard
}

// ═══════════════════════════════════════════════════════════
// Duplicate Detection
// ═══════════════════════════════════════════════════════════

// FindDuplicates identifies prompts used identically across multiple workers.
func (e *Engine) FindDuplicates(fingerprints []domain.PromptFingerprint) []domain.DuplicateCluster {
	var clusters []domain.DuplicateCluster
	for _, fp := range fingerprints {
		if fp.UniqueWorkers >= 2 && fp.OccurrenceCount >= 3 {
			// Redundant cost = total cost minus what one caller would have paid
			oneCopy := fp.AvgCostUSD
			redundantCost := fp.TotalCostUSD - oneCopy
			if redundantCost < 0 {
				redundantCost = 0
			}

			clusters = append(clusters, domain.DuplicateCluster{
				PromptHash:       fp.PromptHash,
				OccurrenceCount:  fp.OccurrenceCount,
				UniqueWorkers:    fp.UniqueWorkers,
				TotalCostUSD:     fp.TotalCostUSD,
				RedundantCostUSD: redundantCost,
				TaskCategory:     fp.CanonicalTaskCategory,
			})
		}
	}

	sort.Slice(clusters, func(i, j int) bool {
		return clusters[i].RedundantCostUSD > clusters[j].RedundantCostUSD
	})

	return clusters
}

// ═══════════════════════════════════════════════════════════
// Zombie Agent Detection
// ═══════════════════════════════════════════════════════════

// DetectZombieAgents identifies agents running with low business value.
// An agent is a "zombie" if it has high activity but low acceptance rate.
func (e *Engine) DetectZombieAgents(events []domain.TokenEvent) []domain.ZombieAgent {
	type workerStats struct {
		workerName    string
		eventCount    int
		totalCost     float64
		acceptedCount int
		outcomeCount  int
	}

	stats := make(map[string]*workerStats)

	now := time.Now().UTC()
	sevenDaysAgo := now.AddDate(0, 0, -7)

	for i := range events {
		ev := &events[i]
		if ev.Timestamp.Before(sevenDaysAgo) {
			continue
		}
		ws, ok := stats[ev.WorkerID]
		if !ok {
			ws = &workerStats{workerName: ev.WorkerName}
			stats[ev.WorkerID] = ws
		}
		ws.eventCount++
		if ev.CostEstimateUSD != nil {
			ws.totalCost += *ev.CostEstimateUSD
		}
		if ev.OutputStatus == domain.OutputAccepted || ev.OutputStatus == domain.OutputSucceeded {
			ws.acceptedCount++
			ws.outcomeCount++
		}
	}

	var zombies []domain.ZombieAgent
	for workerID, ws := range stats {
		if ws.eventCount < 50 || ws.totalCost < 10.0 {
			continue
		}

		acceptanceRate := 0.0
		if ws.eventCount > 0 {
			acceptanceRate = float64(ws.acceptedCount) / float64(ws.eventCount)
		}

		// Zombie score: high cost × low acceptance
		zombieScore := ws.totalCost * (1.0 - acceptanceRate)

		if acceptanceRate < 0.1 || ws.outcomeCount == 0 {
			reason := "low_acceptance_rate"
			if ws.outcomeCount == 0 {
				reason = "no_business_outcomes"
			}
			zombies = append(zombies, domain.ZombieAgent{
				WorkerID:       workerID,
				WorkerName:     ws.workerName,
				EventCount7d:   ws.eventCount,
				TotalCost7dUSD: ws.totalCost,
				AcceptanceRate: acceptanceRate,
				OutcomeCount:   ws.outcomeCount,
				ZombieScore:    zombieScore,
				Reason:         reason,
			})
		}
	}

	sort.Slice(zombies, func(i, j int) bool {
		return zombies[i].ZombieScore > zombies[j].ZombieScore
	})

	return zombies
}

// ═══════════════════════════════════════════════════════════
// Cost Leak Detection
// ═══════════════════════════════════════════════════════════

// DetectCostLeaks finds patterns of silent, invisible spending.
func (e *Engine) DetectCostLeaks(events []domain.TokenEvent) []domain.CostLeak {
	var leaks []domain.CostLeak

	leaks = append(leaks, e.detectRetryStorms(events)...)
	leaks = append(leaks, e.detectContextPadding(events)...)
	leaks = append(leaks, e.detectCacheMisses(events)...)
	leaks = append(leaks, e.detectPromptSlop(events)...)

	sort.Slice(leaks, func(i, j int) bool {
		return leaks[i].CostUSD > leaks[j].CostUSD
	})

	return leaks
}

func (e *Engine) detectRetryStorms(events []domain.TokenEvent) []domain.CostLeak {
	var leaks []domain.CostLeak
	idempotencyBuckets := make(map[string]int)
	idempotencyCost := make(map[string]float64)
	for i := range events {
		ev := &events[i]
		if ev.IdempotencyKey != "" {
			idempotencyBuckets[ev.IdempotencyKey]++
			if ev.CostEstimateUSD != nil {
				idempotencyCost[ev.IdempotencyKey] += *ev.CostEstimateUSD
			}
		}
	}
	for key, count := range idempotencyBuckets {
		if count > 3 {
			leaks = append(leaks, domain.CostLeak{
				Type:        domain.CostLeakRetryStorm,
				Description: fmt.Sprintf("Retry storm detected: %d events with idempotency key %s", count, key[:min(16, len(key))]),
				CostUSD:     idempotencyCost[key],
				EventCount:  count,
				Severity:    "high",
			})
		}
	}
	return leaks
}

func (e *Engine) detectContextPadding(events []domain.TokenEvent) []domain.CostLeak {
	var leaks []domain.CostLeak
	// Model context windows (approximate)
	contextWindows := map[string]int{
		"gpt-4o":            128000,
		"gpt-4o-mini":       128000,
		"claude-3-5-sonnet": 200000,
		"claude-3-5-haiku":  200000,
		"gemini-2.5-pro":    1000000,
		"gemini-2.5-flash":  1000000,
	}

	type modelUsage struct {
		highUsageCount int
		totalCount     int
		totalCost      float64
		workerID       string
	}
	modelPadding := make(map[string]*modelUsage)
	for i := range events {
		ev := &events[i]
		window, hasWindow := contextWindows[ev.ModelID]
		if !hasWindow {
			continue
		}
		mu, ok := modelPadding[ev.ModelID]
		if !ok {
			mu = &modelUsage{}
			modelPadding[ev.ModelID] = mu
		}
		mu.totalCount++
		if ev.CostEstimateUSD != nil {
			mu.totalCost += *ev.CostEstimateUSD
		}
		mu.workerID = ev.WorkerID
		if ev.InputTokens > int(float64(window)*0.8) {
			mu.highUsageCount++
		}
	}
	for modelID, mu := range modelPadding {
		if mu.totalCount >= 5 && float64(mu.highUsageCount)/float64(mu.totalCount) > 0.5 {
			leaks = append(leaks, domain.CostLeak{
				Type:        domain.CostLeakContextPadding,
				Description: fmt.Sprintf("Context window padding: %d/%d events use >80%% of %s context window", mu.highUsageCount, mu.totalCount, modelID),
				CostUSD:     mu.totalCost * 0.3, // Estimate 30% is padding waste
				EventCount:  mu.highUsageCount,
				Severity:    "medium",
				ModelID:     modelID,
			})
		}
	}
	return leaks
}

func (e *Engine) detectCacheMisses(events []domain.TokenEvent) []domain.CostLeak {
	var leaks []domain.CostLeak
	type cacheStats struct {
		totalEvents      int
		uncachedEvents   int
		totalCost        float64
		potentialSavings float64
	}
	cacheBuckets := make(map[string]*cacheStats)
	for i := range events {
		ev := &events[i]
		if ev.PromptExcerpt == "" {
			continue
		}
		hash := HashPrompt(ev.PromptExcerpt)
		cs, ok := cacheBuckets[hash]
		if !ok {
			cs = &cacheStats{}
			cacheBuckets[hash] = cs
		}
		cs.totalEvents++
		if ev.CachedTokens == 0 {
			cs.uncachedEvents++
		}
		if ev.CostEstimateUSD != nil {
			cs.totalCost += *ev.CostEstimateUSD
		}
	}
	totalCacheMissCost := 0.0
	totalCacheMissEvents := 0
	for _, cs := range cacheBuckets {
		if cs.totalEvents >= 3 && float64(cs.uncachedEvents)/float64(cs.totalEvents) > 0.8 {
			// Could save ~50% of input cost with caching
			totalCacheMissCost += cs.totalCost * 0.25
			totalCacheMissEvents += cs.uncachedEvents
		}
	}
	if totalCacheMissEvents > 0 {
		leaks = append(leaks, domain.CostLeak{
			Type:        domain.CostLeakCacheMiss,
			Description: fmt.Sprintf("Cache miss opportunities: %d repeated prompts could benefit from prompt caching", totalCacheMissEvents),
			CostUSD:     totalCacheMissCost,
			EventCount:  totalCacheMissEvents,
			Severity:    "medium",
		})
	}
	return leaks
}

func (e *Engine) detectPromptSlop(events []domain.TokenEvent) []domain.CostLeak {
	var leaks []domain.CostLeak
	slopCount := 0
	slopCost := 0.0

	slopPhrases := []string{
		"please ", "thank you", "as an ai", "i'm sorry", "could you kindly",
		"i would appreciate", "make sure to",
	}

	for i := range events {
		ev := &events[i]
		if ev.PromptExcerpt == "" {
			continue
		}

		lowerPrompt := strings.ToLower(ev.PromptExcerpt)
		isSlop := false
		for _, phrase := range slopPhrases {
			if strings.Contains(lowerPrompt, phrase) {
				isSlop = true
				break
			}
		}

		if isSlop {
			slopCount++
			if ev.CostEstimateUSD != nil {
				// Attribute 5% of the cost to unnecessary conversational slop
				slopCost += *ev.CostEstimateUSD * 0.05
			}
		}
	}

	if slopCount > 0 && slopCost > 0.1 {
		leaks = append(leaks, domain.CostLeak{
			Type:        "conversational_slop",
			Description: fmt.Sprintf("Goblin detected polite AI slop: %d prompts contained unnecessary conversational filler (e.g. 'please', 'thank you'). Goblins don't need manners, they need tokens!", slopCount),
			CostUSD:     slopCost,
			EventCount:  slopCount,
			Severity:    "low",
		})
	}

	return leaks
}

// ═══════════════════════════════════════════════════════════
// Hallucination Heatmap
// ═══════════════════════════════════════════════════════════

// BuildHallucinationHeatmap identifies where failures cluster.
func (e *Engine) BuildHallucinationHeatmap(events []domain.TokenEvent) []domain.HallucinationCell {
	type cellKey struct {
		modelID      string
		taskCategory string
	}

	type cellData struct {
		failureCount int
		totalCount   int
		totalCost    float64
	}

	cells := make(map[cellKey]*cellData)

	for i := range events {
		ev := &events[i]
		if ev.ModelID == "" {
			continue
		}
		key := cellKey{
			modelID:      ev.ModelID,
			taskCategory: ev.TaskCategory,
		}
		cd, ok := cells[key]
		if !ok {
			cd = &cellData{}
			cells[key] = cd
		}
		cd.totalCount++
		if ev.OutputStatus == domain.OutputFailed || ev.OutputStatus == domain.OutputRejected {
			cd.failureCount++
		}
		if ev.CostEstimateUSD != nil {
			cd.totalCost += *ev.CostEstimateUSD
		}
	}

	var heatmap []domain.HallucinationCell
	for key, cd := range cells {
		if cd.totalCount < 5 {
			continue
		}
		failureRate := float64(cd.failureCount) / float64(cd.totalCount)
		if failureRate > 0.05 { // Only surface cells with >5% failure rate
			heatmap = append(heatmap, domain.HallucinationCell{
				ModelID:      key.modelID,
				TaskCategory: key.taskCategory,
				FailureCount: cd.failureCount,
				TotalCount:   cd.totalCount,
				FailureRate:  failureRate,
				TotalCostUSD: cd.totalCost,
			})
		}
	}

	sort.Slice(heatmap, func(i, j int) bool {
		return heatmap[i].FailureRate > heatmap[j].FailureRate
	})

	return heatmap
}

// ═══════════════════════════════════════════════════════════
// Full Waste Report
// ═══════════════════════════════════════════════════════════

// GenerateWasteReport produces a comprehensive waste analysis.
func (e *Engine) GenerateWasteReport(tenantID string, events []domain.TokenEvent) domain.WasteReport {
	fingerprints := e.BuildFingerprints(tenantID, events)
	graveyard := e.FindGraveyardPrompts(fingerprints)
	duplicates := e.FindDuplicates(fingerprints)
	zombies := e.DetectZombieAgents(events)
	costLeaks := e.DetectCostLeaks(events)

	totalWaste := 0.0
	for _, fp := range graveyard {
		totalWaste += fp.TotalCostUSD
	}
	for _, dup := range duplicates {
		totalWaste += dup.RedundantCostUSD
	}
	for _, z := range zombies {
		totalWaste += z.TotalCost7dUSD * (1 - z.AcceptanceRate)
	}
	for _, leak := range costLeaks {
		totalWaste += leak.CostUSD
	}

	return domain.WasteReport{
		TenantID:         tenantID,
		GeneratedAt:      time.Now().UTC(),
		TotalWasteUSD:    totalWaste,
		WastefulPrompts:  graveyard,
		DuplicatePrompts: duplicates,
		ZombieAgents:     zombies,
		CostLeaks:        costLeaks,
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
