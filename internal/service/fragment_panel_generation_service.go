package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/grapestree/fgrapery/grapery/internal/common"
	"github.com/grapestree/fgrapery/grapery/internal/domain"
	"github.com/grapestree/fgrapery/grapery/internal/repository"
)

// ErrFragmentPanelTaskForbidden when task user does not match.
var ErrFragmentPanelTaskForbidden = errors.New("forbidden: task does not belong to user")

// defaultFragmentPanelTopicLabel 与 Voyager「故事碎片」占位一致（存库展示；非 i18n 键）。
const defaultFragmentPanelTopicLabel = "故事碎片"

func panelTopicForFragment(req domain.FragmentPanelGenerationRequest, existing string) string {
	t := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(req.Topic), "#"))
	if t != "" {
		return t
	}
	t = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(existing), "#"))
	if t != "" {
		return t
	}
	return defaultFragmentPanelTopicLabel
}

// ErrPanelGenerationResumeConflict when a resume is requested while the task is already processing.
var ErrPanelGenerationResumeConflict = errors.New("panel generation already in progress")

// ErrPanelGenerationNotResumable when resume preconditions are not met (completed, no plan, bad data, etc.).
var ErrPanelGenerationNotResumable = errors.New("panel generation task cannot be resumed")

// ErrPanelGenerationDraftResetFailed when the draft fragment could not be set back to generating after acquiring resume lock.
var ErrPanelGenerationDraftResetFailed = errors.New("draft reset failed after resume lock")

const panelMaxReferenceImages = 6

// FragmentPanelGenerationService orchestrates multi-panel reference-based fragment generation.
type FragmentPanelGenerationService struct {
	panelRepo            *repository.FragmentPanelGenerationRepository
	fragmentRepo         *repository.FragmentRepository
	repo                 domain.Repository // optional: load user region for Huoshan/Gemini routing
	defaultImageProvider string            // e.g. cfg.AI.ImageProvider; domestic panel images
	aiGen                *AIGenerationService
	aiService            *AIService // 锚点图、多参考出图包装、一致性检查（与普通碎片方案 B 对齐）；可为 nil 则跳过锚点与检查
	logger               *zap.Logger
	notify               *Service // optional: push + in-app when panel generation completes
}

// NewFragmentPanelGenerationService constructs the service.
func NewFragmentPanelGenerationService(
	panelRepo *repository.FragmentPanelGenerationRepository,
	fragmentRepo *repository.FragmentRepository,
	repo domain.Repository,
	defaultImageProvider string,
	aiGen *AIGenerationService,
	aiService *AIService,
	logger *zap.Logger,
) *FragmentPanelGenerationService {
	return &FragmentPanelGenerationService{
		panelRepo:            panelRepo,
		fragmentRepo:         fragmentRepo,
		repo:                 repo,
		defaultImageProvider: defaultImageProvider,
		aiGen:                aiGen,
		aiService:            aiService,
		logger:               logger,
	}
}

// SetNotify wires main Service for completion notifications (nil-safe).
func (s *FragmentPanelGenerationService) SetNotify(svc *Service) {
	s.notify = svc
}

// StartGeneration validates input, creates DB task + draft fragment, and runs the pipeline async.
func (s *FragmentPanelGenerationService) StartGeneration(ctx context.Context, userID string, req domain.FragmentPanelGenerationRequest) (*domain.FragmentPanelGenerationTask, error) {
	if strings.TrimSpace(req.ReferenceImageURL) == "" {
		return nil, fmt.Errorf("referenceImageUrl is required")
	}
	if strings.TrimSpace(req.UserInput) == "" {
		return nil, fmt.Errorf("userInput is required")
	}
	pc := req.PanelCount
	if pc == 0 {
		pc = 3
	}
	if pc < 1 || pc > 9 {
		return nil, fmt.Errorf("panelCount must be 1-9")
	}
	req.PanelCount = pc
	if strings.TrimSpace(req.Style) == "" {
		req.Style = "fantasy"
	}
	if strings.TrimSpace(req.Language) == "" {
		req.Language = inferGenerationLanguage(req.UserInput)
	} else {
		req.Language = normalizeGenerationLanguage(req.Language)
	}
	vis := domain.NormalizeFragmentVisibility(strings.TrimSpace(req.Visibility))
	if vis == "" {
		vis = domain.FragmentVisibilityPrivate
	}
	req.Visibility = vis

	ar := domain.NormalizeFragmentAspectRatio(strings.TrimSpace(req.AspectRatio))
	if ar == "" {
		ar = domain.FragmentAspectDefault
	}
	req.AspectRatio = ar

	if s.aiGen == nil {
		return nil, fmt.Errorf("AI generation service not configured")
	}

	if s.repo != nil {
		if st, err := s.repo.UserSettings(ctx, userID); err == nil && st != nil {
			req.UserRegion = st.Region
		}
	}

	taskID := uuid.New().String()
	nowMs := time.Now().UnixMilli()
	nowSec := time.Now().Unix()

	draft := &domain.Fragment{
		BaseModel: common.BaseModel{
			ID:        uuid.New().String(),
			CreatedAt: nowMs,
			UpdatedAt: nowMs,
		},
		UserID:          userID,
		CreatorID:       userID,
		Content:         "生成中…",
		MediaURLs:       []string{},
		Topic:           panelTopicForFragment(req, ""),
		Visibility:      domain.FragmentVisibilityPrivate,
		IsDraft:         true,
		SourceType:      string(domain.FragmentSourcePanelGeneration),
		SourceID:        taskID,
		AspectRatio:     ar,
		EngagementStats: common.EngagementStats{},
	}

	task := &domain.FragmentPanelGenerationTask{
		ID:              taskID,
		UserID:          userID,
		Status:          "pending",
		Progress:        0,
		CurrentStep:     "queued",
		Request:         req,
		DraftFragmentID: draft.ID,
		CreatedAt:       nowSec,
		UpdatedAt:       nowSec,
		Metrics:         &domain.FragmentPanelMetricsData{Steps: []domain.FragmentPanelStepMetric{}},
	}

	db := s.fragmentRepo.DB()
	if db == nil {
		return nil, fmt.Errorf("database not configured")
	}
	if err := db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := s.fragmentRepo.CreateWithTx(ctx, tx, draft); err != nil {
			return err
		}
		return s.panelRepo.CreateWithTx(ctx, tx, task)
	}); err != nil {
		return nil, fmt.Errorf("create draft fragment and panel task: %w", err)
	}

	go s.process(context.Background(), taskID)

	return task, nil
}

// GetTask returns the task if it exists and belongs to the user.
func (s *FragmentPanelGenerationService) GetTask(ctx context.Context, taskID, userID string) (*domain.FragmentPanelGenerationTask, error) {
	task, err := s.panelRepo.GetByID(ctx, taskID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
		return nil, err
	}
	if task.UserID != userID {
		return nil, ErrFragmentPanelTaskForbidden
	}
	return task, nil
}

// ResumeGeneration restarts a failed (or stuck pending) panel task from the first panel that has no image yet (or runs finalize only if all panels exist).
// Uses a conditional DB update so concurrent resume requests only one wins (others get ErrPanelGenerationResumeConflict).
func (s *FragmentPanelGenerationService) ResumeGeneration(ctx context.Context, userID, taskID string) (*domain.FragmentPanelGenerationTask, error) {
	task, err := s.GetTask(ctx, taskID, userID)
	if err != nil {
		return nil, err
	}
	switch task.Status {
	case "completed":
		return nil, fmt.Errorf("%w: 任务已完成", ErrPanelGenerationNotResumable)
	case "processing":
		return nil, ErrPanelGenerationResumeConflict
	case "failed", "pending":
		// ok
	default:
		return nil, fmt.Errorf("%w: 仅失败或排队中的任务可续跑", ErrPanelGenerationNotResumable)
	}

	// Stuck pending, never entered pipeline: restart full process (same as initial goroutine).
	if task.Status == "pending" && len(task.Plan) == 0 {
		acquired, err := s.panelRepo.TryAcquireResumeProcessing(ctx, taskID, userID, 5, "understanding_reference")
		if err != nil {
			return nil, fmt.Errorf("acquire resume lock: %w", err)
		}
		if !acquired {
			return nil, ErrPanelGenerationResumeConflict
		}
		if err := s.resetDraftGeneratingContent(ctx, task.DraftFragmentID); err != nil {
			_ = s.panelRepo.RevertProcessingToFailed(ctx, taskID, userID, fmt.Sprintf("恢复草稿失败: %v", err))
			return nil, fmt.Errorf("%w: %v", ErrPanelGenerationDraftResetFailed, err)
		}
		s.logger.Info("panel_generation_resume",
			zap.String("task_id", taskID),
			zap.String("user_id", userID),
			zap.String("mode", "full_restart_pending"),
			zap.Int("plan_len", 0),
			zap.Int("start_panel", 0),
		)
		go s.process(context.Background(), taskID)
		out, err := s.panelRepo.GetByID(ctx, taskID)
		if err != nil {
			return task, nil
		}
		return out, nil
	}

	// 规划阶段即失败（如 Ark 超时）时 Plan 为空，与 stuck pending 同理：允许整段重启，不要求用户重新发起创作。
	if task.Status == "failed" && len(task.Plan) == 0 {
		acquired, err := s.panelRepo.TryAcquireResumeProcessing(ctx, taskID, userID, 5, "understanding_reference")
		if err != nil {
			return nil, fmt.Errorf("acquire resume lock: %w", err)
		}
		if !acquired {
			return nil, ErrPanelGenerationResumeConflict
		}
		if err := s.resetDraftGeneratingContent(ctx, task.DraftFragmentID); err != nil {
			_ = s.panelRepo.RevertProcessingToFailed(ctx, taskID, userID, fmt.Sprintf("恢复草稿失败: %v", err))
			return nil, fmt.Errorf("%w: %v", ErrPanelGenerationDraftResetFailed, err)
		}
		s.logger.Info("panel_generation_resume",
			zap.String("task_id", taskID),
			zap.String("user_id", userID),
			zap.String("mode", "full_restart_failed_empty_plan"),
			zap.Int("plan_len", 0),
			zap.Int("start_panel", 0),
		)
		go s.process(context.Background(), taskID)
		out, err := s.panelRepo.GetByID(ctx, taskID)
		if err != nil {
			return task, nil
		}
		return out, nil
	}

	n := len(task.Plan)
	start := 0
	if task.Result != nil {
		start = len(task.Result.Panels)
	}
	if start > n {
		return nil, fmt.Errorf("%w: 任务数据不一致", ErrPanelGenerationNotResumable)
	}

	progress := 28
	currentStep := "plan_ready"
	if start >= n {
		currentStep = "assembling"
		progress = 92
	} else {
		currentStep = fmt.Sprintf("generating_panel_%d", start)
		den := n
		if den < 1 {
			den = 1
		}
		progress = 28 + (62 * start / den)
	}

	acquired, err := s.panelRepo.TryAcquireResumeProcessing(ctx, taskID, userID, progress, currentStep)
	if err != nil {
		return nil, fmt.Errorf("acquire resume lock: %w", err)
	}
	if !acquired {
		return nil, ErrPanelGenerationResumeConflict
	}
	if err := s.resetDraftGeneratingContent(ctx, task.DraftFragmentID); err != nil {
		_ = s.panelRepo.RevertProcessingToFailed(ctx, taskID, userID, fmt.Sprintf("恢复草稿失败: %v", err))
		return nil, fmt.Errorf("%w: %v", ErrPanelGenerationDraftResetFailed, err)
	}

	s.logger.Info("panel_generation_resume",
		zap.String("task_id", taskID),
		zap.String("user_id", userID),
		zap.String("mode", "from_plan"),
		zap.Int("plan_len", n),
		zap.Int("start_panel", start),
		zap.Int("progress", progress),
		zap.String("current_step", currentStep),
	)

	go s.processResume(context.Background(), taskID, start)
	out, err := s.panelRepo.GetByID(ctx, taskID)
	if err != nil {
		return task, nil
	}
	return out, nil
}

func (s *FragmentPanelGenerationService) resetDraftGeneratingContent(ctx context.Context, draftID string) error {
	if draftID == "" {
		return nil
	}
	frag, err := s.fragmentRepo.GetByID(ctx, draftID)
	if err != nil {
		return err
	}
	frag.Content = "生成中…"
	frag.UpdatedAt = time.Now().UnixMilli()
	return s.fragmentRepo.Update(ctx, frag)
}

func (s *FragmentPanelGenerationService) process(ctx context.Context, taskID string) {
	task, err := s.panelRepo.GetByID(ctx, taskID)
	if err != nil {
		s.logger.Error("panel gen: load task", zap.String("task_id", taskID), zap.Error(err))
		return
	}
	if task.Metrics == nil {
		task.Metrics = &domain.FragmentPanelMetricsData{Steps: []domain.FragmentPanelStepMetric{}}
	}
	req := task.Request
	draftID := task.DraftFragmentID

	started := time.Now().Unix()
	task.Status = "processing"
	task.StartedAt = &started
	task.Progress = 5
	task.CurrentStep = "understanding_reference"
	task.UpdatedAt = time.Now().Unix()
	if err := s.panelRepo.Save(ctx, task); err != nil {
		s.logger.Error("panel gen: save task start", zap.Error(err))
		return
	}

	planProv, imgPreferred := ResolvePanelGenerationAIProviders(req.UserRegion, s.defaultImageProvider, s.aiGen)
	imgProv := CoalesceRegisteredImageProvider(s.aiGen.GenAPI(), imgPreferred)
	if s.aiGen.GenAPI() == nil || s.aiGen.GenAPI().GetImageProvider(imgProv) == nil {
		s.failTask(ctx, taskID, draftID, "未配置可用的图片生成服务")
		return
	}

	visualEvidence, visionTok, visionDur := s.analyzePanelVisualEvidence(ctx, taskID, req)
	if len(visualEvidence) > 0 {
		task.VisualEvidence = visualEvidence
		appendPanelMetric(task, "visual_evidence", visionTok, visionDur, visualEvidence[0].Provider, visualEvidence[0].Model)
		task.UpdatedAt = time.Now().Unix()
		_ = s.panelRepo.Save(ctx, task)
	}

	// Step 1 — plan
	planRes, err := s.aiGen.GenerateFragmentPanelPlan(ctx, &GenerateFragmentPanelPlanRequest{
		UserID:            task.UserID,
		ReferenceImageURL: req.ReferenceImageURL,
		UserInput:         req.UserInput,
		Style:             req.Style,
		Language:          req.Language,
		PanelCount:        req.PanelCount,
		RelatedEntityID:   taskID,
		RelatedEntityType: "fragment_panel_generation",
		Metadata:          map[string]interface{}{"step": "panel_gen_step1_plan"},
		UserRegion:        req.UserRegion,
		PlanProvider:      planProv,
		LayoutAddon:       fragmentPanelPlanLayoutAddon(req),
		VisualEvidence:    visualEvidence,
	})
	if err != nil {
		s.failTask(ctx, taskID, draftID, fmt.Sprintf("规划分镜失败: %v", err))
		return
	}
	provLabel := planRes.Provider
	if provLabel == "" {
		provLabel = planProv
	}
	appendPanelMetric(task, "understanding_reference", planRes.TokensUsed, planRes.DurationMs, provLabel, planRes.Model)
	task.Plan = planRes.Plan
	task.VisualBible = planRes.VisualBible
	if task.VisualBible != nil && len(task.VisualBible.SourceEvidence) == 0 {
		task.VisualBible.SourceEvidence = visualEvidence
	}
	task.Progress = 28
	task.CurrentStep = "plan_ready"
	task.UpdatedAt = time.Now().Unix()
	if err := s.panelRepo.Save(ctx, task); err != nil {
		s.logger.Error("panel gen: save after plan", zap.Error(err))
	}

	n := len(task.Plan)
	task.Result = &domain.FragmentPanelResultData{Panels: make([]domain.FragmentPanelResultItem, 0, n)}

	policy := resolvePanelConsistencyPolicy(task, req)
	anchorMap := make(map[string]string)
	if task.VisualBible != nil && s.aiService != nil && policy.EnableReferenceAssets {
		task.CurrentStep = "generating_reference_assets"
		task.Progress = 32
		task.UpdatedAt = time.Now().Unix()
		_ = s.panelRepo.Save(ctx, task)
		anchorStart := time.Now()
		m, recs, anchorTok := s.generatePanelAnchorImages(ctx, task, req, policy)
		anchorMap = m
		task.AnchorImages = recs
		appendPanelMetric(task, "generating_reference_assets", anchorTok, time.Since(anchorStart).Milliseconds(), imgProv, "")
		task.UpdatedAt = time.Now().Unix()
		_ = s.panelRepo.Save(ctx, task)
	}

	if strings.EqualFold(imgProv, "huoshan") {
		if err := s.runPanelImageBatchHuoshan(ctx, task, taskID, draftID, req, imgProv, n, anchorMap, policy); err != nil {
			s.failTask(ctx, taskID, draftID, err.Error())
			return
		}
	} else {
		if err := s.runPanelImageLoop(ctx, task, taskID, draftID, req, imgProv, 0, n, anchorMap, policy); err != nil {
			s.failTask(ctx, taskID, draftID, err.Error())
			return
		}
	}

	checkTok, checkDur, checkProvider := s.runPanelConsistencyCheck(ctx, task, policy)
	if checkTok > 0 {
		if checkProvider == "" {
			checkProvider = provLabel
		}
		appendPanelMetric(task, "checking_consistency", checkTok, checkDur, checkProvider, "")
		task.UpdatedAt = time.Now().Unix()
		_ = s.panelRepo.Save(ctx, task)
	}

	s.completePanelGeneration(ctx, task, taskID, draftID, n)
}

func (s *FragmentPanelGenerationService) analyzePanelVisualEvidence(ctx context.Context, taskID string, req domain.FragmentPanelGenerationRequest) ([]domain.FragmentVisualEvidence, int, int64) {
	if s.aiService == nil {
		return nil, 0, 0
	}
	imageURLs := fragmentPrefillHTTPImageURLs([]string{req.ReferenceImageURL}, 1)
	if len(imageURLs) == 0 {
		return nil, 0, 0
	}
	start := time.Now()
	resp, err := s.aiService.GenerateFragmentVisionJSON(ctx, &FragmentVisionJSONRequest{
		Prompt:            buildFragmentVisualEvidencePrompt(req.UserInput, req.Style, imageURLs),
		ImageURLs:         imageURLs,
		ProviderHint:      NormalizeTextPlanProvider("", req.UserRegion, s.aiGen),
		MaxTokens:         4096,
		Temperature:       0.25,
		RelatedEntityType: "fragment_panel_generation",
		RelatedEntityID:   taskID,
		Step:              "panel_visual_evidence",
	})
	dur := time.Since(start).Milliseconds()
	if err != nil {
		s.logger.Warn("panel visual evidence failed", zap.Error(err))
		return nil, 0, dur
	}
	evidence, perr := parseFragmentVisualEvidence(resp.Raw, imageURLs)
	if perr != nil {
		s.logger.Warn("panel visual evidence parse failed", zap.Error(perr), zap.String("snippet", truncateRunes(resp.Raw, 200)))
		return nil, resp.TokensUsed, dur
	}
	for i := range evidence {
		evidence[i].Provider = resp.Provider
		evidence[i].Model = resp.Model
	}
	return evidence, resp.TokensUsed, resp.DurationMs
}

func (s *FragmentPanelGenerationService) processResume(ctx context.Context, taskID string, startIdx int) {
	s.logger.Info("panel_generation_resume_worker_start",
		zap.String("task_id", taskID),
		zap.Int("start_panel", startIdx),
	)
	task, err := s.panelRepo.GetByID(ctx, taskID)
	if err != nil {
		s.logger.Error("panel resume: load task", zap.String("task_id", taskID), zap.Error(err))
		return
	}
	if task.Metrics == nil {
		task.Metrics = &domain.FragmentPanelMetricsData{Steps: []domain.FragmentPanelStepMetric{}}
	}
	req := task.Request
	draftID := task.DraftFragmentID
	n := len(task.Plan)
	if n == 0 {
		s.failTask(ctx, taskID, draftID, "无分镜规划，无法续跑")
		return
	}

	_, imgPreferred := ResolvePanelGenerationAIProviders(req.UserRegion, s.defaultImageProvider, s.aiGen)
	imgProv := CoalesceRegisteredImageProvider(s.aiGen.GenAPI(), imgPreferred)
	if s.aiGen.GenAPI() == nil || s.aiGen.GenAPI().GetImageProvider(imgProv) == nil {
		s.failTask(ctx, taskID, draftID, "未配置可用的图片生成服务")
		return
	}

	policy := resolvePanelConsistencyPolicy(task, req)
	if startIdx >= n {
		if task.Result == nil || len(task.Result.Panels) != n {
			s.failTask(ctx, taskID, draftID, "任务数据不完整，无法完成")
			return
		}
		checkTok, checkDur, checkProvider := s.runPanelConsistencyCheck(ctx, task, policy)
		if checkTok > 0 {
			planProv, _ := ResolvePanelGenerationAIProviders(req.UserRegion, s.defaultImageProvider, s.aiGen)
			if checkProvider == "" {
				checkProvider = planProv
			}
			appendPanelMetric(task, "checking_consistency", checkTok, checkDur, checkProvider, "")
			_ = s.panelRepo.Save(ctx, task)
		}
		s.completePanelGeneration(ctx, task, taskID, draftID, n)
		return
	}

	if task.Result == nil {
		task.Result = &domain.FragmentPanelResultData{Panels: make([]domain.FragmentPanelResultItem, 0, n)}
	}
	if len(task.Result.Panels) != startIdx {
		s.failTask(ctx, taskID, draftID, "已生成格数与任务不一致，无法续跑")
		return
	}

	anchorMap := anchorMapFromPanelRecords(task.AnchorImages)
	if task.VisualBible != nil && len(anchorMap) == 0 && s.aiService != nil && policy.EnableReferenceAssets {
		task.CurrentStep = "generating_reference_assets"
		task.UpdatedAt = time.Now().Unix()
		_ = s.panelRepo.Save(ctx, task)
		anchorStart := time.Now()
		m, recs, anchorTok := s.generatePanelAnchorImages(ctx, task, req, policy)
		anchorMap = m
		task.AnchorImages = recs
		appendPanelMetric(task, "generating_reference_assets", anchorTok, time.Since(anchorStart).Milliseconds(), imgProv, "")
		task.UpdatedAt = time.Now().Unix()
		_ = s.panelRepo.Save(ctx, task)
	}

	if strings.EqualFold(imgProv, "huoshan") {
		if err := s.runPanelImageBatchHuoshanRange(ctx, task, taskID, draftID, req, imgProv, startIdx, n, anchorMap, policy); err != nil {
			s.failTask(ctx, taskID, draftID, err.Error())
			return
		}
	} else if err := s.runPanelImageLoop(ctx, task, taskID, draftID, req, imgProv, startIdx, n, anchorMap, policy); err != nil {
		s.failTask(ctx, taskID, draftID, err.Error())
		return
	}

	checkTok, checkDur, checkProvider := s.runPanelConsistencyCheck(ctx, task, policy)
	if checkTok > 0 {
		planProv, _ := ResolvePanelGenerationAIProviders(req.UserRegion, s.defaultImageProvider, s.aiGen)
		if checkProvider == "" {
			checkProvider = planProv
		}
		appendPanelMetric(task, "checking_consistency", checkTok, checkDur, checkProvider, "")
		task.UpdatedAt = time.Now().Unix()
		_ = s.panelRepo.Save(ctx, task)
	}

	s.completePanelGeneration(ctx, task, taskID, draftID, n)
}

// buildPanelFinalImagePrompt 将规划模型产出的英文 image_prompt 与风格、长宽比、AI 布局计划拼装为最终文生图/参考生图提示。
func buildPanelFinalImagePrompt(planItem domain.FragmentPanelPlanItem, styleSlug, aspectRatio string, panelIndex, totalPanels int, languages ...string) string {
	language := inferGenerationLanguage(planItem.Caption)
	if len(languages) > 0 && strings.TrimSpace(languages[0]) != "" {
		language = normalizeGenerationLanguage(languages[0])
	}
	planImagePrompt := strings.TrimSpace(planItem.ImagePrompt)
	base := strings.TrimSpace(fmt.Sprintf("Narrative beat (%s; must be preserved): %s.\nVisual execution (English): %s.", generationLanguageName(language), strings.TrimSpace(planItem.Caption), planImagePrompt))
	ar := domain.NormalizeFragmentAspectRatio(aspectRatio)
	if ar == "" {
		ar = domain.FragmentAspectDefault
	}
	st := strings.TrimSpace(styleSlug)
	if st == "" {
		st = "fantasy"
	}
	styleDesc := fragmentStyleDesc(st)
	hdr := fmt.Sprintf(
		"Output aspect ratio: %s. Visual style slug: %s. Style direction for consistency: %s.",
		ar, st, styleDesc,
	)
	var role string
	if panelIndex == 0 {
		role = "Panel role — ANCHOR: Preserve character, setting, and mood identity from the user's reference; reinterpret with creative composition and detail — avoid a literal 1:1 trace of the reference photograph."
	} else {
		role = "Panel role — CONTINUITY: Match the ongoing story and cast from prior panels; this is a NEW beat — different shot, moment, or angle — not a near-duplicate of the previous frame."
	}
	if totalPanels < 1 {
		totalPanels = 1
	}
	order := fmt.Sprintf("Panel %d of %d.", panelIndex+1, totalPanels)
	layout := buildPanelLayoutDirective(planItem)
	comic := buildPanelComicTextDirective(planItem, language)
	canvas := fullBleedCanvasDirective()
	return strings.TrimSpace(fmt.Sprintf("%s\n\n%s\n\n%s\n\n%s\n\n%s\n\n%s\n\n%s", base, hdr, order, role, layout, comic, canvas))
}

func buildPanelLayoutDirective(planItem domain.FragmentPanelPlanItem) string {
	var parts []string
	if v := strings.TrimSpace(planItem.LayoutIntent); v != "" {
		parts = append(parts, "layout_intent="+v)
	}
	if v := strings.TrimSpace(planItem.ShotType); v != "" {
		parts = append(parts, "shot_type="+v)
	}
	if v := strings.TrimSpace(planItem.CompositionPlan); v != "" {
		parts = append(parts, "composition_plan="+v)
	}
	if v := strings.TrimSpace(planItem.VisualHierarchy); v != "" {
		parts = append(parts, "visual_hierarchy="+v)
	}
	if len(parts) == 0 {
		return "Layout directive: one output image. Use either a unified continuous illustration or multiple clearly separated intra-image regions/sub-panels (comic-style zones separated by gutters / spacing) when the story beat gains clarity; prioritize readable order (e.g. top-to-bottom or left-to-right within the canvas). Maintain one consistent art treatment across zones, and keep every separation inside the canvas instead of framing the whole image."
	}
	out := "Layout directive for this single output image: " + strings.Join(parts, "; ") + ". Honor composition_plan / visual_hierarchy: if composition_plan implies several zones or sub-panels, render them as distinct regions in one image with visible internal separation/gutters; if it implies one integrated scene, keep it a single uninterrupted illustration. In both cases never draw an outer panel border around the image."
	if panelPlanWantsComicLayout(planItem) {
		out += " Comic rendering: strong ink line weight, screentones, internal gutter spacing, and sequential reading order. Sub-panel dividers stay inside the canvas and outermost zones bleed off the edges. Lettering is governed only by the separate lettering policy."
	}
	return out
}

func panelPlanWantsComicLayout(planItem domain.FragmentPanelPlanItem) bool {
	if len(planItem.ComicTexts) > 0 {
		return true
	}
	probe := strings.ToLower(strings.Join([]string{
		planItem.LayoutIntent,
		planItem.CompositionPlan,
		planItem.VisualHierarchy,
		planItem.ImagePrompt,
	}, " "))
	return strings.Contains(probe, "comic") || strings.Contains(probe, "manga") || strings.Contains(probe, "manhua") || strings.Contains(probe, "漫画")
}

func buildPanelComicTextDirective(planItem domain.FragmentPanelPlanItem, languages ...string) string {
	language := inferGenerationLanguage(planItem.Caption)
	if len(languages) > 0 && strings.TrimSpace(languages[0]) != "" {
		language = normalizeGenerationLanguage(languages[0])
	}
	visualTexts := fragmentComicTextsToVisual(planItem.ComicTexts, language)
	var b strings.Builder
	b.WriteString(visualSceneLetteringPolicy(language, visualTexts))
	if len(visualTexts) == 0 {
		return b.String()
	}
	b.WriteString("\nComic text elements:\n")
	for _, item := range normalizeFragmentComicTextsForLanguage(planItem.ComicTexts, language) {
		text := sanitizeComicPromptText(item.Text)
		position := strings.TrimSpace(item.Position)
		if position == "" {
			position = "auto"
		}
		speaker := strings.TrimSpace(item.Speaker)
		switch item.Type {
		case "narration":
			fmt.Fprintf(&b, "- Caption/narration box at %s, paint only the exact supplied text %q inside the box.\n", position, text)
		case "dialogue":
			fmt.Fprintf(&b, "- Speech bubble for %s at %s, paint only the exact supplied dialogue %q inside the bubble, tail pointing to the speaker.\n", speaker, position, text)
		case "thought":
			fmt.Fprintf(&b, "- Thought bubble for %s at %s, paint only the exact supplied inner monologue %q inside the bubble.\n", speaker, position, text)
		case "sfx":
			fmt.Fprintf(&b, "- Bold stylized comic SFX lettering at %s, paint only the exact supplied SFX text %q.\n", position, text)
		}
	}
	return strings.TrimSpace(b.String())
}

func anchorMapFromPanelRecords(records []domain.FragmentAnchorImage) map[string]string {
	m := make(map[string]string)
	for _, r := range records {
		k := strings.TrimSpace(r.Key)
		u := strings.TrimSpace(r.ImageURL)
		if k != "" && u != "" {
			m[k] = u
		}
	}
	return m
}

func mergePanelReferenceImages(userHTTPRef string, keys []string, anchorMap map[string]string, prevPanelURL string, maxTotal int) []string {
	if maxTotal <= 0 {
		maxTotal = panelMaxReferenceImages
	}
	var userURLs []string
	if u := strings.TrimSpace(userHTTPRef); u != "" {
		low := strings.ToLower(u)
		if strings.HasPrefix(low, "http://") || strings.HasPrefix(low, "https://") {
			userURLs = append(userURLs, u)
		}
	}
	seen := map[string]struct{}{}
	var out []string
	add := func(u string) {
		u = strings.TrimSpace(u)
		if u == "" {
			return
		}
		if _, ok := seen[u]; ok {
			return
		}
		seen[u] = struct{}{}
		out = append(out, u)
	}
	for _, u := range userURLs {
		add(u)
		if len(out) >= maxTotal {
			return out
		}
	}
	for _, k := range keys {
		if anchorMap == nil {
			continue
		}
		if u := anchorMap[strings.TrimSpace(k)]; u != "" {
			add(u)
			if len(out) >= maxTotal {
				return out
			}
		}
	}
	if u := strings.TrimSpace(prevPanelURL); u != "" {
		add(u)
	}
	return out
}

func resolvePanelConsistencyPolicy(task *domain.FragmentPanelGenerationTask, req domain.FragmentPanelGenerationRequest) *domain.FragmentConsistencyPolicy {
	level := normalizeFragmentConsistencyLevel(req.ConsistencyLevel)
	if level == "off" {
		return &domain.FragmentConsistencyPolicy{
			Level:                 "off",
			EnableReferenceAssets: false,
			Capabilities:          []string{},
		}
	}
	seriesSeed := fragmentStableSeed(task.ID, req.Style, req.AspectRatio)
	scenePlans := panelPlanItemsToScenePlans(task.Plan)
	usage := analyzeFragmentSceneEntityUsage(scenePlans)
	enableRefs := len(scenePlans) > 2 || countRepeatedFragmentCharacters(task.VisualBible, usage) > 1
	if req.EnableReferenceAssets != nil {
		enableRefs = *req.EnableReferenceAssets
	}
	maxChars := 2
	maxProps := 0
	maxLocs := 0
	if level == "strong" {
		maxChars = 3
		maxProps = 1
		maxLocs = 1
		enableRefs = true
	}
	options := map[string]interface{}{
		"consistency_group_id": task.ID,
		"series_seed":          seriesSeed,
		"style_consistency":    true,
	}
	return &domain.FragmentConsistencyPolicy{
		Level:                 level,
		SeriesSeed:            seriesSeed,
		EnableReferenceAssets: enableRefs,
		MaxCharacterAssets:    maxChars,
		MaxPropAssets:         maxProps,
		MaxLocationAssets:     maxLocs,
		ProviderOptions:       options,
		Capabilities:          []string{"seed", "provider_options", "entity_bindings", "reference_assets"},
	}
}

func panelPlanItemsToScenePlans(plan []domain.FragmentPanelPlanItem) []domain.FragmentScenePlan {
	out := make([]domain.FragmentScenePlan, 0, len(plan))
	for i, p := range plan {
		idx := p.Index
		if idx < 0 {
			idx = i
		}
		out = append(out, domain.FragmentScenePlan{
			Index:         idx,
			ImagePrompt:   strings.TrimSpace(p.ImagePrompt),
			SceneDesc:     strings.TrimSpace(p.Caption),
			ReferenceKeys: normalizeFragmentKeyList(p.ReferenceKeys),
			ComicTexts:    normalizeFragmentComicTexts(p.ComicTexts),
		})
	}
	return out
}

func panelSeed(policy *domain.FragmentConsistencyPolicy, _ int, _ []string) int {
	return fragmentStoryImageSeed(policy)
}

func (s *FragmentPanelGenerationService) generatePanelAnchorImages(ctx context.Context, task *domain.FragmentPanelGenerationTask, req domain.FragmentPanelGenerationRequest, policy *domain.FragmentConsistencyPolicy) (map[string]string, []domain.FragmentAnchorImage, int) {
	outMap := make(map[string]string)
	if task.VisualBible == nil || s.aiService == nil || policy == nil || !policy.EnableReferenceAssets {
		return outMap, nil, 0
	}
	bible := task.VisualBible
	ar := domain.NormalizeFragmentAspectRatio(req.AspectRatio)
	if ar == "" {
		ar = domain.FragmentAspectDefault
	}
	styleEN := ""
	if bible.StyleBible != nil {
		styleEN = strings.TrimSpace(bible.StyleBible.ArtStyle)
	}
	styleZH := fragmentStyleDesc(req.Style)
	moodZH := fragmentMoodDesc("")
	var userRef []string
	if u := strings.TrimSpace(req.ReferenceImageURL); u != "" {
		low := strings.ToLower(u)
		if strings.HasPrefix(low, "http://") || strings.HasPrefix(low, "https://") {
			userRef = append(userRef, u)
		}
	}

	scenePlans := panelPlanItemsToScenePlans(task.Plan)
	candidates := selectFragmentReferenceAssetCandidatesWithEvidence(bible, task.VisualEvidence, scenePlans, policy)
	if len(candidates) == 0 {
		return outMap, nil, 0
	}
	firstChar := true
	totalTok := 0
	var records []domain.FragmentAnchorImage
	for _, c := range candidates {
		prompt := buildFragmentReferenceAssetPrompt(c, styleEN, styleZH, moodZH)
		var refs []string
		if c.Kind == "character" && firstChar && len(userRef) > 0 {
			refs = append(refs, userRef...)
			firstChar = false
		}
		seed := fragmentStableSeed(task.ID, c.Kind, c.Key, strings.Join(c.Traits, "|"))
		url, tok, err := s.generateOnePanelAnchorImage(ctx, task.UserID, task.ID, prompt, ar, refs, seed, policy.ProviderOptions)
		if err != nil {
			s.logger.Warn("panel reference asset failed", zap.String("key", c.Key), zap.String("kind", c.Kind), zap.Error(err))
			continue
		}
		outMap[c.Key] = url
		records = append(records, domain.FragmentAnchorImage{Key: c.Key, Kind: c.Kind, ImageURL: url})
		totalTok += tok
	}

	return outMap, records, totalTok
}

func (s *FragmentPanelGenerationService) generateOnePanelAnchorImage(ctx context.Context, userID, panelTaskID, prompt, aspectRatio string, refURLs []string, seed int, options map[string]interface{}) (string, int, error) {
	payload := map[string]interface{}{
		"prompt":      prompt,
		"aspectRatio": aspectRatio,
	}
	if len(refURLs) > 0 {
		payload["referenceImages"] = fragmentPrefillHTTPImageURLs(refURLs, 4)
	}
	if seed > 0 {
		payload["seed"] = seed
	}
	if len(options) > 0 {
		payload["options"] = options
	}
	imgInput, err := json.Marshal(payload)
	if err != nil {
		return "", 0, err
	}
	aiReq := domain.AITask{
		ID:                uuid.New().String(),
		UserID:            userID,
		Type:              domain.AITaskGenerateFragmentImages,
		Status:            domain.AITaskStatusProcessing,
		Provider:          "",
		Input:             string(imgInput),
		RelatedEntityID:   panelTaskID,
		RelatedEntityType: "fragment_panel_generation",
	}
	return s.aiService.GenerateImageForFragment(ctx, &aiReq)
}

// runPanelConsistencyCheck best-effort，与普通碎片 checkFragmentConsistency 对齐；返回 token、耗时(ms) 与 provider。
func (s *FragmentPanelGenerationService) runPanelConsistencyCheck(ctx context.Context, task *domain.FragmentPanelGenerationTask, policy *domain.FragmentConsistencyPolicy) (int, int64, string) {
	if s.aiService == nil || task.VisualBible == nil || task.Result == nil || len(task.Result.Panels) == 0 {
		return 0, 0, ""
	}
	scenes := panelPlanItemsToScenePlans(task.Plan)
	imageURLs := make([]string, 0, len(task.Result.Panels))
	for _, p := range task.Result.Panels {
		imageURLs = append(imageURLs, strings.TrimSpace(p.ImageURL))
	}
	auditURLs, skipped := selectFragmentAuditImageURLs(imageURLs, scenes, policy)
	if len(auditURLs) == 0 {
		if task.Result.GenerationTrace == nil {
			task.Result.GenerationTrace = &domain.FragmentGenerationTrace{}
		}
		task.Result.GenerationTrace.SkippedAuditReason = skipped
		return 0, 0, ""
	}
	start := time.Now()
	vbBytes, _ := json.Marshal(task.VisualBible)
	evidenceBytes, _ := json.Marshal(task.VisualEvidence)
	planBytes, _ := json.Marshal(task.Plan)
	var urlLines strings.Builder
	for i, u := range auditURLs {
		fmt.Fprintf(&urlLines, "%d: %s\n", i, u)
	}
	prompt := fmt.Sprintf(`你是视觉一致性审计员。请直接查看最终配图，基于「视觉圣经」、入口视觉事实、分镜规划（含 caption、reference_keys、image_prompt）与最终配图 URL，列出可能的不一致。无问题时 issues 为空数组。

视觉圣经 JSON：
%s

入口视觉事实 JSON：
%s

分镜规划 JSON：
%s

最终图片 URL（与 index 对应）：
%s

只输出 JSON：{"issues":[{"sceneIndex":0,"entityKey":"可为空","imageUrl":"问题图片URL","severity":"low|medium|high","expected":"应保持的视觉事实","observed":"实际观察","confidence":0.0,"detail":"中文简述"}]}`,
		string(vbBytes), string(evidenceBytes), string(planBytes), urlLines.String())

	resp, err := s.aiService.GenerateFragmentVisionJSON(ctx, &FragmentVisionJSONRequest{
		Prompt:            prompt,
		ImageURLs:         auditURLs,
		MaxTokens:         2048,
		Temperature:       0.25,
		RelatedEntityType: "fragment_panel_generation",
		RelatedEntityID:   task.ID,
		Step:              "panel_consistency_audit",
	})
	dur := time.Since(start).Milliseconds()
	if err != nil {
		s.logger.Warn("panel consistency check failed", zap.Error(err))
		if task.Result.GenerationTrace == nil {
			task.Result.GenerationTrace = &domain.FragmentGenerationTrace{}
		}
		task.Result.GenerationTrace.SkippedAuditReason = err.Error()
		task.Result.GenerationTrace.AuditedImageCount = len(auditURLs)
		return 0, dur, ""
	}
	if resp.DurationMs > 0 {
		dur = resp.DurationMs
	}
	issues, perr := parseFragmentConsistencyIssues(resp.Raw)
	if perr != nil {
		s.logger.Warn("panel consistency parse failed", zap.Error(perr), zap.String("snippet", truncateRunes(resp.Raw, 200)))
		return resp.TokensUsed, dur, resp.Provider
	}
	if task.Result != nil {
		task.Result.ConsistencyIssues = issues
		task.Result.GenerationTrace = &domain.FragmentGenerationTrace{
			VisualBible:         task.VisualBible,
			VisualEvidence:      task.VisualEvidence,
			Scenes:              scenes,
			ConsistencyPolicy:   policy,
			ConsistencyIssues:   issues,
			VisionAuditProvider: resp.Provider,
			AuditedImageCount:   len(auditURLs),
			SkippedAuditReason:  skipped,
		}
	}
	if len(issues) > 0 {
		for _, iss := range issues {
			s.logger.Info("panel consistency issue",
				zap.String("task_id", task.ID),
				zap.Int("scene_index", iss.SceneIndex),
				zap.String("detail", iss.Detail))
		}
	}
	return resp.TokensUsed, dur, resp.Provider
}

func (s *FragmentPanelGenerationService) runPanelImageLoop(ctx context.Context, task *domain.FragmentPanelGenerationTask, taskID, draftID string, req domain.FragmentPanelGenerationRequest, imgProv string, startIdx, n int, anchorMap map[string]string, policy *domain.FragmentConsistencyPolicy) error {
	den := n
	if den < 1 {
		den = 1
	}
	for i := startIdx; i < n; i++ {
		stepName := fmt.Sprintf("generating_panel_%d", i)
		task.CurrentStep = stepName
		task.Progress = 28 + (62 * (i + 1) / den)
		task.UpdatedAt = time.Now().Unix()
		_ = s.panelRepo.Save(ctx, task)

		planItem := task.Plan[i]
		prompt := buildPanelFinalImagePrompt(planItem, req.Style, req.AspectRatio, i, n, req.Language)

		prevURL := ""
		if i > 0 && len(task.Result.Panels) > 0 && i-1 < len(task.Result.Panels) {
			prevURL = strings.TrimSpace(task.Result.Panels[i-1].ImageURL)
		}
		if anchorMap == nil {
			anchorMap = make(map[string]string)
		}
		refURLs := mergePanelReferenceImages(req.ReferenceImageURL, planItem.ReferenceKeys, anchorMap, prevURL, panelMaxReferenceImages)
		if len(refURLs) == 0 {
			if u := strings.TrimSpace(req.ReferenceImageURL); u != "" {
				refURLs = []string{u}
			} else if i > 0 && prevURL != "" {
				refURLs = []string{prevURL}
			}
		}

		refKind := "multi_reference"
		if len(refURLs) == 1 && i == 0 {
			refKind = "user_upload_or_single"
		} else if len(refURLs) == 1 && i > 0 {
			refKind = "previous_panel_or_single"
		}
		imgStart := time.Now()
		ar := domain.NormalizeFragmentAspectRatio(req.AspectRatio)
		if ar == "" {
			ar = domain.FragmentAspectDefault
		}
		seed := panelSeed(policy, i, planItem.ReferenceKeys)
		options := cloneFragmentProviderOptions(policy)
		imgReq := &GenerateImageRequest{
			UserID:            task.UserID,
			Prompt:            prompt,
			Provider:          imgProv,
			Quality:           "standard",
			Style:             req.Style,
			OutputCount:       1,
			ReferenceImages:   refURLs,
			Seed:              seed,
			Options:           options,
			RelatedEntityID:   taskID,
			RelatedEntityType: "fragment_panel_generation",
			Metadata: map[string]interface{}{
				"step":            stepName,
				"panel":           i,
				"reference_kind":  refKind,
				"aspectRatio":     ar,
				"reference_count": len(refURLs),
				"seed":            seed,
			},
		}
		switch imgProv {
		case "huoshan":
			imgReq.Size = domain.FragmentImagePixelSizeForAspectRatio(ar)
		default:
			imgReq.AspectRatio = ar
		}
		imgOut, genErr := s.aiGen.GenerateImage(ctx, imgReq)
		if genErr != nil {
			return fmt.Errorf("第 %d 张图生成失败: %v", i+1, genErr)
		}
		if imgOut == nil || len(imgOut.ImageURLs) == 0 {
			return fmt.Errorf("第 %d 张图无返回", i+1)
		}
		dur := time.Since(imgStart).Milliseconds()
		if imgOut.DurationMs > 0 {
			dur = imgOut.DurationMs
		}
		appendPanelMetric(task, stepName, imgOut.TokensUsed, dur, imgProv, "")

		imageURL := imgOut.ImageURLs[0]
		task.Result.Panels = append(task.Result.Panels, domain.FragmentPanelResultItem{
			Index:    i,
			ImageURL: imageURL,
			Caption:  strings.TrimSpace(planItem.Caption),
		})
		task.UpdatedAt = time.Now().Unix()
		if err := s.panelRepo.Save(ctx, task); err != nil {
			s.logger.Error("panel gen: save after panel", zap.Int("panel", i), zap.Error(err))
		}
		s.syncDraftFromTask(ctx, task)
	}
	return nil
}

func (s *FragmentPanelGenerationService) completePanelGeneration(ctx context.Context, task *domain.FragmentPanelGenerationTask, taskID, draftID string, n int) {
	req := task.Request
	if task.Result == nil || len(task.Result.Panels) != n {
		s.failTask(ctx, taskID, draftID, "任务数据不完整，无法完成")
		return
	}

	task.CurrentStep = "assembling"
	task.Progress = 92
	task.UpdatedAt = time.Now().Unix()
	_ = s.panelRepo.Save(ctx, task)

	var captions []string
	urls := make([]string, 0, len(task.Result.Panels))
	for _, p := range task.Result.Panels {
		urls = append(urls, p.ImageURL)
		if p.Caption != "" {
			captions = append(captions, p.Caption)
		}
	}
	task.Result.CombinedContent = strings.Join(captions, "\n\n")
	task.Result.VisualBible = task.VisualBible
	task.Result.VisualEvidence = task.VisualEvidence
	task.Result.AnchorImages = task.AnchorImages
	if task.Result.GenerationTrace == nil {
		task.Result.GenerationTrace = &domain.FragmentGenerationTrace{
			VisualBible:       task.VisualBible,
			VisualEvidence:    task.VisualEvidence,
			Scenes:            panelPlanItemsToScenePlans(task.Plan),
			ConsistencyPolicy: resolvePanelConsistencyPolicy(task, req),
		}
	}

	task.CurrentStep = "persisting"
	task.Progress = 97
	task.UpdatedAt = time.Now().Unix()
	_ = s.panelRepo.Save(ctx, task)

	imgCount := len(urls)
	if imgCount < 1 {
		imgCount = 1
	}
	styleVal := req.Style
	caption := captionFromPanelCaptions(captions, task.Result.CombinedContent)

	if err := s.finalizeDraft(ctx, draftID, task.Result.CombinedContent, urls, &styleVal, imgCount, caption, req); err != nil {
		s.logger.Error("panel gen: finalize draft", zap.Error(err))
		s.failTask(ctx, taskID, draftID, fmt.Sprintf("保存草稿失败: %v", err))
		return
	}
	s.persistPanelGenerationAssets(ctx, draftID, task, req.AspectRatio, resolvePanelConsistencyPolicy(task, req))

	completed := time.Now().Unix()
	task.Status = "completed"
	task.Progress = 100
	task.CurrentStep = "completed"
	task.CompletedAt = &completed
	task.UpdatedAt = time.Now().Unix()
	if err := s.panelRepo.Save(ctx, task); err != nil {
		s.logger.Error("panel gen: final save", zap.Error(err))
	}

	s.logger.Info("panel fragment generation completed",
		zap.String("task_id", taskID),
		zap.String("draft_id", draftID),
		zap.Int("panels", n))

	if s.notify != nil && strings.TrimSpace(draftID) != "" && strings.TrimSpace(task.UserID) != "" {
		preview := strings.TrimSpace(caption)
		tokens := 0
		if task.Metrics != nil {
			tokens = task.Metrics.TotalTokens
		}
		if err := s.notify.NotifyFragmentGenerationCompleted(context.Background(), task.UserID, draftID, preview, tokens); err != nil {
			s.logger.Warn("panel fragment generation completion notify failed",
				zap.Error(err),
				zap.String("task_id", taskID),
				zap.String("draft_id", draftID))
		}
	}
}

func appendPanelMetric(t *domain.FragmentPanelGenerationTask, name string, tokens int, durationMs int64, provider, model string) {
	if t.Metrics == nil {
		t.Metrics = &domain.FragmentPanelMetricsData{}
	}
	t.Metrics.Steps = append(t.Metrics.Steps, domain.FragmentPanelStepMetric{
		Name:       name,
		Tokens:     tokens,
		DurationMs: durationMs,
		Provider:   provider,
		Model:      model,
	})
	t.Metrics.TotalTokens += tokens
	t.Metrics.TotalDurationMs += durationMs
}

func (s *FragmentPanelGenerationService) syncDraftFromTask(ctx context.Context, task *domain.FragmentPanelGenerationTask) {
	if task.DraftFragmentID == "" || task.Result == nil {
		return
	}
	frag, err := s.fragmentRepo.GetByID(ctx, task.DraftFragmentID)
	if err != nil {
		s.logger.Warn("panel gen: load draft for sync", zap.Error(err))
		return
	}
	urls := make([]string, 0, len(task.Result.Panels))
	var caps []string
	for _, p := range task.Result.Panels {
		urls = append(urls, p.ImageURL)
		if p.Caption != "" {
			caps = append(caps, p.Caption)
		}
	}
	frag.MediaURLs = append([]string(nil), urls...)
	frag.ImageUrls = stringifyPanelImageURLs(urls)
	partial := strings.Join(caps, "\n\n")
	if partial == "" {
		partial = "生成中…"
	}
	frag.Content = partial
	frag.Topic = panelTopicForFragment(task.Request, frag.Topic)
	frag.AspectRatio = domain.NormalizeFragmentAspectRatio(task.Request.AspectRatio)
	frag.UpdatedAt = time.Now().UnixMilli()
	_ = s.fragmentRepo.Update(ctx, frag)
}

func (s *FragmentPanelGenerationService) finalizeDraft(ctx context.Context, draftID, content string, urls []string, style *string, fragmentCount int, caption string, req domain.FragmentPanelGenerationRequest) error {
	frag, err := s.fragmentRepo.GetByID(ctx, draftID)
	if err != nil {
		return err
	}
	frag.Content = content
	if strings.TrimSpace(frag.Content) == "" {
		frag.Content = " "
	}
	frag.MediaURLs = append([]string(nil), urls...)
	frag.ImageUrls = stringifyPanelImageURLs(urls)
	frag.Style = style
	frag.FragmentCount = &fragmentCount
	if caption != "" {
		frag.Caption = caption
	}
	frag.Topic = panelTopicForFragment(req, frag.Topic)
	if ar := domain.NormalizeFragmentAspectRatio(req.AspectRatio); ar != "" {
		frag.AspectRatio = ar
	}
	frag.UpdatedAt = time.Now().UnixMilli()
	return s.fragmentRepo.Update(ctx, frag)
}

func (s *FragmentPanelGenerationService) failTask(ctx context.Context, taskID, draftID, msg string) {
	s.logger.Warn("panel gen failed", zap.String("task_id", taskID), zap.String("msg", msg))
	userID := ""
	t, err := s.panelRepo.GetByID(ctx, taskID)
	if err == nil && t != nil {
		t.Status = "failed"
		t.ErrorMessage = msg
		now := time.Now().Unix()
		t.CompletedAt = &now
		t.UpdatedAt = now
		userID = strings.TrimSpace(t.UserID)
		_ = s.panelRepo.Save(ctx, t)
	} else {
		_ = s.panelRepo.UpdateError(ctx, taskID, msg)
	}
	s.markDraftFailed(ctx, draftID, truncatePanelErr(msg, 240))
	if s.notify != nil && strings.TrimSpace(draftID) != "" && userID != "" {
		if nerr := s.notify.NotifyFragmentGenerationFailed(context.Background(), userID, draftID, strings.TrimSpace(msg)); nerr != nil {
			s.logger.Warn("panel fragment generation failure notify failed",
				zap.Error(nerr),
				zap.String("task_id", taskID),
				zap.String("draft_id", draftID))
		}
	}
}

func (s *FragmentPanelGenerationService) markDraftFailed(ctx context.Context, draftID, shortMsg string) {
	if draftID == "" || shortMsg == "" {
		return
	}
	frag, err := s.fragmentRepo.GetByID(ctx, draftID)
	if err != nil {
		return
	}
	frag.Content = "生成失败：" + shortMsg
	frag.UpdatedAt = time.Now().UnixMilli()
	_ = s.fragmentRepo.Update(ctx, frag)
}

func stringifyPanelImageURLs(urls []string) string {
	if len(urls) == 0 {
		return "[]"
	}
	b, err := json.Marshal(urls)
	if err != nil {
		return "[]"
	}
	return string(b)
}

func captionFromPanelCaptions(captions []string, combined string) string {
	if len(captions) > 0 && strings.TrimSpace(captions[0]) != "" {
		r := []rune(strings.TrimSpace(captions[0]))
		if len(r) > 32 {
			return string(r[:32])
		}
		return string(r)
	}
	c := strings.TrimSpace(combined)
	if c == "" {
		return ""
	}
	r := []rune(c)
	if len(r) > 32 {
		return string(r[:32])
	}
	return c
}

func truncatePanelErr(s string, max int) string {
	s = strings.TrimSpace(s)
	if max <= 0 || len(s) <= max {
		return s
	}
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	return string(runes[:max]) + "…"
}
