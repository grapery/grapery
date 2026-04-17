package service

import (
	"context"
	"encoding/json"
	"fmt"
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
	aiService       *AIService
	logger          *zap.Logger
}

func NewFragmentGenerationService(
	fragmentGenRepo *repository.FragmentGenerationRepository,
	fragmentRepo *repository.FragmentRepository,
	aiService *AIService,
	logger *zap.Logger,
) *FragmentGenerationService {
	return &FragmentGenerationService{
		fragmentGenRepo: fragmentGenRepo,
		fragmentRepo:    fragmentRepo,
		aiService:       aiService,
		logger:          logger,
	}
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

// processFragmentGeneration 处理碎片生成流程：
// Step 1: 元素提取 + 文案生成（一次 LLM 调用）
// Step 2: 场景扩展（将元素+文案扩展为 N 个场景，每个场景含独立的图片提示词）
// Step 3: 逐场景生成图片
func (s *FragmentGenerationService) processFragmentGeneration(ctx context.Context, taskID string) {
	s.fragmentGenRepo.UpdateStatus(ctx, taskID, "processing", 5, "starting")

	task, err := s.fragmentGenRepo.GetByID(ctx, taskID)
	if err != nil {
		s.logger.Error("Failed to get task", zap.Error(err))
		return
	}

	// ── Step 1: 元素提取 + 文案生成 ──
	s.fragmentGenRepo.UpdateStatus(ctx, taskID, "processing", 10, "extracting_elements")

	elemResult, err := s.extractElementsAndGenerateContent(ctx, task.UserID, task.Request)
	if err != nil {
		s.fragmentGenRepo.UpdateError(ctx, taskID, "failed", fmt.Sprintf("Element extraction failed: %v", err))
		return
	}

	resolvedAR := domain.NormalizeFragmentAspectRatio(task.Request.AspectRatio)
	if resolvedAR == "" {
		resolvedAR = domain.NormalizeFragmentAspectRatio(elemResult.AspectRatio)
	}
	if resolvedAR == "" {
		resolvedAR = domain.FragmentAspectDefault
	}

	// ── Step 2: 场景扩展 ──
	s.fragmentGenRepo.UpdateStatus(ctx, taskID, "processing", 35, "expanding_scenes")

	sceneCount := task.Request.ImageCount
	if sceneCount <= 0 {
		sceneCount = 1
	}
	scenesResult, err := s.expandScenes(ctx, task.UserID, task.Request, elemResult, sceneCount, resolvedAR)
	if err != nil {
		s.fragmentGenRepo.UpdateError(ctx, taskID, "failed", fmt.Sprintf("Scene expansion failed: %v", err))
		return
	}

	// ── Step 3: 逐场景生成图片 ──
	s.fragmentGenRepo.UpdateStatus(ctx, taskID, "processing", 55, "generating_images")

	imageResult, err := s.generateImagesFromScenes(ctx, task.UserID, taskID, resolvedAR, scenesResult.Scenes)
	if err != nil {
		s.fragmentGenRepo.UpdateError(ctx, taskID, "failed", fmt.Sprintf("Image generation failed: %v", err))
		return
	}

	// ── 汇总结果 ──
	result := &domain.FragmentGenerationResult{
		Content:     elemResult.Content,
		ImageUrls:   imageResult.ImageUrls,
		AspectRatio: resolvedAR,
		TokensUsed:  elemResult.TokensUsed + scenesResult.TokensUsed + imageResult.TokensUsed,
	}

	if err := s.fragmentGenRepo.UpdateResult(ctx, taskID, result); err != nil {
		s.logger.Error("Failed to update result", zap.Error(err))
	}

	s.fragmentGenRepo.UpdateStatus(ctx, taskID, "completed", 100, "completed")

	// 更新占位草稿
	now := time.Now().UnixMilli()
	imgCount := len(result.ImageUrls)
	if imgCount < 1 {
		imgCount = 1
	}
	style := task.Request.Style
	caption := captionFromGenerationContent(result.Content)

	existing, gerr := s.fragmentRepo.GetBySource(ctx, string(domain.FragmentSourceAIGeneration), taskID)
	if gerr == nil && existing != nil {
		existing.Content = result.Content
		existing.MediaURLs = append([]string(nil), result.ImageUrls...)
		existing.ImageUrls = stringifyGenerationImageURLs(result.ImageUrls)
		existing.Style = &style
		existing.FragmentCount = &imgCount
		existing.UpdatedAt = now
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
			if err := s.fragmentGenRepo.UpdateResult(ctx, taskID, result); err != nil {
				s.logger.Warn("Failed to persist draftFragmentId on generation task", zap.Error(err))
			}
		}
	} else {
		fragment := &domain.Fragment{
			BaseModel: common.BaseModel{
				ID:        uuid.New().String(),
				CreatedAt: now,
				UpdatedAt: now,
			},
			UserID:          task.UserID,
			CreatorID:       task.UserID,
			Content:         result.Content,
			MediaURLs:       append([]string(nil), result.ImageUrls...),
			ImageUrls:       stringifyGenerationImageURLs(result.ImageUrls),
			Style:           &style,
			FragmentCount:   &imgCount,
			Visibility:      domain.FragmentVisibilityPrivate,
			IsDraft:         true,
			SourceType:      string(domain.FragmentSourceOriginal),
			SourceID:        taskID,
			EngagementStats: common.EngagementStats{},
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
			if err := s.fragmentGenRepo.UpdateResult(ctx, taskID, result); err != nil {
				s.logger.Warn("Failed to persist draftFragmentId on generation task", zap.Error(err))
			}
		}
	}

	s.logger.Info("Fragment generation completed",
		zap.String("task_id", taskID),
		zap.Int("tokens_used", result.TokensUsed),
		zap.Int("image_count", len(result.ImageUrls)),
		zap.Int("scenes", len(scenesResult.Scenes)))
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
	Elements    fragmentStoryElements `json:"elements"`
	Content     string                `json:"content"`
	AspectRatio string                `json:"aspectRatio"`
	TokensUsed  int
}

// fragmentStoryElements 从用户输入和参考图中提取的结构化元素。
type fragmentStoryElements struct {
	Weather     string   `json:"weather"`     // 天气：晴朗、雨天、暴风雪等
	Objects     []string `json:"objects"`     // 关键物品
	Scenes      []string `json:"scenes"`      // 场景类型：室内/室外/森林/城市等
	TimeOfDay   string   `json:"timeOfDay"`   // 时间：清晨/正午/黄昏/深夜等
	Location    string   `json:"location"`    // 地点描述
	Characters  []string `json:"characters"`  // 人物/角色描述
	Tendency    string   `json:"tendency"`    // 情感倾向/叙事方向
}

func (s *FragmentGenerationService) extractElementsAndGenerateContent(ctx context.Context, userID string, req domain.FragmentGenerationRequest) (*fragmentElementExtractionResult, error) {
	hasImages := len(fragmentPrefillHTTPImageURLs(req.ImageUrls, 1)) > 0
	prompt := s.buildExtractionAndStoryPrompt(req, hasImages)

	payload := map[string]interface{}{"prompt": prompt}
	if imgs := fragmentPrefillHTTPImageURLs(req.ImageUrls, 10); len(imgs) > 0 {
		payload["imageUrls"] = imgs
	}
	inputBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal extraction input: %w", err)
	}

	aiReq := domain.AITask{
		ID:       uuid.New().String(),
		UserID:   userID,
		Type:     domain.AITaskGenerateFragmentContent,
		Status:   domain.AITaskStatusProcessing,
		Provider: "",
		Input:    string(inputBytes),
	}

	raw, tokensUsed, inferredAR, err := s.aiService.GenerateTextForFragment(ctx, &aiReq)
	if err != nil {
		return nil, fmt.Errorf("AI extraction+story generation failed: %w", err)
	}

	result, perr := parseExtractionResult(raw)
	if perr != nil {
		s.logger.Warn("extraction JSON parse failed, using raw text as content",
			zap.Error(perr),
			zap.String("snippet", truncateRunes(raw, 120)))
		// 回退：JSON 解析失败时把原文作为 content
		result = &fragmentElementExtractionResult{
			Content:     truncateRunes(raw, 500),
			AspectRatio: inferredAR,
		}
	}
	if result.AspectRatio == "" {
		result.AspectRatio = inferredAR
	}
	result.TokensUsed = tokensUsed
	return result, nil
}

// buildExtractionAndStoryPrompt 构建元素提取 + 文案生成的组合提示词。
func (s *FragmentGenerationService) buildExtractionAndStoryPrompt(req domain.FragmentGenerationRequest, hasImages bool) string {
	styleDesc := fragmentStyleDesc(req.Style)
	moodDesc := fragmentMoodDesc(req.Mood)
	lengthDesc := fragmentLengthDesc(req.Length)

	imgNote := ""
	if hasImages {
		imgNote = `

用户提供了参考图。请先理解参考图的画面内容（构图、主体、环境、色彩、情绪），将其作为元素提取的重要来源。`
	}

	return fmt.Sprintf(`你是一位专业的故事碎片创作助手。你的任务分两步：

第一步：从用户的文字描述%s中提取故事元素，包括：
- weather（天气）：天气状况
- objects（物品）：画面中关键物品（最多 5 个）
- scenes（场景）：场景类型/空间描述（最多 3 个）
- timeOfDay（时间）：一天中的时段
- location（地点）：具体地点
- characters（人物）：角色外观与身份描述（最多 3 个）
- tendency（倾向）：整体情感走向与叙事基调

第二步：基于提取的元素，写一个碎片故事。

用户输入：%s

要求：
- 风格：%s
- 情绪：%s
- 长度：%s
- 语言：%s

写作指引：
1. 开头用 1-2 句具体的环境描写建立画面（光线方向、色调、空间感）
2. 角色或物体要有可辨识的外形特征（衣着材质、姿态、表情）
3. 关键动作或转折处用动词驱动，让读者能"看到"画面而非被告知
4. 避免纯抽象心理描写，用场景细节和动作暗示情绪

请只输出一个 JSON 对象（不要 markdown 代码围栏、不要其他说明），字段为：
{
  "elements": {
    "weather": "天气描述（中文）",
    "objects": ["物品1", "物品2"],
    "scenes": ["场景1", "场景2"],
    "timeOfDay": "时段描述（中文）",
    "location": "地点描述（中文）",
    "characters": ["角色1：外观+身份", "角色2"],
    "tendency": "情感倾向与叙事方向（中文）"
  },
  "content": "基于以上元素写出的碎片故事正文（中文），直接可读",
  "aspectRatio": "配图推荐长宽比，必须是以下之一：1:1、16:9、9:16、3:4、4:3。结合画面构图选择；不确定时用 16:9"
}`,
		imgNote,
		req.UserInput, styleDesc, moodDesc, lengthDesc, req.Language)
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
	return &out, nil
}

// ────────────────────── Step 2: 场景扩展 ──────────────────────

// fragmentSceneExpansionResult 场景扩展输出。
type fragmentSceneExpansionResult struct {
	Scenes     []fragmentExpandedScene `json:"scenes"`
	TokensUsed int
}

// fragmentExpandedScene 扩展出的单个场景，含中文描述和英文图片提示词。
type fragmentExpandedScene struct {
	Index         int    `json:"index"`
	SceneDesc     string `json:"sceneDesc"`     // 中文场景描述（面向读者）
	ImagePrompt   string `json:"imagePrompt"`   // 英文图片生成提示词（面向图片模型）
}

func (s *FragmentGenerationService) expandScenes(ctx context.Context, userID string, req domain.FragmentGenerationRequest, elemResult *fragmentElementExtractionResult, sceneCount int, aspectRatio string) (*fragmentSceneExpansionResult, error) {
	elementsJSON, _ := json.Marshal(elemResult.Elements)

	var narrativeHint string
	switch {
	case sceneCount == 1:
		narrativeHint = `1 格：选取故事中最有视觉冲击力的一瞬间——可以是一个悬念、一个反转、一个让人好奇"之前发生了什么"的定格。`
	case sceneCount == 2:
		narrativeHint = `2 格：两格之间制造反差或悬念。可以是从平静到意外、从微观到宏观、从现实到奇幻——让观众看完想"然后呢？"或者"等等，怎么会这样？"`
	case sceneCount == 3:
		narrativeHint = `3 格：不必严格三幕式。可以是一个出乎意料的转折打破常规节奏，比如第二格突然切换视角、时空跳转、或者出现意想不到的元素。惊喜比工整更重要。`
	default:
		narrativeHint = `%d 格：整体是一条故事线，但不必步步紧接。允许在中间插入：
- 意外的视角切换（突然变成俯视、虫眼视角、镜中倒影）
- 时空的跳跃或闪回
- 打破第四面墙的幽默
- 超现实的梦幻/想象片段
- 一个完全出乎意料的新元素闯入

开头建立世界观，中间自由发挥制造惊喜，结尾呼应或留下悬念。让观众看完觉得"没猜到会这样"。`
	}

	if sceneCount >= 4 {
		narrativeHint = fmt.Sprintf(narrativeHint, sceneCount)
	}

	prompt := fmt.Sprintf(`你是一位脑洞大开的漫画分镜师。基于以下故事元素和正文，将故事扩展为 %d 个画面格，合成一组有趣的故事碎片。

%s

故事元素：
%s

故事正文：
%s

风格：%s
情绪：%s
画面长宽比：%s

创作原则：
1. 世界观一致：角色、核心设定、视觉风格在所有格之间统一
2. 但叙事可以自由——允许跳切、闪回、视角反转、超现实片段、意外闯入的新元素
3. 构图要有变化：不要每格都是正面中景，混用远景/特写/俯拍/仰拍/ Dutch angle
4. 光线和色调可以配合情绪变化：暖 → 冷、明亮 → 阴暗、真实 → 梦幻，不必严格对应时间推移
5. 每格 sceneDesc 简要说明画面内容，让观众按顺序浏览时能感受到故事的走向
6. 优先有趣 > 优先合理。一个让人"哇没想到"的画面比一个逻辑完美但无聊的画面好得多

请只输出一个 JSON 对象（不要 markdown 代码围栏、不要其他说明）：
{
  "scenes": [
    {
      "index": 0,
      "sceneDesc": "中文：这格画面的内容描述",
      "imagePrompt": "English: detailed visual description for image generation including art style, character appearance, environment, composition, lighting, color palette, mood. At least 50 words."
    }
  ]
}

硬性规则：
- scenes 数组恰好 %d 项，index 从 0 到 %d
- imagePrompt 必须是英文，sceneDesc 必须是中文
- 每个 imagePrompt 至少 50 个英文单词，内容具体可画
- 角色/核心设定的外观在各格之间保持一致`,
		sceneCount,
		narrativeHint,
		sceneCount,
		string(elementsJSON),
		elemResult.Content,
		req.Style,
		req.Mood,
		aspectRatio,
		sceneCount,
		sceneCount-1)

	payloadBytes, _ := json.Marshal(map[string]interface{}{"prompt": prompt})
	aiReq := domain.AITask{
		ID:       uuid.New().String(),
		UserID:   userID,
		Type:     domain.AITaskGenerateFragmentContent,
		Status:   domain.AITaskStatusProcessing,
		Provider: "",
		Input:    string(payloadBytes),
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

// ────────────────────── Step 3: 逐场景生成图片 ──────────────────────

func (s *FragmentGenerationService) generateImagesFromScenes(ctx context.Context, userID, genTaskID, aspectRatio string, scenes []fragmentExpandedScene) (*domain.FragmentImageGenerationResult, error) {
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

	var allImageUrls []string
	totalTokens := 0

	for _, scene := range scenes {
		imgInput, _ := json.Marshal(map[string]string{
			"prompt":      scene.ImagePrompt,
			"aspectRatio": ar,
		})

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
