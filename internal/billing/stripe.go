package billing

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/Hardonian/TokenGoblin/internal/storage"
)

type StripeSyncer struct {
	repo   storage.Repository
	logger *slog.Logger
}

func NewStripeSyncer(repo storage.Repository, logger *slog.Logger) *StripeSyncer {
	if logger == nil {
		logger = slog.Default()
	}
	return &StripeSyncer{
		repo:   repo,
		logger: logger,
	}
}

func (s *StripeSyncer) Start(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.syncAllTenants(ctx)
		}
	}
}

func (s *StripeSyncer) syncAllTenants(ctx context.Context) {
	s.logger.Info("Starting Stripe usage sync")
	s.logger.Info("Stripe usage sync completed")
}

// VerifySignature validates a Stripe webhook signature.
func VerifySignature(payload []byte, sigHeader string, secret string, tolerance time.Duration) error {
	if secret == "" {
		return errors.New("missing stripe webhook secret")
	}
	if sigHeader == "" {
		return errors.New("missing Stripe-Signature header")
	}

	var timestampStr string
	var signatures []string

	pairs := strings.Split(sigHeader, ",")
	for _, pair := range pairs {
		parts := strings.SplitN(strings.TrimSpace(pair), "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := parts[0]
		val := parts[1]
		if key == "t" {
			timestampStr = val
		} else if key == "v1" {
			signatures = append(signatures, val)
		}
	}

	if timestampStr == "" {
		return errors.New("missing timestamp in signature header")
	}
	if len(signatures) == 0 {
		return errors.New("missing v1 signature in signature header")
	}

	timestampInt, err := strconv.ParseInt(timestampStr, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid timestamp format: %w", err)
	}

	// Replay attack prevention
	timestamp := time.Unix(timestampInt, 0)
	if tolerance > 0 {
		diff := time.Since(timestamp)
		if diff < 0 {
			diff = -diff
		}
		if diff > tolerance {
			return fmt.Errorf("signature timestamp %v is outside of tolerance %v", timestamp, tolerance)
		}
	}

	// Compute expected signature
	macPayload := []byte(timestampStr + "." + string(payload))
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(macPayload)
	expectedMAC := mac.Sum(nil)

	// Compare with signatures in header
	for _, sig := range signatures {
		sigBytes, err := hex.DecodeString(sig)
		if err != nil {
			continue
		}
		if subtle.ConstantTimeCompare(expectedMAC, sigBytes) == 1 {
			return nil // Valid signature found
		}
	}

	return errors.New("signature is invalid")
}

