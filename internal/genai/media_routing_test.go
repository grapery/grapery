package genapi

import (
	"context"
	"testing"
)

type stubMediaProvider struct {
	name     string
	lastReq  *GenerateRequest
	response *GenerateResponse
}

func (s *stubMediaProvider) Name() string { return s.name }

func (s *stubMediaProvider) GenerateImage(_ context.Context, req *GenerateRequest) (*GenerateResponse, error) {
	s.lastReq = req
	return s.response, nil
}

func (s *stubMediaProvider) Generate(_ context.Context, req *GenerateRequest) (*GenerateResponse, error) {
	s.lastReq = req
	return s.response, nil
}

func (s *stubMediaProvider) GetVideoStatus(_ context.Context, _ string) (*GenerateResponse, error) {
	return s.response, nil
}

// Gemini 只用于文本；任何出图/出片请求都必须落到火山，且不能把 imagen 之类的模型名带过去。
func TestGenerateImageReroutesGeminiToHuoshan(t *testing.T) {
	huoshan := &stubMediaProvider{name: "huoshan", response: &GenerateResponse{ImageURLs: []string{"https://example.com/a.png"}}}
	gemini := &stubMediaProvider{name: "gemini", response: &GenerateResponse{ImageURLs: []string{"https://example.com/b.png"}}}

	g := NewGenAPI()
	g.RegisterProvider(huoshan)
	g.RegisterProvider(gemini)

	rsp, err := g.GenerateImage(context.Background(), "gemini", &GenerateRequest{
		Prompt: "a cat",
		Model:  "imagen-3.0-generate-001",
	})
	if err != nil {
		t.Fatalf("GenerateImage failed: %v", err)
	}
	if gemini.lastReq != nil {
		t.Fatal("gemini must not receive image generation requests")
	}
	if huoshan.lastReq == nil {
		t.Fatal("huoshan should have received the rerouted request")
	}
	if huoshan.lastReq.Model != "" {
		t.Fatalf("foreign model should be dropped on reroute, got %q", huoshan.lastReq.Model)
	}
	if len(rsp.ImageURLs) != 1 || rsp.ImageURLs[0] != "https://example.com/a.png" {
		t.Fatalf("expected huoshan response, got %#v", rsp.ImageURLs)
	}
}

func TestGenerateVideoReroutesGeminiToHuoshan(t *testing.T) {
	huoshan := &stubMediaProvider{name: "huoshan", response: &GenerateResponse{TaskID: "task_huoshan"}}
	gemini := &stubMediaProvider{name: "gemini", response: &GenerateResponse{TaskID: "task_gemini"}}

	g := NewGenAPI()
	g.RegisterProvider(huoshan)
	g.RegisterProvider(gemini)

	if _, err := g.GenerateVideo(context.Background(), "gemini", &GenerateRequest{
		Prompt: "a cat walking",
		Model:  "veo-3.1-generate-preview",
	}); err != nil {
		t.Fatalf("GenerateVideo failed: %v", err)
	}
	if gemini.lastReq != nil {
		t.Fatal("gemini must not receive video generation requests")
	}
	if huoshan.lastReq == nil {
		t.Fatal("huoshan should have received the rerouted request")
	}
	if huoshan.lastReq.Model != "" {
		t.Fatalf("foreign model should be dropped on reroute, got %q", huoshan.lastReq.Model)
	}
}

func TestCoalesceProvidersNeverPickGemini(t *testing.T) {
	g := NewGenAPI()
	g.RegisterProvider(&stubMediaProvider{name: "huoshan"})
	g.RegisterProvider(&stubMediaProvider{name: "gemini"})

	if got := g.CoalesceImageProvider("gemini"); got != "huoshan" {
		t.Fatalf("image coalesce should avoid gemini, got %q", got)
	}
	if got := g.CoalesceVideoProvider("gemini"); got != "huoshan" {
		t.Fatalf("video coalesce should avoid gemini, got %q", got)
	}
}

// 方舟只认像素尺寸，只给长宽比的调用方必须被换算，否则会拿到默认尺寸、比例全错。
func TestHuoshanImageSizeDerivesFromAspectRatio(t *testing.T) {
	cases := map[string]string{
		"16:9": "2560x1440",
		"9:16": "1440x2560",
		"1:1":  "1920x1920",
	}
	for aspect, want := range cases {
		if got := HuoshanPixelSizeForAspectRatio(aspect); got != want {
			t.Fatalf("aspect %s: expected %s, got %s", aspect, want, got)
		}
	}

	if got := huoshanImageSize(&GenerateRequest{Size: "1024x1024", AspectRatio: "16:9"}); got != "1024x1024" {
		t.Fatalf("explicit size must win, got %s", got)
	}
	if got := huoshanImageSize(&GenerateRequest{AspectRatio: "2:3"}); got == "" {
		t.Fatal("aspect-only request should get a derived pixel size")
	}
	// 未识别的比例回退到 16:9，而不是留空让 provider 自己猜。
	if got := HuoshanPixelSizeForAspectRatio("nonsense"); got != "2560x1440" {
		t.Fatalf("unparsable aspect should fall back to 16:9, got %s", got)
	}
}
