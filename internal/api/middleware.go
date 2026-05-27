package api

import (
	"context"
	"net/http"
	"strings"

	"github.com/Hardonian/TokenGoblin/internal/moat"
	"github.com/Hardonian/TokenGoblin/internal/storage"
)

type contextKey string

const (
	tenantIDKey contextKey = "tenant_id"
	apiKeyIDKey contextKey = "api_key_id"
)

func AuthMiddleware(repo storage.Repository, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := moat.ExtractBearerToken(r.Header.Get("Authorization"))
		if token != "" {
			parts := strings.SplitN(token, ".", 2)
			if len(parts) != 2 || strings.TrimSpace(parts[0]) == "" || strings.TrimSpace(parts[1]) == "" {
				writeAuthError(w, "api_key_malformed", "Bearer token must use key_id.secret format.")
				return
			}

			keyID := parts[0]
			secret := parts[1]
			apiKey, err := repo.GetAPIKey(r.Context(), keyID)
			if err != nil || apiKey == nil || apiKey.IsRevoked {
				writeAuthError(w, "api_key_invalid", "API key is invalid or unavailable.")
				return
			}
			if !moat.VerifyAPIKey(secret, apiKey.KeyHash) {
				writeAuthError(w, "api_key_invalid", "API key is invalid.")
				return
			}

			go func() {
				_ = repo.UpdateAPIKeyLastUsed(context.Background(), keyID)
			}()

			ctx := context.WithValue(r.Context(), tenantIDKey, apiKey.TenantID)
			ctx = context.WithValue(ctx, apiKeyIDKey, keyID)
			next.ServeHTTP(w, r.WithContext(ctx))
			return
		}

		tenantID := strings.TrimSpace(r.Header.Get("x-tenant-id"))
		if tenantID == "" {
			writeAuthError(w, "tenant_missing", "Missing required x-tenant-id header.")
			return
		}
		ctx := context.WithValue(r.Context(), tenantIDKey, tenantID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func RateLimitMiddleware(limiter *moat.RateLimiter, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tenantID, ok := r.Context().Value(tenantIDKey).(string)
		if !ok || tenantID == "" {
			writeAuthError(w, "tenant_missing", "Missing tenant context for rate limiting.")
			return
		}

		if limiter != nil {
			allowed, err := limiter.AllowIngestion(r.Context(), tenantID)
			if err == nil && !allowed {
				writeJSON(w, http.StatusTooManyRequests, Envelope{
					OK:     false,
					Status: "error",
					Error:  issue("rate_limited", "Ingestion rate limit exceeded."),
				})
				return
			}
		}

		next.ServeHTTP(w, r)
	})
}

func getTenantID(r *http.Request) string {
	if val := r.Context().Value(tenantIDKey); val != nil {
		if tenantID, ok := val.(string); ok {
			return tenantID
		}
	}
	return ""
}

func writeAuthError(w http.ResponseWriter, code string, message string) {
	writeJSON(w, http.StatusUnauthorized, Envelope{
		OK:     false,
		Status: "error",
		Error:  issue(code, message),
	})
}

func CORSMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, x-tenant-id")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}
