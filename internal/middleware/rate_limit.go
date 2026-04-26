package middleware

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/grapestree/fgrapery/grapery/internal/cache"
	"github.com/grapestree/fgrapery/grapery/internal/utils"
	"go.uber.org/zap"
)

// RateLimitConfig defines rate limit parameters for a category of endpoints.
type RateLimitConfig struct {
	Window       time.Duration
	MaxRequests  int64
	KeyPrefix    string
	ErrorMessage string
}

// Pre-defined rate limit tiers.
var (
	RateLimitAIGeneration = RateLimitConfig{
		Window:       time.Minute,
		MaxRequests:  20,
		KeyPrefix:    cache.PrefixRateLimitAI,
		ErrorMessage: "AI generation rate limit exceeded, please try again later",
	}

	RateLimitAuth = RateLimitConfig{
		Window:       time.Minute,
		MaxRequests:  10,
		KeyPrefix:    cache.PrefixRateLimitAuth,
		ErrorMessage: "too many auth attempts, please try again later",
	}

	RateLimitGeneral = RateLimitConfig{
		Window:       time.Minute,
		MaxRequests:  60,
		KeyPrefix:    cache.PrefixRateLimitAPI,
		ErrorMessage: "rate limit exceeded, please slow down",
	}
)

// NewRateLimiter creates a Gin middleware that enforces fixed-window rate limits
// using Redis atomic increment. For authenticated routes it limits per userID;
// for unauthenticated routes it falls back to client IP.
//
// If cacher is nil (e.g. Redis unavailable at startup), returns a no-op middleware
// so routes still work without panicking — same behavior as "fail open" on Redis errors.
func NewRateLimiter(cacher cache.Cache, config RateLimitConfig, logger *zap.Logger) gin.HandlerFunc {
	if cacher == nil {
		log := logger
		if log == nil {
			log = zap.NewNop()
		}
		log.Warn("rate limiter disabled: redis cache is nil")
		return func(c *gin.Context) {
			c.Next()
		}
	}
	if logger == nil {
		logger = zap.NewNop()
	}
	return func(c *gin.Context) {
		identifier := c.GetString("userID")
		if identifier == "" {
			identifier = utils.GetClientIP(
				c.Request.RemoteAddr,
				c.GetHeader("X-Forwarded-For"),
				c.GetHeader("X-Real-IP"),
			)
		}

		windowBucket := time.Now().Unix() / int64(config.Window.Seconds())
		key := cache.RateLimitKey(config.KeyPrefix, identifier, windowBucket)

		ctx := c.Request.Context()
		count, err := cacher.Incr(ctx, key)
		if err != nil {
			// Redis error — fail open to avoid blocking all traffic
			logger.Error("rate limiter redis error, allowing request",
				zap.Error(err),
				zap.String("key", key),
			)
			c.Next()
			return
		}

		// Set TTL on first increment
		if count == 1 {
			_ = cacher.Expire(ctx, key, config.Window*2)
		}

		remaining := config.MaxRequests - count
		if remaining < 0 {
			remaining = 0
		}
		resetAt := time.Now().Truncate(config.Window).Add(config.Window).Unix()

		c.Header("X-RateLimit-Limit", fmt.Sprintf("%d", config.MaxRequests))
		c.Header("X-RateLimit-Remaining", fmt.Sprintf("%d", remaining))
		c.Header("X-RateLimit-Reset", fmt.Sprintf("%d", resetAt))

		if count > config.MaxRequests {
			c.Header("Retry-After", fmt.Sprintf("%d", int(config.Window.Seconds())))
			c.JSON(http.StatusTooManyRequests, gin.H{
				"code":    -7,
				"message": config.ErrorMessage,
			})
			c.Abort()
			return
		}

		c.Next()
	}
}
