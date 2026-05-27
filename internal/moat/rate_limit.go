package moat

import (
	"context"
	"time"

	"github.com/go-redis/redis_rate/v10"
	"github.com/redis/go-redis/v9"
)

type RateLimiter struct {
	limiter *redis_rate.Limiter
}

func NewRateLimiter(redisClient *redis.Client) *RateLimiter {
	if redisClient == nil {
		return nil
	}
	return &RateLimiter{
		limiter: redis_rate.NewLimiter(redisClient),
	}
}

// AllowIngestion checks if the tenant has capacity to ingest.
// Returns a boolean indicating if the request is allowed, and an error if Redis fails.
func (rl *RateLimiter) AllowIngestion(ctx context.Context, tenantID string) (bool, error) {
	if rl == nil || rl.limiter == nil {
		// If Redis is not configured, bypass rate limiting
		return true, nil
	}

	// Example: Limit to 100 requests per second with a burst of 50
	limit := redis_rate.Limit{
		Rate:   100,
		Burst:  50,
		Period: time.Second,
	}

	key := "rate:ingest:" + tenantID
	res, err := rl.limiter.Allow(ctx, key, limit)
	if err != nil {
		return false, err
	}

	return res.Allowed > 0, nil
}
