package intelligence

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/Hardonian/TokenGoblin/internal/storage"
)

// DataLakeExporter simulates streaming TokenEvents to a data lake (like BigQuery or S3)
// by writing them to a JSONL file in batches.
type DataLakeExporter struct {
	repo       storage.Repository
	sinkDir    string
	batchLimit int
	interval   time.Duration

	mu   sync.Mutex
	stop chan struct{}
}

func NewDataLakeExporter(repo storage.Repository, sinkDir string) *DataLakeExporter {
	if sinkDir == "" {
		sinkDir = "./data/datalake"
	}
	os.MkdirAll(sinkDir, 0755)

	return &DataLakeExporter{
		repo:       repo,
		sinkDir:    sinkDir,
		batchLimit: 100,
		interval:   10 * time.Second,
		stop:       make(chan struct{}),
	}
}

func (e *DataLakeExporter) Start() {
	go e.loop()
	log.Printf("[DataLake] Exporter started. Sink dir: %s", e.sinkDir)
}

func (e *DataLakeExporter) Stop() {
	close(e.stop)
}

func (e *DataLakeExporter) loop() {
	ticker := time.NewTicker(e.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			e.exportBatch()
		case <-e.stop:
			// Flush one last time
			e.exportBatch()
			log.Println("[DataLake] Exporter stopped.")
			return
		}
	}
}

func (e *DataLakeExporter) exportBatch() {
	e.mu.Lock()
	defer e.mu.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	events, err := e.repo.GetUnexportedEvents(ctx, e.batchLimit)
	if err != nil {
		log.Printf("[DataLake] Error fetching unexported events: %v", err)
		return
	}

	if len(events) == 0 {
		return
	}

	dateStr := time.Now().Format("2006-01-02")
	filename := filepath.Join(e.sinkDir, "token_events_"+dateStr+".jsonl")
	
	f, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Printf("[DataLake] Failed to open sink file: %v", err)
		return
	}
	defer f.Close()

	encoder := json.NewEncoder(f)
	var exportedIDs []string

	for _, evt := range events {
		if err := encoder.Encode(evt); err != nil {
			log.Printf("[DataLake] Failed to encode event %s: %v", evt.EventID, err)
			continue
		}
		exportedIDs = append(exportedIDs, evt.EventID)
	}

	if len(exportedIDs) > 0 {
		if err := e.repo.MarkEventsExported(ctx, exportedIDs); err != nil {
			log.Printf("[DataLake] Failed to mark events as exported: %v", err)
			return
		}
		log.Printf("[DataLake] Exported %d events to %s", len(exportedIDs), filename)
	}
}
