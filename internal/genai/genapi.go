package genapi

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	huoshanprovider "github.com/grapestree/fgrapery/grapery/internal/genai/providers/huoshan"
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

// ImageStatusFetcher exposes status polling capabilities for image generation tasks.
type ImageStatusFetcher interface {
	GetImageStatus(ctx context.Context, taskID string) (*GenerateResponse, error)
}

// VideoProvider exposes video generation capabilities for a provider.
type VideoProvider interface {
	Provider
	VideoGenerator
	VideoStatusFetcher
}

// UsageRecordContext holds metadata for recording AI generation usage (e.g. to DB).
// Pass via request Metadata or context when calling GenAPI.
type UsageRecordContext struct {
	UserID             string // User ID for attribution
	RelatedEntityID    string // e.g. storyboard ID, story ID
	RelatedEntityType  string // e.g. "storyboard", "story"
	OriginalPrompt     string // Original prompt (for text/image/video)
	EnhancedPrompt     string // Enhanced prompt if any
}

// Context keys for passing UsageRecordContext when request is not available (e.g. GetVideoStatus).
type usageRecordContextKey struct{}

// ContextWithUsageRecord attaches usage record metadata to ctx for TokenUsageRecorder.
func ContextWithUsageRecord(ctx context.Context, userID, relatedEntityID, relatedEntityType string) context.Context {
	if ctx == nil {
		return ctx
	}
	return context.WithValue(ctx, usageRecordContextKey{}, &UsageRecordContext{
		UserID:            userID,
		RelatedEntityID:   relatedEntityID,
		RelatedEntityType: relatedEntityType,
	})
}

// UsageRecordFromContext extracts UsageRecordContext from ctx.
func UsageRecordFromContext(ctx context.Context) *UsageRecordContext {
	if ctx == nil {
		return nil
	}
	v, _ := ctx.Value(usageRecordContextKey{}).(*UsageRecordContext)
	return v
}

// TokenUsageRecorder allows callers to track token/billing usage emitted by providers.
// Receives full request/response/error for both success and failure recording.
type TokenUsageRecorder interface {
	RecordUsage(ctx context.Context, req *GenerateRequest, rsp *GenerateResponse, err error)
}

// TokenUsageRecorderFunc is an adapter to allow ordinary functions to be used as recorders.
type TokenUsageRecorderFunc func(ctx context.Context, req *GenerateRequest, rsp *GenerateResponse, err error)

// RecordUsage implements TokenUsageRecorder.
func (f TokenUsageRecorderFunc) RecordUsage(ctx context.Context, req *GenerateRequest, rsp *GenerateResponse, err error) {
	if f == nil {
		return
	}
	f(ctx, req, rsp, err)
}

type noopTokenUsageRecorder struct{}

func (noopTokenUsageRecorder) RecordUsage(context.Context, *GenerateRequest, *GenerateResponse, error) {}

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

// CoalesceImageProvider picks a registered image provider: prefers preferred, then huoshan when available.
// Uses a single read lock because nested GetImageProvider calls would deadlock on the same goroutine.
func (g *GenAPI) CoalesceImageProvider(preferred string) string {
	p := strings.TrimSpace(preferred)
	if p == "" {
		p = "huoshan"
	}
	if g == nil {
		return p
	}
	g.mu.RLock()
	defer g.mu.RUnlock()
	n := normalizeProviderName(p)
	if _, ok := g.imageProviders[n]; ok && !mediaGenerationDenied(n) {
		return n
	}
	huoshanKey := normalizeProviderName(MediaGenerationProvider)
	if n != huoshanKey {
		if _, ok := g.imageProviders[huoshanKey]; ok {
			return huoshanKey
		}
	}
	return n
}

// HuoshanInternalClient returns the Huoshan Ark client for chat / multimodal text APIs, or nil if Huoshan is not registered.
func (g *GenAPI) HuoshanInternalClient() *huoshanprovider.Client {
	g.mu.RLock()
	defer g.mu.RUnlock()
	p := g.providers[normalizeProviderName("huoshan")]
	if p == nil {
		return nil
	}
	hp, ok := p.(*huoshanProvider)
	if !ok || hp == nil {
		return nil
	}
	return hp.client
}

// GetVideoProvider returns the video provider registered under the given name.
func (g *GenAPI) GetVideoProvider(name string) VideoProvider {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.videoProviders[normalizeProviderName(name)]
}

// CoalesceVideoProvider picks a registered video provider: prefers non-empty preferred, then huoshan when available.
// Uses a single read lock because nested GetVideoProvider calls would deadlock on the same goroutine.
func (g *GenAPI) CoalesceVideoProvider(preferred string) string {
	p := strings.TrimSpace(preferred)
	if p == "" {
		p = "huoshan"
	}
	if g == nil {
		return p
	}
	g.mu.RLock()
	defer g.mu.RUnlock()
	n := normalizeProviderName(p)
	if _, ok := g.videoProviders[n]; ok && !mediaGenerationDenied(n) {
		return n
	}
	huoshanKey := normalizeProviderName(MediaGenerationProvider)
	if n != huoshanKey {
		if _, ok := g.videoProviders[huoshanKey]; ok {
			return huoshanKey
		}
	}
	return n
}

// GenerateImage runs an image generation workflow on the selected provider.
func (g *GenAPI) GenerateImage(ctx context.Context, providerName string, req *GenerateRequest) (*GenerateResponse, error) {
	if req == nil {
		return nil, errors.New("generate request cannot be nil")
	}
	providerName, rerouted := g.resolveImageGenerationProvider(providerName)
	provider := g.GetImageProvider(providerName)
	if provider == nil {
		return nil, fmt.Errorf("image provider %s not registered", providerName)
	}

	cloned := req.Clone()
	if rerouted {
		dropForeignMediaModel(cloned)
	}
	if cloned.Operation == OperationUnknown {
		cloned.Operation = defaultImageOperation(cloned)
	}

	// Log operation start
	logOperation(ctx, "GenerateImage", LogFields{
		Provider:  providerName,
		Operation: cloned.Operation,
		MediaType: MediaTypeImage,
		UserID:    cloned.UserID,
	})

	start := time.Now()
	rsp, err := provider.GenerateImage(ctx, cloned)
	duration := time.Since(start)
	rsp = g.normalizeResponse(rsp, provider.Name(), cloned, MediaTypeImage, start)

	// Log operation complete
	logOperationComplete(ctx, "GenerateImage", LogFields{
		Provider:  providerName,
		Operation: cloned.Operation,
		MediaType: MediaTypeImage,
		TaskID:    rsp.TaskID,
		Duration:  duration,
		Error:     err,
	})

	g.recordUsage(ctx, cloned, rsp, err)
	return rsp, err
}

// GenerateVideo runs a video generation workflow on the selected provider.
func (g *GenAPI) GenerateVideo(ctx context.Context, providerName string, req *GenerateRequest) (*GenerateResponse, error) {
	if req == nil {
		return nil, errors.New("generate request cannot be nil")
	}
	providerName, rerouted := g.resolveVideoGenerationProvider(providerName)
	provider := g.GetVideoProvider(providerName)
	if provider == nil {
		return nil, fmt.Errorf("video provider %s not registered", providerName)
	}

	cloned := req.Clone()
	if rerouted {
		dropForeignMediaModel(cloned)
	}
	if cloned.Operation == OperationUnknown {
		cloned.Operation = defaultVideoOperation(cloned)
	}

	// Log operation start
	logOperation(ctx, "GenerateVideo", LogFields{
		Provider:  providerName,
		Operation: cloned.Operation,
		MediaType: MediaTypeVideo,
		UserID:    cloned.UserID,
	})

	start := time.Now()
	rsp, err := provider.Generate(ctx, cloned)
	duration := time.Since(start)
	rsp = g.normalizeResponse(rsp, provider.Name(), cloned, MediaTypeVideo, start)

	// Log operation complete
	logOperationComplete(ctx, "GenerateVideo", LogFields{
		Provider:  providerName,
		Operation: cloned.Operation,
		MediaType: MediaTypeVideo,
		TaskID:    rsp.TaskID,
		Duration:  duration,
		Error:     err,
	})

	g.recordUsage(ctx, cloned, rsp, err)
	return rsp, err
}

// GetVideoStatus retrieves the status of an async video generation task.
func (g *GenAPI) GetVideoStatus(ctx context.Context, providerName, taskID string) (*GenerateResponse, error) {
	provider := g.GetVideoProvider(providerName)
	if provider == nil {
		return nil, fmt.Errorf("video provider %s not registered", providerName)
	}

	logDebug(ctx, "GetVideoStatus", "provider", providerName, "task_id", taskID)

	start := time.Now()
	rsp, err := provider.GetVideoStatus(ctx, taskID)
	duration := time.Since(start)
	rsp = g.normalizeResponse(rsp, provider.Name(), nil, MediaTypeVideo, start)

	if err != nil {
		logError(ctx, "GetVideoStatus failed", "provider", providerName, "task_id", taskID, "error", err, "duration_ms", duration.Milliseconds())
	}
	g.recordUsage(ctx, nil, rsp, err)
	return rsp, err
}

// GetImageStatus retrieves the status of an async image generation task.
// This is useful for providers like Qwen where image generation is asynchronous.
func (g *GenAPI) GetImageStatus(ctx context.Context, providerName, taskID string) (*GenerateResponse, error) {
	g.mu.RLock()
	provider := g.providers[normalizeProviderName(providerName)]
	g.mu.RUnlock()

	if provider == nil {
		return nil, fmt.Errorf("provider %s not registered", providerName)
	}

	fetcher, ok := provider.(ImageStatusFetcher)
	if !ok {
		return nil, fmt.Errorf("provider %s does not support image status fetching", providerName)
	}

	logDebug(ctx, "GetImageStatus", "provider", providerName, "task_id", taskID)

	start := time.Now()
	rsp, err := fetcher.GetImageStatus(ctx, taskID)
	duration := time.Since(start)
	rsp = g.normalizeResponse(rsp, providerName, nil, MediaTypeImage, start)

	if err != nil {
		logError(ctx, "GetImageStatus failed", "provider", providerName, "task_id", taskID, "error", err, "duration_ms", duration.Milliseconds())
	}
	g.recordUsage(ctx, nil, rsp, err)
	return rsp, err
}

// WaitForImage polls the image status until it reaches a terminal state or context is cancelled.
func (g *GenAPI) WaitForImage(ctx context.Context, providerName, taskID string, pollInterval time.Duration) (*GenerateResponse, error) {
	if pollInterval <= 0 {
		pollInterval = 3 * time.Second
	}

	// Check immediately first
	rsp, err := g.GetImageStatus(ctx, providerName, taskID)
	if err != nil {
		return nil, err
	}
	if IsTerminalStatus(rsp.Status) {
		return rsp, nil
	}

	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-ticker.C:
			rsp, err := g.GetImageStatus(ctx, providerName, taskID)
			if err != nil {
				return nil, err
			}
			if IsTerminalStatus(rsp.Status) {
				return rsp, nil
			}
		}
	}
}

// DownloadVideo retrieves the video binary data for a completed task.
func (g *GenAPI) DownloadVideo(ctx context.Context, providerName, taskID string) ([]byte, error) {
	g.mu.RLock()
	provider := g.providers[normalizeProviderName(providerName)]
	g.mu.RUnlock()

	if provider == nil {
		return nil, fmt.Errorf("provider %s not registered", providerName)
	}

	downloader, ok := provider.(VideoDownloader)
	if !ok {
		return nil, fmt.Errorf("provider %s does not support video download", providerName)
	}

	logDebug(ctx, "DownloadVideo", "provider", providerName, "task_id", taskID)

	start := time.Now()
	data, err := downloader.DownloadVideo(ctx, taskID)
	duration := time.Since(start)

	if err != nil {
		logError(ctx, "DownloadVideo failed", "provider", providerName, "task_id", taskID, "error", err, "duration_ms", duration.Milliseconds())
	} else {
		logDebug(ctx, "DownloadVideo completed", "provider", providerName, "task_id", taskID, "size_bytes", len(data), "duration_ms", duration.Milliseconds())
	}

	return data, err
}

// WaitForVideo polls the video status until it reaches a terminal state or context is cancelled.
// The pollInterval specifies how often to check the status.
func (g *GenAPI) WaitForVideo(ctx context.Context, providerName, taskID string, pollInterval time.Duration) (*GenerateResponse, error) {
	if pollInterval <= 0 {
		pollInterval = 5 * time.Second
	}

	// Check immediately first
	rsp, err := g.GetVideoStatus(ctx, providerName, taskID)
	if err != nil {
		return nil, err
	}
	if IsTerminalStatus(rsp.Status) {
		return rsp, nil
	}

	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-ticker.C:
			rsp, err := g.GetVideoStatus(ctx, providerName, taskID)
			if err != nil {
				return nil, err
			}
			if IsTerminalStatus(rsp.Status) {
				return rsp, nil
			}
		}
	}
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

func (g *GenAPI) recordUsage(ctx context.Context, req *GenerateRequest, rsp *GenerateResponse, err error) {
	if g.usageRecorder == nil {
		return
	}
	// Record both success and failure for analytics; skip only when we have nothing to record
	if rsp == nil && err == nil {
		return
	}
	g.usageRecorder.RecordUsage(ctx, req, rsp, err)
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
