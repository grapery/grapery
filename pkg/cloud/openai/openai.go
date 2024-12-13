package openai

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"image/png"
	"io"
	"os"

	"github.com/jinzhu/copier"
	openai "github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
)

// Constants
const (
	APIKey = ""
)

// OpenAIClient wraps the OpenAI API client
type OpenAIClient struct {
	Client *openai.Client
}

// NewOpenAIClient creates a new OpenAI client instance
func NewOpenAIClient() *OpenAIClient {
	opts := []option.RequestOption{
		option.WithAPIKey(APIKey),
	}
	return &OpenAIClient{
		Client: openai.NewClient(opts...),
	}
}

// Vision related types and methods
type VisionParams struct {
	ImageURL  string
	Prompt    string
	MaxTokens int
}

func (c *OpenAIClient) ImageToText(ctx context.Context, params VisionParams) (string, error) {
	req := openai.VisionRequest{
		Model: openai.GPT4VisionPreview,
		Messages: []openai.ChatMessage{
			{
				Role: openai.ChatMessageRoleUser,
				Content: []openai.ChatContent{
					{
						Type: openai.ChatContentTypeText,
						Text: params.Prompt,
					},
					{
						Type: openai.ChatContentTypeImageURL,
						ImageURL: &openai.ChatContentImageURL{
							URL: params.ImageURL,
						},
					},
				},
			},
		},
		MaxTokens: params.MaxTokens,
	}

	resp, err := c.Client.CreateVision(ctx, req)
	if err != nil {
		return "", fmt.Errorf("vision error: %w", err)
	}

	return resp.Choices[0].Message.Content, nil
}

// Chat related types and methods
type ChatParams struct {
	Prompt    string
	MaxTokens int
	Model     string
}

func (c *OpenAIClient) ChatOneOnOne(ctx context.Context, params ChatParams) (string, error) {
	if params.Model == "" {
		params.Model = openai.GPT4TurboPreview
	}

	req := openai.ChatRequest{
		Model: params.Model,
		Messages: []openai.ChatMessage{
			{
				Role:    openai.ChatMessageRoleUser,
				Content: []openai.ChatContent{{Text: params.Prompt, Type: openai.ChatContentTypeText}},
			},
		},
		MaxTokens: params.MaxTokens,
	}

	resp, err := c.Client.CreateChat(ctx, req)
	if err != nil {
		return "", fmt.Errorf("chat error: %w", err)
	}

	return resp.Choices[0].Message.Content, nil
}

func (c *OpenAIClient) ChatStream(ctx context.Context, params ChatParams) (<-chan string, <-chan error) {
	resultChan := make(chan string)
	errChan := make(chan error, 1)

	if params.Model == "" {
		params.Model = openai.GPT4TurboPreview
	}

	req := openai.ChatRequest{
		Model: params.Model,
		Messages: []openai.ChatMessage{
			{
				Role:    openai.ChatMessageRoleUser,
				Content: []openai.ChatContent{{Text: params.Prompt, Type: openai.ChatContentTypeText}},
			},
		},
		MaxTokens: params.MaxTokens,
		Stream:    true,
	}

	go func() {
		defer close(resultChan)
		defer close(errChan)

		stream, err := c.Client.CreateChatStream(ctx, req)
		if err != nil {
			errChan <- fmt.Errorf("chat stream error: %w", err)
			return
		}
		defer stream.Close()

		for {
			response, err := stream.Recv()
			if errors.Is(err, io.EOF) {
				return
			}
			if err != nil {
				errChan <- fmt.Errorf("stream receive error: %w", err)
				return
			}

			if len(response.Choices) > 0 && response.Choices[0].Delta.Content != "" {
				resultChan <- response.Choices[0].Delta.Content
			}
		}
	}()

	return resultChan, errChan
}

// Image generation related types and methods
type ImageGenParams struct {
	Model          string `json:"model,omitempty"`
	Prompt         string `json:"prompt,omitempty"`
	N              int    `json:"n,omitempty"`
	Quality        string `json:"quality,omitempty"`
	Size           string `json:"size,omitempty"`
	Style          string `json:"style,omitempty"`
	ResponseFormat string `json:"response_format,omitempty"`
	User           string `json:"user,omitempty"`
	TempPath       string `json:"temp_path,omitempty"`
}

type ImageGenDetail struct {
	Data          []byte `json:"data,omitempty"`
	RevisedPrompt string `json:"revised_prompt,omitempty"`
	URL           string `json:"url,omitempty"`
}

type ImageGenResult struct {
	Images    []ImageGenDetail `json:"img_data"`
	TimeStamp int64            `json:"time_stamp,omitempty"`
}

func (c *OpenAIClient) ImageGen(ctx context.Context, params *ImageGenParams) (*ImageGenResult, error) {
	req := openai.ImageRequest{
		Prompt:         params.Prompt,
		Size:           params.Size,
		ResponseFormat: openai.CreateImageResponseFormatURL,
		N:              params.N,
		Quality:        params.Quality,
		Model:          params.Model,
		Style:          params.Style,
		User:           params.User,
	}

	resp, err := c.Client.CreateImage(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("image creation error: %w", err)
	}

	result := &ImageGenResult{
		Images: make([]ImageGenDetail, len(resp.Data)),
	}

	for i, data := range resp.Data {
		imgBytes, err := base64.StdEncoding.DecodeString(data.B64JSON)
		if err != nil {
			return nil, fmt.Errorf("base64 decode error: %w", err)
		}

		result.Images[i] = ImageGenDetail{
			Data:          imgBytes,
			RevisedPrompt: data.RevisedPrompt,
			URL:           data.URL,
		}

		if params.TempPath != "" {
			if err := saveImageToDisk(imgBytes, params.TempPath); err != nil {
				return nil, err
			}
		}
	}

	return result, nil
}

func saveImageToDisk(imgBytes []byte, path string) error {
	r := bytes.NewReader(imgBytes)
	imgData, err := png.Decode(r)
	if err != nil {
		return fmt.Errorf("PNG decode error: %w", err)
	}

	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("file creation error: %w", err)
	}
	defer file.Close()

	if err := png.Encode(file, imgData); err != nil {
		return fmt.Errorf("PNG encode error: %w", err)
	}

	return nil
}

// Audio/Speech related types and methods
type SpeechToTextParams struct {
	ResourceURL string
	Prompt      string
	TargetLang  string
	Model       string
}

type SpeechToTextSegment struct {
	ID          int     `json:"id"`
	Start       float64 `json:"start"`
	Temperature float64 `json:"temperature"`
	End         float64 `json:"end"`
}

type SpeechToTextResult struct {
	Task     string                `json:"task"`
	Language string                `json:"language"`
	Duration float64               `json:"duration"`
	Text     string                `json:"text"`
	Segments []SpeechToTextSegment `json:"segments"`
}

func (c *OpenAIClient) SpeechToText(ctx context.Context, params SpeechToTextParams) (*SpeechToTextResult, error) {
	req := openai.AudioRequest{
		Model:    params.Model,
		FilePath: params.ResourceURL,
		Prompt:   params.Prompt,
		Language: params.TargetLang,
		Format:   openai.AudioResponseFormatJSON,
	}

	resp, err := c.Client.CreateTranscription(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("transcription error: %w", err)
	}

	result := &SpeechToTextResult{
		Task:     resp.Task,
		Language: resp.Language,
		Duration: resp.Duration,
		Text:     resp.Text,
	}

	if err := copier.Copy(&result.Segments, &resp.Segments); err != nil {
		return nil, fmt.Errorf("copy error: %w", err)
	}

	return result, nil
}

// Fine-tuning related types and methods
type FineTuneParams struct {
	TrainingFile    string            `json:"training_file"`
	ValidationFile  string            `json:"validation_file,omitempty"`
	ModelName       string            `json:"model"`
	Hyperparameters map[string]string `json:"hyperparameters,omitempty"`
	Suffix          string            `json:"suffix,omitempty"`
}

type FineTuneStatus struct {
	ID             string  `json:"id"`
	Status         string  `json:"status"`
	Model          string  `json:"fine_tuned_model"`
	TrainingLoss   float64 `json:"training_loss,omitempty"`
	ValidationLoss float64 `json:"validation_loss,omitempty"`
	EpochProgress  int     `json:"epoch_progress"`
	TotalEpochs    int     `json:"total_epochs"`
}

func (c *OpenAIClient) CreateFineTune(ctx context.Context, params FineTuneParams) (*FineTuneStatus, error) {
	req := openai.FineTuningRequest{
		TrainingFile:   params.TrainingFile,
		ValidationFile: params.ValidationFile,
		Model:          params.ModelName,
		Hyperparameters: &openai.FineTuningHyperparameters{
			BatchSize:    params.Hyperparameters["batch_size"],
			LearningRate: params.Hyperparameters["learning_rate"],
			EpochCount:   params.Hyperparameters["epochs"],
		},
		Suffix: params.Suffix,
	}

	resp, err := c.Client.CreateFineTuning(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("create fine-tune error: %w", err)
	}

	return mapFineTuneResponse(resp), nil
}

func (c *OpenAIClient) GetFineTuneStatus(ctx context.Context, fineTuneID string) (*FineTuneStatus, error) {
	resp, err := c.Client.GetFineTuning(ctx, fineTuneID)
	if err != nil {
		return nil, fmt.Errorf("get fine-tune status error: %w", err)
	}

	return mapFineTuneResponse(resp), nil
}

func (c *OpenAIClient) CancelFineTune(ctx context.Context, fineTuneID string) error {
	if err := c.Client.CancelFineTuning(ctx, fineTuneID); err != nil {
		return fmt.Errorf("cancel fine-tune error: %w", err)
	}
	return nil
}

func (c *OpenAIClient) ListFineTunes(ctx context.Context) ([]FineTuneStatus, error) {
	resp, err := c.Client.ListFineTunings(ctx)
	if err != nil {
		return nil, fmt.Errorf("list fine-tunes error: %w", err)
	}

	result := make([]FineTuneStatus, len(resp.Data))
	for i, ft := range resp.Data {
		result[i] = *mapFineTuneResponse(ft)
	}

	return result, nil
}

func mapFineTuneResponse(resp *openai.FineTuning) *FineTuneStatus {
	return &FineTuneStatus{
		ID:             resp.ID,
		Status:         resp.Status,
		Model:          resp.FineTunedModel,
		TrainingLoss:   resp.TrainingMetrics.Loss,
		ValidationLoss: resp.ValidationMetrics.Loss,
		EpochProgress:  resp.CurrentEpoch,
		TotalEpochs:    resp.TotalEpochs,
	}
}
