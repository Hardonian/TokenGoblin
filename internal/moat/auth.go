package moat

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/Hardonian/TokenGoblin/internal/domain"
	"golang.org/x/crypto/bcrypt"
)

const keyPrefix = "tg_"

// GenerateAPIKey creates a new API key and its hashed version.
func GenerateAPIKey(tenantID, name string) (domain.APIKey, string, error) {
	rawBytes := make([]byte, 32)
	if _, err := rand.Read(rawBytes); err != nil {
		return domain.APIKey{}, "", fmt.Errorf("generate random bytes: %w", err)
	}

	keyIDBytes := make([]byte, 8)
	if _, err := rand.Read(keyIDBytes); err != nil {
		return domain.APIKey{}, "", fmt.Errorf("generate key id: %w", err)
	}

	secretKey := keyPrefix + base64.RawURLEncoding.EncodeToString(rawBytes)
	keyID := "key_" + hex.EncodeToString(keyIDBytes)
	compoundToken := keyID + "." + secretKey

	hashed, err := bcrypt.GenerateFromPassword([]byte(secretKey), bcrypt.DefaultCost)
	if err != nil {
		return domain.APIKey{}, "", fmt.Errorf("hash key: %w", err)
	}

	apiKey := domain.APIKey{
		KeyID:     keyID,
		TenantID:  tenantID,
		Name:      name,
		KeyHash:   string(hashed),
		CreatedAt: time.Now().UTC(),
		IsRevoked: false,
	}

	return apiKey, compoundToken, nil
}

// VerifyAPIKey verifies a plaintext secret against a hashed key.
func VerifyAPIKey(secretKey, hash string) bool {
	// First check basic prefix quickly
	if !strings.HasPrefix(secretKey, keyPrefix) {
		return false
	}
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(secretKey))
	return err == nil
}

// ExtractBearerToken safely extracts a bearer token from the Authorization header.
func ExtractBearerToken(authHeader string) string {
	if authHeader == "" {
		return ""
	}
	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) == 2 && strings.ToLower(parts[0]) == "bearer" {
		// Use subtle to prevent some timing attacks on the length check if it matters, though normal equal is fine here.
		return parts[1]
	}
	return ""
}
