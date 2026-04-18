package service

import (
	"testing"

	"github.com/grapestree/fgrapery/grapery/internal/genai/providers/gemini"
)

func TestNormalizeTextPlanProvider_explicitGeminiOrHuoshan(t *testing.T) {
	ai := &AIGenerationService{geminiClient: &gemini.Client{}}
	if got := NormalizeTextPlanProvider("gemini", "CN", nil); got != "gemini" {
		t.Fatalf("gemini explicit: got %q", got)
	}
	if got := NormalizeTextPlanProvider("Huoshan", "US", ai); got != "huoshan" {
		t.Fatalf("huoshan explicit: got %q", got)
	}
}

func TestNormalizeTextPlanProvider_klingResolvesByRegion(t *testing.T) {
	if got := NormalizeTextPlanProvider("kling", "CN", nil); got != "huoshan" {
		t.Fatalf("CN + kling: want huoshan, got %q", got)
	}
	aiGemini := &AIGenerationService{geminiClient: &gemini.Client{}}
	if got := NormalizeTextPlanProvider("kling", "US", aiGemini); got != "huoshan" {
		t.Fatalf("US + gemini + kling requested: want huoshan default, got %q", got)
	}
	if got := NormalizeTextPlanProvider("kling", "US", nil); got != "huoshan" {
		t.Fatalf("US + no gemini + kling: want huoshan, got %q", got)
	}
}

func TestNormalizeTextPlanProvider_emptyUsesResolve(t *testing.T) {
	if got := NormalizeTextPlanProvider("", "CN", nil); got != "huoshan" {
		t.Fatalf("empty CN: want huoshan, got %q", got)
	}
	aiGemini := &AIGenerationService{geminiClient: &gemini.Client{}}
	if got := NormalizeTextPlanProvider("  ", "US", aiGemini); got != "huoshan" {
		t.Fatalf("empty US + gemini: want huoshan default, got %q", got)
	}
}

func TestResolveTextPlanProvider(t *testing.T) {
	aiGemini := &AIGenerationService{geminiClient: &gemini.Client{}}
	if got := ResolveTextPlanProvider("CN", nil); got != "huoshan" {
		t.Fatalf("CN nil ai: got %q", got)
	}
	if got := ResolveTextPlanProvider("US", nil); got != "huoshan" {
		t.Fatalf("US nil ai: got %q", got)
	}
	if got := ResolveTextPlanProvider("US", aiGemini); got != "huoshan" {
		t.Fatalf("US + gemini: want huoshan default, got %q", got)
	}
}
