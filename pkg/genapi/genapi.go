package genapi

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"
)

// Package genapi provides a high-level API for generating media using various providers.
// A unified request/response model is exposed so business code can switch providers without
// dealing with the nuances of each SDK.

// Provider describes the minimal contract every provider adapter must satisfy.
type Provider interface {
	Name() string
}

// ImageProvider exposes image generation capabilities for a provider.
type ImageProvider interface {
	Provider
	GenerateImage(ctx context.Context, req *GenerateRequest) (*GenerateResponse, error)
}

// VideoGenerator defines the common video generation signature.
type VideoGenerator interface {
	Generate(ctx context.Context, req *GenerateRequest) (*GenerateResponse, error)
}

// VideoStatusFetcher exposes status polling capabilities for a video provider.
type VideoStatusFetcher interface {
	GetVideoStatus(ctx context.Context, taskID string) (*GenerateResponse, error)
}

// VideoDownloader exposes optional binary download capabilities for a provider.
type VideoDownloader interface {
	DownloadVideo(ctx context.Context, taskID string) ([]byte, error)
}

// VideoProvider exposes video generation capabilities for a provider.
type VideoProvider interface {
	Provider
	VideoGenerator
	VideoStatusFetcher
}

// TokenUsageRecorder allows callers to track token/billing usage emitted by providers.
type TokenUsageRecorder interface {
	RecordUsage(ctx context.Context, provider string, usage *Usage)
}

// TokenUsageRecorderFunc is an adapter to allow ordinary functions to be used as recorders.
type TokenUsageRecorderFunc func(ctx context.Context, provider string, usage *Usage)

// RecordUsage implements TokenUsageRecorder.
func (f TokenUsageRecorderFunc) RecordUsage(ctx context.Context, provider string, usage *Usage) {
	if f == nil {
		return
	}
	f(ctx, provider, usage)
}

type noopTokenUsageRecorder struct{}

func (noopTokenUsageRecorder) RecordUsage(context.Context, string, *Usage) {}

var (
	recorderMu          sync.RWMutex
	globalUsageRecorder TokenUsageRecorder = noopTokenUsageRecorder{}
)

// SetTokenUsageRecorder installs a custom recorder used by all GenAPI instances created afterwards.
func SetTokenUsageRecorder(recorder TokenUsageRecorder) {
	recorderMu.Lock()
	defer recorderMu.Unlock()
	if recorder == nil {
		globalUsageRecorder = noopTokenUsageRecorder{}
		return
	}
	globalUsageRecorder = recorder
}

// NewGenAPI constructs a GenAPI instance with empty provider registries.
func NewGenAPI() *GenAPI {
	recorderMu.RLock()
	defer recorderMu.RUnlock()

	return &GenAPI{
		providers:      make(map[string]Provider),
		imageProviders: make(map[string]ImageProvider),
		videoProviders: make(map[string]VideoProvider),
		usageRecorder:  globalUsageRecorder,
	}
}

// GenAPI offers unified image/video generation across multiple providers.
type GenAPI struct {
	mu             sync.RWMutex
	providers      map[string]Provider
	imageProviders map[string]ImageProvider
	videoProviders map[string]VideoProvider
	usageRecorder  TokenUsageRecorder
}

// RegisterProvider registers the supplied provider and automatically wires its capabilities.
func (g *GenAPI) RegisterProvider(provider Provider) {
	if provider == nil {
		return
	}
	g.mu.Lock()
	defer g.mu.Unlock()

	name := normalizeProviderName(provider.Name())
	if name == "" {
		return
	}
	g.providers[name] = provider
	if img, ok := provider.(ImageProvider); ok {
		g.imageProviders[name] = img
	}
	if vid, ok := provider.(VideoProvider); ok {
		g.videoProviders[name] = vid
	}
}

// GetProvider retrieves the provider by the given name if registered.
func (g *GenAPI) GetProvider(name string) Provider {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.providers[normalizeProviderName(name)]
}

// GetImageProvider returns the image provider registered under the given name.
func (g *GenAPI) GetImageProvider(name string) ImageProvider {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.imageProviders[normalizeProviderName(name)]
}

// GetVideoProvider returns the video provider registered under the given name.
func (g *GenAPI) GetVideoProvider(name string) VideoProvider {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.videoProviders[normalizeProviderName(name)]
}

// GenerateImage runs an image generation workflow on the selected provider.
func (g *GenAPI) GenerateImage(ctx context.Context, providerName string, req *GenerateRequest) (*GenerateResponse, error) {
	if req == nil {
		return nil, errors.New("generate request cannot be nil")
	}
	provider := g.GetImageProvider(providerName)
	if provider == nil {
		return nil, fmt.Errorf("image provider %s not registered", providerName)
	}

	cloned := req.Clone()
	if cloned.Operation == OperationUnknown {
		cloned.Operation = defaultImageOperation(cloned)
	}
	start := time.Now()
	rsp, err := provider.GenerateImage(ctx, cloned)
	rsp = g.normalizeResponse(rsp, provider.Name(), cloned, MediaTypeImage, start)
	if err == nil {
		g.recordUsage(ctx, rsp)
	}
	return rsp, err
}

// GenerateVideo runs a video generation workflow on the selected provider.
func (g *GenAPI) GenerateVideo(ctx context.Context, providerName string, req *GenerateRequest) (*GenerateResponse, error) {
	if req == nil {
		return nil, errors.New("generate request cannot be nil")
	}
	provider := g.GetVideoProvider(providerName)
	if provider == nil {
		return nil, fmt.Errorf("video provider %s not registered", providerName)
	}

	cloned := req.Clone()
	if cloned.Operation == OperationUnknown {
		cloned.Operation = defaultVideoOperation(cloned)
	}
	start := time.Now()
	rsp, err := provider.Generate(ctx, cloned)
	rsp = g.normalizeResponse(rsp, provider.Name(), cloned, MediaTypeVideo, start)
	if err == nil {
		g.recordUsage(ctx, rsp)
	}
	return rsp, err
}

func (g *GenAPI) normalizeResponse(rsp *GenerateResponse, providerName string, req *GenerateRequest, mediaType MediaType, startedAt time.Time) *GenerateResponse {
	if rsp == nil {
		rsp = &GenerateResponse{}
	}
	if rsp.Provider == "" {
		rsp.Provider = providerName
	}
	if rsp.Operation == OperationUnknown && req != nil {
		rsp.Operation = req.Operation
	}
	if rsp.MediaType == "" {
		rsp.MediaType = mediaType
	}
	if rsp.StartedAt.IsZero() {
		rsp.StartedAt = startedAt
	}
	if rsp.CompletedAt.IsZero() {
		rsp.CompletedAt = time.Now()
	}
	if req != nil {
		rsp.Metadata = mergeMaps(req.Metadata, rsp.Metadata)
	} else if rsp.Metadata == nil {
		rsp.Metadata = make(map[string]interface{})
	}
	if rsp.Raw == nil {
		rsp.Raw = make(map[string]interface{})
	}
	return rsp
}

func (g *GenAPI) recordUsage(ctx context.Context, rsp *GenerateResponse) {
	if rsp == nil || rsp.Usage == nil || rsp.Usage.IsEmpty() {
		return
	}
	if g.usageRecorder == nil {
		return
	}
	g.usageRecorder.RecordUsage(ctx, rsp.Provider, rsp.Usage)
}

func defaultVideoOperation(req *GenerateRequest) OperationType {
	if req == nil {
		return OperationTextToVideo
	}
	switch req.Mode {
	case GenerationModeImage:
		return OperationImageToVideo
	case GenerationModeKeyframe:
		return OperationKeyframeToVideo
	case GenerationModeStoryboard:
		return OperationStoryboardToVideo
	}
	if strings.TrimSpace(req.FirstFrameURL) != "" && strings.TrimSpace(req.LastFrameURL) != "" {
		return OperationKeyframeToVideo
	}
	if len(req.ReferenceImages) > 0 || strings.TrimSpace(req.ReferenceImageURL) != "" {
		return OperationImageToVideo
	}
	if req.Storyboard != nil {
		return OperationStoryboardToVideo
	}
	return OperationTextToVideo
}

func defaultImageOperation(req *GenerateRequest) OperationType {
	if req == nil {
		return OperationTextToImage
	}
	switch req.Mode {
	case GenerationModeImage:
		return OperationImageToImage
	}
	if len(req.ReferenceImages) > 0 || strings.TrimSpace(req.ReferenceImageURL) != "" {
		return OperationImageToImage
	}
	if req.ImageData != nil {
		return OperationImageToImage
	}
	return OperationTextToImage
}
