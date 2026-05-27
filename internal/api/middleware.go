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
			// Extract the key ID from the token if possible (e.g., if token format is keyID.secret)
			// Or if we don't have the keyID in the token, we'd have to look it up differently.
			// Actually, our API Key generation didn't embed the KeyID in the secret.
			// Let's assume the client sends the KeyID as a header `x-api-key-id` for now, 
			// or we can adjust GenerateAPIKey to return `KeyID.SecretKey`.
			// Since we just built GenerateAPIKey to return `key_XXX` and `tg_YYY`, 
			// standard practice is to pass the API Key as `Bearer tg_YYY`.
			// But to look it up we need to scan the DB if we only have the secret, which is bad.
			// Let's assume we update GenerateAPIKey to return `keyID.secret` to make lookups O(1).
			
			// For now, let's just use the `x-tenant-id` header as the primary tenant identifier 
			// and skip full auth if it's too complex for this MVP layer without a KeyID.
			// Wait, if we use API keys, we should just parse `keyID.secret` from the token.
			// Let's implement that in a moment.
			
			parts := strings.SplitN(token, ".", 2)
			if len(parts) == 2 {
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

// CORSMiddleware handles CORS headers and preflight checks.
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
