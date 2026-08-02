package asynctask

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/hibiken/asynq"
	log "github.com/sirupsen/logrus"

	"github.com/grapery/common-protoc/gen"
	"github.com/grapery/grapery/models"
	"github.com/grapery/grapery/pkg/cloud/aliyun"
	"github.com/grapery/grapery/pkg/genapi"
)

const (
	VideoTaskType   = "video:generate"
	VideoQueueName  = "video"
	defaultMaxRetry = 3

	statusPollInterval     = 10 * time.Second
	maxStatusFailures      = 12
	downloadRequestTimeout = 2 * time.Minute
)

// VideoGeneratePayload represents the payload required to trigger a video generation task.
type VideoGeneratePayload struct {
	VideoGenID    int64                  `json:"video_gen_id"`
	TaskID        string                 `json:"task_id"`
	UserID        int64                  `json:"user_id"`
	Platform      string                 `json:"platform"`
	Prompt        string                 `json:"prompt"`
	AspectRatio   string                 `json:"aspect_ratio,omitempty"`
	Duration      int                    `json:"duration,omitempty"`
	Quality       string                 `json:"quality,omitempty"`
	Style         string                 `json:"style,omitempty"`
	Model         string                 `json:"model,omitempty"`
	CallbackURL   string                 `json:"callback_url,omitempty"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
	StartRefImage string                 `json:"start_ref_image,omitempty"`
	EndRefImage   string                 `json:"end_ref_image,omitempty"`
	StoryID       int64                  `json:"story_id,omitempty"`
	BoardID       int64                  `json:"board_id,omitempty"`
	SceneID       int64                  `json:"scene_id,omitempty"`
	RoleID        int64                  `json:"role_id,omitempty"`
}

// NewVideoGenerateTask constructs a new Asynq task for video generation.
func NewVideoGenerateTask(payload *VideoGeneratePayload) (*asynq.Task, error) {
	if payload == nil {
		return nil, fmt.Errorf("video generate payload is nil: %w", asynq.SkipRetry)
	}

	if payload.VideoGenID == 0 {
		return nil, fmt.Errorf("video generate payload missing video_gen_id: %w", asynq.SkipRetry)
	}
	if strings.TrimSpace(payload.TaskID) == "" {
		return nil, fmt.Errorf("video generate payload missing task_id: %w", asynq.SkipRetry)
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal video payload: %w", err)
	}
	return asynq.NewTask(VideoTaskType, data), nil
}

// VideoTaskHandler handles execution of the video generation job.
type VideoTaskHandler struct {
	videoProviders map[string]genapi.VideoProvider
}

// NewVideoTaskHandler creates a handler with the given provider configurations.
func NewVideoTaskHandler(videoProviders map[string]genapi.VideoProvider) *VideoTaskHandler {
	return &VideoTaskHandler{
		videoProviders: videoProviders,
	}
}

// ProcessTask implements the asynq.Handler interface.
func (h *VideoTaskHandler) ProcessTask(ctx context.Context, task *asynq.Task) error {
	var payload VideoGeneratePayload
	if err := json.Unmarshal(task.Payload(), &payload); err != nil {
		log.Errorf("video task payload decode failed: %v", err)
		return fmt.Errorf("decode video payload: %w", err)
	}

	platform := strings.ToLower(strings.TrimSpace(payload.Platform))
	if platform == "" {
		err := fmt.Errorf("video task missing platform: %w", asynq.SkipRetry)
		h.markGenerationError(ctx, &payload, "missing video platform")
		return err
	}
	if platform == "coze" {
		platform = "huoshan"
	}

	provider, ok := h.videoProviders[platform]
	videoProvidersData, _ := json.Marshal(h.videoProviders)
	log.Infof("video providers data: %s", string(videoProvidersData))
	if !ok {
		err := fmt.Errorf("video platform %s not configured: %w", payload.Platform, asynq.SkipRetry)
		h.markGenerationError(ctx, &payload, fmt.Sprintf("platform %s not configured", payload.Platform))
		return err
	}

	videoRecord, err := models.GetVideoGen(ctx, payload.VideoGenID)
	if err != nil {
		log.Errorf("load video task record failed: %v", err)
		h.markGenerationError(ctx, &payload, "load video task record failed")
		return err
	}

	if isTerminalStatus(videoRecord.GenStatus) && strings.TrimSpace(videoRecord.VideoUrl) != "" {
		log.Infof("video task %d already completed (%s), skipping", payload.VideoGenID, videoRecord.GenStatus.String())
		return nil
	}

	runningUpdate := map[string]interface{}{
		"gen_status": gen.StoryGenStatus_STORY_GEN_STATUS_RUNNING,
		"provider":   platform,
	}
	if videoRecord.StartTime == 0 {
		runningUpdate["start_time"] = time.Now().Unix()
	}
	if err := models.UpdateVideoGenFields(ctx, payload.VideoGenID, runningUpdate); err != nil {
		log.Errorf("update video status to running failed: %v", err)
	}
	updateSceneStatus(ctx, payload.SceneID, gen.StoryGenStatus_STORY_GEN_STATUS_RUNNING, map[string]interface{}{
		"task_id": payload.TaskID,
	})

	metadata := make(map[string]interface{})
	for k, v := range payload.Metadata {
		metadata[k] = v
	}
	if trimmed := strings.TrimSpace(payload.StartRefImage); trimmed != "" {
		metadata["start_ref_image"] = trimmed
	}
	if trimmed := strings.TrimSpace(payload.EndRefImage); trimmed != "" {
		metadata["end_ref_image"] = trimmed
	}

	mode := genapi.GenerationModeText
	if strings.TrimSpace(payload.StartRefImage) != "" && strings.TrimSpace(payload.EndRefImage) != "" {
		mode = genapi.GenerationModeKeyframe
	} else if strings.TrimSpace(payload.StartRefImage) != "" {
		mode = genapi.GenerationModeImage
	}

	operation := genapi.OperationTextToVideo
	switch mode {
	case genapi.GenerationModeImage:
		operation = genapi.OperationImageToVideo
	case genapi.GenerationModeKeyframe:
		operation = genapi.OperationKeyframeToVideo
	}

	req := &genapi.GenerateRequest{
		Operation:       operation,
		Prompt:          payload.Prompt,
		Mode:            mode,
		AspectRatio:     payload.AspectRatio,
		DurationSeconds: payload.Duration,
		Quality:         payload.Quality,
		Style:           payload.Style,
		Model:           payload.Model,
		CallbackURL:     payload.CallbackURL,
		Metadata:        metadata,
		Platform:        payload.Platform,
		UserID:          payload.UserID,
	}

	if mode == genapi.GenerationModeImage {
		req.ReferenceImageURL = payload.StartRefImage
	}
	if mode == genapi.GenerationModeKeyframe {
		req.FirstFrameURL = payload.StartRefImage
		req.LastFrameURL = payload.EndRefImage
	}

	providerTaskID := strings.TrimSpace(videoRecord.UUID)
	var latest *genapi.GenerateResponse

	if providerTaskID == "" {
		resp, err := provider.Generate(ctx, req)
		if err != nil {
			log.Errorf("video generation failed: %v", err)
			h.markGenerationError(ctx, &payload, fmt.Sprintf("generate failed: %v", err))
			return err
		}
		if resp == nil {
			h.markGenerationError(ctx, &payload, "provider returned empty response")
			return fmt.Errorf("provider returned empty response")
		}
		resp.Provider = provider.Name()
		providerTaskID = strings.TrimSpace(resp.TaskID)
		if resp.Error != "" {
			log.Errorf("video generation returned error: %s (code: %s)", resp.Error, resp.ErrorCode)
			h.markGenerationError(ctx, &payload, resp.Error)
			return fmt.Errorf("generation error: %s", resp.Error)
		}
		latest = resp
	} else {
		log.Infof("video task %d resumes with provider task id %s", payload.VideoGenID, providerTaskID)
	}

	if providerTaskID == "" {
		h.markGenerationError(ctx, &payload, "provider task id missing")
		return fmt.Errorf("provider task id missing: %w", asynq.SkipRetry)
	}

	if latest == nil {
		statusResp, err := provider.GetVideoStatus(ctx, providerTaskID)
		if err != nil {
			log.Warnf("initial video status fetch failed (task %s): %v", providerTaskID, err)
		} else {
			latest = statusResp
		}
	}

	if latest != nil {
		if strings.TrimSpace(latest.TaskID) == "" {
			latest.TaskID = providerTaskID
		}
		if latest.Provider == "" {
			latest.Provider = provider.Name()
		}
		status := h.persistResult(ctx, &payload, latest)
		if status == gen.StoryGenStatus_STORY_GEN_STATUS_ERROR {
			return nil
		}
		if status == gen.StoryGenStatus_STORY_GEN_STATUS_FINISHED {
			return h.handleSuccess(ctx, provider, &payload, latest)
		}
	}

	return h.pollStatus(ctx, provider, &payload, providerTaskID)
}

func (h *VideoTaskHandler) pollStatus(ctx context.Context, provider genapi.VideoProvider, payload *VideoGeneratePayload, providerTaskID string) error {
	failures := 0
	for {
		if err := ctx.Err(); err != nil {
			return err
		}

		statusResp, err := provider.GetVideoStatus(ctx, providerTaskID)
		if err != nil {
			failures++
			log.Errorf("poll video status failed (task %s): %v", providerTaskID, err)
			if failures >= maxStatusFailures {
				h.markGenerationError(ctx, payload, fmt.Sprintf("status check failed: %v", err))
				return nil
			}
		} else {
			failures = 0
			if statusResp != nil {
				if strings.TrimSpace(statusResp.TaskID) == "" {
					statusResp.TaskID = providerTaskID
				}
				if statusResp.Provider == "" {
					statusResp.Provider = provider.Name()
				}
				status := h.persistResult(ctx, payload, statusResp)
				if status == gen.StoryGenStatus_STORY_GEN_STATUS_FINISHED {
					return h.handleSuccess(ctx, provider, payload, statusResp)
				}
				if status == gen.StoryGenStatus_STORY_GEN_STATUS_ERROR {
					return nil
				}
			}
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(statusPollInterval):
		}
	}
}

func (h *VideoTaskHandler) handleSuccess(ctx context.Context, provider genapi.VideoProvider, payload *VideoGeneratePayload, resp *genapi.GenerateResponse) error {
	if resp == nil {
		return nil
	}

	ossURL, err := h.transferToOSS(ctx, provider, payload, resp)
	if err != nil {
		log.Errorf("transfer video to OSS failed: %v", err)
		h.markGenerationError(ctx, payload, fmt.Sprintf("transfer video failed: %v", err))
		return err
	}

	resp.VideoURL = ossURL
	if strings.TrimSpace(resp.Status) == "" {
		resp.Status = "completed"
	}
	if strings.TrimSpace(resp.Message) == "" {
		resp.Message = resp.Status
	}
	resp.CompletedAt = time.Now()

	status := h.persistResult(ctx, payload, resp)
	if status != gen.StoryGenStatus_STORY_GEN_STATUS_FINISHED {
		if err := models.UpdateVideoGenFields(ctx, payload.VideoGenID, map[string]interface{}{
			"gen_status": gen.StoryGenStatus_STORY_GEN_STATUS_FINISHED,
			"end_time":   time.Now().Unix(),
		}); err != nil {
			log.Errorf("force finish status update failed: %v", err)
		}
		updateSceneStatus(ctx, payload.SceneID, gen.StoryGenStatus_STORY_GEN_STATUS_FINISHED, map[string]interface{}{
			"task_id":   payload.TaskID,
			"video_url": resp.VideoURL,
			"uuid":      resp.TaskID,
		})
	}

	return nil
}

func (h *VideoTaskHandler) transferToOSS(ctx context.Context, provider genapi.VideoProvider, payload *VideoGeneratePayload, resp *genapi.GenerateResponse) (string, error) {
	if resp == nil {
		return "", fmt.Errorf("empty response for transfer")
	}

	taskID := resp.TaskID
	if strings.TrimSpace(taskID) == "" {
		taskID = payload.TaskID
	}
	if strings.TrimSpace(taskID) == "" {
		taskID = fmt.Sprintf("video-%d", time.Now().UnixNano())
	}

	var data []byte
	var err error

	if downloader, ok := provider.(genapi.VideoDownloader); ok {
		data, err = downloader.DownloadVideo(ctx, taskID)
	}
	if len(data) == 0 && err != nil {
		log.Warnf("provider download failed, fallback to direct url: %v", err)
	}

	if len(data) == 0 {
		sourceURL := strings.TrimSpace(resp.VideoURL)
		if sourceURL == "" {
			return "", fmt.Errorf("provider did not return video url for download")
		}
		data, err = downloadViaHTTP(ctx, sourceURL)
	}
	if err != nil {
		return "", err
	}
	if len(data) == 0 {
		return "", fmt.Errorf("downloaded empty video content")
	}

	ossClient := aliyun.GetGlobalClient()
	if ossClient == nil {
		return "", fmt.Errorf("aliyun client not initialized")
	}

	objectKey := fmt.Sprintf("videos/%s.mp4", sanitizeObjectKey(firstNonEmptyString(resp.TaskID, payload.TaskID)))
	url, err := ossClient.UploadVideoBytes(objectKey, data)
	if err != nil {
		return "", err
	}
	return url, nil
}

func downloadViaHTTP(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	client := &http.Client{Timeout: downloadRequestTimeout}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return nil, fmt.Errorf("download failed with status %d: %s", resp.StatusCode, string(body))
	}
	return io.ReadAll(resp.Body)
}

func sanitizeObjectKey(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return fmt.Sprintf("video-%d", time.Now().UnixNano())
	}
	replacer := strings.NewReplacer(
		" ", "-",
		"/", "-",
		"\\", "-",
		":", "-",
		"?", "-",
		"&", "-",
		"=", "-",
		"#", "-",
		"@", "-",
		"+", "-",
		"%", "-",
		"*", "-",
		"|", "-",
		"\n", "-",
		"\r", "-",
	)
	sanitized := replacer.Replace(trimmed)
	sanitized = strings.Trim(sanitized, "-._")
	if sanitized == "" {
		return fmt.Sprintf("video-%d", time.Now().UnixNano())
	}
	return sanitized
}

func firstNonEmptyString(values ...string) string {
	for _, v := range values {
		if trimmed := strings.TrimSpace(v); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func (h *VideoTaskHandler) markGenerationError(ctx context.Context, payload *VideoGeneratePayload, message string) {
	trimmed := strings.TrimSpace(message)
	if trimmed == "" {
		trimmed = "video generation failed"
	}
	if err := models.UpdateVideoGenFields(ctx, payload.VideoGenID, map[string]interface{}{
		"gen_status": gen.StoryGenStatus_STORY_GEN_STATUS_ERROR,
		"message":    trimmed,
		"end_time":   time.Now().Unix(),
	}); err != nil {
		log.Errorf("update video task error status failed: %v", err)
	}
	updateSceneStatus(ctx, payload.SceneID, gen.StoryGenStatus_STORY_GEN_STATUS_ERROR, map[string]interface{}{
		"task_id": payload.TaskID,
		"error":   trimmed,
	})
}

func (h *VideoTaskHandler) persistResult(ctx context.Context, payload *VideoGeneratePayload, resp *genapi.GenerateResponse) gen.StoryGenStatus {
	if resp == nil {
		return gen.StoryGenStatus_STORY_GEN_STATUS_RUNNING
	}

	status := normalizeGeneratorStatus(resp.Status)
	update := map[string]interface{}{
		"gen_status": status,
	}

	if trimmed := strings.TrimSpace(resp.TaskID); trimmed != "" {
		update["uuid"] = trimmed
	}
	if trimmed := strings.TrimSpace(resp.Provider); trimmed != "" {
		update["provider"] = trimmed
	}

	errorMessage := strings.TrimSpace(resp.Error)
	if resp.ErrorCode != "" {
		update["code"] = resp.ErrorCode
	}

	message := strings.TrimSpace(resp.Message)
	if message == "" {
		message = strings.TrimSpace(resp.Status)
	}
	if errorMessage != "" {
		message = errorMessage
	}
	if message != "" {
		update["message"] = message
	}

	if trimmed := strings.TrimSpace(resp.VideoURL); trimmed != "" {
		update["video_url"] = trimmed
	}
	if trimmed := strings.TrimSpace(resp.ThumbnailURL); trimmed != "" {
		update["fisrt_frame"] = trimmed
	}

	if !resp.StartedAt.IsZero() {
		update["start_time"] = resp.StartedAt.Unix()
	}
	if isTerminalStatus(status) {
		completedAt := resp.CompletedAt
		if completedAt.IsZero() {
			completedAt = time.Now()
		}
		update["end_time"] = completedAt.Unix()
	}

	if resp.Usage != nil && resp.Usage.TotalTokens > 0 {
		update["tokens"] = resp.Usage.TotalTokens
		log.Infof("video generation used %d tokens (provider: %s)", resp.Usage.TotalTokens, resp.Provider)
	}

	if err := models.UpdateVideoGenFields(ctx, payload.VideoGenID, update); err != nil {
		log.Errorf("update video task result failed: %v", err)
	}

	extra := map[string]interface{}{
		"task_id": payload.TaskID,
	}
	if trimmed := strings.TrimSpace(resp.VideoURL); trimmed != "" {
		extra["video_url"] = trimmed
	}
	if trimmed := strings.TrimSpace(resp.ThumbnailURL); trimmed != "" {
		extra["image_url"] = trimmed
	}
	if trimmed := strings.TrimSpace(resp.TaskID); trimmed != "" {
		extra["uuid"] = trimmed
	}
	if errorMessage != "" {
		extra["error"] = errorMessage
	}
	updateSceneStatus(ctx, payload.SceneID, status, extra)

	return status
}

func normalizeGeneratorStatus(status string) gen.StoryGenStatus {
	s := strings.ToLower(strings.TrimSpace(status))
	if s == "" {
		return gen.StoryGenStatus_STORY_GEN_STATUS_RUNNING
	}
	switch s {
	case "completed", "finished", "success", "succeeded", "done":
		return gen.StoryGenStatus_STORY_GEN_STATUS_FINISHED
	case "failed", "error", "canceled", "cancelled", "timeout":
		return gen.StoryGenStatus_STORY_GEN_STATUS_ERROR
	case "pending", "queued", "processing", "running", "in_progress":
		return gen.StoryGenStatus_STORY_GEN_STATUS_RUNNING
	default:
		return gen.StoryGenStatus_STORY_GEN_STATUS_RUNNING
	}
}

func isTerminalStatus(status gen.StoryGenStatus) bool {
	return status == gen.StoryGenStatus_STORY_GEN_STATUS_FINISHED || status == gen.StoryGenStatus_STORY_GEN_STATUS_ERROR
}

func updateSceneStatus(ctx context.Context, sceneID int64, status gen.StoryGenStatus, extra map[string]interface{}) {
	if sceneID == 0 {
		return
	}
	values := map[string]interface{}{
		"gen_status": status,
	}
	for k, v := range extra {
		values[k] = v
	}
	if err := models.UpdateStoryBoardSceneMultiColumn(ctx, sceneID, values); err != nil {
		log.Errorf("update scene status failed: %v", err)
	}
}
