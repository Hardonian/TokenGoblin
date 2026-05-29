package billing

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestVerifySignature(t *testing.T) {
	secret := "whsec_test_secret"
	payload := []byte(`{"id": "evt_123", "type": "customer.subscription.created"}`)
	now := time.Now()
	timestampStr := fmt.Sprintf("%d", now.Unix())

	// Compute valid signature
	macPayload := []byte(timestampStr + "." + string(payload))
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(macPayload)
	expectedMAC := hex.EncodeToString(mac.Sum(nil))

	sigHeader := fmt.Sprintf("t=%s,v1=%s", timestampStr, expectedMAC)

	t.Run("Valid Signature", func(t *testing.T) {
		err := VerifySignature(payload, sigHeader, secret, 5*time.Minute)
		assert.NoError(t, err)
	})

	t.Run("Invalid Secret", func(t *testing.T) {
		err := VerifySignature(payload, sigHeader, "whsec_wrong_secret", 5*time.Minute)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "signature is invalid")
	})

	t.Run("Expired Timestamp", func(t *testing.T) {
		oldTimestampStr := fmt.Sprintf("%d", now.Add(-10*time.Minute).Unix())
		oldMacPayload := []byte(oldTimestampStr + "." + string(payload))
		oldMac := hmac.New(sha256.New, []byte(secret))
		oldMac.Write(oldMacPayload)
		oldExpectedMAC := hex.EncodeToString(oldMac.Sum(nil))
		oldSigHeader := fmt.Sprintf("t=%s,v1=%s", oldTimestampStr, oldExpectedMAC)

		err := VerifySignature(payload, oldSigHeader, secret, 5*time.Minute)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "outside of tolerance")
	})

	t.Run("Malformed Header", func(t *testing.T) {
		err := VerifySignature(payload, "t=invalid_time,v1=abc", secret, 5*time.Minute)
		assert.Error(t, err)
	})

	t.Run("Missing v1 Signature", func(t *testing.T) {
		err := VerifySignature(payload, fmt.Sprintf("t=%s", timestampStr), secret, 5*time.Minute)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "missing v1 signature")
	})

	t.Run("Missing Timestamp", func(t *testing.T) {
		err := VerifySignature(payload, fmt.Sprintf("v1=%s", expectedMAC), secret, 5*time.Minute)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "missing timestamp")
	})
}
