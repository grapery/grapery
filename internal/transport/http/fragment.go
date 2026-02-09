package http

import (
	"github.com/gin-gonic/gin"
	"github.com/grapestree/fgrapery/grapery/internal/domain"
	"go.uber.org/zap"
)

// ConvertFragmentToStoryRequest 碎片转故事请求（HTTP 层）
// 注意：实际使用 domain.ConvertFragmentRequest，这里仅用于文档说明
type ConvertFragmentToStoryRequest struct {
	Title             string `json:"title" binding:"required"`
	Description       string `json:"description,omitempty"`
	Genre             string `json:"genre,omitempty"`
	CoverImage        string `json:"coverImage,omitempty"`
	SceneCount        int    `json:"sceneCount,omitempty"`
	UseAI             bool   `json:"useAI,omitempty"`
	CollaborationType string `json:"collaborationType,omitempty"`
}

// ConvertFragmentToStoryResponse 碎片转故事响应（HTTP 层）
type ConvertFragmentToStoryResponse struct {
	Story      *domain.Story      `json:"story"`
	Storyboard *domain.Storyboard `json:"storyboard"`
	FragmentID string             `json:"fragmentId"`
}

// ConvertFragmentToStory 碎片转故事
// POST /api/fragments/:id/convert-to-story
func (h *Handler) ConvertFragmentToStory(c *gin.Context) {
	userID, ok := RequireUserID(c)
	if !ok {
		return
	}

	fragmentID := c.Param("id")

	var req domain.ConvertFragmentRequest
	if !BindJSON(c, &req) {
		return
	}

	// 验证场景数量
	if req.SceneCount < 2 || req.SceneCount > 8 {
		req.SceneCount = 3
	}

	// 调用 service 层方法
	resp, err := h.svc.ConvertFragmentToStory(c.Request.Context(), userID, fragmentID, req)
	if err != nil {
		h.logger.Error("failed to convert fragment to story",
			zap.String("userID", userID),
			zap.String("fragmentID", fragmentID),
			zap.Error(err))
		HandleError(c, err)
		return
	}

	Success(c, resp)
}
