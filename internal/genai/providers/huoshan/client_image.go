package huoshan

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	arkmodel "github.com/volcengine/volcengine-go-sdk/service/arkruntime/model"
)

// ImageMode defines Seedream generation modes (Seedream 5.0 lite).
// 文生图、图文生图、多图融合、单张图生组图、文生组图、多参考图生组图、联网搜索生图
const (
	ImageModeTextToImage       = "text_to_image"        // 文生图
	ImageModeImageTextToImage  = "image_text_to_image"  // 图文生图
	ImageModeMultiImageFusion  = "multi_image_fusion"   // 多图融合（多图输入单图输出）
	ImageModeSingleToImageSet  = "single_to_image_set"  // 单张图生组图
	ImageModeTextToImageSet    = "text_to_image_set"    // 文生组图
	ImageModeMultiRefToImageSet = "multi_ref_to_image_set" // 多参考图生组图
	ImageModeWebSearchToImage  = "web_search_to_image"  // 联网搜索生图（需启用 optimize_prompt）
)

// ImageGenerationRequest represents the payload for Doubao Seedream image generation.
// Supports: 文生图、图文生图、多图融合、单张图生组图、文生组图、多参考图生组图、联网搜索生图
type ImageGenerationRequest struct {
	Model                            string                 `json:"model"`
	Prompt                           string                 `json:"prompt"`
	Image                            interface{}            `json:"image,omitempty"`
	Size                             string                 `json:"size,omitempty"`
	Seed                             int64                  `json:"seed,omitempty"`
	SequentialImageGeneration        string                 `json:"sequential_image_generation,omitempty"`
	SequentialImageGenerationOptions map[string]interface{} `json:"sequential_image_generation_options,omitempty"`
	Stream                           bool                   `json:"stream,omitempty"`
	GuidanceScale                    float64                `json:"guidance_scale,omitempty"`
	ResponseFormat                   string                 `json:"response_format,omitempty"`
	Watermark                        *bool                  `json:"watermark,omitempty"`
	OptimizePromptOptions            map[string]interface{} `json:"optimize_prompt_options,omitempty"`
	// Mode hints Seedream mode; when set, overrides SequentialImageGeneration/Image logic
	Mode       string `json:"mode,omitempty"`
	MaxImages  int    `json:"max_images,omitempty"`  // For image set modes (单张图生组图、文生组图、多参考图生组图)
	WebSearch  bool   `json:"web_search,omitempty"`  // 联网搜索生图: enable web search for prompt
}

// ImageGenerationResponse captures the response from the image generation endpoint.
type ImageGenerationResponse struct {
	Model   string                    `json:"model"`
	Created int64                     `json:"created"`
	Data    []ImageGenerationData     `json:"data"`
	Usage   *ImageGenerationUsage     `json:"usage,omitempty"`
	Error   *ImageGenerationErrorBody `json:"error,omitempty"`
}

// ImageGenerationData describes individual image outputs.
type ImageGenerationData struct {
	URL     string `json:"url,omitempty"`
	B64JSON string `json:"b64_json,omitempty"`
	Size    string `json:"size,omitempty"`
}

// ImageGenerationUsage reports resource usage for image generation.
type ImageGenerationUsage struct {
	GeneratedImages int `json:"generated_images"`
	OutputTokens    int `json:"output_tokens"`
	TotalTokens     int `json:"total_tokens"`
}

// ImageGenerationErrorBody captures error information returned by the API.
type ImageGenerationErrorBody struct {
	Message string `json:"message,omitempty"`
	Type    string `json:"type,omitempty"`
	Param   string `json:"param,omitempty"`
	Code    string `json:"code,omitempty"`
}

func (c *Client) GenerateImage(ctx context.Context, payload *ImageGenerationRequest) (*ImageGenerationResponse, error) {
	return c.generateImage(ctx, payload, false)
}

// GenerateImageWithReference generates an image with one or more reference images.
func (c *Client) GenerateImageWithReference(ctx context.Context, payload *ImageGenerationRequest) (*ImageGenerationResponse, error) {
	return c.generateImage(ctx, payload, true)
}

func (c *Client) generateImage(ctx context.Context, payload *ImageGenerationRequest, requireImage bool) (*ImageGenerationResponse, error) {
	if c.arkClient == nil {
		return nil, fmt.Errorf("image client is not configured")
	}

	req, err := c.prepareImageRequest(payload, requireImage)
	if err != nil {
		return nil, err
	}

	resp, err := c.arkClient.GenerateImages(ctx, req)
	if err != nil {
		return nil, err
	}

	return toImageGenerationResponse(resp), nil
}

func (c *Client) prepareImageRequest(payload *ImageGenerationRequest, requireImage bool) (arkmodel.GenerateImagesRequest, error) {
	if payload == nil {
		return arkmodel.GenerateImagesRequest{}, fmt.Errorf("payload cannot be nil")
	}

	prompt := strings.TrimSpace(payload.Prompt)
	if prompt == "" {
		return arkmodel.GenerateImagesRequest{}, fmt.Errorf("prompt is required")
	}

	modelName := strings.TrimSpace(payload.Model)
	if modelName == "" {
		modelName = strings.TrimSpace(choose(c.config.ImageModel, defaultImageModel))
		if modelName == "" {
			return arkmodel.GenerateImagesRequest{}, fmt.Errorf("model is required")
		}
	}

	// Derive SequentialImageGeneration, OptimizePrompt, and requireImage from Mode
	sequentialMode := strings.TrimSpace(payload.SequentialImageGeneration)
	mode := strings.TrimSpace(payload.Mode)
	effectiveRequireImage := requireImage
	if mode != "" {
		switch mode {
		case ImageModeTextToImageSet, ImageModeSingleToImageSet, ImageModeMultiRefToImageSet:
			sequentialMode = "enabled"
			if mode == ImageModeSingleToImageSet || mode == ImageModeMultiRefToImageSet {
				effectiveRequireImage = true // 单张图生组图、多参考图生组图必须提供参考图
			}
			if payload.SequentialImageGenerationOptions == nil {
				payload.SequentialImageGenerationOptions = make(map[string]interface{})
			}
			if payload.MaxImages > 0 && payload.SequentialImageGenerationOptions["max_images"] == nil {
				payload.SequentialImageGenerationOptions["max_images"] = payload.MaxImages
			} else if payload.MaxImages == 0 && payload.SequentialImageGenerationOptions["max_images"] == nil {
				// 组图模式未指定时默认 4 张
				payload.SequentialImageGenerationOptions["max_images"] = 4
			}
		case ImageModeImageTextToImage, ImageModeMultiImageFusion:
			effectiveRequireImage = true // 图文生图、多图融合必须提供图
			if mode == ImageModeMultiImageFusion {
				sequentialMode = "disabled"
			}
		case ImageModeWebSearchToImage:
			if payload.OptimizePromptOptions == nil {
				payload.OptimizePromptOptions = make(map[string]interface{})
			}
			payload.OptimizePromptOptions["enable"] = true
		default:
			sequentialMode = "disabled"
		}
	}
	if sequentialMode == "" {
		sequentialMode = "disabled"
	}
	seqMode := arkmodel.SequentialImageGeneration(sequentialMode)

	responseFormat := strings.TrimSpace(payload.ResponseFormat)
	if responseFormat == "" {
		responseFormat = "url"
	}

	request := arkmodel.GenerateImagesRequest{
		Model:                     modelName,
		Prompt:                    prompt,
		SequentialImageGeneration: ptr(seqMode),
		ResponseFormat:            ptr(responseFormat),
	}

	if trimmedSize := strings.TrimSpace(payload.Size); trimmedSize != "" {
		request.Size = ptr(trimmedSize)
	}

	if payload.Seed != 0 {
		seed := payload.Seed
		request.Seed = &seed
	}

	if payload.GuidanceScale > 0 {
		guidance := payload.GuidanceScale
		request.GuidanceScale = &guidance
	}

	if payload.Watermark != nil {
		request.Watermark = payload.Watermark
	}

	if len(payload.SequentialImageGenerationOptions) > 0 {
		opts := toSequentialImageGenerationOptions(payload.SequentialImageGenerationOptions)
		if opts != nil {
			request.SequentialImageGenerationOptions = opts
		}
	}

	if payload.WebSearch {
		enable := true
		request.OptimizePrompt = &enable
	}
	if len(payload.OptimizePromptOptions) > 0 {
		if optimize := toBoolPointer(payload.OptimizePromptOptions["enable"]); optimize != nil {
			request.OptimizePrompt = optimize
		}
	}

	if payload.Image != nil {
		normalized, err := normalizeReferenceImages(payload.Image)
		if err != nil {
			return arkmodel.GenerateImagesRequest{}, err
		}
		request.Image = normalized
	} else if effectiveRequireImage {
		if mode != "" {
			return arkmodel.GenerateImagesRequest{}, fmt.Errorf("image is required for mode %s", mode)
		}
		return arkmodel.GenerateImagesRequest{}, fmt.Errorf("image is required")
	}

	return request, nil
}

func toImageGenerationResponse(resp arkmodel.ImagesResponse) *ImageGenerationResponse {
	result := &ImageGenerationResponse{
		Model:   resp.Model,
		Created: resp.Created,
	}

	if len(resp.Data) > 0 {
		result.Data = make([]ImageGenerationData, 0, len(resp.Data))
		for _, item := range resp.Data {
			if item == nil {
				continue
			}
			entry := ImageGenerationData{Size: item.Size}
			if item.Url != nil {
				entry.URL = *item.Url
			}
			if item.B64Json != nil {
				entry.B64JSON = *item.B64Json
			}
			result.Data = append(result.Data, entry)
		}
	}

	if resp.Usage != nil {
		result.Usage = &ImageGenerationUsage{
			GeneratedImages: int(resp.Usage.GeneratedImages),
			OutputTokens:    int(resp.Usage.OutputTokens),
			TotalTokens:     int(resp.Usage.TotalTokens),
		}
	}

	if resp.Error != nil {
		result.Error = &ImageGenerationErrorBody{
			Message: resp.Error.Message,
			Code:    resp.Error.Code,
		}
	}

	return result
}

func toSequentialImageGenerationOptions(options map[string]interface{}) *arkmodel.SequentialImageGenerationOptions {
	if len(options) == 0 {
		return nil
	}

	result := &arkmodel.SequentialImageGenerationOptions{}
	if max, ok := options["max_images"]; ok {
		if converted := toIntPointer(max); converted != nil {
			result.MaxImages = converted
		}
	}
	if result.MaxImages == nil {
		if max, ok := options["maxImages"]; ok {
			if converted := toIntPointer(max); converted != nil {
				result.MaxImages = converted
			}
		}
	}

	if result.MaxImages == nil {
		return nil
	}
	return result
}

func toIntPointer(value interface{}) *int {
	switch v := value.(type) {
	case int:
		val := v
		return &val
	case *int:
		if v == nil {
			return nil
		}
		val := *v
		return &val
	case int32:
		val := int(v)
		return &val
	case *int32:
		if v == nil {
			return nil
		}
		val := int(*v)
		return &val
	case int64:
		val := int(v)
		return &val
	case *int64:
		if v == nil {
			return nil
		}
		val := int(*v)
		return &val
	case uint:
		val := int(v)
		return &val
	case uint32:
		val := int(v)
		return &val
	case uint64:
		val := int(v)
		return &val
	case float32:
		if float32(int(v)) != v {
			return nil
		}
		val := int(v)
		return &val
	case float64:
		if float64(int(v)) != v {
			return nil
		}
		val := int(v)
		return &val
	case string:
		trimmed := strings.TrimSpace(v)
		if trimmed == "" {
			return nil
		}
		parsed, err := strconv.Atoi(trimmed)
		if err != nil {
			return nil
		}
		return &parsed
	case json.Number:
		parsedInt, err := v.Int64()
		if err != nil {
			return nil
		}
		val := int(parsedInt)
		return &val
	default:
		return nil
	}
}

func toBoolPointer(value interface{}) *bool {
	switch v := value.(type) {
	case bool:
		val := v
		return &val
	case *bool:
		if v == nil {
			return nil
		}
		val := *v
		return &val
	case string:
		trimmed := strings.TrimSpace(strings.ToLower(v))
		if trimmed == "" {
			return nil
		}
		if trimmed == "true" || trimmed == "1" {
			val := true
			return &val
		}
		if trimmed == "false" || trimmed == "0" {
			val := false
			return &val
		}
	default:
		return nil
	}
	return nil
}

func normalizeReferenceImages(image interface{}) (interface{}, error) {
	switch v := image.(type) {
	case string:
		trimmed := strings.TrimSpace(v)
		if trimmed == "" {
			return nil, fmt.Errorf("image is required")
		}
		return trimmed, nil
	case *string:
		if v == nil {
			return nil, fmt.Errorf("image is required")
		}
		return normalizeReferenceImages(*v)
	case []string:
		return collapseReferenceSlice(v)
	case *[]string:
		if v == nil {
			return nil, fmt.Errorf("image is required")
		}
		return normalizeReferenceImages(*v)
	case arkmodel.Image:
		return v, nil
	case *arkmodel.Image:
		if v == nil {
			return nil, fmt.Errorf("image is required")
		}
		return v, nil
	case []arkmodel.Image:
		if len(v) == 0 {
			return nil, fmt.Errorf("image is required")
		}
		out := make([]interface{}, 0, len(v))
		for _, img := range v {
			out = append(out, img)
		}
		return out, nil
	case []*arkmodel.Image:
		if len(v) == 0 {
			return nil, fmt.Errorf("image is required")
		}
		out := make([]interface{}, 0, len(v))
		for _, img := range v {
			if img != nil {
				out = append(out, img)
			}
		}
		if len(out) == 0 {
			return nil, fmt.Errorf("image is required")
		}
		return out, nil
	case []interface{}:
		result := make([]interface{}, 0, len(v))
		for _, entry := range v {
			if entry == nil {
				continue
			}
			switch val := entry.(type) {
			case string:
				trimmed := strings.TrimSpace(val)
				if trimmed == "" {
					continue
				}
				result = append(result, trimmed)
			case *string:
				if val == nil {
					continue
				}
				trimmed := strings.TrimSpace(*val)
				if trimmed == "" {
					continue
				}
				copyValue := trimmed
				result = append(result, &copyValue)
			case arkmodel.Image:
				result = append(result, val)
			case *arkmodel.Image:
				if val != nil {
					result = append(result, val)
				}
			default:
				return nil, fmt.Errorf("unsupported image reference type %T", entry)
			}
		}
		if len(result) == 0 {
			return nil, fmt.Errorf("image is required")
		}
		return result, nil
	default:
		return nil, fmt.Errorf("unsupported image type %T", image)
	}
}

func collapseReferenceSlice(values []string) (interface{}, error) {
	cleaned := cleanStrings(values)
	if len(cleaned) == 0 {
		return nil, fmt.Errorf("image is required")
	}
	if len(cleaned) == 1 {
		return cleaned[0], nil
	}
	return cleaned, nil
}

func cleanStrings(values []string) []string {
	var cleaned []string
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			cleaned = append(cleaned, trimmed)
		}
	}
	return cleaned
}
