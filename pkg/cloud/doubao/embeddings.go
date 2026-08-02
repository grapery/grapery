package doubao

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// CreateTextEmbedding creates embeddings for text inputs
func (c *SeedreamClient) CreateTextEmbedding(ctx context.Context, texts []string, options ...func(*TextEmbeddingRequest)) (*TextEmbeddingResponse, error) {
	req := &TextEmbeddingRequest{
		Model:          ModelEmbeddingText,
		Input:          texts,
		EncodingFormat: EncodingFormatFloat,
	}

	// Apply options
	for _, opt := range options {
		opt(req)
	}

	return c.doTextEmbeddingRequest(ctx, req)
}

// CreateMultimodalEmbedding creates embeddings for multimodal inputs
func (c *SeedreamClient) CreateMultimodalEmbedding(ctx context.Context, inputs []MultimodalInput, options ...func(*MultimodalEmbeddingRequest)) (*MultimodalEmbeddingResponse, error) {
	req := &MultimodalEmbeddingRequest{
		Model:          ModelEmbeddingVision,
		Input:          inputs,
		EncodingFormat: EncodingFormatFloat,
	}

	// Apply options
	for _, opt := range options {
		opt(req)
	}

	return c.doMultimodalEmbeddingRequest(ctx, req)
}

// doTextEmbeddingRequest performs the HTTP request for text embedding
func (c *SeedreamClient) doTextEmbeddingRequest(ctx context.Context, req *TextEmbeddingRequest) (*TextEmbeddingResponse, error) {
	// Serialize request
	jsonData, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.Endpoint+"/api/v3/embeddings", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.APIKey)

	// Execute request
	resp, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Parse response
	var response TextEmbeddingResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Check for API errors
	if response.Error != nil {
		return nil, response.Error
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP error %d: %s", resp.StatusCode, string(body))
	}

	return &response, nil
}

// doMultimodalEmbeddingRequest performs the HTTP request for multimodal embedding
func (c *SeedreamClient) doMultimodalEmbeddingRequest(ctx context.Context, req *MultimodalEmbeddingRequest) (*MultimodalEmbeddingResponse, error) {
	// Serialize request
	jsonData, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.Endpoint+"/api/v3/embeddings/multimodal", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.APIKey)

	// Execute request
	resp, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Parse response
	var response MultimodalEmbeddingResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Check for API errors
	if response.Error != nil {
		return nil, response.Error
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP error %d: %s", resp.StatusCode, string(body))
	}

	return &response, nil
}

// Option functions for text embedding requests

// WithTextModel sets the model for text embedding
func WithTextModel(model string) func(*TextEmbeddingRequest) {
	return func(req *TextEmbeddingRequest) {
		req.Model = model
	}
}

// WithTextEncodingFormat sets the encoding format for text embedding
func WithTextEncodingFormat(format string) func(*TextEmbeddingRequest) {
	return func(req *TextEmbeddingRequest) {
		req.EncodingFormat = format
	}
}

// Option functions for multimodal embedding requests

// WithMultimodalModel sets the model for multimodal embedding
func WithMultimodalModel(model string) func(*MultimodalEmbeddingRequest) {
	return func(req *MultimodalEmbeddingRequest) {
		req.Model = model
	}
}

// WithMultimodalEncodingFormat sets the encoding format for multimodal embedding
func WithMultimodalEncodingFormat(format string) func(*MultimodalEmbeddingRequest) {
	return func(req *MultimodalEmbeddingRequest) {
		req.EncodingFormat = format
	}
}
