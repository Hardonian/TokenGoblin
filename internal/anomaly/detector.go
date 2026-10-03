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

type priorAggregates struct {
	costTotal         float64
	costCount         int
	tokenTotal        float64
	tokenCount        int
	latencyTotal      float64
	latencyCount      int
	acceptedCostTotal float64
	acceptedCostCount int
	recentTokens      int
}

func aggregatePriorEvents(event domain.TokenEvent, prior []domain.TokenEvent, thresholds Thresholds) priorAggregates {
	var agg priorAggregates
	for _, item := range prior {
		if item.CostEstimateUSD != nil {
			agg.costTotal += *item.CostEstimateUSD
			agg.costCount++
		}
		if item.TotalTokens > 0 {
			agg.tokenTotal += float64(item.TotalTokens)
			agg.tokenCount++
		}
		if item.WorkerID == event.WorkerID && event.Timestamp.Sub(item.Timestamp).Seconds() <= thresholds.VelocitySpikeSeconds {
			agg.recentTokens += item.TotalTokens
		}
		if item.LatencyMs > 0 {
			agg.latencyTotal += float64(item.LatencyMs)
			agg.latencyCount++
		}
		if isAccepted(item.OutputStatus) && item.ReviewScore != nil && item.CostEstimateUSD != nil {
			agg.acceptedCostTotal += *item.CostEstimateUSD
			agg.acceptedCostCount++
		}
	}
	return agg
}

func Detect(event domain.TokenEvent, prior []domain.TokenEvent, now time.Time, thresholds Thresholds) []domain.AnomalySignal {
	var signals []domain.AnomalySignal

	if event.CostIsDegraded && event.CostDegradedCode == "unknown_model_pricing" {
		signals = append(signals, signal(event, now, domain.AnomalyUnknownModelPricing, domain.SeverityMed,
			"Pricing was unavailable for this provider/model; cost is degraded.", nil, nil))
	}

	agg := aggregatePriorEvents(event, prior, thresholds)

	if sig := detectSpendSpike(event, agg.costTotal, agg.costCount, now, thresholds); sig != nil {
		signals = append(signals, *sig)
	}
	if sig := detectTokenSpike(event, agg.tokenTotal, agg.tokenCount, now, thresholds); sig != nil {
		signals = append(signals, *sig)
	}
	if sig := detectVelocitySpike(event, agg.recentTokens, now, thresholds); sig != nil {
		signals = append(signals, *sig)
	}
	if sig := detectLatencySpike(event, agg.latencyTotal, agg.latencyCount, now, thresholds); sig != nil {
		signals = append(signals, *sig)
	}
	if sig := detectRepeatedFailures(event, prior, now, thresholds); sig != nil {
		signals = append(signals, *sig)
	}
	if sig := detectHighCostAccepted(event, agg.acceptedCostTotal, agg.acceptedCostCount, now, thresholds); sig != nil {
		signals = append(signals, *sig)
	}

	return signals
}

func detectSpendSpike(event domain.TokenEvent, costTotal float64, costCount int, now time.Time, thresholds Thresholds) *domain.AnomalySignal {
	if event.CostEstimateUSD == nil || costCount < 3 {
		return nil
	}
	threshold := math.Max(thresholds.SpendSpikeMinimumUSD, (costTotal/float64(costCount))*thresholds.SpendSpikeMultiplier)
	if *event.CostEstimateUSD > threshold {
		sig := signal(event, now, domain.AnomalySpendSpike, domain.SeverityHigh,
			"Event cost exceeded the deterministic spend spike threshold.", event.CostEstimateUSD, &threshold)
		return &sig
	}
	return nil
}

func detectTokenSpike(event domain.TokenEvent, tokenTotal float64, tokenCount int, now time.Time, thresholds Thresholds) *domain.AnomalySignal {
	if tokenCount < 3 {
		return nil
	}
	threshold := math.Max(thresholds.TokenSpikeMinimumTokens, (tokenTotal/float64(tokenCount))*thresholds.TokenSpikeMultiplier)
	observed := float64(event.TotalTokens)
	if observed > threshold {
		sig := signal(event, now, domain.AnomalyTokenSpike, domain.SeverityHigh,
			"Event tokens exceeded the deterministic token spike threshold.", &observed, &threshold)
		return &sig
	}
	return nil
}

func detectVelocitySpike(event domain.TokenEvent, recentTokens int, now time.Time, thresholds Thresholds) *domain.AnomalySignal {
	totalRecent := recentTokens + event.TotalTokens
	if totalRecent > thresholds.VelocitySpikeTokens {
		observed := float64(totalRecent)
		threshold := float64(thresholds.VelocitySpikeTokens)
		sig := signal(event, now, domain.AnomalyVelocitySpike, domain.SeverityHigh,
			"Worker exceeded token velocity threshold (runaway loop).", &observed, &threshold)
		return &sig
	}
	return nil
}

func detectLatencySpike(event domain.TokenEvent, latencyTotal float64, latencyCount int, now time.Time, thresholds Thresholds) *domain.AnomalySignal {
	if latencyCount < 3 || event.LatencyMs <= 0 {
		return nil
	}
	threshold := math.Max(thresholds.LatencySpikeMinimumMs, (latencyTotal/float64(latencyCount))*thresholds.LatencySpikeMultiplier)
	observed := float64(event.LatencyMs)
	if observed > threshold {
		sig := signal(event, now, domain.AnomalyLatencySpike, domain.SeverityMed,
			"Event latency exceeded the deterministic latency spike threshold.", &observed, &threshold)
		return &sig
	}
	return nil
}

func detectRepeatedFailures(event domain.TokenEvent, prior []domain.TokenEvent, now time.Time, thresholds Thresholds) *domain.AnomalySignal {
	if !isFailure(event.OutputStatus) {
		return nil
	}
	failures := 1
	checked := 0
	for _, item := range prior {
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
		sig := signal(event, now, domain.AnomalyRepeatedFailedOutputs, domain.SeverityHigh,
			"Worker produced repeated failed outputs within the recent event window.", &observed, &threshold)
		return &sig
	}
	return nil
}

func detectHighCostAccepted(event domain.TokenEvent, acceptedCostTotal float64, acceptedCostCount int, now time.Time, thresholds Thresholds) *domain.AnomalySignal {
	if !isAccepted(event.OutputStatus) || event.ReviewScore == nil || event.CostEstimateUSD == nil || acceptedCostCount < 3 {
		return nil
	}
	threshold := math.Max(thresholds.HighCostAcceptedMinimumUSD, (acceptedCostTotal/float64(acceptedCostCount))*thresholds.HighCostAcceptedMultiplier)
	if *event.CostEstimateUSD > threshold {
		sig := signal(event, now, domain.AnomalyHighCostPerAcceptedOutput, domain.SeverityHigh,
			"Accepted reviewed output cost exceeded the deterministic high-cost threshold.", event.CostEstimateUSD, &threshold)
		return &sig
	}
	return nil
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
