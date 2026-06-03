package api

import (
	"context"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/Hardonian/TokenGoblin/internal/config"
	"github.com/Hardonian/TokenGoblin/internal/moat"
	"github.com/Hardonian/TokenGoblin/internal/storage"
)

type contextKey string

const (
	tenantIDKey contextKey = "tenant_id"
	apiKeyIDKey contextKey = "api_key_id"
	roleKey     contextKey = "role"
)

func AuthMiddleware(repo storage.Repository, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := moat.ExtractBearerToken(r.Header.Get("Authorization"))

		if token != "" {
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

				go func() {
					_ = repo.UpdateAPIKeyLastUsed(context.Background(), keyID)
				}()

				ctx := context.WithValue(r.Context(), tenantIDKey, apiKey.TenantID)
				ctx = context.WithValue(ctx, apiKeyIDKey, keyID)
				ctx = context.WithValue(ctx, roleKey, apiKey.Role)
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}
		}

		tenantID := strings.TrimSpace(r.Header.Get("x-tenant-id"))
		if config.IsProduction() && !config.AllowDemoTenantAuth() {
			writeAuthError(w, "api_key_required", "Production routes require API key authentication.")
			return
		}
		if tenantID == "" {
			writeAuthError(w, "tenant_missing", "Missing required x-tenant-id header.")
			return
		}

		ctx := context.WithValue(r.Context(), tenantIDKey, tenantID)
		ctx = context.WithValue(ctx, roleKey, "owner")
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func RateLimitMiddleware(limiter *moat.RateLimiter, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tenantID := r.Context().Value(tenantIDKey).(string)

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

func RequireRole(allowed ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			role := getRole(r)
			for _, item := range allowed {
				if role == item || role == "owner" {
					next.ServeHTTP(w, r)
					return
				}
			}
			writeJSON(w, http.StatusForbidden, Envelope{
				OK:     false,
				Status: "error",
				Error:  issue("forbidden", "This API key role cannot perform the requested action."),
			})
		})
	}
}

func getActor(r *http.Request) string {
	if val := r.Context().Value(apiKeyIDKey); val != nil {
		if keyID, ok := val.(string); ok && keyID != "" {
			return "api_key:" + keyID
		}
	}
	if tenantID := getTenantID(r); tenantID != "" {
		return "tenant:" + tenantID
	}
	return "unknown"
}

func getRole(r *http.Request) string {
	if val := r.Context().Value(roleKey); val != nil {
		if role, ok := val.(string); ok && role != "" {
			return role
		}
	}
	return "viewer"
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

// RecoverMiddleware catches panics and returns a 500 cleanly
func RecoverMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				slog.Error("panic recovered during http request", "error", err, "path", r.URL.Path)
				writeJSON(w, http.StatusInternalServerError, Envelope{
					OK:     false,
					Status: "error",
					Error:  issue("internal_error", "An unexpected internal error occurred."),
				})
			}
		}()
		next.ServeHTTP(w, r)
	})
}

// LoggingMiddleware logs request path, method, status, and latency
type loggingResponseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (lrw *loggingResponseWriter) WriteHeader(code int) {
	lrw.statusCode = code
	lrw.ResponseWriter.WriteHeader(code)
}

func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		lrw := &loggingResponseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		next.ServeHTTP(lrw, r)

		duration := time.Since(start)
		slog.Info("http request",
			"method", r.Method,
			"path", r.URL.Path,
			"status", lrw.statusCode,
			"latency_ms", duration.Milliseconds(),
		)
	})
}

// TimeoutMiddleware sets a hard context timeout
func TimeoutMiddleware(timeout time.Duration, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), timeout)
		defer cancel()
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
