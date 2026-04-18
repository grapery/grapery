package service

import (
	"strings"

	genapi "github.com/grapestree/fgrapery/grapery/internal/genai"
)

// IsOverseasUserRegion returns true when the user's region is treated as non-CN (e.g. US, EU).
// Empty or CN is domestic. Kept for analytics / future routing; generation defaults no longer switch to Gemini solely by region.
func IsOverseasUserRegion(region string) bool {
	r := strings.ToUpper(strings.TrimSpace(region))
	if r == "" || r == "CN" {
		return false
	}
	return true
}

// ResolvePanelGenerationAIProviders chooses Step1 plan + Step2 image providers.
// Text/plan and default image routing prioritize Huoshan; explicit AI_IMAGE_PROVIDER still applies to image when set.
func ResolvePanelGenerationAIProviders(_ string, defaultImage string, _ *AIGenerationService) (planProvider, imageProvider string) {
	imageProvider = strings.TrimSpace(defaultImage)
	if imageProvider == "" {
		imageProvider = "huoshan"
	}
	return "huoshan", imageProvider
}

// ResolveTextPlanProvider returns the default LLM provider for text / multimodal JSON planning: Huoshan first policy.
// Callers may still pass explicit "gemini" or "huoshan" via NormalizeTextPlanProvider.
func ResolveTextPlanProvider(_ string, _ *AIGenerationService) string {
	return "huoshan"
}

// NormalizeTextPlanProvider keeps an explicit "gemini" or "huoshan"; otherwise resolves from userRegion (never kling or other media-only providers).
func NormalizeTextPlanProvider(requested, userRegion string, ai *AIGenerationService) string {
	p := strings.ToLower(strings.TrimSpace(requested))
	if p == "gemini" || p == "huoshan" {
		return p
	}
	return ResolveTextPlanProvider(userRegion, ai)
}

// CoalesceRegisteredVideoProvider returns a video provider name registered on GenAPI, preferring preferred then huoshan.
func CoalesceRegisteredVideoProvider(g *genapi.GenAPI, preferred string) string {
	if g == nil {
		p := strings.TrimSpace(preferred)
		if p == "" {
			return "huoshan"
		}
		return p
	}
	return g.CoalesceVideoProvider(preferred)
}

// CoalesceRegisteredImageProvider returns a provider name that is actually registered on GenAPI, preferring preferred then huoshan.
func CoalesceRegisteredImageProvider(g *genapi.GenAPI, preferred string) string {
	if g == nil {
		p := strings.TrimSpace(preferred)
		if p == "" {
			return "huoshan"
		}
		return p
	}
	return g.CoalesceImageProvider(preferred)
}
