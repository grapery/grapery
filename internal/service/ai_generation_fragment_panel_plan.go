package service

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/grapestree/fgrapery/grapery/internal/common"
	"github.com/grapestree/fgrapery/grapery/internal/domain"
	huoshanark "github.com/grapestree/fgrapery/grapery/internal/genai/providers/huoshan"
	"go.uber.org/zap"
	"google.golang.org/genai"
)

const fragmentPanelPlanGeminiModel = "gemini-2.5-flash"

// GenerateFragmentPanelPlanRequest multimodal plan step for fragment panel generation.
type GenerateFragmentPanelPlanRequest struct {
	UserID            string
	ReferenceImageURL string
	UserInput         string
	Style             string
	PanelCount        int
	RelatedEntityID   string
	RelatedEntityType string
	Metadata          map[string]interface{}
	// UserRegion 与 FragmentPanelGenerationRequest 一致，服务端写入；用于在 PlanProvider 非法时归一化文本侧路由。
	UserRegion string
	// PlanProvider "huoshan" (国内默认) or "gemini"（海外用户）.
	PlanProvider string
}

// GenerateFragmentPanelPlanResult Step1 output + usage.
type GenerateFragmentPanelPlanResult struct {
	Plan       []domain.FragmentPanelPlanItem
	TokensUsed int
	DurationMs int64
	Model      string
	Provider   string // "gemini" | "huoshan"
}

type fragmentPanelPlanEnvelope struct {
	Panels []domain.FragmentPanelPlanItem `json:"panels"`
}

// GenerateFragmentPanelPlan runs Step1: reference + text → JSON panel plan (Huoshan 或 Gemini).
func (s *AIGenerationService) GenerateFragmentPanelPlan(ctx context.Context, req *GenerateFragmentPanelPlanRequest) (*GenerateFragmentPanelPlanResult, error) {
	if req.PanelCount < 1 || req.PanelCount > 9 {
		return nil, fmt.Errorf("panelCount must be 1-9")
	}
	refURL := strings.TrimSpace(req.ReferenceImageURL)
	if refURL == "" {
		return nil, fmt.Errorf("reference image URL is required")
	}

	rawPlan := strings.ToLower(strings.TrimSpace(req.PlanProvider))
	prov := NormalizeTextPlanProvider(req.PlanProvider, req.UserRegion, s)
	if rawPlan != "" && rawPlan != "gemini" && rawPlan != "huoshan" {
		s.logger.Warn("fragment_panel_plan: invalid PlanProvider for text, resolved via user region",
			zap.String("requested", rawPlan),
			zap.String("resolved", prov),
			zap.String("userRegion", req.UserRegion))
	}
	switch prov {
	case "gemini":
		return s.generateFragmentPanelPlanGemini(ctx, req, refURL)
	default:
		return s.generateFragmentPanelPlanHuoshan(ctx, req, refURL)
	}
}

func (s *AIGenerationService) generateFragmentPanelPlanGemini(ctx context.Context, req *GenerateFragmentPanelPlanRequest, refURL string) (*GenerateFragmentPanelPlanResult, error) {
	if s.geminiClient == nil {
		return nil, fmt.Errorf("Gemini client not configured")
	}

	startTime := time.Now()
	estimatedTokens := 8000
	requestID := fmt.Sprintf("panel_plan_gemini_%s_%d", req.UserID, time.Now().UnixNano())
	lockKey := fmt.Sprintf("ai_generation_lock:text:%s", req.UserID)

	var reservation *QuotaReservation
	var lockAcquired bool

	if s.enableQuotaReservation && s.redisDistributedLock != nil {
		var err error
		lockAcquired, err = s.redisDistributedLock.AcquireLock(ctx, lockKey, requestID, 30*time.Second)
		if err != nil {
			return nil, fmt.Errorf("failed to acquire lock: %w", err)
		}
		if !lockAcquired {
			return nil, fmt.Errorf("another AI generation request is in progress, please try again later")
		}
		defer func() {
			if lockAcquired {
				s.redisDistributedLock.ReleaseLock(ctx, lockKey, requestID)
			}
		}()
	}

	if s.enableQuotaReservation && s.quotaReservation != nil {
		metadata := map[string]interface{}{
			"model":     fragmentPanelPlanGeminiModel,
			"type":      "fragment_panel_plan",
			"requestID": requestID,
		}
		var err error
		reservation, err = s.quotaReservation.ReserveQuota(ctx, req.UserID, estimatedTokens, "ai_text_generation", metadata)
		if err != nil {
			return nil, fmt.Errorf("failed to reserve quota: %w", err)
		}
	}

	defer func() {
		if s.quotaReservation != nil && reservation != nil && reservation.Status == string(common.StatusPending) {
			if err := s.quotaReservation.ReleaseQuota(ctx, reservation.ReservationID); err != nil {
				s.logger.Error("failed to release quota reservation", zap.Error(err))
			}
		}
	}()

	promptSummary := fmt.Sprintf("fragment_panel_plan(gemini) panels=%d style=%s", req.PanelCount, truncateString(req.Style, 32))
	record := &domain.AIGenerationRecord{
		UserID:            req.UserID,
		Type:              "text",
		Provider:          "gemini",
		Model:             fragmentPanelPlanGeminiModel,
		OriginalPrompt:    promptSummary,
		Status:            domain.AITaskStatusPending,
		RelatedEntityID:   req.RelatedEntityID,
		RelatedEntityType: req.RelatedEntityType,
		Metadata:          req.Metadata,
		CreatedAt:         startTime.Unix(),
		OutputResult:      "{}",
	}
	inputParams := map[string]interface{}{
		"referenceImageUrl": truncateString(refURL, 200),
		"userInput":         truncateString(req.UserInput, 500),
		"style":             req.Style,
		"panelCount":        req.PanelCount,
		"planProvider":      "gemini",
	}
	if b, err := json.Marshal(inputParams); err == nil {
		record.InputParams = string(b)
	}
	if err := s.repo.CreateAIGenerationRecord(ctx, record); err != nil {
		return nil, fmt.Errorf("failed to create AI record: %w", err)
	}
	processingTime := time.Now().Unix()
	record.Status = domain.AITaskStatusProcessing
	record.StartedAt = &processingTime
	_ = s.repo.UpdateAIGenerationRecord(ctx, record)

	imageData, mimeType, err := downloadImageFromURL(ctx, refURL)
	if err != nil {
		s.failPanelPlanRecord(ctx, record, startTime, err)
		return nil, fmt.Errorf("download reference image: %w", err)
	}
	imgPart, err := encodeImagePart(imageData, mimeType)
	if err != nil {
		s.failPanelPlanRecord(ctx, record, startTime, err)
		return nil, fmt.Errorf("encode reference image: %w", err)
	}

	userText := buildFragmentPanelPlanUserPrompt(req.UserInput, req.Style, req.PanelCount)
	contents := []*genai.Content{{
		Role: genai.RoleUser,
		Parts: []*genai.Part{
			genai.NewPartFromText("User reference image. First infer: main subjects/objects, environment, spatial layout, lighting, color palette, mood, and what the user likely cares about. Treat this as a story anchor (world + identity + atmosphere), not as a frame to copy pixel-by-pixel in every panel."),
			imgPart,
			genai.NewPartFromText(userText),
		},
	}}

	temp := float32(0.35)
	maxTok := int32(8192)
	config := &genai.GenerateContentConfig{
		Temperature:      &temp,
		MaxOutputTokens:  maxTok,
		ResponseMIMEType: "application/json",
	}

	resp, err := s.geminiClient.SDK().Models.GenerateContent(ctx, fragmentPanelPlanGeminiModel, contents, config)
	completedTime := time.Now()
	durationMs := completedTime.Sub(startTime).Milliseconds()

	if err != nil {
		s.failPanelPlanRecord(ctx, record, startTime, err)
		return nil, fmt.Errorf("Gemini panel plan failed: %w", err)
	}

	responseText := strings.TrimSpace(resp.Text())
	if responseText == "" {
		err := fmt.Errorf("empty model response")
		s.failPanelPlanRecord(ctx, record, startTime, err)
		return nil, err
	}

	plan, err := parseFragmentPanelPlanJSON(responseText, req.PanelCount)
	if err != nil {
		s.failPanelPlanRecord(ctx, record, startTime, err)
		return nil, err
	}

	record.Status = domain.AITaskStatusCompleted
	record.Progress = 100
	record.DurationMs = durationMs
	completedUnix := completedTime.Unix()
	record.CompletedAt = &completedUnix
	if resp != nil && resp.UsageMetadata != nil {
		record.InputTokens = int(resp.UsageMetadata.PromptTokenCount)
		record.OutputTokens = int(resp.UsageMetadata.CandidatesTokenCount)
		record.TotalTokens = int(resp.UsageMetadata.TotalTokenCount)
	}
	out := map[string]interface{}{"panelCount": len(plan), "tokens": record.TotalTokens}
	if ob, err := json.Marshal(out); err == nil {
		record.OutputResult = string(ob)
	}
	_ = s.repo.UpdateAIGenerationRecord(ctx, record)

	if s.enableQuotaReservation && s.quotaReservation != nil && reservation != nil && reservation.Status == string(common.StatusPending) {
		actualTokens := record.TotalTokens
		if actualTokens == 0 {
			actualTokens = 1
		}
		if err := s.quotaReservation.ConfirmQuota(ctx, reservation.ReservationID, actualTokens); err != nil {
			s.logger.Error("failed to confirm quota reservation", zap.Error(err))
		} else {
			reservation = nil
		}
	}

	if s.metrics != nil {
		s.metrics.RecordAIGeneration("gemini", "text")
	}

	return &GenerateFragmentPanelPlanResult{
		Plan:       plan,
		TokensUsed: record.TotalTokens,
		DurationMs: durationMs,
		Model:      fragmentPanelPlanGeminiModel,
		Provider:   "gemini",
	}, nil
}

func (s *AIGenerationService) generateFragmentPanelPlanHuoshan(ctx context.Context, req *GenerateFragmentPanelPlanRequest, refURL string) (*GenerateFragmentPanelPlanResult, error) {
	if s.genAPI == nil || s.genAPI.HuoshanInternalClient() == nil {
		return nil, fmt.Errorf("Huoshan client not configured")
	}
	hc := s.genAPI.HuoshanInternalClient()

	startTime := time.Now()
	estimatedTokens := 8000
	requestID := fmt.Sprintf("panel_plan_huoshan_%s_%d", req.UserID, time.Now().UnixNano())
	lockKey := fmt.Sprintf("ai_generation_lock:text:%s", req.UserID)

	var reservation *QuotaReservation
	var lockAcquired bool

	if s.enableQuotaReservation && s.redisDistributedLock != nil {
		var err error
		lockAcquired, err = s.redisDistributedLock.AcquireLock(ctx, lockKey, requestID, 30*time.Second)
		if err != nil {
			return nil, fmt.Errorf("failed to acquire lock: %w", err)
		}
		if !lockAcquired {
			return nil, fmt.Errorf("another AI generation request is in progress, please try again later")
		}
		defer func() {
			if lockAcquired {
				s.redisDistributedLock.ReleaseLock(ctx, lockKey, requestID)
			}
		}()
	}

	if s.enableQuotaReservation && s.quotaReservation != nil {
		metadata := map[string]interface{}{
			"type":      "fragment_panel_plan",
			"requestID": requestID,
			"provider":  "huoshan",
		}
		var err error
		reservation, err = s.quotaReservation.ReserveQuota(ctx, req.UserID, estimatedTokens, "ai_text_generation", metadata)
		if err != nil {
			return nil, fmt.Errorf("failed to reserve quota: %w", err)
		}
	}

	defer func() {
		if s.quotaReservation != nil && reservation != nil && reservation.Status == string(common.StatusPending) {
			if err := s.quotaReservation.ReleaseQuota(ctx, reservation.ReservationID); err != nil {
				s.logger.Error("failed to release quota reservation", zap.Error(err))
			}
		}
	}()

	promptSummary := fmt.Sprintf("fragment_panel_plan(huoshan) panels=%d style=%s", req.PanelCount, truncateString(req.Style, 32))
	record := &domain.AIGenerationRecord{
		UserID:            req.UserID,
		Type:              "text",
		Provider:          "huoshan",
		Model:             "ark-multimodal",
		OriginalPrompt:    promptSummary,
		Status:            domain.AITaskStatusPending,
		RelatedEntityID:   req.RelatedEntityID,
		RelatedEntityType: req.RelatedEntityType,
		Metadata:          req.Metadata,
		CreatedAt:         startTime.Unix(),
		OutputResult:      "{}",
	}
	inputParams := map[string]interface{}{
		"referenceImageUrl": truncateString(refURL, 200),
		"userInput":         truncateString(req.UserInput, 500),
		"style":             req.Style,
		"panelCount":        req.PanelCount,
		"planProvider":      "huoshan",
	}
	if b, err := json.Marshal(inputParams); err == nil {
		record.InputParams = string(b)
	}
	if err := s.repo.CreateAIGenerationRecord(ctx, record); err != nil {
		return nil, fmt.Errorf("failed to create AI record: %w", err)
	}
	processingTime := time.Now().Unix()
	record.Status = domain.AITaskStatusProcessing
	record.StartedAt = &processingTime
	_ = s.repo.UpdateAIGenerationRecord(ctx, record)

	userText := buildFragmentPanelPlanUserPrompt(req.UserInput, req.Style, req.PanelCount)
	hresp, err := hc.GenerateText(ctx, &huoshanark.TextGenerationRequest{
		Prompt:       userText,
		ImageURLs:    []string{refURL},
		JSONResponse: true,
		MaxTokens:    8192,
		Temperature:  0.35,
	})
	completedTime := time.Now()
	durationMs := completedTime.Sub(startTime).Milliseconds()

	if err != nil {
		s.failPanelPlanRecord(ctx, record, startTime, err)
		return nil, fmt.Errorf("Huoshan panel plan failed: %w", err)
	}
	if hresp == nil {
		err := fmt.Errorf("nil Huoshan response")
		s.failPanelPlanRecord(ctx, record, startTime, err)
		return nil, err
	}

	responseText := strings.TrimSpace(hresp.Text)
	if responseText == "" {
		err := fmt.Errorf("empty model response")
		s.failPanelPlanRecord(ctx, record, startTime, err)
		return nil, err
	}

	plan, err := parseFragmentPanelPlanJSON(responseText, req.PanelCount)
	if err != nil {
		s.failPanelPlanRecord(ctx, record, startTime, err)
		return nil, err
	}

	modelName := strings.TrimSpace(hresp.Model)
	if modelName == "" {
		modelName = "huoshan-ark"
	}
	record.Model = modelName

	record.Status = domain.AITaskStatusCompleted
	record.Progress = 100
	record.DurationMs = durationMs
	completedUnix := completedTime.Unix()
	record.CompletedAt = &completedUnix
	record.InputTokens = hresp.InputTokens
	record.OutputTokens = hresp.OutputTokens
	record.TotalTokens = hresp.TotalTokens
	if record.TotalTokens == 0 {
		record.TotalTokens = record.InputTokens + record.OutputTokens
	}
	out := map[string]interface{}{"panelCount": len(plan), "tokens": record.TotalTokens}
	if ob, err := json.Marshal(out); err == nil {
		record.OutputResult = string(ob)
	}
	_ = s.repo.UpdateAIGenerationRecord(ctx, record)

	if s.enableQuotaReservation && s.quotaReservation != nil && reservation != nil && reservation.Status == string(common.StatusPending) {
		actualTokens := record.TotalTokens
		if actualTokens == 0 {
			actualTokens = 1
		}
		if err := s.quotaReservation.ConfirmQuota(ctx, reservation.ReservationID, actualTokens); err != nil {
			s.logger.Error("failed to confirm quota reservation", zap.Error(err))
		} else {
			reservation = nil
		}
	}

	if s.metrics != nil {
		s.metrics.RecordAIGeneration("huoshan", "text")
	}

	return &GenerateFragmentPanelPlanResult{
		Plan:       plan,
		TokensUsed: record.TotalTokens,
		DurationMs: durationMs,
		Model:      modelName,
		Provider:   "huoshan",
	}, nil
}

func parseFragmentPanelPlanJSON(responseText string, panelCount int) ([]domain.FragmentPanelPlanItem, error) {
	var env fragmentPanelPlanEnvelope
	if err := json.Unmarshal([]byte(responseText), &env); err != nil {
		cleaned := extractJSON(responseText)
		if err2 := json.Unmarshal([]byte(cleaned), &env); err2 != nil {
			return nil, fmt.Errorf("parse panel plan JSON: %w", err2)
		}
	}
	return normalizeFragmentPanelPlan(env.Panels, panelCount)
}

func (s *AIGenerationService) failPanelPlanRecord(ctx context.Context, record *domain.AIGenerationRecord, startTime time.Time, genErr error) {
	record.Status = domain.AITaskStatusFailed
	record.ErrorMessage = genErr.Error()
	record.DurationMs = time.Since(startTime).Milliseconds()
	completedUnix := time.Now().Unix()
	record.CompletedAt = &completedUnix
	_ = s.repo.UpdateAIGenerationRecord(ctx, record)
}

func buildFragmentPanelPlanUserPrompt(userInput, style string, panelCount int) string {
	ui := strings.TrimSpace(userInput)
	st := strings.TrimSpace(style)
	if st == "" {
		st = "unspecified"
	}
	return fmt.Sprintf(`You are the director for a %d-panel visual story (comic / story fragments). The user attached a reference image AND text. Your job has two phases (do both before writing JSON):

PHASE 1 — UNDERSTAND THE IMAGE (mental checklist, do not output this as prose):
- Identify concrete entities: people/characters, objects, architecture, nature, props, text/signs if any.
- Scene: indoor/outdoor, time of day, weather, spatial relationships (foreground/background).
- Aesthetic signals: palette, lighting, camera feel (wide vs close), mood.
- What the user probably wants to preserve (e.g. same characters, same trip, same vibe) vs what can vary.

PHASE 2 — PLAN PANELS AROUND IMAGE + USER INTENT:
- Merge the user's text (Chinese or English) with your scene understanding. The story should EXTEND and DIVERSIFY from the reference: new moments, beats, angles, distances, actions — while keeping thematic continuity (same world, recognizable subjects where relevant).
- Do NOT treat every panel as "redraw the reference photo with small edits". Later panels should advance the narrative or show different facets of the same story.
- Panel 0 may establish the situation (can relate closely to the reference) but should still allow creative composition; panels 1..%d should clearly change shot type, action, or story time unless the user explicitly asks for repetition.

Visual style slug for downstream image models: %s

User text:
%s

Return ONLY valid JSON (no markdown, no commentary) with this exact shape:
{"panels":[{"index":0,"image_prompt":"English, for the image model: one panel, concrete visual description including camera/composition/lighting; emphasize creative expansion from the anchor image, not slavish duplication","caption":"Short Chinese caption for readers"}, ...]}

Hard rules:
- Exactly %d items in "panels", index 0..%d in order.
- image_prompt: English only; specific enough to draw; explicitly call for variation across panels (different framing, focal length, or story beat) where appropriate.
- caption: one concise Chinese line per panel.
- image_prompt must NOT instruct to "copy", "duplicate", or "recreate the reference exactly" for every panel; anchor identity/mood, but encourage diverse storytelling visuals.`,
		panelCount,
		panelCount-1,
		st,
		ui,
		panelCount,
		panelCount-1,
	)
}

func normalizeFragmentPanelPlan(raw []domain.FragmentPanelPlanItem, want int) ([]domain.FragmentPanelPlanItem, error) {
	if len(raw) != want {
		return nil, fmt.Errorf("plan has %d panels, want %d", len(raw), want)
	}
	panels := append([]domain.FragmentPanelPlanItem(nil), raw...)
	sort.Slice(panels, func(i, j int) bool { return panels[i].Index < panels[j].Index })
	out := make([]domain.FragmentPanelPlanItem, want)
	for i := 0; i < want; i++ {
		p := panels[i]
		if strings.TrimSpace(p.ImagePrompt) == "" {
			return nil, fmt.Errorf("panel %d has empty image_prompt", i)
		}
		p.Index = i
		out[i] = p
	}
	return out, nil
}
