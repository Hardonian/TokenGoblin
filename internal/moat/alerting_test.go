package moat

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Hardonian/TokenGoblin/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWebhookAlerter_Alert(t *testing.T) {
	t.Run("empty URL returns immediately", func(t *testing.T) {
		alerter := NewWebhookAlerter()
		err := alerter.Alert(context.Background(), "tenant-1", nil)
		assert.NoError(t, err)
	})

	t.Run("successful alert", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "POST", r.Method)
			assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

			body, err := io.ReadAll(r.Body)
			require.NoError(t, err)

			var payload map[string]interface{}
			err = json.Unmarshal(body, &payload)
			require.NoError(t, err)

			assert.Equal(t, "tenant-1", payload["tenant_id"])
			assert.Contains(t, payload, "timestamp")
			assert.Contains(t, payload, "anomalies")

			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		alerter := NewWebhookAlerter()
		alerter.URL = server.URL

		anomalies := []domain.AnomalySignal{
			{
				Type:     domain.AnomalySpendSpike,
				Severity: domain.SeverityHigh,
				Description:  "unexpected spike",
			},
		}

		err := alerter.Alert(context.Background(), "tenant-1", anomalies)
		assert.NoError(t, err)
	})

	t.Run("failed alert returns error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusBadRequest)
		}))
		defer server.Close()

		alerter := NewWebhookAlerter()
		alerter.URL = server.URL

		err := alerter.Alert(context.Background(), "tenant-1", nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "webhook returned status 400")
	})

	t.Run("client error (e.g. invalid URL)", func(t *testing.T) {
		alerter := NewWebhookAlerter()
		alerter.URL = "http://invalid-url-that-does-not-exist"

		// Reduce timeout for the test to avoid hanging
		alerter.client.Timeout = 10 * time.Millisecond

		err := alerter.Alert(context.Background(), "tenant-1", nil)
		assert.Error(t, err)
	})

	t.Run("invalid URL scheme error", func(t *testing.T) {
		alerter := NewWebhookAlerter()
		alerter.URL = "://invalid"

		err := alerter.Alert(context.Background(), "tenant-1", nil)
		assert.Error(t, err)
	})
}
