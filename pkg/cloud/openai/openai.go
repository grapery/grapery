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
)

const (
	APiKey = ""
)

type OpenAIClient struct {
	Clent *openai.Client
}

func NewOpenAIClient() *OpenAIClient {
	return &OpenAIClient{
		Clent: openai.NewClient(APiKey),
	}
}

func (c *OpenAIClient) ImageToText(ctx context.Context) {
	req := openai.CompletionRequest{
		Model:     openai.GPT3Ada,
		MaxTokens: 5,
		Prompt:    "Lorem ipsum",
	}
	resp, err := c.Clent.CreateCompletion(ctx, req)
	if err != nil {
		fmt.Printf("Completion error: %v\n", err)
		return
	}
	fmt.Println(resp.Choices[0].Text)
}

func (c *OpenAIClient) ChatOneOnOne(ctx context.Context) {
	req := openai.CompletionRequest{
		Model:     openai.GPT3Ada,
		MaxTokens: 5,
		Prompt:    "Lorem ipsum",
	}
	resp, err := c.Clent.CreateCompletion(ctx, req)
	if err != nil {
		fmt.Printf("Completion error: %v\n", err)
		return
	}
	fmt.Println(resp.Choices[0].Text)
}

func (c *OpenAIClient) ChatStream(ctx context.Context) {
	req := openai.CompletionRequest{
		Model:     openai.GPT3Ada,
		MaxTokens: 5,
		Prompt:    "Lorem ipsum",
		Stream:    true,
	}
	stream, err := c.Clent.CreateCompletionStream(ctx, req)
	if err != nil {
		fmt.Printf("CompletionStream error: %v\n", err)
		return
	}
	defer stream.Close()

	for {
		response, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			fmt.Println("Stream finished")
			return
		}

		if err != nil {
			fmt.Printf("Stream error: %v\n", err)
			return
		}

		fmt.Printf("Stream response: %v\n", response)
	}
}

type ImageGenParams struct {
	Model          string `json:"model,omitempty"`
	Prompt         string `json:"prompt,omitempty"`
	N              int    `json:"n,omitempty"`
	Quality        string `json:"quality,omitempty"`
	Size           string `json:"size,omitempty"`
	Style          string `json:"style,omitempty"`
	ResponseFormat string `json:"response_format,omitempty"`
	User           string `json:"user,omitempty"`

	TempPath string `json:"temp_path,omitempty"`
}

type ImagemageGenDetail struct {
	Data          []byte `json:"data,omitempty"`
	RevisedPrompt string `json:"revised_prompt,omitempty"`
	Url           string `json:"url,omitempty"`
}

type ImageGenResult struct {
	Images    []ImagemageGenDetail `json:"img_data"`
	TimeStamp int64                `json:"time_stamp,omitempty"`
}

func (c *OpenAIClient) ImageGen(ctx context.Context, params *ImageGenParams) (*ImageGenResult, error) {
	// Sample image by link
	reqUrl := openai.ImageRequest{
		Prompt:         params.Prompt,
		Size:           params.Size,
		ResponseFormat: openai.CreateImageResponseFormatURL,
		N:              params.N,
		Quality:        params.Quality,
		Model:          params.Model,
		Style:          params.Style,
		User:           params.User,
	}

	var result = new(ImageGenResult)
	resp, err := c.Clent.CreateImage(ctx, reqUrl)
	if err != nil {
		fmt.Printf("Image creation error: %v\n", err)
		return nil, err
	}
	fmt.Println(resp.Data[0].URL)

	imgBytes, err := base64.StdEncoding.DecodeString(resp.Data[0].B64JSON)
	if err != nil {
		fmt.Printf("Base64 decode error: %v\n", err)
		return nil, err
	}

	r := bytes.NewReader(imgBytes)
	imgData, err := png.Decode(r)
	if err != nil {
		fmt.Printf("PNG decode error: %v\n", err)
		return nil, err
	}

	file, err := os.Create(params.TempPath)
	if err != nil {
		fmt.Printf("File creation error: %v\n", err)
		return nil, err
	}
	defer file.Close()

	if err := png.Encode(file, imgData); err != nil {
		fmt.Printf("PNG encode error: %v\n", err)
		return nil, err
	}

	return result, nil
}

type SpeechToTextParams struct {
	ResourceUrl string
	Prompt      string
	TargetLang  string
	Model       string
}

type SpeechToTextResult struct {
	Task     string  `json:"task"`
	Language string  `json:"language"`
	Duration float64 `json:"duration"`
	Text     string  `json:"text"`
	Segments []struct {
		ID          int     `json:"id"`
		Start       float64 `json:"start"`
		Temperature float64 `json:"temperature"`
		End         float64 `json:"end"`
	}
}

func (c *OpenAIClient) SpeechToText(ctx context.Context, params SpeechToTextParams) (*SpeechToTextResult, error) {
	req := openai.AudioRequest{
		Model:    params.Model,
		FilePath: params.ResourceUrl,
		Prompt:   params.Prompt,
		Language: params.TargetLang,
		Format:   openai.AudioResponseFormatJSON,
	}
	resp, err := c.Clent.CreateTranscription(ctx, req)
	if err != nil {
		fmt.Printf("Transcription error: %v\n", err)
		return nil, err
	}
	ret := &SpeechToTextResult{
		Task:     resp.Task,
		Language: resp.Language,
		Duration: resp.Duration,
		Text:     resp.Text,
	}
	err = copier.Copy(&ret.Segments, &resp.Segments)
	if err != nil {
		fmt.Printf("Copy error: %v\n", err)
		return nil, err
	}
	return ret, nil
}
