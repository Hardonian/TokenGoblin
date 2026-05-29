package anomaly

import (
	"fmt"
	"math"
	"time"

	"github.com/Hardonian/TokenGoblin/internal/domain"
)

type Thresholds struct {
	SpendSpikeMultiplier       float64
	SpendSpikeMinimumUSD       float64
	TokenSpikeMultiplier       float64
	TokenSpikeMinimumTokens    float64
	LatencySpikeMultiplier     float64
	LatencySpikeMinimumMs      float64
	RepeatedFailureWindow      int
	RepeatedFailureMinimum     int
	HighCostAcceptedMultiplier float64
	HighCostAcceptedMinimumUSD float64
	VelocitySpikeTokens        int
	VelocitySpikeSeconds       float64
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
		VelocitySpikeTokens:        50000,
		VelocitySpikeSeconds:       60,
	}
}

func Detect(event domain.TokenEvent, prior []domain.TokenEvent, now time.Time, thresholds Thresholds) []domain.AnomalySignal {
	var signals []domain.AnomalySignal

	if event.CostIsDegraded && event.CostDegradedCode == "unknown_model_pricing" {
		signals = append(signals, signal(event, now, domain.AnomalyUnknownModelPricing, domain.SeverityMed,
			"Pricing was unavailable for this provider/model; cost is degraded.", nil, nil))
	}

	var costTotal float64
	var costCount int
	var tokenTotal float64
	var tokenCount int
	var latencyTotal float64
	var latencyCount int
	var acceptedCostTotal float64
	var acceptedCostCount int
	var recentTokens int

	for _, item := range prior {
		if item.CostEstimateUSD != nil {
			costTotal += *item.CostEstimateUSD
			costCount++
		}
		if item.TotalTokens > 0 {
			tokenTotal += float64(item.TotalTokens)
			tokenCount++
		}
		if item.WorkerID == event.WorkerID && event.Timestamp.Sub(item.Timestamp).Seconds() <= thresholds.VelocitySpikeSeconds {
			recentTokens += item.TotalTokens
		}
		if item.LatencyMs > 0 {
			latencyTotal += float64(item.LatencyMs)
			latencyCount++
		}
		if isAccepted(item.OutputStatus) && item.ReviewScore != nil && item.CostEstimateUSD != nil {
			acceptedCostTotal += *item.CostEstimateUSD
			acceptedCostCount++
		}
	}

	if event.CostEstimateUSD != nil {
		if costCount >= 3 {
			threshold := math.Max(thresholds.SpendSpikeMinimumUSD, (costTotal/float64(costCount))*thresholds.SpendSpikeMultiplier)
			if *event.CostEstimateUSD > threshold {
				signals = append(signals, signal(event, now, domain.AnomalySpendSpike, domain.SeverityHigh,
					"Event cost exceeded the deterministic spend spike threshold.", event.CostEstimateUSD, &threshold))
			}
		}
	}

	if tokenCount >= 3 {
		threshold := math.Max(thresholds.TokenSpikeMinimumTokens, (tokenTotal/float64(tokenCount))*thresholds.TokenSpikeMultiplier)
		observed := float64(event.TotalTokens)
		if observed > threshold {
			signals = append(signals, signal(event, now, domain.AnomalyTokenSpike, domain.SeverityHigh,
				"Event tokens exceeded the deterministic token spike threshold.", &observed, &threshold))
		}
	}

	recentTokens += event.TotalTokens
	if recentTokens > thresholds.VelocitySpikeTokens {
		observed := float64(recentTokens)
		threshold := float64(thresholds.VelocitySpikeTokens)
		signals = append(signals, signal(event, now, domain.AnomalyVelocitySpike, domain.SeverityHigh,
			"Worker exceeded token velocity threshold (runaway loop).", &observed, &threshold))
	}

	if latencyCount >= 3 && event.LatencyMs > 0 {
		threshold := math.Max(thresholds.LatencySpikeMinimumMs, (latencyTotal/float64(latencyCount))*thresholds.LatencySpikeMultiplier)
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
			// Early exit if it's mathematically impossible to reach the minimum failures threshold
			// given the remaining items allowed to be checked in the window.
			if failures+(thresholds.RepeatedFailureWindow-checked) < thresholds.RepeatedFailureMinimum {
				break
			}

			if item.WorkerID != event.WorkerID {
				continue
			}
			checked++
			if isFailure(item.OutputStatus) {
				failures++
			}
			if failures >= thresholds.RepeatedFailureMinimum {
				break
			}
			if failures+(thresholds.RepeatedFailureWindow-checked) < thresholds.RepeatedFailureMinimum {
				break
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
		if acceptedCostCount >= 3 {
			threshold := math.Max(thresholds.HighCostAcceptedMinimumUSD, (acceptedCostTotal/float64(acceptedCostCount))*thresholds.HighCostAcceptedMultiplier)
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

func isFailure(status domain.OutputStatus) bool {
	return status == domain.OutputFailed || status == domain.OutputRejected
}

func isAccepted(status domain.OutputStatus) bool {
	return status == domain.OutputAccepted || status == domain.OutputSucceeded
}
