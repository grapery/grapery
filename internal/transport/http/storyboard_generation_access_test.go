package http

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/grapestree/fgrapery/grapery/internal/common"
	"github.com/grapestree/fgrapery/grapery/internal/config"
	"github.com/grapestree/fgrapery/grapery/internal/domain"
	"github.com/grapestree/fgrapery/grapery/internal/service"
	"go.uber.org/zap"
)

type storyboardDeniedGenerationRepository struct{ domain.Repository }

func (*storyboardDeniedGenerationRepository) StoryboardByID(context.Context, string) (*domain.Storyboard, error) {
	return &domain.Storyboard{BaseModel: common.BaseModel{ID: "board"}, UserID: "owner", WorkflowStatus: domain.WorkflowStatusDraft}, nil
}

func TestStoryboardGenerationEndpointsDenyForeignOwnerBeforeAnyJobs(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewHandler(service.New(&storyboardDeniedGenerationRepository{}, zap.NewNop(), config.RecommendationConfig{}), nil, zap.NewNop())
	for name, handler := range map[string]gin.HandlerFunc{
		"content": h.GenerateContent, "scene-details": h.GenerateSceneDetails, "image": h.GenerateStoryboardImage, "images": h.GenerateAllStoryboardImages, "comic-page": h.GenerateStoryboardComicPage, "comic-pages": h.GenerateAllStoryboardComicPages, "video": h.GenerateStoryboardVideo, "retry-failed-images": h.RetryFailedStoryboardImages,
	} {
		t.Run(name, func(t *testing.T) {
			router := gin.New()
			router.Use(func(c *gin.Context) { c.Set("userID", "stranger"); c.Next() })
			router.POST("/storyboards/:id/"+name, handler)
			response := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodPost, "/storyboards/board/"+name, strings.NewReader(`{}`))
			request.Header.Set("Content-Type", "application/json")
			router.ServeHTTP(response, request)
			if response.Code != http.StatusForbidden {
				t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
			}
		})
	}
}

func TestStoryboardGenerationTraceAndPublishAreCreatorOnly(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := NewHandler(service.New(&storyboardDeniedGenerationRepository{}, zap.NewNop(), config.RecommendationConfig{}), nil, zap.NewNop())
	for name, handler := range map[string]gin.HandlerFunc{"progress": h.GetGenerationProgress, "publish": h.PublishStoryboard} {
		t.Run(name, func(t *testing.T) {
			router := gin.New()
			router.Use(func(c *gin.Context) { c.Set("userID", "stranger"); c.Next() })
			router.POST("/storyboards/:id/"+name, handler)
			response := httptest.NewRecorder()
			router.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/storyboards/board/"+name, nil))
			if response.Code != http.StatusForbidden {
				t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
			}
		})
	}
}
