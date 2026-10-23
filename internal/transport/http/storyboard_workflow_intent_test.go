package http

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/grapestree/fgrapery/grapery/internal/domain"
	"github.com/grapestree/fgrapery/grapery/internal/service"
)

func TestStoryboardWorkflowIntentConflictReturnsHTTP409(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/", func(c *gin.Context) {
		handleStoryboardWorkflowStageError(c, fmt.Errorf("context: %w: %w (targetSceneIds)", domain.ErrAlreadyExists, service.ErrStoryboardRequestIntentConflict))
	})
	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/", nil))
	if response.Code != http.StatusConflict || !strings.Contains(response.Body.String(), "storyboard_request_intent_conflict") {
		t.Fatalf("wrong conflict response %d %s", response.Code, response.Body.String())
	}
}
