package service

import (
	"context"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/grapestree/fgrapery/grapery/internal/common"
	"github.com/grapestree/fgrapery/grapery/internal/domain"
	"github.com/grapestree/fgrapery/grapery/internal/repository"
)

type FragmentGenerationService struct {
	fragmentGenRepo *repository.FragmentGenerationRepository
	fragmentRepo    *repository.FragmentRepository
	repo            domain.Repository
	aiService       *AIService
	logger          *zap.Logger
	notify          *Service // optional: push + in-app when generation completes
}

func NewFragmentGenerationService(
	fragmentGenRepo *repository.FragmentGenerationRepository,
	fragmentRepo *repository.FragmentRepository,
	repo domain.Repository,
	aiService *AIService,
	logger *zap.Logger,
) *FragmentGenerationService {
	return &FragmentGenerationService{
		fragmentGenRepo: fragmentGenRepo,
		fragmentRepo:    fragmentRepo,
		repo:            repo,
		aiService:       aiService,
		logger:          logger,
	}
}

// SetNotify wires main Service for completion notifications (nil-safe).
func (s *FragmentGenerationService) SetNotify(svc *Service) {
	s.notify = svc
}

// GenerateFragment 创建生成任务并立即落库占位草稿（与多格参考图流程对齐），返回 task 与 draftFragmentId。
func (s *FragmentGenerationService) GenerateFragment(ctx context.Context, userID string, req domain.FragmentGenerationRequest) (*domain.FragmentGenerationTask, string, error) {
	taskID := uuid.New().String()
	task := &domain.FragmentGenerationTask{
		ID:        taskID,
		UserID:    userID,
		Status:    "pending",
		Request:   req,
		Progress:  0,
		CreatedAt: currentTime(),
	}

	if err := s.fragmentGenRepo.Create(ctx, task); err != nil {
		return nil, "", fmt.Errorf("failed to create generation task: %w", err)
	}

	nowMs := time.Now().UnixMilli()
	ar := domain.NormalizeFragmentAspectRatio(strings.TrimSpace(req.AspectRatio))
	if ar == "" {
		ar = domain.FragmentAspectDefault
	}
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
		ImageUrls:       "[]",
		Visibility:      domain.FragmentVisibilityPrivate,
		IsDraft:         true,
		SourceType:      string(domain.FragmentSourceAIGeneration),
		SourceID:        taskID,
		AspectRatio:     ar,
		EngagementStats: common.EngagementStats{},
	}
	if err := s.fragmentRepo.Create(ctx, draft); err != nil {
		if delErr := s.fragmentGenRepo.Delete(ctx, taskID); delErr != nil {
			s.logger.Warn("rollback: failed to delete generation task after draft create error",
				zap.String("task_id", taskID), zap.Error(delErr))
		}
		return nil, "", fmt.Errorf("failed to create placeholder draft: %w", err)
	}

	s.logger.Info("Fragment generation task created",
		zap.String("task_id", taskID),
		zap.String("draft_id", draft.ID),
		zap.String("user_id", userID),
		zap.Int("image_count", req.ImageCount))

	go s.processFragmentGeneration(context.Background(), taskID)

	return task, draft.ID, nil
}

const (
	fragmentMaxAnchorCharacters     = 3
	fragmentMaxAnchorProps          = 3
	fragmentMaxAnchorLocations      = 2
	fragmentMaxSceneReferenceImages = 6
)

// processFragmentGeneration 处理碎片生成流程（方案 B）：
// Step 1: 元素提取 + 文案 + VisualBible
// Step 2: 场景扩展（含 referenceKeys）
// Step 3: 锚点图
// Step 4: 逐场景出图（多参考图）
// Step 5: 一致性检查（仅记录）
func (s *FragmentGenerationService) processFragmentGeneration(ctx context.Context, taskID string) {
	s.fragmentGenRepo.UpdateStatus(ctx, taskID, "processing", 5, "starting")

	task, err := s.fragmentGenRepo.GetByID(ctx, taskID)
	if err != nil {
		s.logger.Error("Failed to get task", zap.Error(err))
		return
	}

	// ── Step 1: 元素提取 + 文案 + VisualBible ──
	s.fragmentGenRepo.UpdateStatus(ctx, taskID, "processing", 10, "extracting_elements")

	elemResult, err := s.extractElementsAndGenerateContent(ctx, task.UserID, taskID, task.Request)
	if err != nil {
		s.fragmentGenRepo.UpdateError(ctx, taskID, "failed", fmt.Sprintf("Element extraction failed: %v", err))
		s.notifyFragmentGenFailed(context.Background(), task, taskID, fmt.Sprintf("元素提取失败：%v", err))
		return
	}

	totalTokens := elemResult.TokensUsed

	resolvedAR := domain.NormalizeFragmentAspectRatio(task.Request.AspectRatio)
	if resolvedAR == "" {
		resolvedAR = domain.NormalizeFragmentAspectRatio(elemResult.AspectRatio)
	}
	if resolvedAR == "" {
		resolvedAR = domain.FragmentAspectDefault
	}

	// ── Step 2: 场景扩展 ──
	s.fragmentGenRepo.UpdateStatus(ctx, taskID, "processing", 28, "expanding_scenes")

	sceneCount := task.Request.ImageCount
	if sceneCount <= 0 {
		sceneCount = 1
	}
	scenesResult, err := s.expandScenes(ctx, task.UserID, taskID, task.Request, elemResult, sceneCount, resolvedAR)
	if err != nil {
		s.fragmentGenRepo.UpdateError(ctx, taskID, "failed", fmt.Sprintf("Scene expansion failed: %v", err))
		s.notifyFragmentGenFailed(context.Background(), task, taskID, fmt.Sprintf("场景扩展失败：%v", err))
		return
	}
	totalTokens += scenesResult.TokensUsed

	scenePlans := fragmentScenesToPlans(scenesResult.Scenes)
	entityUsage := analyzeFragmentSceneEntityUsage(scenePlans)
	policy := s.resolveFragmentConsistencyPolicy(task, elemResult.VisualBible, resolvedAR, scenePlans, entityUsage)
	trace := &domain.FragmentGenerationTrace{
		VisualBible:       elemResult.VisualBible,
		VisualEvidence:    elemResult.VisualEvidence,
		Scenes:            scenePlans,
		EntityUsage:       entityUsage,
		ConsistencyPolicy: policy,
		ProviderOptions:   policy.ProviderOptions,
		Metrics: []domain.FragmentGenerationStepMetric{
			{Name: "extracting_elements", Tokens: elemResult.TokensUsed},
			{Name: "expanding_scenes", Tokens: scenesResult.TokensUsed},
		},
	}
	partial := &domain.FragmentGenerationResult{
		Content:           elemResult.Content,
		AspectRatio:       resolvedAR,
		TokensUsed:        totalTokens,
		VisualBible:       elemResult.VisualBible,
		VisualEvidence:    elemResult.VisualEvidence,
		ScenePlan:         append([]domain.FragmentScenePlan(nil), scenePlans...),
		ConsistencyPolicy: policy,
		GenerationTrace:   trace,
	}
	if err := s.fragmentGenRepo.UpdateResult(ctx, taskID, partial); err != nil {
		s.logger.Warn("failed to persist fragment generation trace after planning", zap.Error(err))
	}

	// ── Step 3: 按需参考资产 ──
	s.fragmentGenRepo.UpdateStatus(ctx, taskID, "processing", 45, "generating_reference_assets")
	assetStart := time.Now()
	referenceAssets, refTok := s.generateReferenceAssets(ctx, task.UserID, taskID, task.Request, elemResult.VisualBible, elemResult.VisualEvidence, resolvedAR, scenePlans, policy)
	totalTokens += refTok
	trace.ReferenceAssets = referenceAssets
	trace.Metrics = append(trace.Metrics, domain.FragmentGenerationStepMetric{Name: "reference_assets", Tokens: refTok, DurationMs: time.Since(assetStart).Milliseconds()})
	partial.TokensUsed = totalTokens
	partial.ReferenceAssets = referenceAssets
	if err := s.fragmentGenRepo.UpdateResult(ctx, taskID, partial); err != nil {
		s.logger.Warn("failed to persist fragment generation trace after reference assets", zap.Error(err))
	}

	// ── Step 4: 逐场景生成图片 ──
	s.fragmentGenRepo.UpdateStatus(ctx, taskID, "processing", 62, "generating_images")
	userRefs := fragmentPrefillHTTPImageURLs(task.Request.ImageUrls, 2)
	imageStart := time.Now()
	imageResult, err := s.generateImagesFromScenes(ctx, task.UserID, taskID, resolvedAR, elemResult.VisualBible, scenePlans, referenceAssets, userRefs, policy)
	if err != nil {
		s.fragmentGenRepo.UpdateError(ctx, taskID, "failed", fmt.Sprintf("Image generation failed: %v", err))
		s.notifyFragmentGenFailed(context.Background(), task, taskID, fmt.Sprintf("配图生成失败：%v", err))
		return
	}
	totalTokens += imageResult.TokensUsed
	trace.Scenes = scenePlans
	trace.Metrics = append(trace.Metrics, domain.FragmentGenerationStepMetric{Name: "scene_images", Tokens: imageResult.TokensUsed, DurationMs: time.Since(imageStart).Milliseconds()})

	// ── Step 5: 一致性检查（不阻断） ──
	s.fragmentGenRepo.UpdateStatus(ctx, taskID, "processing", 82, "checking_consistency")
	checkStart := time.Now()
	issues, checkTok, auditProvider, auditedCount, skippedAuditReason := s.checkFragmentConsistency(ctx, taskID, elemResult.VisualBible, elemResult.VisualEvidence, scenePlans, referenceAssets, imageResult.ImageUrls, policy)
	totalTokens += checkTok
	trace.ConsistencyIssues = issues
	trace.VisionAuditProvider = auditProvider
	trace.AuditedImageCount = auditedCount
	trace.SkippedAuditReason = skippedAuditReason
	trace.Metrics = append(trace.Metrics, domain.FragmentGenerationStepMetric{Name: "checking_consistency", Tokens: checkTok, DurationMs: time.Since(checkStart).Milliseconds()})
	if len(issues) > 0 {
		for _, iss := range issues {
			s.logger.Info("fragment consistency issue",
				zap.String("task_id", taskID),
				zap.Int("scene_index", iss.SceneIndex),
				zap.String("severity", iss.Severity),
				zap.String("detail", iss.Detail))
		}
	}

	result := &domain.FragmentGenerationResult{
		Content:           elemResult.Content,
		ImageUrls:         imageResult.ImageUrls,
		AspectRatio:       resolvedAR,
		TokensUsed:        totalTokens,
		VisualBible:       elemResult.VisualBible,
		VisualEvidence:    elemResult.VisualEvidence,
		ReferenceAssets:   referenceAssets,
		ScenePlan:         scenePlans,
		ConsistencyPolicy: policy,
		GenerationTrace:   trace,
		ConsistencyIssues: issues,
	}

	// 更新占位草稿
	now := time.Now().UnixMilli()
	imgCount := len(result.ImageUrls)
	if imgCount < 1 {
		imgCount = 1
	}
	style := task.Request.Style
	caption := captionFromGenerationContent(result.Content)
	generationMetadata := marshalFragmentGenerationMetadata(result.GenerationTrace)

	existing, gerr := s.fragmentRepo.GetBySource(ctx, string(domain.FragmentSourceAIGeneration), taskID)
	if gerr == nil && existing != nil {
		existing.Content = result.Content
		existing.MediaURLs = append([]string(nil), result.ImageUrls...)
		existing.ImageUrls = stringifyGenerationImageURLs(result.ImageUrls)
		existing.Style = &style
		existing.FragmentCount = &imgCount
		existing.GenerationTaskID = taskID
		existing.GenerationMetadata = generationMetadata
		existing.UpdatedAt = now
		existing.AspectRatio = resolvedAR
		if caption != "" {
			existing.Caption = caption
		}
		if err := s.fragmentRepo.Update(ctx, existing); err != nil {
			s.logger.Error("Failed to update draft fragment from generation",
				zap.String("task_id", taskID),
				zap.String("draft_id", existing.ID),
				zap.Error(err))
		} else {
			result.DraftFragmentID = existing.ID
		}
	} else {
		fragment := &domain.Fragment{
			BaseModel: common.BaseModel{
				ID:        uuid.New().String(),
				CreatedAt: now,
				UpdatedAt: now,
			},
			UserID:             task.UserID,
			CreatorID:          task.UserID,
			Content:            result.Content,
			MediaURLs:          append([]string(nil), result.ImageUrls...),
			ImageUrls:          stringifyGenerationImageURLs(result.ImageUrls),
			Style:              &style,
			FragmentCount:      &imgCount,
			Visibility:         domain.FragmentVisibilityPrivate,
			IsDraft:            true,
			SourceType:         string(domain.FragmentSourceAIGeneration),
			SourceID:           taskID,
			GenerationTaskID:   taskID,
			GenerationMetadata: generationMetadata,
			AspectRatio:        resolvedAR,
			EngagementStats:    common.EngagementStats{},
		}
		if caption != "" {
			fragment.Caption = caption
		}
		if err := s.fragmentRepo.Create(ctx, fragment); err != nil {
			s.logger.Error("Failed to create draft fragment from generation",
				zap.String("task_id", taskID),
				zap.Error(err))
		} else {
			result.DraftFragmentID = fragment.ID
		}
	}
	if result.DraftFragmentID != "" {
		s.persistFragmentGenerationAssets(ctx, result.DraftFragmentID, taskID, task.Request, result)
	}

	if err := s.fragmentGenRepo.UpdateResult(ctx, taskID, result); err != nil {
		s.logger.Error("Failed to update result", zap.Error(err))
	}

	s.fragmentGenRepo.UpdateStatus(ctx, taskID, "completed", 100, "completed")

	s.logger.Info("Fragment generation completed",
		zap.String("task_id", taskID),
		zap.Int("tokens_used", result.TokensUsed),
		zap.Int("image_count", len(result.ImageUrls)),
		zap.Int("scenes", len(scenesResult.Scenes)),
		zap.Int("reference_assets", len(referenceAssets)))

	if s.notify != nil && strings.TrimSpace(result.DraftFragmentID) != "" {
		preview := captionFromGenerationContent(result.Content)
		if preview == "" && len(result.ImageUrls) > 0 {
			preview = "新碎片"
		}
		if err := s.notify.NotifyFragmentGenerationCompleted(context.Background(), task.UserID, result.DraftFragmentID, preview, result.TokensUsed); err != nil {
			s.logger.Warn("fragment generation completion notify failed",
				zap.Error(err),
				zap.String("task_id", taskID),
				zap.String("fragment_id", result.DraftFragmentID))
		}
	}
}

func (s *FragmentGenerationService) notifyFragmentGenFailed(ctx context.Context, task *domain.FragmentGenerationTask, taskID, userFacingReason string) {
	if s.notify == nil || task == nil {
		return
	}
	frag, err := s.fragmentRepo.GetBySource(ctx, string(domain.FragmentSourceAIGeneration), taskID)
	if err != nil || frag == nil {
		return
	}
	draftID := strings.TrimSpace(frag.ID)
	if draftID == "" || strings.TrimSpace(task.UserID) == "" {
		return
	}
	if err := s.notify.NotifyFragmentGenerationFailed(ctx, task.UserID, draftID, userFacingReason); err != nil {
		s.logger.Warn("fragment generation failure notify failed",
			zap.Error(err),
			zap.String("task_id", taskID),
			zap.String("draft_id", draftID))
	}
}

func stringifyGenerationImageURLs(urls []string) string {
	if len(urls) == 0 {
		return "[]"
	}
	b, err := json.Marshal(urls)
	if err != nil {
		return "[]"
	}
	return string(b)
}

func marshalFragmentGenerationMetadata(trace *domain.FragmentGenerationTrace) string {
	if trace == nil {
		return "{}"
	}
	b, err := json.Marshal(trace)
	if err != nil {
		return "{}"
	}
	return string(b)
}

func captionFromGenerationContent(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	runes := []rune(s)
	if len(runes) > 32 {
		return string(runes[:32])
	}
	return s
}

// ────────────────────── Step 1: 元素提取 + 文案生成 ──────────────────────

// fragmentElementExtractionResult 元素提取 + 文案生成的输出。
type fragmentElementExtractionResult struct {
	Elements       fragmentStoryElements           `json:"elements"`
	VisualBible    *domain.FragmentVisualBible     `json:"visualBible,omitempty"`
	VisualEvidence []domain.FragmentVisualEvidence `json:"visualEvidence,omitempty"`
	Content        string                          `json:"content"`
	AspectRatio    string                          `json:"aspectRatio"`
	TokensUsed     int
}

// fragmentStoryElements 从用户输入和参考图中提取的结构化元素。
type fragmentStoryElements struct {
	Weather    string   `json:"weather"`    // 天气：晴朗、雨天、暴风雪等
	Objects    []string `json:"objects"`    // 关键物品
	Scenes     []string `json:"scenes"`     // 场景类型：室内/室外/森林/城市等
	TimeOfDay  string   `json:"timeOfDay"`  // 时间：清晨/正午/黄昏/深夜等
	Location   string   `json:"location"`   // 地点描述
	Characters []string `json:"characters"` // 人物/角色描述
	Tendency   string   `json:"tendency"`   // 情感倾向/叙事方向
}

func (s *FragmentGenerationService) extractElementsAndGenerateContent(ctx context.Context, userID, taskID string, req domain.FragmentGenerationRequest) (*fragmentElementExtractionResult, error) {
	imageURLs := fragmentPrefillHTTPImageURLs(req.ImageUrls, 10)
	hasImages := len(imageURLs) > 0
	var visualEvidence []domain.FragmentVisualEvidence
	visionTokens := 0
	if hasImages {
		evidence, tokens, err := s.analyzeFragmentVisualEvidence(ctx, taskID, imageURLs, req.UserInput, req.Style, "")
		if err != nil {
			s.logger.Warn("fragment visual evidence analysis failed; falling back to legacy multimodal extraction", zap.Error(err))
		} else {
			visualEvidence = evidence
			visionTokens = tokens
		}
	}
	prompt := s.buildExtractionAndStoryPrompt(req, hasImages)
	if len(visualEvidence) > 0 {
		prompt = prompt + "\n\n" + formatFragmentVisualEvidenceForPrompt(visualEvidence)
	}

	payload := map[string]interface{}{"prompt": prompt}
	if len(visualEvidence) == 0 && len(imageURLs) > 0 {
		payload["imageUrls"] = imageURLs
	}
	inputBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal extraction input: %w", err)
	}

	aiReq := domain.AITask{
		ID:                uuid.New().String(),
		UserID:            userID,
		Type:              domain.AITaskGenerateFragmentContent,
		Status:            domain.AITaskStatusProcessing,
		Provider:          "",
		Input:             string(inputBytes),
		RelatedEntityID:   taskID,
		RelatedEntityType: "fragment_generation",
	}

	raw, tokensUsed, err := s.aiService.GenerateFragmentExtractionJSON(ctx, &aiReq)
	if err != nil {
		return nil, fmt.Errorf("AI extraction+story generation failed: %w", err)
	}

	result, perr := parseExtractionResult(raw)
	if perr != nil {
		s.logger.Warn("extraction JSON parse failed, using raw text as content",
			zap.Error(perr),
			zap.String("snippet", truncateRunes(raw, 120)))
		if c, ar, e2 := parseFragmentMultimodalStoryJSON(raw); e2 == nil {
			result = &fragmentElementExtractionResult{
				Content:     c,
				AspectRatio: ar,
			}
		} else {
			result = &fragmentElementExtractionResult{
				Content: truncateRunes(raw, 500),
			}
		}
	}
	if result.AspectRatio == "" {
		if _, ar, e2 := parseFragmentMultimodalStoryJSON(raw); e2 == nil {
			result.AspectRatio = ar
		}
	}
	result.VisualEvidence = visualEvidence
	if result.VisualBible != nil && len(result.VisualBible.SourceEvidence) == 0 {
		result.VisualBible.SourceEvidence = visualEvidence
	}
	result.TokensUsed = tokensUsed + visionTokens
	return result, nil
}

// buildExtractionAndStoryPrompt 构建元素提取 + 文案生成的组合提示词。
func (s *FragmentGenerationService) buildExtractionAndStoryPrompt(req domain.FragmentGenerationRequest, hasImages bool) string {
	styleDesc := fragmentStyleDesc(req.Style)
	moodDesc := fragmentMoodDesc(req.Mood)
	lengthDesc := fragmentLengthDesc(req.Length)
	language := strings.TrimSpace(req.Language)
	if language == "" {
		language = "中文"
	}

	imgNote := ""
	if hasImages {
		imgNote = `

【参考图深度分析】用户提供了参考图，你不仅要"看懂"这张图，还要像一位电影美术指导一样把它拆解为可复用的视觉资产库。

第一阶段——直觉反应（先不要分析，记录你的第一感觉）：
- 这张图给你的第一情绪是什么？用一个形容词回答（如：孤独的、温暖的、诡异的、怀旧的……）
- 你的视线最先落在画面的哪个位置？是被什么吸引过去的？（色彩对比？人物表情？光线的方向？某个反常的细节？）
- 如果这是一部电影的一帧截图，你会猜测这部电影是什么类型？什么年代？讲什么故事？

第二阶段——系统性拆解（逐层分析画面的视觉构成）：

  A. 人物/主体层：
  - 外观精确描述：年龄感（少年/青年/中年/老年）、体型（高矮胖瘦）、发型发色、肤色
  - 穿着细节：服装款式（T恤/衬衫/裙子/外套）、颜色、材质（棉/丝/皮/毛）、是否有图案或文字
  - 姿态与动作：站/坐/跑/蹲/躺/跳、手在做什么、身体朝向（正面/45度侧/全侧/背面）、重心偏移
  - 面部信息：表情（或模糊/遮挡/背影）、视线方向（看镜头/看画外/低头/仰望）
  - 标志性视觉特征：纹身、疤痕、饰品（项链/戒指/耳环）、帽子/围巾/背包等配饰

  B. 环境空间层：
  - 空间类型：室内/室外/半室外（走廊/阳台/天台）、人造环境/自然环境/混合
  - 建筑或地貌：建筑风格（现代/古典/赛博朋克/废墟）、年代痕迹（新旧程度、破损、苔藓、灰尘）
  - 季节与温度：季节暗示（落叶/新绿/积雪/蝉鸣）、温度感（炎热蒸腾/寒冷刺骨/凉爽舒适）
  - 空间层次：前景有什么（遮挡物/框架元素）、中景主体、背景延伸（天际线/远山/墙壁截止）
  - 纵深感：是否有明确的透视线（道路/走廊/河流引导视线）、空间是开阔还是封闭

  C. 光影层：
  - 光源类型与方向：自然光（太阳从哪个方向）还是人造光（灯光/火光/屏幕光）、主光源位置
  - 色温：暖黄光（日落/烛火/白炽灯）、冷蓝光（月光/荧光灯/电子屏幕）、中性白（正午日光）
  - 阴影：阴影的浓淡（硬阴影/柔阴影）、阴影的形状和方向、是否有趣味性阴影图案
  - 高光：画面中最亮的部分在哪里、高光是集中还是散射、是否有镜头光晕或光斑

  D. 色彩层：
  - 主色调：画面面积最大的颜色是什么、这个颜色的情绪暗示
  - 点缀色：与主色调形成对比或互补的颜色、这些颜色出现在画面中的哪些位置
  - 饱和度：整体是高饱和（鲜艳）还是低饱和（灰调/褪色）、饱和度在画面中是否均匀
  - 色彩渐变：画面中是否有色彩渐变（天空从蓝到橙、光线从暖到冷的过渡）

  E. 构图层：
  - 景别：特写（面部或细节）/ 近景（上半身）/ 中景（全身）/ 远景（人物渺小）
  - 角度：平视/俯拍/仰拍/Dutch angle（倾斜）
  - 画面重心：主体在画面的什么位置（居中/三分法/边缘）
  - 引导线：画面中是否有线条（道路/建筑边缘/光影分界）引导视线

  F. 细节彩蛋层：
  - 画面中容易被忽略但有趣的微观细节（墙角的涂鸦、书桌上的便利贴、远处的动物剪影、口袋里露出的信封一角、地上的水洼倒影）
  - 这些细节可能暗示了什么故事信息

第三阶段——视觉信息的故事化转译：
- 人物外观 → characters 的视觉身份卡（发型/服装/标志性特征）
- 环境与季节 → scenes 的空间氛围描述
- 光影与色彩 → weather 的情绪化天气描述
- 构图与细节 → tendency 的核心感受方向
- 确保生成的故事与参考图在画面上有真实呼应，而不是只基于文字做文字推理
- 从参考图中提取的元素应该让读者感觉"这个故事确实可能发生在那张图片里"

第四阶段——冲突裁决与创意边界：
- 若参考图与用户文字存在轻微冲突：以用户文字为剧情主线，以参考图为视觉锚点
- 允许创造惊喜与反转，但不能引入与图文都无关的核心设定
- 每个关键转折都要能在 elements 或原始输入中找到伏笔或视觉依据`
	}

	legacyPrompt := fmt.Sprintf(`你是一位兼具文学才华和电影美术指导素养的故事碎片创作大师。你同时拥有小说家的叙事直觉、摄影师的画面敏感度、和漫画编辑的商业嗅觉——你知道什么样的内容能让人停下手指、点开、读完、然后截图转发。你需要完成两件事：

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
第一件事：元素提取（把模糊的感觉变成精确的视觉指令）
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

从用户的文字描述%s中提炼出以下故事元素。

核心原则：元素不是关键词罗列，而是带有画面感和叙事暗示的具体描述。检验标准是——把这些元素交给一位从未读过原文的插画师，他仅凭这些描述就能画出"对的画面"。

逐项说明（每项都附有好的和差的示例，请向好的看齐）：

1. weather（天气）—— 情绪的物理化
   天气不是背景板，它是情绪的放大器和隐喻。不要只写"晴天/雨天"，要写出天气如何渗透进画面和角色的感受中。
   好的示例："暴雨将至，天色发黄，空气黏稠得像裹了一层保鲜膜，远处传来闷雷但还没下雨"
   好的示例："深秋的晴夜，冷得能看见自己呼出的白气，月光把地上的落叶冻成了银色"
   差的示例："阴天" / "下雨"
   如果天气确实不重要，写"无特别天气"——但优先考虑天气能如何强化你想讲的故事的情绪。

2. objects（物品）—— 故事的道具箱（最多 5 个）
   每个物品都是一个潜在的叙事线索。写出：外观 + 材质 + 当前状态 + 与角色的空间关系。
   好的示例："一把半透明的红伞，伞面有几道裂痕，被随手靠在咖啡店门边，伞尖的水渍在地砖上洇开一小片"
   好的示例："一只翻盖手机，外壳贴满了已经褪色的贴纸，屏幕上的时间停在 3:17"
   差的示例："伞" / "手机"

3. scenes（场景）—— 五感的空间（最多 3 个）
   场景不只是一个地点名称，它是一个可以用五感去体验的空间。写出视觉之外的感官细节。
   好的示例："老式电车内部，木质座椅磨得发亮，窗外是模糊的樱花隧道，车厢里有铁锈和甜食混合的味道，地板随着转弯发出低沉的吱嘎声"
   好的示例："凌晨三点的便利店，冷柜嗡嗡作响，日光灯管其中一根在闪，空气里有加热便当和消毒水的混合味道"
   差的示例："电车上" / "便利店"

4. timeOfDay（时间）—— 光线即情绪
   不要只写时段，要写出那个时段特有的光线、温度、以及它暗示的情绪氛围。
   好的示例："黄昏，太阳刚好卡在两栋楼之间，整条街被染成橘红色，路人的影子拖得像另一个人"
   好的示例："凌晨四点半，天还没亮但已经不是完全的黑，东边的天际线有一道极细的灰蓝色"
   差的示例："傍晚" / "早上"

5. location（地点）—— 有故事感的空间
   写出地点的辨识度、时代感、和空间性格（开阔/逼仄/纵深/封闭）。
   好的示例："一个被废弃的室内游乐场，旋转木马还在缓慢转，彩灯只剩红和蓝还亮着，地上散着褪色的入场券和爆米花桶"
   好的示例："高铁站的 7 号站台，电子屏显示着已经发车的班次，不锈钢长椅上只剩一个没喝完的纸杯"
   差的示例："游乐场" / "火车站"

6. characters（人物）—— 视觉身份卡（最多 3 个）
   想象你在给一位从未见过这个角色的画师做口头描述。包含：体型轮廓、穿着（款式+颜色+材质）、标志性视觉特征、当前在做的事情和表情。
   **若用户输入或正文中已有姓名、昵称或稳定称呼，描述开头须用括号注明（如「（阿明）」），且须与 visualBible 中对应角色的 name 字段完全一致，不得另起别名。**
   好的示例："瘦高的女生，穿oversized黑卫衣帽子压得很低，露出一截染了蓝色的发尾，手里攥着一杯已经凉透的拿铁，站在斑马线中间像在犹豫要不要过去"
   好的示例："中年男人，头发灰白但梳得整齐，穿着一件洗到发白的蓝色工装外套，右手插在口袋里，左手拎着一个系着红绳的旧皮箱"
   差的示例："一个女生" / "一个男人"

7. tendency（倾向）—— 这段故事的一行宣发语
   不是分类标签，而是"如果这是一张专辑封面，封面上只印这一句话"。要让读者看到这句话就想点进去看故事。
   好的示例："所有出口都标着入口的方向"
   好的示例："她等的那班列车三年前就停运了"
   好的示例："世界上最远的距离不是生与死，是你坐在对面但信号只有一格"
   差的示例："悬疑风格" / "温馨治愈"

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
第二件事：碎片故事（用文字创造让人截图转发的瞬间）
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

基于提取的元素，写一个让人看完想转发的故事碎片。

用户输入：%s

要求：
- 风格：%s
- 情绪：%s
- 长度：%s
- 语言：%s（elements 与 content 的自然语言都必须与该语言一致）

创作优先级（从高到低，不能颠倒）：
- 第一优先级：贴合用户输入与参考图锚点
- 第二优先级：画面感与可传播性
- 第三优先级：反转、奇观和风格实验

故事创作心法（这是你的创作工具箱，按主题分类）：

【钩子：第一句就锁住读者】
- 绝对不要用的开头："有一天""在很久以前""那是一个""故事开始于"——这些是读者滑走的信号
- 用以下方式之一开局：
  · 反常细节："钥匙插进锁孔的时候，她发现门是从里面锁着的。"
  · 动作进行时："他跑到便利店的时候，雨已经停了。但他的伞还是撑着的。"
  · 对话切入："你刚才说你有妹妹？" "我说过吗？"
  · 悬念前置："那张照片里多出来的人，不是后期P的。"
  · 感官冲击："空气里有烧焦的味道，但不是食物。"

【画面感：用文字画画】
核心原则：动词 > 形容词，具体名词 > 抽象概念，可观察的行为 > 内心独白。
- 不要写"她感到无比悲伤和孤独"，要写"她把第二碗面推到对面空着的座位前"
- 不要写"天空非常美丽壮观"，要写"云层裂开一道缝，光柱打在她脚边半米处"
- 不要写"他很紧张"，要写"他把同一页书翻了三次"
- 不要写"气氛很尴尬"，要写"两个人同时伸手去拿桌上的遥控器，又同时缩回去"
- 角色的情绪要"演"出来，不要"说"出来：不要写"她很害怕"，写"她的手在口袋里攥成了拳头，指甲掐进掌心"

【留白：不说完比说完更高级】
- 最好的结尾不是句号，而是让读者自己脑补下一秒发生了什么
- 不要解释你的隐喻——如果读者需要你解释，说明隐喻不够好
- 可以在最高潮处戛然而止，比写完整个结局更有余韵
- 留白的位置通常在这些地方：做出选择的那一刻、真相揭晓的前一秒、转身之后
- 技巧：最后一句可以是完全平静的日常动作——越是平静，越衬托前面的波澜

【转折：一个就够】
- 不需要反转再反转，一个"等等，什么？"的时刻比三个小反转有效十倍
- 好的转折来自前面埋下的细节——读者回头看时会发现"原来早就有暗示了"
- 转折类型参考：
  · 视角的转折："我"其实不是人类（是一只猫、一个AI、一颗树）
  · 时间的转折：这已经是十年前的回忆 / 这段话是遗书 / 这是未来的人在看旧照片
  · 身份的转折：对面的人其实不认识她 / "妹妹"其实是想象中的
  · 空间的转折：门外面不是走廊，是天空 / 一直以为是室内的场景其实在外面

【世界观：用细节播种，让读者自己收获】
- 不要写"这是一个魔法与科技并存的世界"，要写"咖啡杯飘在半空，她懒得去接"
- 不要写"未来社会严禁自由恋爱"，要写"结婚申请表上有一栏：请填写恋爱许可证编号"
- 一个反常的日常细节比三句世界观说明更有说服力
- 让读者自己拼凑规则，不要喂给他们
- 世界观的细节应该"长"在叙事里，而不是"贴"在上面——通过角色的行为和环境的描写自然呈现

【节奏控制：快慢交替】
- 短句加速（制造紧张感、转折冲击）："她回头。没人。"
- 长句减速（铺陈氛围、内心沉淀）："她站在站台上看着列车尾灯在隧道口慢慢缩成一个红点，手里的信封被风吹得哗哗响，但她没有追过去"
- 一段安静的描写之后突然一个短句转折，效果拔群
- 句子长度本身就是情绪的呼吸——全篇匀速的文字是没有生命力的

%s

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
第三件事：视觉圣经 visualBible（多实体一致性锚点，必须输出）
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
- visualBible 必须与 elements、content 自洽；immutableTraits 使用与 elements/content 相同的自然语言（由上方 language 决定）。
- characters最多 3 项，props 最多 5 项，locations 1–2 项；每项必须有全局唯一 key（小写英文+下划线，如 char_main、prop_umbrella、loc_station）。
- **每个 visualBible.characters[] 条目必须填写 name（必填，不得省略或留空）：**表示故事中用于展示与下游「故事角色」物料的稳定称呼。**优先与用户输入、正文里对已出现人物的称呼完全一致**（真名、昵称、职务称呼均可）；若全文未命名，则用与 language 一致的**简短识别名**（如中文 2～8 个字：概括外貌/身份，如「戴帽少年」「工装店员」）；**禁止**输出「角色一」「无名氏」或与正文无关的泛称。**同一角色在正文中的称呼必须与该项 name 一致。** props / locations 的 name 若字段存在则需同理可称呼，可选用简短物体名或地点名。
- immutableTraits 为字符串数组：每条描述一个不可随意更改的视觉事实（发色、服装剪裁与主色、道具外形、建筑特征等）。
- negativeTraits 为字符串数组：写出该实体绝对不能和其他实体混淆的特征（例如“不要穿 char_child 的红雨衣”）。
- ownership 用来表达归属关系：角色可以列出随身物，道具写关联角色 key；无法判断则留空。
- roleImportance 必须是 core / supporting / background，用于后续决定是否生成 reference asset。
- styleBible.artStyle 必须用英文写出可执行的总体画法（媒介、线稿/渲染、时代感、参考流派），供后续所有配图 prompt 对齐；其他 styleBible 字段可选。
- 若风格/内容偏漫画，styleBible.lineQuality / palette / lightingMood 必须结构化体现：paneling、gutter、screentone/halftone、heavy ink、speech-bubble lettering、SFX/effect-lines 的总体策略；冲击感语义触发时写入 high contrast / chiaroscuro / extreme angle / action lines 等具体选择。

请只输出一个 JSON 对象（不要 markdown 代码围栏、不要其他说明），字段为：
{
  "elements": {
    "weather": "天气氛围描述——写出天气如何渗透进画面和情绪（与上方语言要求一致）",
    "objects": ["物品1：外观+材质+状态+与角色空间关系", "物品2"],
    "scenes": ["场景1：五感空间描述（视觉+听觉+嗅觉+触感+温度）", "场景2"],
    "timeOfDay": "时段+该时段特有的光线氛围（与上方语言要求一致）",
    "location": "具体地点+时代感+空间性格（开阔/封闭/纵深）+标志性细节（与上方语言要求一致）",
    "characters": ["角色1：体型+穿着（款式颜色材质）+标志性特征+当前动作和表情", "角色2"],
    "tendency": "核心感受一句话——像专辑封面或电影海报上印的那一行字（与上方语言要求一致）"
  },
  "visualBible": {
    "styleBible": {
      "artStyle": "English: overall rendering and line quality for all images",
      "lineQuality": "optional English",
      "palette": "optional English",
      "lightingMood": "optional English"
    },
    "characters": [
      { "key": "char_main", "name": "与正文及用户输入一致的故事角色名或简短识别名（必填）", "immutableTraits": ["trait1", "trait2"], "negativeTraits": ["must not share traits with char_other"], "ownership": ["prop_key"], "roleImportance": "core" }
    ],
    "props": [
      { "key": "prop_key", "immutableTraits": ["..."], "negativeTraits": ["..."], "ownership": "char_main", "roleImportance": "core" }
    ],
    "locations": [
      { "key": "loc_key", "immutableTraits": ["..."], "negativeTraits": ["..."], "roleImportance": "supporting" }
    ]
  },
  "content": "碎片故事正文（与上方语言要求一致），直接可读，不要标题，不要前缀说明",
  "aspectRatio": "推荐长宽比：1:1、16:9、9:16、3:4、4:3，结合画面构图选择；不确定用 16:9"
}`,
		imgNote,
		req.UserInput, styleDesc, moodDesc, lengthDesc, language,
		structuredMangaLanguageGuidance())

	return renderPromptDSL(PromptDSL{
		Role:         "你是一位兼具文学叙事与漫画视觉导演能力的故事碎片创作大师。",
		Task:         "先做元素提取与视觉圣经，再生成可传播的故事碎片正文，并保证后续可用于分镜/配图。",
		Inputs:       map[string]any{"userInput": req.UserInput, "styleSlug": req.Style, "styleDescription": styleDesc, "mood": req.Mood, "moodDescription": moodDesc, "length": req.Length, "lengthDescription": lengthDesc, "language": language, "hasReferenceImages": hasImages},
		GlobalConfig: structuredMangaLanguageGuidance(),
		Sections: []PromptDSLSection{
			{Title: "Detailed Instructions", Kind: "text", Body: legacyPrompt},
		},
	})
}

type fragmentVisualEvidenceRoot struct {
	Evidence []domain.FragmentVisualEvidence `json:"evidence"`
	Images   []domain.FragmentVisualEvidence `json:"images"`
}

func (s *FragmentGenerationService) analyzeFragmentVisualEvidence(ctx context.Context, taskID string, imageURLs []string, userInput, style, providerHint string) ([]domain.FragmentVisualEvidence, int, error) {
	imageURLs = fragmentPrefillHTTPImageURLs(imageURLs, 10)
	if len(imageURLs) == 0 || s.aiService == nil {
		return nil, 0, nil
	}
	prompt := buildFragmentVisualEvidencePrompt(userInput, style, imageURLs)
	resp, err := s.aiService.GenerateFragmentVisionJSON(ctx, &FragmentVisionJSONRequest{
		Prompt:            prompt,
		ImageURLs:         imageURLs,
		ProviderHint:      providerHint,
		MaxTokens:         4096,
		Temperature:       0.25,
		RelatedEntityType: "fragment_generation",
		RelatedEntityID:   taskID,
		Step:              "visual_evidence",
	})
	if err != nil {
		return nil, 0, err
	}
	evidence, err := parseFragmentVisualEvidence(resp.Raw, imageURLs)
	if err != nil {
		return nil, resp.TokensUsed, err
	}
	for i := range evidence {
		if evidence[i].Provider == "" {
			evidence[i].Provider = resp.Provider
		}
		if evidence[i].Model == "" {
			evidence[i].Model = resp.Model
		}
	}
	return evidence, resp.TokensUsed, nil
}

func buildFragmentVisualEvidencePrompt(userInput, style string, imageURLs []string) string {
	return fmt.Sprintf(`你是视觉事实分析员。请只描述图片中能直接观察到的视觉事实，不要编故事，不要把用户文字里的设定当成图片事实。

用户文字（仅用于理解用户关注点，不可当作图片事实）：
%s

风格倾向：%s

请逐张图片输出 JSON，格式必须是：
{"evidence":[{"imageUrl":"对应图片URL","summary":"一句话视觉概括","subjects":["主体"],"entities":[{"key":"char_0|prop_0|loc_0","kind":"character|prop|location","name":"若为 character 必填：可被读者称呼的简短名字或识别名","traits":["稳定可见特征"],"position":"画面位置","ownerKey":"可为空","confidence":0.0}],"palette":["主色"],"lighting":"光线事实","composition":"构图事实","immutableTraits":["最应保持的不可变视觉事实"],"confidence":0.0}]}

要求：
- **kind 为 character 时，实体 name 字段必填**：结合用户文字中已给出的称呼优先使用（用户文字虽未当作图片事实，但可用于命名）；若没有明确姓名则根据可见外观写与 language 一致的简短识别名（中文建议 2～8 字，如「棒球帽男子」），便于后续生成「故事角色」标题。**禁止**对人像留空或使用「人物1」类占位。**kind 非 character 时** name 可用简短道具/地点称谓或简述。
- traits 和 immutableTraits 只能写可见事实，例如发型、服装颜色材质、体型轮廓、标志物、建筑/空间特征、主色、光源方向。
- 对不确定信息降低 confidence，不要补全不可见的脸、衣服或背景。
- key 要稳定，人物用 char_0/char_1，道具用 prop_0/prop_1，地点用 loc_0/loc_1。
- imageUrl 必须从以下 URL 中选择：%s`, strings.TrimSpace(userInput), strings.TrimSpace(style), strings.Join(imageURLs, ", "))
}

func parseFragmentVisualEvidence(raw string, fallbackURLs []string) ([]domain.FragmentVisualEvidence, error) {
	s := strings.TrimSpace(raw)
	if m := jsonFenceRE.FindStringSubmatch(s); len(m) > 1 {
		s = strings.TrimSpace(m[1])
	}
	s = extractJSON(s)
	var root fragmentVisualEvidenceRoot
	if err := json.Unmarshal([]byte(s), &root); err != nil {
		var arr []domain.FragmentVisualEvidence
		if err2 := json.Unmarshal([]byte(s), &arr); err2 != nil {
			return nil, fmt.Errorf("parse visual evidence JSON: %w", err)
		}
		root.Evidence = arr
	}
	out := root.Evidence
	if len(out) == 0 {
		out = root.Images
	}
	for i := range out {
		if strings.TrimSpace(out[i].ImageURL) == "" && i < len(fallbackURLs) {
			out[i].ImageURL = fallbackURLs[i]
		}
		if out[i].Confidence < 0 {
			out[i].Confidence = 0
		}
		if out[i].Confidence > 1 {
			out[i].Confidence = 1
		}
		for j := range out[i].Entities {
			if out[i].Entities[j].Confidence < 0 {
				out[i].Entities[j].Confidence = 0
			}
			if out[i].Entities[j].Confidence > 1 {
				out[i].Entities[j].Confidence = 1
			}
		}
	}
	return out, nil
}

func formatFragmentVisualEvidenceForPrompt(evidence []domain.FragmentVisualEvidence) string {
	if len(evidence) == 0 {
		return ""
	}
	b, err := json.Marshal(evidence)
	if err != nil {
		return ""
	}
	return "【多模态视觉事实】以下 JSON 是模型对参考图的直接观察。生成 visualBible 时，immutableTraits 必须优先来自这些可见事实；若需要创作补充，不能写入不可变特征。\n" + string(b)
}

// parseExtractionResult 解析元素提取 + 文案的 JSON 输出。
func parseExtractionResult(raw string) (*fragmentElementExtractionResult, error) {
	s := strings.TrimSpace(raw)
	if m := jsonFenceRE.FindStringSubmatch(s); len(m) > 1 {
		s = strings.TrimSpace(m[1])
	}
	var out fragmentElementExtractionResult
	if err := json.Unmarshal([]byte(s), &out); err != nil {
		return nil, fmt.Errorf("parse extraction JSON: %w", err)
	}
	if strings.TrimSpace(out.Content) == "" {
		return nil, fmt.Errorf("extraction result missing content")
	}
	normalizeFragmentVisualBible(&out.VisualBible)
	return &out, nil
}

func normalizeFragmentVisualBible(b **domain.FragmentVisualBible) {
	if b == nil || *b == nil {
		return
	}
	bb := *b
	for i := range bb.Characters {
		if strings.TrimSpace(bb.Characters[i].Key) == "" {
			bb.Characters[i].Key = fmt.Sprintf("char_%d", i)
		}
	}
	for i := range bb.Props {
		if strings.TrimSpace(bb.Props[i].Key) == "" {
			bb.Props[i].Key = fmt.Sprintf("prop_%d", i)
		}
	}
	for i := range bb.Locations {
		if strings.TrimSpace(bb.Locations[i].Key) == "" {
			bb.Locations[i].Key = fmt.Sprintf("loc_%d", i)
		}
	}
	if bb.StyleBible != nil {
		empty := strings.TrimSpace(bb.StyleBible.ArtStyle) == "" &&
			strings.TrimSpace(bb.StyleBible.LineQuality) == "" &&
			strings.TrimSpace(bb.StyleBible.Palette) == "" &&
			strings.TrimSpace(bb.StyleBible.LightingMood) == ""
		if empty {
			bb.StyleBible = nil
		}
	}
	hasAssets := len(bb.Characters) > 0 || len(bb.Props) > 0 || len(bb.Locations) > 0 || bb.StyleBible != nil
	if !hasAssets {
		*b = nil
	}
}

func collectVisualBibleKeys(b *domain.FragmentVisualBible) []string {
	if b == nil {
		return nil
	}
	var keys []string
	for _, c := range b.Characters {
		if k := strings.TrimSpace(c.Key); k != "" {
			keys = append(keys, k)
		}
	}
	for _, p := range b.Props {
		if k := strings.TrimSpace(p.Key); k != "" {
			keys = append(keys, k)
		}
	}
	for _, loc := range b.Locations {
		if k := strings.TrimSpace(loc.Key); k != "" {
			keys = append(keys, k)
		}
	}
	return keys
}

func formatVisualBibleKeyListForPrompt(b *domain.FragmentVisualBible) string {
	if len(collectVisualBibleKeys(b)) == 0 {
		return "（无可用 key；每格 referenceKeys 输出 []）"
	}
	return strings.Join(collectVisualBibleKeys(b), ", ")
}

func mergeFragmentSceneReferenceImages(userURLs []string, scene fragmentExpandedScene, anchorMap map[string]string, maxTotal int) []string {
	if maxTotal <= 0 {
		maxTotal = fragmentMaxSceneReferenceImages
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
	for _, k := range scene.ReferenceKeys {
		if u := anchorMap[strings.TrimSpace(k)]; u != "" {
			add(u)
			if len(out) >= maxTotal {
				return out
			}
		}
	}
	return out
}

func fragmentScenesToPlans(scenes []fragmentExpandedScene) []domain.FragmentScenePlan {
	out := make([]domain.FragmentScenePlan, 0, len(scenes))
	for i, sc := range scenes {
		idx := sc.Index
		if idx < 0 {
			idx = i
		}
		out = append(out, domain.FragmentScenePlan{
			Index:          idx,
			SceneDesc:      strings.TrimSpace(sc.SceneDesc),
			ImagePrompt:    strings.TrimSpace(sc.ImagePrompt),
			ReferenceKeys:  normalizeFragmentKeyList(sc.ReferenceKeys),
			EntityBindings: sc.EntityBindings,
			ComicTexts:     normalizeFragmentComicTexts(sc.ComicTexts),
		})
	}
	return out
}

func normalizeFragmentKeyList(keys []string) []string {
	seen := map[string]struct{}{}
	var out []string
	for _, k := range keys {
		k = strings.TrimSpace(k)
		if k == "" {
			continue
		}
		if _, ok := seen[k]; ok {
			continue
		}
		seen[k] = struct{}{}
		out = append(out, k)
	}
	return out
}

func normalizeFragmentComicTexts(texts []domain.FragmentComicText) []domain.FragmentComicText {
	if len(texts) == 0 {
		return nil
	}
	out := make([]domain.FragmentComicText, 0, len(texts))
	for _, item := range texts {
		typ := strings.TrimSpace(strings.ToLower(item.Type))
		switch typ {
		case "narration", "dialogue", "thought", "sfx":
		default:
			continue
		}
		text := strings.TrimSpace(item.Text)
		if text == "" {
			continue
		}
		out = append(out, domain.FragmentComicText{
			Type:     typ,
			Text:     truncateRunes(text, 40),
			Speaker:  strings.TrimSpace(item.Speaker),
			Position: strings.TrimSpace(item.Position),
		})
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func analyzeFragmentSceneEntityUsage(scenes []domain.FragmentScenePlan) map[string]int {
	usage := make(map[string]int)
	for _, sc := range scenes {
		for _, k := range normalizeFragmentKeyList(sc.ReferenceKeys) {
			usage[k]++
		}
	}
	return usage
}

func (s *FragmentGenerationService) resolveFragmentConsistencyPolicy(task *domain.FragmentGenerationTask, bible *domain.FragmentVisualBible, aspectRatio string, scenes []domain.FragmentScenePlan, usage map[string]int) *domain.FragmentConsistencyPolicy {
	level := normalizeFragmentConsistencyLevel(task.Request.ConsistencyLevel)
	enableRefs := level != "off"
	if task.Request.EnableReferenceAssets != nil {
		enableRefs = *task.Request.EnableReferenceAssets
	}
	if len(scenes) <= 2 && countRepeatedFragmentCharacters(bible, usage) <= 1 && level == "standard" {
		enableRefs = false
	}
	maxChars, maxProps, maxLocs := 2, 0, 0
	if level == "strong" {
		maxChars, maxProps, maxLocs = 3, 1, 1
	}
	seriesSeed := fragmentStableSeed(task.ID, task.Request.Style, aspectRatio)
	options := map[string]interface{}{
		"consistency_group_id": task.ID,
		"series_seed":          seriesSeed,
		"style_consistency":    true,
	}
	if len(scenes) > 1 {
		options["sequential_image_generation"] = "auto"
		options["sequential_image_generation_options"] = map[string]interface{}{
			"consistency": level,
			"scene_count": len(scenes),
		}
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

func normalizeFragmentConsistencyLevel(level string) string {
	switch strings.TrimSpace(strings.ToLower(level)) {
	case "off", "none":
		return "off"
	case "strong", "high":
		return "strong"
	default:
		return "standard"
	}
}

func fragmentStableSeed(parts ...string) int {
	h := fnv.New32a()
	for _, p := range parts {
		_, _ = h.Write([]byte(strings.TrimSpace(p)))
		_, _ = h.Write([]byte{0})
	}
	v := int(h.Sum32() & 0x7fffffff)
	if v == 0 {
		return 1
	}
	return v
}

// fragmentStoryImageSeed is the RNG seed shared by every image in one fragment task (same as provider options.series_seed).
func fragmentStoryImageSeed(policy *domain.FragmentConsistencyPolicy) int {
	if policy == nil {
		return 0
	}
	if policy.SeriesSeed > 0 {
		return policy.SeriesSeed
	}
	return 1
}

func countRepeatedFragmentCharacters(bible *domain.FragmentVisualBible, usage map[string]int) int {
	if bible == nil {
		n := 0
		for k, c := range usage {
			if strings.HasPrefix(k, "char_") && c > 1 {
				n++
			}
		}
		return n
	}
	n := 0
	for _, ch := range bible.Characters {
		if usage[strings.TrimSpace(ch.Key)] > 1 {
			n++
		}
	}
	return n
}

type fragmentReferenceAssetCandidate struct {
	Key        string
	Kind       string
	Name       string
	Traits     []string
	UsageCount int
}

func (s *FragmentGenerationService) generateReferenceAssets(ctx context.Context, userID, genTaskID string, req domain.FragmentGenerationRequest, bible *domain.FragmentVisualBible, evidence []domain.FragmentVisualEvidence, aspectRatio string, scenes []domain.FragmentScenePlan, policy *domain.FragmentConsistencyPolicy) ([]domain.FragmentReferenceAsset, int) {
	if bible == nil || policy == nil || !policy.EnableReferenceAssets {
		return nil, 0
	}
	candidates := selectFragmentReferenceAssetCandidatesWithEvidence(bible, evidence, scenes, policy)
	if len(candidates) == 0 {
		return nil, 0
	}
	ar := domain.NormalizeFragmentAspectRatio(aspectRatio)
	if ar == "" {
		ar = domain.FragmentAspectDefault
	}
	styleEN := ""
	if bible.StyleBible != nil {
		styleEN = strings.TrimSpace(bible.StyleBible.ArtStyle)
	}
	styleZH := fragmentStyleDesc(req.Style)
	moodZH := fragmentMoodDesc(req.Mood)
	userRef := fragmentPrefillHTTPImageURLs(req.ImageUrls, 1)
	firstChar := true
	totalTok := 0
	assets := make([]domain.FragmentReferenceAsset, 0, len(candidates))
	for _, c := range candidates {
		prompt := buildFragmentReferenceAssetPrompt(c, styleEN, styleZH, moodZH)
		var refs []string
		if c.Kind == "character" && firstChar && len(userRef) > 0 {
			refs = append(refs, userRef...)
			firstChar = false
		}
		seed := fragmentStableSeed(genTaskID, c.Kind, c.Key, strings.Join(c.Traits, "|"))
		url, tok, err := s.generateOneFragmentImageWithOptions(ctx, userID, genTaskID, prompt, ar, refs, seed, policy.ProviderOptions, 0)
		if err != nil {
			s.logger.Warn("fragment reference asset generation failed", zap.String("key", c.Key), zap.String("kind", c.Kind), zap.Error(err))
			continue
		}
		assets = append(assets, domain.FragmentReferenceAsset{
			Key:               c.Key,
			Kind:              c.Kind,
			ImageURL:          url,
			Source:            "generated",
			UsageCount:        c.UsageCount,
			GeneratedByPolicy: true,
			TraitsHash:        fragmentTraitsHash(c.Kind, c.Key, c.Traits, req.Style, ar),
			TokensUsed:        tok,
		})
		totalTok += tok
	}
	return assets, totalTok
}

func selectFragmentReferenceAssetCandidates(bible *domain.FragmentVisualBible, scenes []domain.FragmentScenePlan, policy *domain.FragmentConsistencyPolicy) []fragmentReferenceAssetCandidate {
	return selectFragmentReferenceAssetCandidatesWithEvidence(bible, nil, scenes, policy)
}

func selectFragmentReferenceAssetCandidatesWithEvidence(bible *domain.FragmentVisualBible, evidence []domain.FragmentVisualEvidence, scenes []domain.FragmentScenePlan, policy *domain.FragmentConsistencyPolicy) []fragmentReferenceAssetCandidate {
	usage := analyzeFragmentSceneEntityUsage(scenes)
	var out []fragmentReferenceAssetCandidate
	charLimit := policy.MaxCharacterAssets
	for _, ch := range bible.Characters {
		if charLimit <= 0 {
			break
		}
		key := strings.TrimSpace(ch.Key)
		if key == "" {
			continue
		}
		u := usage[key]
		core := strings.EqualFold(strings.TrimSpace(ch.RoleImportance), "core")
		if u > 1 || (policy.Level == "strong" && core && u > 0) {
			out = append(out, fragmentReferenceAssetCandidate{Key: key, Kind: "character", Name: ch.Name, Traits: mergeFragmentCandidateTraits(ch.ImmutableTraits, fragmentVisualEvidenceTraits(evidence, key, "character")), UsageCount: u})
			charLimit--
		}
	}
	propLimit := policy.MaxPropAssets
	for _, p := range bible.Props {
		if propLimit <= 0 {
			break
		}
		key := strings.TrimSpace(p.Key)
		if key == "" {
			continue
		}
		u := usage[key]
		if u > 1 && strings.EqualFold(strings.TrimSpace(p.RoleImportance), "core") {
			out = append(out, fragmentReferenceAssetCandidate{Key: key, Kind: "prop", Name: p.Name, Traits: mergeFragmentCandidateTraits(p.ImmutableTraits, fragmentVisualEvidenceTraits(evidence, key, "prop")), UsageCount: u})
			propLimit--
		}
	}
	locLimit := policy.MaxLocationAssets
	for _, loc := range bible.Locations {
		if locLimit <= 0 {
			break
		}
		key := strings.TrimSpace(loc.Key)
		if key == "" {
			continue
		}
		u := usage[key]
		if u > 1 && policy.Level == "strong" {
			out = append(out, fragmentReferenceAssetCandidate{Key: key, Kind: "location", Name: loc.Name, Traits: mergeFragmentCandidateTraits(loc.ImmutableTraits, fragmentVisualEvidenceTraits(evidence, key, "location")), UsageCount: u})
			locLimit--
		}
	}
	return out
}

func fragmentVisualEvidenceTraits(evidence []domain.FragmentVisualEvidence, key, kind string) []string {
	key = strings.TrimSpace(key)
	kind = strings.TrimSpace(kind)
	var out []string
	for _, ev := range evidence {
		for _, ent := range ev.Entities {
			if strings.EqualFold(strings.TrimSpace(ent.Kind), kind) && strings.TrimSpace(ent.Key) == key {
				out = append(out, ent.Traits...)
			}
		}
		if kind == "location" {
			out = append(out, ev.Lighting, ev.Composition)
		}
	}
	return normalizeFragmentKeyList(out)
}

func mergeFragmentCandidateTraits(base, evidence []string) []string {
	seen := map[string]struct{}{}
	var out []string
	add := func(v string) {
		v = strings.TrimSpace(v)
		if v == "" {
			return
		}
		k := strings.ToLower(v)
		if _, ok := seen[k]; ok {
			return
		}
		seen[k] = struct{}{}
		out = append(out, v)
	}
	for _, v := range evidence {
		add(v)
	}
	for _, v := range base {
		add(v)
	}
	return out
}

func buildFragmentReferenceAssetPrompt(c fragmentReferenceAssetCandidate, styleEN, styleZH, moodZH string) string {
	traits := strings.Join(c.Traits, "; ")
	prefix := strings.TrimSpace(c.Name)
	if prefix != "" {
		prefix += ". "
	}
	switch c.Kind {
	case "character":
		return fmt.Sprintf("%s%s Character design reference, single person, neutral standing pose facing camera, full body visible, clean simple studio background, high detail concept art. Immutable traits: %s. Overall mood reference: %s. Style tag: %s.", prefix, styleEN, traits, moodZH, styleZH)
	case "prop":
		return fmt.Sprintf("%s%s Single hero prop object on neutral surface, studio product shot, sharp focus, no people. Object immutable traits: %s. Style: %s.", prefix, styleEN, traits, styleZH)
	default:
		return fmt.Sprintf("%s%s Wide environmental establishing shot, empty of people, architectural or landscape concept art, clear readable space. Location immutable traits: %s. Style: %s.", prefix, styleEN, traits, styleZH)
	}
}

func fragmentTraitsHash(parts ...interface{}) string {
	h := fnv.New32a()
	for _, p := range parts {
		switch v := p.(type) {
		case string:
			_, _ = h.Write([]byte(v))
		case []string:
			_, _ = h.Write([]byte(strings.Join(v, "|")))
		}
		_, _ = h.Write([]byte{0})
	}
	return fmt.Sprintf("%08x", h.Sum32())
}

func fragmentReferenceAssetMap(assets []domain.FragmentReferenceAsset) map[string]string {
	out := make(map[string]string)
	for _, a := range assets {
		if k := strings.TrimSpace(a.Key); k != "" && strings.TrimSpace(a.ImageURL) != "" {
			out[k] = strings.TrimSpace(a.ImageURL)
		}
	}
	return out
}

func mergeFragmentSceneReferenceAssets(userURLs []string, scene domain.FragmentScenePlan, assets []domain.FragmentReferenceAsset, maxTotal int) []string {
	return mergeFragmentSceneReferenceImages(userURLs, fragmentExpandedScene{ReferenceKeys: scene.ReferenceKeys}, fragmentReferenceAssetMap(assets), maxTotal)
}

func (s *FragmentGenerationService) generateAnchorImages(ctx context.Context, userID, genTaskID string, req domain.FragmentGenerationRequest, bible *domain.FragmentVisualBible, aspectRatio string) (map[string]string, []domain.FragmentAnchorImage, int) {
	outMap := make(map[string]string)
	if bible == nil {
		return outMap, nil, 0
	}
	ar := domain.NormalizeFragmentAspectRatio(aspectRatio)
	if ar == "" {
		ar = domain.FragmentAspectDefault
	}
	styleEN := ""
	if bible.StyleBible != nil {
		styleEN = strings.TrimSpace(bible.StyleBible.ArtStyle)
	}
	styleZH := fragmentStyleDesc(req.Style)
	moodZH := fragmentMoodDesc(req.Mood)
	userRef := fragmentPrefillHTTPImageURLs(req.ImageUrls, 1)

	totalTok := 0
	var records []domain.FragmentAnchorImage
	firstChar := true

	for i, ch := range bible.Characters {
		if i >= fragmentMaxAnchorCharacters {
			break
		}
		key := strings.TrimSpace(ch.Key)
		if key == "" {
			continue
		}
		traits := strings.Join(ch.ImmutableTraits, "; ")
		prompt := fmt.Sprintf(
			"%s Character design reference, single person, neutral standing pose facing camera, full body visible, clean simple studio background, high detail concept art. Immutable traits: %s. Overall mood (reference): %s. Style tag: %s.",
			styleEN, traits, moodZH, styleZH)
		if n := strings.TrimSpace(ch.Name); n != "" {
			prompt = n + ". " + prompt
		}
		var refs []string
		if firstChar && len(userRef) > 0 {
			refs = append(refs, userRef...)
			firstChar = false
		}
		url, tok, err := s.generateOneFragmentImage(ctx, userID, genTaskID, prompt, ar, refs)
		if err != nil {
			s.logger.Warn("anchor character image failed", zap.String("key", key), zap.Error(err))
			continue
		}
		outMap[key] = url
		records = append(records, domain.FragmentAnchorImage{Key: key, Kind: "character", ImageURL: url})
		totalTok += tok
	}

	for i, p := range bible.Props {
		if i >= fragmentMaxAnchorProps {
			break
		}
		key := strings.TrimSpace(p.Key)
		if key == "" {
			continue
		}
		traits := strings.Join(p.ImmutableTraits, "; ")
		prompt := fmt.Sprintf(
			"%s Single hero prop object on neutral surface, studio product shot, sharp focus, no people. Object traits: %s. Style: %s.",
			styleEN, traits, styleZH)
		url, tok, err := s.generateOneFragmentImage(ctx, userID, genTaskID, prompt, ar, nil)
		if err != nil {
			s.logger.Warn("anchor prop image failed", zap.String("key", key), zap.Error(err))
			continue
		}
		outMap[key] = url
		records = append(records, domain.FragmentAnchorImage{Key: key, Kind: "prop", ImageURL: url})
		totalTok += tok
	}

	for i, loc := range bible.Locations {
		if i >= fragmentMaxAnchorLocations {
			break
		}
		key := strings.TrimSpace(loc.Key)
		if key == "" {
			continue
		}
		traits := strings.Join(loc.ImmutableTraits, "; ")
		prompt := fmt.Sprintf(
			"%s Wide environmental establishing shot, empty of people, architectural or landscape concept art, clear readable space. Location: %s. Style: %s.",
			styleEN, traits, styleZH)
		url, tok, err := s.generateOneFragmentImage(ctx, userID, genTaskID, prompt, ar, nil)
		if err != nil {
			s.logger.Warn("anchor location image failed", zap.String("key", key), zap.Error(err))
			continue
		}
		outMap[key] = url
		records = append(records, domain.FragmentAnchorImage{Key: key, Kind: "location", ImageURL: url})
		totalTok += tok
	}

	return outMap, records, totalTok
}

func (s *FragmentGenerationService) generateOneFragmentImage(ctx context.Context, userID, genTaskID, prompt, aspectRatio string, refURLs []string) (string, int, error) {
	return s.generateOneFragmentImageWithOptions(ctx, userID, genTaskID, prompt, aspectRatio, refURLs, 0, nil, 0)
}

func (s *FragmentGenerationService) generateOneFragmentImageWithOptions(ctx context.Context, userID, genTaskID, prompt, aspectRatio string, refURLs []string, seed int, options map[string]interface{}, guidanceScale float64) (string, int, error) {
	payload := map[string]interface{}{
		"prompt":      prompt,
		"aspectRatio": aspectRatio,
	}
	if len(refURLs) > 0 {
		payload["referenceImages"] = refURLs
	}
	if seed > 0 {
		payload["seed"] = seed
	}
	if len(options) > 0 {
		payload["options"] = options
	}
	if guidanceScale > 0 {
		payload["guidanceScale"] = guidanceScale
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
		RelatedEntityID:   genTaskID,
		RelatedEntityType: "fragment_generation",
	}
	return s.aiService.GenerateImageForFragment(ctx, &aiReq)
}

func (s *FragmentGenerationService) checkFragmentConsistency(ctx context.Context, taskID string, bible *domain.FragmentVisualBible, evidence []domain.FragmentVisualEvidence, scenes []domain.FragmentScenePlan, referenceAssets []domain.FragmentReferenceAsset, imageURLs []string, policy *domain.FragmentConsistencyPolicy) ([]domain.FragmentConsistencyIssue, int, string, int, string) {
	auditURLs, skipped := selectFragmentAuditImageURLs(imageURLs, scenes, policy)
	if len(auditURLs) == 0 {
		return nil, 0, "", 0, skipped
	}
	vbBytes, _ := json.Marshal(bible)
	evidenceBytes, _ := json.Marshal(evidence)
	scenesBytes, _ := json.Marshal(scenes)
	assetBytes, _ := json.Marshal(referenceAssets)
	var urlLines strings.Builder
	for i, u := range auditURLs {
		fmt.Fprintf(&urlLines, "%d: %s\n", i, u)
	}
	prompt := fmt.Sprintf(`你是视觉一致性审计员。请直接查看最终配图，基于「视觉圣经」、入口视觉事实、各格场景描述与参考资产，列出可能的不一致（角色外观漂移、关键道具缺失、环境与圣经矛盾等）。无问题时 issues 为空数组。

视觉圣经 JSON：
%s

入口视觉事实 JSON：
%s

场景 JSON（含 sceneDesc、referenceKeys、index）：
%s

参考资产 JSON（按需生成，可能为空）：
%s

最终图片 URL（按生成顺序排列；尽量与场景 index 对应）：
%s

只输出 JSON：{"issues":[{"sceneIndex":0,"entityKey":"可为空","imageUrl":"问题图片URL","severity":"low|medium|high","expected":"应保持的视觉事实","observed":"实际观察","confidence":0.0,"detail":"中文简述"}]}`,
		string(vbBytes), string(evidenceBytes), string(scenesBytes), string(assetBytes), urlLines.String())

	resp, err := s.aiService.GenerateFragmentVisionJSON(ctx, &FragmentVisionJSONRequest{
		Prompt:            prompt,
		ImageURLs:         auditURLs,
		MaxTokens:         2048,
		Temperature:       0.25,
		RelatedEntityType: "fragment_generation",
		RelatedEntityID:   taskID,
		Step:              "fragment_consistency_audit",
	})
	if err != nil {
		s.logger.Warn("fragment consistency check request failed", zap.Error(err))
		return nil, 0, "", len(auditURLs), err.Error()
	}
	issues, perr := parseFragmentConsistencyIssues(resp.Raw)
	if perr != nil {
		s.logger.Warn("fragment consistency check parse failed", zap.Error(perr), zap.String("snippet", truncateRunes(resp.Raw, 200)))
		return nil, resp.TokensUsed, resp.Provider, len(auditURLs), "parse_failed"
	}
	return issues, resp.TokensUsed, resp.Provider, len(auditURLs), skipped
}

func parseFragmentConsistencyIssues(raw string) ([]domain.FragmentConsistencyIssue, error) {
	s := strings.TrimSpace(raw)
	if m := jsonFenceRE.FindStringSubmatch(s); len(m) > 1 {
		s = strings.TrimSpace(m[1])
	}
	var env struct {
		Issues []domain.FragmentConsistencyIssue `json:"issues"`
	}
	if err := json.Unmarshal([]byte(s), &env); err != nil {
		return nil, err
	}
	return env.Issues, nil
}

func selectFragmentAuditImageURLs(imageURLs []string, scenes []domain.FragmentScenePlan, policy *domain.FragmentConsistencyPolicy) ([]string, string) {
	if len(imageURLs) == 0 {
		return nil, "no_images"
	}
	if policy != nil && policy.Level == "off" {
		return nil, "consistency_off"
	}
	if policy != nil && policy.Level == "strong" {
		return append([]string(nil), imageURLs...), ""
	}
	seen := map[int]struct{}{}
	var idxs []int
	add := func(i int) {
		if i < 0 || i >= len(imageURLs) {
			return
		}
		if _, ok := seen[i]; ok {
			return
		}
		seen[i] = struct{}{}
		idxs = append(idxs, i)
	}
	add(0)
	add(len(imageURLs) - 1)
	for i, scene := range scenes {
		if len(idxs) >= 3 {
			break
		}
		if len(scene.ReferenceKeys) > 1 || len(scene.EntityBindings) > 1 {
			add(i)
		}
	}
	out := make([]string, 0, len(idxs))
	for _, i := range idxs {
		out = append(out, imageURLs[i])
	}
	return out, ""
}

// ────────────────────── Step 2: 场景扩展 ──────────────────────

// fragmentSceneExpansionResult 场景扩展输出。
type fragmentSceneExpansionResult struct {
	Scenes     []fragmentExpandedScene `json:"scenes"`
	TokensUsed int
}

// fragmentExpandedScene 扩展出的单个场景，含中文描述、实体绑定和英文图片提示词。
type fragmentExpandedScene struct {
	Index          int                            `json:"index"`
	SceneDesc      string                         `json:"sceneDesc"`               // 中文场景描述（面向读者）
	ImagePrompt    string                         `json:"imagePrompt"`             // 英文图片生成提示词（面向图片模型）
	ReferenceKeys  []string                       `json:"referenceKeys,omitempty"` // 引用 visualBible / 参考资产的 key
	EntityBindings []domain.FragmentEntityBinding `json:"entityBindings,omitempty"`
	ComicTexts     []domain.FragmentComicText     `json:"comicTexts,omitempty"`
}

func (s *FragmentGenerationService) expandScenes(ctx context.Context, userID, taskID string, req domain.FragmentGenerationRequest, elemResult *fragmentElementExtractionResult, sceneCount int, aspectRatio string) (*fragmentSceneExpansionResult, error) {
	elementsJSON, _ := json.Marshal(elemResult.Elements)
	vbJSON := "{}"
	if elemResult.VisualBible != nil {
		if b, err := json.Marshal(elemResult.VisualBible); err == nil {
			vbJSON = string(b)
		}
	}
	keyHint := formatVisualBibleKeyListForPrompt(elemResult.VisualBible)

	var narrativeHint string
	switch {
	case sceneCount == 1:
		narrativeHint = "1 格：这一帧必须是整条故事中最有视觉冲击力的瞬间。不要选平淡的叙述时刻——选悬念最浓的那一秒、反转刚发生的那一帧、或者一个让人立刻想问\"之前到底发生了什么\"的定格。想象电影海报：一个画面就让观众脑补出一整部电影。让这一帧的构图、光影、角色表情本身就在讲故事。如果故事有高潮，这就是高潮被凝固的那一毫秒；如果故事没有高潮，这就是最让人不安的那一秒——一切看似正常但有什么东西不太对。"
	case sceneCount == 2:
		narrativeHint = "2 格：核心技法是\"认知落差\"——第一格建立预期，第二格打破它。具体可以用的落差类型：视角落差（第一格是人眼中的世界，第二格是虫子眼中的同一场景——完全不同的尺度，完全不同的恐惧）、情绪落差（温馨 → 惊悚，平静 → 狂喜，搞笑 → 心碎）、尺度落差（微观 → 宏观，一只蚂蚁的特写 → 整个城市的俯瞰）、时间落差（现在 → 十年前，白天 → 深夜，春天 → 冬天）。观众看完两格后的反应应该是\"等等，怎么会这样？\"或\"所以第一格里那个其实是……\"。第二格的第一反应应该是意外，第二反应才是理解。"
	case sceneCount == 3:
		narrativeHint = "3 格：不要三幕式！三幕式会产出四平八稳但无趣的内容。自由节奏才是王道——第一格抛出引子（一个画面就让人好奇\"这是哪里/这个人是谁/发生了什么\"），第二格可以突然转向（换视角、换时空、换叙事者、或者一个完全出乎意料的元素闯入），第三格收束但不给答案——留一个让人回味的尾巴。可以尝试的高级结构：假结局（第三格看似结束但其实暗示了更大的故事）、环形结构（第三格呼应第一格但信息量不同，读者对比两格才发现真相）、打破第四面墙（角色意识到了读者的存在）、时间嵌套（第三格揭示第一格其实是回忆中的回忆）。惊喜感比工整重要十倍。"
	default:
		narrativeHint = fmt.Sprintf("%d 格：一条故事线但绝不要线性平铺直叙。在格与格之间可以大胆穿插——视角突变（从人类视角突然切到猫的视角、从地面仰视突然变卫星俯瞰、从一个角色的POV切到另一个角色的POV看着同一个场景）、时空跳跃或闪回（这一格还是现在，下一格突然是十年前，再下一格又回到现在但时间已过）、打破第四面墙（角色看向\"镜头\"外、文字溢出画框、角色似乎在跟读者说话）、超现实梦境（前一秒还在教室，下一秒教室漂浮在星空中、水面变成了天空、书本里的文字活了过来）、意料之外的新元素闯入（一个路过的蝴蝶改变了整条故事线、一只手从画外伸进来拿走了桌上的杯子）。开头建立世界观和角色，中间尽情放飞制造\"没想到\"的惊喜，结尾可以呼应开头、可以留悬念、可以推翻之前所有设定——只要让观众看完觉得\"这趟旅程值了\"。中间某些格可以是氛围画面——不推进剧情但传递情绪（空椅子、飘落的羽毛、雨中的路灯）——这种\"停顿\"本身是节奏的一部分。", sceneCount)
	}

	legacyPrompt := fmt.Sprintf(`你是一位脑洞大开、同时精通电影摄影和漫画分镜的视觉故事创作者，兼具导演的叙事把控力、摄影师的画面敏感度和概念艺术家的想象力。你的任务是把一段故事文字拆解为 %d 个画面格——每一格都是一幅能独立吸引目光、按顺序浏览又能串联成完整故事的视觉作品。你不是在"配图"，你是在"用画面重新讲故事"。

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
叙事节奏指引
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

%s

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
输入素材
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

故事元素（从用户输入中提取的结构化元素）：
%s

视觉圣经 visualBible（JSON；用于每格 referenceKeys；若无则为 {}）：
%s

referenceKeys 可用的稳定 key 列表（必须从下列 key 中选择子集填入每格；禁止自造 key；若列表说明为无 key 则每格 referenceKeys=[]）：
%s

故事正文（AI 生成的碎片故事文案）：
%s

风格：%s
情绪：%s
画面长宽比：%s

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
创作方法论（你的导演工具箱）
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

%s

【一、世界观一致性——这是底线，不是上限】
- 角色的外貌（发型、服装、体型、标志性特征）必须在每一格中完全一致——除非剧情需要角色发生变化（被雨淋湿、受伤、换装等需要明确交代）
- 空间设定（室内布局、城市天际线、自然地貌）保持可辨识的连续性——读者应该感觉这些格发生在同一个世界里
- artStyle（艺术风格）全程统一——如果第一格是"水彩插画风"，最后一格也必须是水彩插画风。风格是故事的视觉签名
- 色彩基调可以随情绪渐变，但不要突然跳到完全不同的色彩体系

【一补、漫画版式与文字层——让画面像真实漫画】
- 如果风格、故事或用户输入适合漫画表达，imagePrompt 必须描述 manga/comic panel layout：清晰格框、粗墨线边框、gutter 间距、从左到右/从上到下的阅读顺序。
- 每一格可以包含漫画文字元素，但必须分类：旁白 narration 放在 caption box；角色台词 dialogue 放在 speech bubble，气泡尾巴指向 speaker；内心独白 thought 放在 thought bubble；语气词/拟声词 sfx 用夸张音效字。
- 文字数量必须严格受限：每格最多 1 个 narration、1-2 个 dialogue、最多 1 个 sfx；thought 仅在必要时出现且最多 1 个。单条中文建议不超过 12 个汉字，不要把整段 sceneDesc 塞进气泡。
- imagePrompt 中应给气泡/旁白框预留干净空间，避免遮挡人物脸部和关键道具。
- 漫画文字必须由图片模型直接画进最终图片中，不能依赖 App 后续叠加；imagePrompt 要明确要求 render the exact Chinese text inside the image，字体清晰、可读、与漫画风格一致；禁止额外随机文字或英文假字。

【二、叙事自由度——打破常规才是你的常规】
- 允许并且鼓励：
  · 跳切：时间突然推进，中间的过程留白让读者脑补
  · 闪回：突然回到过去，揭示之前没展示的信息
  · 视角反转：突然换成配角、动物、甚至无生命物体的视角
  · 超现实片段：梦境/幻觉/想象——可以让光影与空间反常，但仍保持同一主风格体系
  · 意外闯入：可以有新角色/物体突然出现，但必须与已有元素或故事正文存在因果关联
  · 静帧与留白：一格完全静止的画面（空房间、雨中的长椅、桌面上的物品）——不推进剧情但传递情绪，这种"停顿"本身就是节奏
- 每一格都可以是不同类型的画面：
  · 写实叙事场景（角色在行动）
  · 氛围/意境画面（雨滴落在水洼、风铃在摇晃、夕阳照进空教室）
  · 特写细节（一只手、一封信、地上的影子）
  · 全景鸟瞰（城市天际线、从太空看地球）
  · 角色内心可视化（恐惧具象化成黑色的手、回忆用虚线勾勒、想象用不同的色调区分）
  · 象征/隐喻画面（断裂的桥、倒走的钟、镜中不同的自己）
- 不必每一格都在推进剧情——有时一格氛围画面比剧情画面更有力量。读者会在安静的画面中感受到前面积累的情绪

【三、构图多样性——你的镜头语言词库】
每格必须使用不同的镜头语言，以下是你可用的完整构图类型库（混搭使用，拒绝连续两格相同组合）：

  景别工具箱：
  - 极特写：一只眼睛、嘴唇、手指尖、物品的某个局部——传递极度的亲密或紧张
  - 特写：面部表情、手部动作、物品全貌——情绪和细节的放大镜
  - 近景：胸部以上，能看到表情和上半身动作——对话和情感交流的标配
  - 中景：膝盖以上，能看到角色全身姿态和环境的一小部分——叙事的主力景别
  - 全景：全身 + 周围环境，角色与环境的关系清晰——交代场景和空间关系
  - 远景：人物在画面中很小，环境占据主导——表达孤独、渺小、天地之大
  - 极远景/大远景：地标级画面，几乎看不到人物——故事的"呼吸"，给读者喘息的空间

  角度工具箱：
  - 平视：最接近人眼的日常视角，真实感最强
  - 俯拍/高角度：上帝视角，角色显得渺小无助，或展示场景的空间布局
  - 仰拍/低角度：角色显得高大/威严/压迫，或者模拟儿童/动物视角
  - Dutch angle（倾斜角度）：画面歪斜，制造不安、失衡、精神异常的感觉
  - 鸟瞰：正上方往下拍，展示平面的图案和布局（桌面、街道、棋盘）
  - 虫眼：贴地往上拍，夸大物体的高度和压迫感

  非人类视角工具箱（这是制造惊喜的秘密武器）：
  - 猫/狗视角：低角度，人类的腿变成了柱子，世界变得巨大
  - 鸟的视角：高空俯瞰，一切变得像模型，人变成了小点
  - 鱼的视角：从水下往上看，水面是扭曲的亮面，岸上的世界是摇晃的
  - 物品视角：钥匙孔视角（窥视感）、镜中倒影（虚实对照）、手机屏幕视角（被观看的感觉）、时钟视角（往下看人）、书本视角（被翻开的感觉）
  - 完全抽象：用色块和线条表达情绪而非具象画面——适合内心独白或超现实段落

  硬性规则：相邻两格不能使用相同的景别+角度组合。这是最低要求——理想状态是每 3 格内不重复

【四、光影与色彩——情绪的精密调色板】
  光影工具（选择与场景情绪匹配的光影方案）：
  - 逆光/轮廓光：主体变成剪影或边缘发光 → 神秘、英雄感、告别、未知
  - 侧光：一半亮一半暗，强烈的明暗对比 → 戏剧冲突、内心矛盾、揭示秘密
  - 顶光：从正上方打下来，眼窝和鼻子下方投下浓重阴影 → 压抑、审判、精神压力
  - 底光：从下方照亮面部（鬼故事经典打光） → 恐怖、诡异、非自然
  - 散射光/柔光：没有明确方向，阴影柔和 → 日常、平静、回忆、安全
  - 剪影：完全背光，只剩轮廓 → 未知、威胁、悬念、分离
  - 斑驳光：透过树叶/窗户/格栅的碎光 → 怀旧、监狱/困住、梦境

  色温与情绪的映射：
  - 暖黄/橙色（日落、烛火） → 安全、怀旧、温馨、即将结束
  - 冷蓝/青色（月光、荧光灯） → 疏离、科技、忧郁、冷静
  - 中性白（正午日光） → 真实、日常、客观
  - 红色（灯光、血色、警报） → 危险、激情、愤怒、警告
  - 绿色（自然、毒气、监控） → 自然、生长、毒性、被监视

  色彩饱和度的情绪曲线：
  - 高饱和 → 活力、梦境、奇幻、童年回忆
  - 中饱和 → 现实、日常、叙事进行时
  - 低饱和/去饱和 → 压抑、回忆褪色、末日、疲惫

  格间光影变化规则：
  - 光源方向和强度在格间可以变化，但要服务于情绪走向
  - 例如：故事走向紧张 → 光线从明亮逐步变暗，阴影逐渐拉长
  - 例如：故事走向释然 → 从冷色调逐渐回暖，阴影变柔和
  - 允许一格突然跳到完全不同的光影方案（如闪回用怀旧柔光，回到现实用冷硬侧光）

【五、sceneDesc 写法——面向读者的画面脚本】
- 一到两句话，让读者在脑中"看到"这一格的画面
- 要传达四个信息：谁在画面里、在做什么、什么氛围、在故事的哪个节点
- 不要写成剧情概要，要写成"如果你闭上眼睛想象这一格，你会看到什么"
- 好的示例："她站在空荡荡的站台，身后是已经驶远的列车尾灯，手里还攥着没来得及递出去的信"
- 差的示例："第二幕，她错过了火车"（这是剧情概要，不是画面描述）
- 好的示例："特写：一只手慢慢松开，信封从指缝间滑落，背面写着'已过期'三个字"
- 差的示例："信掉了"（太简略，没有画面感）

【六、imagePrompt 写法——给图片模型下达的精确视觉指令】
- 这是在给一位顶级概念艺术家下 brief，必须具体到每个视觉决策
- 必须按以下 8 层结构依次描述，每层用句号分隔，形成一段完整的英文视觉描述：

  第 (1) 层 artStyle —— 整体艺术风格
  不要用笼统的"anime"或"illustration"。要写出具体的混合风格，让画师一眼知道用什么技法。
  好的示例："cinematic watercolor with digital color grading, muted palette with selective vivid accents"
  好的示例："charcoal sketch style with selective watercolor highlights, rough texture, visible stroke marks"
  好的示例："hyperrealistic 3D render with soft bloom lighting, slight lens aberration"
  好的示例："retro cel-shaded animation style, flat colors with bold ink outlines, 90s anime aesthetic"
  差的示例："anime style" / "illustration"

  第 (2) 层 subject —— 画面核心主体
  谁或什么在画面中，具体到可以被直接画出来。包括：角色完整外貌（发型发色、服装款式颜色材质、体型年龄）、当前姿态（站/坐/跑/蹲/转身）、表情或情绪暗示、手中持有的物品。
  好的示例："a tall slender woman in her mid-20s with shoulder-length dyed blue hair tips, wearing an oversized black hoodie with the hood down, standing still at the center of a pedestrian crossing, holding a paper coffee cup with both hands, her gaze directed slightly downward, expression neutral but tired"
  差的示例："a woman standing"（缺少所有视觉细节）

  第 (3) 层 environment —— 背景和场景空间
  角色所处的完整空间，包括空间类型、建筑/自然特征、前景中景背景的层次。
  好的示例："an abandoned indoor amusement park at night, a faded carousel with peeling paint horses slowly rotating in the midground, only red and blue neon tubes still flickering, dusty concrete floor scattered with old admission tickets and collapsed cotton candy cones, a collapsed banner reading GRAND OPENING visible in the far background"
  差的示例："an amusement park"（没有细节、没有层次）

  第 (4) 层 composition —— 镜头构图
  景别 + 角度 + 画面重心位置 + 引导线（如有）。
  好的示例："medium shot from a low angle, subject positioned at the right third of the frame, the carousel filling the left two-thirds of the background, leading lines from the floor tiles converging toward the subject"
  差的示例："medium shot"（缺少角度和构图位置）

  第 (5) 层 lighting —— 光照方案
  光源方向、类型、色温、阴影特征、高光位置。要具体到画师能据此画出光影。
  好的示例："primary light from the red neon sign on the left casting warm crimson shadows across the subject's face, secondary cool blue light from a flickering neon tube behind creating a rim light on the subject's hair, no ambient natural light, deep shadows in the background with no visible detail"
  差的示例："neon lights"（方向？色温？阴影？）

  第 (6) 层 colorPalette —— 完整色彩方案
  主色调 + 点缀色 + 对比度 + 色彩分布。
  好的示例："dominant dark teal and navy blue in the shadows, accent warm red from the neon sign creating high-contrast focal points, muted purple undertones in the midtones, skin tones slightly desaturated, overall low-key color scheme"
  差的示例："dark colors with red"（太笼统）

  第 (7) 层 mood —— 情绪氛围
  这一格应该让观众感受到什么。用复合情绪词，不要只写一个形容词。
  好的示例："melancholic solitude with a hint of nostalgia, quiet and still, the feeling of being the last person in a place that used to be full of laughter"
  差的示例："sad"（太简单）

  第 (8) 层 extra details —— 额外的画面丰富层
  让画面从"正确"升级到"有灵魂"的细节：微粒、反光、景深、材质质感、天气效果。
  好的示例："dust particles drifting through the beams of neon light, faint blurry reflection of the carousel lights on the glossy floor, shallow depth of field with bokeh balls on the background neon signs, a single old ticket caught mid-air as if just dropped"
  差的示例："some dust"（太简单）

【七、entityBindings 写法——多人物/多物品一致性的结构化约束】
- 每格必须列出 referenceKeys 中实际出现的实体绑定。不要只写 key，要写清楚该实体在本格的角色、画面位置、动作、道具归属。
- 多人物同框时必须区分位置与外观归属，例如 char_a 在左侧拿 prop_key，char_b 在右侧，不要交换服装/发型/道具。
- 物品必须写 ownerKey；地点写空间作用（background / main location / memory location）。
- consistencyNote 要明确“不能和谁混淆、哪些特征必须保持”。

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
输出格式
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

请只输出一个 JSON 对象（不要 markdown 代码围栏、不要其他说明文字、不要注释）：
{
  "scenes": [
    {
      "index": 0,
      "sceneDesc": "中文：这格画面的内容描述，1-2句，让读者脑中浮现画面",
      "imagePrompt": "English: (1) artStyle. (2) subject. (3) environment. (4) composition. (5) lighting. (6) colorPalette. (7) mood. (8) extra details. At least 70 words total. Be specific and concrete — every word should help the image model make a visual decision. Do not use vague terms.",
      "referenceKeys": ["char_main", "prop_laptop", "loc_office"],
      "entityBindings": [
        { "key": "char_main", "kind": "character", "role": "main subject", "position": "left foreground", "action": "holding prop_laptop", "ownerKey": "", "consistencyNote": "keep all immutableTraits; do not swap clothing or facial traits with other characters" },
        { "key": "prop_laptop", "kind": "prop", "role": "story clue", "position": "in char_main hands", "action": "screen glowing", "ownerKey": "char_main", "consistencyNote": "belongs only to char_main" }
      ],
      "comicTexts": [
        { "type": "narration", "text": "三分钟前，一切还很安静。", "position": "top-left" },
        { "type": "dialogue", "text": "你听见了吗？", "speaker": "char_main", "position": "speech-bubble" },
        { "type": "sfx", "text": "砰！", "position": "mid-frame" }
      ]
    }
  ]
}

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
硬性规则（不可违反）
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

- scenes 数组恰好 %d 项，index 从 0 到 %d
- imagePrompt 必须是英文，sceneDesc 必须是中文
- 每个 imagePrompt 至少 70 个英文单词，必须覆盖上述全部 8 个视觉层
- 所有格的 artStyle 必须以相同的风格描述开头，确保视觉风格统一
- 角色外貌（发型、服装颜色款式、体型、标志性特征）在各格之间完全一致
- 相邻两格的 composition 必须有明显差异（不同景别或不同角度，不能连续两格正面中景）
- 每一格必须引用至少 2 个可追溯锚点（来自 elements 或故事正文的具体角色/物品/场景细节），禁止无依据“硬反转”
- 每一格必须包含 referenceKeys：1–5 个字符串，且必须是上方「稳定 key 列表」中的 key；若列表为空则 referenceKeys 为 []
- 每一格必须包含 entityBindings；其 key 必须来自本格 referenceKeys，不允许自造实体
- comicTexts 可为空数组；若有 dialogue/thought，speaker 必须来自本格 referenceKeys 中的角色 key；每格最多 1 narration、1-2 dialogue、最多 1 sfx、最多 1 thought，单条中文建议 <=12 汉字
- imagePrompt 必须把 comicTexts 对应的旁白框、对话气泡、思想气泡、拟声词版式写成英文视觉指令；不要只在 JSON 里列文字而不影响画面
- imagePrompt 中绝对不要出现"copy the reference image"或"exactly like the reference"——每格都应该是原创的视觉创作
- sceneDesc 中不要出现格式化标记（不要 #、不要 **、不要列表符号）
- 不要在 JSON 之外输出任何文字（包括开头和结尾的说明）`,
		sceneCount,
		narrativeHint,
		string(elementsJSON),
		vbJSON,
		keyHint,
		elemResult.Content,
		req.Style,
		req.Mood,
		aspectRatio,
		structuredMangaLanguageGuidance(),
		sceneCount,
		sceneCount-1)

	prompt := renderPromptDSL(PromptDSL{
		Role:         "你是一位视觉叙事导演，负责将故事正文转为结构化分镜 JSON。",
		Task:         "输出 scenes[]，每格包含 sceneDesc/imagePrompt/referenceKeys/entityBindings/comicTexts。",
		Inputs:       map[string]any{"sceneCount": sceneCount, "aspectRatio": aspectRatio, "style": req.Style, "mood": req.Mood, "narrativeHint": narrativeHint, "storyElements": json.RawMessage(elementsJSON), "visualBible": json.RawMessage(vbJSON), "referenceKeysHint": keyHint, "storyContent": elemResult.Content},
		GlobalConfig: structuredMangaLanguageGuidance(),
		Sections: []PromptDSLSection{
			{Title: "Detailed Instructions", Kind: "text", Body: legacyPrompt},
		},
	})

	payloadBytes, _ := json.Marshal(map[string]interface{}{
		"prompt": prompt,
		"step":   "fragment_scene_expansion",
	})
	aiReq := domain.AITask{
		ID:                uuid.New().String(),
		UserID:            userID,
		Type:              domain.AITaskGenerateFragmentContent,
		Status:            domain.AITaskStatusProcessing,
		Provider:          "",
		Input:             string(payloadBytes),
		RelatedEntityID:   taskID,
		RelatedEntityType: "fragment_generation",
	}

	raw, tokensUsed, _, err := s.aiService.GenerateTextForFragment(ctx, &aiReq)
	if err != nil {
		return nil, fmt.Errorf("AI scene expansion failed: %w", err)
	}

	scenes, perr := parseSceneExpansion(raw, sceneCount)
	if perr != nil {
		s.logger.Warn("scene expansion JSON parse failed, generating single fallback scene",
			zap.Error(perr))
		// 回退：生成一个默认场景
		fallback := fmt.Sprintf("%s, %s style, %s mood, aspect ratio %s",
			truncateRunes(elemResult.Content, 100), req.Style, req.Mood, aspectRatio)
		scenes = []fragmentExpandedScene{
			{Index: 0, SceneDesc: truncateRunes(elemResult.Content, 50), ImagePrompt: fallback},
		}
	}

	return &fragmentSceneExpansionResult{
		Scenes:     scenes,
		TokensUsed: tokensUsed,
	}, nil
}

// parseSceneExpansion 解析场景扩展 JSON 输出。
func parseSceneExpansion(raw string, want int) ([]fragmentExpandedScene, error) {
	s := strings.TrimSpace(raw)
	if m := jsonFenceRE.FindStringSubmatch(s); len(m) > 1 {
		s = strings.TrimSpace(m[1])
	}
	var env struct {
		Scenes []fragmentExpandedScene `json:"scenes"`
	}
	if err := json.Unmarshal([]byte(s), &env); err != nil {
		return nil, fmt.Errorf("parse scene expansion JSON: %w", err)
	}
	if len(env.Scenes) == 0 {
		return nil, fmt.Errorf("scene expansion returned 0 scenes")
	}
	// 校验每个场景都有 imagePrompt
	for i, sc := range env.Scenes {
		if strings.TrimSpace(sc.ImagePrompt) == "" {
			return nil, fmt.Errorf("scene %d has empty imagePrompt", i)
		}
		sc.Index = i
		env.Scenes[i] = sc
	}
	return env.Scenes, nil
}

// ────────────────────── Step 4: 逐场景生成图片（多参考图） ──────────────────────

func cloneFragmentProviderOptions(policy *domain.FragmentConsistencyPolicy) map[string]interface{} {
	if policy == nil || len(policy.ProviderOptions) == 0 {
		return nil
	}
	out := make(map[string]interface{}, len(policy.ProviderOptions))
	for k, v := range policy.ProviderOptions {
		out[k] = v
	}
	return out
}

func buildFragmentSceneImagePrompt(bible *domain.FragmentVisualBible, scene domain.FragmentScenePlan) string {
	var b strings.Builder
	if bible != nil && bible.StyleBible != nil && strings.TrimSpace(bible.StyleBible.ArtStyle) != "" {
		fmt.Fprintf(&b, "Global art style: %s.\n", strings.TrimSpace(bible.StyleBible.ArtStyle))
	}
	fmt.Fprintf(&b, "Scene: %s.\n", strings.TrimSpace(scene.ImagePrompt))
	writeFragmentActiveEntities(&b, bible, scene.ReferenceKeys)
	writeFragmentComicLayoutDirective(&b, bible, scene)
	if len(scene.EntityBindings) > 0 {
		b.WriteString("Entity binding rules:\n")
		for _, bind := range scene.EntityBindings {
			fmt.Fprintf(&b, "- %s (%s): role=%s; position=%s; action=%s; owner=%s; consistency=%s.\n",
				bind.Key, bind.Kind, bind.Role, bind.Position, bind.Action, bind.OwnerKey, bind.ConsistencyNote)
		}
	}
	if len(scene.ReferenceKeys) > 1 {
		b.WriteString("Do not merge or swap identities, clothing, props, positions, or ownership between the listed entities.\n")
	}
	b.WriteString("Keep immutable traits exactly consistent across the whole image series while still creating an original scene.\n")
	return strings.TrimSpace(b.String())
}

func writeFragmentActiveEntities(b *strings.Builder, bible *domain.FragmentVisualBible, keys []string) {
	if bible == nil || len(keys) == 0 {
		return
	}
	keySet := map[string]struct{}{}
	for _, k := range keys {
		keySet[strings.TrimSpace(k)] = struct{}{}
	}
	b.WriteString("Active visual bible entities:\n")
	for _, ch := range bible.Characters {
		if _, ok := keySet[strings.TrimSpace(ch.Key)]; !ok {
			continue
		}
		fmt.Fprintf(b, "- %s character %s immutable traits: %s. Avoid: %s.\n",
			ch.Key, ch.Name, strings.Join(ch.ImmutableTraits, "; "), strings.Join(ch.NegativeTraits, "; "))
	}
	for _, p := range bible.Props {
		if _, ok := keySet[strings.TrimSpace(p.Key)]; !ok {
			continue
		}
		fmt.Fprintf(b, "- %s prop %s immutable traits: %s. Owner: %s. Avoid: %s.\n",
			p.Key, p.Name, strings.Join(p.ImmutableTraits, "; "), p.Ownership, strings.Join(p.NegativeTraits, "; "))
	}
	for _, loc := range bible.Locations {
		if _, ok := keySet[strings.TrimSpace(loc.Key)]; !ok {
			continue
		}
		fmt.Fprintf(b, "- %s location %s immutable traits: %s. Avoid: %s.\n",
			loc.Key, loc.Name, strings.Join(loc.ImmutableTraits, "; "), strings.Join(loc.NegativeTraits, "; "))
	}
}

func writeFragmentComicLayoutDirective(b *strings.Builder, bible *domain.FragmentVisualBible, scene domain.FragmentScenePlan) {
	if b == nil {
		return
	}
	if !fragmentSceneWantsComicLayout(bible, scene) {
		return
	}
	b.WriteString("Comic layout directive: render the image as a manga/comic panel with bold ink panel borders, clear gutters or internal comic-style zones when useful, and a readable left-to-right/top-to-bottom flow. Paint all comic text directly into the final image, not as placeholders and not for app overlay. Render the exact Chinese characters inside bubbles/caption boxes/SFX lettering as clearly as possible with large legible hand-lettered glyphs. Reserve clean negative space for text elements; do not cover faces, hands, or key props with bubbles. Do not add random extra words.\n")
	if len(scene.ComicTexts) == 0 {
		b.WriteString("Include at most 1 narration box, 1-2 dialogue bubbles, and at most 1 SFX lettering. If text appears, it must be drawn directly in-image, each Chinese phrase short (about <=12 characters), legible, and visually readable.\n")
		return
	}
	b.WriteString("Comic text elements:\n")
	for _, item := range normalizeFragmentComicTexts(scene.ComicTexts) {
		text := sanitizeComicPromptText(item.Text)
		speaker := strings.TrimSpace(item.Speaker)
		position := strings.TrimSpace(item.Position)
		if position == "" {
			position = "auto"
		}
		switch item.Type {
		case "narration":
			fmt.Fprintf(b, "- Caption/narration box at %s: rectangular comic caption box, paint the exact Chinese text %q inside the box.\n", position, text)
		case "dialogue":
			if speaker == "" {
				fmt.Fprintf(b, "- Speech bubble at %s: oval white bubble, paint the exact Chinese dialogue %q inside the bubble, tail pointing to the speaking character.\n", position, text)
			} else {
				fmt.Fprintf(b, "- Speech bubble for %s at %s: oval white bubble, paint the exact Chinese dialogue %q inside the bubble, tail pointing to entity key %s.\n", speaker, position, text, speaker)
			}
		case "thought":
			if speaker == "" {
				fmt.Fprintf(b, "- Thought bubble at %s: cloud-shaped bubble, paint the exact Chinese inner monologue %q inside the bubble.\n", position, text)
			} else {
				fmt.Fprintf(b, "- Thought bubble for %s at %s: cloud-shaped bubble, paint the exact Chinese inner monologue %q inside the bubble, linked to entity key %s.\n", speaker, position, text, speaker)
			}
		case "sfx":
			fmt.Fprintf(b, "- Sound effect lettering at %s: paint the exact Chinese SFX text %q as bold stylized comic lettering, integrated into the action without a speech bubble.\n", position, text)
		}
	}
}

func fragmentSceneWantsComicLayout(bible *domain.FragmentVisualBible, scene domain.FragmentScenePlan) bool {
	if len(scene.ComicTexts) > 0 {
		return true
	}
	var probes []string
	probes = append(probes, scene.ImagePrompt, scene.SceneDesc)
	if bible != nil && bible.StyleBible != nil {
		probes = append(probes, bible.StyleBible.ArtStyle)
	}
	for _, p := range probes {
		low := strings.ToLower(strings.TrimSpace(p))
		if strings.Contains(low, "comic") || strings.Contains(low, "manga") || strings.Contains(low, "manhua") || strings.Contains(low, "漫画") {
			return true
		}
	}
	return false
}

func sanitizeComicPromptText(text string) string {
	text = strings.ReplaceAll(strings.TrimSpace(text), "\n", " ")
	text = strings.ReplaceAll(text, "\r", " ")
	return truncateRunes(text, 40)
}

func (s *FragmentGenerationService) generateImagesFromScenes(ctx context.Context, userID, genTaskID, aspectRatio string, bible *domain.FragmentVisualBible, scenes []domain.FragmentScenePlan, referenceAssets []domain.FragmentReferenceAsset, userRefURLs []string, policy *domain.FragmentConsistencyPolicy) (*domain.FragmentImageGenerationResult, error) {
	if len(scenes) == 0 {
		return &domain.FragmentImageGenerationResult{
			ImageUrls:  []string{},
			TokensUsed: 0,
		}, nil
	}

	ar := domain.NormalizeFragmentAspectRatio(aspectRatio)
	if ar == "" {
		ar = domain.FragmentAspectDefault
	}

	imgProv := s.aiService.ResolveFragmentImageProvider(ctx, userID, "")
	if strings.EqualFold(imgProv, "huoshan") {
		n := len(scenes)
		batchPrompt := buildFragmentScenesBatchHuoshanPrompt(bible, scenes, n)
		refBatch := mergeFragmentScenesBatchReferenceImages(userRefURLs, scenes, referenceAssets, fragmentMaxSceneReferenceImages)
		sharedSeed := fragmentStoryImageSeed(policy)
		options := cloneFragmentProviderOptions(policy)
		meta := map[string]interface{}{
			"gen_task_id": genTaskID,
		}
		urls, tok, err := s.aiService.GenerateBatchImagesForFragment(ctx, userID, genTaskID, "fragment_generation", batchPrompt, ar, refBatch, n, options, sharedSeed, 0, meta)
		if err == nil && len(urls) == n {
			for i := range scenes {
				scenes[i].Seed = sharedSeed
				scenes[i].ProviderOptions = options
				scenes[i].FinalImagePrompt = buildFragmentSceneImagePrompt(bible, scenes[i])
				scenes[i].GeneratedImageURL = urls[i]
			}
			return &domain.FragmentImageGenerationResult{
				ImageUrls:  urls,
				TokensUsed: tok,
			}, nil
		}
		if err != nil {
			s.logger.Warn("Huoshan batch narrative images failed, falling back to per-scene",
				zap.Error(err), zap.String("gen_task_id", genTaskID))
		} else {
			s.logger.Warn("Huoshan batch narrative images count mismatch, falling back to per-scene",
				zap.Int("got", len(urls)), zap.Int("need", n), zap.String("gen_task_id", genTaskID))
		}
	}

	var allImageUrls []string
	totalTokens := 0

	for i := range scenes {
		scene := &scenes[i]
		scene.Seed = fragmentStoryImageSeed(policy)
		scene.ProviderOptions = cloneFragmentProviderOptions(policy)
		scene.FinalImagePrompt = buildFragmentSceneImagePrompt(bible, *scene)
		refImgs := mergeFragmentSceneReferenceAssets(userRefURLs, *scene, referenceAssets, fragmentMaxSceneReferenceImages)
		payload := map[string]interface{}{
			"prompt":      scene.FinalImagePrompt,
			"aspectRatio": ar,
			"seed":        scene.Seed,
		}
		if len(scene.ProviderOptions) > 0 {
			payload["options"] = scene.ProviderOptions
		}
		if len(refImgs) > 0 {
			payload["referenceImages"] = refImgs
		}
		imgInput, _ := json.Marshal(payload)

		aiReq := domain.AITask{
			ID:                uuid.New().String(),
			UserID:            userID,
			Type:              domain.AITaskGenerateFragmentImages,
			Status:            domain.AITaskStatusProcessing,
			Provider:          "",
			Input:             string(imgInput),
			RelatedEntityID:   genTaskID,
			RelatedEntityType: "fragment_generation",
		}

		imageURL, tokens, err := s.aiService.GenerateImageForFragment(ctx, &aiReq)
		if err != nil {
			s.logger.Warn("Image generation failed for scene",
				zap.Error(err),
				zap.Int("scene_index", scene.Index))
			continue
		}
		scene.GeneratedImageURL = imageURL
		allImageUrls = append(allImageUrls, imageURL)
		totalTokens += tokens
	}

	return &domain.FragmentImageGenerationResult{
		ImageUrls:  allImageUrls,
		TokensUsed: totalTokens,
	}, nil
}

// ────────────────────── 风格/情绪/长度映射 ──────────────────────

func fragmentStyleDesc(style string) string {
	switch strings.TrimSpace(style) {
	case "fantasy":
		return "奇幻风格，充满魔法和冒险元素"
	case "realistic":
		return "现实主义风格，贴近日常生活"
	case "anime":
		return "动漫风格，角色形象鲜明"
	case "scifi":
		return "科幻风格，未来科技感"
	default:
		if style != "" {
			return fmt.Sprintf("视觉与叙事贴近「%s」风格（漫画/插画向）", style)
		}
		return "生动有趣的故事风格"
	}
}

func fragmentMoodDesc(mood string) string {
	switch strings.TrimSpace(mood) {
	case "happy":
		return "轻松愉快，积极向上"
	case "sad":
		return "感人至深，略带忧伤"
	case "mysterious":
		return "神秘莫测，引人入胜"
	case "romantic":
		return "浪漫温馨，情感细腻"
	default:
		return "情感丰富，引人共鸣"
	}
}

func fragmentLengthDesc(length string) string {
	switch strings.TrimSpace(length) {
	case "short":
		return "50-100字"
	case "medium":
		return "100-200字"
	case "long":
		return "200-500字"
	default:
		return "100-200字"
	}
}

// currentTime 获取当前时间戳
func currentTime() int64 {
	return time.Now().Unix()
}

// GetTask retrieves a generation task by ID
func (s *FragmentGenerationService) GetTask(ctx context.Context, taskID string) (*domain.FragmentGenerationTask, error) {
	task, err := s.fragmentGenRepo.GetByID(ctx, taskID)
	if err != nil {
		s.logger.Error("Failed to get task", zap.Error(err), zap.String("task_id", taskID))
		return nil, fmt.Errorf("failed to get task: %w", err)
	}
	return task, nil
}

// ListTasks retrieves a list of generation tasks for a user
func (s *FragmentGenerationService) ListTasks(ctx context.Context, userID string, page, limit int) ([]*domain.FragmentGenerationTask, int64, error) {
	offset := (page - 1) * limit
	tasks, total, err := s.fragmentGenRepo.GetByUserID(ctx, userID, limit, offset)
	if err != nil {
		s.logger.Error("Failed to list tasks",
			zap.Error(err),
			zap.String("user_id", userID),
			zap.Int("page", page),
			zap.Int("limit", limit))
		return nil, 0, fmt.Errorf("failed to list tasks: %w", err)
	}
	return tasks, total, nil
}

// CancelTask cancels a pending or processing generation task
func (s *FragmentGenerationService) CancelTask(ctx context.Context, taskID, userID string) error {
	task, err := s.fragmentGenRepo.GetByID(ctx, taskID)
	if err != nil {
		return fmt.Errorf("failed to get task: %w", err)
	}

	// Verify ownership
	if task.UserID != userID {
		return fmt.Errorf("unauthorized: task does not belong to user")
	}

	// Only pending or processing tasks can be cancelled
	if task.Status != "pending" && task.Status != "processing" {
		return fmt.Errorf("task cannot be cancelled: current status is %s", task.Status)
	}

	// Update status to cancelled
	if err := s.fragmentGenRepo.UpdateStatus(ctx, taskID, "cancelled", task.Progress, "cancelled by user"); err != nil {
		s.logger.Error("Failed to cancel task", zap.Error(err), zap.String("task_id", taskID))
		return fmt.Errorf("failed to cancel task: %w", err)
	}

	s.logger.Info("Task cancelled", zap.String("task_id", taskID), zap.String("user_id", userID))
	return nil
}
