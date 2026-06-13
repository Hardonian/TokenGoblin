package moat

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Hardonian/TokenGoblin/internal/domain"
	"github.com/stretchr/testify/assert"
)

func TestWebhookAlerter_Alert(t *testing.T) {
	anomalies := []domain.AnomalySignal{
		{
			AnomalyID: "test-anomaly",
			TenantID:  "tenant-1",
			Severity:  domain.SeverityHigh,
			Type:      domain.AnomalySpendSpike,
		},
	}

	t.Run("empty webhook URL returns early", func(t *testing.T) {
		alerter := NewWebhookAlerter()
		alerter.WebhookURL = ""

		err := alerter.Alert(context.Background(), "tenant-1", anomalies)
		assert.NoError(t, err)
	})

	t.Run("successful webhook call", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "POST", r.Method)
			assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

			body, err := io.ReadAll(r.Body)
			assert.NoError(t, err)

			var payload map[string]interface{}
			err = json.Unmarshal(body, &payload)
			assert.NoError(t, err)

			assert.Equal(t, "tenant-1", payload["tenant_id"])
			assert.NotNil(t, payload["timestamp"])

			anomaliesPayload, ok := payload["anomalies"].([]interface{})
			assert.True(t, ok)
			assert.Len(t, anomaliesPayload, 1)

			anomalyMap, ok := anomaliesPayload[0].(map[string]interface{})
			assert.True(t, ok)
			assert.Equal(t, "test-anomaly", anomalyMap["anomaly_id"])

			w.WriteHeader(http.StatusOK)
		}))
		defer ts.Close()

		alerter := NewWebhookAlerter()
		alerter.WebhookURL = ts.URL

		err := alerter.Alert(context.Background(), "tenant-1", anomalies)
		assert.NoError(t, err)
	})

	t.Run("server error response", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer ts.Close()

		alerter := NewWebhookAlerter()
		alerter.WebhookURL = ts.URL

		err := alerter.Alert(context.Background(), "tenant-1", anomalies)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "webhook returned status 500")
	})

	t.Run("network/connection error", func(t *testing.T) {
		alerter := NewWebhookAlerter()
		alerter.WebhookURL = "http://localhost:1" // Port 1 is typically unused/closed

		err := alerter.Alert(context.Background(), "tenant-1", anomalies)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "connection refused")
	})

	t.Run("context cancellation error", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Do nothing, just hang until context is cancelled
		}))
		defer ts.Close()

		alerter := NewWebhookAlerter()
		alerter.WebhookURL = ts.URL

		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		err := alerter.Alert(ctx, "tenant-1", anomalies)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "context canceled")
	})

	t.Run("invalid URL format", func(t *testing.T) {
		alerter := NewWebhookAlerter()
		alerter.WebhookURL = "://invalid-url"

		err := alerter.Alert(context.Background(), "tenant-1", anomalies)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "missing protocol scheme")
	})
}
