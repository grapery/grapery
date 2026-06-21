package http

import (
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

// ListGenerationAudits GET /api/v1/generation-audits?runId=&taskId=
// 用户 JWT 鉴权，仅返回属于当前用户的审计记录。
func (h *Handler) ListGenerationAudits(c *gin.Context) {
	userID, ok := RequireUserID(c)
	if !ok {
		return
	}
	if h.generationAudit == nil {
		InternalError(c, "generation audit service unavailable")
		return
	}
	runID := strings.TrimSpace(c.Query("runId"))
	taskID := strings.TrimSpace(c.Query("taskId"))
	if runID == "" && taskID == "" {
		InvalidParams(c, "runId or taskId is required")
		return
	}
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "200"))
	rows, err := h.generationAudit.ListForUser(c.Request.Context(), userID, runID, taskID, limit)
	if err != nil {
		HandleError(c, err)
		return
	}
	Success(c, gin.H{"records": rows})
}
