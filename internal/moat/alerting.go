package moat

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/Hardonian/TokenGoblin/internal/domain"
)

type Alerter interface {
	Alert(ctx context.Context, tenantID string, anomalies []domain.AnomalySignal) error
}

type WebhookAlerter struct {
	client      *http.Client
	URLResolver func(tenantID string) string
}

func NewWebhookAlerter() *WebhookAlerter {
	return &WebhookAlerter{
		client:      &http.Client{Timeout: 5 * time.Second},
		URLResolver: func(tenantID string) string { return "" },
	}
}

func (w *WebhookAlerter) Alert(ctx context.Context, tenantID string, anomalies []domain.AnomalySignal) error {
	// In a real application, we would fetch the tenant's webhook URLs from the database.
	// For this stub, we just simulate sending an alert.
	var webhookURL string
	if w.URLResolver != nil {
		webhookURL = w.URLResolver(tenantID)
	}
	if webhookURL == "" {
		return nil
	}

	payload := map[string]interface{}{
		"tenant_id": tenantID,
		"timestamp": time.Now().UTC(),
		"anomalies": anomalies,
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", webhookURL, bytes.NewBuffer(data))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := w.client.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("webhook returned status %d", resp.StatusCode)
	}

	return nil
}
