package intelligence

import (
	"testing"
	"time"

	"github.com/Hardonian/TokenGoblin/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func ptr(f float64) *float64 { return &f }

func TestBuildFingerprints(t *testing.T) {
	eng := NewEngine()
	now := time.Now().UTC()

	events := []domain.TokenEvent{
		{EventID: "1", TenantID: "t1", WorkerID: "w1", PromptExcerpt: "Write a hello world program", OutputStatus: domain.OutputAccepted, CostEstimateUSD: ptr(0.05), OutputTokens: 100, Timestamp: now},
		{EventID: "2", TenantID: "t1", WorkerID: "w1", PromptExcerpt: "Write a hello world program", OutputStatus: domain.OutputAccepted, CostEstimateUSD: ptr(0.05), OutputTokens: 110, Timestamp: now},
		{EventID: "3", TenantID: "t1", WorkerID: "w2", PromptExcerpt: "Write a hello world program", OutputStatus: domain.OutputFailed, CostEstimateUSD: ptr(0.05), OutputTokens: 50, Timestamp: now},
		{EventID: "4", TenantID: "t1", WorkerID: "w1", PromptExcerpt: "Summarize this document", OutputStatus: domain.OutputRejected, CostEstimateUSD: ptr(0.10), OutputTokens: 200, Timestamp: now},
	}

	fps := eng.BuildFingerprints("t1", events)
	require.Len(t, fps, 2, "should have 2 unique prompt fingerprints")

	// Find the hello world fingerprint
	var helloFP *domain.PromptFingerprint
	for i := range fps {
		if fps[i].OccurrenceCount == 3 {
			helloFP = &fps[i]
			break
		}
	}
	require.NotNil(t, helloFP)
	assert.Equal(t, 3, helloFP.OccurrenceCount)
	assert.Equal(t, 2, helloFP.UniqueWorkers, "used by w1 and w2")
	assert.InDelta(t, 0.667, helloFP.AvgAcceptanceRate, 0.01)
	assert.InDelta(t, 0.15, helloFP.TotalCostUSD, 0.001)
}

func TestFindGraveyardPrompts(t *testing.T) {
	eng := NewEngine()

	fingerprints := []domain.PromptFingerprint{
		{FingerprintID: "fp1", OccurrenceCount: 5, AvgAcceptanceRate: 0, TotalCostUSD: 5.00},
		{FingerprintID: "fp2", OccurrenceCount: 10, AvgAcceptanceRate: 0.8, TotalCostUSD: 3.00},
		{FingerprintID: "fp3", OccurrenceCount: 2, AvgAcceptanceRate: 0, TotalCostUSD: 10.00}, // Too few occurrences
		{FingerprintID: "fp4", OccurrenceCount: 3, AvgAcceptanceRate: 0, TotalCostUSD: 0.50},  // Too cheap
		{FingerprintID: "fp5", OccurrenceCount: 3, AvgAcceptanceRate: 0, TotalCostUSD: 2.00},
	}

	graveyard := eng.FindGraveyardPrompts(fingerprints)
	assert.Len(t, graveyard, 2, "should find 2 graveyard prompts (fp1 and fp5)")
}

func TestFindDuplicates(t *testing.T) {
	eng := NewEngine()

	fingerprints := []domain.PromptFingerprint{
		{PromptHash: "abc123", OccurrenceCount: 5, UniqueWorkers: 3, TotalCostUSD: 1.50, AvgCostUSD: 0.30},
		{PromptHash: "def456", OccurrenceCount: 10, UniqueWorkers: 1, TotalCostUSD: 5.00, AvgCostUSD: 0.50},
		{PromptHash: "ghi789", OccurrenceCount: 2, UniqueWorkers: 2, TotalCostUSD: 0.20, AvgCostUSD: 0.10}, // Too few
	}

	dupes := eng.FindDuplicates(fingerprints)
	assert.Len(t, dupes, 1, "should find 1 duplicate cluster")
	assert.Equal(t, "abc123", dupes[0].PromptHash)
	assert.InDelta(t, 1.20, dupes[0].RedundantCostUSD, 0.01)
}

func TestDetectZombieAgents(t *testing.T) {
	eng := NewEngine()
	now := time.Now().UTC()

	var events []domain.TokenEvent
	// Zombie: 60 events, 0 accepted
	for i := 0; i < 60; i++ {
		events = append(events, domain.TokenEvent{
			WorkerID:        "zombie-bot",
			WorkerName:      "Zombie Bot",
			CostEstimateUSD: ptr(0.50),
			OutputStatus:    domain.OutputFailed,
			Timestamp:       now.Add(-time.Duration(i) * time.Hour),
		})
	}
	// Healthy: 60 events, 50 accepted
	for i := 0; i < 60; i++ {
		status := domain.OutputAccepted
		if i < 10 {
			status = domain.OutputFailed
		}
		events = append(events, domain.TokenEvent{
			WorkerID:        "good-bot",
			WorkerName:      "Good Bot",
			CostEstimateUSD: ptr(0.50),
			OutputStatus:    status,
			Timestamp:       now.Add(-time.Duration(i) * time.Hour),
		})
	}

	zombies := eng.DetectZombieAgents(events)
	assert.Len(t, zombies, 1, "should detect 1 zombie")
	assert.Equal(t, "zombie-bot", zombies[0].WorkerID)
	assert.Equal(t, "no_business_outcomes", zombies[0].Reason)
}

func TestDetectCostLeaks_RetryStorm(t *testing.T) {
	eng := NewEngine()
	now := time.Now().UTC()

	var events []domain.TokenEvent
	// Retry storm: 5 events with same idempotency key
	for i := 0; i < 5; i++ {
		events = append(events, domain.TokenEvent{
			EventID:         "evt" + string(rune('a'+i)),
			IdempotencyKey:  "idem_12345",
			CostEstimateUSD: ptr(0.10),
			Timestamp:       now,
		})
	}

	leaks := eng.DetectCostLeaks(events)
	require.NotEmpty(t, leaks)

	found := false
	for _, l := range leaks {
		if l.Type == domain.CostLeakRetryStorm {
			found = true
			assert.Equal(t, 5, l.EventCount)
			assert.Equal(t, "high", l.Severity)
		}
	}
	assert.True(t, found, "should detect retry storm")
}

func TestBuildHallucinationHeatmap(t *testing.T) {
	eng := NewEngine()
	now := time.Now().UTC()

	var events []domain.TokenEvent
	// 10 events: 4 failed for code_gen on gpt-4o (40% failure rate)
	for i := 0; i < 10; i++ {
		status := domain.OutputAccepted
		if i < 4 {
			status = domain.OutputFailed
		}
		events = append(events, domain.TokenEvent{
			ModelID:         "gpt-4o",
			TaskCategory:    "code_gen",
			OutputStatus:    status,
			CostEstimateUSD: ptr(0.50),
			Timestamp:       now,
		})
	}
	// 10 events: 0 failed for summarization on gpt-4o (0% failure rate)
	for i := 0; i < 10; i++ {
		events = append(events, domain.TokenEvent{
			ModelID:         "gpt-4o",
			TaskCategory:    "summarization",
			OutputStatus:    domain.OutputAccepted,
			CostEstimateUSD: ptr(0.10),
			Timestamp:       now,
		})
	}

	heatmap := eng.BuildHallucinationHeatmap(events)
	assert.Len(t, heatmap, 1, "should find 1 failure hotspot")
	assert.Equal(t, "gpt-4o", heatmap[0].ModelID)
	assert.Equal(t, "code_gen", heatmap[0].TaskCategory)
	assert.InDelta(t, 0.40, heatmap[0].FailureRate, 0.01)
}

func TestGenerateWasteReport(t *testing.T) {
	eng := NewEngine()
	now := time.Now().UTC()

	var events []domain.TokenEvent
	// Some wasteful prompts
	for i := 0; i < 5; i++ {
		events = append(events, domain.TokenEvent{
			WorkerID:        "waste-bot",
			WorkerName:      "Waste Bot",
			PromptExcerpt:   "do something useless",
			CostEstimateUSD: ptr(1.00),
			OutputStatus:    domain.OutputRejected,
			OutputTokens:    500,
			Timestamp:       now,
		})
	}

	report := eng.GenerateWasteReport("t1", events)
	assert.Equal(t, "t1", report.TenantID)
	assert.True(t, report.TotalWasteUSD > 0, "should detect waste")
}
