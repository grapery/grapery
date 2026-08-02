package hailuo

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

type MusicGenerationRequest struct {
	Model        string             `json:"model"`
	Prompt       string             `json:"prompt"`
	Lyrics       string             `json:"lyrics"`
	Stream       *bool              `json:"stream,omitempty"`
	OutputFormat string             `json:"output_format,omitempty"`
	AudioSetting *MusicAudioSetting `json:"audio_setting,omitempty"`
}

type MusicAudioSetting struct {
	SampleRate int    `json:"sample_rate,omitempty"`
	Bitrate    int    `json:"bitrate,omitempty"`
	Format     string `json:"format,omitempty"`
}

type MusicGenerationResponse struct {
	Data         *MusicGenerationData   `json:"data"`
	TraceID      string                 `json:"trace_id"`
	ExtraInfo    *MusicGenerationDetail `json:"extra_info"`
	AnalysisInfo json.RawMessage        `json:"analysis_info"`
	BaseResponse APIBaseResponse        `json:"base_resp"`
}

type MusicGenerationData struct {
	Audio  string `json:"audio"`
	Status int    `json:"status"`
	URL    string `json:"url,omitempty"`
}

type MusicGenerationDetail struct {
	MusicDuration   int `json:"music_duration"`
	MusicSampleRate int `json:"music_sample_rate"`
	MusicChannel    int `json:"music_channel"`
	Bitrate         int `json:"bitrate"`
	MusicSize       int `json:"music_size"`
}

func (c *Client) GenerateMusic(ctx context.Context, payload *MusicGenerationRequest) (*MusicGenerationResponse, error) {
	if payload == nil {
		return nil, fmt.Errorf("payload cannot be nil")
	}

	payload.Prompt = strings.TrimSpace(payload.Prompt)
	if payload.Prompt == "" {
		return nil, fmt.Errorf("prompt is required")
	}

	payload.Lyrics = strings.TrimSpace(payload.Lyrics)
	if payload.Lyrics == "" {
		return nil, fmt.Errorf("lyrics are required")
	}

	model := strings.TrimSpace(payload.Model)
	if model == "" {
		payload.Model = choose(c.config.MusicModel, "music-1.5")
		if strings.TrimSpace(payload.Model) == "" {
			return nil, fmt.Errorf("model is required")
		}
	} else {
		payload.Model = model
	}

	resp := &MusicGenerationResponse{}
	if err := c.doRequest(ctx, http.MethodPost, musicGenerationEndpoint, payload, resp); err != nil {
		return nil, err
	}
	return resp, nil
}
