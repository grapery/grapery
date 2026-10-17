package http

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// RecheckQuality never restarts generation; clients receive the same complete
// status contract used by polling after the server-owned metadata is committed.
func (h *FragmentGenerationHandler) RecheckQuality(c *gin.Context) {
	userID := c.GetString("userID")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	if _, err := h.fragmentGenService.RecheckQuality(c.Request.Context(), c.Param("taskId"), userID); err != nil {
		status, code, message := fragmentGenerationHTTPError(err)
		c.JSON(status, gin.H{"code": code, "message": message})
		return
	}
	h.GetGenerationStatus(c)
}
