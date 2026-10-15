package http

import (
	"encoding/json"
	stdhttp "net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestWorkflowRegistryMiddlewareAbortsWhenAPIKeyIsNotConfigured(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := NewWorkflowRegistryHandler(nil, "")
	assertInternalMiddlewareAborts(t, handler.internalAPIKeyMiddleware(), "configured-key", stdhttp.StatusInternalServerError)
}

func TestWorkflowRegistryMiddlewareAbortsInvalidAPIKey(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := NewWorkflowRegistryHandler(nil, "expected-key")
	assertInternalMiddlewareAborts(t, handler.internalAPIKeyMiddleware(), "wrong-key", stdhttp.StatusUnauthorized)
}

func TestAgentPolicyMiddlewareAbortsWhenAPIKeyIsNotConfigured(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := NewAgentPolicyHandler(nil, nil, nil, nil, nil, "", "", nil)
	assertInternalMiddlewareAborts(t, handler.internalAPIKeyMiddleware(), "configured-key", stdhttp.StatusInternalServerError)
}

func assertInternalMiddlewareAborts(t *testing.T, middleware gin.HandlerFunc, key string, wantStatus int) {
	t.Helper()
	called := false
	router := gin.New()
	router.GET("/internal", middleware, func(c *gin.Context) {
		called = true
		Success(c, gin.H{"unexpected": true})
	})

	request := httptest.NewRequest(stdhttp.MethodGet, "/internal", nil)
	request.Header.Set("X-Internal-Api-Key", key)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	if called {
		t.Fatal("protected handler ran after middleware rejected the request")
	}
	if recorder.Code != wantStatus {
		t.Fatalf("status = %d, want %d", recorder.Code, wantStatus)
	}
	var response Response
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("middleware must emit one valid JSON response: %v; body=%q", err, recorder.Body.String())
	}
}
