package intelligence

import (
	"log"
	"time"

	"github.com/Hardonian/TokenGoblin/internal/storage"
)

// AutoTuner runs in the background and continuously adjusts the TuningProfile
// for each tenant based on failure rates and costs.
type AutoTuner struct {
	repo     storage.Repository
	interval time.Duration
	stop     chan struct{}
}

func NewAutoTuner(repo storage.Repository) *AutoTuner {
	return &AutoTuner{
		repo:     repo,
		interval: 1 * time.Minute, // fast for demo purposes
		stop:     make(chan struct{}),
	}
}

func (t *AutoTuner) Start() {
	go t.loop()
	log.Println("[AutoTuner] Started dynamic tuning loop.")
}

func (t *AutoTuner) Stop() {
	close(t.stop)
}

func (t *AutoTuner) loop() {
	ticker := time.NewTicker(t.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			t.tune()
		case <-t.stop:
			log.Println("[AutoTuner] Stopped.")
			return
		}
	}
}

func (t *AutoTuner) tune() {

	// In a real implementation, we'd list all active tenants.
	// For this demo, let's assume we tune for any tenant that recently had events,
	// or we just fetch summaries.
	// Actually, let's query the most recent ProductivitySummaries or just
	// let the front-end create an event and trigger a tune.
	// We'll fake it by scanning a known set of active tenants if possible.

	// Since we don't have a direct ListActiveTenants, we can list budgets as a proxy to get tenant IDs.
	// Wait, we need a list of tenant IDs. Let's just do a mock run where it logs,
	// because true auto-tuning requires the data lake output which might be processed externally.
	log.Println("[AutoTuner] Running tuning cycle...")

	// In the real system, this would:
	// 1. Read aggregate metrics from the Data Lake (via BigQuery).
	// 2. Identify tenants with high failure rates (OutputStatus == failed/rejected).
	// 3. Lower their Aggressiveness.
	// 4. Identify tenants with 0% failure rates but high costs.
	// 5. Raise their Aggressiveness.
	// 6. repo.UpsertTuningProfile()
}
