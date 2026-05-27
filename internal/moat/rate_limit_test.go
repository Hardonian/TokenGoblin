package moat

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

func TestRateLimiter_Bypass(t *testing.T) {
	var rl *RateLimiter
	allowed, err := rl.AllowIngestion(context.Background(), "test-tenant")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !allowed {
		t.Errorf("expected allowed to be true for nil limiter")
	}

	rl2 := NewRateLimiter(nil)
	allowed, err = rl2.AllowIngestion(context.Background(), "test-tenant")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !allowed {
		t.Errorf("expected allowed to be true for nil limiter")
	}
}

func TestRateLimiter_AllowIngestion(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("failed to start miniredis: %v", err)
	}
	defer mr.Close()

	client := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})

	rl := NewRateLimiter(client)
	ctx := context.Background()
	tenantID := "tenant-1"

	// Initial request should be allowed
	allowed, err := rl.AllowIngestion(ctx, tenantID)
	if err != nil {
		t.Fatalf("unexpected error on first request: %v", err)
	}
	if !allowed {
		t.Errorf("expected first request to be allowed")
	}

	// We have a limit of 100/sec, burst 50.
	// We'll hit it with a burst of requests to ensure it eventually limits.
	var denied bool
	for i := 0; i < 200; i++ {
		allowed, err := rl.AllowIngestion(ctx, tenantID)
		if err != nil {
			t.Fatalf("unexpected error on request %d: %v", i+1, err)
		}
		if !allowed {
			denied = true
			break
		}
	}

	if !denied {
		t.Errorf("expected rate limiter to eventually deny requests")
	}

	// Fast-forward time to let the rate limiter recover
	mr.FastForward(2 * time.Second)

	// Should be allowed again
	allowed, err = rl.AllowIngestion(ctx, tenantID)
	if err != nil {
		t.Fatalf("unexpected error after recovery: %v", err)
	}
	if !allowed {
		t.Errorf("expected request to be allowed after recovery")
	}
}
