package http

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/grapestree/fgrapery/grapery/internal/config"
	"github.com/grapestree/fgrapery/grapery/internal/domain"
	"github.com/grapestree/fgrapery/grapery/internal/service"
	"go.uber.org/zap"
)

type hiddenStoryboardPageRepository struct {
	domain.Repository
	limit, offset int
}

func (r *hiddenStoryboardPageRepository) StoryboardsByCreator(_ context.Context, _ string, limit, offset int) ([]*domain.Storyboard, error) {
	r.limit, r.offset = limit, offset
	return []*domain.Storyboard{{StoryID: "hidden-1"}, {StoryID: "hidden-2"}}, nil
}

func (r *hiddenStoryboardPageRepository) UserSettings(context.Context, string) (*domain.UserSettings, error) {
	return &domain.UserSettings{ShowPublicStories: false}, nil
}

func TestGetUserStoryboardsHTTPPaginationIncludesHasMoreOnHiddenPage(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &hiddenStoryboardPageRepository{}
	logger := zap.NewNop()
	h := NewHandler(service.New(repo, logger, config.RecommendationConfig{}), nil, logger)
	router := gin.New()
	router.GET("/users/:id/storyboards", h.GetUserStoryboards)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/users/owner/storyboards?limit=2&offset=-1", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d, body=%s", w.Code, w.Body.String())
	}
	var response struct {
		Code int                        `json:"code"`
		Data map[string]json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if response.Code != CodeSuccess || string(response.Data["hasMore"]) != "true" || string(response.Data["count"]) != "0" || string(response.Data["storyboards"]) != "[]" {
		t.Fatalf("unexpected pagination contract: %s", w.Body.String())
	}
	if _, ok := response.Data["has_more"]; ok {
		t.Fatalf("pagination must use camelCase hasMore: %s", w.Body.String())
	}
	if repo.limit != 2 || repo.offset != 0 {
		t.Fatalf("repository pagination=(%d, %d), want (2, 0)", repo.limit, repo.offset)
	}
}

func TestGetUserStoryboardsHTTPNormalizesPagination(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, tc := range []struct {
		name, query   string
		limit, offset int
	}{
		{"defaults", "", 20, 0},
		{"invalid", "?limit=nope&offset=nope", 20, 0},
		{"overflow", "?limit=999999999999999999999999999&offset=999999999999999999999999999", 20, 0},
		{"negative", "?limit=-1&offset=-4", 20, 0},
		{"clamped", "?limit=101&offset=7", 100, 7},
	} {
		t.Run(tc.name, func(t *testing.T) {
			repo := &hiddenStoryboardPageRepository{}
			logger := zap.NewNop()
			h := NewHandler(service.New(repo, logger, config.RecommendationConfig{}), nil, logger)
			router := gin.New()
			router.GET("/users/:id/storyboards", h.GetUserStoryboards)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/users/owner/storyboards"+tc.query, nil))
			if w.Code != http.StatusOK || repo.limit != tc.limit || repo.offset != tc.offset {
				t.Fatalf("status=%d, pagination=(%d, %d), want (200, %d, %d)", w.Code, repo.limit, repo.offset, tc.limit, tc.offset)
			}
		})
	}
}
