package ingestion

import (
	"context"
	"sync"
	"time"
)

// IngestionConfig configures the ingestion pipeline
type IngestionConfig struct {
	BatchSize          int           // Events per batch
	BatchTimeout       time.Duration // Max time to wait for batch to fill
	BufferSize         int           // Channel buffer size
	MaxRetries         int           // Max retry attempts
	RetryBackoff       time.Duration // Initial backoff duration
	MaxBackoff         time.Duration // Max backoff duration
	WorkerCount        int           // Number of worker goroutines
	RateLimitRPS       int           // Rate limit (requests per second)
	EnableDeduplication bool         // Enable prompt fingerprint deduplication
}

// DefaultConfig returns sensible defaults
func DefaultConfig() IngestionConfig {
	return IngestionConfig{
		BatchSize:          1000,
		BatchTimeout:       5 * time.Second,
		BufferSize:         10000,
		MaxRetries:         3,
		RetryBackoff:       100 * time.Millisecond,
		MaxBackoff:         30 * time.Second,
		WorkerCount:        4,
		RateLimitRPS:       10000,
		EnableDeduplication: true,
	}
}

// Event represents a token usage event
type Event struct {
	ID            string
	TenantID      string
	UserID        string
	Model         string
	Feature       string
	PromptTokens  int64
	CompletionTokens int64
	TotalTokens   int64
	CostUSD       float64
	Timestamp     time.Time
	PromptFingerprint string
	Metadata      map[string]string
}

// EventBatch is a batch of events ready for insertion
type EventBatch struct {
	Events    []Event
	CreatedAt time.Time
	Attempt   int
}

// IngestionPipeline handles event ingestion with backpressure
type IngestionPipeline struct {
	config  IngestionConfig
	input   chan Event
	batches chan EventBatch
	wg      sync.WaitGroup
	mu      sync.RWMutex
	closed  bool
	stats   PipelineStats
}

// PipelineStats tracks ingestion metrics
type PipelineStats struct {
	EventsReceived    int64
	EventsBatched     int64
	EventsInserted    int64
	EventsFailed      int64
	BatchesSubmitted  int64
	BatchesRetried    int64
	BackpressureCount int64
	DeduplicatedCount int64
	LastError         string
	LastErrorTime     time.Time
}

// BackpressureSignal signals when the pipeline is under pressure
type BackpressureSignal struct {
	Level         string     // "normal", "elevated", "critical"
	QueueDepth    int
	ProcessingLag time.Duration
	Timestamp     time.Time
}

// NewIngestionPipeline creates a new ingestion pipeline
func NewIngestionPipeline(config IngestionConfig) *IngestionPipeline {
	if config.BatchSize <= 0 {
		config.BatchSize = DefaultConfig().BatchSize
	}
	if config.BufferSize <= 0 {
		config.BufferSize = DefaultConfig().BufferSize
	}
	if config.WorkerCount <= 0 {
		config.WorkerCount = DefaultConfig().WorkerCount
	}

	return &IngestionPipeline{
		config:  config,
		input:   make(chan Event, config.BufferSize),
		batches: make(chan EventBatch, config.WorkerCount*2),
	}
}

// Start begins the ingestion pipeline workers
func (p *IngestionPipeline) Start(ctx context.Context, inserter Inserter) {
	// Start batching worker
	p.wg.Add(1)
	go p.batcher(ctx)

	// Start insertion workers
	for i := 0; i < p.config.WorkerCount; i++ {
		p.wg.Add(1)
		go p.insertWorker(ctx, inserter)
	}
}

// Ingest adds an event to the pipeline (non-blocking with backpressure signal)
func (p *IngestionPipeline) Ingest(event Event) (bool, BackpressureSignal) {
	p.mu.RLock()
	if p.closed {
		p.mu.RUnlock()
		return false, BackpressureSignal{Level: "critical", QueueDepth: len(p.input)}
	}
	p.mu.RUnlock()

	select {
	case p.input <- event:
		p.stats.EventsReceived++
		return true, p.getBackpressureSignal()
	default:
		p.stats.BackpressureCount++
		return false, p.getBackpressureSignal()
	}
}

// IngestBlocking adds an event, blocking until accepted
func (p *IngestionPipeline) IngestBlocking(ctx context.Context, event Event) error {
	select {
	case p.input <- event:
		p.stats.EventsReceived++
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// getBackpressureSignal returns current backpressure status
func (p *IngestionPipeline) getBackpressureSignal() BackpressureSignal {
	queueDepth := len(p.input)
	level := "normal"
	if queueDepth > p.config.BufferSize*8/10 {
		level = "critical"
	} else if queueDepth > p.config.BufferSize*5/10 {
		level = "elevated"
	}
	return BackpressureSignal{
		Level:      level,
		QueueDepth: queueDepth,
		Timestamp:  time.Now(),
	}
}

// batcher collects events into batches
func (p *IngestionPipeline) batcher(ctx context.Context) {
	defer p.wg.Done()

	ticker := time.NewTicker(p.config.BatchTimeout)
	defer ticker.Stop()

	var currentBatch []Event
	batchStart := time.Now()

	for {
		select {
		case <-ctx.Done():
			if len(currentBatch) > 0 {
				p.batches <- EventBatch{Events: currentBatch, CreatedAt: batchStart}
			}
			return

		case event, ok := <-p.input:
			if !ok {
				if len(currentBatch) > 0 {
					p.batches <- EventBatch{Events: currentBatch, CreatedAt: batchStart}
				}
				close(p.batches)
				return
			}
			currentBatch = append(currentBatch, event)
			if len(currentBatch) == 1 {
				batchStart = time.Now()
			}

			if len(currentBatch) >= p.config.BatchSize {
				p.batches <- EventBatch{Events: currentBatch, CreatedAt: batchStart}
				p.stats.EventsBatched += int64(len(currentBatch))
				currentBatch = nil
				ticker.Reset(p.config.BatchTimeout)
			}

		case <-ticker.C:
			if len(currentBatch) > 0 {
				p.batches <- EventBatch{Events: currentBatch, CreatedAt: batchStart}
				p.stats.EventsBatched += int64(len(currentBatch))
				currentBatch = nil
			}
		}
	}
}

// insertWorker processes batches with retry logic
func (p *IngestionPipeline) insertWorker(ctx context.Context, inserter Inserter) {
	defer p.wg.Done()

	for batch := range p.batches {
		p.mu.RLock()
		closed := p.closed
		p.mu.RUnlock()
		if closed {
			return
		}

		err := p.insertWithRetry(ctx, inserter, batch)
		if err != nil {
			p.stats.EventsFailed += int64(len(batch.Events))
			p.mu.Lock()
			p.stats.LastError = err.Error()
			p.stats.LastErrorTime = time.Now()
			p.mu.Unlock()
		} else {
			p.stats.EventsInserted += int64(len(batch.Events))
			p.stats.BatchesSubmitted++
		}
	}
}

// insertWithRetry attempts to insert a batch with exponential backoff
func (p *IngestionPipeline) insertWithRetry(ctx context.Context, inserter Inserter, batch EventBatch) error {
	backoff := p.config.RetryBackoff
	for attempt := 0; attempt <= p.config.MaxRetries; attempt++ {
		batch.Attempt = attempt
		err := inserter.Insert(ctx, batch.Events)
		if err == nil {
			if attempt > 0 {
				p.stats.BatchesRetried++
			}
			return nil
		}

		if attempt < p.config.MaxRetries {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(backoff):
				backoff = minDuration(backoff*2, p.config.MaxBackoff)
			}
		}
	}
	return nil
}

// Close stops the pipeline gracefully
func (p *IngestionPipeline) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.closed {
		return nil
	}
	p.closed = true
	close(p.input)
	p.wg.Wait()
	return nil
}

// Stats returns current pipeline statistics
func (p *IngestionPipeline) Stats() PipelineStats {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.stats
}

// Inserter interface for storage backends
type Inserter interface {
	Insert(ctx context.Context, events []Event) error
}

func minDuration(a, b time.Duration) time.Duration {
	if a < b {
		return a
	}
	return b
}