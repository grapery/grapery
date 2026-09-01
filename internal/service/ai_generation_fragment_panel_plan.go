package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
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
	Language          string
	PanelCount        int
	RelatedEntityID   string
	RelatedEntityType string
	Metadata          map[string]interface{}
	// UserRegion 与 FragmentPanelGenerationRequest 一致，服务端写入；用于在 PlanProvider 非法时归一化文本侧路由。
	UserRegion string
	// PlanProvider 显式 "huoshan" 或 "gemini"；默认由服务端归一为火山优先。
	PlanProvider string
	// LayoutAddon 由 FragmentPanelGenerationService 从版式/对白选项生成，并入分镜规划提示。
	LayoutAddon string
	// VisualEvidence 是入口多模态视觉事实分析结果，用于避免分镜阶段重新臆测参考图。
	VisualEvidence []domain.FragmentVisualEvidence
}

// GenerateFragmentPanelPlanResult Step1 output + usage.
type GenerateFragmentPanelPlanResult struct {
	Plan        []domain.FragmentPanelPlanItem
	VisualBible *domain.FragmentVisualBible
	TokensUsed  int
	DurationMs  int64
	Model       string
	Provider    string // "gemini" | "huoshan"
}

func (s *AIGenerationService) recordPanelPlanPromptAudit(ctx context.Context, req *GenerateFragmentPanelPlanRequest, provider, model, systemPrompt, userPrompt, refURL, output string, temperature float64, maxTokens int, totalTokens int) {
	if s == nil || s.repo == nil || req == nil || strings.TrimSpace(userPrompt) == "" {
		return
	}
	entityType := strings.TrimSpace(req.RelatedEntityType)
	if entityType == "" {
		entityType = "fragment_panel_generation"
	}
	tokenUsage := map[string]any{"totalTokens": totalTokens}
	metadata := map[string]any{
		"source":       "fragment_panel_plan",
		"panelCount":   req.PanelCount,
		"style":        req.Style,
		"planProvider": provider,
	}
	for k, v := range req.Metadata {
		metadata[k] = v
	}
	record := &domain.AIPromptAuditRecord{
		ID:                    uuid.NewString(),
		RelatedEntityType:     entityType,
		RelatedEntityID:       req.RelatedEntityID,
		Step:                  "fragment_panel_plan",
		PromptKind:            "multimodal_json_plan",
		PromptTemplateVersion: "fragment_panel_plan_v1",
		FullPromptHash:        stableSHA256(systemPrompt + "\n" + userPrompt),
		SystemPrompt:          systemPrompt,
		UserPrompt:            userPrompt,
		ReferencePreamble:     systemPrompt,
		FinalPrompt:           userPrompt,
		ReferenceImageURLs:    fragmentPrefillHTTPImageURLs([]string{refURL}, 1),
		Provider:              provider,
		Model:                 model,
		Temperature:           temperature,
		MaxTokens:             maxTokens,
		Output:                output,
		TokenUsageJSON:        mustJSON(tokenUsage, "{}"),
		MetadataJSON:          mustJSON(metadata, "{}"),
		CreatedAt:             time.Now().Unix(),
	}
	if err := s.repo.CreateAIPromptAuditRecord(ctx, record); err != nil {
		s.logger.Warn("failed to create fragment panel plan prompt audit",
			zap.String("relatedEntityID", req.RelatedEntityID),
			zap.Error(err))
	}
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

	userText := buildFragmentPanelPlanUserPrompt(req.UserInput, req.Style, req.PanelCount, panelPlanLayoutWithVisualEvidence(req.LayoutAddon, req.VisualEvidence), req.Language)
	record.OriginalPrompt = userText
	record.SystemPrompt = fragmentPanelGeminiReferenceImagePreamble
	_ = s.repo.UpdateAIGenerationRecord(ctx, record)
	contents := []*genai.Content{{
		Role: genai.RoleUser,
		Parts: []*genai.Part{
			genai.NewPartFromText(fragmentPanelGeminiReferenceImagePreamble),
			imgPart,
			genai.NewPartFromText(userText),
		},
	}}

	temp := float32(0.35)
	maxTok := int32(8192)
	// 5 格条漫版式时 JSON 体量大，8192 输出易被截断导致 panels 数量不足或 JSON 残缺。
	if req.PanelCount >= 5 {
		maxTok = 16384
	}
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

	plan, visualBible, err := parseFragmentPanelPlanJSON(responseText, req.PanelCount)
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
	record.OutputResult = responseText
	_ = s.repo.UpdateAIGenerationRecord(ctx, record)
	s.recordPanelPlanPromptAudit(ctx, req, "gemini", fragmentPanelPlanGeminiModel, fragmentPanelGeminiReferenceImagePreamble, userText, refURL, responseText, float64(temp), int(maxTok), record.TotalTokens)

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
		Plan:        plan,
		VisualBible: visualBible,
		TokensUsed:  record.TotalTokens,
		DurationMs:  durationMs,
		Model:       fragmentPanelPlanGeminiModel,
		Provider:    "gemini",
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

	userText := buildFragmentPanelPlanUserPrompt(req.UserInput, req.Style, req.PanelCount, panelPlanLayoutWithVisualEvidence(req.LayoutAddon, req.VisualEvidence), req.Language)
	record.OriginalPrompt = userText
	_ = s.repo.UpdateAIGenerationRecord(ctx, record)
	maxHuoshanTok := 8192
	if req.PanelCount >= 5 {
		maxHuoshanTok = 16384
	}
	var hresp *huoshanark.TextGenerationResponse
	var err error
	const panelPlanHuoshanMaxAttempts = 3
retryHuoshanPanelPlan:
	for attempt := 1; attempt <= panelPlanHuoshanMaxAttempts; attempt++ {
		hresp, err = hc.GenerateText(ctx, &huoshanark.TextGenerationRequest{
			Prompt:       userText,
			ImageURLs:    []string{refURL},
			JSONResponse: true,
			MaxTokens:    maxHuoshanTok,
			Temperature:  0.35,
		})
		if err == nil || !isHuoshanPanelPlanTimeoutLike(err) {
			break retryHuoshanPanelPlan
		}
		if attempt < panelPlanHuoshanMaxAttempts {
			if s.logger != nil {
				s.logger.Warn("Huoshan panel plan request timed out, retrying",
					zap.Int("attempt", attempt),
					zap.Int("maxAttempts", panelPlanHuoshanMaxAttempts),
					zap.Error(err))
			}
			select {
			case <-ctx.Done():
				err = ctx.Err()
				break retryHuoshanPanelPlan
			case <-time.After(time.Duration(attempt) * time.Second):
			}
		}
	}
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

	plan, visualBible, err := parseFragmentPanelPlanJSON(responseText, req.PanelCount)
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
	record.OutputResult = responseText
	_ = s.repo.UpdateAIGenerationRecord(ctx, record)
	s.recordPanelPlanPromptAudit(ctx, req, "huoshan", modelName, "", userText, refURL, responseText, 0.35, maxHuoshanTok, record.TotalTokens)

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
		Plan:        plan,
		VisualBible: visualBible,
		TokensUsed:  record.TotalTokens,
		DurationMs:  durationMs,
		Model:       modelName,
		Provider:    "huoshan",
	}, nil
}

func parseFragmentPanelPlanJSON(responseText string, panelCount int) ([]domain.FragmentPanelPlanItem, *domain.FragmentVisualBible, error) {
	blob := strings.TrimSpace(responseText)
	if blob == "" {
		return nil, nil, fmt.Errorf("parse panel plan JSON: empty response")
	}

	// Gemini / Ark 偶有额外前后缀或非严格 JSON。注意：通用的 `extractJSON` 在遇到 JSON 数组
	// （如 `[ {...} ]`）时只会截取第一个 `{` 到最后一个 `}`，会把数组裁成单体对象，
	// 进而误走「缺 panels」分支；因此只有当顶层不以 `[` 开头时才套用 `extractJSON`。
	variants := []string{blob}
	tr := strings.TrimSpace(blob)
	if tr != "" && !strings.HasPrefix(tr, "[") {
		if j := strings.TrimSpace(extractJSON(tr)); j != "" && j != tr {
			variants = append(variants, j)
		}
	}

	var lastErr error
	for _, v := range variants {
		plan, vb, err := parseFragmentPanelPlanJSONOnce(strings.TrimSpace(v), panelCount)
		if err == nil {
			return plan, vb, nil
		}
		lastErr = err
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("unable to parse")
	}
	return nil, nil, fmt.Errorf("parse panel plan JSON: %w", lastErr)
}

func (s *AIGenerationService) failPanelPlanRecord(ctx context.Context, record *domain.AIGenerationRecord, startTime time.Time, genErr error) {
	record.Status = domain.AITaskStatusFailed
	record.ErrorMessage = genErr.Error()
	record.DurationMs = time.Since(startTime).Milliseconds()
	completedUnix := time.Now().Unix()
	record.CompletedAt = &completedUnix
	_ = s.repo.UpdateAIGenerationRecord(ctx, record)
}

func normalizeFragmentPanelPlan(raw []domain.FragmentPanelPlanItem, want int) ([]domain.FragmentPanelPlanItem, error) {
	panels := append([]domain.FragmentPanelPlanItem(nil), raw...)
	sort.Slice(panels, func(i, j int) bool { return panels[i].Index < panels[j].Index })
	if len(panels) > want {
		panels = panels[:want]
	}
	if len(panels) != want {
		return nil, fmt.Errorf("plan has %d panels, want %d", len(panels), want)
	}
	out := make([]domain.FragmentPanelPlanItem, want)
	for i := 0; i < want; i++ {
		p := panels[i]
		if strings.TrimSpace(p.ImagePrompt) == "" {
			return nil, fmt.Errorf("panel %d has empty image_prompt", i)
		}
		p.Index = i
		if strings.TrimSpace(p.LayoutIntent) == "" {
			p.LayoutIntent = "single_image_story_composition"
		}
		if strings.TrimSpace(p.CompositionPlan) == "" {
			p.CompositionPlan = "Lay out this panel beat in one generated image—either one continuous scene or, when clearer storytelling, multiple clearly separated intra-image zones/sub-panels (gutters, reading order). Describe foreground, subject, background or each zone distinctly."
		}
		if strings.TrimSpace(p.ShotType) == "" {
			p.ShotType = "medium_shot"
		}
		if strings.TrimSpace(p.VisualHierarchy) == "" {
			p.VisualHierarchy = "Primary subject first, key story prop or action second, environment mood third."
		}
		out[i] = p
	}
	return out, nil
}

// isHuoshanPanelPlanTimeoutLike returns true when the error is a client-side wait timeout
// (e.g. http.Client.Timeout before response headers) or context deadline — worth one or more retries.
func isHuoshanPanelPlanTimeoutLike(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}
	var ne net.Error
	if errors.As(err, &ne) && ne.Timeout() {
		return true
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "context deadline exceeded") ||
		strings.Contains(msg, "timeout exceeded while awaiting headers") ||
		strings.Contains(msg, "client.timeout") ||
		(strings.Contains(msg, "timeout") && strings.Contains(msg, "deadline"))
}
