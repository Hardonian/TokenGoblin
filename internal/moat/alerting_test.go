package moat

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Hardonian/TokenGoblin/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWebhookAlerter_Alert(t *testing.T) {
	anomalies := []domain.AnomalySignal{
		{
			AnomalyID:      "123",
			TenantID:       "tenant-1",
			Severity:       domain.SeverityHigh,
			Type:           domain.AnomalySpendSpike,
			Description:    "Spend spike detected",
			DetectedAt:     time.Now(),
		},
	}

	t.Run("successful alert", func(t *testing.T) {
		var receivedPayload map[string]interface{}
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "POST", r.Method)
			assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

			err := json.NewDecoder(r.Body).Decode(&receivedPayload)
			require.NoError(t, err)

			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		alerter := NewWebhookAlerter()
		alerter.WebhookURL = server.URL

		err := alerter.Alert(context.Background(), "tenant-1", anomalies)
		assert.NoError(t, err)

		assert.Equal(t, "tenant-1", receivedPayload["tenant_id"])
		// checking that anomalies were sent, basic structure check
		receivedAnomalies, ok := receivedPayload["anomalies"].([]interface{})
		assert.True(t, ok)
		assert.Len(t, receivedAnomalies, 1)
	})

	t.Run("empty webhook url", func(t *testing.T) {
		alerter := NewWebhookAlerter()
		err := alerter.Alert(context.Background(), "tenant-1", anomalies)
		assert.NoError(t, err)
	})

	t.Run("webhook returns error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer server.Close()

		alerter := NewWebhookAlerter()
		alerter.WebhookURL = server.URL

		err := alerter.Alert(context.Background(), "tenant-1", anomalies)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "webhook returned status 500")
	})

	t.Run("webhook context cancel", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			time.Sleep(100 * time.Millisecond)
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		alerter := NewWebhookAlerter()
		alerter.WebhookURL = server.URL

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
		defer cancel()

		err := alerter.Alert(ctx, "tenant-1", anomalies)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "context deadline exceeded")
	})
}
