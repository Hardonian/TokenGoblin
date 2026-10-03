package intelligence

import (
	"testing"
	"time"

	"github.com/Hardonian/TokenGoblin/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEngine_DegradedPaths(t *testing.T) {
	engine := NewEngine()

	t.Run("Empty Events Slice Does Not Panic", func(t *testing.T) {
		fingerprints := engine.BuildFingerprints("tenant-1", nil)
		assert.Empty(t, fingerprints)

		graveyard := engine.FindGraveyardPrompts(nil)
		assert.Empty(t, graveyard)

		duplicates := engine.FindDuplicates(nil)
		assert.Empty(t, duplicates)

		zombies := engine.DetectZombieAgents(nil)
		assert.Empty(t, zombies)

		leaks := engine.DetectCostLeaks(nil)
		assert.Empty(t, leaks)

		heatmap := engine.BuildHallucinationHeatmap(nil)
		assert.Empty(t, heatmap)

		report := engine.GenerateWasteReport("tenant-1", nil)
		assert.Equal(t, "tenant-1", report.TenantID)
		assert.Equal(t, 0.0, report.TotalWasteUSD)
		assert.Empty(t, report.WastefulPrompts)
		assert.Empty(t, report.DuplicatePrompts)
		assert.Empty(t, report.ZombieAgents)
		assert.Empty(t, report.CostLeaks)
	})

	t.Run("Events With Missing Or Nil Fields Handled Gracefully", func(t *testing.T) {
		now := time.Now().UTC()
		degradedEvents := []domain.TokenEvent{
			{
				EventID:         "evt_degraded_1",
				TenantID:        "tenant-1",
				Timestamp:       now,
				CostEstimateUSD: nil,
				PromptExcerpt:   "",
				ModelID:         "",
				WorkerID:        "",
			},
			{
				EventID:         "evt_degraded_2",
				TenantID:        "tenant-1",
				Timestamp:       now.Add(-time.Hour),
				CostEstimateUSD: nil,
				PromptExcerpt:   "   ", // whitespace only
				ModelID:         "unknown",
				WorkerID:        "worker-x",
			},
		}

		fingerprints := engine.BuildFingerprints("tenant-1", degradedEvents)
		assert.Empty(t, fingerprints)

		zombies := engine.DetectZombieAgents(degradedEvents)
		assert.Empty(t, zombies)

		leaks := engine.DetectCostLeaks(degradedEvents)
		assert.Empty(t, leaks)

		heatmap := engine.BuildHallucinationHeatmap(degradedEvents)
		assert.Empty(t, heatmap)

		report := engine.GenerateWasteReport("tenant-1", degradedEvents)
		require.NotNil(t, report)
		assert.Equal(t, 0.0, report.TotalWasteUSD)
	})
}
