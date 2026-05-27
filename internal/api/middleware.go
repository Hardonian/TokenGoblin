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

// AuthMiddleware wraps a handler to authenticate API keys.
// If an API key is valid, it injects the tenant ID into the context.
// For the MVP, we also allow the `x-tenant-id` header if no API key is provided,
// but in a strict enterprise mode we would reject requests without a Bearer token.
func AuthMiddleware(repo storage.Repository, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := moat.ExtractBearerToken(r.Header.Get("Authorization"))

		if token != "" {
			parts := strings.SplitN(token, ".", 2)
			if len(parts) != 2 {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}

			keyID := parts[0]
			secret := parts[1]

			apiKey, err := repo.GetAPIKey(r.Context(), keyID)
			if err != nil || apiKey == nil || apiKey.IsRevoked {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}

			if !moat.VerifyAPIKey(secret, apiKey.KeyHash) {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}

			// Update last used (async to not block)
			go func() {
				_ = repo.UpdateAPIKeyLastUsed(context.Background(), keyID)
			}()

			ctx := context.WithValue(r.Context(), tenantIDKey, apiKey.TenantID)
			ctx = context.WithValue(ctx, apiKeyIDKey, keyID)
			next.ServeHTTP(w, r.WithContext(ctx))
			return
		}

		// Fallback to x-tenant-id for backward compatibility in MVP
		tenantID := r.Header.Get("x-tenant-id")
		if tenantID == "" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		ctx := context.WithValue(r.Context(), tenantIDKey, tenantID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// RateLimitMiddleware wraps a handler with token bucket rate limiting.
func RateLimitMiddleware(limiter *moat.RateLimiter, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tenantID := r.Context().Value(tenantIDKey).(string)

		if limiter != nil {
			allowed, err := limiter.AllowIngestion(r.Context(), tenantID)
			if err != nil {
				// If Redis is down, we log it and allow the request (fail open)
				// In a strict environment we might fail closed.
			} else if !allowed {
				w.WriteHeader(http.StatusTooManyRequests)
				return
			}
		}

		next.ServeHTTP(w, r)
	})
}

func getTenantID(r *http.Request) string {
	if val := r.Context().Value(tenantIDKey); val != nil {
		return val.(string)
	}
	return ""
}
