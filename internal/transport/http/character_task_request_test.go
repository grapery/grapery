package http

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/grapestree/fgrapery/grapery/internal/config"
	"github.com/grapestree/fgrapery/grapery/internal/domain"
	"github.com/grapestree/fgrapery/grapery/internal/service"
	"go.uber.org/zap"
)

type characterRequestConflictRepo struct {
	domain.Repository
	domain.CharacterTaskExecutionRepository
}

func (r *characterRequestConflictRepo) StoryByID(context.Context, string) (*domain.Story, error) {
	return &domain.Story{UserID: "owner"}, nil
}
func (r *characterRequestConflictRepo) CreateCharacterTaskIdempotent(context.Context, *domain.CharacterGenerationTask) (*domain.CharacterGenerationTask, bool, error) {
	return nil, false, domain.ErrCharacterTaskRequestConflict
}
func TestCharacterTaskRequestConflictReturnsHTTP409(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := service.New(&characterRequestConflictRepo{}, zap.NewNop(), config.RecommendationConfig{})
	handler := &Handler{svc: svc}
	router := gin.New()
	router.POST("/", func(c *gin.Context) { c.Set("userID", "owner"); handler.StartCharacterGenerationTask(c) })
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"clientRequestId":"same-key","storyId":"story","sourceType":"manual_form","name":"雨","description":"changed intent"}`))
	req.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, req)
	if response.Code != http.StatusConflict || !strings.Contains(response.Body.String(), "character_request_conflict") {
		t.Fatal(response.Code, response.Body.String())
	}
}
