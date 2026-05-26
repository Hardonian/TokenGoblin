package anomaly

import (
	"fmt"
	"math"
	"time"

	"github.com/Hardonian/TokenGoblin/internal/domain"
)

type Thresholds struct {
	SpendSpikeMultiplier            float64
	SpendSpikeMinimumUSD            float64
	TokenSpikeMultiplier            float64
	TokenSpikeMinimumTokens         float64
	LatencySpikeMultiplier          float64
	LatencySpikeMinimumMs           float64
	RepeatedFailureWindow           int
	RepeatedFailureMinimum          int
	HighCostAcceptedMultiplier      float64
	HighCostAcceptedMinimumUSD      float64
}

func DefaultThresholds() Thresholds {
	return Thresholds{
		SpendSpikeMultiplier:       3,
		SpendSpikeMinimumUSD:       1,
		TokenSpikeMultiplier:       3,
		TokenSpikeMinimumTokens:    10_000,
		LatencySpikeMultiplier:     3,
		LatencySpikeMinimumMs:      10_000,
		RepeatedFailureWindow:      10,
		RepeatedFailureMinimum:     3,
		HighCostAcceptedMultiplier: 4,
		HighCostAcceptedMinimumUSD: 2,
	}
}

func Detect(event domain.TokenEvent, prior []domain.TokenEvent, now time.Time, thresholds Thresholds) []domain.AnomalySignal {
	var signals []domain.AnomalySignal

	if event.CostIsDegraded && event.CostDegradedCode == "unknown_model_pricing" {
		signals = append(signals, signal(event, now, domain.AnomalyUnknownModelPricing, domain.SeverityMed,
			"Pricing was unavailable for this provider/model; cost is degraded.", nil, nil))
	}

	if event.CostEstimateUSD != nil {
		costs := numericPrior(prior, func(item domain.TokenEvent) (float64, bool) {
			if item.CostEstimateUSD == nil {
				return 0, false
			}
			return *item.CostEstimateUSD, true
		})
		if len(costs) >= 3 {
			threshold := math.Max(thresholds.SpendSpikeMinimumUSD, mean(costs)*thresholds.SpendSpikeMultiplier)
			if *event.CostEstimateUSD > threshold {
				signals = append(signals, signal(event, now, domain.AnomalySpendSpike, domain.SeverityHigh,
					"Event cost exceeded the deterministic spend spike threshold.", event.CostEstimateUSD, &threshold))
			}
		}
	}

	tokens := numericPrior(prior, func(item domain.TokenEvent) (float64, bool) {
		return float64(item.TotalTokens), item.TotalTokens > 0
	})
	if len(tokens) >= 3 {
		threshold := math.Max(thresholds.TokenSpikeMinimumTokens, mean(tokens)*thresholds.TokenSpikeMultiplier)
		observed := float64(event.TotalTokens)
		if observed > threshold {
			signals = append(signals, signal(event, now, domain.AnomalyTokenSpike, domain.SeverityHigh,
				"Event tokens exceeded the deterministic token spike threshold.", &observed, &threshold))
		}
	}

	latencies := numericPrior(prior, func(item domain.TokenEvent) (float64, bool) {
		return float64(item.LatencyMs), item.LatencyMs > 0
	})
	if len(latencies) >= 3 && event.LatencyMs > 0 {
		threshold := math.Max(thresholds.LatencySpikeMinimumMs, mean(latencies)*thresholds.LatencySpikeMultiplier)
		observed := float64(event.LatencyMs)
		if observed > threshold {
			signals = append(signals, signal(event, now, domain.AnomalyLatencySpike, domain.SeverityMed,
				"Event latency exceeded the deterministic latency spike threshold.", &observed, &threshold))
		}
	}

	if isFailure(event.OutputStatus) {
		failures := 1
		checked := 0
		for _, item := range prior {
			if item.WorkerID != event.WorkerID {
				continue
			}
			checked++
			if isFailure(item.OutputStatus) {
				failures++
			}
			if checked >= thresholds.RepeatedFailureWindow {
				break
			}
		}
		if failures >= thresholds.RepeatedFailureMinimum {
			observed := float64(failures)
			threshold := float64(thresholds.RepeatedFailureMinimum)
			signals = append(signals, signal(event, now, domain.AnomalyRepeatedFailedOutputs, domain.SeverityHigh,
				"Worker produced repeated failed outputs within the recent event window.", &observed, &threshold))
		}
	}

	if isAccepted(event.OutputStatus) && event.ReviewScore != nil && event.CostEstimateUSD != nil {
		acceptedCosts := numericPrior(prior, func(item domain.TokenEvent) (float64, bool) {
			if !isAccepted(item.OutputStatus) || item.ReviewScore == nil || item.CostEstimateUSD == nil {
				return 0, false
			}
			return *item.CostEstimateUSD, true
		})
		if len(acceptedCosts) >= 3 {
			threshold := math.Max(thresholds.HighCostAcceptedMinimumUSD, mean(acceptedCosts)*thresholds.HighCostAcceptedMultiplier)
			if *event.CostEstimateUSD > threshold {
				signals = append(signals, signal(event, now, domain.AnomalyHighCostPerAcceptedOutput, domain.SeverityHigh,
					"Accepted reviewed output cost exceeded the deterministic high-cost threshold.", event.CostEstimateUSD, &threshold))
			}
		}
	}

	return signals
}

func signal(event domain.TokenEvent, now time.Time, signalType domain.AnomalyType, severity domain.Severity, description string, observed *float64, threshold *float64) domain.AnomalySignal {
	detectedAt := event.Timestamp
	if detectedAt.IsZero() {
		detectedAt = now
	}
	return domain.AnomalySignal{
		AnomalyID:      fmt.Sprintf("%s:%s", event.EventID, signalType),
		TenantID:       event.TenantID,
		EventID:        event.EventID,
		WorkerID:       event.WorkerID,
		DetectedAt:     detectedAt.UTC(),
		Severity:       severity,
		Type:           signalType,
		Description:    description,
		ObservedValue:  observed,
		ThresholdValue: threshold,
		Details: map[string]interface{}{
			"provider":      event.Provider,
			"model_id":      event.ModelID,
			"task_category": event.TaskCategory,
			"output_status": string(event.OutputStatus),
		},
	}
}

func numericPrior(items []domain.TokenEvent, value func(domain.TokenEvent) (float64, bool)) []float64 {
	values := make([]float64, 0, len(items))
	for _, item := range items {
		next, ok := value(item)
		if ok {
			values = append(values, next)
		}
	}
	return values
}

func mean(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	var total float64
	for _, value := range values {
		total += value
	}
	return total / float64(len(values))
}

func isFailure(status domain.OutputStatus) bool {
	return status == domain.OutputFailed || status == domain.OutputRejected
}

func isAccepted(status domain.OutputStatus) bool {
	return status == domain.OutputAccepted || status == domain.OutputSucceeded
}
