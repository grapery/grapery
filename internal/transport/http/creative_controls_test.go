package http

import (
	"encoding/json"
	"github.com/gin-gonic/gin"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/grapestree/fgrapery/grapery/internal/domain"
	"github.com/grapestree/fgrapery/grapery/internal/service"
)

func TestCreationRequestCreativeControlsRejectUnknownCapabilities(t *testing.T) {
	for _, req := range []any{&GenerateFragmentRequest{}, &service.GenerateCharacterRequest{}, &service.CharacterGenerationTaskRequest{}, &domain.Storyboard{}, &continueStoryboardPayload{}} {
		if err := json.Unmarshal([]byte(`{"creativeControls":{"systemPrompt":"override identity","retry":30}}`), req); err == nil {
			t.Fatalf("unsafe controls accepted by %T", req)
		}
		if err := json.Unmarshal([]byte(`{"creativeControls":{"version":1,"pacing":"slow","examples":[{"input":"quiet moment","output":"show one observable action"}]}}`), req); err != nil {
			t.Fatalf("valid optional controls rejected by %T: %v", req, err)
		}
	}
}

func TestCreativeRequestBindingRejectsExplicitNullButAllowsOmission(t *testing.T) {
	for _, tc := range []struct {
		body  string
		valid bool
	}{
		{`{}`, true}, {`{"creativeControls":{}}`, true}, {`{"creativeControls":null}`, false},
		{`{"creativeControls":{"version":0}}`, false}, {`{"creativeControls":{"tone":""}}`, false},
		{`{"creativeControls":{"pacing":null}}`, false}, {`{"creativeControls":{"examples":[{"input":"x","output":null}]}}`, false},
	} {
		context, _ := gin.CreateTestContext(httptest.NewRecorder())
		context.Request = httptest.NewRequest("POST", "/", strings.NewReader(tc.body))
		context.Request.Header.Set("Content-Type", "application/json")
		var request struct {
			CreativeControls *domain.CreativeControls `json:"creativeControls,omitempty"`
		}
		err := bindCreativeRequest(context, &request)
		if (err == nil) != tc.valid {
			t.Fatalf("wire contract %s: %v", tc.body, err)
		}
	}
}
