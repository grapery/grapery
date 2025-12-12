package hailuo

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

type TextToVideoRequest struct {
	Prompt           string                  `json:"prompt"`
	Model            string                  `json:"model"`
	Duration         int                     `json:"duration,omitempty"`
	Resolution       string                  `json:"resolution,omitempty"`
	PromptOptimizer  *bool                   `json:"prompt_optimizer,omitempty"`
	FastPretreatment *bool                   `json:"fast_pretreatment,omitempty"`
	AIGCWatermark    *bool                   `json:"aigc_watermark,omitempty"`
	CallbackURL      string                  `json:"callback_url,omitempty"`
	SubjectReference []VideoSubjectReference `json:"subject_reference,omitempty"`
	Metadata         map[string]interface{}  `json:"-"`
	StylePreset      string                  `json:"-"`
}

type ImageToVideoRequest struct {
	FirstFrameImage  string                 `json:"first_frame_image"`
	Prompt           string                 `json:"prompt,omitempty"`
	Model            string                 `json:"model"`
	Duration         int                    `json:"duration,omitempty"`
	Resolution       string                 `json:"resolution,omitempty"`
	PromptOptimizer  *bool                  `json:"prompt_optimizer,omitempty"`
	FastPretreatment *bool                  `json:"fast_pretreatment,omitempty"`
	AIGCWatermark    *bool                  `json:"aigc_watermark,omitempty"`
	CallbackURL      string                 `json:"callback_url,omitempty"`
	Metadata         map[string]interface{} `json:"-"`
}

type FirstLastFrameRequest struct {
	FirstFrameImage  string                 `json:"first_frame_image"`
	LastFrameImage   string                 `json:"last_frame_image"`
	Prompt           string                 `json:"prompt,omitempty"`
	Model            string                 `json:"model"`
	Duration         int                    `json:"duration,omitempty"`
	Resolution       string                 `json:"resolution,omitempty"`
	PromptOptimizer  *bool                  `json:"prompt_optimizer,omitempty"`
	FastPretreatment *bool                  `json:"fast_pretreatment,omitempty"`
	AIGCWatermark    *bool                  `json:"aigc_watermark,omitempty"`
	CallbackURL      string                 `json:"callback_url,omitempty"`
	Metadata         map[string]interface{} `json:"-"`
}

type StoryboardToVideoRequest struct {
	Storyboard  map[string]interface{} `json:"storyboard"`
	Model       string                 `json:"model"`
	CallbackURL string                 `json:"callback_url,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

type VideoSubjectReference struct {
	Type  string   `json:"type"`
	Image []string `json:"image"`
}

type StatusResponse struct {
	TaskID       string          `json:"task_id"`
	Status       string          `json:"status"`
	FileID       string          `json:"file_id"`
	VideoWidth   int             `json:"video_width"`
	VideoHeight  int             `json:"video_height"`
	BaseResponse APIBaseResponse `json:"base_resp"`
}

type FileRetrieveResponse struct {
	File         *FileMetadata   `json:"file"`
	BaseResponse APIBaseResponse `json:"base_resp"`
}

type FileMetadata struct {
	FileID      string `json:"file_id"`
	Bytes       int64  `json:"bytes"`
	CreatedAt   int64  `json:"created_at"`
	Filename    string `json:"filename"`
	Purpose     string `json:"purpose"`
	DownloadURL string `json:"download_url"`
}

func (c *Client) SubmitTextToVideo(ctx context.Context, payload *TextToVideoRequest) (*TaskResponse, error) {
	if payload == nil {
		return nil, fmt.Errorf("payload cannot be nil")
	}
	payload.Prompt = strings.TrimSpace(payload.Prompt)
	if payload.Prompt == "" {
		return nil, fmt.Errorf("prompt is required")
	}

	payload.SubjectReference = normalizeSubjectReferences(payload.SubjectReference)
	withSubjectReference := len(payload.SubjectReference) > 0

	payload.Model = strings.TrimSpace(payload.Model)
	if payload.Model == "" {
		configModel := strings.TrimSpace(c.config.Model)
		if strings.EqualFold(configModel, "hailuo-video-1") {
			configModel = ""
		}
		var defaultModel string
		if withSubjectReference {
			defaultModel = "S2V-01"
			if strings.EqualFold(configModel, "MiniMax-Hailuo-02") {
				configModel = ""
			}
		} else {
			defaultModel = "MiniMax-Hailuo-02"
		}
		payload.Model = strings.TrimSpace(choose(configModel, defaultModel))
		if payload.Model == "" {
			return nil, fmt.Errorf("model is required")
		}
	}

	if payload.Duration <= 0 {
		payload.Duration = defaultVideoGenerationDuration
	}

	payload.Resolution = strings.TrimSpace(payload.Resolution)
	payload.CallbackURL = strings.TrimSpace(payload.CallbackURL)
	return c.post(ctx, videoGenerationEndpoint, payload)
}

func (c *Client) SubmitImageToVideo(ctx context.Context, payload *ImageToVideoRequest) (*TaskResponse, error) {
	if payload == nil {
		return nil, fmt.Errorf("payload cannot be nil")
	}

	payload.FirstFrameImage = strings.TrimSpace(payload.FirstFrameImage)
	if payload.FirstFrameImage == "" {
		return nil, fmt.Errorf("first_frame_image is required")
	}

	payload.Prompt = strings.TrimSpace(payload.Prompt)
	payload.Model = strings.TrimSpace(payload.Model)
	if payload.Model == "" {
		configModel := strings.TrimSpace(c.config.Model)
		if strings.EqualFold(configModel, "hailuo-video-1") {
			configModel = ""
		}
		payload.Model = strings.TrimSpace(choose(configModel, "MiniMax-Hailuo-02"))
		if payload.Model == "" {
			return nil, fmt.Errorf("model is required")
		}
	}

	if payload.Duration <= 0 {
		payload.Duration = defaultVideoGenerationDuration
	}

	payload.Resolution = strings.TrimSpace(payload.Resolution)
	payload.CallbackURL = strings.TrimSpace(payload.CallbackURL)
	return c.post(ctx, videoGenerationEndpoint, payload)
}

func (c *Client) SubmitFirstLastFrame(ctx context.Context, payload *FirstLastFrameRequest) (*TaskResponse, error) {
	if payload == nil {
		return nil, fmt.Errorf("payload cannot be nil")
	}
	payload.FirstFrameImage = strings.TrimSpace(payload.FirstFrameImage)
	if payload.FirstFrameImage == "" {
		return nil, fmt.Errorf("first_frame_image is required")
	}

	payload.LastFrameImage = strings.TrimSpace(payload.LastFrameImage)
	if payload.LastFrameImage == "" {
		return nil, fmt.Errorf("last_frame_image is required")
	}

	payload.Prompt = strings.TrimSpace(payload.Prompt)

	payload.Model = strings.TrimSpace(payload.Model)
	if payload.Model == "" {
		configModel := strings.TrimSpace(c.config.Model)
		if strings.EqualFold(configModel, "hailuo-video-1") {
			configModel = ""
		}
		payload.Model = strings.TrimSpace(choose(configModel, "MiniMax-Hailuo-02"))
		if payload.Model == "" {
			return nil, fmt.Errorf("model is required")
		}
	}

	if payload.Duration <= 0 {
		payload.Duration = defaultVideoGenerationDuration
	}

	payload.Resolution = strings.TrimSpace(payload.Resolution)
	payload.CallbackURL = strings.TrimSpace(payload.CallbackURL)
	return c.post(ctx, videoGenerationEndpoint, payload)
}

func (c *Client) SubmitStoryboard(ctx context.Context, payload *StoryboardToVideoRequest) (*TaskResponse, error) {
	if payload == nil {
		return nil, fmt.Errorf("payload cannot be nil")
	}
	if payload.Model == "" {
		payload.Model = choose(c.config.Model, "hailuo-video-1")
	}
	return c.post(ctx, storyboardToVideoEndpoint, payload)
}

func (c *Client) GetTaskStatus(ctx context.Context, taskID string) (*StatusResponse, error) {
	trimmed := strings.TrimSpace(taskID)
	if trimmed == "" {
		return nil, fmt.Errorf("task_id is required")
	}

	base := c.endpoint(videoStatusQueryEndpoint)
	requestURL := base + "?task_id=" + url.QueryEscape(trimmed)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.config.APIKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("hailuo status query failed with status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	statusResp := &StatusResponse{}
	if err := json.NewDecoder(resp.Body).Decode(statusResp); err != nil {
		return nil, fmt.Errorf("decode hailuo status response: %w", err)
	}
	return statusResp, nil
}

func (c *Client) RetrieveFile(ctx context.Context, fileID string) (*FileRetrieveResponse, error) {
	trimmed := strings.TrimSpace(fileID)
	if trimmed == "" {
		return nil, fmt.Errorf("file_id is required")
	}

	base := c.endpoint(fileRetrieveEndpoint)
	requestURL := base + "?file_id=" + url.QueryEscape(trimmed)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.config.APIKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("hailuo file retrieve failed with status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	fileResp := &FileRetrieveResponse{}
	if err := json.NewDecoder(resp.Body).Decode(fileResp); err != nil {
		return nil, fmt.Errorf("decode hailuo file response: %w", err)
	}
	return fileResp, nil
}

func (c *Client) Download(ctx context.Context, taskID string) ([]byte, error) {
	endpoint := c.endpoint(fmt.Sprintf(downloadEndpointTemplate, taskID))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.config.APIKey)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("hailuo download failed with status %d: %s", resp.StatusCode, string(body))
	}
	return io.ReadAll(resp.Body)
}

func normalizeSubjectReferences(refs []VideoSubjectReference) []VideoSubjectReference {
	if len(refs) == 0 {
		return nil
	}
	var cleaned []VideoSubjectReference
	for _, ref := range refs {
		typeValue := strings.TrimSpace(ref.Type)
		if typeValue == "" {
			continue
		}
		var images []string
		for _, img := range ref.Image {
			if trimmed := strings.TrimSpace(img); trimmed != "" {
				images = append(images, trimmed)
			}
		}
		if len(images) == 0 {
			continue
		}
		cleaned = append(cleaned, VideoSubjectReference{
			Type:  typeValue,
			Image: images,
		})
	}
	return cleaned
}
