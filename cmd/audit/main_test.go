package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAuditGenerator_CSV(t *testing.T) {
	tempDir := t.TempDir()

	csvContent := `model,prompt_tokens,completion_tokens,cached_tokens,cost,team,feature
gpt-4o,100000,20000,10000,0.45,data-platform,summarization
claude-3-5-sonnet,50000,10000,5000,0.30,product-eng,chat
o1,20000,5000,0,0.60,research,reasoning
gpt-4o-mini,200000,50000,20000,0.06,growth,classification
`
	inputPath := filepath.Join(tempDir, "sample_usage.csv")
	require.NoError(t, os.WriteFile(inputPath, []byte(csvContent), 0o600))

	outDir := filepath.Join(tempDir, "out")
	require.NoError(t, os.MkdirAll(outDir, 0o750))

	records, err := loadRecords(inputPath, "auto")
	require.NoError(t, err)
	require.Len(t, records, 4)

	report := analyzeRecords("Acme AI Corp", records)
	assert.Equal(t, "Acme AI Corp", report.TenantName)
	assert.Equal(t, 4, report.TotalEvents)
	assert.Equal(t, 455000, report.TotalTokens)
	assert.Equal(t, 370000, report.TotalPromptTokens)
	assert.Equal(t, 85000, report.TotalCompTokens)
	assert.InDelta(t, 1.41, report.TotalSpendUSD, 0.01)

	// Check savings opportunities
	require.Len(t, report.Opportunities, 4)
	assert.True(t, report.NetMonthlySavings > 0)
	assert.True(t, report.AnnualizedSavings > 0)
	assert.True(t, report.NetSavingsPct >= 30.0 && report.NetSavingsPct <= 50.0)

	// Generate reports
	htmlPath := filepath.Join(outDir, "audit_report.html")
	require.NoError(t, generateHTMLReport(report, htmlPath))
	htmlBytes, err := os.ReadFile(htmlPath)
	require.NoError(t, err)
	assert.Contains(t, string(htmlBytes), "Acme AI Corp")
	assert.Contains(t, string(htmlBytes), "gpt-4o")
	assert.Contains(t, string(htmlBytes), "Projected Net Monthly Savings")

	mdPath := filepath.Join(outDir, "audit_report.md")
	require.NoError(t, generateMarkdownReport(report, mdPath))
	mdBytes, err := os.ReadFile(mdPath)
	require.NoError(t, err)
	assert.Contains(t, string(mdBytes), "# LLM Spend Audit & Optimization Report")
	assert.Contains(t, string(mdBytes), "Acme AI Corp")

	csvPath := filepath.Join(outDir, "breakdown.csv")
	require.NoError(t, generateBreakdownCSV(report, csvPath))
	csvBytes, err := os.ReadFile(csvPath)
	require.NoError(t, err)
	assert.Contains(t, string(csvBytes), "dimension,item,provider")
	assert.Contains(t, string(csvBytes), "model,gpt-4o")
	assert.Contains(t, string(csvBytes), "team,data-platform")
}

func TestAuditGenerator_JSON(t *testing.T) {
	tempDir := t.TempDir()

	jsonContent := `[
  {
    "model": "gpt-4o",
    "prompt_tokens": 10000,
    "completion_tokens": 2000,
    "team": "ai-infra",
    "feature": "agent-orchestrator"
  },
  {
    "model": "claude-3-5-haiku",
    "prompt_tokens": 50000,
    "completion_tokens": 10000,
    "team": "customer-support",
    "feature": "auto-reply"
  }
]`
	inputPath := filepath.Join(tempDir, "sample_usage.json")
	require.NoError(t, os.WriteFile(inputPath, []byte(jsonContent), 0o600))

	records, err := loadRecords(inputPath, "auto")
	require.NoError(t, err)
	require.Len(t, records, 2)

	// In the JSON fixture, cost was not provided, so registry should have calculated it
	assert.True(t, records[0].CostUSD > 0, "gpt-4o cost should be calculated by registry")
	assert.True(t, records[1].CostUSD > 0, "claude-3-5-haiku cost should be calculated by registry")

	report := analyzeRecords("CyberCorp", records)
	assert.Equal(t, 2, report.TotalEvents)
	assert.True(t, report.TotalSpendUSD > 0)
	assert.Equal(t, 2, len(report.Models))
}

func TestInferProvider(t *testing.T) {
	assert.Equal(t, "openai", inferProvider("gpt-4o"))
	assert.Equal(t, "openai", inferProvider("o1-mini"))
	assert.Equal(t, "anthropic", inferProvider("claude-3-5-sonnet"))
	assert.Equal(t, "google", inferProvider("gemini-2.5-pro"))
	assert.Equal(t, "xai", inferProvider("grok-3"))
	assert.Equal(t, "deepseek", inferProvider("deepseek-chat"))
	assert.Equal(t, "openai", inferProvider("custom-model"))
}
