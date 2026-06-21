package http

import (
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/grapestree/fgrapery/grapery/internal/service"
	"go.uber.org/zap"
)

// issueAgentAccessTokenBody 是 POST /api/v1/agent-access-tokens 的请求体。
type issueAgentAccessTokenBody struct {
	Agent     string `json:"agent"`
	Operation string `json:"operation"` // chat | generate
	Kind      string `json:"kind,omitempty"`
	SessionID string `json:"sessionId,omitempty"`
	TaskID    string `json:"taskId,omitempty"`
	MaxTokens int    `json:"maxTokens,omitempty"`
	MaxImages int    `json:"maxImages,omitempty"`
}

var agentAccessAllowed = map[string]bool{
	"fragment-panel": true,
	"fragment":       true,
	"character":      true,
	"storyboard":     true,
	"story":          true,
	"branch":         true,
}

const recommendedAgentBase = "/api/v1"

// IssueAgentAccessToken POST /api/v1/agent-access-tokens
func (h *Handler) IssueAgentAccessToken(c *gin.Context) {
	userID, ok := RequireUserID(c)
	if !ok {
		return
	}
	if h.agentTokenSigner == nil || !h.agentTokenSigner.IsConfigured() {
		InternalError(c, "agent access token signing is not configured")
		return
	}

	var body issueAgentAccessTokenBody
	if !BindJSON(c, &body) {
		return
	}

	agent := strings.TrimSpace(strings.ToLower(body.Agent))
	if agent == "" || !agentAccessAllowed[agent] {
		InvalidParams(c, "unsupported or missing agent")
		return
	}

	operation := strings.TrimSpace(strings.ToLower(body.Operation))
	switch operation {
	case "", "chat":
		operation = "chat"
	case "generate":
	default:
		InvalidParams(c, "operation must be chat or generate")
		return
	}

	quotaMode := "chat_only"
	if operation == "generate" {
		quotaMode = "budget"
	}

	var policyResult *service.AgentAccessIssuanceResult
	if h.agentPolicy != nil {
		var err error
		policyResult, err = h.agentPolicy.PrepareIssuance(c.Request.Context(), service.AgentAccessIssuanceInput{
			UserID:    userID,
			Agent:     agent,
			Operation: operation,
			Kind:      strings.TrimSpace(body.Kind),
			TaskID:    strings.TrimSpace(body.TaskID),
			MaxTokens: body.MaxTokens,
			MaxImages: body.MaxImages,
		})
		if err != nil {
			Forbidden(c, err.Error())
			return
		}
	}

	scope := ""
	reservationID := ""
	if policyResult != nil {
		scope = policyResult.Scope
		reservationID = policyResult.QuotaReservationID
	}

	result, err := h.agentTokenSigner.Issue(service.AgentAccessTokenRequest{
		UserID:             userID,
		Agent:              agent,
		Operation:          operation,
		Kind:               strings.TrimSpace(body.Kind),
		SessionID:          strings.TrimSpace(body.SessionID),
		QuotaMode:          quotaMode,
		QuotaReservationID: reservationID,
		Scope:              scope,
		MaxTokens:          body.MaxTokens,
		MaxImages:          body.MaxImages,
	})
	if err != nil {
		HandleError(c, err)
		return
	}

	ttl := h.agentTokenSigner.TTL() + 30*time.Second
	if h.agentPolicy != nil {
		if err := h.agentPolicy.StoreJTI(c.Request.Context(), result.JTI, ttl); err != nil && h.logger != nil {
			h.logger.Warn("failed to record agent access token jti", zap.Error(err))
		}
	}

	if h.logger != nil {
		h.logger.Info("issued agent access token",
			zap.String("userID", userID),
			zap.String("agent", agent),
			zap.String("operation", operation),
			zap.String("scope", scope),
			zap.String("requestID", result.RequestID),
			zap.String("jti", result.JTI),
		)
	}

	endpoint := recommendedAgentBase
	if operation == "chat" {
		endpoint = recommendedAgentBase + "/agent/" + agent + "/chat"
	} else {
		endpoint = recommendedAgentBase + "/generation/" + agentPathForKind(agent) + "/stream"
	}

	Success(c, gin.H{
		"agentAccessToken":   result.AgentAccessToken,
		"tokenType":          result.TokenType,
		"expiresAt":          result.ExpiresAt,
		"expiresInSec":       result.ExpiresInSec,
		"requestId":          result.RequestID,
		"jti":                result.JTI,
		"agent":              result.Agent,
		"operation":          result.Operation,
		"scope":              scope,
		"quotaMode":          result.QuotaMode,
		"quotaReservationId": result.QuotaReservationID,
		"agentEndpoint":      endpoint,
		"agentEndpointBase":  recommendedAgentBase,
	})
}

func agentPathForKind(agent string) string {
	switch agent {
	case "fragment-panel":
		return "fragment-panels"
	case "fragment":
		return "fragments"
	case "character":
		return "characters"
	case "storyboard":
		return "storyboards"
	case "story":
		return "stories"
	case "branch":
		return "branches"
	default:
		return agent
	}
}

// CancelAgentAccessToken POST /api/v1/agent-access-tokens/:requestId/cancel
func (h *Handler) CancelAgentAccessToken(c *gin.Context) {
	if _, ok := RequireUserID(c); !ok {
		return
	}
	requestID := strings.TrimSpace(c.Param("requestId"))
	if requestID == "" {
		InvalidParams(c, "requestId is required")
		return
	}
	var body struct {
		JTI string `json:"jti"`
	}
	if !BindJSON(c, &body) {
		return
	}
	jti := strings.TrimSpace(body.JTI)
	if jti == "" {
		InvalidParams(c, "jti is required to revoke a token")
		return
	}
	if h.agentPolicy == nil || h.agentTokenSigner == nil {
		InternalError(c, "token revocation is not configured")
		return
	}
	if err := h.agentPolicy.RevokeJTI(c.Request.Context(), jti, h.agentTokenSigner.TTL()+30*time.Second); err != nil {
		HandleError(c, err)
		return
	}
	Success(c, gin.H{"requestId": requestID, "jti": jti, "revoked": true})
}
