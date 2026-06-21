package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/go-redis/redis/v8"
)

type memCache struct {
	m map[string]string
}

func (m *memCache) Get(ctx context.Context, key string, dest interface{}) error {
	v, ok := m.m[key]
	if !ok {
		return errors.New("cache miss")
	}
	if p, ok := dest.(*string); ok {
		*p = v
	}
	return nil
}
func (m *memCache) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	if m.m == nil {
		m.m = map[string]string{}
	}
	if s, ok := value.(string); ok {
		m.m[key] = s
	}
	return nil
}
func (m *memCache) Delete(ctx context.Context, keys ...string) error { return nil }
func (m *memCache) Exists(ctx context.Context, key string) (bool, error) { return false, nil }
func (m *memCache) Expire(ctx context.Context, key string, expiration time.Duration) error { return nil }
func (m *memCache) LPush(ctx context.Context, key string, values ...interface{}) error     { return nil }
func (m *memCache) LRange(ctx context.Context, key string, start, stop int64) ([]string, error) {
	return nil, nil
}
func (m *memCache) LLen(ctx context.Context, key string) (int64, error) { return 0, nil }
func (m *memCache) SAdd(ctx context.Context, key string, members ...interface{}) error { return nil }
func (m *memCache) SMembers(ctx context.Context, key string) ([]string, error)           { return nil, nil }
func (m *memCache) SIsMember(ctx context.Context, key string, member interface{}) (bool, error) {
	return false, nil
}
func (m *memCache) SRem(ctx context.Context, key string, members ...interface{}) error { return nil }
func (m *memCache) ZAdd(ctx context.Context, key string, members ...*redis.Z) error     { return nil }
func (m *memCache) ZRange(ctx context.Context, key string, start, stop int64) ([]string, error) {
	return nil, nil
}
func (m *memCache) ZRevRange(ctx context.Context, key string, start, stop int64) ([]string, error) {
	return nil, nil
}
func (m *memCache) ZCard(ctx context.Context, key string) (int64, error) { return 0, nil }
func (m *memCache) ZRemRangeByRank(ctx context.Context, key string, start, stop int64) error {
	return nil
}
func (m *memCache) ZScore(ctx context.Context, key string, member string) (float64, error) {
	return 0, nil
}
func (m *memCache) ZIncrBy(ctx context.Context, key string, increment float64, member string) (float64, error) {
	return 0, nil
}
func (m *memCache) HSet(ctx context.Context, key string, values ...interface{}) error { return nil }
func (m *memCache) HGet(ctx context.Context, key, field string) (string, error)     { return "", nil }
func (m *memCache) HGetAll(ctx context.Context, key string) (map[string]string, error) {
	return nil, nil
}
func (m *memCache) HIncrBy(ctx context.Context, key, field string, incr int64) (int64, error) {
	return 0, nil
}
func (m *memCache) Incr(ctx context.Context, key string) (int64, error)                   { return 0, nil }
func (m *memCache) Decr(ctx context.Context, key string) (int64, error)                   { return 0, nil }
func (m *memCache) IncrBy(ctx context.Context, key string, value int64) (int64, error)    { return 0, nil }
func (m *memCache) Eval(ctx context.Context, script string, keys []string, args ...interface{}) (interface{}, error) {
	return nil, nil
}
func (m *memCache) Close() error { return nil }

func TestBuildScope(t *testing.T) {
	if got := BuildScope("fragment-panel", "chat"); got != "agent:fragment-panel:chat" {
		t.Fatalf("scope=%q", got)
	}
}

func TestAgentAccessPolicy_JTIRevokeAndConsume(t *testing.T) {
	c := &memCache{m: map[string]string{}}
	p := NewAgentAccessPolicyService(nil, c, nil, nil, AgentAccessPolicyConfig{ReplayCacheEnabled: true})
	ctx := context.Background()
	_ = p.StoreJTI(ctx, "jti1", time.Minute)
	if p.JTIStatus(ctx, "jti1") != agentJTIStatusIssued {
		t.Fatal("expected issued")
	}
	if err := p.ConsumeJTI(ctx, "jti1", time.Minute); err != nil {
		t.Fatal(err)
	}
	if p.JTIStatus(ctx, "jti1") != agentJTIStatusConsumed {
		t.Fatal("expected consumed")
	}
	if err := p.ConsumeJTI(ctx, "jti1", time.Minute); err == nil {
		t.Fatal("expected double consume error")
	}
	_ = p.RevokeJTI(ctx, "jti2", time.Minute)
	if p.JTIStatus(ctx, "jti2") != agentJTIStatusRevoked {
		t.Fatal("expected revoked")
	}
}
