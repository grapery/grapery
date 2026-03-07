package gemini

// Model IDs for Gemini API, aligned with official documentation:
// https://ai.google.dev/gemini-api/docs/models
// https://ai.google.dev/gemini-api/docs/text-generation
// https://ai.google.dev/gemini-api/docs/image-generation
// https://ai.google.dev/gemini-api/docs/video

const (
	// DefaultTextModel is the default model for text generation.
	// Gemini 3.1 Flash-Lite: most cost-efficient, 20% higher success rate, 60% faster inference.
	DefaultTextModel = "gemini-3.1-flash-lite-preview"

	// DefaultImageModel is the default model for conversational image generation.
	// Nano Banana 2: text-to-image, image-to-image, multi-image reference, 1K/2K/4K output.
	DefaultImageModel = "gemini-3.1-flash-image-preview"

	// DefaultVideoModel is the default model for video generation.
	// Veo 3.1: up to 8s video with native audio, 720p/1080p/4K, 16:9 or 9:16.
	DefaultVideoModel = "veo-3.1-generate-preview"
)

// Imagen model IDs (use GenerateImages API, not generateContent).
// These are used when req.Model contains "imagen".
const (
	// Imagen3Model is Imagen 3 for high-quality image generation.
	Imagen3Model = "imagen-3.0-generate-002"
	// Imagen4Model is Imagen 4 flagship model with improved text rendering.
	Imagen4Model = "imagen-4.0-generate-001"
)
