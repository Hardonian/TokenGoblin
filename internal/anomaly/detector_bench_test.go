package anomaly

import (
	"testing"
	"time"

	"github.com/Hardonian/TokenGoblin/internal/domain"
)

func BenchmarkDetectRepeatedFailure_Impossible(b *testing.B) {
	thresholds := DefaultThresholds() // Window: 10, Min: 3
	now := time.Now()

	event := domain.TokenEvent{
		EventID:      "evt-target",
		WorkerID:     "worker-1",
		OutputStatus: domain.OutputFailed,
		Timestamp:    now,
	}

	// prior: 1000 events, all successes for worker-1
	prior := make([]domain.TokenEvent, 1000)
	for i := 0; i < 1000; i++ {
		worker := "worker-other"
		if i%10 == 0 {
			worker = "worker-1"
		}
		prior[i] = domain.TokenEvent{
			EventID:      "evt-prior",
			WorkerID:     worker,
			OutputStatus: domain.OutputSucceeded,
			Timestamp:    now.Add(-time.Duration(i) * time.Second),
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Detect(event, prior, now, thresholds)
	}
}

func BenchmarkDetectRepeatedFailure_Typical(b *testing.B) {
	thresholds := DefaultThresholds() // Window: 10, Min: 3
	now := time.Now()

	event := domain.TokenEvent{
		EventID:      "evt-target",
		WorkerID:     "worker-1",
		OutputStatus: domain.OutputFailed,
		Timestamp:    now,
	}

	// typical prior: 1000 events, mostly successes, worker-1 has some events scattered
	prior := make([]domain.TokenEvent, 1000)
	for i := 0; i < 1000; i++ {
		worker := "worker-other"
		if i%10 == 0 {
			worker = "worker-1"
		}
		status := domain.OutputSucceeded
		if i%50 == 0 {
			status = domain.OutputFailed
		}
		prior[i] = domain.TokenEvent{
			EventID:      "evt-prior",
			WorkerID:     worker,
			OutputStatus: status,
			Timestamp:    now.Add(-time.Duration(i) * time.Second),
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Detect(event, prior, now, thresholds)
	}
}
