package service

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/grapestree/fgrapery/grapery/internal/cache"
	"go.uber.org/zap"
)

// AITextAdmissionGate limits cluster-wide concurrent outbound LLM text HTTP calls using a Redis counter.
// Waiting for a slot is not a hard "job timeout": callers may block until ctx is canceled (e.g. upstream HTTP ctx)
// or Redis errors. Job-level wall time should remain decoupled from this gate.
type AITextAdmissionGate struct {
	cache   cache.Cache
	logger  *zap.Logger
	key     string
	max     int
	poll    time.Duration
	enabled bool
}

const (
	aiTextAdmissionAcquireScript = `
local max = tonumber(ARGV[1])
local cur = tonumber(redis.call("GET", KEYS[1]) or "0")
if cur >= max then return 0 end
redis.call("INCR", KEYS[1])
return 1
`
	aiTextAdmissionReleaseScript = `
local cur = tonumber(redis.call("GET", KEYS[1]) or "0")
if cur <= 0 then return 0 end
return redis.call("DECR", KEYS[1])
`
)

// NewAITextAdmissionGate returns nil when max <= 0 or Redis is unavailable (fail-open).
func NewAITextAdmissionGate(c cache.Cache, max int, logger *zap.Logger) *AITextAdmissionGate {
	if cache.IsEffectivelyNil(c) || max <= 0 {
		return nil
	}
	log := logger
	if log == nil {
		log = zap.NewNop()
	}
	return &AITextAdmissionGate{
		cache:   c,
		logger:  log,
		key:     cache.AITextProviderInflightKey(),
		max:     max,
		poll:    100 * time.Millisecond,
		enabled: true,
	}
}

// Acquire waits until this process may start an outbound text LLM HTTP request.
// Returned release MUST be invoked exactly once after the provider round-trip completes (success or failure).
func (g *AITextAdmissionGate) Acquire(ctx context.Context) (release func(), err error) {
	if g == nil || !g.enabled {
		return func() {}, nil
	}

	t0 := time.Now()
	acquireLoops := 0
	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		raw, evalErr := g.cache.Eval(ctx, aiTextAdmissionAcquireScript, []string{g.key}, g.max)
		if evalErr != nil {
			return nil, fmt.Errorf("text admission acquire: %w", evalErr)
		}

		ok, converr := redisBoolInt(raw)
		if converr != nil {
			return nil, fmt.Errorf("text admission acquire: unexpected redis reply %v (%w)", raw, converr)
		}
		if ok {
			if acquireLoops > 0 {
				waited := time.Since(t0)
				g.logger.Info("global text inference slot acquired after wait",
					zap.Duration("queued", waited),
					zap.Int("maxConcurrent", g.max),
					zap.String("redisKey", g.key))
			}

			var once sync.Once
			return func() {
				once.Do(func() {
					bg, cancel := context.WithTimeout(context.Background(), 5*time.Second)
					defer cancel()
					if _, relErr := g.cache.Eval(bg, aiTextAdmissionReleaseScript, []string{g.key}); relErr != nil {
						g.logger.Warn("text admission release failed",
							zap.Error(relErr),
							zap.String("redisKey", g.key))
					}
				})
			}, nil
		}

		acquireLoops++
		if acquireLoops == 1 {
			g.logger.Debug("global text inference saturated; waiting for slot",
				zap.Int("maxConcurrent", g.max),
				zap.String("redisKey", g.key))
		}

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(g.poll):
		}
	}
}

func redisBoolInt(v interface{}) (one bool, err error) {
	switch t := v.(type) {
	case int64:
		return t == 1, nil
	case int:
		return t == 1, nil
	case nil:
		return false, fmt.Errorf("nil")
	default:
		return false, fmt.Errorf("type %T", v)
	}
}
