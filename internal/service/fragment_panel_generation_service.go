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

// ErrPanelGenerationResumeConflict when a resume is requested while the task is already processing.
var ErrPanelGenerationResumeConflict = errors.New("panel generation already in progress")

// ErrPanelGenerationNotResumable when resume preconditions are not met (completed, no plan, bad data, etc.).
var ErrPanelGenerationNotResumable = errors.New("panel generation task cannot be resumed")

// ErrPanelGenerationDraftResetFailed when the draft fragment could not be set back to generating after acquiring resume lock.
var ErrPanelGenerationDraftResetFailed = errors.New("draft reset failed after resume lock")

// FragmentPanelGenerationService orchestrates multi-panel reference-based fragment generation.
type FragmentPanelGenerationService struct {
	panelRepo             *repository.FragmentPanelGenerationRepository
	fragmentRepo          *repository.FragmentRepository
	repo                  domain.Repository // optional: load user region for Huoshan/Gemini routing
	defaultImageProvider  string            // e.g. cfg.AI.ImageProvider; domestic panel images
	aiGen                 *AIGenerationService
	logger                *zap.Logger
}

// NewFragmentPanelGenerationService constructs the service.
func NewFragmentPanelGenerationService(
	panelRepo *repository.FragmentPanelGenerationRepository,
	fragmentRepo *repository.FragmentRepository,
	repo domain.Repository,
	defaultImageProvider string,
	aiGen *AIGenerationService,
	logger *zap.Logger,
) *FragmentPanelGenerationService {
	return &FragmentPanelGenerationService{
		panelRepo:            panelRepo,
		fragmentRepo:         fragmentRepo,
		repo:                 repo,
		defaultImageProvider: defaultImageProvider,
		aiGen:                aiGen,
		logger:               logger,
	}
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
	vis := domain.NormalizeFragmentVisibility(strings.TrimSpace(req.Visibility))
	if vis == "" {
		vis = domain.FragmentVisibilityPrivate
	}
	req.Visibility = vis

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
		Visibility:      domain.FragmentVisibilityPrivate,
		IsDraft:         true,
		SourceType:      string(domain.FragmentSourcePanelGeneration),
		SourceID:        taskID,
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

	if task.Status == "failed" && len(task.Plan) == 0 {
		return nil, fmt.Errorf("%w: 无分镜规划，请重新创作", ErrPanelGenerationNotResumable)
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

	// Step 1 — plan
	planRes, err := s.aiGen.GenerateFragmentPanelPlan(ctx, &GenerateFragmentPanelPlanRequest{
		UserID:            task.UserID,
		ReferenceImageURL: req.ReferenceImageURL,
		UserInput:         req.UserInput,
		Style:             req.Style,
		PanelCount:        req.PanelCount,
		RelatedEntityID:   taskID,
		RelatedEntityType: "fragment_panel_generation",
		Metadata:          map[string]interface{}{"step": "panel_gen_step1_plan"},
		UserRegion:        req.UserRegion,
		PlanProvider:      planProv,
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
	task.Progress = 28
	task.CurrentStep = "plan_ready"
	task.UpdatedAt = time.Now().Unix()
	if err := s.panelRepo.Save(ctx, task); err != nil {
		s.logger.Error("panel gen: save after plan", zap.Error(err))
	}

	n := len(task.Plan)
	task.Result = &domain.FragmentPanelResultData{Panels: make([]domain.FragmentPanelResultItem, 0, n)}

	if err := s.runPanelImageLoop(ctx, task, taskID, draftID, req, imgProv, 0, n); err != nil {
		s.failTask(ctx, taskID, draftID, err.Error())
		return
	}
	s.completePanelGeneration(ctx, task, taskID, draftID, n)
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

	if startIdx >= n {
		if task.Result == nil || len(task.Result.Panels) != n {
			s.failTask(ctx, taskID, draftID, "任务数据不完整，无法完成")
			return
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

	if err := s.runPanelImageLoop(ctx, task, taskID, draftID, req, imgProv, startIdx, n); err != nil {
		s.failTask(ctx, taskID, draftID, err.Error())
		return
	}
	s.completePanelGeneration(ctx, task, taskID, draftID, n)
}

func (s *FragmentPanelGenerationService) runPanelImageLoop(ctx context.Context, task *domain.FragmentPanelGenerationTask, taskID, draftID string, req domain.FragmentPanelGenerationRequest, imgProv string, startIdx, n int) error {
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
		base := strings.TrimSpace(planItem.ImagePrompt)
		var prompt string
		if i == 0 {
			prompt = fmt.Sprintf("%s Visual style: %s. Anchor: use the user's reference for character/setting/mood identity; reinterpret with creative composition and detail — avoid a literal 1:1 copy of the reference photo.", base, req.Style)
		} else {
			prompt = fmt.Sprintf("%s Visual style: %s. Continuity: match the ongoing story and cast from prior panels; this is a NEW beat — different shot, moment, or angle — not a near-duplicate of the previous frame.", base, req.Style)
		}

		refURL := strings.TrimSpace(req.ReferenceImageURL)
		if i > 0 && len(task.Result.Panels) > 0 && i-1 < len(task.Result.Panels) {
			prev := strings.TrimSpace(task.Result.Panels[i-1].ImageURL)
			if prev != "" {
				refURL = prev
			}
		}

		refKind := "previous_panel"
		if i == 0 {
			refKind = "user_upload"
		}
		imgStart := time.Now()
		imgOut, genErr := s.aiGen.GenerateImage(ctx, &GenerateImageRequest{
			UserID:            task.UserID,
			Prompt:            prompt,
			Provider:          imgProv,
			Size:              "1024x1024",
			Quality:           "standard",
			Style:             req.Style,
			OutputCount:       1,
			ReferenceImages:   []string{refURL},
			RelatedEntityID:   taskID,
			RelatedEntityType: "fragment_panel_generation",
			Metadata: map[string]interface{}{
				"step":           stepName,
				"panel":          i,
				"reference_kind": refKind,
			},
		})
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

	if err := s.finalizeDraft(ctx, draftID, task.Result.CombinedContent, urls, &styleVal, imgCount, caption); err != nil {
		s.logger.Error("panel gen: finalize draft", zap.Error(err))
		s.failTask(ctx, taskID, draftID, fmt.Sprintf("保存草稿失败: %v", err))
		return
	}

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
	frag.UpdatedAt = time.Now().UnixMilli()
	_ = s.fragmentRepo.Update(ctx, frag)
}

func (s *FragmentPanelGenerationService) finalizeDraft(ctx context.Context, draftID, content string, urls []string, style *string, fragmentCount int, caption string) error {
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
	frag.UpdatedAt = time.Now().UnixMilli()
	return s.fragmentRepo.Update(ctx, frag)
}

func (s *FragmentPanelGenerationService) failTask(ctx context.Context, taskID, draftID, msg string) {
	s.logger.Warn("panel gen failed", zap.String("task_id", taskID), zap.String("msg", msg))
	t, err := s.panelRepo.GetByID(ctx, taskID)
	if err == nil {
		t.Status = "failed"
		t.ErrorMessage = msg
		now := time.Now().Unix()
		t.CompletedAt = &now
		t.UpdatedAt = now
		_ = s.panelRepo.Save(ctx, t)
	} else {
		_ = s.panelRepo.UpdateError(ctx, taskID, msg)
	}
	s.markDraftFailed(ctx, draftID, truncatePanelErr(msg, 240))
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
