package moat

import (
	"strings"
	"testing"
)

func TestGenerateAPIKey(t *testing.T) {
	tenantID := "tenant-xyz"
	name := "test-key"

	apiKey, compoundToken, err := GenerateAPIKey(tenantID, name)
	if err != nil {
		t.Fatalf("unexpected error generating api key: %w", err)
	}

	if apiKey.TenantID != tenantID {
		t.Errorf("expected tenantID %s, got %s", tenantID, apiKey.TenantID)
	}

	if apiKey.Name != name {
		t.Errorf("expected name %s, got %s", name, apiKey.Name)
	}

	if !strings.HasPrefix(apiKey.KeyID, "key_") {
		t.Errorf("expected KeyID to start with 'key_', got %s", apiKey.KeyID)
	}

	if apiKey.KeyHash == "" {
		t.Errorf("expected KeyHash to be populated")
	}

	parts := strings.SplitN(compoundToken, ".", 2)
	if len(parts) != 2 {
		t.Fatalf("expected compound token to have 2 parts separated by '.', got %s", compoundToken)
	}

	if parts[0] != apiKey.KeyID {
		t.Errorf("expected compound token to start with KeyID, got %s", parts[0])
	}

	if !strings.HasPrefix(parts[1], keyPrefix) {
		t.Errorf("expected secret part to have prefix '%s', got %s", keyPrefix, parts[1])
	}
}

func TestVerifyAPIKey(t *testing.T) {
	apiKey, rawToken, err := GenerateAPIKey("test", "test")
	if err != nil {
		t.Fatalf("error generating key: %w", err)
	}

	validParts := strings.Split(rawToken, ".")
	if len(validParts) != 2 {
		t.Fatalf("invalid token format")
	}
	validRawSecret := validParts[1]
	validHash := apiKey.KeyHash

	tests := []struct {
		name      string
		secretKey string
		hash      string
		expected  bool
	}{
		{
			name:      "valid key and hash",
			secretKey: validRawSecret,
			hash:      validHash,
			expected:  true,
		},
		{
			name:      "invalid secret (wrong value)",
			secretKey: validRawSecret + "invalid",
			hash:      validHash,
			expected:  false,
		},
		{
			name:      "invalid secret (missing prefix)",
			secretKey: "invalid_prefix_" + strings.TrimPrefix(validRawSecret, keyPrefix),
			hash:      validHash,
			expected:  false,
		},
		{
			name:      "invalid hash",
			secretKey: validRawSecret,
			hash:      "invalid_hash",
			expected:  false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := VerifyAPIKey(tc.secretKey, tc.hash)
			if result != tc.expected {
				t.Errorf("expected VerifyAPIKey to be %v, got %v", tc.expected, result)
			}
		})
	}
}

func TestExtractBearerToken(t *testing.T) {
	tests := []struct {
		name       string
		authHeader string
		expected   string
	}{
		{
			name:       "valid bearer token",
			authHeader: "Bearer token123",
			expected:   "token123",
		},
		{
			name:       "valid bearer token lowercase",
			authHeader: "bearer token456",
			expected:   "token456",
		},
		{
			name:       "missing header",
			authHeader: "",
			expected:   "",
		},
		{
			name:       "wrong scheme",
			authHeader: "Basic base64data",
			expected:   "",
		},
		{
			name:       "malformed header",
			authHeader: "Bearer",
			expected:   "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := ExtractBearerToken(tc.authHeader)
			if result != tc.expected {
				t.Errorf("expected ExtractBearerToken to be %q, got %q", tc.expected, result)
			}
		})
	}
}
