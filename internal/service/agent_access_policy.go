package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/grapestree/fgrapery/grapery/internal/cache"
	"github.com/grapestree/fgrapery/grapery/internal/common"
	"github.com/grapestree/fgrapery/grapery/internal/domain"
	"go.uber.org/zap"
)

const (
	agentJTIStatusIssued   = "issued"
	agentJTIStatusConsumed = "consumed"
	agentJTIStatusRevoked  = "revoked"
)

// AgentAccessPolicyConfig 控制 agent token 签发策略。
type AgentAccessPolicyConfig struct {
	PublicParallelEnabled    bool
	ExecFragmentPanelEnabled   bool
	ReplayCacheEnabled         bool
	DefaultChatEstimateTokens  int
	DefaultGenerateEstTokens   int
	DefaultGenerateEstImages   int
}

// AgentAccessIssuanceInput 签发前策略输入。
type AgentAccessIssuanceInput struct {
	UserID    string
	Agent     string
	Operation string
	Kind      string
	TaskID    string
	MaxTokens int
	MaxImages int
}

// AgentAccessIssuanceResult 签发前策略结果（quota 预留等）。
type AgentAccessIssuanceResult struct {
	QuotaReservationID string
	EstimatedTokens    int
	Scope              string
}

// AgentAccessPolicyService grapery 侧 agent token 控制面：权益/额度预检、jti 生命周期。
type AgentAccessPolicyService struct {
	repo   domain.Repository
	cache  cache.Cache
	aiGen  *AIGenerationService
	logger *zap.Logger
	cfg    AgentAccessPolicyConfig
}

func NewAgentAccessPolicyService(repo domain.Repository, c cache.Cache, aiGen *AIGenerationService, logger *zap.Logger, cfg AgentAccessPolicyConfig) *AgentAccessPolicyService {
	if cfg.DefaultChatEstimateTokens <= 0 {
		cfg.DefaultChatEstimateTokens = 1000
	}
	if cfg.DefaultGenerateEstTokens <= 0 {
		cfg.DefaultGenerateEstTokens = 5000
	}
	if cfg.DefaultGenerateEstImages <= 0 {
		cfg.DefaultGenerateEstImages = 4
	}
	return &AgentAccessPolicyService{repo: repo, cache: c, aiGen: aiGen, logger: logger, cfg: cfg}
}

// BuildScope 生成计划中的 scope 字符串，例如 agent:fragment-panel:chat。
func BuildScope(agent, operation string) string {
	agent = strings.TrimSpace(strings.ToLower(agent))
	op := strings.TrimSpace(strings.ToLower(operation))
	if op == "" {
		op = "chat"
	}
	return fmt.Sprintf("agent:%s:%s", agent, op)
}

// PrepareIssuance 在签发 token 前执行策略检查，并在 generate 模式下尝试预留 quota。
func (p *AgentAccessPolicyService) PrepareIssuance(ctx context.Context, in AgentAccessIssuanceInput) (*AgentAccessIssuanceResult, error) {
	if p == nil {
		return &AgentAccessIssuanceResult{Scope: BuildScope(in.Agent, in.Operation)}, nil
	}
	agent := strings.TrimSpace(strings.ToLower(in.Agent))
	op := strings.TrimSpace(strings.ToLower(in.Operation))
	if op == "" {
		op = "chat"
	}
	if !p.cfg.PublicParallelEnabled && op == "generate" {
		return nil, fmt.Errorf("agent parallel generation is not enabled (set AGENT_PUBLIC_PARALLEL_ENABLED=true)")
	}
	if agent == "fragment-panel" && strings.EqualFold(op, "generate") && !p.cfg.ExecFragmentPanelEnabled {
		return nil, fmt.Errorf("fragment-panel agent execution is not enabled")
	}

	userID := strings.TrimSpace(in.UserID)
	if userID == "" {
		return nil, fmt.Errorf("userId is required")
	}

	est := in.MaxTokens
	if est <= 0 {
		if op == "generate" {
			est = p.cfg.DefaultGenerateEstTokens
			if in.MaxImages > 0 {
				est = in.MaxImages * common.AIImageBillingUnitTokens
			} else {
				est += p.cfg.DefaultGenerateEstImages * common.AIImageBillingUnitTokens
			}
		} else {
			est = p.cfg.DefaultChatEstimateTokens
		}
	}

	out := &AgentAccessIssuanceResult{
		EstimatedTokens: est,
		Scope:           BuildScope(agent, op),
	}

	if p.repo == nil {
		return out, nil
	}
	balance, err := p.repo.GetTokenBalance(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to check token balance: %w", err)
	}
	if balance < est {
		return nil, fmt.Errorf("insufficient token balance: have %d, need %d", balance, est)
	}

	// generate 模式：在 Redis quota 预留可用时创建 reservation，写入 token claims。
	if op == "generate" && p.aiGen != nil && p.aiGen.enableQuotaReservation && p.aiGen.quotaReservation != nil {
		meta := map[string]interface{}{
			"agent":     agent,
			"operation": op,
			"kind":      strings.TrimSpace(in.Kind),
			"taskId":    strings.TrimSpace(in.TaskID),
			"scope":     out.Scope,
		}
		res, err := p.aiGen.quotaReservation.ReserveQuota(ctx, userID, est, "agent_access_generate", meta)
		if err != nil {
			return nil, fmt.Errorf("quota reservation failed: %w", err)
		}
		out.QuotaReservationID = res.ReservationID
	}
	return out, nil
}

// StoreJTI 记录已签发的 jti（issued）。
func (p *AgentAccessPolicyService) StoreJTI(ctx context.Context, jti string, ttl time.Duration) error {
	if p == nil || !p.cfg.ReplayCacheEnabled || cache.IsEffectivelyNil(p.cache) || strings.TrimSpace(jti) == "" {
		return nil
	}
	return p.cache.Set(ctx, cache.AgentAccessJTIKey(jti), agentJTIStatusIssued, ttl)
}

// RevokeJTI 撤销 jti。
func (p *AgentAccessPolicyService) RevokeJTI(ctx context.Context, jti string, ttl time.Duration) error {
	if p == nil || cache.IsEffectivelyNil(p.cache) || strings.TrimSpace(jti) == "" {
		return fmt.Errorf("replay cache is not available")
	}
	return p.cache.Set(ctx, cache.AgentAccessJTIKey(jti), agentJTIStatusRevoked, ttl)
}

// JTIStatus 查询 jti 状态：issued / consumed / revoked / unknown。
func (p *AgentAccessPolicyService) JTIStatus(ctx context.Context, jti string) string {
	if cache.IsEffectivelyNil(p.cache) || strings.TrimSpace(jti) == "" {
		return "unknown"
	}
	var status string
	if err := p.cache.Get(ctx, cache.AgentAccessJTIKey(jti), &status); err != nil {
		return "unknown"
	}
	return status
}

// ConsumeJTI 一次性消费 jti（issued -> consumed）。已 consumed/revoked 返回 error。
func (p *AgentAccessPolicyService) ConsumeJTI(ctx context.Context, jti string, ttl time.Duration) error {
	if !p.cfg.ReplayCacheEnabled || cache.IsEffectivelyNil(p.cache) {
		return nil
	}
	jti = strings.TrimSpace(jti)
	if jti == "" {
		return fmt.Errorf("jti is required")
	}
	key := cache.AgentAccessJTIKey(jti)
	cur := p.JTIStatus(ctx, jti)
	switch cur {
	case agentJTIStatusRevoked:
		return fmt.Errorf("token revoked")
	case agentJTIStatusConsumed:
		return fmt.Errorf("token already consumed")
	case agentJTIStatusIssued:
		return p.cache.Set(ctx, key, agentJTIStatusConsumed, ttl)
	default:
		// replay 开启但无记录：拒绝，防止伪造 jti 绕过。
		return fmt.Errorf("token jti not found")
	}
}

// QuotaSnapshot 返回用户 token 余额快照（agent 适配器用）。
func (p *AgentAccessPolicyService) QuotaSnapshot(ctx context.Context, userID string) (balance int, err error) {
	if p == nil || p.repo == nil {
		return 0, fmt.Errorf("quota service unavailable")
	}
	return p.repo.GetTokenBalance(ctx, userID)
}

// ConfirmQuotaReservation 确认配额预留（agent 生成成功后调用）。
func (p *AgentAccessPolicyService) ConfirmQuotaReservation(ctx context.Context, reservationID string, actualTokens int) error {
	if p == nil || p.aiGen == nil {
		return fmt.Errorf("quota service unavailable")
	}
	reservationID = strings.TrimSpace(reservationID)
	if reservationID == "" {
		return fmt.Errorf("reservationId is required")
	}
	return p.aiGen.ConfirmQuotaReservation(ctx, reservationID, actualTokens)
}

// ReleaseQuotaReservation 释放配额预留（agent 生成失败/取消时调用）。
func (p *AgentAccessPolicyService) ReleaseQuotaReservation(ctx context.Context, reservationID string) error {
	if p == nil || p.aiGen == nil {
		return fmt.Errorf("quota service unavailable")
	}
	reservationID = strings.TrimSpace(reservationID)
	if reservationID == "" {
		return fmt.Errorf("reservationId is required")
	}
	return p.aiGen.ReleaseQuotaReservation(ctx, reservationID)
}
