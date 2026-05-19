package huoshan

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

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

// Seedream 5.0（doubao-seedream-5-0-260128）六种生成形态，与火山方舟文档一一对应：
// 文生图单张 / 文生图组图 / 单图参考单张 / 单图参考组图 / 多图参考单张 / 多图参考组图。
const (
	Seedream50ModeTextSingle       = "seedream50_text_single"
	Seedream50ModeTextSet          = "seedream50_text_set"
	Seedream50ModeImageSingleSingle = "seedream50_i2i_1_single"
	Seedream50ModeImageSingleSet    = "seedream50_i2i_1_set"
	Seedream50ModeImageMultiSingle  = "seedream50_i2i_n_single"
	Seedream50ModeImageMultiSet     = "seedream50_i2i_n_set"
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

	req, useStream, err := c.prepareImageRequest(payload, requireImage)
	if err != nil {
		return nil, err
	}
	if useStream {
		return c.generateImageStreaming(ctx, req)
	}

	resp, err := c.arkClient.GenerateImages(ctx, req)
	if err != nil {
		return nil, err
	}

	return toImageGenerationResponse(resp), nil
}

func (c *Client) prepareImageRequest(payload *ImageGenerationRequest, requireImage bool) (arkmodel.GenerateImagesRequest, bool, error) {
	if payload == nil {
		return arkmodel.GenerateImagesRequest{}, false, fmt.Errorf("payload cannot be nil")
	}
	modelName := strings.TrimSpace(payload.Model)
	if modelName == "" {
		modelName = strings.TrimSpace(choose(c.config.ImageModel, DefaultHuoshanImageModelID))
	}
	if isSeedream50Model(modelName) {
		return c.prepareSeedream50ImageRequest(payload, requireImage, modelName)
	}
	req, err := c.prepareLegacyImageRequest(payload, requireImage)
	return req, false, err
}

func isSeedream50Model(model string) bool {
	return strings.TrimSpace(model) == defaultImageModelLevel2
}

func (c *Client) prepareLegacyImageRequest(payload *ImageGenerationRequest, requireImage bool) (arkmodel.GenerateImagesRequest, error) {
	if payload == nil {
		return arkmodel.GenerateImagesRequest{}, fmt.Errorf("payload cannot be nil")
	}

	prompt := strings.TrimSpace(payload.Prompt)
	if prompt == "" {
		return arkmodel.GenerateImagesRequest{}, fmt.Errorf("prompt is required")
	}

	modelName := strings.TrimSpace(payload.Model)
	if modelName == "" {
		modelName = strings.TrimSpace(choose(c.config.ImageModel, DefaultHuoshanImageModelID))
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

func (c *Client) generateImageStreaming(ctx context.Context, req arkmodel.GenerateImagesRequest) (*ImageGenerationResponse, error) {
	stream, err := c.arkClient.GenerateImagesStreaming(ctx, req)
	if err != nil {
		return nil, err
	}
	defer stream.Close()

	var data []ImageGenerationData
	var usage *ImageGenerationUsage
	var streamModel string

	for {
		recv, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		if recv.Model != "" {
			streamModel = recv.Model
		}
		switch recv.Type {
		case arkmodel.ImageGenerationStreamEventPartialFailed:
			if recv.Error != nil && strings.EqualFold(recv.Error.Code, "InternalServiceError") {
				return nil, fmt.Errorf("image generation stream failed: %s", recv.Error.Message)
			}
		case arkmodel.ImageGenerationStreamEventPartialSucceeded:
			if recv.Error == nil && recv.Url != nil && strings.TrimSpace(*recv.Url) != "" {
				data = append(data, ImageGenerationData{URL: *recv.Url, Size: recv.Size})
			}
		case arkmodel.ImageGenerationStreamEventCompleted:
			if recv.Usage != nil {
				usage = &ImageGenerationUsage{
					GeneratedImages: int(recv.Usage.GeneratedImages),
					OutputTokens:    int(recv.Usage.OutputTokens),
					TotalTokens:     int(recv.Usage.TotalTokens),
				}
			}
		}
	}
	if len(data) == 0 {
		return nil, fmt.Errorf("streaming image generation returned no images")
	}
	outModel := streamModel
	if outModel == "" {
		outModel = req.Model
	}
	return &ImageGenerationResponse{
		Model:   outModel,
		Created: time.Now().Unix(),
		Data:    data,
		Usage:   usage,
	}, nil
}

func (c *Client) prepareSeedream50ImageRequest(payload *ImageGenerationRequest, requireImage bool, model string) (arkmodel.GenerateImagesRequest, bool, error) {
	prompt := strings.TrimSpace(payload.Prompt)
	if prompt == "" {
		return arkmodel.GenerateImagesRequest{}, false, fmt.Errorf("prompt is required")
	}

	effectiveMode := normalizeSeedream50EffectiveMode(strings.TrimSpace(payload.Mode), payload, requireImage)
	if effectiveMode == ImageModeWebSearchToImage {
		req, err := c.prepareLegacyImageRequest(payload, requireImage)
		if err != nil {
			return arkmodel.GenerateImagesRequest{}, false, err
		}
		req.Model = model
		return req, false, nil
	}

	refURLs, err := extractRefStringsForSeedream(payload.Image)
	if err != nil {
		return arkmodel.GenerateImagesRequest{}, false, err
	}

	responseFormat := strings.TrimSpace(payload.ResponseFormat)
	if responseFormat == "" {
		responseFormat = arkmodel.GenerateImagesResponseFormatURL
	}

	buildBase := func() arkmodel.GenerateImagesRequest {
		r := arkmodel.GenerateImagesRequest{
			Model:          model,
			Prompt:         prompt,
			ResponseFormat: ptr(responseFormat),
		}
		if s := strings.TrimSpace(payload.Size); s != "" {
			r.Size = ptr(s)
		}
		if payload.Seed != 0 {
			sd := payload.Seed
			r.Seed = &sd
		}
		if payload.GuidanceScale > 0 {
			g := payload.GuidanceScale
			r.GuidanceScale = &g
		}
		if payload.Watermark != nil {
			r.Watermark = payload.Watermark
		}
		if payload.WebSearch {
			e := true
			r.OptimizePrompt = &e
		}
		if len(payload.OptimizePromptOptions) > 0 {
			if o := toBoolPointer(payload.OptimizePromptOptions["enable"]); o != nil {
				r.OptimizePrompt = o
			}
		}
		return r
	}

	seqAuto := arkmodel.SequentialImageGeneration(arkmodel.SequentialImageGenerationAuto)
	seqDisabled := arkmodel.SequentialImageGeneration(arkmodel.SequentialImageGenerationDisabled)
	maxOpts := func() *arkmodel.SequentialImageGenerationOptions {
		n := seedream50MaxImages(payload, 4)
		return &arkmodel.SequentialImageGenerationOptions{MaxImages: &n}
	}

	switch effectiveMode {
	case Seedream50ModeTextSingle:
		if len(refURLs) > 0 {
			return arkmodel.GenerateImagesRequest{}, false, fmt.Errorf("mode %s does not accept reference images", effectiveMode)
		}
		r := buildBase()
		r.SequentialImageGeneration = &seqDisabled
		return r, false, nil

	case Seedream50ModeTextSet:
		if len(refURLs) > 0 {
			return arkmodel.GenerateImagesRequest{}, false, fmt.Errorf("mode %s does not accept reference images", effectiveMode)
		}
		r := buildBase()
		r.SequentialImageGeneration = &seqAuto
		r.SequentialImageGenerationOptions = maxOpts()
		return r, true, nil

	case Seedream50ModeImageSingleSingle:
		if len(refURLs) != 1 {
			return arkmodel.GenerateImagesRequest{}, false, fmt.Errorf("mode %s requires exactly one reference image", effectiveMode)
		}
		r := buildBase()
		r.SequentialImageGeneration = &seqDisabled
		r.Image = refURLs[0]
		return r, false, nil

	case Seedream50ModeImageSingleSet:
		if len(refURLs) != 1 {
			return arkmodel.GenerateImagesRequest{}, false, fmt.Errorf("mode %s requires exactly one reference image", effectiveMode)
		}
		r := buildBase()
		r.Image = refURLs[0]
		r.SequentialImageGeneration = &seqAuto
		r.SequentialImageGenerationOptions = maxOpts()
		return r, true, nil

	case Seedream50ModeImageMultiSingle:
		if len(refURLs) < 2 {
			return arkmodel.GenerateImagesRequest{}, false, fmt.Errorf("mode %s requires at least two reference images", effectiveMode)
		}
		r := buildBase()
		r.SequentialImageGeneration = &seqDisabled
		r.Image = refURLs
		return r, false, nil

	case Seedream50ModeImageMultiSet:
		if len(refURLs) < 2 {
			return arkmodel.GenerateImagesRequest{}, false, fmt.Errorf("mode %s requires at least two reference images", effectiveMode)
		}
		r := buildBase()
		r.Image = refURLs
		r.SequentialImageGeneration = &seqAuto
		r.SequentialImageGenerationOptions = maxOpts()
		return r, true, nil

	default:
		return arkmodel.GenerateImagesRequest{}, false, fmt.Errorf("unsupported seedream 5.0 image mode %q", effectiveMode)
	}
}

func normalizeSeedream50EffectiveMode(mode string, payload *ImageGenerationRequest, requireImage bool) string {
	mode = strings.TrimSpace(mode)
	if mode == "" {
		return inferSeedream50Mode(payload, requireImage)
	}
	switch mode {
	case ImageModeTextToImage:
		return Seedream50ModeTextSingle
	case ImageModeTextToImageSet:
		return Seedream50ModeTextSet
	case ImageModeImageTextToImage:
		return Seedream50ModeImageSingleSingle
	case ImageModeSingleToImageSet:
		return Seedream50ModeImageSingleSet
	case ImageModeMultiImageFusion:
		return Seedream50ModeImageMultiSingle
	case ImageModeMultiRefToImageSet:
		return Seedream50ModeImageMultiSet
	case ImageModeWebSearchToImage:
		return ImageModeWebSearchToImage
	default:
		return mode
	}
}

func inferSeedream50Mode(payload *ImageGenerationRequest, requireImage bool) string {
	_ = requireImage
	if payload.WebSearch {
		return ImageModeWebSearchToImage
	}
	urls, _ := extractRefStringsForSeedream(payload.Image)
	seq := strings.TrimSpace(payload.SequentialImageGeneration)
	maxFromOpts := 0
	if payload.SequentialImageGenerationOptions != nil {
		if n := toIntPointer(payload.SequentialImageGenerationOptions["max_images"]); n != nil {
			maxFromOpts = *n
		} else if n := toIntPointer(payload.SequentialImageGenerationOptions["maxImages"]); n != nil {
			maxFromOpts = *n
		}
	}
	seqLow := strings.ToLower(seq)
	var sequentialLikely bool
	switch {
	case seqLow == "disabled" || seqLow == "off":
		sequentialLikely = false
	case seqLow == "auto" || seqLow == "enabled" || seqLow == "on":
		sequentialLikely = true
	default:
		sequentialLikely = payload.MaxImages > 1 || maxFromOpts > 1
	}

	switch len(urls) {
	case 0:
		if sequentialLikely {
			return Seedream50ModeTextSet
		}
		return Seedream50ModeTextSingle
	case 1:
		if sequentialLikely {
			return Seedream50ModeImageSingleSet
		}
		return Seedream50ModeImageSingleSingle
	default:
		if sequentialLikely {
			return Seedream50ModeImageMultiSet
		}
		return Seedream50ModeImageMultiSingle
	}
}

func seedream50MaxImages(payload *ImageGenerationRequest, fallback int) int {
	if payload.MaxImages > 0 {
		return payload.MaxImages
	}
	if payload.SequentialImageGenerationOptions == nil {
		return fallback
	}
	if n := toIntPointer(payload.SequentialImageGenerationOptions["max_images"]); n != nil {
		return *n
	}
	if n := toIntPointer(payload.SequentialImageGenerationOptions["maxImages"]); n != nil {
		return *n
	}
	return fallback
}

func extractRefStringsForSeedream(image interface{}) ([]string, error) {
	if image == nil {
		return nil, nil
	}
	norm, err := normalizeReferenceImages(image)
	if err != nil {
		return nil, err
	}
	switch v := norm.(type) {
	case string:
		s := strings.TrimSpace(v)
		if s == "" {
			return nil, nil
		}
		return []string{s}, nil
	case []string:
		return cleanStrings(v), nil
	case []interface{}:
		var out []string
		for _, e := range v {
			switch t := e.(type) {
			case string:
				if x := strings.TrimSpace(t); x != "" {
					out = append(out, x)
				}
			case *string:
				if t != nil {
					if x := strings.TrimSpace(*t); x != "" {
						out = append(out, x)
					}
				}
			}
		}
		return out, nil
	default:
		return nil, fmt.Errorf("unsupported normalized reference type %T", norm)
	}
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
