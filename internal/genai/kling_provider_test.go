package genapi

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestKlingText2VideoCreatesCompositeTaskID(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && r.URL.Path == "/v1/videos/text2video" {
			b, _ := io.ReadAll(r.Body)
			if !strings.Contains(string(b), `"prompt":"hello"`) {
				t.Errorf("body=%s", string(b))
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"code":0,"message":"ok","request_id":"r1","data":{"task_id":"tid-1","task_status":"submitted"}}`))
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()

	p, err := newKlingProvider(&Config{
		Provider: ProviderKling,
		APIKey:   "kling-access",
		Secret:   "kling-secret-key-material",
		BaseURL:  srv.URL,
	})
	if err != nil {
		t.Fatal(err)
	}
	resp, err := p.Generate(context.Background(), &GenerateRequest{
		Operation: OperationTextToVideo,
		Prompt:    "hello",
		Model:     "kling-v1",
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.TaskID != "kling:t2v:tid-1" {
		t.Fatalf("task id: %q", resp.TaskID)
	}
}

func TestKlingOmniVideoCapabilityRoute(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && r.URL.Path == "/v1/videos/omni-video" {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"code":0,"message":"ok","request_id":"r2","data":{"task_id":"ov-1","task_status":"submitted"}}`))
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()

	p, err := newKlingProvider(&Config{Provider: ProviderKling, APIKey: "a", Secret: "b", BaseURL: srv.URL})
	if err != nil {
		t.Fatal(err)
	}
	resp, err := p.Generate(context.Background(), &GenerateRequest{
		Operation: OperationTextToVideo,
		Prompt:    "a video <<<image_1>>>",
		Model:     "kling-video-o1",
		Options: map[string]interface{}{
			"kling_capability": KlingCapabilityOmniVideo,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.TaskID != "kling:ovid:ov-1" {
		t.Fatalf("task id: %q", resp.TaskID)
	}
}

func TestKlingCallbackURLFromGenerateRequest(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && r.URL.Path == "/v1/videos/text2video" {
			b, _ := io.ReadAll(r.Body)
			if !strings.Contains(string(b), `"callback_url":"https://cb.example/hook"`) {
				t.Errorf("missing callback in body: %s", string(b))
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"code":0,"message":"ok","request_id":"r","data":{"task_id":"t1","task_status":"submitted"}}`))
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()

	p, err := newKlingProvider(&Config{Provider: ProviderKling, APIKey: "a", Secret: "b", BaseURL: srv.URL})
	if err != nil {
		t.Fatal(err)
	}
	_, err = p.Generate(context.Background(), &GenerateRequest{
		Operation:   OperationTextToVideo,
		Prompt:      "x",
		Model:       "kling-v1",
		CallbackURL: "https://cb.example/hook",
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestKlingGetVideoStatusMapsSucceed(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/v1/videos/text2video/") {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"code":0,"message":"ok","request_id":"r3","data":{"task_id":"tid-1","task_status":"succeed","task_result":{"videos":[{"url":"https://example.com/v.mp4"}]}}}`))
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()

	p, err := newKlingProvider(&Config{Provider: ProviderKling, APIKey: "a", Secret: "b", BaseURL: srv.URL})
	if err != nil {
		t.Fatal(err)
	}
	resp, err := p.GetVideoStatus(context.Background(), "kling:t2v:tid-1")
	if err != nil {
		t.Fatal(err)
	}
	if resp.Status != string(StatusCompleted) || resp.VideoURL != "https://example.com/v.mp4" {
		t.Fatalf("status=%q url=%q", resp.Status, resp.VideoURL)
	}
}
