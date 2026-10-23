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

func TestCharacterMalformedVisualRequestDoesNotDefaultToBillableGeneration(t *testing.T) {
	for _, kind := range []string{"portrait", "threeViews"} {
		for _, body := range []string{`{"clientRequestId":12}`, `{"clientRequestId":"bad`, `{"clientRequestId":"` + strings.Repeat("x", 257) + `"}`} {
			t.Run(kind+body[:10], func(t *testing.T) {
				gin.SetMode(gin.TestMode)
				router := gin.New()
				handler := &Handler{} // No service: invalid request cannot enter model/billing.
				router.POST("/:id", func(c *gin.Context) {
					c.Set("userID", "owner")
					if kind == "portrait" {
						handler.GenerateCharacterPortrait(c)
					} else {
						handler.GenerateCharacterThreeViews(c)
					}
				})
				req := httptest.NewRequest(http.MethodPost, "/character", strings.NewReader(body))
				req.Header.Set("Content-Type", "application/json")
				response := httptest.NewRecorder()
				router.ServeHTTP(response, req)
				if response.Code != http.StatusBadRequest {
					t.Fatal(response.Code, response.Body.String())
				}
			})
		}
	}
}

type characterCropOwnerRepo struct{ domain.Repository }

func (*characterCropOwnerRepo) CharacterByID(context.Context, string) (*domain.Character, error) {
	return &domain.Character{BaseModel: common.BaseModel{ID: "character"}, Author: &domain.User{BaseModel: common.BaseModel{ID: "owner"}}, Avatar: "owner-avatar", Portrait: "portrait"}, nil
}
func TestCharacterCropAvatarRejectsCrossUserHTTP403(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := &Handler{svc: service.New(&characterCropOwnerRepo{}, zap.NewNop(), config.RecommendationConfig{})}
	router := gin.New()
	router.POST("/:id", func(c *gin.Context) { c.Set("userID", "attacker"); handler.CropAvatarFromPortrait(c) })
	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/character", nil))
	if response.Code != http.StatusForbidden {
		t.Fatal(response.Code, response.Body.String())
	}
}
