package clickhouse

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
)

// ClickHouseConfig configures the ClickHouse connection pool
type ClickHouseConfig struct {
	Addresses    []string
	Database     string
	Username     string
	Password     string
	MaxOpenConns int
	MaxIdleConns int
	ConnMaxLife  time.Duration
	Settings     map[string]any
	Compression  bool
}

// DefaultConfig returns a sensible default configuration
func DefaultConfig() ClickHouseConfig {
	return ClickHouseConfig{
		Addresses:    []string{"localhost:9000"},
		Database:     "tokengoblin",
		Username:     "default",
		Password:     "",
		MaxOpenConns: 100,
		MaxIdleConns: 20,
		ConnMaxLife:  time.Hour,
		Compression:  true,
	}
}

// ClickHouseClient wraps the ClickHouse driver with connection pooling
type ClickHouseClient struct {
	conn   driver.Conn
	mu     sync.RWMutex
	closed bool
}

// NewClient creates a new ClickHouse client with connection pooling
func NewClient(cfg ClickHouseConfig) (*ClickHouseClient, error) {
	opts := &clickhouse.Options{
		Addr: cfg.Addresses,
		Auth: clickhouse.Auth{
			Database: cfg.Database,
			Username: cfg.Username,
			Password: cfg.Password,
		},
		MaxOpenConns:    cfg.MaxOpenConns,
		MaxIdleConns:    cfg.MaxIdleConns,
		ConnMaxLifetime: cfg.ConnMaxLife,
		Settings:        clickhouse.Settings{},
	}

	for k, v := range cfg.Settings {
		opts.Settings[k] = v
	}

	if cfg.Compression {
		opts.Compression = &clickhouse.Compression{
			Method: clickhouse.CompressionLZ4,
		}
	}

	conn, err := clickhouse.Open(opts)
	if err != nil {
		return nil, fmt.Errorf("failed to open clickhouse connection: %w", err)
	}

	return &ClickHouseClient{
		conn: conn,
	}, nil
}

// Close closes the connection pool
func (c *ClickHouseClient) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return nil
	}
	c.closed = true
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

// Health checks the connection
func (c *ClickHouseClient) Health(ctx context.Context) error {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.closed {
		return fmt.Errorf("connection closed")
	}
	if c.conn != nil {
		return c.conn.Ping(ctx)
	}
	return fmt.Errorf("connection not initialized")
}

// Execute executes a query without returning rows
func (c *ClickHouseClient) Execute(ctx context.Context, query string, args ...any) error {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.closed {
		return fmt.Errorf("connection closed")
	}
	if c.conn != nil {
		return c.conn.Exec(ctx, query, args...)
	}
	return fmt.Errorf("connection not initialized")
}

// Query executes a query returning rows
func (c *ClickHouseClient) Query(ctx context.Context, query string, args ...any) (any, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.closed {
		return nil, fmt.Errorf("connection closed")
	}
	if c.conn != nil {
		return c.conn.Query(ctx, query, args...)
	}
	return nil, fmt.Errorf("connection not initialized")
}

// Insert inserts a batch of rows using native block format
func (c *ClickHouseClient) Insert(ctx context.Context, table string, columns []string, rows [][]any) error {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.closed {
		return fmt.Errorf("connection closed")
	}
	if c.conn == nil {
		return fmt.Errorf("connection not initialized")
	}

	query := fmt.Sprintf("INSERT INTO %s", table)
	if len(columns) > 0 {
		query = fmt.Sprintf("INSERT INTO %s (%s)", table, strings.Join(columns, ", "))
	}
	batch, err := c.conn.PrepareBatch(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to prepare batch: %w", err)
	}

	for _, row := range rows {
		if err := batch.Append(row...); err != nil {
			return fmt.Errorf("failed to append row to batch: %w", err)
		}
	}

	if err := batch.Send(); err != nil {
		return fmt.Errorf("failed to send batch: %w", err)
	}

	return nil
}

// ClickHouseRepository defines the repository interface for token cost data
type ClickHouseRepository interface {
	// Event ingestion
	InsertTokenEvents(ctx context.Context, events []TokenEvent) error
	InsertUsageAggregates(ctx context.Context, aggregates []UsageAggregate) error

	// Query operations
	GetCostByTenant(ctx context.Context, tenantID string, start, end time.Time) ([]CostSummary, error)
	GetCostByModel(ctx context.Context, tenantID string, start, end time.Time) ([]ModelCostSummary, error)
	GetCostByFeature(ctx context.Context, tenantID string, start, end time.Time) ([]FeatureCostSummary, error)
	GetAnomalies(ctx context.Context, tenantID string, start, end time.Time) ([]AnomalyRecord, error)
	GetZombieAgents(ctx context.Context, tenantID string, threshold float64, window time.Duration) ([]ZombieAgentRecord, error)

	// Schema management
	InitSchema(ctx context.Context) error
	Migrate(ctx context.Context, version int) error
}

// TokenEvent represents a token usage event for ingestion
type TokenEvent struct {
	ID                string
	TenantID          string
	UserID            string
	Model             string
	Feature           string
	PromptTokens      int
	CompletionTokens  int
	TotalTokens       int
	CostUSD           float64
	Timestamp         time.Time
	PromptFingerprint string
	Metadata          map[string]string
}

// UsageAggregate represents pre-aggregated usage metrics
type UsageAggregate struct {
	TenantID         string
	PeriodStart      time.Time
	PeriodEnd        time.Time
	Model            string
	Feature          string
	RequestCount     int64
	TotalTokens      int64
	PromptTokens     int64
	CompletionTokens int64
	CostUSD          float64
	UniqueUsers      int64
}

// CostSummary represents cost aggregation by tenant
type CostSummary struct {
	TenantID     string
	PeriodStart  time.Time
	PeriodEnd    time.Time
	TotalCostUSD float64
	TotalTokens  int64
	RequestCount int64
}

// ModelCostSummary represents cost aggregation by model
type ModelCostSummary struct {
	Model         string
	TotalCostUSD  float64
	TotalTokens   int64
	RequestCount  int64
	AvgCostPerReq float64
}

// FeatureCostSummary represents cost attribution by feature
type FeatureCostSummary struct {
	Feature      string
	TotalCostUSD float64
	TotalTokens  int64
	RequestCount int64
	ROI          float64 // cost per successful outcome
}

// AnomalyRecord represents a cost anomaly detection
type AnomalyRecord struct {
	ID          string
	TenantID    string
	Type        string // spike, drift, zombie
	Severity    string // low, medium, high, critical
	Description string
	Timestamp   time.Time
	MetricValue float64
	Threshold   float64
	Metadata    map[string]any
}

// ZombieAgentRecord represents a detected zombie agent
type ZombieAgentRecord struct {
	AgentID        string
	TenantID       string
	AcceptanceRate float64
	TotalCost      float64
	TotalRequests  int64
	LastActivity   time.Time
	Recommendation string // quarantine, investigate, alert
}
