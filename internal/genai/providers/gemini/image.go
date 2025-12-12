package gemini

import (
	"context"
	"fmt"
	"strings"

	"google.golang.org/genai"
)

// defaultImageModel is the fallback model when none is provided.
const defaultImageModel = "gemini-2.5-flash-image"

// ImageGenerationOptions configures conversational image generation requests.
type ImageGenerationOptions struct {
	Config      *genai.GenerateContentConfig
	AspectRatio string
}

// ImageAsset represents an inline image payload used as multimodal context.
type ImageAsset struct {
	Data     []byte
	MimeType string
}

// GenerateImages calls the Imagen-style API via the official SDK.
func (c *Client) GenerateImages(ctx context.Context, model, prompt string, cfg *genai.GenerateImagesConfig) (*genai.GenerateImagesResponse, error) {
	prompt = strings.TrimSpace(prompt)
	if prompt == "" {
		return nil, fmt.Errorf("prompt cannot be empty")
	}
	resolvedModel := choose(model, c.config.DefaultModel, defaultImageModel)
	resp, err := c.sdk.Models.GenerateImages(ctx, resolvedModel, prompt, cfg)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// GenerateImageFromText simplifies text-to-image generation returning decoded bytes.
func (c *Client) GenerateImageFromText(ctx context.Context, model, prompt string) ([][]byte, *genai.GenerateImagesResponse, error) {
	resp, err := c.GenerateImages(ctx, model, prompt, &genai.GenerateImagesConfig{OutputMIMEType: "image/png", NumberOfImages: 1})
	if err != nil {
		return nil, nil, err
	}
	var images [][]byte
	for _, generated := range resp.GeneratedImages {
		if generated == nil || generated.Image == nil || len(generated.Image.ImageBytes) == 0 {
			continue
		}
		images = append(images, append([]byte(nil), generated.Image.ImageBytes...))
	}
	return images, resp, nil
}

// GenerateConversationalImageFromText produces images via the multimodal generateContent API using only a text prompt.
func (c *Client) GenerateConversationalImageFromText(ctx context.Context, model, prompt string, opts *ImageGenerationOptions) ([][]byte, *genai.GenerateContentResponse, error) {
	prompt = strings.TrimSpace(prompt)
	if prompt == "" {
		return nil, nil, fmt.Errorf("prompt cannot be empty")
	}
	contents := []*genai.Content{{
		Role:  genai.RoleUser,
		Parts: []*genai.Part{genai.NewPartFromText(prompt)},
	}}
	return c.generateImageWithContents(ctx, model, contents, opts)
}

// EditImageWithPrompt applies textual guidance to a provided image and returns the edited result.
func (c *Client) EditImageWithPrompt(ctx context.Context, model string, image []byte, mimeType, prompt string, opts *ImageGenerationOptions) ([][]byte, *genai.GenerateContentResponse, error) {
	prompt = strings.TrimSpace(prompt)
	if prompt == "" {
		return nil, nil, fmt.Errorf("prompt cannot be empty")
	}
	imagePart, err := encodeInlineImagePart(image, mimeType)
	if err != nil {
		return nil, nil, err
	}
	contents := []*genai.Content{{
		Role: genai.RoleUser,
		Parts: []*genai.Part{
			genai.NewPartFromText(prompt),
			imagePart,
		},
	}}
	return c.generateImageWithContents(ctx, model, contents, opts)
}

// ModifyImageElement is a convenience wrapper for targeted edits such as adding or removing objects.
func (c *Client) ModifyImageElement(ctx context.Context, model string, image []byte, mimeType, prompt string, opts *ImageGenerationOptions) ([][]byte, *genai.GenerateContentResponse, error) {
	return c.EditImageWithPrompt(ctx, model, image, mimeType, prompt, opts)
}

// RepaintImageRegion handles iterative refinements where the prompt focuses on altering specific areas.
func (c *Client) RepaintImageRegion(ctx context.Context, model string, image []byte, mimeType, prompt string, opts *ImageGenerationOptions) ([][]byte, *genai.GenerateContentResponse, error) {
	return c.EditImageWithPrompt(ctx, model, image, mimeType, prompt, opts)
}

// TransferImageStyle merges a content image with a style reference to create a stylized output.
func (c *Client) TransferImageStyle(ctx context.Context, model string, contentImage []byte, contentMime string, styleImage []byte, styleMime, prompt string, opts *ImageGenerationOptions) ([][]byte, *genai.GenerateContentResponse, error) {
	prompt = strings.TrimSpace(prompt)
	if prompt == "" {
		return nil, nil, fmt.Errorf("prompt cannot be empty")
	}
	basePart, err := encodeInlineImagePart(contentImage, contentMime)
	if err != nil {
		return nil, nil, err
	}
	stylePart, err := encodeInlineImagePart(styleImage, styleMime)
	if err != nil {
		return nil, nil, err
	}
	contents := []*genai.Content{{
		Role: genai.RoleUser,
		Parts: []*genai.Part{
			basePart,
			stylePart,
			genai.NewPartFromText(prompt),
		},
	}}
	return c.generateImageWithContents(ctx, model, contents, opts)
}

// ComposeImages synthesizes a new scene from multiple reference images guided by the provided prompt.
func (c *Client) ComposeImages(ctx context.Context, model string, images []ImageAsset, prompt string, opts *ImageGenerationOptions) ([][]byte, *genai.GenerateContentResponse, error) {
	if len(images) == 0 {
		return nil, nil, fmt.Errorf("at least one image asset is required")
	}
	prompt = strings.TrimSpace(prompt)
	if prompt == "" {
		return nil, nil, fmt.Errorf("prompt cannot be empty")
	}
	parts := make([]*genai.Part, 0, len(images)+1)
	for idx, asset := range images {
		part, err := encodeInlineImagePart(asset.Data, asset.MimeType)
		if err != nil {
			return nil, nil, fmt.Errorf("encode image asset %d: %w", idx, err)
		}
		parts = append(parts, part)
	}
	parts = append(parts, genai.NewPartFromText(prompt))
	contents := []*genai.Content{{
		Role:  genai.RoleUser,
		Parts: parts,
	}}
	return c.generateImageWithContents(ctx, model, contents, opts)
}

// GenerateImageImageOnly configures the response to return image payloads without textual commentary.
func (c *Client) GenerateImageImageOnly(ctx context.Context, model, prompt string, aspectRatio string) ([][]byte, *genai.GenerateContentResponse, error) {
	opts := &ImageGenerationOptions{
		Config: &genai.GenerateContentConfig{
			ResponseModalities: []string{"IMAGE"},
		},
		AspectRatio: aspectRatio,
	}
	return c.GenerateConversationalImageFromText(ctx, model, prompt, opts)
}

// GenerateImageWithAspectRatio is a helper for setting only the desired output aspect ratio.
func (c *Client) GenerateImageWithAspectRatio(ctx context.Context, model, prompt, aspectRatio string) ([][]byte, *genai.GenerateContentResponse, error) {
	opts := &ImageGenerationOptions{AspectRatio: aspectRatio}
	return c.GenerateConversationalImageFromText(ctx, model, prompt, opts)
}

func (c *Client) generateImageWithContents(ctx context.Context, model string, contents []*genai.Content, opts *ImageGenerationOptions) ([][]byte, *genai.GenerateContentResponse, error) {
	if len(contents) == 0 {
		return nil, nil, fmt.Errorf("contents cannot be empty")
	}
	cfg := buildImageGenerationConfig(opts)
	resolvedModel := choose(model, c.config.DefaultModel, defaultImageModel)
	resp, err := c.sdk.Models.GenerateContent(ctx, resolvedModel, contents, cfg)
	if err != nil {
		return nil, nil, err
	}
	images, decodeErr := extractInlineImageBytes(resp)
	if decodeErr != nil {
		return nil, resp, decodeErr
	}
	if len(images) == 0 {
		return nil, resp, fmt.Errorf("generateContent returned no image data")
	}
	return images, resp, nil
}

func buildImageGenerationConfig(opts *ImageGenerationOptions) *genai.GenerateContentConfig {
	cfg := &genai.GenerateContentConfig{}
	if opts != nil && opts.Config != nil {
		clone := *opts.Config
		cfg = &clone
	}
	if len(cfg.ResponseModalities) == 0 {
		cfg.ResponseModalities = []string{"IMAGE"}
	} else {
		cfg.ResponseModalities = ensureModality(cfg.ResponseModalities, "IMAGE")
	}
	if opts != nil && strings.TrimSpace(opts.AspectRatio) != "" {
		if cfg.HTTPOptions == nil {
			cfg.HTTPOptions = &genai.HTTPOptions{}
		}
		if cfg.HTTPOptions.ExtraBody == nil {
			cfg.HTTPOptions.ExtraBody = make(map[string]any)
		}
		cfg.HTTPOptions.ExtraBody["imageConfig"] = map[string]any{
			"aspectRatio": strings.TrimSpace(opts.AspectRatio),
		}
	}
	return cfg
}

func ensureModality(modalities []string, modality string) []string {
	for _, m := range modalities {
		if strings.EqualFold(m, modality) {
			return modalities
		}
	}
	return append(modalities, modality)
}

func encodeInlineImagePart(image []byte, mimeType string) (*genai.Part, error) {
	if len(image) == 0 {
		return nil, fmt.Errorf("image data cannot be empty")
	}
	if strings.TrimSpace(mimeType) == "" {
		mimeType = "image/png"
	}
	return genai.NewPartFromBytes(image, mimeType), nil
}

func extractInlineImageBytes(resp *genai.GenerateContentResponse) ([][]byte, error) {
	if resp == nil {
		return nil, fmt.Errorf("response cannot be nil")
	}
	var images [][]byte
	for _, candidate := range resp.Candidates {
		if candidate == nil || candidate.Content == nil {
			continue
		}
		for _, part := range candidate.Content.Parts {
			if part == nil || part.InlineData == nil || len(part.InlineData.Data) == 0 {
				continue
			}
			images = append(images, append([]byte(nil), part.InlineData.Data...))
		}
	}
	return images, nil
}
