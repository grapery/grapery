package auth

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	connect "connectrpc.com/connect"
	"github.com/grapery/grapery/config"
	utils "github.com/grapery/grapery/utils"
	rcache "github.com/grapery/grapery/utils/cache"
	"github.com/grapery/grapery/utils/jwt"
)

// RedisRateLimitInterceptor 使用 Redis 实现的全局限流拦截器
// 策略：固定窗口（默认 60 秒），按 IP + 接口名 计数，超过阈值则拒绝
// 最小修改：常量在此定义，后续可改为从配置加载
const (
	defaultWindowSeconds  = 60
	defaultLimitPerWindow = 120 // 每窗口最大请求数
)

type RedisRateLimitInterceptor struct{}

func (RedisRateLimitInterceptor) WrapUnary(next connect.UnaryFunc) connect.UnaryFunc {
	return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
		// 读取配置，允许按路由/按用户覆盖
		limit, window := resolveRateLimit(req)
		if !allowByRedis(ctx, req, limit, window) {
			return nil, connect.NewError(connect.CodeResourceExhausted, fmt.Errorf("rate limit exceeded"))
		}
		return next(ctx, req)
	}
}

func (RedisRateLimitInterceptor) WrapStreamingClient(next connect.StreamingClientFunc) connect.StreamingClientFunc {
	return next
}
func (RedisRateLimitInterceptor) WrapStreamingHandler(next connect.StreamingHandlerFunc) connect.StreamingHandlerFunc {
	return next
}

// allowByRedis 固定窗口计数法
func allowByRedis(ctx context.Context, req connect.AnyRequest, limit, windowSeconds int) bool {
	client := rcache.GetCacheClient()
	if client == nil {
		// Redis 不可用时，放行以避免影响主流程
		return true
	}

	// 优先使用用户ID作为限流维度（如果配置 PreferUser=true 且能从 JWT 提取）
	cfg := config.GlobalConfig
	useUser := cfg != nil && cfg.RateLimit != nil && cfg.RateLimit.PreferUser

	var identity string
	if useUser {
		if uidStr, ok := extractUserIDFromJWT(req); ok && uidStr != "" {
			identity = "u:" + uidStr
		}
	}
	if identity == "" {
		// 回退到 IP 维度
		ip := clientKeyFromHeaders(req)
		if ip == "" {
			ip = "unknown"
		}
		identity = "i:" + ip
	}
	procedure := req.Spec().Procedure
	window := time.Now().Unix() / int64(windowSeconds)
	key := strings.Join([]string{"rl", procedure, identity, strconv.FormatInt(window, 10)}, ":")

	// INCR，并在第一次设置过期时间
	cnt, err := client.Incr(key).Result()
	if err != nil {
		return true
	}
	if cnt == 1 {
		// 仅第一次设置 TTL
		_ = client.Expire(key, time.Duration(windowSeconds)*time.Second).Err()
	}
	return cnt <= int64(limit)
}

func clientKeyFromHeaders(req connect.AnyRequest) string {
	hdr := req.Header()
	if xff := hdr.Get("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		if len(parts) > 0 {
			return strings.TrimSpace(parts[0])
		}
		return xff
	}
	if xrip := hdr.Get("X-Real-IP"); xrip != "" {
		return xrip
	}
	if ua := hdr.Get("User-Agent"); ua != "" {
		return ua
	}
	return ""
}

// extractUserIDFromJWT 从请求头中的 JWT 提取用户ID
// 与 Connect 认证逻辑保持一致，读取 utils.GrpcGateWayCookie 作为 token
func extractUserIDFromJWT(req connect.AnyRequest) (string, bool) {
	hdr := req.Header()
	token := hdr.Get(utils.GrpcGateWayCookie)
	if token == "" {
		return "", false
	}
	j := jwt.NewJwtWrapper(utils.SecretKey, utils.ExpirationHours)
	info, err := j.ValidateToken(token)
	if err != nil {
		return "", false
	}
	return strconv.FormatInt(info.UID, 10), true
}

// resolveRateLimit 解析限流配置：默认 → 按路由覆盖 → 按用户覆盖（可选）
func resolveRateLimit(req connect.AnyRequest) (limit int, window int) {
	// 默认值
	limit = defaultLimitPerWindow
	window = defaultWindowSeconds

	cfg := config.GlobalConfig
	if cfg == nil || cfg.RateLimit == nil {
		return
	}
	if cfg.RateLimit.WindowSeconds > 0 {
		window = cfg.RateLimit.WindowSeconds
	}
	if cfg.RateLimit.DefaultLimit > 0 {
		limit = cfg.RateLimit.DefaultLimit
	}

	// 按接口覆盖
	if cfg.RateLimit.PerRoute != nil {
		if v, ok := cfg.RateLimit.PerRoute[req.Spec().Procedure]; ok && v > 0 {
			limit = v
		}
	}

	// 可选：按用户覆盖（需要从 Header 获取用户 ID）
	if cfg.RateLimit.PreferUser && cfg.RateLimit.PerUser != nil {
		if uid := req.Header().Get("X-User-ID"); uid != "" {
			if v, ok := cfg.RateLimit.PerUser[uid]; ok && v > 0 {
				limit = v
			}
		}
	}
	return
}
