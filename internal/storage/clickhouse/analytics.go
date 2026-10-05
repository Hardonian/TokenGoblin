package clickhouse

import (
	"context"
	"fmt"
	"time"
)

// Analytics answers cost-intelligence queries from the mirrored token_events.
// It reads ONLY the ClickHouse mirror: the relational repository stays the
// transactional source of truth, and an analytics query must never fabricate
// results when the mirror is missing data.
type Analytics struct {
	client *ClickHouseClient
}

// NewAnalytics wraps a client for read-only analytics queries.
func NewAnalytics(client *ClickHouseClient) *Analytics {
	return &Analytics{client: client}
}

// Analytics returns a read-only query surface over the same connection pool.
func (s *EventSink) Analytics() *Analytics {
	return NewAnalytics(s.client)
}

// Query errors are wrapped so callers can surface a stable message.
type QueryError struct {
	Op  string
	Err error
}

func (e *QueryError) Error() string { return fmt.Sprintf("clickhouse analytics %s: %v", e.Op, e.Err) }
func (e *QueryError) Unwrap() error { return e.Err }

// GetCostByTenant aggregates cost over a window.
func (a *Analytics) GetCostByTenant(ctx context.Context, tenantID string, start, end time.Time) (CostSummary, error) {
	rowsAny, err := a.client.Query(ctx,
		`SELECT sum(cost_usd), sum(total_tokens), count()
		 FROM token_events
		 WHERE tenant_id = ? AND timestamp >= ? AND timestamp < ?`,
		tenantID, start, end)
	if err != nil {
		return CostSummary{}, &QueryError{"cost_by_tenant", err}
	}
	rows, ok := rowsAny.(interface {
		Next() bool
		Scan(...any) error
		Close() error
	})
	if !ok {
		return CostSummary{}, &QueryError{"cost_by_tenant", fmt.Errorf("unexpected rows type %T", rowsAny)}
	}
	defer func() { _ = rows.Close() }()
	summary := CostSummary{TenantID: tenantID, PeriodStart: start, PeriodEnd: end}
	var tokens uint64
	var requests uint64
	if rows.Next() {
		if err := rows.Scan(&summary.TotalCostUSD, &tokens, &requests); err != nil {
			return CostSummary{}, &QueryError{"cost_by_tenant", err}
		}
	}
	summary.TotalTokens = int64(tokens)
	summary.RequestCount = int64(requests)
	return summary, nil
}

// GetCostByModel breaks cost down per model.
func (a *Analytics) GetCostByModel(ctx context.Context, tenantID string, start, end time.Time) ([]ModelCostSummary, error) {
	rowsAny, err := a.client.Query(ctx,
		`SELECT model, sum(cost_usd), sum(total_tokens), count()
		 FROM token_events
		 WHERE tenant_id = ? AND timestamp >= ? AND timestamp < ?
		 GROUP BY model
		 ORDER BY sum(cost_usd) DESC`,
		tenantID, start, end)
	if err != nil {
		return nil, &QueryError{"cost_by_model", err}
	}
	rows, ok := rowsAny.(interface {
		Next() bool
		Scan(...any) error
		Close() error
	})
	if !ok {
		return nil, &QueryError{"cost_by_model", fmt.Errorf("unexpected rows type %T", rowsAny)}
	}
	defer func() { _ = rows.Close() }()
	var out []ModelCostSummary
	for rows.Next() {
		var (
			item   ModelCostSummary
			tokens uint64
			reqs   uint64
		)
		if err := rows.Scan(&item.Model, &item.TotalCostUSD, &tokens, &reqs); err != nil {
			return nil, &QueryError{"cost_by_model", err}
		}
		item.TotalTokens = int64(tokens)
		item.RequestCount = int64(reqs)
		if reqs > 0 {
			item.AvgCostPerReq = item.TotalCostUSD / float64(reqs)
		}
		out = append(out, item)
	}
	return out, nil
}

// GetCostByFeature breaks cost down per feature (task category) with ROI
// defined as cost per accepted output — the expensive-vs-productive axis.
func (a *Analytics) GetCostByFeature(ctx context.Context, tenantID string, start, end time.Time) ([]FeatureCostSummary, error) {
	rowsAny, err := a.client.Query(ctx,
		`SELECT feature, sum(cost_usd), sum(total_tokens), count(),
		        countIf(metadata['output_status'] = 'accepted')
		 FROM token_events
		 WHERE tenant_id = ? AND timestamp >= ? AND timestamp < ?
		 GROUP BY feature
		 ORDER BY sum(cost_usd) DESC`,
		tenantID, start, end)
	if err != nil {
		return nil, &QueryError{"cost_by_feature", err}
	}
	rows, ok := rowsAny.(interface {
		Next() bool
		Scan(...any) error
		Close() error
	})
	if !ok {
		return nil, &QueryError{"cost_by_feature", fmt.Errorf("unexpected rows type %T", rowsAny)}
	}
	defer func() { _ = rows.Close() }()
	var out []FeatureCostSummary
	for rows.Next() {
		var (
			item     FeatureCostSummary
			tokens   uint64
			reqs     uint64
			accepted uint64
		)
		if err := rows.Scan(&item.Feature, &item.TotalCostUSD, &tokens, &reqs, &accepted); err != nil {
			return nil, &QueryError{"cost_by_feature", err}
		}
		item.TotalTokens = int64(tokens)
		item.RequestCount = int64(reqs)
		if accepted > 0 {
			item.ROI = item.TotalCostUSD / float64(accepted)
		}
		out = append(out, item)
	}
	return out, nil
}

// GetZombieAgents finds agents (workers) burning money with an acceptance rate
// below the threshold inside the window — the signature enterprise signal.
// Only offenders are returned.
func (a *Analytics) GetZombieAgents(ctx context.Context, tenantID string, threshold float64, window time.Duration) ([]ZombieAgentRecord, error) {
	start := time.Now().UTC().Add(-window)
	rowsAny, err := a.client.Query(ctx,
		`SELECT user_id, sum(cost_usd), count(), max(timestamp),
		        countIf(metadata['output_status'] = 'accepted')
		 FROM token_events
		 WHERE tenant_id = ? AND timestamp >= ?
		 GROUP BY user_id
		 ORDER BY sum(cost_usd) DESC`,
		tenantID, start)
	if err != nil {
		return nil, &QueryError{"zombie_agents", err}
	}
	rows, ok := rowsAny.(interface {
		Next() bool
		Scan(...any) error
		Close() error
	})
	if !ok {
		return nil, &QueryError{"zombie_agents", fmt.Errorf("unexpected rows type %T", rowsAny)}
	}
	defer func() { _ = rows.Close() }()
	var out []ZombieAgentRecord
	for rows.Next() {
		var (
			record   ZombieAgentRecord
			reqs     uint64
			accepted uint64
		)
		if err := rows.Scan(&record.AgentID, &record.TotalCost, &reqs, &record.LastActivity, &accepted); err != nil {
			return nil, &QueryError{"zombie_agents", err}
		}
		record.TenantID = tenantID
		record.TotalRequests = int64(reqs)
		if reqs > 0 {
			record.AcceptanceRate = float64(accepted) / float64(reqs)
		}
		if record.AcceptanceRate >= threshold {
			continue
		}
		rec, ok := zombieRecommendation(record.AcceptanceRate, record.TotalCost, record.TotalRequests)
		if !ok {
			continue
		}
		record.Recommendation = rec
		out = append(out, record)
	}
	return out, nil
}

// zombieRecommendation classifies a below-threshold agent. Priced-but-useless
// agents with visible spend are quarantine candidates; agents whose spend is
// INVISIBLE (unknown pricing) still count — zero-acceptance with unpriced
// tokens is exactly the zombie the product exists to expose. Returns false
// when the agent should not be reported at all.
func zombieRecommendation(acceptanceRate, totalCost float64, totalRequests int64) (string, bool) {
	switch {
	case acceptanceRate == 0 && totalCost > 0:
		return "quarantine", true
	case totalCost > 0 || totalRequests > 0:
		return "investigate", true
	default:
		return "", false
	}
}

// GetAnomalies reads the mirrored anomaly signals (populated by the ingestion
// pipeline's secondary mirror alongside the primary anomaly store).
func (a *Analytics) GetAnomalies(ctx context.Context, tenantID string, start, end time.Time) ([]AnomalyRecord, error) {
	rowsAny, err := a.client.Query(ctx,
		`SELECT id, anomaly_type, severity, description, timestamp, metric_value, threshold, metadata
		 FROM anomalies
		 WHERE tenant_id = ? AND timestamp >= ? AND timestamp < ?
		 ORDER BY timestamp DESC`,
		tenantID, start, end)
	if err != nil {
		return nil, &QueryError{"anomalies", err}
	}
	rows, ok := rowsAny.(interface {
		Next() bool
		Scan(...any) error
		Close() error
	})
	if !ok {
		return nil, &QueryError{"anomalies", fmt.Errorf("unexpected rows type %T", rowsAny)}
	}
	defer func() { _ = rows.Close() }()
	var out []AnomalyRecord
	for rows.Next() {
		var (
			item AnomalyRecord
			meta map[string]string
		)
		if err := rows.Scan(&item.ID, &item.Type, &item.Severity, &item.Description, &item.Timestamp, &item.MetricValue, &item.Threshold, &meta); err != nil {
			return nil, &QueryError{"anomalies", err}
		}
		item.TenantID = tenantID
		if len(meta) > 0 {
			item.Metadata = make(map[string]any, len(meta))
			for k, v := range meta {
				item.Metadata[k] = v
			}
		}
		out = append(out, item)
	}
	return out, nil
}