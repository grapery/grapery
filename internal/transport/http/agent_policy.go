package http

import (
	"encoding/base64"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/grapestree/fgrapery/grapery/internal/domain"
	"github.com/grapestree/fgrapery/grapery/internal/service"
	"gorm.io/gorm"
)

// AgentPolicyHandler 暴露给 grapery-agent 的 policy/query 原语（服务间 API Key 认证）。
type AgentPolicyHandler struct {
	policy   *service.AgentAccessPolicyService
	signer   *service.AgentAccessTokenSigner
	audit    *service.GenerationAuditService
	panelGen *service.FragmentPanelGenerationService
	runtime  *service.GenerationRuntimeService
	apiKey   string
}

func NewAgentPolicyHandler(
	policy *service.AgentAccessPolicyService,
	signer *service.AgentAccessTokenSigner,
	audit *service.GenerationAuditService,
	panelGen *service.FragmentPanelGenerationService,
	runtime *service.GenerationRuntimeService,
	apiKey string,
) *AgentPolicyHandler {
	return &AgentPolicyHandler{
		policy:   policy,
		signer:   signer,
		audit:    audit,
		panelGen: panelGen,
		runtime:  runtime,
		apiKey:   strings.TrimSpace(apiKey),
	}
}

func (h *AgentPolicyHandler) RegisterRoutes(r *gin.Engine) {
	g := r.Group("/api/v1/agent-policy")
	g.Use(h.internalAPIKeyMiddleware())
	{
		g.GET("/tokens/:jti/status", h.tokenStatus)
		g.POST("/tokens/:jti/consume", h.consumeToken)
		g.GET("/users/:userId/quota", h.quotaSnapshot)

		g.POST("/generation-audits", h.recordGenerationAudits)
		g.GET("/generation-audits", h.listGenerationAudits)
		g.PUT("/generation-executions/:id", h.saveGenerationExecution)
		g.GET("/generation-executions/:id", h.getGenerationExecution)
		g.GET("/generation-executions", h.listGenerationExecutions)
		g.GET("/generation-executions/:id/events", h.listGenerationExecutionEvents)
		g.POST("/generation-executions/:id/lease", h.acquireGenerationLease)
		g.PUT("/generation-executions/:id/lease", h.renewGenerationLease)
		g.DELETE("/generation-executions/:id/lease", h.releaseGenerationLease)
		g.PUT("/generation-checkpoints/:id", h.saveGenerationCheckpoint)
		g.GET("/generation-checkpoints/:id", h.getGenerationCheckpoint)

		g.POST("/quota/reservations/:id/confirm", h.confirmQuotaReservation)
		g.POST("/quota/reservations/:id/release", h.releaseQuotaReservation)

		pg := g.Group("/fragment-panels")
		{
			pg.POST("/generate", h.agentStartPanelGeneration)
			pg.GET("/generate/:taskId", h.agentGetPanelGeneration)
		}
	}
}

func (h *AgentPolicyHandler) acquireGenerationLease(c *gin.Context) {
	if h.runtime == nil {
		InternalError(c, "generation runtime unavailable")
		return
	}
	var body struct {
		Owner      string `json:"owner"`
		TTLSeconds int    `json:"ttlSeconds"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		Error(c, CodeInvalidParams, err.Error())
		return
	}
	lease, acquired, err := h.runtime.AcquireLease(c.Request.Context(), c.Param("id"), body.Owner, time.Duration(body.TTLSeconds)*time.Second)
	if err != nil {
		HandleError(c, err)
		return
	}
	Success(c, gin.H{"acquired": acquired, "lease": lease})
}

func (h *AgentPolicyHandler) renewGenerationLease(c *gin.Context) {
	if h.runtime == nil {
		InternalError(c, "generation runtime unavailable")
		return
	}
	var body struct {
		Value      string `json:"value"`
		TTLSeconds int    `json:"ttlSeconds"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		Error(c, CodeInvalidParams, err.Error())
		return
	}
	renewed, err := h.runtime.RenewLease(c.Request.Context(), c.Param("id"), body.Value, time.Duration(body.TTLSeconds)*time.Second)
	if err != nil {
		HandleError(c, err)
		return
	}
	Success(c, gin.H{"renewed": renewed})
}

func (h *AgentPolicyHandler) releaseGenerationLease(c *gin.Context) {
	if h.runtime == nil {
		InternalError(c, "generation runtime unavailable")
		return
	}
	var body struct {
		Value string `json:"value"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		Error(c, CodeInvalidParams, err.Error())
		return
	}
	if err := h.runtime.ReleaseLease(c.Request.Context(), c.Param("id"), body.Value); err != nil {
		HandleError(c, err)
		return
	}
	Success(c, gin.H{"released": true})
}

func (h *AgentPolicyHandler) saveGenerationExecution(c *gin.Context) {
	if h.runtime == nil {
		InternalError(c, "generation runtime unavailable")
		return
	}
	var run domain.GenerationExecution
	if err := c.ShouldBindJSON(&run); err != nil {
		Error(c, CodeInvalidParams, err.Error())
		return
	}
	run.ID = strings.TrimSpace(c.Param("id"))
	if leaseValue := strings.TrimSpace(c.GetHeader("X-Generation-Lease")); leaseValue != "" {
		valid, err := h.runtime.VerifyLease(c.Request.Context(), run.ID, leaseValue)
		if err != nil {
			HandleError(c, err)
			return
		}
		if !valid {
			Error(c, CodeConflict, "generation execution lease lost")
			return
		}
	}
	eventType := strings.TrimSpace(c.Query("eventType"))
	saved, err := h.runtime.SaveExecution(c.Request.Context(), &run, eventType)
	if err != nil {
		HandleError(c, err)
		return
	}
	Success(c, saved)
}

func (h *AgentPolicyHandler) getGenerationExecution(c *gin.Context) {
	if h.runtime == nil {
		InternalError(c, "generation runtime unavailable")
		return
	}
	run, err := h.runtime.GetExecution(c.Request.Context(), c.Param("id"))
	if err != nil {
		HandleError(c, err)
		return
	}
	Success(c, run)
}

func (h *AgentPolicyHandler) listGenerationExecutions(c *gin.Context) {
	if h.runtime == nil {
		InternalError(c, "generation runtime unavailable")
		return
	}
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	runs, err := h.runtime.ListExecutions(c.Request.Context(), c.Query("kind"), limit)
	if err != nil {
		HandleError(c, err)
		return
	}
	Success(c, gin.H{"runs": runs})
}

func (h *AgentPolicyHandler) listGenerationExecutionEvents(c *gin.Context) {
	if h.runtime == nil {
		InternalError(c, "generation runtime unavailable")
		return
	}
	after, _ := strconv.ParseInt(c.DefaultQuery("afterSequence", "0"), 10, 64)
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "500"))
	events, err := h.runtime.ListEvents(c.Request.Context(), c.Param("id"), after, limit)
	if err != nil {
		HandleError(c, err)
		return
	}
	Success(c, gin.H{"events": events})
}

func (h *AgentPolicyHandler) saveGenerationCheckpoint(c *gin.Context) {
	if h.runtime == nil {
		InternalError(c, "generation runtime unavailable")
		return
	}
	var body struct {
		State string `json:"state" binding:"required"`
		RunID string `json:"runId"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		Error(c, CodeInvalidParams, err.Error())
		return
	}
	state, err := base64.StdEncoding.DecodeString(body.State)
	if err != nil {
		Error(c, CodeInvalidParams, "state must be base64 encoded")
		return
	}
	if body.RunID = strings.TrimSpace(body.RunID); body.RunID != "" {
		leaseValue := strings.TrimSpace(c.GetHeader("X-Generation-Lease"))
		if leaseValue == "" {
			Error(c, CodeConflict, "generation checkpoint lease is required")
			return
		}
		valid, verifyErr := h.runtime.VerifyLease(c.Request.Context(), body.RunID, leaseValue)
		if verifyErr != nil {
			HandleError(c, verifyErr)
			return
		}
		if !valid {
			Error(c, CodeConflict, "generation checkpoint lease lost")
			return
		}
	}
	if err := h.runtime.SaveCheckpoint(c.Request.Context(), c.Param("id"), state); err != nil {
		HandleError(c, err)
		return
	}
	Success(c, gin.H{"checkpointId": c.Param("id"), "saved": true})
}

func (h *AgentPolicyHandler) getGenerationCheckpoint(c *gin.Context) {
	if h.runtime == nil {
		InternalError(c, "generation runtime unavailable")
		return
	}
	state, ok, err := h.runtime.GetCheckpoint(c.Request.Context(), c.Param("id"))
	if err != nil {
		HandleError(c, err)
		return
	}
	if !ok {
		Error(c, CodeNotFound, "checkpoint not found")
		return
	}
	Success(c, gin.H{"checkpointId": c.Param("id"), "state": base64.StdEncoding.EncodeToString(state)})
}

func (h *AgentPolicyHandler) internalAPIKeyMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if h.apiKey == "" {
			InternalError(c, "internal agent policy API is not configured")
			return
		}
		key := strings.TrimSpace(c.GetHeader("X-Internal-Api-Key"))
		if key == "" {
			auth := c.GetHeader("Authorization")
			if strings.HasPrefix(auth, "Bearer ") {
				key = strings.TrimSpace(strings.TrimPrefix(auth, "Bearer "))
			}
		}
		if key == "" || key != h.apiKey {
			Unauthorized(c, "invalid internal api key")
			return
		}
		c.Next()
	}
}

func (h *AgentPolicyHandler) tokenStatus(c *gin.Context) {
	if h.policy == nil {
		InternalError(c, "agent policy service unavailable")
		return
	}
	jti := strings.TrimSpace(c.Param("jti"))
	Success(c, gin.H{"jti": jti, "status": h.policy.JTIStatus(c.Request.Context(), jti)})
}

func (h *AgentPolicyHandler) consumeToken(c *gin.Context) {
	if h.policy == nil || h.signer == nil {
		InternalError(c, "agent policy service unavailable")
		return
	}
	jti := strings.TrimSpace(c.Param("jti"))
	ttl := h.signer.TTL() + 30*time.Second
	if err := h.policy.ConsumeJTI(c.Request.Context(), jti, ttl); err != nil {
		Error(c, CodeForbidden, err.Error())
		return
	}
	Success(c, gin.H{"jti": jti, "consumed": true})
}

func (h *AgentPolicyHandler) quotaSnapshot(c *gin.Context) {
	if h.policy == nil {
		InternalError(c, "agent policy service unavailable")
		return
	}
	userID := strings.TrimSpace(c.Param("userId"))
	bal, err := h.policy.QuotaSnapshot(c.Request.Context(), userID)
	if err != nil {
		HandleError(c, err)
		return
	}
	Success(c, gin.H{"userId": userID, "tokenBalance": bal})
}

type agentPolicyAuditBody struct {
	Records []*domain.GenerationStepAuditRecord `json:"records"`
}

func (h *AgentPolicyHandler) recordGenerationAudits(c *gin.Context) {
	if h.audit == nil {
		InternalError(c, "generation audit service unavailable")
		return
	}
	var body agentPolicyAuditBody
	if err := c.ShouldBindJSON(&body); err != nil {
		Error(c, CodeInvalidParams, err.Error())
		return
	}
	if len(body.Records) == 0 {
		Error(c, CodeInvalidParams, "records is required")
		return
	}
	if err := h.audit.RecordBatch(c.Request.Context(), body.Records); err != nil {
		HandleError(c, err)
		return
	}
	Success(c, gin.H{"count": len(body.Records)})
}

func (h *AgentPolicyHandler) listGenerationAudits(c *gin.Context) {
	if h.audit == nil {
		InternalError(c, "generation audit service unavailable")
		return
	}
	runID := strings.TrimSpace(c.Query("runId"))
	taskID := strings.TrimSpace(c.Query("taskId"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "200"))
	var (
		rows []*domain.GenerationStepAuditRecord
		err  error
	)
	switch {
	case runID != "":
		rows, err = h.audit.ListByRunID(c.Request.Context(), runID, limit)
	case taskID != "":
		rows, err = h.audit.ListByTaskID(c.Request.Context(), taskID, limit)
	default:
		Error(c, CodeInvalidParams, "runId or taskId is required")
		return
	}
	if err != nil {
		HandleError(c, err)
		return
	}
	Success(c, gin.H{"records": rows})
}

func (h *AgentPolicyHandler) confirmQuotaReservation(c *gin.Context) {
	if h.policy == nil {
		InternalError(c, "agent policy service unavailable")
		return
	}
	id := strings.TrimSpace(c.Param("id"))
	var body struct {
		ActualTokens int `json:"actualTokens"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		Error(c, CodeInvalidParams, err.Error())
		return
	}
	if body.ActualTokens < 0 {
		Error(c, CodeInvalidParams, "actualTokens must be >= 0")
		return
	}
	if err := h.policy.ConfirmQuotaReservation(c.Request.Context(), id, body.ActualTokens); err != nil {
		HandleError(c, err)
		return
	}
	Success(c, gin.H{"reservationId": id, "confirmed": true, "actualTokens": body.ActualTokens})
}

func (h *AgentPolicyHandler) releaseQuotaReservation(c *gin.Context) {
	if h.policy == nil {
		InternalError(c, "agent policy service unavailable")
		return
	}
	id := strings.TrimSpace(c.Param("id"))
	if err := h.policy.ReleaseQuotaReservation(c.Request.Context(), id); err != nil {
		HandleError(c, err)
		return
	}
	Success(c, gin.H{"reservationId": id, "released": true})
}

type agentPanelGenerateBody struct {
	UserID                 string `json:"userId" binding:"required"`
	UserInput              string `json:"userInput" binding:"required,min=1,max=2000"`
	ReferenceImageURL      string `json:"referenceImageUrl" binding:"required"`
	Style                  string `json:"style"`
	Language               string `json:"language"`
	PanelCount             int    `json:"panelCount"`
	Visibility             string `json:"visibility"`
	Topic                  string `json:"topic"`
	AspectRatio            string `json:"aspectRatio"`
	DialogueMode           string `json:"dialogueMode"`
	ConsistencyLevel       string `json:"consistencyLevel"`
	EnableReferenceAssets  *bool  `json:"enableReferenceAssets"`
	IncludeGenerationTrace bool   `json:"includeGenerationTrace"`
}

func (h *AgentPolicyHandler) agentStartPanelGeneration(c *gin.Context) {
	if h.panelGen == nil {
		InternalError(c, "fragment panel generation service unavailable")
		return
	}
	var req agentPanelGenerateBody
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, CodeInvalidParams, err.Error())
		return
	}
	domainReq := domain.FragmentPanelGenerationRequest{
		UserInput:              strings.TrimSpace(req.UserInput),
		ReferenceImageURL:      strings.TrimSpace(req.ReferenceImageURL),
		Style:                  strings.TrimSpace(req.Style),
		Language:               strings.TrimSpace(req.Language),
		PanelCount:             req.PanelCount,
		Visibility:             strings.TrimSpace(req.Visibility),
		Topic:                  strings.TrimSpace(req.Topic),
		AspectRatio:            strings.TrimSpace(req.AspectRatio),
		DialogueMode:           strings.TrimSpace(req.DialogueMode),
		ConsistencyLevel:       strings.TrimSpace(req.ConsistencyLevel),
		EnableReferenceAssets:  req.EnableReferenceAssets,
		IncludeGenerationTrace: req.IncludeGenerationTrace,
	}
	task, err := h.panelGen.StartGeneration(c.Request.Context(), strings.TrimSpace(req.UserID), domainReq)
	if err != nil {
		Error(c, CodeInvalidParams, err.Error())
		return
	}
	Success(c, gin.H{
		"taskId":          task.ID,
		"draftFragmentId": task.DraftFragmentID,
		"status":          task.Status,
		"progress":        task.Progress,
		"currentStep":     task.CurrentStep,
	})
}

func (h *AgentPolicyHandler) agentGetPanelGeneration(c *gin.Context) {
	if h.panelGen == nil {
		InternalError(c, "fragment panel generation service unavailable")
		return
	}
	userID := strings.TrimSpace(c.Query("userId"))
	if userID == "" {
		Error(c, CodeInvalidParams, "userId query is required")
		return
	}
	taskID := strings.TrimSpace(c.Param("taskId"))
	task, err := h.panelGen.GetTask(c.Request.Context(), taskID, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			NotFound(c, "task not found")
			return
		}
		if errors.Is(err, service.ErrFragmentPanelTaskForbidden) {
			Error(c, CodeForbidden, "forbidden")
			return
		}
		InternalError(c, "failed to load task")
		return
	}
	resp := gin.H{
		"taskId":          task.ID,
		"status":          task.Status,
		"progress":        task.Progress,
		"currentStep":     task.CurrentStep,
		"draftFragmentId": task.DraftFragmentID,
		"error":           task.ErrorMessage,
	}
	if task.Result != nil {
		if task.Result.CombinedContent != "" {
			resp["combinedContent"] = task.Result.CombinedContent
		}
		if len(task.Result.Panels) > 0 {
			panels := make([]gin.H, 0, len(task.Result.Panels))
			for _, p := range task.Result.Panels {
				panels = append(panels, gin.H{"index": p.Index, "imageUrl": p.ImageURL, "caption": p.Caption})
			}
			resp["panels"] = panels
		}
	}
	if task.Metrics != nil {
		resp["metrics"] = gin.H{"totalTokens": task.Metrics.TotalTokens}
	}
	c.JSON(http.StatusOK, gin.H{"code": 1, "message": "success", "data": resp})
}
