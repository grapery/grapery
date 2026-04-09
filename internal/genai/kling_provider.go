package genapi

import (
	"context"
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"
	"time"

	kling "github.com/grapestree/fgrapery/grapery/internal/genai/providers/kling"
)

// Kling capability values (GenerateRequest.Options["kling_capability"]).
const (
	KlingCapabilityOmniImage        = "omni_image"
	KlingCapabilityOmniVideo        = "omni_video"
	KlingCapabilityVideoExtension   = "video_extension"
	KlingCapabilityImageExpansion   = "image_expansion"
	KlingCapabilityMultiImage2Image = "multi_image2image"
	KlingCapabilityMultiImage2Video = "multi_image2video"
	KlingCapabilityMultiElements    = "multi_elements"
)

type klingProvider struct {
	name   string
	client *kling.Client
}

func newKlingProvider(cfg *Config) (*klingProvider, error) {
	if cfg == nil {
		return nil, fmt.Errorf("kling configuration is required")
	}
	access := strings.TrimSpace(cfg.APIKey)
	secret := strings.TrimSpace(cfg.Secret)
	if access == "" || secret == "" {
		return nil, fmt.Errorf("kling access key and secret key are required (APIKey + Secret)")
	}
	base := strings.TrimSpace(cfg.BaseURL)
	cli, err := kling.New(kling.Config{
		AccessKey: access,
		SecretKey: secret,
		BaseURL:   base,
		Timeout:   cfg.Timeout,
	})
	if err != nil {
		return nil, err
	}
	name := strings.TrimSpace(string(cfg.Provider))
	if name == "" {
		name = string(ProviderKling)
	}
	return &klingProvider{name: name, client: cli}, nil
}

func (p *klingProvider) Name() string {
	return p.name
}

func (p *klingProvider) Generate(ctx context.Context, req *GenerateRequest) (*GenerateResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("generate request cannot be nil")
	}
	cap := strings.TrimSpace(stringFromOptions(req.Options, "kling_capability", "klingCapability"))
	switch cap {
	case KlingCapabilityOmniVideo, KlingCapabilityMultiElements:
		return p.omniVideo(ctx, req)
	case KlingCapabilityVideoExtension:
		return p.videoExtend(ctx, req)
	case KlingCapabilityMultiImage2Video:
		return p.multiImage2Video(ctx, req)
	}

	switch req.Operation {
	case OperationTextToVideo:
		return p.text2video(ctx, req)
	case OperationImageToVideo:
		if n := len(collectImages(req.ReferenceImageURL, req.ReferenceImages, 0)); n > 1 {
			return p.multiImage2Video(ctx, req)
		}
		return p.image2video(ctx, req)
	case OperationKeyframeToVideo:
		return p.image2video(ctx, req)
	default:
		return nil, fmt.Errorf("kling: unsupported video operation %s (set Options[\"kling_capability\"] for omni/extension/multi-image)", req.Operation)
	}
}

func (p *klingProvider) GenerateImage(ctx context.Context, req *GenerateRequest) (*GenerateResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("generate request cannot be nil")
	}
	cap := strings.TrimSpace(stringFromOptions(req.Options, "kling_capability", "klingCapability"))
	switch cap {
	case KlingCapabilityOmniImage, KlingCapabilityMultiElements:
		return p.omniImage(ctx, req)
	case KlingCapabilityImageExpansion:
		return p.imageExpand(ctx, req)
	case KlingCapabilityMultiImage2Image:
		return p.multiImage2Image(ctx, req)
	}

	switch req.Operation {
	case OperationImageToImage:
		if n := len(collectImages(req.ReferenceImageURL, req.ReferenceImages, 0)); n > 1 {
			return p.multiImage2Image(ctx, req)
		}
		return p.imageGeneration(ctx, req)
	case OperationTextToImage:
		return p.imageGeneration(ctx, req)
	default:
		return nil, fmt.Errorf("kling: unsupported image operation %s", req.Operation)
	}
}

func (p *klingProvider) GetVideoStatus(ctx context.Context, taskID string) (*GenerateResponse, error) {
	return p.queryTask(ctx, taskID, MediaTypeVideo)
}

func (p *klingProvider) GetImageStatus(ctx context.Context, taskID string) (*GenerateResponse, error) {
	return p.queryTask(ctx, taskID, MediaTypeImage)
}

func (p *klingProvider) DownloadVideo(ctx context.Context, taskID string) ([]byte, error) {
	rsp, err := p.GetVideoStatus(ctx, taskID)
	if err != nil {
		return nil, err
	}
	if rsp.VideoURL == "" {
		return nil, fmt.Errorf("kling: no video url for task")
	}
	return p.client.DownloadURL(ctx, rsp.VideoURL)
}

func (p *klingProvider) text2video(ctx context.Context, req *GenerateRequest) (*GenerateResponse, error) {
	prompt := strings.TrimSpace(req.Prompt)
	if prompt == "" {
		return nil, fmt.Errorf("kling text2video: prompt is required")
	}
	payload := map[string]interface{}{
		"model_name": firstNonEmpty(strings.TrimSpace(req.Model), "kling-v1"),
		"prompt":     prompt,
	}
	if np := strings.TrimSpace(req.NegativePrompt); np != "" {
		payload["negative_prompt"] = np
	}
	if ar := strings.TrimSpace(req.AspectRatio); ar != "" {
		payload["aspect_ratio"] = ar
	}
	payload["duration"] = klingDuration(req.DurationSeconds)
	if m := strings.TrimSpace(stringFromOptions(req.Options, "mode", "kling_mode")); m != "" {
		payload["mode"] = m
	} else if strings.TrimSpace(req.Quality) != "" {
		payload["mode"] = strings.ToLower(strings.TrimSpace(req.Quality))
	}
	mergeKlingFlatOptions(payload, req.Options, []string{
		"sound", "cfg_scale", "camera_control", "callback_url", "external_task_id",
	})
	mergeKlingNested(req.Options, payload)
	applyKlingCallbackURL(payload, req)

	data, err := p.client.CreateText2Video(ctx, payload)
	if err != nil {
		return nil, err
	}
	if err := kling.ValidateCreateTask("text2video", data); err != nil {
		return nil, err
	}
	return p.taskCreateResponse(req, kling.TaskText2Video, data, klingString(payload["model_name"])), nil
}

func (p *klingProvider) image2video(ctx context.Context, req *GenerateRequest) (*GenerateResponse, error) {
	img, tail, err := p.resolveKeyframeImages(req)
	if err != nil {
		return nil, err
	}
	payload := map[string]interface{}{
		"model_name": firstNonEmpty(strings.TrimSpace(req.Model), "kling-v1"),
		"image":      img,
	}
	if tail != "" {
		payload["image_tail"] = tail
	}
	if pr := strings.TrimSpace(req.Prompt); pr != "" {
		payload["prompt"] = pr
	}
	if np := strings.TrimSpace(req.NegativePrompt); np != "" {
		payload["negative_prompt"] = np
	}
	payload["duration"] = klingDuration(req.DurationSeconds)
	if ar := strings.TrimSpace(req.AspectRatio); ar != "" {
		payload["aspect_ratio"] = ar
	}
	if m := strings.TrimSpace(stringFromOptions(req.Options, "mode", "kling_mode")); m != "" {
		payload["mode"] = m
	}
	mergeKlingFlatOptions(payload, req.Options, []string{
		"voice_list", "dynamic_masks", "static_mask", "cfg_scale", "camera_control",
		"callback_url", "external_task_id", "sound",
	})
	mergeKlingNested(req.Options, payload)
	applyKlingCallbackURL(payload, req)

	data, err := p.client.CreateImage2Video(ctx, payload)
	if err != nil {
		return nil, err
	}
	if err := kling.ValidateCreateTask("image2video", data); err != nil {
		return nil, err
	}
	return p.taskCreateResponse(req, kling.TaskImage2Video, data, klingString(payload["model_name"])), nil
}

func (p *klingProvider) multiImage2Video(ctx context.Context, req *GenerateRequest) (*GenerateResponse, error) {
	refs := collectImages(req.ReferenceImageURL, req.ReferenceImages, 0)
	if len(refs) == 0 {
		return nil, fmt.Errorf("kling multi-image2video: at least one reference image is required")
	}
	list := make([]map[string]interface{}, 0, len(refs))
	for _, u := range refs {
		list = append(list, map[string]interface{}{"image": stripDataURI(u)})
	}
	payload := map[string]interface{}{
		"model_name": firstNonEmpty(strings.TrimSpace(req.Model), "kling-v1-6"),
		"image_list": list,
		"prompt":     strings.TrimSpace(req.Prompt),
	}
	if payload["prompt"] == "" {
		return nil, fmt.Errorf("kling multi-image2video: prompt is required")
	}
	if np := strings.TrimSpace(req.NegativePrompt); np != "" {
		payload["negative_prompt"] = np
	}
	payload["duration"] = klingDuration(req.DurationSeconds)
	if ar := strings.TrimSpace(req.AspectRatio); ar != "" {
		payload["aspect_ratio"] = ar
	}
	if m := strings.TrimSpace(stringFromOptions(req.Options, "mode", "kling_mode")); m != "" {
		payload["mode"] = m
	}
	mergeKlingFlatOptions(payload, req.Options, []string{"callback_url", "external_task_id"})
	mergeKlingNested(req.Options, payload)
	applyKlingCallbackURL(payload, req)

	data, err := p.client.CreateMultiImage2Video(ctx, payload)
	if err != nil {
		return nil, err
	}
	if err := kling.ValidateCreateTask("multi-image2video", data); err != nil {
		return nil, err
	}
	return p.taskCreateResponse(req, kling.TaskMultiImage2Video, data, klingString(payload["model_name"])), nil
}

func (p *klingProvider) videoExtend(ctx context.Context, req *GenerateRequest) (*GenerateResponse, error) {
	vid := strings.TrimSpace(stringFromOptions(req.Options, "kling_video_id", "video_id"))
	if vid == "" && req.Metadata != nil {
		if s, ok := req.Metadata["kling_video_id"].(string); ok {
			vid = strings.TrimSpace(s)
		}
	}
	if vid == "" {
		return nil, fmt.Errorf("kling video extension: set Options[\"kling_video_id\"] to source video_id")
	}
	payload := map[string]interface{}{"video_id": vid}
	if pr := strings.TrimSpace(req.Prompt); pr != "" {
		payload["prompt"] = pr
	}
	if np := strings.TrimSpace(req.NegativePrompt); np != "" {
		payload["negative_prompt"] = np
	}
	mergeKlingFlatOptions(payload, req.Options, []string{"cfg_scale", "callback_url"})
	mergeKlingNested(req.Options, payload)
	applyKlingCallbackURL(payload, req)

	data, err := p.client.CreateVideoExtend(ctx, payload)
	if err != nil {
		return nil, err
	}
	if err := kling.ValidateCreateTask("video-extend", data); err != nil {
		return nil, err
	}
	r := p.taskCreateResponse(req, kling.TaskVideoExtend, data, strings.TrimSpace(req.Model))
	if r.Metadata == nil {
		r.Metadata = map[string]interface{}{}
	}
	r.Metadata["kling_video_id"] = vid
	return r, nil
}

func (p *klingProvider) omniVideo(ctx context.Context, req *GenerateRequest) (*GenerateResponse, error) {
	prompt := strings.TrimSpace(req.Prompt)
	if prompt == "" {
		return nil, fmt.Errorf("kling omni-video: prompt is required")
	}
	payload := map[string]interface{}{
		"model_name": firstNonEmpty(strings.TrimSpace(req.Model), "kling-video-o1"),
		"prompt":     prompt,
	}
	payload["duration"] = klingDuration(req.DurationSeconds)
	if ar := strings.TrimSpace(req.AspectRatio); ar != "" {
		payload["aspect_ratio"] = ar
	}
	mergeKlingFlatOptions(payload, req.Options, []string{
		"callback_url", "external_task_id", "video_list", "element_list", "image_list",
	})
	mergeKlingNested(req.Options, payload)
	applyKlingCallbackURL(payload, req)
	// Allow image URLs from ReferenceImages -> image_list if not already set
	if _, has := payload["image_list"]; !has {
		refs := collectImages(req.ReferenceImageURL, req.ReferenceImages, 0)
		if len(refs) > 0 {
			il := make([]map[string]interface{}, 0, len(refs))
			for _, u := range refs {
				il = append(il, map[string]interface{}{"image_url": stripDataURI(u)})
			}
			payload["image_list"] = il
		}
	}

	data, err := p.client.CreateOmniVideo(ctx, payload)
	if err != nil {
		return nil, err
	}
	if err := kling.ValidateCreateTask("omni-video", data); err != nil {
		return nil, err
	}
	return p.taskCreateResponse(req, kling.TaskOmniVideo, data, klingString(payload["model_name"])), nil
}

func (p *klingProvider) imageGeneration(ctx context.Context, req *GenerateRequest) (*GenerateResponse, error) {
	prompt := strings.TrimSpace(req.Prompt)
	if prompt == "" {
		return nil, fmt.Errorf("kling image generation: prompt is required")
	}
	payload := map[string]interface{}{
		"model_name": firstNonEmpty(strings.TrimSpace(req.Model), "kling-v1"),
		"prompt":     prompt,
	}
	if np := strings.TrimSpace(req.NegativePrompt); np != "" {
		payload["negative_prompt"] = np
	}
	if ar := strings.TrimSpace(req.AspectRatio); ar != "" {
		payload["aspect_ratio"] = ar
	}
	if req.OutputCount > 0 {
		payload["n"] = req.OutputCount
	}
	if res := strings.TrimSpace(req.Resolution); res != "" {
		payload["resolution"] = strings.ToLower(res)
	}
	refs := collectImages(req.ReferenceImageURL, req.ReferenceImages, 0)
	if len(refs) > 0 {
		payload["image"] = stripDataURI(refs[0])
	}
	if t := strings.TrimSpace(stringFromOptions(req.Options, "image_reference", "imageReference")); t != "" {
		payload["image_reference"] = t
	}
	mergeKlingFlatOptions(payload, req.Options, []string{
		"image_fidelity", "human_fidelity", "callback_url", "external_task_id",
	})
	mergeKlingNested(req.Options, payload)
	applyKlingCallbackURL(payload, req)

	data, err := p.client.CreateImageGeneration(ctx, payload)
	if err != nil {
		return nil, err
	}
	if err := kling.ValidateCreateTask("image-generations", data); err != nil {
		return nil, err
	}
	return p.taskCreateResponse(req, kling.TaskImageGen, data, klingString(payload["model_name"])), nil
}

func (p *klingProvider) imageExpand(ctx context.Context, req *GenerateRequest) (*GenerateResponse, error) {
	img, err := p.firstImageSource(req)
	if err != nil {
		return nil, err
	}
	payload := map[string]interface{}{"image": stripDataURI(img)}
	mergeKlingNested(req.Options, payload)
	if pr := strings.TrimSpace(req.Prompt); pr != "" {
		payload["prompt"] = pr
	}
	applyExpansionRatios(payload, req.Options)
	applyExpansionRatiosFromPayload(payload)
	mergeKlingFlatOptions(payload, req.Options, []string{"callback_url", "external_task_id"})
	applyKlingCallbackURL(payload, req)

	data, err := p.client.CreateImageExpand(ctx, payload)
	if err != nil {
		return nil, err
	}
	if err := kling.ValidateCreateTask("image-expand", data); err != nil {
		return nil, err
	}
	return p.taskCreateResponse(req, kling.TaskImageExpand, data, ""), nil
}

func (p *klingProvider) omniImage(ctx context.Context, req *GenerateRequest) (*GenerateResponse, error) {
	prompt := strings.TrimSpace(req.Prompt)
	if prompt == "" {
		return nil, fmt.Errorf("kling omni-image: prompt is required")
	}
	payload := map[string]interface{}{
		"model_name": firstNonEmpty(strings.TrimSpace(req.Model), "kling-image-o1"),
		"prompt":     prompt,
	}
	if ar := strings.TrimSpace(req.AspectRatio); ar != "" {
		payload["aspect_ratio"] = ar
	}
	if req.OutputCount > 0 {
		payload["n"] = req.OutputCount
	}
	if res := strings.TrimSpace(req.Resolution); res != "" {
		payload["resolution"] = strings.ToLower(res)
	}
	mergeKlingFlatOptions(payload, req.Options, []string{
		"callback_url", "external_task_id", "element_list", "image_list",
	})
	mergeKlingNested(req.Options, payload)
	applyKlingCallbackURL(payload, req)
	if _, has := payload["image_list"]; !has {
		refs := collectImages(req.ReferenceImageURL, req.ReferenceImages, 0)
		if len(refs) > 0 {
			il := make([]map[string]interface{}, 0, len(refs))
			for _, u := range refs {
				il = append(il, map[string]interface{}{"image": stripDataURI(u)})
			}
			payload["image_list"] = il
		}
	}

	data, err := p.client.CreateOmniImage(ctx, payload)
	if err != nil {
		return nil, err
	}
	if err := kling.ValidateCreateTask("omni-image", data); err != nil {
		return nil, err
	}
	return p.taskCreateResponse(req, kling.TaskOmniImage, data, klingString(payload["model_name"])), nil
}

func (p *klingProvider) multiImage2Image(ctx context.Context, req *GenerateRequest) (*GenerateResponse, error) {
	refs := collectImages(req.ReferenceImageURL, req.ReferenceImages, 0)
	if len(refs) == 0 {
		return nil, fmt.Errorf("kling multi-image2image: subject images required")
	}
	subs := make([]map[string]interface{}, 0, len(refs))
	for _, u := range refs {
		subs = append(subs, map[string]interface{}{"subject_image": stripDataURI(u)})
	}
	payload := map[string]interface{}{
		"model_name":          firstNonEmpty(strings.TrimSpace(req.Model), "kling-v2"),
		"subject_image_list":  subs,
	}
	if pr := strings.TrimSpace(req.Prompt); pr != "" {
		payload["prompt"] = pr
	}
	if ar := strings.TrimSpace(req.AspectRatio); ar != "" {
		payload["aspect_ratio"] = ar
	}
	if req.OutputCount > 0 {
		payload["n"] = req.OutputCount
	}
	if si := strings.TrimSpace(stringFromOptions(req.Options, "kling_scene_image", "scene_image")); si != "" {
		payload["scene_image"] = stripDataURI(si)
	}
	if st := strings.TrimSpace(stringFromOptions(req.Options, "kling_style_image", "style_image")); st != "" {
		payload["style_image"] = stripDataURI(st)
	}
	mergeKlingFlatOptions(payload, req.Options, []string{"callback_url", "external_task_id"})
	mergeKlingNested(req.Options, payload)
	applyKlingCallbackURL(payload, req)

	data, err := p.client.CreateMultiImage2Image(ctx, payload)
	if err != nil {
		return nil, err
	}
	if err := kling.ValidateCreateTask("multi-image2image", data); err != nil {
		return nil, err
	}
	return p.taskCreateResponse(req, kling.TaskMultiImage2Image, data, klingString(payload["model_name"])), nil
}

func (p *klingProvider) queryTask(ctx context.Context, composite string, _ MediaType) (*GenerateResponse, error) {
	kind, raw, err := kling.ParseTaskID(composite)
	if err != nil {
		return nil, err
	}
	path, err := kling.QueryPath(kind, raw)
	if err != nil {
		return nil, err
	}
	data, err := p.client.QueryTask(ctx, path)
	if err != nil {
		return nil, err
	}
	media := mediaForKind(kind)
	rsp := p.queryToResponse(kind, media, data)
	rsp.TaskID = composite
	return rsp, nil
}

func (p *klingProvider) taskCreateResponse(req *GenerateRequest, kind string, data *kling.CreateTaskData, model string) *GenerateResponse {
	now := time.Now()
	taskID := ""
	status := string(StatusProcessing)
	if data != nil {
		taskID = kling.FormatTaskID(kind, data.TaskID)
		if strings.TrimSpace(data.TaskStatus) != "" {
			status = NormalizeStatus(data.TaskStatus)
		}
	}
	meta := make(map[string]interface{})
	if m := strings.TrimSpace(model); m != "" {
		meta["model"] = m
	}
	if req != nil && req.Metadata != nil {
		for k, v := range req.Metadata {
			meta[k] = v
		}
	}
	op := OperationUnknown
	if req != nil {
		op = req.Operation
	}
	return &GenerateResponse{
		Provider:    p.name,
		Operation:   op,
		MediaType:   mediaForKind(kind),
		TaskID:      taskID,
		Status:      status,
		Message:     status,
		Metadata:    meta,
		Raw:         map[string]interface{}{"create_task": data},
		StartedAt:   now,
		CompletedAt: now,
	}
}

func mediaForKind(kind string) MediaType {
	switch kind {
	case kling.TaskImageGen, kling.TaskImageExpand, kling.TaskOmniImage, kling.TaskMultiImage2Image:
		return MediaTypeImage
	default:
		return MediaTypeVideo
	}
}

func (p *klingProvider) queryToResponse(kind string, media MediaType, data *kling.QueryTaskData) *GenerateResponse {
	now := time.Now()
	rsp := &GenerateResponse{
		Provider:  p.name,
		MediaType: media,
		Raw:       map[string]interface{}{"query_task": data},
		StartedAt: now, CompletedAt: now,
	}
	if data == nil {
		rsp.Status = string(StatusUnknown)
		return rsp
	}
	if data.TaskID != "" {
		rsp.TaskID = kling.FormatTaskID(kind, data.TaskID)
	} else {
		rsp.TaskID = kling.FormatTaskID(kind, "")
	}
	rsp.Status = NormalizeStatus(data.TaskStatus)
	rsp.Message = firstNonEmpty(strings.TrimSpace(data.TaskStatusMsg), rsp.Status)
	rsp.Metadata = map[string]interface{}{}
	if data.TaskInfo != nil {
		if m := klingString(data.TaskInfo["model_name"]); m != "" {
			rsp.Metadata["model"] = m
		}
		if ext := klingString(data.TaskInfo["external_task_id"]); ext != "" {
			rsp.Metadata["external_task_id"] = ext
		}
	}
	if data.TaskResult != nil {
		for _, v := range data.TaskResult.Videos {
			if u := strings.TrimSpace(v.URL); u != "" {
				rsp.VideoURL = u
				if thumb := firstNonEmpty(strings.TrimSpace(v.CoverURL), strings.TrimSpace(v.Thumbnail)); thumb != "" {
					rsp.ThumbnailURL = thumb
				}
				break
			}
		}
		for _, im := range data.TaskResult.Images {
			if u := strings.TrimSpace(im.URL); u != "" {
				rsp.ImageURLs = append(rsp.ImageURLs, u)
			}
		}
	}
	if rsp.Status == string(StatusCompleted) {
		if media == MediaTypeVideo && rsp.VideoURL != "" {
			rsp.Usage = &Usage{
				VideoCount:   1,
				OutputTokens: 1,
				TotalTokens:  1,
				Additional:   map[string]interface{}{"kling": "video_succeed"},
			}
		}
		if media == MediaTypeImage && len(rsp.ImageURLs) > 0 {
			n := len(rsp.ImageURLs)
			rsp.Usage = &Usage{
				ImageCount:   n,
				OutputTokens: n,
				TotalTokens:  n,
				Additional:   map[string]interface{}{"kling": "image_succeed"},
			}
		}
	}
	if rsp.Status == string(StatusFailed) {
		rsp.Error = rsp.Message
	}
	return rsp
}

func (p *klingProvider) resolveKeyframeImages(req *GenerateRequest) (image string, imageTail string, err error) {
	first := strings.TrimSpace(req.FirstFrameURL)
	last := strings.TrimSpace(req.LastFrameURL)
	if first == "" && len(req.FirstFrameData) > 0 {
		first = base64.StdEncoding.EncodeToString(req.FirstFrameData)
	}
	if last == "" && len(req.LastFrameData) > 0 {
		last = base64.StdEncoding.EncodeToString(req.LastFrameData)
	}
	if first == "" {
		refs := collectImages(req.ReferenceImageURL, req.ReferenceImages, 0)
		if len(refs) > 0 {
			first = refs[0]
		}
	}
	if first == "" {
		return "", "", fmt.Errorf("kling image2video: image (start frame) is required")
	}
	if req.Operation == OperationKeyframeToVideo {
		if last == "" {
			return "", "", fmt.Errorf("kling keyframe: last frame (image_tail) is required")
		}
		return stripDataURI(first), stripDataURI(last), nil
	}
	return stripDataURI(first), stripDataURI(last), nil
}

func (p *klingProvider) firstImageSource(req *GenerateRequest) (string, error) {
	refs := collectImages(req.ReferenceImageURL, req.ReferenceImages, 0)
	if len(refs) > 0 {
		return refs[0], nil
	}
	if len(req.ImageData) > 0 {
		return base64.StdEncoding.EncodeToString(req.ImageData), nil
	}
	return "", fmt.Errorf("kling: reference image or ImageData is required")
}

// klingDuration maps to API-allowed values "5" or "10" only.
func klingDuration(seconds int) string {
	if seconds <= 0 {
		return "5"
	}
	if seconds >= 10 {
		return "10"
	}
	if seconds >= 8 {
		return "10"
	}
	return "5"
}

func stripDataURI(s string) string {
	s = strings.TrimSpace(s)
	const prefix = "data:"
	const mid = "base64,"
	if strings.HasPrefix(s, prefix) {
		if i := strings.Index(s, mid); i >= 0 {
			return strings.TrimSpace(s[i+len(mid):])
		}
	}
	return s
}

func klingString(v interface{}) string {
	if v == nil {
		return ""
	}
	switch t := v.(type) {
	case string:
		return strings.TrimSpace(t)
	case float64:
		if t == float64(int64(t)) {
			return strconv.FormatInt(int64(t), 10)
		}
		return strconv.FormatFloat(t, 'g', -1, 64)
	case int:
		return strconv.Itoa(t)
	case int64:
		return strconv.FormatInt(t, 10)
	default:
		return strings.TrimSpace(fmt.Sprint(t))
	}
}

func applyKlingCallbackURL(payload map[string]interface{}, req *GenerateRequest) {
	if req == nil || payload == nil {
		return
	}
	if _, exists := payload["callback_url"]; exists {
		return
	}
	if cb := strings.TrimSpace(req.CallbackURL); cb != "" {
		payload["callback_url"] = cb
	}
}

func mergeKlingNested(opts map[string]interface{}, payload map[string]interface{}) {
	if opts == nil {
		return
	}
	nested, ok := opts["kling"].(map[string]interface{})
	if !ok || nested == nil {
		return
	}
	for k, v := range nested {
		payload[k] = v
	}
}

func mergeKlingFlatOptions(payload map[string]interface{}, opts map[string]interface{}, keys []string) {
	if opts == nil {
		return
	}
	for _, k := range keys {
		if v, ok := opts[k]; ok {
			payload[k] = v
		}
	}
}

func applyExpansionRatios(payload map[string]interface{}, opts map[string]interface{}) {
	if opts == nil {
		return
	}
	copyExpansionKeys(payload, opts)
	if m, ok := opts["kling_expansion_ratio"].(map[string]interface{}); ok {
		mapExpansionDirections(m, payload)
	}
	if nested, ok := opts["kling"].(map[string]interface{}); ok {
		copyExpansionKeys(payload, nested)
		if m, ok := nested["expansion_ratio"].(map[string]interface{}); ok {
			mapExpansionDirections(m, payload)
		}
		if m, ok := nested["kling_expansion_ratio"].(map[string]interface{}); ok {
			mapExpansionDirections(m, payload)
		}
	}
}

func applyExpansionRatiosFromPayload(payload map[string]interface{}) {
	if m, ok := payload["expansion_ratio"].(map[string]interface{}); ok {
		mapExpansionDirections(m, payload)
	}
}

func copyExpansionKeys(dst, src map[string]interface{}) {
	keys := []struct{ from, to string }{
		{"kling_up_expansion_ratio", "up_expansion_ratio"},
		{"kling_down_expansion_ratio", "down_expansion_ratio"},
		{"kling_left_expansion_ratio", "left_expansion_ratio"},
		{"kling_right_expansion_ratio", "right_expansion_ratio"},
		{"up_expansion_ratio", "up_expansion_ratio"},
		{"down_expansion_ratio", "down_expansion_ratio"},
		{"left_expansion_ratio", "left_expansion_ratio"},
		{"right_expansion_ratio", "right_expansion_ratio"},
	}
	for _, k := range keys {
		if v, ok := src[k.from]; ok {
			if _, exists := dst[k.to]; !exists {
				dst[k.to] = v
			}
		}
	}
}

func mapExpansionDirections(m map[string]interface{}, payload map[string]interface{}) {
	dir := map[string]string{"top": "up_expansion_ratio", "bottom": "down_expansion_ratio", "left": "left_expansion_ratio", "right": "right_expansion_ratio"}
	for side, api := range dir {
		if v, ok := m[side]; ok {
			if _, exists := payload[api]; !exists {
				payload[api] = v
			}
		}
	}
}
