package clickhouse

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/Hardonian/TokenGoblin/internal/domain"
)

// EventSink mirrors token usage events into ClickHouse for high-volume
// analytics. It is a SECONDARY store: the relational repository (Postgres /
// SQLite) remains the source of truth, and a sink failure must never fail or
// roll back the primary write path.
type EventSink struct {
	client *ClickHouseClient
}

// NewEventSink wraps a client. InitSchema must have succeeded before sinking.
func NewEventSink(client *ClickHouseClient) *EventSink {
	return &EventSink{client: client}
}

// EventSinkConfigFromEnv reads TG_CLICKHOUSE_ADDR (comma-separated host:port
// list), TG_CLICKHOUSE_DATABASE, TG_CLICKHOUSE_USER, TG_CLICKHOUSE_PASSWORD.
// Returns (nil, nil) when TG_CLICKHOUSE_ADDR is unset — telemetry disabled.
func EventSinkConfigFromEnv() (*ClickHouseConfig, bool) {
	addr := strings.TrimSpace(envOr("TG_CLICKHOUSE_ADDR", ""))
	if addr == "" {
		return nil, false
	}
	cfg := DefaultConfig()
	cfg.Addresses = strings.Split(addr, ",")
	cfg.Database = envOr("TG_CLICKHOUSE_DATABASE", cfg.Database)
	cfg.Username = envOr("TG_CLICKHOUSE_USER", cfg.Username)
	cfg.Password = envOr("TG_CLICKHOUSE_PASSWORD", cfg.Password)
	return &cfg, true
}

func envOr(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}

// InitSchema ensures the configured database exists and applies the
// TokenGoblin DDL (idempotent: every statement is CREATE ... IF NOT EXISTS).
func (s *EventSink) InitSchema(ctx context.Context) error {
	cfg := s.client.config
	if !validIdentifier(cfg.Database) {
		return fmt.Errorf("init clickhouse schema: invalid database name %q", cfg.Database)
	}
	if err := s.client.Execute(ctx, "CREATE DATABASE IF NOT EXISTS "+cfg.Database); err != nil {
		return fmt.Errorf("init clickhouse database: %w", err)
	}
	for _, stmt := range splitStatements(SchemaDDL) {
		if err := s.client.Execute(ctx, stmt); err != nil {
			return fmt.Errorf("init clickhouse schema: %w", err)
		}
	}
	return nil
}

// validIdentifier guards the interpolated CREATE DATABASE identifier.
func validIdentifier(name string) bool {
	if name == "" {
		return false
	}
	for _, r := range name {
		if (r < 'a' || r > 'z') && (r < 'A' || r > 'Z') && (r < '0' || r > '9') && r != '_' {
			return false
		}
	}
	return true
}

// splitStatements splits the DDL block on ';' boundaries and drops empty or
// comment-only chunks (the DDL ends with a commented-out ALTER statement).
func splitStatements(ddl string) []string {
	var stmts []string
	for _, chunk := range strings.Split(ddl, ";") {
		var kept []string
		for _, line := range strings.Split(chunk, "\n") {
			trimmed := strings.TrimSpace(line)
			if trimmed == "" || strings.HasPrefix(trimmed, "--") {
				continue
			}
			kept = append(kept, line)
		}
		if len(kept) == 0 {
			continue
		}
		stmts = append(stmts, strings.TrimSpace(strings.Join(kept, "\n")))
	}
	return stmts
}

// SinkTokenEvent mirrors one event into the token_events table. Metadata keeps
// the dimension fields (provider, worker, run, cost quality) queryable via the
// Map(String, String) column.
func (s *EventSink) SinkTokenEvent(ctx context.Context, e domain.TokenEvent) error {
	row := buildEventRow(e)
	return s.client.Insert(ctx, "token_events", tokenEventColumns, [][]any{row})
}

var tokenEventColumns = []string{
	"id", "tenant_id", "user_id", "model", "feature",
	"prompt_tokens", "completion_tokens", "total_tokens",
	"cost_usd", "timestamp", "prompt_fingerprint", "metadata",
}

// buildEventRow maps a domain event onto the token_events column order.
func buildEventRow(e domain.TokenEvent) []any {
	cost := 0.0
	if e.CostEstimateUSD != nil {
		cost = *e.CostEstimateUSD
	}
	feature := e.TaskCategory
	if feature == "" {
		feature = e.TaskType
	}
	total := e.TotalTokens
	if total == 0 {
		total = e.PromptTokens + e.CompletionTokens
	}
	ts := e.Timestamp
	if ts.IsZero() {
		ts = e.CreatedAt
	}
	return []any{
		e.EventID,
		e.TenantID,
		e.WorkerID,
		e.ModelID,
		feature,
		tokenCount(e.PromptTokens),
		tokenCount(e.CompletionTokens),
		tokenCount(total),
		cost,
		ts,
		e.Fingerprint,
		buildMetadata(e),
	}
}

// tokenCount adapts a validated token count to the UInt64 columns. G115 is a
// false positive on this conversion: the value is clamped non-negative and an
// int's maximum always fits in uint64, so overflow is impossible — the only
// hazard (negative wraparound) is removed by the clamp.
// #nosec G115
func tokenCount(v int) uint64 {
	if v < 0 {
		return 0
	}
	return uint64(v)
}

func buildMetadata(e domain.TokenEvent) map[string]string {
	meta := map[string]string{}
	for k, v := range e.Tags {
		if k != "" && v != "" {
			meta[k] = v
		}
	}
	put := func(k, v string) {
		if v != "" {
			meta[k] = v
		}
	}
	put("provider", e.Provider)
	put("worker_name", e.WorkerName)
	put("job_id", e.JobID)
	put("session_id", e.SessionID)
	put("run_id", e.RunID)
	put("task_type", e.TaskType)
	put("output_status", string(e.OutputStatus))
	put("cost_currency", e.CostCurrency)
	if e.CostIsDegraded {
		put("cost_degraded_code", e.CostDegradedCode)
	}
	if e.ReviewScore != nil {
		put("review_score", fmt.Sprintf("%g", *e.ReviewScore))
	}
	return meta
}

// pingDeadline bounds health checks against a slow/unavailable cluster.
const pingDeadline = 5 * time.Second

// MaybeEventSinkFromEnv builds a schema-initialized sink when
// TG_CLICKHOUSE_ADDR is set; returns (nil, nil) when telemetry is not
// configured. Connection or schema failure returns an error — callers decide
// whether to degrade (server) or fail (seed verification).
func MaybeEventSinkFromEnv(ctx context.Context) (*EventSink, error) {
	cfg, ok := EventSinkConfigFromEnv()
	if !ok {
		return nil, nil
	}
	if !validIdentifier(cfg.Database) {
		return nil, fmt.Errorf("clickhouse: invalid database name %q", cfg.Database)
	}
	// Bootstrap: the driver connects WITH the target database, so a fresh
	// cluster must have the database created via the 'default' database first.
	bootstrapCfg := *cfg
	bootstrapCfg.Database = "default"
	bootstrap, err := NewClient(bootstrapCfg)
	if err != nil {
		return nil, err
	}
	if err := bootstrap.Execute(ctx, "CREATE DATABASE IF NOT EXISTS "+cfg.Database); err != nil {
		_ = bootstrap.Close()
		return nil, fmt.Errorf("clickhouse bootstrap: %w", err)
	}
	_ = bootstrap.Close()

	client, err := NewClient(*cfg)
	if err != nil {
		return nil, err
	}
	sink := NewEventSink(client)
	if err := sink.InitSchema(ctx); err != nil {
		_ = client.Close()
		return nil, err
	}
	return sink, nil
}

// Close releases the underlying pool.
func (s *EventSink) Close() error {
	return s.client.Close()
}

// SinkAnomalySignals mirrors detected anomaly signals into the anomalies
// table. Same best-effort contract as SinkTokenEvent.
func (s *EventSink) SinkAnomalySignals(ctx context.Context, signals []domain.AnomalySignal) error {
	if len(signals) == 0 {
		return nil
	}
	rows := make([][]any, 0, len(signals))
	for _, signal := range signals {
		rows = append(rows, buildAnomalyRow(signal))
	}
	return s.client.Insert(ctx, "anomalies", anomalyColumns, rows)
}

var anomalyColumns = []string{
	"id", "tenant_id", "anomaly_type", "severity", "description",
	"timestamp", "metric_value", "threshold", "metadata",
}

// buildAnomalyRow maps a domain signal onto the anomalies column order.
// EventID/WorkerID and the free-form Details land in the metadata map (the
// schema has no dedicated columns for them).
func buildAnomalyRow(signal domain.AnomalySignal) []any {
	observed := 0.0
	if signal.ObservedValue != nil {
		observed = *signal.ObservedValue
	}
	threshold := 0.0
	if signal.ThresholdValue != nil {
		threshold = *signal.ThresholdValue
	}
	meta := map[string]string{}
	if signal.EventID != "" {
		meta["event_id"] = signal.EventID
	}
	if signal.WorkerID != "" {
		meta["worker_id"] = signal.WorkerID
	}
	for k, v := range signal.Details {
		if k != "" && v != nil {
			meta[k] = fmt.Sprintf("%v", v)
		}
	}
	return []any{
		signal.AnomalyID,
		signal.TenantID,
		string(signal.Type),
		string(signal.Severity),
		signal.Description,
		signal.DetectedAt,
		observed,
		threshold,
		meta,
	}
}

// DeleteTenantEvents clears a tenant's mirrored data. The token_events clear
// is a lightweight delete — synchronous in visibility and REQUIRED (the API
// and seed verification read that table). The aggregate tables are cleared
// with async mutations (lightweight DELETE is unsupported on them) as
// best-effort hygiene only.
func (s *EventSink) DeleteTenantEvents(ctx context.Context, tenantID string) error {
	if err := s.client.Execute(ctx, "DELETE FROM token_events WHERE tenant_id = ?", tenantID); err != nil {
		return fmt.Errorf("clear clickhouse mirror token_events: %w", err)
	}
	if err := s.client.Execute(ctx, "DELETE FROM anomalies WHERE tenant_id = ?", tenantID); err != nil {
		return fmt.Errorf("clear clickhouse mirror anomalies: %w", err)
	}
	for _, table := range []string{"usage_aggregates_hourly", "usage_aggregates_daily", "usage_aggregates"} {
		if err := s.client.Execute(ctx, "ALTER TABLE "+table+" DELETE WHERE tenant_id = ?", tenantID); err != nil {
			// Hygiene only: an async mutation failure must not fail the reset.
			fmt.Fprintf(os.Stderr, "clickhouse mirror hygiene: clear %s: %v\n", table, err)
		}
	}
	return nil
}

// CountTokenEvents returns how many mirrored events exist for a tenant —
// used by verification tooling to confirm what actually landed.
func (s *EventSink) CountTokenEvents(ctx context.Context, tenantID string) (int64, error) {
	rowsAny, err := s.client.Query(ctx,
		"SELECT count() FROM token_events WHERE tenant_id = ?", tenantID)
	if err != nil {
		return 0, fmt.Errorf("count clickhouse token events: %w", err)
	}
	rows, ok := rowsAny.(driver.Rows)
	if !ok {
		return 0, fmt.Errorf("count clickhouse token events: unexpected rows type %T", rowsAny)
	}
	defer func() { _ = rows.Close() }()
	if !rows.Next() {
		return 0, fmt.Errorf("count clickhouse token events: no result row")
	}
	var count uint64
	if err := rows.Scan(&count); err != nil {
		return 0, fmt.Errorf("count clickhouse token events: %w", err)
	}
	return int64(count), nil
}

// Ready reports whether the sink can accept writes right now.
func (s *EventSink) Ready(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, pingDeadline)
	defer cancel()
	return s.client.Health(ctx)
}
