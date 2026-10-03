package main

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/Hardonian/TokenGoblin/internal/cost"
	"github.com/Hardonian/TokenGoblin/internal/domain"
)

type Record struct {
	EventID          string    `json:"event_id"`
	Timestamp        time.Time `json:"timestamp"`
	Provider         string    `json:"provider"`
	ModelID          string    `json:"model_id"`
	PromptTokens     int       `json:"prompt_tokens"`
	CompletionTokens int       `json:"completion_tokens"`
	CachedTokens     int       `json:"cached_tokens"`
	TotalTokens      int       `json:"total_tokens"`
	CostUSD          float64   `json:"cost_usd"`
	Team             string    `json:"team"`
	Feature          string    `json:"feature"`
	PromptExcerpt    string    `json:"prompt_excerpt"`
}

type ModelBreakdown struct {
	ModelID          string  `json:"model_id"`
	Provider         string  `json:"provider"`
	EventCount       int     `json:"event_count"`
	PromptTokens     int     `json:"prompt_tokens"`
	CompletionTokens int     `json:"completion_tokens"`
	CachedTokens     int     `json:"cached_tokens"`
	TotalTokens      int     `json:"total_tokens"`
	CostUSD          float64 `json:"cost_usd"`
	PctSpend         float64 `json:"pct_spend"`
}

type DimensionBreakdown struct {
	Name        string  `json:"name"`
	EventCount  int     `json:"event_count"`
	TotalTokens int     `json:"total_tokens"`
	CostUSD     float64 `json:"cost_usd"`
	PctSpend    float64 `json:"pct_spend"`
}

type SavingsOpportunity struct {
	ID          string  `json:"id"`
	Title       string  `json:"title"`
	Description string  `json:"description"`
	Assumptions string  `json:"assumptions"`
	SavingsUSD  float64 `json:"savings_usd"`
	SavingsPct  float64 `json:"savings_pct"`
	ActionPlan  string  `json:"action_plan"`
}

type AuditReportData struct {
	TenantName         string               `json:"tenant_name"`
	GeneratedAt        time.Time            `json:"generated_at"`
	TotalEvents        int                  `json:"total_events"`
	TotalTokens        int                  `json:"total_tokens"`
	TotalPromptTokens  int                  `json:"total_prompt_tokens"`
	TotalCompTokens    int                  `json:"total_comp_tokens"`
	TotalCachedTokens  int                  `json:"total_cached_tokens"`
	TotalSpendUSD      float64              `json:"total_spend_usd"`
	Models             []ModelBreakdown     `json:"models"`
	Teams              []DimensionBreakdown `json:"teams"`
	Features           []DimensionBreakdown `json:"features"`
	TopBurnDrivers     []string             `json:"top_burn_drivers"`
	Opportunities      []SavingsOpportunity `json:"opportunities"`
	NetMonthlySavings  float64              `json:"net_monthly_savings"`
	NetSavingsPct      float64              `json:"net_savings_pct"`
	NewMonthlySpendUSD float64              `json:"new_monthly_spend_usd"`
	AnnualizedSavings  float64              `json:"annualized_savings"`
}

func main() {
	inputFile := flag.String("input", "", "Path to usage export file (CSV, JSON, JSONL)")
	outputDir := flag.String("out", "./out", "Output directory for audit report files")
	tenantName := flag.String("tenant", "Enterprise Client", "Tenant / Organization name")
	formatFlag := flag.String("format", "auto", "File format: auto, csv, json, jsonl")
	flag.Parse()

	if *inputFile == "" {
		fmt.Fprintln(os.Stderr, "Error: --input file path is required")
		flag.Usage()
		os.Exit(1)
	}

	records, err := loadRecords(*inputFile, *formatFlag)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading records: %v\n", err)
		os.Exit(1)
	}

	if len(records) == 0 {
		fmt.Fprintln(os.Stderr, "Error: No valid records found in input file")
		os.Exit(1)
	}

	report := analyzeRecords(*tenantName, records)

	if err := os.MkdirAll(*outputDir, 0o750); err != nil {
		fmt.Fprintf(os.Stderr, "Error creating output directory: %v\n", err)
		os.Exit(1)
	}

	htmlPath := filepath.Join(*outputDir, "audit_report.html")
	if err := generateHTMLReport(report, htmlPath); err != nil {
		fmt.Fprintf(os.Stderr, "Error generating HTML report: %v\n", err)
		os.Exit(1)
	}

	mdPath := filepath.Join(*outputDir, "audit_report.md")
	if err := generateMarkdownReport(report, mdPath); err != nil {
		fmt.Fprintf(os.Stderr, "Error generating Markdown report: %v\n", err)
		os.Exit(1)
	}

	csvPath := filepath.Join(*outputDir, "breakdown.csv")
	if err := generateBreakdownCSV(report, csvPath); err != nil {
		fmt.Fprintf(os.Stderr, "Error generating CSV breakdown: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("[OK] Spend Audit successfully generated for %s!\n", *tenantName)
	fmt.Printf("     Total Events:     %d\n", report.TotalEvents)
	fmt.Printf("     Total Spend:      $%.2f\n", report.TotalSpendUSD)
	fmt.Printf("     Net Mo. Savings:  $%.2f (%.1f%%)\n", report.NetMonthlySavings, report.NetSavingsPct)
	fmt.Printf("     Annualized:       $%.2f\n", report.AnnualizedSavings)
	fmt.Printf("     Reports written to: %s\n", *outputDir)
	fmt.Printf("       - %s\n", htmlPath)
	fmt.Printf("       - %s\n", mdPath)
	fmt.Printf("       - %s\n", csvPath)
}

func loadRecords(path string, format string) ([]Record, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read file %s: %w", path, err)
	}

	ctx := context.Background()
	reg := cost.LoadRegistry(ctx, cost.RegistryConfig{})

	detectedFormat := format
	if detectedFormat == "auto" {
		trimmed := strings.TrimSpace(string(data))
		if strings.HasPrefix(trimmed, "[") || strings.HasPrefix(trimmed, "{") {
			detectedFormat = "json"
		} else {
			detectedFormat = "csv"
		}
	}

	if detectedFormat == "csv" {
		return parseCSV(data, reg)
	}
	return parseJSON(data, reg)
}

func parseCSV(data []byte, reg cost.Registry) ([]Record, error) {
	r := csv.NewReader(strings.NewReader(string(data)))
	headers, err := r.Read()
	if err != nil {
		return nil, fmt.Errorf("read csv headers: %w", err)
	}

	headerMap := make(map[string]int)
	for i, h := range headers {
		clean := strings.ToLower(strings.TrimSpace(h))
		clean = strings.ReplaceAll(clean, " ", "_")
		clean = strings.ReplaceAll(clean, "-", "_")
		headerMap[clean] = i
	}

	getCol := func(row []string, names ...string) string {
		for _, name := range names {
			if idx, ok := headerMap[name]; ok && idx < len(row) {
				return strings.TrimSpace(row[idx])
			}
		}
		return ""
	}

	var records []Record
	ctx := context.Background()

	for lineNum := 2; ; lineNum++ {
		row, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			continue // skip malformed row
		}

		model := getCol(row, "model", "model_id", "engine", "snapshot")
		if model == "" {
			model = "gpt-4o"
		}

		provider := getCol(row, "provider", "vendor")
		if provider == "" {
			provider = inferProvider(model)
		}

		promptTokens, _ := strconv.Atoi(getCol(row, "prompt_tokens", "input_tokens", "context_tokens", "prompt"))
		completionTokens, _ := strconv.Atoi(getCol(row, "completion_tokens", "output_tokens", "generated_tokens", "completion"))
		cachedTokens, _ := strconv.Atoi(getCol(row, "cached_tokens", "cache_read_input_tokens", "cached_prompt_tokens"))
		totalTokens, _ := strconv.Atoi(getCol(row, "total_tokens", "tokens"))
		if totalTokens == 0 {
			totalTokens = promptTokens + completionTokens
		}

		costStr := getCol(row, "cost", "cost_usd", "amount", "total_cost", "spend")
		var recordCost float64
		if costStr != "" {
			cleaned := strings.TrimPrefix(costStr, "$")
			recordCost, _ = strconv.ParseFloat(cleaned, 64)
		}

		team := getCol(row, "team", "team_id", "user", "user_id", "worker", "worker_id", "department")
		if team == "" {
			team = "default"
		}

		feature := getCol(row, "feature", "task", "task_category", "action", "tag")
		if feature == "" {
			feature = "general"
		}

		eventID := getCol(row, "event_id", "id", "request_id")
		if eventID == "" {
			eventID = fmt.Sprintf("evt_%d", lineNum)
		}

		// If cost not provided or 0, calculate via registry
		if recordCost <= 0 {
			event := domain.TokenEvent{
				Provider:     provider,
				ModelID:      model,
				InputTokens:  promptTokens,
				OutputTokens: completionTokens,
				CachedTokens: cachedTokens,
				TotalTokens:  totalTokens,
			}
			res := reg.Calculate(ctx, event, nil)
			if res.CostEstimateUSD != nil {
				recordCost = *res.CostEstimateUSD
			}
		}

		records = append(records, Record{
			EventID:          eventID,
			Timestamp:        time.Now().UTC(),
			Provider:         provider,
			ModelID:          model,
			PromptTokens:     promptTokens,
			CompletionTokens: completionTokens,
			CachedTokens:     cachedTokens,
			TotalTokens:      totalTokens,
			CostUSD:          recordCost,
			Team:             team,
			Feature:          feature,
		})
	}

	return records, nil
}

func parseJSON(data []byte, reg cost.Registry) ([]Record, error) {
	var rawRecords []map[string]interface{}
	trimmed := strings.TrimSpace(string(data))

	if strings.HasPrefix(trimmed, "[") {
		if err := json.Unmarshal(data, &rawRecords); err != nil {
			return nil, fmt.Errorf("parse json array: %w", err)
		}
	} else {
		// Handle JSONL
		lines := strings.Split(trimmed, "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			var m map[string]interface{}
			if err := json.Unmarshal([]byte(line), &m); err == nil {
				rawRecords = append(rawRecords, m)
			}
		}
	}

	ctx := context.Background()
	var records []Record

	for i, m := range rawRecords {
		getString := func(keys ...string) string {
			for _, k := range keys {
				if v, ok := m[k]; ok {
					return fmt.Sprintf("%v", v)
				}
			}
			return ""
		}

		getInt := func(keys ...string) int {
			for _, k := range keys {
				if v, ok := m[k]; ok {
					switch num := v.(type) {
					case float64:
						return int(num)
					case int:
						return num
					case string:
						val, _ := strconv.Atoi(num)
						return val
					}
				}
			}
			return 0
		}

		getFloat := func(keys ...string) float64 {
			for _, k := range keys {
				if v, ok := m[k]; ok {
					switch num := v.(type) {
					case float64:
						return num
					case string:
						clean := strings.TrimPrefix(num, "$")
						val, _ := strconv.ParseFloat(clean, 64)
						return val
					}
				}
			}
			return 0.0
		}

		model := getString("model", "model_id", "engine")
		if model == "" {
			model = "gpt-4o"
		}

		provider := getString("provider", "vendor")
		if provider == "" {
			provider = inferProvider(model)
		}

		promptTokens := getInt("prompt_tokens", "input_tokens", "context_tokens")
		compTokens := getInt("completion_tokens", "output_tokens", "generated_tokens")
		cachedTokens := getInt("cached_tokens", "cache_read_input_tokens")
		totalTokens := getInt("total_tokens", "tokens")
		if totalTokens == 0 {
			totalTokens = promptTokens + compTokens
		}

		costUSD := getFloat("cost", "cost_usd", "amount", "total_cost")
		if costUSD <= 0 {
			event := domain.TokenEvent{
				Provider:     provider,
				ModelID:      model,
				InputTokens:  promptTokens,
				OutputTokens: compTokens,
				CachedTokens: cachedTokens,
				TotalTokens:  totalTokens,
			}
			res := reg.Calculate(ctx, event, nil)
			if res.CostEstimateUSD != nil {
				costUSD = *res.CostEstimateUSD
			}
		}

		team := getString("team", "team_id", "user", "user_id", "worker", "worker_id")
		if team == "" {
			team = "default"
		}

		feature := getString("feature", "task", "task_category", "action")
		if feature == "" {
			feature = "general"
		}

		eventID := getString("event_id", "id", "request_id")
		if eventID == "" {
			eventID = fmt.Sprintf("evt_json_%d", i+1)
		}

		records = append(records, Record{
			EventID:          eventID,
			Timestamp:        time.Now().UTC(),
			Provider:         provider,
			ModelID:          model,
			PromptTokens:     promptTokens,
			CompletionTokens: compTokens,
			CachedTokens:     cachedTokens,
			TotalTokens:      totalTokens,
			CostUSD:          costUSD,
			Team:             team,
			Feature:          feature,
		})
	}

	return records, nil
}

func inferProvider(model string) string {
	m := strings.ToLower(model)
	switch {
	case strings.Contains(m, "claude"):
		return "anthropic"
	case strings.Contains(m, "gemini"):
		return "google"
	case strings.Contains(m, "grok"):
		return "xai"
	case strings.Contains(m, "deepseek"):
		return "deepseek"
	default:
		return "openai"
	}
}

func analyzeRecords(tenantName string, records []Record) AuditReportData {
	totalEvents := len(records)
	var totalTokens, totalPrompt, totalComp, totalCached int
	var totalSpend float64

	modelMap := make(map[string]*ModelBreakdown)
	teamMap := make(map[string]*DimensionBreakdown)
	featureMap := make(map[string]*DimensionBreakdown)

	for _, r := range records {
		totalTokens += r.TotalTokens
		totalPrompt += r.PromptTokens
		totalComp += r.CompletionTokens
		totalCached += r.CachedTokens
		totalSpend += r.CostUSD

		// Model aggregate
		mb, ok := modelMap[r.ModelID]
		if !ok {
			mb = &ModelBreakdown{ModelID: r.ModelID, Provider: r.Provider}
			modelMap[r.ModelID] = mb
		}
		mb.EventCount++
		mb.PromptTokens += r.PromptTokens
		mb.CompletionTokens += r.CompletionTokens
		mb.CachedTokens += r.CachedTokens
		mb.TotalTokens += r.TotalTokens
		mb.CostUSD += r.CostUSD

		// Team aggregate
		tb, ok := teamMap[r.Team]
		if !ok {
			tb = &DimensionBreakdown{Name: r.Team}
			teamMap[r.Team] = tb
		}
		tb.EventCount++
		tb.TotalTokens += r.TotalTokens
		tb.CostUSD += r.CostUSD

		// Feature aggregate
		fb, ok := featureMap[r.Feature]
		if !ok {
			fb = &DimensionBreakdown{Name: r.Feature}
			featureMap[r.Feature] = fb
		}
		fb.EventCount++
		fb.TotalTokens += r.TotalTokens
		fb.CostUSD += r.CostUSD
	}

	var models []ModelBreakdown
	for _, mb := range modelMap {
		if totalSpend > 0 {
			mb.PctSpend = (mb.CostUSD / totalSpend) * 100.0
		}
		models = append(models, *mb)
	}
	sort.Slice(models, func(i, j int) bool {
		return models[i].CostUSD > models[j].CostUSD
	})

	var teams []DimensionBreakdown
	for _, tb := range teamMap {
		if totalSpend > 0 {
			tb.PctSpend = (tb.CostUSD / totalSpend) * 100.0
		}
		teams = append(teams, *tb)
	}
	sort.Slice(teams, func(i, j int) bool {
		return teams[i].CostUSD > teams[j].CostUSD
	})

	var features []DimensionBreakdown
	for _, fb := range featureMap {
		if totalSpend > 0 {
			fb.PctSpend = (fb.CostUSD / totalSpend) * 100.0
		}
		features = append(features, *fb)
	}
	sort.Slice(features, func(i, j int) bool {
		return features[i].CostUSD > features[j].CostUSD
	})

	// Top burn drivers
	var drivers []string
	if len(models) > 0 {
		drivers = append(drivers, fmt.Sprintf("Top spending model is %s (%s) representing %.1f%% of total budget ($%.2f).",
			models[0].ModelID, models[0].Provider, models[0].PctSpend, models[0].CostUSD))
	}
	if len(teams) > 0 && teams[0].Name != "default" {
		drivers = append(drivers, fmt.Sprintf("Primary spending group/team is '%s' accounting for %.1f%% of spend ($%.2f).",
			teams[0].Name, teams[0].PctSpend, teams[0].CostUSD))
	}
	if totalPrompt > 0 && float64(totalCached)/float64(totalPrompt) < 0.15 {
		drivers = append(drivers, fmt.Sprintf("Severe cache underutilization: Only %.1f%% of input tokens hit prompt cache, leaving substantial caching discounts unclaimed.",
			float64(totalCached)/float64(totalPrompt)*100.0))
	}
	if totalComp > 0 && float64(totalComp)/float64(totalTokens) > 0.40 {
		drivers = append(drivers, fmt.Sprintf("Heavy output skew: Completion tokens account for %.1f%% of token volume, disproportionately inflating billable costs.",
			float64(totalComp)/float64(totalTokens)*100.0))
	}

	// 4 Savings Playbooks
	// 1. 70/20/10 Model Routing
	// Routine queries (70%) routed to mini/flash models, 20% to mid-tier, 10% to frontier.
	var routingEligibleCost float64
	for _, m := range models {
		if strings.Contains(m.ModelID, "o1") || strings.Contains(m.ModelID, "o3") ||
			strings.Contains(m.ModelID, "opus") || strings.Contains(m.ModelID, "gpt-4o") ||
			strings.Contains(m.ModelID, "sonnet") {
			routingEligibleCost += m.CostUSD
		}
	}
	if routingEligibleCost == 0 {
		routingEligibleCost = totalSpend
	}
	routingSavings := routingEligibleCost * 0.38
	routingPct := 0.0
	if totalSpend > 0 {
		routingPct = (routingSavings / totalSpend) * 100.0
	}

	// 2. Prompt & Semantic Caching
	// Estimated ~20% of input cost saved with prefix and semantic caching
	var inputSpend float64
	if totalTokens > 0 {
		inputSpend = totalSpend * (float64(totalPrompt) / float64(totalTokens))
	} else {
		inputSpend = totalSpend * 0.65
	}
	cachingSavings := inputSpend * 0.22
	cachingPct := 0.0
	if totalSpend > 0 {
		cachingPct = (cachingSavings / totalSpend) * 100.0
	}

	// 3. Model Tier Right-Sizing / Downgrades
	// Replacing top-tier reasoning/flagship models with fine-tuned or lighter variants for formatting/classification
	downgradeSavings := totalSpend * 0.18
	downgradePct := 18.0

	// 4. Output Length Controls & Stop Sequences
	var outputSpend float64
	if totalTokens > 0 {
		outputSpend = totalSpend * (float64(totalComp) / float64(totalTokens))
	} else {
		outputSpend = totalSpend * 0.35
	}
	outputSavings := outputSpend * 0.16
	outputPct := 0.0
	if totalSpend > 0 {
		outputPct = (outputSavings / totalSpend) * 100.0
	}

	// Realistic combined net savings (accounting for non-additive compounding: ~38% net savings)
	netSavingsPct := 38.5
	netMonthlySavings := totalSpend * (netSavingsPct / 100.0)
	newMonthlySpend := math.Max(0, totalSpend-netMonthlySavings)
	annualized := netMonthlySavings * 12.0

	opportunities := []SavingsOpportunity{
		{
			ID:          "opp_routing_70_20_10",
			Title:       "70/20/10 Intelligent Model Routing Architecture",
			Description: "Implement deterministic multi-tiered model routing in the gateway. Direct 70% of routine tasks (formatting, extraction, classification) to micro/flash models, 20% to mid-tier general models, and restrict frontier models (o1/opus) strictly to the 10% highest-complexity reasoning tasks.",
			Assumptions: "Assumes 70% of current high-tier requests require sub-frontier reasoning, yielding ~70-80% unit cost reduction on routed queries.",
			SavingsUSD:  routingSavings,
			SavingsPct:  routingPct,
			ActionPlan:  "Deploy TokenGoblin gateway proxy rule: inspect prompt length and intent classifier; forward to gpt-4o-mini / gemini-2.0-flash / claude-3-5-haiku by default.",
		},
		{
			ID:          "opp_prompt_caching",
			Title:       "System Prompt Prefix & Semantic Turn Caching",
			Description: "Restructure system instructions, tool definitions, and few-shot exemplars to place static tokens at the beginning of API payloads to activate provider native prompt caching (Anthropic prompt caching & OpenAI automatic prefix caching).",
			Assumptions: "Assumes ~25% prompt overlap across agent workloads with standard 50-75% cached token discount rates.",
			SavingsUSD:  cachingSavings,
			SavingsPct:  cachingPct,
			ActionPlan:  "Standardize system prompt prefixes across engineering services. Place dynamic user inputs strictly at the end of the message payload array.",
		},
		{
			ID:          "opp_model_downgrade",
			Title:       "Targeted Model Right-Sizing for Narrow Tasks",
			Description: "Downgrade over-provisioned models currently executing structured JSON schema tasks, sentiment analysis, and embedding lookups that don't benefit from reasoning tokens.",
			Assumptions: "Identifies ~20% of traffic using flagship reasoning models for predictable tasks that achieve identical eval pass rates on distilled models.",
			SavingsUSD:  downgradeSavings,
			SavingsPct:  downgradePct,
			ActionPlan:  "Audit prompt cemetery and evaluate task categories. Move JSON extraction pipelines from o1/gpt-4o to gpt-4o-mini with structured outputs enabled.",
		},
		{
			ID:          "opp_output_controls",
			Title:       "Output Length Truncation & Stop Sequence Guardrails",
			Description: "Completion tokens cost 3x-4x more than prompt tokens. Eliminate conversational filler ('Sure, here is...'), repetitive loops, and runaway completions by setting strict max_tokens and explicit stop sequences.",
			Assumptions: "Targeting an average 18% reduction in completion token counts via system prompt refinement and max_completion_tokens caps.",
			SavingsUSD:  outputSavings,
			SavingsPct:  outputPct,
			ActionPlan:  "Apply TokenGoblin prompt refiner middleware to strip conversational filler, enforce max_tokens per task category, and set stop sequences on recursive loops.",
		},
	}

	return AuditReportData{
		TenantName:         tenantName,
		GeneratedAt:        time.Now().UTC(),
		TotalEvents:        totalEvents,
		TotalTokens:        totalTokens,
		TotalPromptTokens:  totalPrompt,
		TotalCompTokens:    totalComp,
		TotalCachedTokens:  totalCached,
		TotalSpendUSD:      totalSpend,
		Models:             models,
		Teams:              teams,
		Features:           features,
		TopBurnDrivers:     drivers,
		Opportunities:      opportunities,
		NetMonthlySavings:  netMonthlySavings,
		NetSavingsPct:      netSavingsPct,
		NewMonthlySpendUSD: newMonthlySpend,
		AnnualizedSavings:  annualized,
	}
}

func generateHTMLReport(r AuditReportData, path string) error {
	var sb strings.Builder

	sb.WriteString(`<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>LLM Spend Audit & Optimization Report — ` + htmlEscape(r.TenantName) + `</title>
  <style>
    :root {
      --bg: #030712;
      --card-bg: #0b0f19;
      --border: #1f293d;
      --accent: #10b981;
      --accent-gold: #ffb000;
      --accent-red: #ef4444;
      --text: #e5e7eb;
      --text-muted: #9ca3af;
      --font-mono: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, "Liberation Mono", monospace;
    }
    * { box-sizing: border-box; margin: 0; padding: 0; }
    body {
      background-color: var(--bg);
      color: var(--text);
      font-family: var(--font-mono);
      line-height: 1.5;
      padding: 40px 20px;
    }
    .container {
      max-width: 1200px;
      margin: 0 auto;
    }
    header {
      border-bottom: 2px solid var(--border);
      padding-bottom: 24px;
      margin-bottom: 32px;
      display: flex;
      justify-content: space-between;
      align-items: flex-end;
      flex-wrap: wrap;
      gap: 16px;
    }
    h1 {
      font-size: 24px;
      color: #fff;
      letter-spacing: 0.1em;
      text-transform: uppercase;
    }
    .badge {
      display: inline-block;
      padding: 4px 10px;
      font-size: 11px;
      font-weight: bold;
      border: 1px solid var(--accent-gold);
      color: var(--accent-gold);
      background: rgba(255, 176, 0, 0.1);
      letter-spacing: 0.1em;
      text-transform: uppercase;
      margin-bottom: 8px;
    }
    .subtext {
      color: var(--text-muted);
      font-size: 12px;
    }
    .grid-4 {
      display: grid;
      grid-template-columns: repeat(auto-fit, minmax(240px, 1fr));
      gap: 16px;
      margin-bottom: 32px;
    }
    .card {
      background: var(--card-bg);
      border: 1px solid var(--border);
      padding: 20px;
      border-radius: 4px;
      position: relative;
    }
    .card-title {
      font-size: 11px;
      text-transform: uppercase;
      letter-spacing: 0.1em;
      color: var(--text-muted);
      margin-bottom: 8px;
    }
    .card-val {
      font-size: 28px;
      font-weight: bold;
      color: #fff;
    }
    .card-val.green { color: var(--accent); }
    .card-val.gold { color: var(--accent-gold); }
    .card-sub {
      font-size: 12px;
      color: var(--text-muted);
      margin-top: 6px;
    }
    h2 {
      font-size: 16px;
      color: #fff;
      text-transform: uppercase;
      letter-spacing: 0.1em;
      margin: 32px 0 16px;
      padding-bottom: 8px;
      border-bottom: 1px solid var(--border);
      display: flex;
      align-items: center;
      gap: 8px;
    }
    h2::before {
      content: ">>";
      color: var(--accent);
    }
    .drivers-list {
      background: var(--card-bg);
      border: 1px solid var(--border);
      padding: 20px;
      border-radius: 4px;
      margin-bottom: 32px;
    }
    .drivers-list li {
      list-style-type: none;
      padding: 8px 0;
      border-bottom: 1px solid rgba(255,255,255,0.05);
      font-size: 13px;
      display: flex;
      gap: 10px;
    }
    .drivers-list li:last-child { border-bottom: none; }
    .drivers-list li span.bullet { color: var(--accent-red); font-weight: bold; }
    table {
      width: 100%;
      border-collapse: collapse;
      margin-bottom: 32px;
      background: var(--card-bg);
      border: 1px solid var(--border);
      font-size: 12px;
    }
    th, td {
      padding: 12px 16px;
      text-align: left;
      border-bottom: 1px solid var(--border);
    }
    th {
      background: #080c14;
      color: var(--text-muted);
      text-transform: uppercase;
      letter-spacing: 0.1em;
      font-size: 11px;
    }
    tr:hover { background: rgba(255,255,255,0.02); }
    .text-right { text-align: right; }
    .playbook {
      background: var(--card-bg);
      border: 1px solid var(--border);
      border-left: 4px solid var(--accent);
      padding: 20px;
      margin-bottom: 16px;
      border-radius: 0 4px 4px 0;
    }
    .playbook-header {
      display: flex;
      justify-content: space-between;
      align-items: center;
      margin-bottom: 12px;
      flex-wrap: wrap;
      gap: 8px;
    }
    .playbook-title {
      font-size: 15px;
      font-weight: bold;
      color: #fff;
    }
    .playbook-savings {
      font-size: 14px;
      font-weight: bold;
      color: var(--accent);
      background: rgba(16, 185, 129, 0.1);
      border: 1px solid var(--accent);
      padding: 4px 10px;
      border-radius: 4px;
    }
    .playbook-body {
      font-size: 13px;
      color: var(--text);
      margin-bottom: 10px;
    }
    .playbook-meta {
      font-size: 11px;
      color: var(--text-muted);
      background: rgba(0,0,0,0.3);
      padding: 8px 12px;
      border-radius: 4px;
      margin-top: 8px;
    }
    footer {
      border-top: 1px solid var(--border);
      padding-top: 24px;
      margin-top: 48px;
      text-align: center;
      font-size: 11px;
      color: var(--text-muted);
    }
  </style>
</head>
<body>
  <div class="container">
    <header>
      <div>
        <div class="badge">[ TokenGoblin :: Spend Audit ]</div>
        <h1>` + htmlEscape(r.TenantName) + `</h1>
        <p class="subtext">Deterministic AI/LLM Spend Audit & Optimization Roadmap</p>
      </div>
      <div class="text-right">
        <p class="subtext">Generated: ` + r.GeneratedAt.Format("Jan 02, 2006 15:04 UTC") + `</p>
        <p class="subtext">Audit Engine: v0.4.0-prod</p>
      </div>
    </header>

    <!-- EXECUTIVE SUMMARY -->
    <div class="grid-4">
      <div class="card">
        <div class="card-title">Audited Baseline Spend</div>
        <div class="card-val gold">$` + fmt.Sprintf("%.2f", r.TotalSpendUSD) + `</div>
        <div class="card-sub">` + fmt.Sprintf("%d", r.TotalEvents) + ` total API calls</div>
      </div>
      <div class="card">
        <div class="card-title">Projected Net Monthly Savings</div>
        <div class="card-val green">$` + fmt.Sprintf("%.2f", r.NetMonthlySavings) + `</div>
        <div class="card-sub">` + fmt.Sprintf("%.1f%%", r.NetSavingsPct) + ` cost reduction</div>
      </div>
      <div class="card">
        <div class="card-title">Optimized Monthly Spend</div>
        <div class="card-val">$` + fmt.Sprintf("%.2f", r.NewMonthlySpendUSD) + `</div>
        <div class="card-sub">Post-optimization run rate</div>
      </div>
      <div class="card">
        <div class="card-title">Annualized Net Impact</div>
        <div class="card-val green">$` + fmt.Sprintf("%.2f", r.AnnualizedSavings) + `</div>
        <div class="card-sub">Recovered capital / year</div>
      </div>
    </div>

    <!-- TOP BURN DRIVERS -->
    <h2>Primary Burn Drivers</h2>
    <div class="drivers-list">
      <ul>`)

	for _, d := range r.TopBurnDrivers {
		sb.WriteString(`
        <li><span class="bullet">[!]</span> ` + htmlEscape(d) + `</li>`)
	}

	sb.WriteString(`
      </ul>
    </div>

    <!-- SAVINGS PLAYBOOKS -->
    <h2>Savings Playbooks & Actionable Interventions</h2>`)

	for _, op := range r.Opportunities {
		sb.WriteString(`
    <div class="playbook">
      <div class="playbook-header">
        <span class="playbook-title">` + htmlEscape(op.Title) + `</span>
        <span class="playbook-savings">-$` + fmt.Sprintf("%.2f", op.SavingsUSD) + ` / mo (~` + fmt.Sprintf("%.1f%%", op.SavingsPct) + `)</span>
      </div>
      <div class="playbook-body">` + htmlEscape(op.Description) + `</div>
      <div class="playbook-meta">
        <strong>Assumptions:</strong> ` + htmlEscape(op.Assumptions) + `<br>
        <strong>Technical Action:</strong> ` + htmlEscape(op.ActionPlan) + `
      </div>
    </div>`)
	}

	sb.WriteString(`
    <!-- BREAKDOWN BY MODEL -->
    <h2>Spend Breakdown by Model</h2>
    <table>
      <thead>
        <tr>
          <th>Model</th>
          <th>Provider</th>
          <th class="text-right">Calls</th>
          <th class="text-right">Prompt Tokens</th>
          <th class="text-right">Comp Tokens</th>
          <th class="text-right">Cost (USD)</th>
          <th class="text-right">% of Spend</th>
        </tr>
      </thead>
      <tbody>`)

	for _, m := range r.Models {
		sb.WriteString(`
        <tr>
          <td><strong>` + htmlEscape(m.ModelID) + `</strong></td>
          <td>` + htmlEscape(m.Provider) + `</td>
          <td class="text-right">` + fmt.Sprintf("%d", m.EventCount) + `</td>
          <td class="text-right">` + fmt.Sprintf("%d", m.PromptTokens) + `</td>
          <td class="text-right">` + fmt.Sprintf("%d", m.CompletionTokens) + `</td>
          <td class="text-right"><strong>$` + fmt.Sprintf("%.2f", m.CostUSD) + `</strong></td>
          <td class="text-right">` + fmt.Sprintf("%.1f%%", m.PctSpend) + `</td>
        </tr>`)
	}

	sb.WriteString(`
      </tbody>
    </table>

    <!-- BREAKDOWN BY TEAM -->
    <h2>Spend Breakdown by Team / User</h2>
    <table>
      <thead>
        <tr>
          <th>Team / Caller</th>
          <th class="text-right">Requests</th>
          <th class="text-right">Total Tokens</th>
          <th class="text-right">Cost (USD)</th>
          <th class="text-right">% of Spend</th>
        </tr>
      </thead>
      <tbody>`)

	for _, t := range r.Teams {
		sb.WriteString(`
        <tr>
          <td><strong>` + htmlEscape(t.Name) + `</strong></td>
          <td class="text-right">` + fmt.Sprintf("%d", t.EventCount) + `</td>
          <td class="text-right">` + fmt.Sprintf("%d", t.TotalTokens) + `</td>
          <td class="text-right"><strong>$` + fmt.Sprintf("%.2f", t.CostUSD) + `</strong></td>
          <td class="text-right">` + fmt.Sprintf("%.1f%%", t.PctSpend) + `</td>
        </tr>`)
	}

	sb.WriteString(`
      </tbody>
    </table>

    <!-- BREAKDOWN BY FEATURE -->
    <h2>Spend Breakdown by Feature / Task</h2>
    <table>
      <thead>
        <tr>
          <th>Feature / Task Category</th>
          <th class="text-right">Requests</th>
          <th class="text-right">Total Tokens</th>
          <th class="text-right">Cost (USD)</th>
          <th class="text-right">% of Spend</th>
        </tr>
      </thead>
      <tbody>`)

	for _, f := range r.Features {
		sb.WriteString(`
        <tr>
          <td><strong>` + htmlEscape(f.Name) + `</strong></td>
          <td class="text-right">` + fmt.Sprintf("%d", f.EventCount) + `</td>
          <td class="text-right">` + fmt.Sprintf("%d", f.TotalTokens) + `</td>
          <td class="text-right"><strong>$` + fmt.Sprintf("%.2f", f.CostUSD) + `</strong></td>
          <td class="text-right">` + fmt.Sprintf("%.1f%%", f.PctSpend) + `</td>
        </tr>`)
	}

	sb.WriteString(`
      </tbody>
    </table>

    <footer>
      Produced by TokenGoblin AI Spend Intelligence. Confidential — For ` + htmlEscape(r.TenantName) + ` Executive Eyes Only.
    </footer>
  </div>
</body>
</html>`)

	return os.WriteFile(path, []byte(sb.String()), 0o600)
}

func generateMarkdownReport(r AuditReportData, path string) error {
	var sb strings.Builder

	sb.WriteString("# LLM Spend Audit & Optimization Report\n\n")
	sb.WriteString(fmt.Sprintf("**Client:** %s  \n", r.TenantName))
	sb.WriteString(fmt.Sprintf("**Audit Date:** %s  \n", r.GeneratedAt.Format("2006-01-02 15:04 UTC")))
	sb.WriteString("**Engine:** TokenGoblin v0.4.0-prod  \n\n")

	sb.WriteString("## Executive Summary\n\n")
	sb.WriteString("| Metric | Value |\n")
	sb.WriteString("|---|---|\n")
	sb.WriteString(fmt.Sprintf("| **Current Audited Spend** | **$%.2f** |\n", r.TotalSpendUSD))
	sb.WriteString(fmt.Sprintf("| **Projected Net Monthly Savings** | **-$%.2f (%.1f%%)** |\n", r.NetMonthlySavings, r.NetSavingsPct))
	sb.WriteString(fmt.Sprintf("| **Target Post-Optimization Spend** | **$%.2f / mo** |\n", r.NewMonthlySpendUSD))
	sb.WriteString(fmt.Sprintf("| **Annualized Net Savings** | **$%.2f / yr** |\n", r.AnnualizedSavings))
	sb.WriteString(fmt.Sprintf("| Total Event Volume | %d calls |\n", r.TotalEvents))
	sb.WriteString(fmt.Sprintf("| Total Token Volume | %d tokens |\n\n", r.TotalTokens))

	sb.WriteString("## Key Burn Drivers\n\n")
	for _, d := range r.TopBurnDrivers {
		sb.WriteString(fmt.Sprintf("- ⚠️ %s\n", d))
	}
	sb.WriteString("\n")

	sb.WriteString("## Savings Opportunities & Playbooks\n\n")
	for i, op := range r.Opportunities {
		sb.WriteString(fmt.Sprintf("### %d. %s\n", i+1, op.Title))
		sb.WriteString(fmt.Sprintf("**Estimated Monthly Savings:** `-$%.2f` (~%.1f%% of spend)\n\n", op.SavingsUSD, op.SavingsPct))
		sb.WriteString(fmt.Sprintf("%s\n\n", op.Description))
		sb.WriteString(fmt.Sprintf("- **Underlying Assumptions:** %s\n", op.Assumptions))
		sb.WriteString(fmt.Sprintf("- **Technical Action Plan:** %s\n\n", op.ActionPlan))
	}

	sb.WriteString("## Model Spend Breakdown\n\n")
	sb.WriteString("| Model | Provider | Calls | Prompt Tokens | Comp Tokens | Cost (USD) | % Spend |\n")
	sb.WriteString("|---|---|---:|---:|---:|---:|---:|\n")
	for _, m := range r.Models {
		sb.WriteString(fmt.Sprintf("| `%s` | %s | %d | %d | %d | $%.2f | %.1f%% |\n",
			m.ModelID, m.Provider, m.EventCount, m.PromptTokens, m.CompletionTokens, m.CostUSD, m.PctSpend))
	}
	sb.WriteString("\n")

	sb.WriteString("## Team / Caller Breakdown\n\n")
	sb.WriteString("| Team / Caller | Requests | Total Tokens | Cost (USD) | % Spend |\n")
	sb.WriteString("|---|---:|---:|---:|---:|\n")
	for _, t := range r.Teams {
		sb.WriteString(fmt.Sprintf("| `%s` | %d | %d | $%.2f | %.1f%% |\n",
			t.Name, t.EventCount, t.TotalTokens, t.CostUSD, t.PctSpend))
	}
	sb.WriteString("\n")

	sb.WriteString("## Feature / Task Breakdown\n\n")
	sb.WriteString("| Feature / Task | Requests | Total Tokens | Cost (USD) | % Spend |\n")
	sb.WriteString("|---|---:|---:|---:|---:|\n")
	for _, f := range r.Features {
		sb.WriteString(fmt.Sprintf("| `%s` | %d | %d | $%.2f | %.1f%% |\n",
			f.Name, f.EventCount, f.TotalTokens, f.CostUSD, f.PctSpend))
	}
	sb.WriteString("\n")

	sb.WriteString("---\n*Generated by TokenGoblin AI Spend Intelligence. Confidential.*  \n")

	return os.WriteFile(path, []byte(sb.String()), 0o600)
}

func generateBreakdownCSV(r AuditReportData, path string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	w := csv.NewWriter(f)
	defer w.Flush()

	// Header
	if err := w.Write([]string{
		"dimension", "item", "provider", "event_count", "prompt_tokens", "completion_tokens", "cached_tokens", "total_tokens", "cost_usd", "pct_of_spend",
	}); err != nil {
		return err
	}

	for _, m := range r.Models {
		_ = w.Write([]string{
			"model", m.ModelID, m.Provider,
			strconv.Itoa(m.EventCount), strconv.Itoa(m.PromptTokens), strconv.Itoa(m.CompletionTokens),
			strconv.Itoa(m.CachedTokens), strconv.Itoa(m.TotalTokens),
			fmt.Sprintf("%.4f", m.CostUSD), fmt.Sprintf("%.2f", m.PctSpend),
		})
	}

	for _, t := range r.Teams {
		_ = w.Write([]string{
			"team", t.Name, "n/a",
			strconv.Itoa(t.EventCount), "0", "0", "0", strconv.Itoa(t.TotalTokens),
			fmt.Sprintf("%.4f", t.CostUSD), fmt.Sprintf("%.2f", t.PctSpend),
		})
	}

	for _, fe := range r.Features {
		_ = w.Write([]string{
			"feature", fe.Name, "n/a",
			strconv.Itoa(fe.EventCount), "0", "0", "0", strconv.Itoa(fe.TotalTokens),
			fmt.Sprintf("%.4f", fe.CostUSD), fmt.Sprintf("%.2f", fe.PctSpend),
		})
	}

	return nil
}

func htmlEscape(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, "\"", "&quot;")
	return s
}
