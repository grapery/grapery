package http

import (
	"github.com/gin-gonic/gin"
	"github.com/grapestree/fgrapery/grapery/internal/domain"
	"github.com/grapestree/fgrapery/grapery/internal/service"
	"go.uber.org/zap"
)

// ConvertFragmentToStoryRequest 碎片转故事请求（HTTP 层文档；Bind 使用 domain.ConvertFragmentRequest）
type ConvertFragmentToStoryRequest struct {
	Title             string `json:"title" binding:"required"`
	Description       string `json:"description,omitempty"`
	Genre             string `json:"genre,omitempty"`
	CoverImage        string `json:"coverImage,omitempty"`
	SceneCount        int    `json:"sceneCount,omitempty"` // 2–8，缺省由 Handler 纠为 3
	UseAI             bool   `json:"useAI,omitempty"`      // 客户端「AI 一键续写」传 true
	CollaborationType string `json:"collaborationType,omitempty"`
}

// ConvertFragmentToStoryResponse 碎片转故事响应（HTTP 层）
// 仅创建 Story；storyboard 字段省略（旧版曾自动创建根故事板）。
type ConvertFragmentToStoryResponse struct {
	Story      *domain.Story      `json:"story"`
	Storyboard *domain.Storyboard `json:"storyboard,omitempty"`
	FragmentID string             `json:"fragmentId"`
}

// GetFragmentAssetsRequest query for fragment image assets.
type GetFragmentAssetsRequest struct {
	Kind       string `form:"kind"`
	EntityKind string `form:"entityKind"`
	EntityKey  string `form:"entityKey"`
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

// ExpandFragmentStoryPrefillAI POST /api/v1/fragments/:id/story-prefill-ai
func (h *Handler) ExpandFragmentStoryPrefillAI(c *gin.Context) {
	userID, ok := RequireUserID(c)
	if !ok {
		return
	}
	fragmentID := c.Param("id")

	var req domain.FragmentStoryPrefillAIRequest
	_ = c.ShouldBindJSON(&req) // 允许空 body
	if req.SceneCount < 2 || req.SceneCount > 8 {
		req.SceneCount = 3
	}

	resp, err := h.svc.ExpandFragmentStoryPrefillAI(c.Request.Context(), userID, fragmentID, req)
	if err != nil {
		h.logger.Error("fragment story prefill AI failed",
			zap.String("userID", userID),
			zap.String("fragmentID", fragmentID),
			zap.Error(err))
		HandleError(c, err)
		return
	}

	Success(c, resp)
}

// GetFragmentGenerationAssets 查询碎片生成图片资产（含一致性辅助图）。
// GET /api/v1/fragments/:id/assets
func (h *Handler) GetFragmentGenerationAssets(c *gin.Context) {
	userID, ok := RequireUserID(c)
	if !ok {
		return
	}
	fragmentID := c.Param("id")
	if fragmentID == "" {
		InvalidParams(c, "fragment id is required")
		return
	}
	var req GetFragmentAssetsRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		InvalidParams(c, "invalid query")
		return
	}
	items, err := h.svc.ListFragmentGenerationAssets(c.Request.Context(), userID, fragmentID, service.FragmentAssetQuery{
		Kind:       req.Kind,
		EntityKind: req.EntityKind,
		EntityKey:  req.EntityKey,
	})
	if err != nil {
		HandleError(c, err)
		return
	}
	Success(c, gin.H{"fragmentId": fragmentID, "items": items})
}
