package moat_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Hardonian/TokenGoblin/internal/domain"
	"github.com/Hardonian/TokenGoblin/internal/moat"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWebhookAlerter_EmptyURL(t *testing.T) {
	alerter := moat.NewWebhookAlerter()
	// By default URLResolver returns ""

	err := alerter.Alert(context.Background(), "test-tenant", nil)
	assert.NoError(t, err)
}

func TestWebhookAlerter_Success(t *testing.T) {
	anomalies := []domain.AnomalySignal{
		{
			WorkerID: "worker-1",
			Type:     "spike",
			Severity: "high",
		},
	}
	tenantID := "test-tenant-1"

	var receivedPayload map[string]interface{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

		err := json.NewDecoder(r.Body).Decode(&receivedPayload)
		require.NoError(t, err)

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	alerter := moat.NewWebhookAlerter()
	alerter.URLResolver = func(tID string) string {
		assert.Equal(t, tenantID, tID)
		return server.URL
	}

	err := alerter.Alert(context.Background(), tenantID, anomalies)
	require.NoError(t, err)

	// Verify payload
	require.NotNil(t, receivedPayload)
	assert.Equal(t, tenantID, receivedPayload["tenant_id"])
	assert.NotNil(t, receivedPayload["timestamp"])

	receivedAnomalies, ok := receivedPayload["anomalies"].([]interface{})
	require.True(t, ok)
	require.Len(t, receivedAnomalies, 1)

	firstAnomaly := receivedAnomalies[0].(map[string]interface{})

	anomalyData, err := json.Marshal(firstAnomaly)
	require.NoError(t, err)

	var receivedSignal domain.AnomalySignal
	err = json.Unmarshal(anomalyData, &receivedSignal)
	require.NoError(t, err)

	assert.Equal(t, "worker-1", receivedSignal.WorkerID)
	assert.Equal(t, domain.AnomalyType("spike"), receivedSignal.Type)
	assert.Equal(t, domain.Severity("high"), receivedSignal.Severity)
}

func TestWebhookAlerter_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	alerter := moat.NewWebhookAlerter()
	alerter.URLResolver = func(tID string) string {
		return server.URL
	}

	err := alerter.Alert(context.Background(), "test-tenant", nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "webhook returned status 500")
}

func TestWebhookAlerter_BadURL(t *testing.T) {
	alerter := moat.NewWebhookAlerter()
	alerter.URLResolver = func(tID string) string {
		return "http://invalid-url-that-does-not-exist.local"
	}

	// Make a short timeout so it fails quickly if it tries to connect
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	err := alerter.Alert(ctx, "test-tenant", nil)
	require.Error(t, err)
}

func TestWebhookAlerter_ContextTimeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	alerter := moat.NewWebhookAlerter()
	alerter.URLResolver = func(tID string) string {
		return server.URL
	}

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	err := alerter.Alert(ctx, "test-tenant", nil)
	require.Error(t, err)
	assert.ErrorIs(t, err, context.DeadlineExceeded)
}
