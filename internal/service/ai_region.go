package service

import (
	"strings"

	genapi "github.com/grapestree/fgrapery/grapery/internal/genai"
)

// IsOverseasUserRegion returns true when the user should prefer Gemini (海外) over Huoshan (国内默认).
// Empty or CN is treated as domestic; any other region code (e.g. US, EU, JP, OTHER) is overseas.
func IsOverseasUserRegion(region string) bool {
	r := strings.ToUpper(strings.TrimSpace(region))
	if r == "" || r == "CN" {
		return false
	}
	return true
}

// PreferGeminiForFragmentText 海外用户且已配置 Gemini 时优先用 Gemini 生成碎片文案。
func PreferGeminiForFragmentText(region string, geminiConfigured bool) bool {
	return geminiConfigured && IsOverseasUserRegion(region)
}

// ResolvePanelGenerationAIProviders chooses Step1 plan + Step2 image providers: 国内默认 Huoshan；海外且 Gemini 可用时用 Gemini，否则回退 Huoshan。
func ResolvePanelGenerationAIProviders(userRegion, defaultImage string, ai *AIGenerationService) (planProvider, imageProvider string) {
	imageProvider = strings.TrimSpace(defaultImage)
	if imageProvider == "" {
		imageProvider = "huoshan"
	}
	if PreferGeminiForFragmentText(userRegion, ai != nil && ai.GeminiAvailable()) {
		return "gemini", "gemini"
	}
	return "huoshan", imageProvider
}

// ResolveTextPlanProvider returns the LLM provider for text / multimodal JSON planning only: "gemini" or "huoshan".
// It reuses the same region + Gemini availability rules as fragment story text (see ResolvePanelGenerationAIProviders).
func ResolveTextPlanProvider(userRegion string, ai *AIGenerationService) string {
	p, _ := ResolvePanelGenerationAIProviders(userRegion, "", ai)
	return p
}

// NormalizeTextPlanProvider keeps an explicit "gemini" or "huoshan"; otherwise resolves from userRegion (never kling or other media-only providers).
func NormalizeTextPlanProvider(requested, userRegion string, ai *AIGenerationService) string {
	p := strings.ToLower(strings.TrimSpace(requested))
	if p == "gemini" || p == "huoshan" {
		return p
	}
	return ResolveTextPlanProvider(userRegion, ai)
}

// CoalesceRegisteredImageProvider returns a provider name that is actually registered on GenAPI, preferring `preferred` then huoshan.
func CoalesceRegisteredImageProvider(g *genapi.GenAPI, preferred string) string {
	p := strings.TrimSpace(preferred)
	if p == "" {
		p = "huoshan"
	}
	if g == nil {
		return p
	}
	if g.GetImageProvider(p) != nil {
		return p
	}
	if p != "huoshan" && g.GetImageProvider("huoshan") != nil {
		return "huoshan"
	}
	return p
}
