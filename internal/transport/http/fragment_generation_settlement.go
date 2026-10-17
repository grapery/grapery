package http

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func (h *FragmentGenerationHandler) SettleGeneration(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	if _, err := h.fragmentGenService.SettleGeneration(c.Request.Context(), c.Param("taskId"), userID); err != nil {
		status, code, message := fragmentGenerationHTTPError(err)
		c.JSON(status, gin.H{"code": code, "message": message})
		return
	}
	h.GetGenerationStatus(c)
}
