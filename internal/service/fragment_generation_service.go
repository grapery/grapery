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

用户提供了参考图。请仔细观察参考图，提取以下视觉信息：
- 画面中的主体人物/物体（外观、姿态、表情、穿着）
- 环境场景（室内/室外、建筑/自然、季节特征）
- 色调与光影（暖色/冷色、光源方向、明暗对比）
- 构图特征（景别、视角、前景/背景层次）
- 整体氛围与情绪暗示

将以上视觉信息融入元素提取，确保生成的故事与参考图有画面上的呼应，而不是只基于文字。`
	}

	return fmt.Sprintf(`你是一位擅长在短篇幅中制造惊喜的故事碎片创作助手。你需要完成两件事：

── 第一件事：元素提取 ──

从用户的文字描述%s中提炼出以下故事元素。元素不是简单的关键词罗列，而是带有画面感和叙事暗示的具体描述：

- weather（天气）：不只是"晴天/雨天"，要写出天气的氛围感（如"雨后初晴，空气里有青草被碾碎的味道"）
- objects（物品）：画面中关键物品，写清材质、状态、与角色的关系（最多 5 个）
- scenes（场景）：场景的空间感和感官细节——声音、气味、触感（最多 3 个）
- timeOfDay（时间）：时段 + 光线状态（如"正午，日光像白色刀刃劈进走廊"）
- location（地点）：具体地点，包含时代感和空间特征
- characters（人物）：角色的视觉身份卡——体型、穿着、标志性特征、当前状态（最多 3 个）
- tendency（倾向）：用一句话概括这段故事想传达的核心感受，不是分类标签而是"如果这是一首歌的封面，上面写着什么"

── 第二件事：碎片故事 ──

基于提取的元素，写一个让人看完想转发的碎片故事。

用户输入：%s

要求：
- 风格：%s
- 情绪：%s
- 长度：%s
- 语言：%s

故事心法：
- 第一句话就要有画面——读者点进来不是因为"你好"开头的故事
- 少用形容词堆砌，多用动词和具体名词推动画面（"她把伞扔进河里" > "她感到无比解脱"）
- 留白比写满好——最好的结尾是让读者自己脑补下一秒
- 可以有一句出人意料的话或转折，但不强求
- 如果是奇幻/科幻风格，用一个小细节建立世界观就够了（"咖啡杯飘在半空，她懒得去接"）

请只输出一个 JSON 对象（不要 markdown 代码围栏、不要其他说明），字段为：
{
  "elements": {
    "weather": "天气氛围描述（中文）",
    "objects": ["物品1：具体描述", "物品2"],
    "scenes": ["场景1：感官细节", "场景2"],
    "timeOfDay": "时段+光线（中文）",
    "location": "地点+时代感（中文）",
    "characters": ["角色1：外观+身份+当前状态", "角色2"],
    "tendency": "核心感受一句话（中文）"
  },
  "content": "碎片故事正文（中文），直接可读，不要标题",
  "aspectRatio": "推荐长宽比：1:1、16:9、9:16、3:4、4:3，结合画面构图选择；不确定用 16:9"
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
		narrativeHint = "1 格：选取故事中最有视觉冲击力的一瞬间——可以是一个悬念、一个反转、一个让人好奇\"之前发生了什么\"的定格。让这一帧本身就充满故事张力。"
	case sceneCount == 2:
		narrativeHint = "2 格：制造认知落差。第一格建立预期，第二格打破它——可以是视角的反转、情绪的急转、尺度的突变。观众看完两格后应该产生\"等等，怎么会这样？\"的感觉。"
	case sceneCount == 3:
		narrativeHint = "3 格：自由节奏，不要三幕式。第一格抽出引子，第二格可以突然转向（换视角、换时空、换叙事者），第三格收束但不一定给答案——留一个让人回味的尾巴。惊喜感比工整重要十倍。"
	default:
		narrativeHint = fmt.Sprintf("%d 格：一条故事线但不拘泥线性叙事。鼓励在这些格中穿插：视角突变（俯视/虫眼/镜中倒影/万物视角）、时空跳跃或闪回、打破第四面墙、超现实梦境、意料之外的新元素闯入。开头建立世界观，中间尽情放飞制造惊喜，结尾可以呼应开头也可以留悬念。", sceneCount)
	}

	prompt := fmt.Sprintf(`你是一位脑洞大开的视觉故事创作者。基于以下故事元素和正文，将故事扩展为 %d 个画面格，创造一组让人过目不忘的故事碎片。

%s

故事元素：
%s

故事正文：
%s

风格：%s
情绪：%s
画面长宽比：%s

创作原则：
1. 世界观一致：角色外貌、核心设定、视觉风格在所有格之间统一
2. 叙事自由奔放——跳切、闪回、视角反转、超现实片段、意外闯入的新元素，全部欢迎
3. 构图大胆变化：远景/特写/俯拍/仰拍/Dutch angle/非人类视角（虫眼/鸟瞰/鱼眼），拒绝连续两格相同构图
4. 光影和色调为情绪服务：暖转冷、明亮转阴暗、真实转梦幻，不必对应时间推移
5. sceneDesc 是给读者的：按顺序浏览时应能感受到故事走向，同时保留悬念和惊喜
6. 有趣 > 合理。一个让人"哇没想到"的画面远胜于逻辑完美但无聊的画面
7. imagePrompt 要像给一位顶级插画师下 brief：具体到色彩、材质、光线方向、镜头语言

请只输出一个 JSON 对象（不要 markdown 代码围栏、不要其他说明）：
{
  "scenes": [
    {
      "index": 0,
      "sceneDesc": "中文：这格画面的内容描述，简洁有画面感",
      "imagePrompt": "English: structured visual description. Must include these layers separated by periods: (1) artStyle: overall visual approach, e.g. 'cinematic watercolor', 'hyperrealistic 3D render', 'charcoal sketch with digital color' (2) subject: who/what is in frame, specific appearance and pose (3) environment: background, setting, depth cues (4) composition: camera angle, shot type, framing, rule-of-thirds or centered or asymmetrical (5) lighting: direction, color temperature, shadows, highlights (6) colorPalette: dominant colors, contrast level (7) mood: emotional atmosphere (8) any extra details: textures, particles, weather effects, lens effects. At least 60 words total."
    }
  ]
}

硬性规则：
- scenes 数组恰好 %d 项，index 从 0 到 %d
- imagePrompt 必须是英文，sceneDesc 必须是中文
- 每个 imagePrompt 至少 60 个英文单词，覆盖上述 8 个视觉层
- 所有格的 artStyle 必须一致（相同的艺术风格前缀），角色外观在各格之间保持一致
- 相邻格的构图必须有明显差异（不能连续两格正面中景）`,
		sceneCount,
		narrativeHint,
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
