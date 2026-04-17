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

// processFragmentGeneration 处理碎片生成流程
func (s *FragmentGenerationService) processFragmentGeneration(ctx context.Context, taskID string) {
	// 更新状态为处理中
	s.fragmentGenRepo.UpdateStatus(ctx, taskID, "processing", 10, "starting")

	task, err := s.fragmentGenRepo.GetByID(ctx, taskID)
	if err != nil {
		s.logger.Error("Failed to get task", zap.Error(err))
		return
	}

	// 步骤1: 生成文字内容
	s.fragmentGenRepo.UpdateStatus(ctx, taskID, "processing", 20, "generating_content")

	contentResult, err := s.generateContent(ctx, task.UserID, task.Request)
	if err != nil {
		s.fragmentGenRepo.UpdateError(ctx, taskID, "failed", fmt.Sprintf("Content generation failed: %v", err))
		return
	}

	// 步骤2: 生成图片
	s.fragmentGenRepo.UpdateStatus(ctx, taskID, "processing", 50, "generating_images")

	imageResult, err := s.generateImages(ctx, task.UserID, taskID, task.Request, contentResult.Content, contentResult.AspectRatio)
	if err != nil {
		s.fragmentGenRepo.UpdateError(ctx, taskID, "failed", fmt.Sprintf("Image generation failed: %v", err))
		return
	}

	// 步骤3: 完成并保存结果
	result := &domain.FragmentGenerationResult{
		Content:     contentResult.Content,
		ImageUrls:   imageResult.ImageUrls,
		AspectRatio: contentResult.AspectRatio,
		TokensUsed:  contentResult.TokensUsed + imageResult.TokensUsed,
	}

	if err := s.fragmentGenRepo.UpdateResult(ctx, taskID, result); err != nil {
		s.logger.Error("Failed to update result", zap.Error(err))
	}

	s.fragmentGenRepo.UpdateStatus(ctx, taskID, "completed", 100, "completed")

	// 更新任务创建时已落库的占位草稿（source = ai_fragment_generation + taskID）
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
		// 兼容旧数据：无占位草稿时仍创建新行
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
		zap.Int("image_count", len(result.ImageUrls)))
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

// generateContent 生成文字内容
func (s *FragmentGenerationService) generateContent(ctx context.Context, userID string, req domain.FragmentGenerationRequest) (*domain.FragmentContentGenerationResult, error) {
	jsonStory := len(fragmentPrefillHTTPImageURLs(req.ImageUrls, 1)) > 0
	prompt := s.buildFragmentStoryPrompt(req, jsonStory)

	payload := map[string]interface{}{"prompt": prompt}
	if imgs := fragmentPrefillHTTPImageURLs(req.ImageUrls, 10); len(imgs) > 0 {
		payload["imageUrls"] = imgs
	}
	inputBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal fragment text input: %w", err)
	}

	aiReq := domain.AITask{
		ID:       uuid.New().String(),
		UserID:   userID,
		Type:     domain.AITaskGenerateFragmentContent,
		Status:   domain.AITaskStatusProcessing,
		Provider: "", // 由 AIService 按用户区域选择火山 / Gemini
		Input:    string(inputBytes),
	}

	content, tokensUsed, inferredAR, err := s.aiService.GenerateTextForFragment(ctx, &aiReq)
	if err != nil {
		return nil, fmt.Errorf("AI text generation failed: %w", err)
	}

	resolved := domain.NormalizeFragmentAspectRatio(req.AspectRatio)
	if resolved == "" {
		resolved = domain.NormalizeFragmentAspectRatio(inferredAR)
	}
	if resolved == "" {
		resolved = domain.FragmentAspectDefault
	}

	return &domain.FragmentContentGenerationResult{
		Content:     content,
		AspectRatio: resolved,
		TokensUsed:  tokensUsed,
	}, nil
}

// generateImages 生成图片（顺序调用：与 AIGenerationService 按 userID 的图片分布式锁一致，避免同用户并发多张时互斥失败）
func (s *FragmentGenerationService) generateImages(ctx context.Context, userID, genTaskID string, req domain.FragmentGenerationRequest, content, aspectRatio string) (*domain.FragmentImageGenerationResult, error) {
	if req.ImageCount <= 0 {
		return &domain.FragmentImageGenerationResult{
			ImageUrls:  []string{},
			TokensUsed: 0,
		}, nil
	}

	ar := domain.NormalizeFragmentAspectRatio(aspectRatio)
	if ar == "" {
		ar = domain.FragmentAspectDefault
	}
	imagePrompt := s.buildImagePrompt(req, content, ar)

	imgInput, err := json.Marshal(map[string]string{
		"prompt":      imagePrompt,
		"aspectRatio": ar,
	})
	if err != nil {
		return nil, fmt.Errorf("marshal fragment image input: %w", err)
	}

	var allImageUrls []string
	totalTokens := 0

	for i := 0; i < req.ImageCount; i++ {
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
			s.logger.Warn("Image generation failed",
				zap.Error(err),
				zap.Int("index", i))
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

// buildFragmentStoryPrompt 构建碎片故事文案提示词；jsonOutput 为 true 时要求输出含 aspectRatio 的 JSON（多模态参考图路径）。
func (s *FragmentGenerationService) buildFragmentStoryPrompt(req domain.FragmentGenerationRequest, jsonOutput bool) string {
	styleDesc := ""
	switch req.Style {
	case "fantasy":
		styleDesc = "奇幻风格，充满魔法和冒险元素"
	case "realistic":
		styleDesc = "现实主义风格，贴近日常生活"
	case "anime":
		styleDesc = "动漫风格，角色形象鲜明"
	case "scifi":
		styleDesc = "科幻风格，未来科技感"
	default:
		if strings.TrimSpace(req.Style) != "" {
			styleDesc = fmt.Sprintf("视觉与叙事贴近「%s」风格（漫画/插画向）", req.Style)
		} else {
			styleDesc = "生动有趣的故事风格"
		}
	}

	moodDesc := ""
	switch req.Mood {
	case "happy":
		moodDesc = "轻松愉快，积极向上"
	case "sad":
		moodDesc = "感人至深，略带忧伤"
	case "mysterious":
		moodDesc = "神秘莫测，引人入胜"
	case "romantic":
		moodDesc = "浪漫温馨，情感细腻"
	default:
		moodDesc = "情感丰富，引人共鸣"
	}

	lengthDesc := ""
	switch req.Length {
	case "short":
		lengthDesc = "50-100字"
	case "medium":
		lengthDesc = "100-200字"
	case "long":
		lengthDesc = "200-500字"
	default:
		lengthDesc = "100-200字"
	}

	base := fmt.Sprintf(`用户输入：%s

要求：
- 风格：%s
- 情绪：%s
- 长度：%s
- 语言：%s`, req.UserInput, styleDesc, moodDesc, lengthDesc, req.Language)

	if jsonOutput {
		return fmt.Sprintf(`请根据用户提供的参考图（若有）与上述文字，理解画面构图与叙事意图，生成一个碎片故事。

%s

请只输出一个 JSON 对象（不要 markdown 代码围栏、不要其他说明），字段为：
- "content": 字符串，故事正文，直接可读；
- "aspectRatio": 字符串，必须是以下之一：1:1、16:9、9:16、3:4、4:3。请结合参考图构图与叙事选择最合适的配图长宽比；不确定时用 "16:9"。

内容要简洁有力，适合社交媒体分享。`, base)
	}

	return fmt.Sprintf(`请根据以下要求生成一个碎片故事：

%s

请直接输出故事内容，不要有任何前缀或解释。内容要简洁有力，适合社交媒体分享。`, base)
}

// buildImagePrompt 构建图片生成提示词（将已定比例写入提示，便于各文生图模型对齐构图）。
func (s *FragmentGenerationService) buildImagePrompt(req domain.FragmentGenerationRequest, content, aspectRatio string) string {
	ar := aspectRatio
	if ar == "" {
		ar = domain.FragmentAspectDefault
	}
	return fmt.Sprintf("%s，%s风格，%s氛围，画面长宽比 %s",
		extractKeywords(content, 50),
		req.Style,
		req.Mood,
		ar)
}

// extractKeywords 从文本中提取关键词
func extractKeywords(text string, maxWords int) string {
	// 简化的关键词提取逻辑
	// 实际项目中可以使用更复杂的 NLP 算法
	words := []string{}
	wordCount := 0

	// 简单分词（按空格和标点）
	currentWord := ""
	for _, char := range text {
		if (char >= 'a' && char <= 'z') || (char >= 'A' && char <= 'Z') ||
			(char >= '0' && char <= '9') || (char >= 0x4e00 && char <= 0x9fff) {
			currentWord += string(char)
		} else {
			if len(currentWord) > 0 {
				words = append(words, currentWord)
				wordCount++
				if wordCount >= maxWords {
					break
				}
				currentWord = ""
			}
		}
	}

	if len(currentWord) > 0 {
		words = append(words, currentWord)
	}

	// 简单去重并拼接
	uniqueWords := []string{}
	seen := make(map[string]bool)
	for _, word := range words {
		if !seen[word] {
			uniqueWords = append(uniqueWords, word)
			seen[word] = true
		}
	}

	result := ""
	for i, word := range uniqueWords {
		if i > 0 && i < 5 {
			result += ", "
		}
		if i >= 5 {
			break
		}
		result += word
	}

	return result
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
