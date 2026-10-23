package http

import (
	"github.com/gin-gonic/gin"
	"net/http"
)

func (h *FragmentGenerationHandler) RetryGeneration(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	task, draftID, err := h.fragmentGenService.RetryGeneration(c.Request.Context(), c.Param("taskId"), userID)
	if err != nil {
		status, code, message := fragmentGenerationHTTPError(err)
		c.JSON(status, gin.H{"error": message, "code": code})
		return
	}
	c.JSON(http.StatusAccepted, gin.H{"taskId": task.ID, "draftFragmentId": draftID, "status": task.Status, "progress": task.Progress})
}
