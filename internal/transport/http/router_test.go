package http

import (
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// SetupRouter registers overlapping static and wildcard segments (e.g. /storyboards/feed
// next to /storyboards/:id) across several groups; gin panics on a conflict, and nothing
// else in the test suite builds the tree. This keeps that failure out of production boot.
func TestSetupRouterHasNoRouteConflicts(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := SetupRouter(&HandlerDependencies{Logger: zap.NewNop()})
	if router == nil {
		t.Fatal("SetupRouter returned nil")
	}

	want := []struct{ method, path string }{
		{http.MethodGet, "/api/v1/stories/:id"},
		{http.MethodGet, "/api/v1/storyboards/:id"},
		{http.MethodGet, "/api/v1/characters/:id"},
		{http.MethodGet, "/api/v1/storyboards/feed"},
		{http.MethodGet, "/api/v1/public/share/preview"},
		{http.MethodPost, "/api/v1/public/share/open"},
		{http.MethodGet, "/api/public/share/preview"},
		{http.MethodPost, "/api/public/share/open"},
	}

	registered := make(map[string]struct{})
	for _, r := range router.Routes() {
		registered[r.Method+" "+r.Path] = struct{}{}
	}
	for _, w := range want {
		if _, ok := registered[w.method+" "+w.path]; !ok {
			t.Errorf("route not registered: %s %s", w.method, w.path)
		}
	}
}
