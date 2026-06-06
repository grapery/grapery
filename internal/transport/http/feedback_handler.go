package http

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/grapestree/fgrapery/grapery/internal/service"
)

// FeedbackHandler handles /api/v1/feedback
type FeedbackHandler struct {
	svc service.FeedbackService
}

// NewFeedbackHandler creates handler
func NewFeedbackHandler(svc service.FeedbackService) *FeedbackHandler {
	return &FeedbackHandler{svc: svc}
}

const publicSupportGuestUserID = "public-support-guest"

// RegisterFeedbackRoutes under authenticated /v1 group
func (h *FeedbackHandler) RegisterFeedbackRoutes(r *gin.RouterGroup) {
	r.POST("/feedback", h.SubmitFeedback)
	r.GET("/feedback", h.ListMyFeedback)
}

// RegisterPublicSupportRoutes under unauthenticated /api group
func (h *FeedbackHandler) RegisterPublicSupportRoutes(r *gin.RouterGroup) {
	r.POST("/public/support/feedback", h.SubmitPublicSupportFeedback)
}

type submitFeedbackBody struct {
	Category    string `json:"category" binding:"required"`
	Content     string `json:"content" binding:"required"`
	ContactInfo string `json:"contactInfo"`
}

// SubmitFeedback POST /api/v1/feedback
func (h *FeedbackHandler) SubmitFeedback(c *gin.Context) {
	userID, ok := RequireUserID(c)
	if !ok {
		return
	}
	var body submitFeedbackBody
	if !BindJSON(c, &body) {
		return
	}
	fb, err := h.svc.SubmitFeedback(c.Request.Context(), userID, body.Category, body.Content, body.ContactInfo)
	if err != nil {
		HandleError(c, err)
		return
	}
	Success(c, gin.H{
		"id":        fb.ID,
		"status":    fb.Status,
		"message":   "Feedback received",
		"createdAt": fb.CreatedAt,
	})
}

type submitPublicSupportFeedbackBody struct {
	Category    string `json:"category" binding:"required"`
	Content     string `json:"content" binding:"required"`
	ContactInfo string `json:"contactInfo" binding:"required"`
}

// SubmitPublicSupportFeedback POST /api/public/support/feedback
func (h *FeedbackHandler) SubmitPublicSupportFeedback(c *gin.Context) {
	var body submitPublicSupportFeedbackBody
	if !BindJSON(c, &body) {
		return
	}
	fb, err := h.svc.SubmitFeedback(c.Request.Context(), publicSupportGuestUserID, body.Category, body.Content, body.ContactInfo)
	if err != nil {
		HandleError(c, err)
		return
	}
	Success(c, gin.H{
		"id":        fb.ID,
		"status":    fb.Status,
		"message":   "Support request received",
		"createdAt": fb.CreatedAt,
	})
}

// ListMyFeedback GET /api/v1/feedback?limit=&offset=
func (h *FeedbackHandler) ListMyFeedback(c *gin.Context) {
	userID, ok := RequireUserID(c)
	if !ok {
		return
	}
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	items, total, err := h.svc.ListMyFeedback(c.Request.Context(), userID, limit, offset)
	if err != nil {
		HandleError(c, err)
		return
	}
	Success(c, gin.H{
		"feedbacks": items,
		"total":     total,
		"limit":     limit,
		"offset":    offset,
	})
}
