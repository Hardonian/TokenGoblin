package moat

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

func TestAllowIngestion(t *testing.T) {
	t.Run("nil receiver bypasses rate limiting", func(t *testing.T) {
		var rl *RateLimiter
		allowed, err := rl.AllowIngestion(context.Background(), "tenant1")
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if !allowed {
			t.Errorf("expected allowed=true for nil RateLimiter")
		}
	})

	t.Run("nil client to NewRateLimiter bypasses rate limiting", func(t *testing.T) {
		rl := NewRateLimiter(nil)
		allowed, err := rl.AllowIngestion(context.Background(), "tenant1")
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if !allowed {
			t.Errorf("expected allowed=true for nil limiter")
		}
	})

	t.Run("nil limiter bypasses rate limiting", func(t *testing.T) {
		rl := &RateLimiter{}
		allowed, err := rl.AllowIngestion(context.Background(), "tenant1")
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if !allowed {
			t.Errorf("expected allowed=true for nil limiter")
		}
	})

	t.Run("rate limiting enforces limits", func(t *testing.T) {
		mr, err := miniredis.Run()
		if err != nil {
			t.Fatalf("failed to start miniredis: %v", err)
		}
		defer mr.Close()

		client := redis.NewClient(&redis.Options{
			Addr: mr.Addr(),
		})
		defer func() { _ = client.Close() }()

		rl := NewRateLimiter(client)
		if rl == nil {
			t.Fatalf("expected RateLimiter, got nil")
		}

		ctx := context.Background()
		tenantID := "test-tenant"

		allowedCount := 0
		blockedCount := 0

		for i := 0; i < 100; i++ {
			allowed, err := rl.AllowIngestion(ctx, tenantID)
			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
			if allowed {
				allowedCount++
			} else {
				blockedCount++
			}
		}

		if allowedCount == 0 {
			t.Errorf("expected some requests to be allowed, got 0")
		}
		if blockedCount == 0 {
			t.Errorf("expected some requests to be blocked, got 0")
		}

		// The exact numbers depend on miniredis timing and the redis_rate algorithm,
		// but since burst is 50, at least 50 should be allowed, and since we send 100 instantly,
		// some should be blocked.
		if allowedCount < 50 {
			t.Errorf("expected at least 50 allowed requests, got %d", allowedCount)
		}
	})

	t.Run("redis error returns false and error", func(t *testing.T) {
		mr, err := miniredis.Run()
		if err != nil {
			t.Fatalf("failed to start miniredis: %v", err)
		}

		client := redis.NewClient(&redis.Options{
			Addr:        mr.Addr(),
			MaxRetries:  1,
			DialTimeout: 10 * time.Millisecond,
			ReadTimeout: 10 * time.Millisecond,
		})
		defer func() { _ = client.Close() }()

		rl := NewRateLimiter(client)

		// Close miniredis to simulate redis failure
		mr.Close()

		// Setting a tight context timeout to avoid the default backoff delays in go-redis
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()

		allowed, err := rl.AllowIngestion(ctx, "tenant-error")
		if err == nil {
			t.Errorf("expected error when redis is unavailable, got nil")
		}
		if allowed {
			t.Errorf("expected allowed=false when redis is unavailable")
		}
	})
}
