package http

import (
	"strconv"

	"github.com/gin-gonic/gin"
)

func (h *Handler) AddStoryTags(c *gin.Context) {
	storyID := c.Param("id")
	var req struct {
		Tags []string `json:"tags" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		InvalidParams(c, err.Error())
		return
	}

	if err := h.svc.AddStoryTags(c.Request.Context(), storyID, req.Tags); err != nil {
		InternalError(c, err.Error())
		return
	}

	Success(c, gin.H{"message": "tags added successfully"})
}

func (h *Handler) GetStoryTags(c *gin.Context) {
	storyID := c.Param("id")
	tags, err := h.svc.GetStoryTags(c.Request.Context(), storyID)
	if err != nil {
		InternalError(c, err.Error())
		return
	}

	Success(c, gin.H{"tags": tags})
}

func (h *Handler) GetStoriesByTag(c *gin.Context) {
	tagID := c.Param("id")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	stories, err := h.svc.GetStoriesByTag(c.Request.Context(), tagID, limit, offset)
	if err != nil {
		InternalError(c, err.Error())
		return
	}

	Success(c, gin.H{"stories": stories, "count": len(stories)})
}

func (h *Handler) GetPopularTags(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	tags, err := h.svc.GetPopularTags(c.Request.Context(), limit)
	if err != nil {
		InternalError(c, err.Error())
		return
	}

	Success(c, gin.H{"tags": tags})
}
