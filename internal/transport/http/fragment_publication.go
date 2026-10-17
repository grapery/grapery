package http

import (
	"github.com/gin-gonic/gin"
	"net/http"
)

func (h *FragmentGenerationHandler) PreparePublication(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	var req struct {
		ImageURLs []string `json:"imageUrls" binding:"required,min=1,max=8,dive,required,max=2048"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid publication image manifest"})
		return
	}
	if _, err := h.fragmentGenService.PreparePublication(c.Request.Context(), c.Param("taskId"), userID, req.ImageURLs); err != nil {
		status, code, message := fragmentGenerationHTTPError(err)
		c.JSON(status, gin.H{"code": code, "message": message})
		return
	}
	h.GetGenerationStatus(c)
}
