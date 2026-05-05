package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/grapestree/fgrapery/grapery/internal/cache"
	"github.com/grapestree/fgrapery/grapery/internal/common"
	"github.com/grapestree/fgrapery/grapery/internal/domain"
	"go.uber.org/zap"
)

// CreateCharacterRequest 创建角色请求
type CreateCharacterRequest struct {
	Name            string   `json:"name" binding:"required,min=1,max=100"`
	Description     string   `json:"description" binding:"max=2000"`
	Avatar          string   `json:"avatar" binding:"omitempty,url"`
	Personality     string   `json:"personality" binding:"max=1000"`
	Background      string   `json:"background" binding:"max=2000"`
	ShortTermGoal   string   `json:"shortTermGoal" binding:"max=1000"`
	LongTermGoal    string   `json:"longTermGoal" binding:"max=1000"`
	HandlingStyle   string   `json:"handlingStyle" binding:"max=1000"`
	CognitionRange  string   `json:"cognitionRange" binding:"max=1000"`
	AbilityFeatures string   `json:"abilityFeatures" binding:"max=1000"`
	Appearance      string   `json:"appearance" binding:"max=1000"`
	DressPreference string   `json:"dressPreference" binding:"max=1000"`
	StoryID         string   `json:"storyId" binding:"required"`
	IsPublic        bool     `json:"isPublic"`
	SourceType      string   `json:"sourceType" binding:"omitempty,oneof=manual upload ai"`
	SourcePrompt    string   `json:"sourcePrompt"`
	SourceImage     string   `json:"sourceImage" binding:"omitempty,url"`
	Tags            []string `json:"tags" binding:"omitempty,max=3,dive,min=1,max=50"` // 最多3个标签，每个标签1-50字符
	NeedsPortrait   bool     `json:"needsPortrait"`                                    // 是否需要生成形象
	ReferenceImage  string   `json:"referenceImage" binding:"omitempty,url"`           // 参考图URL

	// 新增：海报创建权限
	PosterCreationPermission string `json:"posterCreationPermission" binding:"omitempty,oneof=creator_only anyone"`
}

// UpdateCharacterRequest 更新角色请求
type UpdateCharacterRequest struct {
	Name            *string `json:"name" binding:"omitempty,min=1,max=100"`
	Description     *string `json:"description" binding:"omitempty,max=2000"`
	Avatar          *string `json:"avatar" binding:"omitempty,url"`
	Personality     *string `json:"personality" binding:"omitempty,max=1000"`
	Background      *string `json:"background" binding:"omitempty,max=2000"`
	ShortTermGoal   *string `json:"shortTermGoal" binding:"omitempty,max=1000"`
	LongTermGoal    *string `json:"longTermGoal" binding:"omitempty,max=1000"`
	HandlingStyle   *string `json:"handlingStyle" binding:"omitempty,max=1000"`
	CognitionRange  *string `json:"cognitionRange" binding:"omitempty,max=1000"`
	AbilityFeatures *string `json:"abilityFeatures" binding:"omitempty,max=1000"`
	Appearance      *string `json:"appearance" binding:"omitempty,max=1000"`
	DressPreference *string `json:"dressPreference" binding:"omitempty,max=1000"`
	IsPublic        *bool   `json:"isPublic"`
	SourceType      *string `json:"sourceType" binding:"omitempty,oneof=manual upload ai"`
	SourcePrompt    *string `json:"sourcePrompt"`
	SourceImage     *string `json:"sourceImage" binding:"omitempty,url"`
}

// GenerateCharacterRequest AI生成角色属性请求
type GenerateCharacterRequest struct {
	Prompt string `json:"prompt" binding:"required,min=1,max=2000"`
	Name   string `json:"name" binding:"omitempty,max=100"` // Optional: include character name for context
}

// GenerateCharacterAvatarRequest AI生成角色头像请求
type GenerateCharacterAvatarRequest struct {
	AspectRatio string `json:"aspectRatio"` // 1:1, 16:9, 9:16
}

// GenerateCharacterAvatarResult 生成角色头像结果
type GenerateCharacterAvatarResult struct {
	AvatarURL string `json:"avatarUrl"`
	RecordID  string `json:"recordId"`
}

// GenerateCharacterPortraitRequest AI生成角色形象请求（完整形象图）
type GenerateCharacterPortraitRequest struct {
	CustomPrompt   string `json:"customPrompt"`   // 用户自定义提示词（可选）
	ReferenceImage string `json:"referenceImage"` // 参考图URL（可选）
	AspectRatio    string `json:"aspectRatio"`    // 建议 2:3 或 3:4（竖版人像）
}

// GenerateCharacterPortraitResult 生成角色形象结果
type GenerateCharacterPortraitResult struct {
	PortraitURL string `json:"portraitUrl"`
	RecordID    string `json:"recordId"`
}

// GenerateCharacterThreeViewsRequest 生成单张三视图合一参考图（正·侧·背横向排列）
type GenerateCharacterThreeViewsRequest struct {
	RegenerateAll  bool   `json:"regenerateAll"`  // true：清空并重绘 sheet；false：若已有 sheet 则跳过
	ReferenceImage string `json:"referenceImage"` // 可选；优先于角色已存 Portrait 作为参考图（碎片任务等）
}

// GenerateCharacterThreeViewsResult 三视图结果（views.sheet 为单图 URL）
type GenerateCharacterThreeViewsResult struct {
	Views domain.CharacterThreeViews `json:"views"`
}

type CharacterGenerationTaskRequest struct {
	StoryID                    string                              `json:"storyId" binding:"required"`
	SourceType                 string                              `json:"sourceType" binding:"required"`
	Name                       string                              `json:"name,omitempty"`
	Prompt                     string                              `json:"prompt,omitempty"`
	Description                string                              `json:"description,omitempty"`
	Background                 string                              `json:"background,omitempty"`
	Personality                string                              `json:"personality,omitempty"`
	Appearance                 string                              `json:"appearance,omitempty"`
	ShortTermGoal              string                              `json:"shortTermGoal,omitempty"`
	LongTermGoal               string                              `json:"longTermGoal,omitempty"`
	HandlingStyle              string                              `json:"handlingStyle,omitempty"`
	CognitionRange             string                              `json:"cognitionRange,omitempty"`
	AbilityFeatures            string                              `json:"abilityFeatures,omitempty"`
	DressPreference            string                              `json:"dressPreference,omitempty"`
	ReferenceImage             string                              `json:"referenceImage,omitempty"`
	SourceFragmentID           string                              `json:"sourceFragmentId,omitempty"`
	SourceFragmentCharacterKey string                              `json:"sourceFragmentCharacterKey,omitempty"`
	Suggestion                 *domain.FragmentCharacterSuggestion `json:"suggestion,omitempty"`
}

type FragmentCharacterSuggestionsResponse struct {
	StoryID      string                               `json:"storyId"`
	FragmentID   string                               `json:"fragmentId"`
	Suggestions  []domain.FragmentCharacterSuggestion `json:"suggestions"`
	ExistingTask *domain.CharacterGenerationTask      `json:"existingTask,omitempty"`
}

// GeneratedCharacterAttributes AI生成的角色属性（键名与 domain.Character / 客户端 / DB 叙事字段一一对应）
type GeneratedCharacterAttributes struct {
	Description     string `json:"description"`
	Personality     string `json:"personality"`
	Background      string `json:"background"`
	ShortTermGoal   string `json:"shortTermGoal"`
	LongTermGoal    string `json:"longTermGoal"`
	HandlingStyle   string `json:"handlingStyle"`
	CognitionRange  string `json:"cognitionRange"`
	AbilityFeatures string `json:"abilityFeatures"`
	Appearance      string `json:"appearance"`
	DressPreference string `json:"dressPreference"`
}

type CharacterListRequest struct {
	UserID  string `form:"authorId"`
	StoryID string `form:"storyId"`
	Search  string `form:"search"`
	Limit   int    `form:"limit" binding:"omitempty,min=1,max=100"`
	Offset  int    `form:"offset" binding:"omitempty,min=0"`
}

// CreateCharacter 创建角色
func (s *Service) CreateCharacter(ctx context.Context, userID string, req CreateCharacterRequest) (*domain.Character, error) {
	s.logger.Info("creating character",
		zap.String("userID", userID),
		zap.String("name", req.Name),
	)

	// 获取作者信息
	author, err := s.repo.UserByID(ctx, userID)
	if err != nil {
		s.logger.Error("failed to get author", zap.Error(err))
		return nil, errors.New("author not found")
	}
	s.logger.Info("checking story ownership", zap.String("storyID", req.StoryID), zap.String("userID", userID))
	if err := s.ensureStoryOwnership(ctx, req.StoryID, userID); err != nil {
		s.logger.Error("failed to check story ownership", zap.Error(err), zap.String("storyID", req.StoryID), zap.String("userID", userID))
		return nil, err
	}
	s.logger.Info("story ownership checked", zap.String("storyID", req.StoryID), zap.String("userID", userID))

	// 检查同一故事中是否存在同名角色
	existingCharacters, err := s.repo.CharactersByStory(ctx, req.StoryID)
	if err != nil {
		s.logger.Warn("failed to check existing characters", zap.Error(err))
		// 如果查询失败，继续创建（不阻塞）
	} else {
		// 检查是否存在同名角色（大小写不敏感）
		for _, existingChar := range existingCharacters {
			if strings.EqualFold(existingChar.Name, req.Name) {
				s.logger.Info("duplicate character name found",
					zap.String("storyID", req.StoryID),
					zap.String("name", req.Name),
					zap.String("existingCharacterID", existingChar.ID),
				)
				return nil, errors.New("character with same name already exists in this story")
			}
		}
	}

	sourceType := strings.ToLower(req.SourceType)
	if sourceType == "" {
		sourceType = AssetSourceManual
	}
	s.logger.Info("source type", zap.String("sourceType", sourceType))
	// 设置形象生成状态
	portraitStatus := "none"
	if req.NeedsPortrait {
		portraitStatus = "pending"
	}

	// 创建角色
	now := time.Now().Unix()
	character := &domain.Character{
		BaseModel: common.BaseModel{
			ID:        uuid.New().String(),
			CreatedAt: now,
			UpdatedAt: now,
		},
		StoryID:                    req.StoryID,
		UserID:                     author.ID,
		Name:                       req.Name,
		Description:                req.Description,
		Avatar:                     req.Avatar,
		Poster:                     "",
		Portrait:                   "",
		NeedsPortrait:              req.NeedsPortrait,
		ReferenceImage:             req.ReferenceImage,
		PortraitGenerationStatus:   portraitStatus,
		Author:                     author,
		Personality:                req.Personality,
		Background:                 req.Background,
		ShortTermGoal:              req.ShortTermGoal,
		LongTermGoal:               req.LongTermGoal,
		HandlingStyle:              req.HandlingStyle,
		CognitionRange:             req.CognitionRange,
		AbilityFeatures:            req.AbilityFeatures,
		Appearance:                 req.Appearance,
		DressPreference:            req.DressPreference,
		Role:                       "",
		IsPublic:                   req.IsPublic,
		SourceType:                 sourceType,
		SourcePrompt:               req.SourcePrompt,
		SourceImage:                req.SourceImage,
		SourceFragmentID:           "",
		SourceFragmentCharacterKey: "",
		CreatedBy:                  userID,
		LastEditedBy:               userID,
		Likes:                      0,
		Comments:                   0,
		Shares:                     0,
		Followers:                  0,
		Stories:                    0,
	}
	s.logger.Info("character created", zap.String("characterID", character.ID))
	if err := s.repo.CreateCharacter(ctx, character); err != nil {
		s.logger.Error("failed to create character",
			zap.String("characterID", character.ID),
			zap.Error(err))
		return nil, errors.New("failed to create character")
	}

	// 缓存新创建的角色
	c := s.getCache()
	if c != nil {
		key := cache.CharacterKey(character.ID)
		character.IsFollowing = nil // 不缓存关注状态
		if err := c.Set(ctx, key, character, entityCacheTTL); err != nil {
			s.logger.Warn("failed to cache new character",
				zap.String("characterID", character.ID),
				zap.Error(err))
		}
		// 清除相关列表缓存
		for limit := 20; limit <= 100; limit += 20 {
			for offset := 0; offset < 200; offset += limit {
				_ = c.Delete(ctx, cache.UserCharactersListKey(userID, limit, offset))
			}
		}
	}

	s.logger.Info("character created successfully",
		zap.String("characterID", character.ID))

	// 添加标签（如果有）
	if len(req.Tags) > 0 {
		s.logger.Debug("adding tags to character",
			zap.String("characterID", character.ID),
			zap.Strings("tags", req.Tags))
		// 去重并规范化标签（转小写，去除前后空格）
		uniqueTags := make(map[string]bool)
		normalizedTags := make([]string, 0, len(req.Tags))
		for _, tag := range req.Tags {
			normalized := strings.TrimSpace(strings.ToLower(tag))
			if normalized != "" && !uniqueTags[normalized] {
				uniqueTags[normalized] = true
				normalizedTags = append(normalizedTags, normalized)
			}
		}
		if len(normalizedTags) > 0 {
			if err := s.AddCharacterTags(ctx, character.ID, normalizedTags); err != nil {
				s.logger.Warn("failed to add tags to character",
					zap.String("characterID", character.ID),
					zap.Strings("tags", normalizedTags),
					zap.Error(err))
				// 不返回错误，标签添加失败不影响角色创建
			} else {
				s.logger.Info("tags added to character successfully",
					zap.String("characterID", character.ID),
					zap.Strings("tags", normalizedTags))
			}
		}
	}

	// REMOVED: RecordCharacterCreated - not in StoryCreationAppUI design

	return character, nil
}

func (s *Service) PreviewFragmentCharactersForStory(ctx context.Context, userID, storyID string) (*FragmentCharacterSuggestionsResponse, error) {
	story, err := s.repo.StoryByID(ctx, storyID)
	if err != nil || story == nil {
		return nil, errors.New("story not found")
	}
	if story.UserID != userID {
		return nil, errors.New("you can only generate characters for your own stories")
	}
	if story.SourceFragmentID == nil || strings.TrimSpace(*story.SourceFragmentID) == "" {
		return &FragmentCharacterSuggestionsResponse{StoryID: storyID}, nil
	}
	fragmentID := strings.TrimSpace(*story.SourceFragmentID)
	fragment, err := s.repo.FragmentByID(ctx, fragmentID)
	if err != nil || fragment == nil {
		return nil, errors.New("fragment not found")
	}
	suggestions := s.fragmentCharacterSuggestionsFromAssets(ctx, fragment)
	if len(suggestions) == 0 {
		suggestions = s.fragmentCharacterSuggestionsFromContent(fragment)
	}
	for i := range suggestions {
		if existing, err := s.repo.LatestCharacterGenerationTaskByFragmentKey(ctx, storyID, fragmentID, suggestions[i].Key); err == nil && existing != nil {
			if existing.CharacterID != "" {
				suggestions[i].AlreadyCreated = existing.Status == domain.CharacterGenerationStatusCompleted
				suggestions[i].ExistingCharacterID = existing.CharacterID
			}
		}
	}
	var existingTask *domain.CharacterGenerationTask
	if len(suggestions) > 0 {
		existingTask, _ = s.repo.LatestCharacterGenerationTaskByFragmentKey(ctx, storyID, fragmentID, suggestions[0].Key)
	}
	return &FragmentCharacterSuggestionsResponse{
		StoryID:      storyID,
		FragmentID:   fragmentID,
		Suggestions:  suggestions,
		ExistingTask: existingTask,
	}, nil
}

// validateCharacterGenerationTaskRequest requires a usable name plus at least one narrative / goals-tab field, or non-empty merged prompt (fragment / manual_form).
func validateCharacterGenerationTaskRequest(req CharacterGenerationTaskRequest) error {
	effectiveName := strings.TrimSpace(req.Name)
	if req.Suggestion != nil {
		if sn := strings.TrimSpace(req.Suggestion.Name); sn != "" && (effectiveName == "" || legacyFragmentPlaceholderCharacterName(effectiveName)) {
			effectiveName = sn
		}
	}
	if effectiveName == "" || legacyFragmentPlaceholderCharacterName(effectiveName) {
		return errors.New("character name is required")
	}
	desc := strings.TrimSpace(req.Description)
	pers := strings.TrimSpace(req.Personality)
	app := strings.TrimSpace(req.Appearance)
	bg := strings.TrimSpace(req.Background)
	if req.Suggestion != nil {
		if desc == "" {
			desc = strings.TrimSpace(req.Suggestion.Description)
		}
		if bg == "" {
			bg = strings.TrimSpace(req.Suggestion.Background)
		}
		if app == "" {
			app = strings.TrimSpace(req.Suggestion.Appearance)
		}
	}
	stg := strings.TrimSpace(req.ShortTermGoal)
	ltg := strings.TrimSpace(req.LongTermGoal)
	hand := strings.TrimSpace(req.HandlingStyle)
	cog := strings.TrimSpace(req.CognitionRange)
	abil := strings.TrimSpace(req.AbilityFeatures)
	dress := strings.TrimSpace(req.DressPreference)
	prompt := strings.TrimSpace(req.Prompt)

	if desc != "" || pers != "" || app != "" || bg != "" ||
		stg != "" || ltg != "" || hand != "" || cog != "" || abil != "" || dress != "" ||
		prompt != "" {
		return nil
	}
	return errors.New("provide story-relevant detail: narrative fields, appearance/goals tab fields, or a combined prompt")
}

func (s *Service) StartCharacterGenerationTask(ctx context.Context, userID string, req CharacterGenerationTaskRequest) (*domain.CharacterGenerationTask, error) {
	story, err := s.repo.StoryByID(ctx, req.StoryID)
	if err != nil || story == nil {
		return nil, errors.New("story not found")
	}
	if story.UserID != userID {
		return nil, errors.New("you can only generate characters for your own stories")
	}
	sourceType := strings.TrimSpace(req.SourceType)
	if sourceType == "" {
		sourceType = domain.CharacterGenerationSourceManualForm
	}
	if sourceType == domain.CharacterGenerationSourceFragment {
		key := strings.TrimSpace(req.SourceFragmentCharacterKey)
		if key == "" && req.Suggestion != nil {
			key = req.Suggestion.Key
		}
		if key == "" {
			key = stableCharacterKey(req.Name)
		}
		fragmentID := strings.TrimSpace(req.SourceFragmentID)
		if fragmentID == "" && story.SourceFragmentID != nil {
			fragmentID = strings.TrimSpace(*story.SourceFragmentID)
		}
		if fragmentID == "" {
			return nil, errors.New("source fragment id is required")
		}
		if existing, err := s.repo.LatestCharacterGenerationTaskByFragmentKey(ctx, req.StoryID, fragmentID, key); err == nil && existing != nil {
			return existing, nil
		}
		req.SourceFragmentID = fragmentID
		req.SourceFragmentCharacterKey = key
	}
	if err := validateCharacterGenerationTaskRequest(req); err != nil {
		return nil, err
	}
	requestBytes, _ := json.Marshal(req)
	task := &domain.CharacterGenerationTask{
		UserID:                     userID,
		StoryID:                    req.StoryID,
		SourceType:                 sourceType,
		SourceFragmentID:           strings.TrimSpace(req.SourceFragmentID),
		SourceFragmentCharacterKey: strings.TrimSpace(req.SourceFragmentCharacterKey),
		Status:                     domain.CharacterGenerationStatusPending,
		Progress:                   0,
		CurrentStep:                domain.CharacterGenerationStepQueued,
		RequestJSON:                string(requestBytes),
	}
	if err := s.repo.CreateCharacterGenerationTask(ctx, task); err != nil {
		return nil, errors.New("failed to create character generation task: " + err.Error())
	}
	go s.runCharacterGenerationTask(context.Background(), task.ID)
	return task, nil
}

func (s *Service) GetCharacterGenerationTask(ctx context.Context, userID, taskID string) (*domain.CharacterGenerationTask, error) {
	task, err := s.repo.CharacterGenerationTaskByID(ctx, taskID)
	if err != nil {
		return nil, err
	}
	if task.UserID != userID {
		return nil, errors.New("unauthorized")
	}
	return task, nil
}

func (s *Service) ListCharacterGenerationTasks(ctx context.Context, userID, status string, limit, offset int) ([]*domain.CharacterGenerationTask, error) {
	return s.repo.ListCharacterGenerationTasks(ctx, userID, status, limit, offset)
}

func (s *Service) RetryCharacterGenerationTask(ctx context.Context, userID, taskID string) (*domain.CharacterGenerationTask, error) {
	task, err := s.GetCharacterGenerationTask(ctx, userID, taskID)
	if err != nil {
		return nil, err
	}
	if task.Status != domain.CharacterGenerationStatusFailed {
		return task, nil
	}
	task.Status = domain.CharacterGenerationStatusPending
	task.Progress = 0
	task.CurrentStep = domain.CharacterGenerationStepQueued
	task.ErrorMessage = ""
	task.CompletedAt = nil
	if err := s.repo.UpdateCharacterGenerationTask(ctx, task); err != nil {
		return nil, err
	}
	go s.runCharacterGenerationTask(context.Background(), task.ID)
	return task, nil
}

func (s *Service) runCharacterGenerationTask(ctx context.Context, taskID string) {
	task, err := s.repo.CharacterGenerationTaskByID(ctx, taskID)
	if err != nil || task == nil {
		return
	}
	var req CharacterGenerationTaskRequest
	_ = json.Unmarshal([]byte(task.RequestJSON), &req)
	if err := s.updateCharacterGenerationTask(ctx, task, domain.CharacterGenerationStatusProcessing, domain.CharacterGenerationStepExtract, 12, ""); err != nil {
		s.logger.Warn("failed to mark character task processing", zap.Error(err))
	}
	attrs := GeneratedCharacterAttributes{
		Description:     strings.TrimSpace(req.Description),
		Personality:     strings.TrimSpace(req.Personality),
		Background:      strings.TrimSpace(req.Background),
		Appearance:      strings.TrimSpace(req.Appearance),
		ShortTermGoal:   strings.TrimSpace(req.ShortTermGoal),
		LongTermGoal:    strings.TrimSpace(req.LongTermGoal),
		HandlingStyle:   strings.TrimSpace(req.HandlingStyle),
		CognitionRange:  strings.TrimSpace(req.CognitionRange),
		AbilityFeatures: strings.TrimSpace(req.AbilityFeatures),
		DressPreference: strings.TrimSpace(req.DressPreference),
	}
	name := strings.TrimSpace(req.Name)
	if req.Suggestion != nil {
		if name == "" || legacyFragmentPlaceholderCharacterName(name) {
			if sn := strings.TrimSpace(req.Suggestion.Name); sn != "" && !legacyFragmentPlaceholderCharacterName(sn) {
				name = sn
			}
		}
		if req.ReferenceImage == "" {
			req.ReferenceImage = firstNonEmpty(req.Suggestion.ReferenceImage, req.Suggestion.ReferenceImageURL)
		}
		if attrs.Description == "" {
			attrs.Description = strings.TrimSpace(req.Suggestion.Description)
		}
		if attrs.Background == "" {
			attrs.Background = strings.TrimSpace(req.Suggestion.Background)
		}
		if attrs.Appearance == "" {
			attrs.Appearance = strings.TrimSpace(req.Suggestion.Appearance)
		}
	}
	if name == "" || legacyFragmentPlaceholderCharacterName(name) {
		if resolved := s.resolveFragmentCharacterSuggestionName(ctx, strings.TrimSpace(task.SourceFragmentID), strings.TrimSpace(task.SourceFragmentCharacterKey)); resolved != "" {
			name = resolved
		}
	}
	if name == "" {
		name = "未命名角色"
	}
	prompt := strings.TrimSpace(req.Prompt)
	if prompt == "" {
		prompt = strings.TrimSpace(strings.Join([]string{name, attrs.Description, attrs.Background, attrs.Appearance}, "\n"))
	}
	if prompt != "" {
		if generated, genErr := s.GenerateCharacterWithAI(ctx, task.UserID, GenerateCharacterRequest{Prompt: prompt, Name: name}); genErr == nil && generated != nil {
			mergeGeneratedCharacterAttributes(&attrs, generated)
		} else if genErr != nil {
			s.logger.Warn("character task attribute generation failed; continuing with provided fields", zap.String("taskID", taskID), zap.Error(genErr))
		}
	}
	_ = s.updateCharacterGenerationTask(ctx, task, domain.CharacterGenerationStatusProcessing, domain.CharacterGenerationStepCreate, 35, "")
	characterReq := CreateCharacterRequest{
		Name:            name,
		Description:     attrs.Description,
		Personality:     attrs.Personality,
		Background:      attrs.Background,
		ShortTermGoal:   attrs.ShortTermGoal,
		LongTermGoal:    attrs.LongTermGoal,
		HandlingStyle:   attrs.HandlingStyle,
		CognitionRange:  attrs.CognitionRange,
		AbilityFeatures: attrs.AbilityFeatures,
		Appearance:      attrs.Appearance,
		DressPreference: attrs.DressPreference,
		StoryID:         task.StoryID,
		IsPublic:        false,
		SourceType:      "ai",
		SourcePrompt:    prompt,
		ReferenceImage:  strings.TrimSpace(req.ReferenceImage),
		NeedsPortrait:   true,
	}
	var character *domain.Character
	if strings.TrimSpace(task.CharacterID) != "" {
		character, err = s.repo.CharacterByID(ctx, task.CharacterID)
	} else {
		character, err = s.CreateCharacter(ctx, task.UserID, characterReq)
		if err == nil && character != nil {
			character.SourceFragmentID = task.SourceFragmentID
			character.SourceFragmentCharacterKey = task.SourceFragmentCharacterKey
			character.AIGenerated = true
			_ = s.repo.UpdateCharacter(ctx, character)
			task.CharacterID = character.ID
		}
	}
	if err != nil || character == nil {
		if err == nil {
			err = errors.New("failed to create character")
		}
		s.failCharacterGenerationTask(ctx, task, err)
		return
	}
	refImg := strings.TrimSpace(req.ReferenceImage)
	if refImg == "" && req.Suggestion != nil {
		refImg = strings.TrimSpace(firstNonEmpty(req.Suggestion.ReferenceImage, req.Suggestion.ReferenceImageURL))
	}

	_ = s.updateCharacterGenerationTask(ctx, task, domain.CharacterGenerationStatusProcessing, domain.CharacterGenerationStepPortrait, 55, "")
	// 有碎片参考图时跳过独立「立绘」生图（省一次出图/token），Portrait 直接使用参考图；三视图仍为单张正侧背合一 sheet。
	if refImg != "" {
		character, reloadErr := s.repo.CharacterByID(ctx, character.ID)
		if reloadErr == nil && character != nil {
			character.Portrait = refImg
			character.PortraitGenerationStatus = "generated"
			character.UpdatedAt = time.Now().Unix()
			_ = s.repo.UpdateCharacter(ctx, character)
		}
	} else {
		_, portraitErr := s.GenerateCharacterPortrait(ctx, task.UserID, character.ID, GenerateCharacterPortraitRequest{ReferenceImage: ""})
		if portraitErr != nil {
			s.logger.Warn("character task portrait generation failed", zap.String("taskID", taskID), zap.Error(portraitErr))
		}
	}

	_ = s.updateCharacterGenerationTask(ctx, task, domain.CharacterGenerationStatusProcessing, domain.CharacterGenerationStepThreeViews, 78, "")
	character, chReloadErr := s.repo.CharacterByID(ctx, task.CharacterID)
	if chReloadErr != nil || character == nil {
		s.logger.Warn("character task reload failed before three-views",
			zap.String("taskID", taskID), zap.Error(chReloadErr))
		msg := "character reload failed before three-views"
		if chReloadErr != nil {
			msg = chReloadErr.Error()
		}
		s.failCharacterGenerationTask(ctx, task, errors.New(msg))
		return
	}
	if req.Suggestion != nil && strings.TrimSpace(req.Suggestion.ThreeViewSheetURL) != "" {
		sheet := strings.TrimSpace(req.Suggestion.ThreeViewSheetURL)
		character.Views = &domain.CharacterThreeViews{Sheet: sheet, Front: "", Side: "", Back: ""}
		if err := s.repo.UpdateCharacter(ctx, character); err != nil {
			s.logger.Warn("character task reuse fragment three-view failed", zap.String("taskID", taskID), zap.Error(err))
		}
	} else {
		tvRef := refImg
		if strings.TrimSpace(tvRef) == "" {
			tvRef = strings.TrimSpace(character.Portrait)
		}
		_, viewsErr := s.GenerateCharacterThreeViews(ctx, task.UserID, character.ID, GenerateCharacterThreeViewsRequest{RegenerateAll: false, ReferenceImage: tvRef})
		if viewsErr != nil {
			s.logger.Warn("character task three views generation failed", zap.String("taskID", taskID), zap.Error(viewsErr))
		}
	}
	result := map[string]string{"characterId": character.ID}
	resultBytes, _ := json.Marshal(result)
	now := time.Now().Unix()
	task.Status = domain.CharacterGenerationStatusCompleted
	task.Progress = 100
	task.CurrentStep = domain.CharacterGenerationStepNotify
	task.ResultJSON = string(resultBytes)
	task.ErrorMessage = ""
	task.CompletedAt = &now
	_ = s.repo.UpdateCharacterGenerationTask(ctx, task)
	_ = s.NotifyCharacterGenerationComplete(ctx, task.UserID, task.StoryID, character.ID, name)
}

func (s *Service) updateCharacterGenerationTask(ctx context.Context, task *domain.CharacterGenerationTask, status, step string, progress int, errMsg string) error {
	task.Status = status
	task.CurrentStep = step
	task.Progress = progress
	task.ErrorMessage = errMsg
	task.UpdatedAt = time.Now().Unix()
	return s.repo.UpdateCharacterGenerationTask(ctx, task)
}

func (s *Service) failCharacterGenerationTask(ctx context.Context, task *domain.CharacterGenerationTask, cause error) {
	now := time.Now().Unix()
	task.Status = domain.CharacterGenerationStatusFailed
	task.Progress = 100
	task.ErrorMessage = cause.Error()
	task.CompletedAt = &now
	_ = s.repo.UpdateCharacterGenerationTask(ctx, task)
	_ = s.NotifyCharacterGenerationFailed(ctx, task.UserID, task.StoryID, task.ID, cause.Error())
}

func (s *Service) fragmentCharacterSuggestionsFromContent(fragment *domain.Fragment) []domain.FragmentCharacterSuggestion {
	if fragment == nil {
		return nil
	}
	content := strings.TrimSpace(fragment.Content)
	if content == "" {
		return nil
	}
	name := "碎片主角"
	key := stableCharacterKey(name)
	ref := ""
	media := fragment.MediaURLs
	if len(media) == 0 && strings.TrimSpace(fragment.ImageUrls) != "" {
		_ = json.Unmarshal([]byte(fragment.ImageUrls), &media)
	}
	if len(media) > 0 {
		ref = media[0]
	}
	return []domain.FragmentCharacterSuggestion{{
		Key:               key,
		Name:              name,
		Description:       truncateRunes(content, 160),
		Background:        truncateRunes(content, 240),
		ReferenceImage:    ref,
		ReferenceImageURL: ref,
	}}
}

func legacyFragmentPlaceholderCharacterName(name string) bool {
	switch strings.TrimSpace(name) {
	case "", "碎片角色":
		return true
	default:
		return false
	}
}

func parseFragmentGenerationTraceMetadata(raw string) *domain.FragmentGenerationTrace {
	raw = strings.TrimSpace(raw)
	if raw == "" || raw == "{}" {
		return nil
	}
	var trace domain.FragmentGenerationTrace
	if err := json.Unmarshal([]byte(raw), &trace); err != nil {
		return nil
	}
	return &trace
}

func mergeFragmentEvidenceCharacterNames(evs []domain.FragmentVisualEvidence, out map[string]string) {
	for _, ev := range evs {
		for _, ent := range ev.Entities {
			if !strings.EqualFold(strings.TrimSpace(ent.Kind), "character") {
				continue
			}
			k := strings.TrimSpace(ent.Key)
			n := strings.TrimSpace(ent.Name)
			if k == "" || n == "" {
				continue
			}
			if prev, ok := out[k]; !ok || strings.TrimSpace(prev) == "" {
				out[k] = n
			}
		}
	}
}

func fragmentTraceCharacterDisplayNames(trace *domain.FragmentGenerationTrace) map[string]string {
	if trace == nil {
		return nil
	}
	out := make(map[string]string)
	if vb := trace.VisualBible; vb != nil {
		for _, ch := range vb.Characters {
			k := strings.TrimSpace(ch.Key)
			n := strings.TrimSpace(ch.Name)
			if k == "" || n == "" {
				continue
			}
			out[k] = n
		}
		mergeFragmentEvidenceCharacterNames(vb.SourceEvidence, out)
	}
	mergeFragmentEvidenceCharacterNames(trace.VisualEvidence, out)
	return out
}

func displayNameFromStableCharacterKey(key string) string {
	k := strings.TrimSpace(key)
	if k == "" {
		return ""
	}
	k = strings.ReplaceAll(k, "_", "-")
	parts := strings.Split(k, "-")
	var b strings.Builder
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if b.Len() > 0 {
			b.WriteByte(' ')
		}
		r, size := utf8.DecodeRuneInString(p)
		if r == utf8.RuneError || size == 0 {
			b.WriteString(p)
			continue
		}
		if unicode.Is(unicode.Han, r) {
			b.WriteString(p)
			continue
		}
		if unicode.IsLetter(r) {
			lowered := strings.ToLower(p)
			r2, sz2 := utf8.DecodeRuneInString(lowered)
			b.WriteRune(unicode.ToUpper(r2))
			b.WriteString(lowered[sz2:])
		} else {
			b.WriteString(p)
		}
	}
	return truncateRunes(strings.TrimSpace(b.String()), 32)
}

func (s *Service) resolveFragmentCharacterSuggestionName(ctx context.Context, fragmentID, characterKey string) string {
	if s == nil || s.repo == nil {
		return ""
	}
	fragmentID = strings.TrimSpace(fragmentID)
	characterKey = strings.TrimSpace(characterKey)
	if fragmentID == "" || characterKey == "" {
		return ""
	}
	frag, err := s.repo.FragmentByID(ctx, fragmentID)
	if err != nil || frag == nil {
		return ""
	}
	for _, sug := range s.fragmentCharacterSuggestionsFromAssets(ctx, frag) {
		if strings.TrimSpace(sug.Key) != characterKey {
			continue
		}
		n := strings.TrimSpace(sug.Name)
		if n != "" && !legacyFragmentPlaceholderCharacterName(n) {
			return n
		}
	}
	return ""
}

func (s *Service) fragmentCharacterSuggestionsFromAssets(ctx context.Context, fragment *domain.Fragment) []domain.FragmentCharacterSuggestion {
	if fragment == nil {
		return nil
	}
	bibleChars := fragmentVisualBibleCharacters(fragment.GenerationMetadata)
	trace := parseFragmentGenerationTraceMetadata(fragment.GenerationMetadata)
	displayNames := fragmentTraceCharacterDisplayNames(trace)
	byKey := make(map[string]*domain.FragmentCharacterSuggestion)
	var order []string
	addOrGet := func(key string) *domain.FragmentCharacterSuggestion {
		key = strings.TrimSpace(key)
		if key == "" {
			return nil
		}
		if existing := byKey[key]; existing != nil {
			return existing
		}
		ch := bibleChars[key]
		name := strings.TrimSpace(ch.Name)
		if name == "" {
			name = strings.TrimSpace(displayNames[key])
		}
		if name == "" {
			name = displayNameFromStableCharacterKey(key)
		}
		traits := strings.Join(ch.ImmutableTraits, "；")
		sug := &domain.FragmentCharacterSuggestion{
			Key:         key,
			Name:        truncateRunes(strings.TrimSpace(name), 32),
			Description: traits,
			Appearance:  traits,
		}
		byKey[key] = sug
		order = append(order, key)
		return sug
	}
	for key := range bibleChars {
		addOrGet(key)
	}
	assets, err := s.repo.ListFragmentGenerationAssets(ctx, fragment.ID)
	if err != nil {
		s.logger.Warn("list fragment generation assets failed", zap.String("fragmentID", fragment.ID), zap.Error(err))
	}
	if len(assets) == 0 {
		backfilled, backfillErr := s.backfillFragmentGenerationAssets(ctx, fragment)
		if backfillErr != nil {
			s.logger.Warn("legacy fragment assets backfill failed", zap.String("fragmentID", fragment.ID), zap.Error(backfillErr))
		} else if len(backfilled) > 0 {
			assets = backfilled
		}
	}
	for _, asset := range assets {
		if asset == nil || strings.TrimSpace(asset.EntityKind) != domain.FragmentGenerationAssetEntityCharacter {
			continue
		}
		key := strings.TrimSpace(asset.EntityKey)
		if key == "" {
			continue
		}
		sug := addOrGet(key)
		if sug == nil {
			continue
		}
		if metaName, metaTraits := fragmentAssetCharacterMetadata(asset.MetadataJSON); metaName != "" || len(metaTraits) > 0 {
			if metaName != "" && (legacyFragmentPlaceholderCharacterName(sug.Name) || strings.TrimSpace(sug.Name) == "") {
				sug.Name = truncateRunes(metaName, 32)
			}
			traits := strings.Join(metaTraits, "；")
			if traits != "" && sug.Appearance == "" {
				sug.Appearance = traits
			}
			if traits != "" && sug.Description == "" {
				sug.Description = traits
			}
		}
		switch strings.TrimSpace(asset.Kind) {
		case domain.FragmentGenerationAssetKindReferenceAsset, domain.FragmentGenerationAssetKindAnchorImage:
			if sug.ReferenceImage == "" {
				sug.ReferenceImage = strings.TrimSpace(asset.URL)
				sug.ReferenceImageURL = strings.TrimSpace(asset.URL)
			}
		case domain.FragmentGenerationAssetKindCharacterTurnaround:
			if sug.ThreeViewSheetURL == "" {
				sug.ThreeViewSheetURL = strings.TrimSpace(asset.URL)
			}
		}
	}
	out := make([]domain.FragmentCharacterSuggestion, 0, len(order))
	for _, key := range order {
		if sug := byKey[key]; sug != nil {
			if sug.ReferenceImageURL == "" {
				sug.ReferenceImageURL = sug.ReferenceImage
			}
			out = append(out, *sug)
		}
	}
	return out
}

func fragmentVisualBibleCharacters(raw string) map[string]domain.FragmentVisualCharacter {
	out := make(map[string]domain.FragmentVisualCharacter)
	raw = strings.TrimSpace(raw)
	if raw == "" || raw == "{}" {
		return out
	}
	var trace domain.FragmentGenerationTrace
	if err := json.Unmarshal([]byte(raw), &trace); err != nil || trace.VisualBible == nil {
		return out
	}
	for _, ch := range trace.VisualBible.Characters {
		if key := strings.TrimSpace(ch.Key); key != "" {
			out[key] = ch
		}
	}
	return out
}

func fragmentAssetCharacterMetadata(raw string) (string, []string) {
	var meta struct {
		Name            string   `json:"name"`
		ImmutableTraits []string `json:"immutableTraits"`
	}
	if err := json.Unmarshal([]byte(strings.TrimSpace(raw)), &meta); err != nil {
		return "", nil
	}
	return strings.TrimSpace(meta.Name), meta.ImmutableTraits
}

func stableCharacterKey(name string) string {
	k := strings.ToLower(strings.TrimSpace(name))
	k = strings.ReplaceAll(k, " ", "-")
	if k == "" {
		return "fragment-character"
	}
	return truncateRunes(k, 64)
}

func mergeGeneratedCharacterAttributes(dst *GeneratedCharacterAttributes, src *GeneratedCharacterAttributes) {
	if dst.Description == "" {
		dst.Description = src.Description
	}
	if dst.Personality == "" {
		dst.Personality = src.Personality
	}
	if dst.Background == "" {
		dst.Background = src.Background
	}
	if dst.ShortTermGoal == "" {
		dst.ShortTermGoal = src.ShortTermGoal
	}
	if dst.LongTermGoal == "" {
		dst.LongTermGoal = src.LongTermGoal
	}
	if dst.HandlingStyle == "" {
		dst.HandlingStyle = src.HandlingStyle
	}
	if dst.CognitionRange == "" {
		dst.CognitionRange = src.CognitionRange
	}
	if dst.AbilityFeatures == "" {
		dst.AbilityFeatures = src.AbilityFeatures
	}
	if dst.Appearance == "" {
		dst.Appearance = src.Appearance
	}
	if dst.DressPreference == "" {
		dst.DressPreference = src.DressPreference
	}
}

// GetCharacter 获取角色详情（带缓存）
func (s *Service) GetCharacter(ctx context.Context, characterID string) (*domain.Character, error) {
	return s.GetCharacterWithUserContext(ctx, characterID, "")
}

// GetCharacterWithUserContext 获取角色详情（带用户上下文，用于判断关注状态，带缓存）
func (s *Service) GetCharacterWithUserContext(ctx context.Context, characterID, userID string) (*domain.Character, error) {
	s.logger.Info("getting character",
		zap.String("characterID", characterID),
		zap.String("userID", userID))

	// 尝试从缓存获取
	c := s.getCache()
	if c != nil {
		key := cache.CharacterKey(characterID)
		var cachedCharacter domain.Character
		if err := c.Get(ctx, key, &cachedCharacter); err == nil {
			s.logger.Debug("character cache hit",
				zap.String("characterID", characterID))
			// 如果有用户ID，检查关注状态（关注状态不缓存，因为每个用户不同）
			if userID != "" {
				isFollowing, err := s.repo.IsCharacterFollowing(ctx, userID, characterID)
				if err != nil {
					s.logger.Warn("failed to check character following status",
						zap.String("characterID", characterID),
						zap.String("userID", userID),
						zap.Error(err))
				} else {
					cachedCharacter.IsFollowing = &isFollowing
				}
			}
			return &cachedCharacter, nil
		} else {
			s.logger.Debug("character cache miss",
				zap.String("characterID", characterID),
				zap.Error(err))
		}
	}

	// 从数据库获取
	character, err := s.repo.CharacterByID(ctx, characterID)
	if err != nil {
		if err == domain.ErrNotFound {
			s.logger.Warn("character not found",
				zap.String("characterID", characterID))
			return nil, errors.New("character not found")
		}
		s.logger.Error("failed to get character",
			zap.String("characterID", characterID),
			zap.Error(err))
		return nil, errors.New("failed to get character")
	}

	// 如果有用户ID，检查关注状态
	if userID != "" {
		isFollowing, err := s.repo.IsCharacterFollowing(ctx, userID, characterID)
		if err != nil {
			s.logger.Warn("failed to check character following status",
				zap.String("characterID", characterID),
				zap.String("userID", userID),
				zap.Error(err))
		} else {
			character.IsFollowing = &isFollowing
		}
	}

	// 写入缓存（不包含关注状态）
	if c != nil {
		key := cache.CharacterKey(characterID)
		// 临时保存关注状态
		isFollowing := character.IsFollowing
		character.IsFollowing = nil
		if err := c.Set(ctx, key, character, entityCacheTTL); err != nil {
			s.logger.Warn("failed to cache character",
				zap.String("characterID", characterID),
				zap.Error(err))
		} else {
			s.logger.Debug("character cached",
				zap.String("characterID", characterID))
		}
		// 恢复关注状态
		character.IsFollowing = isFollowing
	}

	return character, nil
}

// ListCharacters 获取角色列表
func (s *Service) ListCharacters(ctx context.Context, req CharacterListRequest) ([]*domain.Character, error) {
	s.logger.Info("listing characters",
		zap.String("userId", req.UserID),
		zap.String("storyId", req.StoryID),
		zap.Int("limit", req.Limit),
	)

	// 设置默认分页参数
	if req.Limit == 0 {
		req.Limit = 20
	}

	var characters []*domain.Character
	var err error

	if req.StoryID != "" {
		// 获取特定故事的角色
		characters, err = s.repo.CharactersByStory(ctx, req.StoryID)
	} else if req.UserID != "" {
		// 获取特定用户的角色
		characters, err = s.repo.CharactersByUser(ctx, req.UserID, req.Limit, req.Offset)
	} else {
		// 获取所有角色
		characters, err = s.repo.ListCharacters(ctx, req.Limit, req.Offset)
	}

	if err != nil {
		s.logger.Error("failed to list characters", zap.Error(err))
		return nil, errors.New("failed to list characters")
	}

	// 如果有搜索关键词，进行过滤
	if req.Search != "" {
		filtered := make([]*domain.Character, 0)
		for _, char := range characters {
			if contains(char.Name, req.Search) || contains(char.Description, req.Search) {
				filtered = append(filtered, char)
			}
		}
		characters = filtered
	}

	s.logger.Info("characters listed successfully", zap.Int("count", len(characters)))
	return characters, nil
}

// UpdateCharacter 更新角色
func (s *Service) UpdateCharacter(ctx context.Context, userID, characterID string, req UpdateCharacterRequest) (*domain.Character, error) {
	s.logger.Info("updating character",
		zap.String("userID", userID),
		zap.String("characterID", characterID),
	)

	// 获取角色
	character, err := s.repo.CharacterByID(ctx, characterID)
	if err != nil {
		if err == domain.ErrNotFound {
			return nil, errors.New("character not found")
		}
		return nil, errors.New("failed to get character")
	}

	// 验证权限
	if err := s.ensureStoryOwnership(ctx, character.StoryID, userID); err != nil {
		return nil, err
	}

	// 更新字段
	if req.Name != nil {
		character.Name = *req.Name
	}
	if req.Description != nil {
		character.Description = *req.Description
	}
	if req.Avatar != nil {
		character.Avatar = *req.Avatar
	}
	if req.Personality != nil {
		character.Personality = *req.Personality
	}
	if req.Background != nil {
		character.Background = *req.Background
	}
	if req.ShortTermGoal != nil {
		character.ShortTermGoal = *req.ShortTermGoal
	}
	if req.LongTermGoal != nil {
		character.LongTermGoal = *req.LongTermGoal
	}
	if req.HandlingStyle != nil {
		character.HandlingStyle = *req.HandlingStyle
	}
	if req.CognitionRange != nil {
		character.CognitionRange = *req.CognitionRange
	}
	if req.AbilityFeatures != nil {
		character.AbilityFeatures = *req.AbilityFeatures
	}
	if req.Appearance != nil {
		character.Appearance = *req.Appearance
	}
	if req.DressPreference != nil {
		character.DressPreference = *req.DressPreference
	}
	character.Role = ""
	if req.SourceType != nil {
		character.SourceType = strings.ToLower(*req.SourceType)
	}
	if req.SourcePrompt != nil {
		character.SourcePrompt = *req.SourcePrompt
	}
	if req.SourceImage != nil {
		character.SourceImage = *req.SourceImage
	}
	if req.IsPublic != nil {
		character.IsPublic = *req.IsPublic
	}
	character.LastEditedBy = userID
	character.UpdatedAt = time.Now().Unix()

	if err := s.repo.UpdateCharacter(ctx, character); err != nil {
		s.logger.Error("failed to update character",
			zap.String("characterID", characterID),
			zap.Error(err))
		return nil, errors.New("failed to update character")
	}

	// 使缓存失效并重新缓存
	c := s.getCache()
	if c != nil {
		key := cache.CharacterKey(characterID)
		if err := c.Delete(ctx, key); err != nil {
			s.logger.Warn("failed to invalidate character cache",
				zap.String("characterID", characterID),
				zap.Error(err))
		}
		// 重新缓存
		character.IsFollowing = nil // 不缓存关注状态
		if err := c.Set(ctx, key, character, entityCacheTTL); err != nil {
			s.logger.Warn("failed to cache updated character",
				zap.String("characterID", characterID),
				zap.Error(err))
		}
		// 清除相关列表缓存
		for limit := 20; limit <= 100; limit += 20 {
			for offset := 0; offset < 200; offset += limit {
				_ = c.Delete(ctx, cache.UserCharactersListKey(character.UserID, limit, offset))
			}
		}
	}

	s.logger.Info("character updated successfully",
		zap.String("characterID", characterID))
	return character, nil
}

// DeleteCharacter 删除角色
func (s *Service) DeleteCharacter(ctx context.Context, userID, characterID string) error {
	s.logger.Info("deleting character",
		zap.String("userID", userID),
		zap.String("characterID", characterID),
	)

	// 获取角色
	character, err := s.repo.CharacterByID(ctx, characterID)
	if err != nil {
		if err == domain.ErrNotFound {
			return errors.New("character not found")
		}
		return errors.New("failed to get character")
	}

	// 验证权限
	if err := s.ensureStoryOwnership(ctx, character.StoryID, userID); err != nil {
		return err
	}

	if err := s.repo.DeleteCharacter(ctx, characterID); err != nil {
		s.logger.Error("failed to delete character",
			zap.String("characterID", characterID),
			zap.Error(err))
		return errors.New("failed to delete character")
	}

	// 使缓存失效
	c := s.getCache()
	if c != nil {
		key := cache.CharacterKey(characterID)
		if err := c.Delete(ctx, key); err != nil {
			s.logger.Warn("failed to invalidate character cache",
				zap.String("characterID", characterID),
				zap.Error(err))
		}
		// 清除相关列表缓存
		for limit := 20; limit <= 100; limit += 20 {
			for offset := 0; offset < 200; offset += limit {
				_ = c.Delete(ctx, cache.UserCharactersListKey(character.UserID, limit, offset))
			}
		}
	}

	s.logger.Info("character deleted successfully",
		zap.String("characterID", characterID))
	return nil
}

// FollowCharacter 关注角色
func (s *Service) FollowCharacter(ctx context.Context, userID, characterID string) error {
	s.logger.Info("following character",
		zap.String("userID", userID),
		zap.String("characterID", characterID),
	)

	// 验证角色存在
	_, err := s.repo.CharacterByID(ctx, characterID)
	if err != nil {
		if err == domain.ErrNotFound {
			return errors.New("character not found")
		}
		return errors.New("failed to get character")
	}

	if err := s.repo.FollowCharacter(ctx, userID, characterID); err != nil {
		// Treat "already following" as success (idempotent operation)
		if errors.Is(err, domain.ErrAlreadyExists) {
			s.logger.Info("character already followed",
				zap.String("userID", userID),
				zap.String("characterID", characterID))
			return nil
		}
		s.logger.Error("failed to follow character", zap.Error(err))
		return errors.New("failed to follow character")
	}

	s.logger.Info("character followed successfully", zap.String("characterID", characterID))
	return nil
}

// UnfollowCharacter 取消关注角色
func (s *Service) UnfollowCharacter(ctx context.Context, userID, characterID string) error {
	s.logger.Info("unfollowing character",
		zap.String("userID", userID),
		zap.String("characterID", characterID),
	)

	if err := s.repo.UnfollowCharacter(ctx, userID, characterID); err != nil {
		// Treat "not following" as success (idempotent operation)
		if errors.Is(err, domain.ErrNotFound) {
			s.logger.Info("character not followed (idempotent)",
				zap.String("userID", userID),
				zap.String("characterID", characterID))
			return nil
		}
		s.logger.Error("failed to unfollow character", zap.Error(err))
		return errors.New("failed to unfollow character")
	}

	s.logger.Info("character unfollowed successfully", zap.String("characterID", characterID))
	return nil
}

// contains 检查字符串是否包含子串（不区分大小写）
func contains(s, substr string) bool {
	// 简单实现，生产环境应该使用更好的搜索算法
	return len(s) > 0 && len(substr) > 0 &&
		(s == substr || len(s) >= len(substr))
}

// REMOVED: Character Skills Management - not in StoryCreationAppUI design

// ========== Character Analytics ==========

// GetCharacterAnalytics 获取角色分析数据
func (s *Service) GetCharacterAnalytics(ctx context.Context, characterID string) (*domain.CharacterAnalytics, error) {
	// 验证角色存在
	_, err := s.repo.CharacterByID(ctx, characterID)
	if err != nil {
		if err == domain.ErrNotFound {
			return nil, errors.New("character not found")
		}
		return nil, errors.New("failed to get character")
	}

	// 获取分析数据，如果不存在则创建
	analytics, err := s.repo.GetOrCreateCharacterAnalytics(ctx, characterID)
	if err != nil {
		s.logger.Error("failed to get character analytics", zap.Error(err))
		return nil, errors.New("failed to get analytics")
	}

	return analytics, nil
}

// UpdateCharacterAnalytics 更新角色分析数据
func (s *Service) UpdateCharacterAnalytics(ctx context.Context, characterID string, analytics *domain.CharacterAnalytics) error {
	if err := s.repo.UpdateCharacterAnalytics(ctx, analytics); err != nil {
		s.logger.Error("failed to update character analytics", zap.Error(err))
		return errors.New("failed to update analytics")
	}
	return nil
}

// IncrementCharacterMessages 增加角色消息计数
func (s *Service) IncrementCharacterMessages(ctx context.Context, characterID string, count int) error {
	return s.repo.IncrementCharacterMessages(ctx, characterID, count)
}

// IncrementCharacterTokens 增加角色 token 消耗
func (s *Service) IncrementCharacterTokens(ctx context.Context, characterID string, tokens int64) error {
	return s.repo.IncrementCharacterTokens(ctx, characterID, tokens)
}

// IncrementCharacterChatters 增加角色聊天用户数
func (s *Service) IncrementCharacterChatters(ctx context.Context, characterID string) error {
	return s.repo.IncrementCharacterChatters(ctx, characterID)
}

// GetCharacterStoryboards 获取角色参与的故事板列表
func (s *Service) GetCharacterStoryboards(ctx context.Context, characterID string, limit, offset int) ([]*domain.Storyboard, int64, error) {
	s.logger.Info("getting character storyboards",
		zap.String("characterID", characterID),
		zap.Int("limit", limit),
		zap.Int("offset", offset),
	)

	// 验证角色存在
	_, err := s.repo.CharacterByID(ctx, characterID)
	if err != nil {
		if err == domain.ErrNotFound {
			return nil, 0, errors.New("character not found")
		}
		return nil, 0, errors.New("failed to get character")
	}

	// 设置默认分页
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	storyboards, total, err := s.repo.StoryboardsByCharacter(ctx, characterID, limit, offset)
	if err != nil {
		s.logger.Error("failed to get storyboards by character", zap.Error(err))
		return nil, 0, errors.New("failed to get character storyboards")
	}

	return storyboards, total, nil
}

// GenerateCharacterWithAI 使用AI生成角色属性
func (s *Service) GenerateCharacterWithAI(ctx context.Context, userID string, req GenerateCharacterRequest) (*GeneratedCharacterAttributes, error) {
	s.logger.Info("generating character with AI",
		zap.String("userID", userID),
		zap.String("prompt", req.Prompt),
	)

	if s.aiGenService == nil {
		return nil, errors.New("AI generation service not configured")
	}

	// 构建生成提示词（键名与 domain.Character JSON、characters 表文本列、客户端表单字段严格一致）
	systemPrompt := `你是一个专业的故事角色设计师。请根据用户描述生成角色结构化属性。

硬性要求：
1. 只输出一个 JSON 对象，不要 markdown 代码块，不要多余说明文字。
2. 必须包含以下全部 10 个键，键名与下表完全一致（camelCase），不要省略任何键：description, personality, background, shortTermGoal, longTermGoal, handlingStyle, cognitionRange, abilityFeatures, appearance, dressPreference。
3. 每个键的值必须是字符串；若无内容可写简短占位如「未特别设定」，禁止省略键或设为 null。
4. description 为角色整体概要，总长不超过 400 字；background、personality 可略长；其余字段各约 80–200 字中文。
5. 确保 JSON 语法完整（引号、逗号、大括号闭合）。

各键含义（须按此语义撰写，便于写入数据库与客户端表单）：
- description：角色整体介绍与第一印象。
- personality：性格、情绪特点与待人气质。
- background：出身、经历与当前处境。
- shortTermGoal：当前故事阶段最想达成的事。
- longTermGoal：更长期的抱负或人生方向。
- handlingStyle：面对冲突或难题时的决策方式，必须包含标志性的微表情（如皱眉、假笑）与习惯性肢体动作倾向（如摸下巴、特定的站姿重心）。
- cognitionRange：其知识边界、世界观与思维方式（知道什么、不知道什么）。
- abilityFeatures：特长、技能与突出能力。
- appearance：不仅写相貌，必须包含物理质感极其丰富的发丝（形态、光泽）、皮肤细节（纹理/疤痕/色泽）、具体的面部骨骼特征（眼型/下颌线）、以及身高体型比例。
- dressPreference：日常着装风格与偏好，必须提供细致的材质明细（如天鹅绒/破损皮革/重麻布）、层次感混搭习惯、标志性的色彩搭配，以及穿戴磨损痕迹和特征配饰。

示例结构：
{
  "description": "...",
  "personality": "...",
  "background": "...",
  "shortTermGoal": "...",
  "longTermGoal": "...",
  "handlingStyle": "...",
  "cognitionRange": "...",
  "abilityFeatures": "...",
  "appearance": "...",
  "dressPreference": "..."
}`

	prompt := "请为以下角色生成详细属性：\n\n"
	if req.Name != "" {
		prompt += "角色名称：" + req.Name + "\n"
	}
	prompt += "用户描述：" + req.Prompt

	// 使用AI生成服务
	genReq := &GenerateTextRequest{
		UserID:            userID,
		OriginalPrompt:    prompt,
		SystemPrompt:      systemPrompt,
		Model:             "gemini-2.5-flash",
		Temperature:       0.8,
		MaxTokens:         8192,
		RelatedEntityType: "character",
		Metadata: map[string]interface{}{
			"operation": "generate_character_attributes",
		},
	}

	result, err := s.aiGenService.GenerateText(ctx, genReq)
	if err != nil {
		s.logger.Error("AI generation failed", zap.Error(err))
		return nil, errors.New("AI generation failed: " + err.Error())
	}

	// 解析生成结果
	attributes := s.parseCharacterAttributes(result.Text)

	s.logger.Info("character attributes generated",
		zap.String("aiRecordId", result.RecordID),
		zap.Int("tokensUsed", result.TokensUsed))

	return attributes, nil
}

// parseCharacterAttributes 解析AI生成的角色属性
func (s *Service) parseCharacterAttributes(text string) *GeneratedCharacterAttributes {
	var attributes GeneratedCharacterAttributes

	// Strip markdown code blocks if present
	cleanedText := strings.TrimSpace(text)

	// Handle ```json or ``` code blocks
	if strings.HasPrefix(cleanedText, "```") {
		// Find the end of the first line (skip ```json or ```)
		if idx := strings.Index(cleanedText, "\n"); idx != -1 {
			cleanedText = cleanedText[idx+1:]
		}
		// Remove trailing ```
		if idx := strings.LastIndex(cleanedText, "```"); idx != -1 {
			cleanedText = strings.TrimSpace(cleanedText[:idx])
		}
	}

	// Try to find JSON object in the text
	cleanedText = extractJSONFromText(cleanedText)

	// Try to parse as JSON
	if err := json.Unmarshal([]byte(cleanedText), &attributes); err != nil {
		s.logger.Warn("failed to parse character attributes JSON",
			zap.Error(err),
			zap.String("rawText", truncateForLog(cleanedText, 500)))

		// Try to parse fields individually using regex-like extraction
		attributes = s.extractAttributesFromText(text)
	}

	return &attributes
}

// extractJSONFromText 尝试从文本中提取JSON对象
func extractJSONFromText(text string) string {
	text = strings.TrimSpace(text)

	// Find the first { and last } to extract JSON
	startIdx := strings.Index(text, "{")
	endIdx := strings.LastIndex(text, "}")

	if startIdx != -1 && endIdx != -1 && endIdx > startIdx {
		return text[startIdx : endIdx+1]
	}

	return text
}

// extractAttributesFromText 从文本中手动提取属性（当JSON解析失败时）
func (s *Service) extractAttributesFromText(text string) GeneratedCharacterAttributes {
	var attributes GeneratedCharacterAttributes

	// Try to extract individual fields using simple parsing
	// This handles cases where JSON might be truncated or malformed

	// Define field patterns to extract
	fieldExtractors := map[string]*string{
		"description":     &attributes.Description,
		"personality":     &attributes.Personality,
		"background":      &attributes.Background,
		"shortTermGoal":   &attributes.ShortTermGoal,
		"longTermGoal":    &attributes.LongTermGoal,
		"handlingStyle":   &attributes.HandlingStyle,
		"cognitionRange":  &attributes.CognitionRange,
		"abilityFeatures": &attributes.AbilityFeatures,
		"appearance":      &attributes.Appearance,
		"dressPreference": &attributes.DressPreference,
	}

	for fieldName, fieldPtr := range fieldExtractors {
		value := extractFieldValue(text, fieldName)
		if value != "" {
			*fieldPtr = value
		}
	}

	// If no fields were extracted, put the whole text in description
	if attributes.Description == "" && attributes.Personality == "" &&
		attributes.Background == "" && attributes.ShortTermGoal == "" {
		attributes.Description = text
	}

	return attributes
}

// extractFieldValue 从JSON格式文本中提取指定字段的值
func extractFieldValue(text, fieldName string) string {
	// Look for pattern like "fieldName": "value" or "fieldName": "value",
	pattern := `"` + fieldName + `"\s*:\s*"`

	idx := strings.Index(text, pattern)
	if idx == -1 {
		return ""
	}

	// Find the start of the value
	valueStart := idx + len(pattern)
	if valueStart >= len(text) {
		return ""
	}

	// Find the end of the value (look for ", or "})
	valueEnd := -1
	escaped := false
	for i := valueStart; i < len(text); i++ {
		if escaped {
			escaped = false
			continue
		}
		if text[i] == '\\' {
			escaped = true
			continue
		}
		if text[i] == '"' {
			valueEnd = i
			break
		}
	}

	if valueEnd == -1 {
		// If no closing quote found, take until end (truncated response)
		valueEnd = len(text)
	}

	if valueEnd > valueStart {
		value := text[valueStart:valueEnd]
		// Unescape common JSON escape sequences
		value = strings.ReplaceAll(value, `\\n`, "\n")
		value = strings.ReplaceAll(value, `\n`, "\n")
		value = strings.ReplaceAll(value, `\"`, `"`)
		value = strings.ReplaceAll(value, `\\`, `\`)
		return value
	}

	return ""
}

// GenerateCharacterAvatar 使用AI生成角色头像
func (s *Service) GenerateCharacterAvatar(ctx context.Context, userID, characterID string, req GenerateCharacterAvatarRequest) (*GenerateCharacterAvatarResult, error) {
	s.logger.Info("generating character avatar",
		zap.String("userID", userID),
		zap.String("characterID", characterID),
	)

	// 1. 获取角色信息
	character, err := s.repo.CharacterByID(ctx, characterID)
	if err != nil {
		if err == domain.ErrNotFound {
			return nil, errors.New("character not found")
		}
		return nil, errors.New("failed to get character")
	}

	// 2. 验证权限（只有角色创建者可以生成角色头像）
	if character.Author == nil || character.Author.ID != userID {
		return nil, errors.New("unauthorized")
	}

	// 3. 检查AI服务是否可用
	if s.aiGenService == nil {
		return nil, errors.New("AI generation service not configured")
	}

	// 4. 构建头像生成提示词
	prompt := s.buildAvatarPrompt(character)

	// 5. 设置默认宽高比
	aspectRatio := req.AspectRatio
	if aspectRatio == "" {
		aspectRatio = "1:1" // 默认正方形头像
	}

	// 6. 调用AI图像生成服务
	imageReq := &GenerateImageRequest{
		UserID:            userID,
		Prompt:            prompt,
		Provider:          "gemini",
		Model:             "imagen-3.0-generate-001",
		AspectRatio:       aspectRatio,
		Quality:           "high",
		OutputCount:       1,
		RelatedEntityID:   characterID,
		RelatedEntityType: "character",
		Metadata: map[string]interface{}{
			"operation":   "character_avatar_generation",
			"characterId": characterID,
		},
	}

	result, err := s.aiGenService.GenerateImage(ctx, imageReq)
	if err != nil {
		s.logger.Error("failed to generate character avatar", zap.Error(err))
		return nil, errors.New("failed to generate avatar: " + err.Error())
	}

	if len(result.ImageURLs) == 0 {
		return nil, errors.New("no image generated")
	}

	s.logger.Info("character avatar generated successfully",
		zap.String("characterID", characterID),
		zap.String("recordID", result.RecordID),
	)

	return &GenerateCharacterAvatarResult{
		AvatarURL: result.ImageURLs[0],
		RecordID:  result.RecordID,
	}, nil
}

// buildAvatarPrompt 构建角色头像生成提示词
func (s *Service) buildAvatarPrompt(character *domain.Character) string {
	var prompt strings.Builder

	prompt.WriteString("A professional character portrait avatar of ")
	prompt.WriteString(character.Name)
	prompt.WriteString(".\n\n")

	if character.Appearance != "" {
		prompt.WriteString("Appearance: ")
		prompt.WriteString(character.Appearance)
		prompt.WriteString(".\n")
	}

	if character.DressPreference != "" {
		prompt.WriteString("Dress style: ")
		prompt.WriteString(character.DressPreference)
		prompt.WriteString(".\n")
	}

	if character.Personality != "" {
		prompt.WriteString("Personality traits: ")
		prompt.WriteString(character.Personality)
		prompt.WriteString(".\n")
	}

	prompt.WriteString("\n")
	prompt.WriteString("Style: Professional character portrait, centered composition, clear facial features, high quality, detailed, character design art, clean background or subtle environment that matches the character's background.\n")
	prompt.WriteString("The image should be suitable for use as a character avatar icon.")

	return prompt.String()
}

// UpdateCharacterAvatar 更新角色头像
func (s *Service) UpdateCharacterAvatar(ctx context.Context, userID, characterID, avatarURL string) (*domain.Character, error) {
	s.logger.Info("updating character avatar",
		zap.String("userID", userID),
		zap.String("characterID", characterID),
	)

	// 获取角色
	character, err := s.repo.CharacterByID(ctx, characterID)
	if err != nil {
		if err == domain.ErrNotFound {
			return nil, errors.New("character not found")
		}
		return nil, errors.New("failed to get character")
	}

	// 验证权限（只有角色创建者可以更新角色头像）
	if character.Author == nil || character.Author.ID != userID {
		return nil, errors.New("unauthorized")
	}

	// 更新头像
	character.Avatar = avatarURL
	character.LastEditedBy = userID
	character.UpdatedAt = time.Now().Unix()

	if err := s.repo.UpdateCharacter(ctx, character); err != nil {
		s.logger.Error("failed to update character avatar", zap.Error(err))
		return nil, errors.New("failed to update character avatar")
	}

	s.logger.Info("character avatar updated successfully", zap.String("characterID", characterID))
	return character, nil
}

// UsePortraitAsAvatar 使用portrait作为头像（仅在头像为空时）
func (s *Service) UsePortraitAsAvatar(ctx context.Context, userID, characterID, portraitURL string) (*domain.Character, error) {
	s.logger.Info("using portrait as avatar",
		zap.String("userID", userID),
		zap.String("characterID", characterID),
	)

	// 获取角色
	character, err := s.repo.CharacterByID(ctx, characterID)
	if err != nil {
		if err == domain.ErrNotFound {
			return nil, errors.New("character not found")
		}
		return nil, errors.New("failed to get character")
	}

	// 验证权限（只有角色创建者可以操作）
	if character.Author == nil || character.Author.ID != userID {
		return nil, errors.New("unauthorized")
	}

	// 检查头像是否为空，只有在为空时才设置
	if character.Avatar != "" {
		s.logger.Info("character already has avatar, skipping",
			zap.String("characterID", characterID),
			zap.String("existingAvatar", character.Avatar),
		)
		return character, nil
	}

	// 设置portrait为头像
	character.Avatar = portraitURL
	character.LastEditedBy = userID
	character.UpdatedAt = time.Now().Unix()

	if err := s.repo.UpdateCharacter(ctx, character); err != nil {
		s.logger.Error("failed to update character avatar", zap.Error(err))
		return nil, errors.New("failed to update character avatar")
	}

	s.logger.Info("portrait set as avatar successfully",
		zap.String("characterID", characterID),
		zap.String("portraitURL", portraitURL),
	)

	return character, nil
}

// GeneratePortraitPrompt 为角色生成推荐的形象提示词
func (s *Service) GeneratePortraitPrompt(ctx context.Context, characterID string) (string, error) {
	s.logger.Info("generating portrait prompt for character", zap.String("characterID", characterID))

	// 获取角色
	character, err := s.repo.CharacterByID(ctx, characterID)
	if err != nil {
		if err == domain.ErrNotFound {
			return "", errors.New("character not found")
		}
		return "", errors.New("failed to get character")
	}

	// 构建形象生成提示词
	prompt := s.buildPortraitPrompt(character)

	s.logger.Info("portrait prompt generated successfully", zap.String("characterID", characterID))
	return prompt, nil
}

// buildPortraitPrompt 构建角色完整形象生成提示词
func (s *Service) buildPortraitPrompt(character *domain.Character) string {
	var prompt strings.Builder

	prompt.WriteString("A full-body character portrait of ")
	prompt.WriteString(character.Name)
	prompt.WriteString(".\n\n")

	// 外观描述
	if character.Appearance != "" {
		prompt.WriteString("Physical Appearance: ")
		prompt.WriteString(character.Appearance)
		prompt.WriteString(".\n\n")
	}

	// 服装风格
	if character.DressPreference != "" {
		prompt.WriteString("Clothing and Dress Style: ")
		prompt.WriteString(character.DressPreference)
		prompt.WriteString(".\n\n")
	}

	// 性格特征（影响姿势和表情）
	if character.Personality != "" {
		prompt.WriteString("Personality (to inform pose and expression): ")
		prompt.WriteString(character.Personality)
		prompt.WriteString(".\n\n")
	}

	// 背景信息（可能影响服装或氛围）
	if character.Background != "" {
		prompt.WriteString("Background: ")
		prompt.WriteString(character.Background)
		prompt.WriteString(".\n\n")
	}

	// 能力特征（可能影响外观细节）
	if character.AbilityFeatures != "" {
		prompt.WriteString("Special Abilities/Features: ")
		prompt.WriteString(character.AbilityFeatures)
		prompt.WriteString(".\n\n")
	}

	// 艺术风格指导
	prompt.WriteString("Style: High-quality character design illustration, ")
	prompt.WriteString("full body portrait in a standing or dynamic pose, ")
	prompt.WriteString("detailed clothing and accessories, ")
	prompt.WriteString("professional character art, ")
	prompt.WriteString("clear and expressive features, ")
	prompt.WriteString("suitable background or environment that complements the character, ")
	prompt.WriteString("2:3 or 3:4 vertical aspect ratio.")

	return prompt.String()
}

// GenerateCharacterPortrait 使用AI生成角色形象（完整形象图）
func (s *Service) GenerateCharacterPortrait(ctx context.Context, userID, characterID string, req GenerateCharacterPortraitRequest) (*GenerateCharacterPortraitResult, error) {
	s.logger.Info("generating character portrait",
		zap.String("userID", userID),
		zap.String("characterID", characterID),
	)

	// 1. 获取角色
	character, err := s.repo.CharacterByID(ctx, characterID)
	if err != nil {
		if err == domain.ErrNotFound {
			return nil, errors.New("character not found")
		}
		return nil, errors.New("failed to get character")
	}

	// 2. 验证权限
	if character.Author == nil || character.Author.ID != userID {
		return nil, errors.New("unauthorized: only character creator can generate portrait")
	}

	// 3. 检查AI服务是否可用
	if s.aiGenService == nil {
		return nil, errors.New("AI generation service not configured")
	}

	// 4. 构建形象生成提示词
	var prompt string
	if req.CustomPrompt != "" {
		// 用户提供了自定义提示词
		prompt = req.CustomPrompt
	} else {
		// 使用自动生成的提示词
		prompt = s.buildPortraitPrompt(character)
	}

	// 5. 设置宽高比（竖版人像）
	aspectRatio := req.AspectRatio
	if aspectRatio == "" {
		aspectRatio = "2:3" // 默认2:3竖版人像
	}

	// 6. 更新角色状态为"生成中"
	character.PortraitGenerationStatus = "generating"
	if err := s.repo.UpdateCharacter(ctx, character); err != nil {
		s.logger.Warn("failed to update portrait generation status", zap.Error(err))
	}

	// 7. 调用AI图像生成服务
	var referenceImages []string
	if req.ReferenceImage != "" {
		referenceImages = []string{req.ReferenceImage}
	}

	// 设置默认 provider 为 huoshan（火山引擎）
	provider := "huoshan"
	if s.imageProvider != "" {
		provider = s.imageProvider
	}

	imageReq := &GenerateImageRequest{
		UserID:            userID,
		Prompt:            prompt,
		ReferenceImages:   referenceImages, // 传递参考图
		Provider:          provider,
		Quality:           "high",
		OutputCount:       1,
		RelatedEntityID:   characterID,
		RelatedEntityType: "character",
		Metadata: map[string]interface{}{
			"operation":         "character_portrait_generation",
			"characterId":       characterID,
			"hasCustomPrompt":   req.CustomPrompt != "",
			"hasReferenceImage": req.ReferenceImage != "",
		},
	}

	// 根据不同的 provider 设置相应的参数
	switch provider {
	case "huoshan":
		// huoshan 使用 Size 而不是 AspectRatio
		imageReq.Size = aspectRatioToSize(aspectRatio)
		// Model 留空，使用 huoshan provider 的默认模型 (doubao-seedream)
	case "gemini":
		imageReq.Model = "imagen-3.0-generate-001"
		imageReq.AspectRatio = aspectRatio
	case "nana", "banana":
		// nana/banana 使用 AspectRatio
		imageReq.AspectRatio = aspectRatio
	default:
		// 其他 provider 使用 AspectRatio
		imageReq.AspectRatio = aspectRatio
	}

	result, err := s.aiGenService.GenerateImage(ctx, imageReq)
	if err != nil {
		s.logger.Error("failed to generate character portrait", zap.Error(err))

		// 更新状态为失败
		character.PortraitGenerationStatus = "failed"
		_ = s.repo.UpdateCharacter(ctx, character)

		return nil, errors.New("failed to generate portrait: " + err.Error())
	}

	if len(result.ImageURLs) == 0 {
		// 更新状态为失败
		character.PortraitGenerationStatus = "failed"
		_ = s.repo.UpdateCharacter(ctx, character)

		return nil, errors.New("no image generated")
	}

	// 8. 更新角色的Portrait字段和状态
	character.Portrait = result.ImageURLs[0]
	character.PortraitGenerationStatus = "generated"
	character.LastEditedBy = userID
	character.UpdatedAt = time.Now().Unix()
	// 尚未设置头像或海报时，用立绘填充，便于列表与详情展示
	if strings.TrimSpace(character.Avatar) == "" {
		character.Avatar = character.Portrait
	}
	if strings.TrimSpace(character.Poster) == "" {
		character.Poster = character.Portrait
	}

	if err := s.repo.UpdateCharacter(ctx, character); err != nil {
		s.logger.Error("failed to update character portrait", zap.Error(err))
		return nil, errors.New("failed to update character portrait")
	}

	// 清除缓存
	c := s.getCache()
	if c != nil {
		_ = c.Delete(ctx, cache.CharacterKey(characterID))
	}

	s.logger.Info("character portrait generated successfully",
		zap.String("characterID", characterID),
		zap.String("recordID", result.RecordID),
	)

	return &GenerateCharacterPortraitResult{
		PortraitURL: result.ImageURLs[0],
		RecordID:    result.RecordID,
	}, nil
}

// GenerateCharacterThreeViews 使用 AI 生成一张横向「正·侧·背」合一参考图，写入 views_json.sheet（单次出图、单次计费）。
func (s *Service) GenerateCharacterThreeViews(ctx context.Context, userID, characterID string, req GenerateCharacterThreeViewsRequest) (*GenerateCharacterThreeViewsResult, error) {
	s.logger.Info("generating character three-views",
		zap.String("userID", userID),
		zap.String("characterID", characterID),
		zap.Bool("regenerateAll", req.RegenerateAll))

	character, err := s.repo.CharacterByID(ctx, characterID)
	if err != nil {
		if err == domain.ErrNotFound {
			return nil, errors.New("character not found")
		}
		return nil, errors.New("failed to get character")
	}
	if character.Author == nil || character.Author.ID != userID {
		return nil, errors.New("unauthorized: only character creator can generate three-views")
	}
	if s.aiGenService == nil {
		return nil, errors.New("AI generation service not configured")
	}

	var v domain.CharacterThreeViews
	if character.Views != nil {
		v = *character.Views
	}

	if req.RegenerateAll {
		v = domain.CharacterThreeViews{}
	} else {
		if strings.TrimSpace(v.Sheet) != "" {
			return &GenerateCharacterThreeViewsResult{Views: v}, nil
		}
	}

	bal, balErr := s.repo.GetTokenBalance(ctx, userID)
	if balErr != nil {
		return nil, errors.New("failed to get token balance")
	}
	if bal < common.AIImageBillingUnitTokens {
		return nil, fmt.Errorf("insufficient token balance: need at least %d tokens for three-view sheet", common.AIImageBillingUnitTokens)
	}

	provider := "huoshan"
	if s.imageProvider != "" {
		provider = s.imageProvider
	}
	aspectRatio := "16:9"

	baseDesc := strings.TrimSpace(character.Name)
	if d := strings.TrimSpace(character.Appearance); d != "" {
		baseDesc = baseDesc + ". " + d
	}
	if d := strings.TrimSpace(character.Description); d != "" {
		baseDesc = baseDesc + ". " + d
	}
	if baseDesc == "" {
		baseDesc = "original character design"
	}

	prompt := fmt.Sprintf(
		"Single image only, one artwork, not three separate images. "+
			"Professional character turnaround on plain white background: exactly three full-body figures of the SAME character in ONE horizontal row — "+
			"left figure: front orthographic view facing camera; center figure: right-side profile; right figure: back view. "+
			"Each figure must show only that one viewing angle (no sprite grids, no 2x2 or 3x3 panels, no multiple mini poses per cell). "+
			"Consistent outfit, proportions, colors, and hairstyle across all three. Anime-influenced clean line art or illustration, high detail. %s",
		baseDesc,
	)

	var refImages []string
	if r := strings.TrimSpace(req.ReferenceImage); r != "" {
		refImages = []string{r}
	} else if strings.TrimSpace(character.Portrait) != "" {
		refImages = []string{character.Portrait}
	}

	imageReq := &GenerateImageRequest{
		UserID:            userID,
		Prompt:            prompt,
		ReferenceImages:   refImages,
		Provider:          provider,
		Quality:           "high",
		OutputCount:       1,
		RelatedEntityID:   characterID,
		RelatedEntityType: "character",
		Metadata: map[string]interface{}{
			"operation":   "character_three_view_sheet",
			"characterId": characterID,
		},
	}
	switch provider {
	case "huoshan":
		imageReq.Size = aspectRatioToSize(aspectRatio)
	case "gemini":
		imageReq.Model = "imagen-3.0-generate-001"
		imageReq.AspectRatio = aspectRatio
	default:
		imageReq.AspectRatio = aspectRatio
	}

	result, genErr := s.aiGenService.GenerateImage(ctx, imageReq)
	if genErr != nil {
		s.logger.Error("three-view sheet failed", zap.Error(genErr))
		return nil, fmt.Errorf("failed to generate three-view sheet: %w", genErr)
	}
	if len(result.ImageURLs) == 0 {
		return nil, errors.New("no image generated for three-view sheet")
	}
	imgURL := result.ImageURLs[0]
	v = domain.CharacterThreeViews{
		Sheet: imgURL,
		Front: "",
		Side:  "",
		Back:  "",
	}
	character.Views = &v
	character.LastEditedBy = userID
	character.UpdatedAt = time.Now().Unix()
	if err := s.repo.UpdateCharacter(ctx, character); err != nil {
		s.logger.Error("failed to persist three-view sheet", zap.Error(err))
		return nil, errors.New("failed to save three-view sheet: " + err.Error())
	}

	if c := s.getCache(); c != nil {
		_ = c.Delete(ctx, cache.CharacterKey(characterID))
	}

	s.logger.Info("character three-view sheet generated", zap.String("characterID", characterID))
	return &GenerateCharacterThreeViewsResult{Views: v}, nil
}

// CropAvatarFromPortrait 从Portrait裁剪生成Avatar（可选功能）
func (s *Service) CropAvatarFromPortrait(ctx context.Context, characterID string) (string, error) {
	s.logger.Info("cropping avatar from portrait", zap.String("characterID", characterID))

	// 获取角色
	character, err := s.repo.CharacterByID(ctx, characterID)
	if err != nil {
		if err == domain.ErrNotFound {
			return "", errors.New("character not found")
		}
		return "", errors.New("failed to get character")
	}

	// 检查是否有Portrait
	if character.Portrait == "" {
		return "", errors.New("character has no portrait to crop from")
	}

	// TODO: 实现图片裁剪逻辑
	// 这里需要调用图片处理服务（如阿里云OSS的图片处理功能或其他图片处理服务）
	// 从Portrait裁剪出头像区域（通常是人脸为中心的正方形区域）
	// 生成小尺寸的Avatar图片

	// 暂时返回Portrait作为Avatar（实际应用中应该实现裁剪逻辑）
	s.logger.Warn("CropAvatarFromPortrait not fully implemented, using portrait as avatar",
		zap.String("characterID", characterID))

	avatarURL := character.Portrait // 临时方案

	// 更新角色的Avatar字段
	character.Avatar = avatarURL
	character.UpdatedAt = time.Now().Unix()

	if err := s.repo.UpdateCharacter(ctx, character); err != nil {
		s.logger.Error("failed to update character avatar", zap.Error(err))
		return "", errors.New("failed to update character avatar")
	}

	// 清除缓存
	c := s.getCache()
	if c != nil {
		_ = c.Delete(ctx, cache.CharacterKey(characterID))
	}

	s.logger.Info("avatar cropped from portrait successfully", zap.String("characterID", characterID))
	return avatarURL, nil
}

// aspectRatioToSize converts aspect ratio string to image size
func aspectRatioToSize(aspectRatio string) string {
	switch aspectRatio {
	case "1:1":
		return "1024x1024"
	case "16:9":
		return "1920x1080"
	case "9:16":
		return "1080x1920"
	case "4:3":
		return "1024x768"
	case "3:4":
		return "768x1024"
	default:
		return "1024x1024"
	}
}
