package doubao

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

var (
	DoubaoAPIKey = os.Getenv("DOUBAO_API_KEY")
)

// SeedreamClient represents a client for the Doubao SeedDream image generation and embedding service
type SeedreamClient struct {
	APIKey     string
	Endpoint   string
	HTTPClient *http.Client
}

// NewSeedreamClient creates a new SeedDream client
func NewSeedreamClient(apiKey string) *SeedreamClient {
	if apiKey == "" {
		apiKey = DoubaoAPIKey
	}
	return &SeedreamClient{
		APIKey:   apiKey,
		Endpoint: "https://ark.cn-beijing.volces.com",
		HTTPClient: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

// TextToImageRequest represents the request parameters for text-to-image generation
type TextToImageRequest struct {
	Model          string `json:"model"`
	Prompt         string `json:"prompt"`
	NegativePrompt string `json:"negative_prompt,omitempty"`
	ResponseFormat string `json:"response_format,omitempty"` // "url" or "b64_json"
	Size           string `json:"size,omitempty"`            // "1024x1024", "1024x1792", "1792x1024", etc.
	GuidanceScale  int    `json:"guidance_scale,omitempty"`  // 1-20, default 3
	Watermark      *bool  `json:"watermark,omitempty"`       // default true
	Seed           *int64 `json:"seed,omitempty"`            // for reproducible results
	N              *int   `json:"n,omitempty"`               // number of images to generate, 1-4
	Quality        string `json:"quality,omitempty"`         // "standard" or "hd"
	Style          string `json:"style,omitempty"`           // "vivid" or "natural"
}

// ImageGenerationResponse represents the response from the image generation API
type ImageGenerationResponse struct {
	Model   string `json:"model"`
	Created int64  `json:"created"`
	Data    []struct {
		URL           string `json:"url,omitempty"`
		B64JSON       string `json:"b64_json,omitempty"`
		RevisedPrompt string `json:"revised_prompt,omitempty"`
	} `json:"data"`
	Usage struct {
		GeneratedImages int `json:"generated_images"`
	} `json:"usage"`
	Error *APIError `json:"error,omitempty"`
}

// APIError represents an error response from the API
type APIError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Type    string `json:"type"`
	Param   string `json:"param,omitempty"`
}

// Error implements the error interface
func (e *APIError) Error() string {
	return fmt.Sprintf("API Error [%s]: %s", e.Code, e.Message)
}

// Image generation models
const (
	ModelSeedream40       = "doubao-seedream-4-0-250828"     // SeedDream 4.0
	ModelSeedream30T2I    = "doubao-seedream-3-0-t2i-250415" // SeedDream 3.0 text-to-image
	ModelSeedream30T2IPro = "doubao-seedream-3-0-t2i-pro"    // SeedDream 3.0 text-to-image Pro
	ModelSeedream21T2I    = "doubao-seedream-2-1-t2i"        // SeedDream 2.1 text-to-image
	ModelSeedream20T2I    = "doubao-seedream-2-0-t2i"        // SeedDream 2.0 text-to-image
)

// Common image sizes
const (
	SizeSquare1024    = "1024x1024"
	SizePortrait1024  = "1024x1792"
	SizeLandscape1024 = "1792x1024"
	SizeSquare512     = "512x512"
	SizeSquare768     = "768x768"
	Size2K            = "2K" // New 2K size for SeedDream 4.0
	Size4K            = "4K" // New 4K size for SeedDream 4.0
)

// Response formats
const (
	ResponseFormatURL     = "url"
	ResponseFormatB64JSON = "b64_json"
)

// Quality levels
const (
	QualityStandard = "standard"
	QualityHD       = "hd"
)

// Styles
const (
	StyleVivid   = "vivid"
	StyleNatural = "natural"
)

// Sequential image generation modes
const (
	SequentialDisabled = "disabled"
	SequentialAuto     = "auto"
	SequentialManual   = "manual"
)

// SeedDream4Request represents the new SeedDream 4.0 API request format
type SeedDream4Request struct {
	Model                            string                            `json:"model"`
	Prompt                           string                            `json:"prompt"`
	Image                            interface{}                       `json:"image,omitempty"`                       // string or []string
	Size                             string                            `json:"size,omitempty"`                        // "2K", "4K", etc.
	SequentialImageGeneration        string                            `json:"sequential_image_generation,omitempty"` // "disabled", "auto", "manual"
	SequentialImageGenerationOptions *SequentialImageGenerationOptions `json:"sequential_image_generation_options,omitempty"`
	Stream                           *bool                             `json:"stream,omitempty"`
	ResponseFormat                   string                            `json:"response_format,omitempty"`
	Watermark                        *bool                             `json:"watermark,omitempty"`
}

// SequentialImageGenerationOptions represents options for sequential image generation
type SequentialImageGenerationOptions struct {
	MaxImages int `json:"max_images,omitempty"` // Maximum number of images to generate
}

// SeedDream4Response represents the new SeedDream 4.0 API response format
type SeedDream4Response struct {
	Model   string `json:"model"`
	Created int64  `json:"created"`
	Data    []struct {
		URL  string `json:"url,omitempty"`
		Size string `json:"size,omitempty"` // "3104x1312", "2048x2048", etc.
	} `json:"data"`
	Usage struct {
		GeneratedImages int `json:"generated_images"`
		OutputTokens    int `json:"output_tokens,omitempty"`
		TotalTokens     int `json:"total_tokens,omitempty"`
	} `json:"usage"`
	Error *APIError `json:"error,omitempty"`
}

// Text Embedding API structures

// TextEmbeddingRequest represents the request for text embedding
type TextEmbeddingRequest struct {
	Model          string   `json:"model"`
	Input          []string `json:"input"`
	EncodingFormat string   `json:"encoding_format,omitempty"` // "float" or "base64"
}

// TextEmbeddingResponse represents the response from text embedding API
type TextEmbeddingResponse struct {
	Created int64  `json:"created"`
	ID      string `json:"id"`
	Data    []struct {
		Embedding []float32 `json:"embedding"`
		Index     int       `json:"index"`
		Object    string    `json:"object"`
	} `json:"data"`
	Model  string `json:"model"`
	Object string `json:"object"`
	Usage  struct {
		PromptTokens int `json:"prompt_tokens"`
		TotalTokens  int `json:"total_tokens"`
	} `json:"usage"`
	Error *APIError `json:"error,omitempty"`
}

// Multimodal Embedding API structures

// MultimodalInput represents different types of input for multimodal embedding
type MultimodalInput struct {
	Type     string                `json:"type"` // "text", "image_url", "video_url"
	Text     string                `json:"text,omitempty"`
	ImageURL *MultimodalURLContent `json:"image_url,omitempty"`
	VideoURL *MultimodalURLContent `json:"video_url,omitempty"`
}

// MultimodalURLContent represents URL content for images or videos
type MultimodalURLContent struct {
	URL string `json:"url"`
}

// MultimodalEmbeddingRequest represents the request for multimodal embedding
type MultimodalEmbeddingRequest struct {
	Model          string            `json:"model"`
	Input          []MultimodalInput `json:"input"`
	EncodingFormat string            `json:"encoding_format,omitempty"` // "float" or "base64"
}

// MultimodalEmbeddingResponse represents the response from multimodal embedding API
type MultimodalEmbeddingResponse struct {
	Created int64  `json:"created"`
	ID      string `json:"id"`
	Data    struct {
		Embedding []float32 `json:"embedding"`
		Object    string    `json:"object"`
	} `json:"data"`
	Model  string `json:"model"`
	Object string `json:"object"`
	Usage  struct {
		PromptTokens        int `json:"prompt_tokens"`
		PromptTokensDetails struct {
			ImageTokens int `json:"image_tokens"`
			TextTokens  int `json:"text_tokens"`
		} `json:"prompt_tokens_details"`
		TotalTokens int `json:"total_tokens"`
	} `json:"usage"`
	Error *APIError `json:"error,omitempty"`
}

// Embedding models
const (
	ModelEmbeddingText   = "doubao-embedding-text-240715"   // Text embedding model
	ModelEmbeddingVision = "doubao-embedding-vision-250615" // Multimodal embedding model
)

// Encoding formats
const (
	EncodingFormatFloat  = "float"
	EncodingFormatBase64 = "base64"
)

/*
文生图，单张图片生成
输入示例：
curl -X POST https://ark.cn-beijing.volces.com/api/v3/images/generations \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $ARK_API_KEY" \
  -d '{
    "model": "doubao-seedream-4-0-250828",
    "prompt": "星际穿越，黑洞，黑洞里冲出一辆快支离破碎的复古列车，抢视觉冲击力，电影大片，末日既视感，动感，对比色，oc渲染，光线追踪，动态模糊，景深，超现实主义，深蓝，画面通过细腻的丰富的色彩层次塑造主体与场景，质感真实，暗黑风背景的光影效果营造出氛围，整体兼具艺术幻想感，夸张的广角透视效果，耀光，反射，极致的光影，强引力，吞噬",
    "size": "2K",
    "sequential_image_generation": "disabled",
    "stream": false,
    "response_format": "url",
    "watermark": true
}'

响应结果：
{
    "model": "doubao-seedream-4-0-250828",
    "created": 1757321139,
    "data": [
        {
            "url": "https://...",
            "size": "3104x1312"
        }
    ],
    "usage": {
        "generated_images": 1,
        "output_tokens": xxx,
        "total_tokens": xxx
    }
}
*/

/*
图生图，单张图片生成

输入示例：
curl -X POST https://ark.cn-beijing.volces.com/api/v3/images/generations \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $ARK_API_KEY" \
  -d '{
    "model": "doubao-seedream-4-0-250828",
    "prompt": "生成狗狗趴在草地上的近景画面",
    "image": "https://ark-project.tos-cn-beijing.volces.com/doc_image/seedream4_imageToimage.png",
    "size": "2K",
    "sequential_image_generation": "disabled",
    "stream": false,
    "response_format": "url",
    "watermark": true
}'

响应结果：
{
    "model": "doubao-seedream-4-0-250828",
    "created": 1757321139,
    "data": [
        {
            "url": "https://...",
            "size": "3104x1312"
        }
    ],
    "usage": {
        "generated_images": 1,
        "output_tokens": xxx,
        "total_tokens": xxx
    }
}


*/

/*
图生图，多张图片生成一张图片

输入示例：
curl -X POST https://ark.cn-beijing.volces.com/api/v3/images/generations \

  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $ARK_API_KEY" \
  -d '{
    "model": "doubao-seedream-4-0-250828",
    "prompt": "将图1的服装换为图2的服装",
    "image": ["https://ark-project.tos-cn-beijing.volces.com/doc_image/seedream4_imagesToimage_1.png", "https://ark-project.tos-cn-beijing.volces.com/doc_image/seedream4_imagesToimage_2.png"],
    "size": "2K",
    "sequential_image_generation": "disabled",
    "stream": false,
    "response_format": "url",
    "watermark": true
}'

响应结果：
{
    "model": "doubao-seedream-4-0-250828",
    "created": 1757323851,
    "data": [
        {
            "url": "https://...",
            "size": "2048x2048"
        }
    ],
    "usage": {
        "generated_images": 1,
        "output_tokens": 16384,
        "total_tokens": 16384
    }
}


文生组图，生成多张图片

输入示例：
curl -X POST https://ark.cn-beijing.volces.com/api/v3/images/generations \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $ARK_API_KEY" \
  -d '{
    "model": "doubao-seedream-4-0-250828",
    "prompt": "生成一组共4张连贯插画，核心为同一庭院一角的四季变迁，以统一风格展现四季独特色彩、元素与氛围",
    "size": "2K",
    "sequential_image_generation": "auto",
    "sequential_image_generation_options": {
        "max_images": 4
    },
    "stream": false,
    "response_format": "url",
    "watermark": true
}'

响应结果：
{
    "model": "doubao-seedream-4-0-250828",
    "created": 1757322902,
    "data": [
        {
            "url": "https://...",
            "size": "2336x1760"
        },
        {
            "url": "https://...",
            "size": "2336x1760"
        },
        {
            "url": "https://...",
            "size": "2336x1760"
        },
        {
            "url": "https://...",
            "size": "2336x1760"
        }
    ],
    "usage": {
        "generated_images": 4,
        "output_tokens": 64240,
        "total_tokens": 64240
    }
}

*/

// GenerateTextToImage generates a single image from text using SeedDream 4.0
func (c *SeedreamClient) GenerateTextToImage(ctx context.Context, prompt string, options ...func(*SeedDream4Request)) (*SeedDream4Response, error) {
	req := &SeedDream4Request{
		Model:                     ModelSeedream40,
		Prompt:                    prompt,
		Size:                      Size2K,
		SequentialImageGeneration: SequentialDisabled,
		Stream:                    BoolPtr(false),
		ResponseFormat:            ResponseFormatURL,
		Watermark:                 BoolPtr(true),
	}

	// Apply options
	for _, option := range options {
		option(req)
	}

	return c.generateSeedDream4Image(ctx, req)
}

// GenerateImageToImage generates an image from another image using SeedDream 4.0
func (c *SeedreamClient) GenerateImageToImage(ctx context.Context, prompt string, imageURL string, options ...func(*SeedDream4Request)) (*SeedDream4Response, error) {
	req := &SeedDream4Request{
		Model:                     ModelSeedream40,
		Prompt:                    prompt,
		Image:                     imageURL,
		Size:                      Size2K,
		SequentialImageGeneration: SequentialDisabled,
		Stream:                    BoolPtr(false),
		ResponseFormat:            ResponseFormatURL,
		Watermark:                 BoolPtr(true),
	}

	// Apply options
	for _, option := range options {
		option(req)
	}

	return c.generateSeedDream4Image(ctx, req)
}

// GenerateMultiImageToImage generates an image from multiple input images using SeedDream 4.0
func (c *SeedreamClient) GenerateMultiImageToImage(ctx context.Context, prompt string, imageURLs []string, options ...func(*SeedDream4Request)) (*SeedDream4Response, error) {
	req := &SeedDream4Request{
		Model:                     ModelSeedream40,
		Prompt:                    prompt,
		Image:                     imageURLs,
		Size:                      Size2K,
		SequentialImageGeneration: SequentialDisabled,
		Stream:                    BoolPtr(false),
		ResponseFormat:            ResponseFormatURL,
		Watermark:                 BoolPtr(true),
	}

	// Apply options
	for _, option := range options {
		option(req)
	}

	return c.generateSeedDream4Image(ctx, req)
}

// GenerateSequentialImages generates multiple coherent images using SeedDream 4.0
func (c *SeedreamClient) GenerateSequentialImages(ctx context.Context, prompt string, maxImages int, options ...func(*SeedDream4Request)) (*SeedDream4Response, error) {
	req := &SeedDream4Request{
		Model:                     ModelSeedream40,
		Prompt:                    prompt,
		Size:                      Size2K,
		SequentialImageGeneration: SequentialAuto,
		SequentialImageGenerationOptions: &SequentialImageGenerationOptions{
			MaxImages: maxImages,
		},
		Stream:         BoolPtr(false),
		ResponseFormat: ResponseFormatURL,
		Watermark:      BoolPtr(true),
	}

	// Apply options
	for _, option := range options {
		option(req)
	}

	return c.generateSeedDream4Image(ctx, req)
}

// generateSeedDream4Image is the core method that handles SeedDream 4.0 API calls
func (c *SeedreamClient) generateSeedDream4Image(ctx context.Context, req *SeedDream4Request) (*SeedDream4Response, error) {
	// Marshal request body
	reqBody, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, "POST",
		c.Endpoint+"/api/v3/images/generations",
		bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.APIKey)

	// Send request
	resp, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Parse response
	var result SeedDream4Response
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Check for API errors
	if resp.StatusCode != http.StatusOK {
		if result.Error != nil {
			return nil, result.Error
		}
		return nil, fmt.Errorf("request failed with status %d: %s", resp.StatusCode, string(body))
	}

	return &result, nil
}

// Option functions for SeedDream 4.0 API
func WithSeedDream4Size(size string) func(*SeedDream4Request) {
	return func(req *SeedDream4Request) {
		req.Size = size
	}
}

func WithSeedDream4Watermark(watermark bool) func(*SeedDream4Request) {
	return func(req *SeedDream4Request) {
		req.Watermark = &watermark
	}
}

func WithSeedDream4ResponseFormat(format string) func(*SeedDream4Request) {
	return func(req *SeedDream4Request) {
		req.ResponseFormat = format
	}
}

func WithSeedDream4Stream(stream bool) func(*SeedDream4Request) {
	return func(req *SeedDream4Request) {
		req.Stream = &stream
	}
}

func WithSeedDream4SequentialMode(mode string) func(*SeedDream4Request) {
	return func(req *SeedDream4Request) {
		req.SequentialImageGeneration = mode
	}
}

func WithSeedDream4MaxImages(maxImages int) func(*SeedDream4Request) {
	return func(req *SeedDream4Request) {
		if req.SequentialImageGenerationOptions == nil {
			req.SequentialImageGenerationOptions = &SequentialImageGenerationOptions{}
		}
		req.SequentialImageGenerationOptions.MaxImages = maxImages
	}
}

// Helper functions to create pointers
func BoolPtr(b bool) *bool {
	return &b
}

func Int64Ptr(i int64) *int64 {
	return &i
}

func IntPtr(i int) *int {
	return &i
}

// Legacy GenerateImage method for backward compatibility
func (c *SeedreamClient) GenerateImage(ctx context.Context, req *TextToImageRequest) (*ImageGenerationResponse, error) {
	// Set default values
	if req.Model == "" {
		req.Model = ModelSeedream30T2I
	}
	if req.ResponseFormat == "" {
		req.ResponseFormat = ResponseFormatURL
	}
	if req.Size == "" {
		req.Size = SizeSquare1024
	}
	if req.GuidanceScale == 0 {
		req.GuidanceScale = 3
	}
	if req.Watermark == nil {
		watermark := true
		req.Watermark = &watermark
	}
	if req.N == nil {
		n := 1
		req.N = &n
	}
	if req.Quality == "" {
		req.Quality = QualityStandard
	}
	if req.Style == "" {
		req.Style = StyleVivid
	}

	// Marshal request body
	reqBody, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, "POST",
		c.Endpoint+"/api/v3/images/generations",
		bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.APIKey)

	// Send request
	resp, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Parse response
	var result ImageGenerationResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Check for API errors
	if resp.StatusCode != http.StatusOK {
		if result.Error != nil {
			return nil, result.Error
		}
		return nil, fmt.Errorf("request failed with status %d: %s", resp.StatusCode, string(body))
	}

	return &result, nil
}

// SimpleGenerateImage provides a simplified interface for image generation
func (c *SeedreamClient) SimpleGenerateImage(ctx context.Context, prompt string, options ...func(*TextToImageRequest)) (*ImageGenerationResponse, error) {
	req := &TextToImageRequest{
		Prompt: prompt,
	}

	// Apply options
	for _, option := range options {
		option(req)
	}

	return c.GenerateImage(ctx, req)
}

// Option functions for SimpleGenerateImage
func WithModel(model string) func(*TextToImageRequest) {
	return func(req *TextToImageRequest) {
		req.Model = model
	}
}

func WithNegativePrompt(negativePrompt string) func(*TextToImageRequest) {
	return func(req *TextToImageRequest) {
		req.NegativePrompt = negativePrompt
	}
}

func WithSize(size string) func(*TextToImageRequest) {
	return func(req *TextToImageRequest) {
		req.Size = size
	}
}

func WithGuidanceScale(scale int) func(*TextToImageRequest) {
	return func(req *TextToImageRequest) {
		req.GuidanceScale = scale
	}
}

func WithWatermark(watermark bool) func(*TextToImageRequest) {
	return func(req *TextToImageRequest) {
		req.Watermark = &watermark
	}
}

func WithSeed(seed int64) func(*TextToImageRequest) {
	return func(req *TextToImageRequest) {
		req.Seed = &seed
	}
}

func WithImageCount(n int) func(*TextToImageRequest) {
	return func(req *TextToImageRequest) {
		req.N = &n
	}
}

func WithQuality(quality string) func(*TextToImageRequest) {
	return func(req *TextToImageRequest) {
		req.Quality = quality
	}
}

func WithStyle(style string) func(*TextToImageRequest) {
	return func(req *TextToImageRequest) {
		req.Style = style
	}
}

func WithResponseFormat(format string) func(*TextToImageRequest) {
	return func(req *TextToImageRequest) {
		req.ResponseFormat = format
	}
}

// Embedding API Methods

// GenerateTextEmbedding generates embeddings for text inputs
func (c *SeedreamClient) GenerateTextEmbedding(ctx context.Context, texts []string, options ...func(*TextEmbeddingRequest)) (*TextEmbeddingResponse, error) {
	req := &TextEmbeddingRequest{
		Model:          ModelEmbeddingText,
		Input:          texts,
		EncodingFormat: EncodingFormatFloat,
	}

	// Apply options
	for _, option := range options {
		option(req)
	}

	return c.generateTextEmbedding(ctx, req)
}

// GenerateMultimodalEmbedding generates embeddings for multimodal inputs (text, images, videos)
func (c *SeedreamClient) GenerateMultimodalEmbedding(ctx context.Context, inputs []MultimodalInput, options ...func(*MultimodalEmbeddingRequest)) (*MultimodalEmbeddingResponse, error) {
	req := &MultimodalEmbeddingRequest{
		Model:          ModelEmbeddingVision,
		Input:          inputs,
		EncodingFormat: EncodingFormatFloat,
	}

	// Apply options
	for _, option := range options {
		option(req)
	}

	return c.generateMultimodalEmbedding(ctx, req)
}

// generateTextEmbedding is the core method for text embedding API calls
func (c *SeedreamClient) generateTextEmbedding(ctx context.Context, req *TextEmbeddingRequest) (*TextEmbeddingResponse, error) {
	// Marshal request body
	reqBody, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, "POST",
		c.Endpoint+"/api/v3/embeddings",
		bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.APIKey)

	// Send request
	resp, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Parse response
	var result TextEmbeddingResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Check for API errors
	if resp.StatusCode != http.StatusOK {
		if result.Error != nil {
			return nil, result.Error
		}
		return nil, fmt.Errorf("request failed with status %d: %s", resp.StatusCode, string(body))
	}

	return &result, nil
}

// generateMultimodalEmbedding is the core method for multimodal embedding API calls
func (c *SeedreamClient) generateMultimodalEmbedding(ctx context.Context, req *MultimodalEmbeddingRequest) (*MultimodalEmbeddingResponse, error) {
	// Marshal request body
	reqBody, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, "POST",
		c.Endpoint+"/api/v3/embeddings/multimodal",
		bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.APIKey)

	// Send request
	resp, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Parse response
	var result MultimodalEmbeddingResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Check for API errors
	if resp.StatusCode != http.StatusOK {
		if result.Error != nil {
			return nil, result.Error
		}
		return nil, fmt.Errorf("request failed with status %d: %s", resp.StatusCode, string(body))
	}

	return &result, nil
}

// Helper functions for creating multimodal inputs

// NewTextInput creates a text input for multimodal embedding
func NewTextInput(text string) MultimodalInput {
	return MultimodalInput{
		Type: "text",
		Text: text,
	}
}

// NewImageURLInput creates an image URL input for multimodal embedding
func NewImageURLInput(url string) MultimodalInput {
	return MultimodalInput{
		Type:     "image_url",
		ImageURL: &MultimodalURLContent{URL: url},
	}
}

// NewVideoURLInput creates a video URL input for multimodal embedding
func NewVideoURLInput(url string) MultimodalInput {
	return MultimodalInput{
		Type:     "video_url",
		VideoURL: &MultimodalURLContent{URL: url},
	}
}

// Option functions for embedding APIs

// WithTextEmbeddingModel sets the model for text embedding
func WithTextEmbeddingModel(model string) func(*TextEmbeddingRequest) {
	return func(req *TextEmbeddingRequest) {
		req.Model = model
	}
}

// WithTextEmbeddingFormat sets the encoding format for text embedding
func WithTextEmbeddingFormat(format string) func(*TextEmbeddingRequest) {
	return func(req *TextEmbeddingRequest) {
		req.EncodingFormat = format
	}
}

// WithMultimodalEmbeddingModel sets the model for multimodal embedding
func WithMultimodalEmbeddingModel(model string) func(*MultimodalEmbeddingRequest) {
	return func(req *MultimodalEmbeddingRequest) {
		req.Model = model
	}
}

// WithMultimodalEmbeddingFormat sets the encoding format for multimodal embedding
func WithMultimodalEmbeddingFormat(format string) func(*MultimodalEmbeddingRequest) {
	return func(req *MultimodalEmbeddingRequest) {
		req.EncodingFormat = format
	}
}
