package gemini

import (
	"context"
	"fmt"
	"strings"

	"google.golang.org/genai"
)


// GenerateVideoResponse captures the essential details of a long-running video operation.
type GenerateVideoResponse struct {
	Name     string                 `json:"name"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
	State    string                 `json:"state,omitempty"`
}

// VideoStatusResponse represents the current state of a generated video resource.
type VideoStatusResponse struct {
	Name     string                 `json:"name"`
	State    string                 `json:"state"`
	Error    map[string]interface{} `json:"error,omitempty"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
	Assets   []VideoAsset           `json:"assets,omitempty"`
}

// VideoAsset describes a generated asset in the status response.
type VideoAsset struct {
	Type string `json:"type"`
	URI  string `json:"uri"`
}

// GenerateVideo triggers a video generation request using the official SDK.
func (c *Client) GenerateVideo(ctx context.Context, model string, source *genai.GenerateVideosSource, config *genai.GenerateVideosConfig) (*GenerateVideoResponse, error) {
	if source == nil {
		return nil, fmt.Errorf("source cannot be nil")
	}
	resolvedModel := strings.TrimSpace(model)
	if resolvedModel == "" {
		candidate := strings.TrimSpace(c.config.DefaultModel)
		if candidate != "" && looksLikeVideoModel(candidate) {
			resolvedModel = candidate
		} else {
			resolvedModel = DefaultVideoModel
		}
	}
	op, err := c.sdk.Models.GenerateVideosFromSource(ctx, resolvedModel, source, config)
	if err != nil {
		return nil, err
	}
	return &GenerateVideoResponse{
		Name:     op.Name,
		Metadata: op.Metadata,
		State:    deriveOperationState(op),
	}, nil
}

// GetVideoStatus retrieves the state of a long-running video generation request.
func (c *Client) GetVideoStatus(ctx context.Context, taskID string) (*VideoStatusResponse, error) {
	name := strings.TrimSpace(taskID)
	if name == "" {
		return nil, fmt.Errorf("taskID is required")
	}
	op, err := c.sdk.Operations.GetVideosOperation(ctx, &genai.GenerateVideosOperation{Name: name}, nil)
	if err != nil {
		return nil, err
	}
	return buildVideoStatus(op), nil
}

// DownloadVideo retrieves the first generated video asset for the given task.
func (c *Client) DownloadVideo(ctx context.Context, taskID string) ([]byte, error) {
	name := strings.TrimSpace(taskID)
	if name == "" {
		return nil, fmt.Errorf("taskID is required")
	}
	op, err := c.sdk.Operations.GetVideosOperation(ctx, &genai.GenerateVideosOperation{Name: name}, nil)
	if err != nil {
		return nil, err
	}
	if op.Response == nil || len(op.Response.GeneratedVideos) == 0 {
		return nil, fmt.Errorf("operation %s has no generated videos", name)
	}
	data, err := c.sdk.Files.Download(ctx, genai.NewDownloadURIFromGeneratedVideo(op.Response.GeneratedVideos[0]), nil)
	if err != nil {
		return nil, err
	}
	return data, nil
}

func deriveOperationState(op *genai.GenerateVideosOperation) string {
	if op == nil {
		return ""
	}
	if !op.Done {
		return "PROCESSING"
	}
	if op.Error != nil && len(op.Error) > 0 {
		return "FAILED"
	}
	return "COMPLETED"
}

func buildVideoStatus(op *genai.GenerateVideosOperation) *VideoStatusResponse {
	if op == nil {
		return nil
	}
	status := &VideoStatusResponse{
		Name:     op.Name,
		State:    deriveOperationState(op),
		Metadata: op.Metadata,
	}
	if op.Error != nil {
		status.Error = op.Error
	}
	if op.Response != nil {
		for _, video := range op.Response.GeneratedVideos {
			if video == nil || video.Video == nil {
				continue
			}
			assetType := strings.TrimSpace(video.Video.MIMEType)
			if assetType == "" {
				assetType = "video"
			}
			status.Assets = append(status.Assets, VideoAsset{
				Type: assetType,
				URI:  video.Video.URI,
			})
		}
	}
	return status
}

func looksLikeVideoModel(model string) bool {
	lower := strings.ToLower(model)
	return strings.Contains(lower, "video") || strings.HasPrefix(lower, "veo")
}
