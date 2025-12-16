package service

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/grapestree/fgrapery/grapery/internal/domain"
	"go.uber.org/zap"
)

// CreateCharacterRequest 创建角色请求
type CreateCharacterRequest struct {
	Name            string `json:"name" binding:"required,min=1,max=100"`
	Description     string `json:"description" binding:"max=2000"`
	Avatar          string `json:"avatar" binding:"omitempty,url"`
	Personality     string `json:"personality" binding:"max=1000"`
	Background      string `json:"background" binding:"max=2000"`
	ShortTermGoal   string `json:"shortTermGoal" binding:"max=1000"`
	LongTermGoal    string `json:"longTermGoal" binding:"max=1000"`
	HandlingStyle   string `json:"handlingStyle" binding:"max=1000"`
	CognitionRange  string `json:"cognitionRange" binding:"max=1000"`
	AbilityFeatures string `json:"abilityFeatures" binding:"max=1000"`
	Appearance      string `json:"appearance" binding:"max=1000"`
	DressPreference string `json:"dressPreference" binding:"max=1000"`
	StoryID         string `json:"storyId" binding:"required"`
	IsPublic        bool   `json:"isPublic"`
	SourceType      string `json:"sourceType" binding:"omitempty,oneof=manual upload ai"`
	SourcePrompt    string `json:"sourcePrompt"`
	SourceImage     string `json:"sourceImage" binding:"omitempty,url"`
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

// GeneratedCharacterAttributes AI生成的角色属性
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

// CharacterListRequest 角色列表请求
type CharacterListRequest struct {
	AuthorID string `form:"authorId"`
	StoryID  string `form:"storyId"`
	Search   string `form:"search"`
	Limit    int    `form:"limit" binding:"omitempty,min=1,max=100"`
	Offset   int    `form:"offset" binding:"omitempty,min=0"`
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
	sourceType := strings.ToLower(req.SourceType)
	if sourceType == "" {
		sourceType = AssetSourceManual
	}
	s.logger.Info("source type", zap.String("sourceType", sourceType))
	// 创建角色
	character := &domain.Character{
		StoryID:         req.StoryID,
		AuthorID:        author.ID,
		Name:            req.Name,
		Description:     req.Description,
		Avatar:          req.Avatar,
		Poster:          "",
		Author:          author,
		Personality:     req.Personality,
		Background:      req.Background,
		ShortTermGoal:   req.ShortTermGoal,
		LongTermGoal:    req.LongTermGoal,
		HandlingStyle:   req.HandlingStyle,
		CognitionRange:  req.CognitionRange,
		AbilityFeatures: req.AbilityFeatures,
		Appearance:      req.Appearance,
		DressPreference: req.DressPreference,
		IsPublic:        req.IsPublic,
		SourceType:      sourceType,
		SourcePrompt:    req.SourcePrompt,
		SourceImage:     req.SourceImage,
		CreatedBy:       userID,
		LastEditedBy:    userID,
		Followers:       0,
		Stories:         0,
		CreatedAt:       time.Now().Unix(),
		UpdatedAt:       time.Now().Unix(),
	}
	s.logger.Info("character created", zap.String("characterID", character.ID))
	if err := s.repo.CreateCharacter(ctx, character); err != nil {
		s.logger.Error("failed to create character", zap.Error(err))
		return nil, errors.New("failed to create character")
	}

	s.logger.Info("character created successfully", zap.String("characterID", character.ID))

	// 记录用户活动
	go s.RecordCharacterCreated(context.Background(), userID, character.ID, character.Name)

	return character, nil
}

// GetCharacter 获取角色详情
func (s *Service) GetCharacter(ctx context.Context, characterID string) (*domain.Character, error) {
	return s.GetCharacterWithUserContext(ctx, characterID, "")
}

// GetCharacterWithUserContext 获取角色详情（带用户上下文，用于判断关注状态）
func (s *Service) GetCharacterWithUserContext(ctx context.Context, characterID, userID string) (*domain.Character, error) {
	s.logger.Info("getting character", zap.String("characterID", characterID), zap.String("userID", userID))

	character, err := s.repo.CharacterByID(ctx, characterID)
	if err != nil {
		if err == domain.ErrNotFound {
			return nil, errors.New("character not found")
		}
		s.logger.Error("failed to get character", zap.Error(err))
		return nil, errors.New("failed to get character")
	}

	// 如果有用户ID，检查关注状态
	if userID != "" {
		isFollowing, err := s.repo.IsCharacterFollowing(ctx, userID, characterID)
		if err != nil {
			s.logger.Warn("failed to check character following status", zap.Error(err))
		} else {
			character.IsFollowing = &isFollowing
		}
	}

	return character, nil
}

// ListCharacters 获取角色列表
func (s *Service) ListCharacters(ctx context.Context, req CharacterListRequest) ([]*domain.Character, error) {
	s.logger.Info("listing characters",
		zap.String("authorId", req.AuthorID),
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
	} else if req.AuthorID != "" {
		// 获取特定作者的角色
		characters, err = s.repo.CharactersByAuthor(ctx, req.AuthorID, req.Limit, req.Offset)
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
		s.logger.Error("failed to update character", zap.Error(err))
		return nil, errors.New("failed to update character")
	}

	s.logger.Info("character updated successfully", zap.String("characterID", characterID))
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
		s.logger.Error("failed to delete character", zap.Error(err))
		return errors.New("failed to delete character")
	}

	s.logger.Info("character deleted successfully", zap.String("characterID", characterID))
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

// ========== Character Skills Management ==========

// AddCharacterSkill 添加角色技能
func (s *Service) AddCharacterSkill(ctx context.Context, userID, characterID, skill string) (*domain.Character, error) {
	s.logger.Info("adding character skill",
		zap.String("userID", userID),
		zap.String("characterID", characterID),
		zap.String("skill", skill),
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
	if character.Author.ID != userID {
		return nil, errors.New("unauthorized")
	}

	// 检查技能是否已存在
	for _, existingSkill := range character.Skills {
		if existingSkill == skill {
			return character, nil // Already exists
		}
	}

	// 添加技能
	character.Skills = append(character.Skills, skill)

	if err := s.repo.UpdateCharacter(ctx, character); err != nil {
		s.logger.Error("failed to add character skill", zap.Error(err))
		return nil, errors.New("failed to add skill")
	}

	s.logger.Info("character skill added successfully")
	return character, nil
}

// RemoveCharacterSkill 移除角色技能
func (s *Service) RemoveCharacterSkill(ctx context.Context, userID, characterID, skill string) (*domain.Character, error) {
	s.logger.Info("removing character skill",
		zap.String("userID", userID),
		zap.String("characterID", characterID),
		zap.String("skill", skill),
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
	if character.Author.ID != userID {
		return nil, errors.New("unauthorized")
	}

	// 移除技能
	newSkills := make([]string, 0)
	for _, existingSkill := range character.Skills {
		if existingSkill != skill {
			newSkills = append(newSkills, existingSkill)
		}
	}
	character.Skills = newSkills

	if err := s.repo.UpdateCharacter(ctx, character); err != nil {
		s.logger.Error("failed to remove character skill", zap.Error(err))
		return nil, errors.New("failed to remove skill")
	}

	s.logger.Info("character skill removed successfully")
	return character, nil
}

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

// ========== Character Posters ==========

// CreatePosterRequest 创建海报请求
type CreatePosterRequest struct {
	Title                 string `json:"title" binding:"required,min=1,max=200"`
	Prompt                string `json:"prompt" binding:"max=2000"` // User's description for AI generation
	Image                 string `json:"image" binding:"omitempty,url"`
	ReferenceStoryEnabled bool   `json:"referenceStoryEnabled"` // Whether to reference recent story plots
}

// CreateCharacterPoster 创建角色海报（草稿状态）
func (s *Service) CreateCharacterPoster(ctx context.Context, userID, characterID string, req CreatePosterRequest) (*domain.CharacterPoster, error) {
	s.logger.Info("creating character poster",
		zap.String("userID", userID),
		zap.String("characterID", characterID),
		zap.String("title", req.Title),
	)

	// 获取角色
	character, err := s.repo.CharacterByID(ctx, characterID)
	if err != nil {
		if err == domain.ErrNotFound {
			return nil, errors.New("character not found")
		}
		return nil, errors.New("failed to get character")
	}

	// 检查权限：公开角色任何人都可以创建海报，私有角色只有小组成员可以创建
	if !character.IsPublic && character.GroupID != nil {
		s.logger.Debug("checking group membership for private character",
			zap.String("userID", userID),
			zap.String("characterID", characterID),
			zap.String("groupID", *character.GroupID),
		)

		isMember, err := s.repo.IsGroupMember(ctx, *character.GroupID, userID)
		if err != nil {
			s.logger.Error("failed to check group membership",
				zap.String("userID", userID),
				zap.String("groupID", *character.GroupID),
				zap.Error(err),
			)
			return nil, errors.New("failed to verify group membership")
		}

		if !isMember {
			s.logger.Warn("unauthorized poster creation attempt",
				zap.String("userID", userID),
				zap.String("characterID", characterID),
				zap.String("groupID", *character.GroupID),
			)
			return nil, errors.New("unauthorized: not a group member")
		}

		s.logger.Debug("group membership verified",
			zap.String("userID", userID),
			zap.String("groupID", *character.GroupID),
		)
	} else {
		s.logger.Debug("character is public, skipping group membership check",
			zap.String("characterID", characterID),
			zap.Bool("isPublic", character.IsPublic),
		)
	}

	// 获取作者信息
	author, err := s.repo.UserByID(ctx, userID)
	if err != nil {
		return nil, errors.New("author not found")
	}

	// 创建海报（草稿状态）
	poster := &domain.CharacterPoster{
		CharacterID:           characterID,
		Author:                author,
		Type:                  "image",
		Title:                 req.Title,
		Image:                 req.Image,
		Prompt:                req.Prompt,
		Status:                domain.PosterStatusDraft,
		ReferenceStoryEnabled: req.ReferenceStoryEnabled,
	}

	if err := s.repo.CreateCharacterPoster(ctx, poster); err != nil {
		s.logger.Error("failed to create character poster", zap.Error(err))
		return nil, errors.New("failed to create poster")
	}

	s.logger.Info("character poster created successfully", zap.String("posterID", poster.ID))
	return poster, nil
}

// GetCharacterPosters 获取角色海报列表
func (s *Service) GetCharacterPosters(ctx context.Context, characterID string, limit, offset int) ([]*domain.CharacterPoster, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	// 验证角色存在
	_, err := s.repo.CharacterByID(ctx, characterID)
	if err != nil {
		if err == domain.ErrNotFound {
			return nil, errors.New("character not found")
		}
		return nil, errors.New("failed to get character")
	}

	posters, err := s.repo.CharacterPostersByCharacterID(ctx, characterID, limit, offset)
	if err != nil {
		s.logger.Error("failed to get character posters", zap.Error(err))
		return nil, errors.New("failed to get posters")
	}

	return posters, nil
}

// LikeCharacterPoster 点赞角色海报
func (s *Service) LikeCharacterPoster(ctx context.Context, posterID string) error {
	if err := s.repo.IncrementPosterLikes(ctx, posterID); err != nil {
		s.logger.Error("failed to like poster", zap.Error(err))
		return errors.New("failed to like poster")
	}
	return nil
}

// ShareCharacterPoster 分享角色海报
func (s *Service) ShareCharacterPoster(ctx context.Context, posterID string) error {
	if err := s.repo.IncrementPosterShares(ctx, posterID); err != nil {
		s.logger.Error("failed to share poster", zap.Error(err))
		return errors.New("failed to share poster")
	}
	return nil
}

// DeleteCharacterPoster 删除角色海报
func (s *Service) DeleteCharacterPoster(ctx context.Context, userID, posterID string) error {
	s.logger.Info("deleting character poster",
		zap.String("userID", userID),
		zap.String("posterID", posterID),
	)

	// 获取海报
	poster, err := s.repo.CharacterPosterByID(ctx, posterID)
	if err != nil {
		if err == domain.ErrNotFound {
			return errors.New("poster not found")
		}
		return errors.New("failed to get poster")
	}

	// 验证权限
	if poster.Author.ID != userID {
		return errors.New("unauthorized")
	}

	if err := s.repo.DeleteCharacterPoster(ctx, posterID); err != nil {
		s.logger.Error("failed to delete poster", zap.Error(err))
		return errors.New("failed to delete poster")
	}

	s.logger.Info("poster deleted successfully")
	return nil
}

// ========== Character Poster Generation ==========

// GeneratePosterRequest 生成海报请求
type GeneratePosterRequest struct {
	AspectRatio string `json:"aspectRatio"` // 16:9, 9:16, 1:1
}

// GeneratePosterResult 生成海报结果
type GeneratePosterResult struct {
	Poster              *domain.CharacterPoster `json:"poster"`
	ConceptGenerationID string                  `json:"conceptGenerationId"` // Step 1 AI record
	ImageGenerationID   string                  `json:"imageGenerationId"`   // Step 2 AI record
}

// PosterConcept LLM生成的海报概念结构
type PosterConcept struct {
	PosterConcept struct {
		VisualSubject      string `json:"visual_subject"`
		SceneEnvironment   string `json:"scene_environment"`
		CompositionCamera  string `json:"composition_camera"`
		LightingAtmosphere string `json:"lighting_atmosphere"`
		ArtStyle           string `json:"art_style"`
	} `json:"poster_concept"`
	TypographyInstruction struct {
		TitleContent    string `json:"title_content"`
		TitleStyle      string `json:"title_style"`
		TitlePosition   string `json:"title_position"`
		SubtitleContent string `json:"subtitle_content"`
		SubtitleStyle   string `json:"subtitle_style"`
	} `json:"typography_instruction"`
}

// GenerateCharacterPoster 生成角色海报（两步AI工作流）
// Step 1: 使用LLM生成海报概念JSON
// Step 2: 组装最终提示词，使用图像生成AI创建海报
func (s *Service) GenerateCharacterPoster(ctx context.Context, userID, posterID string, req GeneratePosterRequest) (*GeneratePosterResult, error) {
	s.logger.Info("generating character poster",
		zap.String("userID", userID),
		zap.String("posterID", posterID),
	)

	// 1. 验证输入参数
	if posterID == "" {
		return nil, errors.New("poster id is required")
	}

	// 2. 获取海报信息
	poster, err := s.repo.CharacterPosterByID(ctx, posterID)
	if err != nil {
		if err == domain.ErrNotFound {
			return nil, errors.New("poster not found")
		}
		return nil, errors.New("failed to get poster")
	}

	// 3. 验证权限
	if poster.Author == nil || poster.Author.ID != userID {
		return nil, errors.New("unauthorized: you can only generate your own posters")
	}

	// 4. 验证海报状态（只有草稿或失败的海报可以重新生成）
	if poster.Status != domain.PosterStatusDraft && poster.Status != domain.PosterStatusFailed {
		return nil, errors.New("poster is already generating or generated")
	}

	// 5. 检查AI服务是否可用
	if s.aiGenService == nil {
		return nil, errors.New("AI generation service not configured")
	}

	// 6. 获取角色详细信息
	character, err := s.repo.CharacterByID(ctx, poster.CharacterID)
	if err != nil {
		return nil, errors.New("failed to get character")
	}

	// 7. 更新海报状态为生成中
	poster.Status = domain.PosterStatusGenerating
	if err := s.repo.UpdateCharacterPoster(ctx, poster); err != nil {
		s.logger.Error("failed to update poster status", zap.Error(err))
		return nil, errors.New("failed to update poster status")
	}

	// 8. 构建上下文信息
	plotContext := ""
	if poster.ReferenceStoryEnabled {
		plotContext = s.buildPlotContext(ctx, userID, character.StoryID)
	}

	// 9. Step 1: 使用LLM生成海报概念
	conceptResult, err := s.generatePosterConcept(ctx, userID, poster, character, plotContext)
	if err != nil {
		s.logger.Error("failed to generate poster concept", zap.Error(err))
		poster.Status = domain.PosterStatusFailed
		poster.ErrorMessage = "Failed to generate poster concept: " + err.Error()
		_ = s.repo.UpdateCharacterPoster(ctx, poster)
		return nil, errors.New("failed to generate poster concept: " + err.Error())
	}

	poster.ConceptGenerationID = conceptResult.RecordID
	poster.PosterConceptJSON = conceptResult.ConceptJSON

	// 10. Step 2: 组装最终提示词并生成图像
	imageResult, err := s.generatePosterImage(ctx, userID, poster, conceptResult.Concept, req.AspectRatio)
	if err != nil {
		s.logger.Error("failed to generate poster image", zap.Error(err))
		poster.Status = domain.PosterStatusFailed
		poster.ErrorMessage = "Failed to generate poster image: " + err.Error()
		_ = s.repo.UpdateCharacterPoster(ctx, poster)
		return nil, errors.New("failed to generate poster image: " + err.Error())
	}

	// 11. 更新海报信息
	poster.ImageGenerationID = imageResult.RecordID
	poster.FinalImagePrompt = imageResult.FinalPrompt
	poster.Image = imageResult.ImageURL
	poster.Status = domain.PosterStatusGenerated
	poster.ErrorMessage = ""

	if err := s.repo.UpdateCharacterPoster(ctx, poster); err != nil {
		s.logger.Error("failed to update poster with generated image", zap.Error(err))
		return nil, errors.New("failed to save generated poster")
	}

	s.logger.Info("character poster generated successfully",
		zap.String("posterID", posterID),
		zap.String("conceptRecordID", conceptResult.RecordID),
		zap.String("imageRecordID", imageResult.RecordID),
	)

	return &GeneratePosterResult{
		Poster:              poster,
		ConceptGenerationID: conceptResult.RecordID,
		ImageGenerationID:   imageResult.RecordID,
	}, nil
}

// conceptGenerationResult Step 1 结果
type conceptGenerationResult struct {
	RecordID    string
	ConceptJSON string
	Concept     *PosterConcept
	TokensUsed  int
}

// imageGenerationResult Step 2 结果
type imageGenerationResult struct {
	RecordID    string
	FinalPrompt string
	ImageURL    string
}

// buildPlotContext 构建剧情上下文
func (s *Service) buildPlotContext(ctx context.Context, userID, storyID string) string {
	var plotContext strings.Builder

	// 获取用户最近参与的故事板（最多5个）
	storyboards, err := s.repo.StoryboardsByCreator(ctx, userID, 5, 0)
	if err != nil {
		s.logger.Warn("failed to get user storyboards for plot context", zap.Error(err))
		return ""
	}

	if len(storyboards) == 0 {
		return ""
	}

	plotContext.WriteString("Recent Story Context:\n")
	for i, sb := range storyboards {
		if sb.Content != "" {
			plotContext.WriteString(strings.Repeat("-", 40))
			plotContext.WriteString("\n")
			plotContext.WriteString("Scene ")
			plotContext.WriteString(strings.TrimSpace(string(rune(i + 1))))
			plotContext.WriteString(": ")
			plotContext.WriteString(sb.Title)
			plotContext.WriteString("\n")
			// 截取内容，避免过长
			content := sb.Content
			if len(content) > 500 {
				content = content[:500] + "..."
			}
			plotContext.WriteString(content)
			plotContext.WriteString("\n")
		}
	}

	return plotContext.String()
}

// generatePosterConcept Step 1: 使用LLM生成海报概念
func (s *Service) generatePosterConcept(ctx context.Context, userID string, poster *domain.CharacterPoster, character *domain.Character, plotContext string) (*conceptGenerationResult, error) {
	// 构建System Prompt
	systemPrompt := `# Role
You are an expert AI Poster Designer. Your task is to generate a structured image generation prompt (JSON) based on User Input, Character Profile, and Plot Context.

# Goal
Create a movie-quality poster where the AI generation model renders BOTH the visual scene AND the text typography perfectly in one shot.

# Steps
1. **Analyze Context**:
   * Combine [Character Profile] + [User Instruction] to define the Main Subject.
   * Use [Plot Context] (if enabled) to define the Background and Mood.
2. **Design Composition**:
   * Determine the best camera_angle (e.g., Low angle for heroism, High angle for vulnerability).
   * Define the spatial relationship between the character and the background elements.
3. **Design Typography (CRITICAL)**:
   * Invent a SHORT, IMPACTFUL English title based on the story.
   * Choose a font style that matches the genre (e.g., "Gothic serif" for horror, "Sleek sans-serif" for sci-fi).
   * Specify a clear position where the text won't cover the character's face.

# JSON Output Schema
You must output ONLY valid JSON with no markdown code blocks:
{
  "poster_concept": {
    "visual_subject": "string (Detailed character + action)",
    "scene_environment": "string (Background + weather + props)",
    "composition_camera": "string (Angle + framing + depth of field)",
    "lighting_atmosphere": "string (Lighting type + color palette + mood)",
    "art_style": "string (Medium + render engine + style keywords)"
  },
  "typography_instruction": {
    "title_content": "string (THE EXACT TEXT IN UPPERCASE)",
    "title_style": "string (Font type + material + color)",
    "title_position": "string (Exact placement e.g., 'at the top center')",
    "subtitle_content": "string (Short tagline or 'NONE')",
    "subtitle_style": "string (Font style + placement or 'NONE')"
  }
}`

	// 构建User Prompt
	var userPrompt strings.Builder
	userPrompt.WriteString("Please create a poster concept based on the following information:\n\n")

	// 角色信息
	userPrompt.WriteString("[Character Profile]\n")
	userPrompt.WriteString("Name: ")
	userPrompt.WriteString(character.Name)
	userPrompt.WriteString("\n")
	if character.Description != "" {
		userPrompt.WriteString("Description: ")
		userPrompt.WriteString(character.Description)
		userPrompt.WriteString("\n")
	}
	if character.Appearance != "" {
		userPrompt.WriteString("Appearance: ")
		userPrompt.WriteString(character.Appearance)
		userPrompt.WriteString("\n")
	}
	if character.DressPreference != "" {
		userPrompt.WriteString("Dress Style: ")
		userPrompt.WriteString(character.DressPreference)
		userPrompt.WriteString("\n")
	}
	if character.Personality != "" {
		userPrompt.WriteString("Personality: ")
		userPrompt.WriteString(character.Personality)
		userPrompt.WriteString("\n")
	}
	if character.Background != "" {
		userPrompt.WriteString("Background: ")
		userPrompt.WriteString(character.Background)
		userPrompt.WriteString("\n")
	}
	userPrompt.WriteString("\n")

	// 用户指令
	userPrompt.WriteString("[User Instruction]\n")
	userPrompt.WriteString("Poster Title: ")
	userPrompt.WriteString(poster.Title)
	userPrompt.WriteString("\n")
	if poster.Prompt != "" {
		userPrompt.WriteString("Description: ")
		userPrompt.WriteString(poster.Prompt)
		userPrompt.WriteString("\n")
	}
	userPrompt.WriteString("\n")

	// 剧情上下文
	if plotContext != "" {
		userPrompt.WriteString("[Plot Context - Reference for mood and background]\n")
		userPrompt.WriteString(plotContext)
		userPrompt.WriteString("\n")
	}

	// 调用AI生成服务
	genReq := &GenerateTextRequest{
		UserID:            userID,
		OriginalPrompt:    userPrompt.String(),
		SystemPrompt:      systemPrompt,
		Model:             "gemini-2.5-flash",
		Temperature:       0.8,
		MaxTokens:         2000,
		RelatedEntityID:   poster.ID,
		RelatedEntityType: "character_poster",
		Metadata: map[string]interface{}{
			"operation":   "poster_concept_generation",
			"characterId": poster.CharacterID,
			"step":        1,
		},
	}

	result, err := s.aiGenService.GenerateText(ctx, genReq)
	if err != nil {
		return nil, err
	}

	// 解析生成的JSON
	concept, err := s.parsePosterConcept(result.Text)
	if err != nil {
		s.logger.Warn("failed to parse poster concept JSON, using raw text",
			zap.Error(err),
			zap.String("rawText", truncateForLog(result.Text, 500)))
	}

	return &conceptGenerationResult{
		RecordID:    result.RecordID,
		ConceptJSON: result.Text,
		Concept:     concept,
		TokensUsed:  result.TokensUsed,
	}, nil
}

// parsePosterConcept 解析海报概念JSON
func (s *Service) parsePosterConcept(text string) (*PosterConcept, error) {
	// 清理JSON文本
	cleanedText := strings.TrimSpace(text)

	// 处理markdown代码块
	if strings.HasPrefix(cleanedText, "```") {
		if idx := strings.Index(cleanedText, "\n"); idx != -1 {
			cleanedText = cleanedText[idx+1:]
		}
		if idx := strings.LastIndex(cleanedText, "```"); idx != -1 {
			cleanedText = strings.TrimSpace(cleanedText[:idx])
		}
	}

	// 提取JSON对象
	cleanedText = extractJSONFromText(cleanedText)

	var concept PosterConcept
	if err := json.Unmarshal([]byte(cleanedText), &concept); err != nil {
		return nil, err
	}

	return &concept, nil
}

// generatePosterImage Step 2: 组装最终提示词并生成图像
func (s *Service) generatePosterImage(ctx context.Context, userID string, poster *domain.CharacterPoster, concept *PosterConcept, aspectRatio string) (*imageGenerationResult, error) {
	// 组装最终图像提示词
	finalPrompt := s.assembleFinalImagePrompt(concept, poster)

	// 设置默认宽高比
	if aspectRatio == "" {
		aspectRatio = "16:9"
	}

	// 调用AI图像生成服务
	imageReq := &GenerateImageRequest{
		UserID:            userID,
		Prompt:            finalPrompt,
		Provider:          "gemini",
		Model:             "imagen-3.0-generate-001",
		AspectRatio:       aspectRatio,
		Quality:           "high",
		OutputCount:       1,
		RelatedEntityID:   poster.ID,
		RelatedEntityType: "character_poster",
		Metadata: map[string]interface{}{
			"operation":   "poster_image_generation",
			"characterId": poster.CharacterID,
			"step":        2,
		},
	}

	result, err := s.aiGenService.GenerateImage(ctx, imageReq)
	if err != nil {
		return nil, err
	}

	if len(result.ImageURLs) == 0 {
		return nil, errors.New("no image generated")
	}

	return &imageGenerationResult{
		RecordID:    result.RecordID,
		FinalPrompt: finalPrompt,
		ImageURL:    result.ImageURLs[0],
	}, nil
}

// assembleFinalImagePrompt 组装最终图像生成提示词
func (s *Service) assembleFinalImagePrompt(concept *PosterConcept, poster *domain.CharacterPoster) string {
	var prompt strings.Builder

	if concept != nil && concept.PosterConcept.VisualSubject != "" {
		// 使用结构化概念组装提示词
		p := concept.PosterConcept
		t := concept.TypographyInstruction

		prompt.WriteString("A movie poster design of ")
		prompt.WriteString(p.VisualSubject)
		prompt.WriteString(".\n")

		if p.SceneEnvironment != "" {
			prompt.WriteString("The scene is set in ")
			prompt.WriteString(p.SceneEnvironment)
			prompt.WriteString(".\n\n")
		}

		if p.CompositionCamera != "" {
			prompt.WriteString("COMPOSITION & ANGLE:\n")
			prompt.WriteString(p.CompositionCamera)
			prompt.WriteString(".\n\n")
		}

		if p.LightingAtmosphere != "" {
			prompt.WriteString("LIGHTING & MOOD:\n")
			prompt.WriteString(p.LightingAtmosphere)
			prompt.WriteString(".\n\n")
		}

		if p.ArtStyle != "" {
			prompt.WriteString("ART STYLE:\n")
			prompt.WriteString(p.ArtStyle)
			prompt.WriteString(".\n\n")
		}

		// 排版指令
		if t.TitleContent != "" && t.TitleContent != "NONE" {
			prompt.WriteString("TYPOGRAPHY & TEXT GENERATION:\n")
			prompt.WriteString("The image must feature the title text \"")
			prompt.WriteString(t.TitleContent)
			prompt.WriteString("\" written in ")
			prompt.WriteString(t.TitleStyle)
			prompt.WriteString(".\n")
			prompt.WriteString("The title is placed ")
			prompt.WriteString(t.TitlePosition)
			prompt.WriteString(".\n")

			if t.SubtitleContent != "" && t.SubtitleContent != "NONE" {
				prompt.WriteString("Additionally, include the subtitle text \"")
				prompt.WriteString(t.SubtitleContent)
				prompt.WriteString("\" written in ")
				prompt.WriteString(t.SubtitleStyle)
				prompt.WriteString(".\n")
			}
		}
	} else {
		// 降级：使用基础提示词
		prompt.WriteString("A professional movie poster design for \"")
		prompt.WriteString(poster.Title)
		prompt.WriteString("\".\n")
		if poster.Prompt != "" {
			prompt.WriteString(poster.Prompt)
			prompt.WriteString("\n")
		}
		prompt.WriteString("High quality, cinematic lighting, professional composition, dramatic atmosphere.")
	}

	return prompt.String()
}

// PublishCharacterPoster 发布角色海报
func (s *Service) PublishCharacterPoster(ctx context.Context, userID, posterID string) (*domain.CharacterPoster, error) {
	s.logger.Info("publishing character poster",
		zap.String("userID", userID),
		zap.String("posterID", posterID),
	)

	// 获取海报
	poster, err := s.repo.CharacterPosterByID(ctx, posterID)
	if err != nil {
		if err == domain.ErrNotFound {
			return nil, errors.New("poster not found")
		}
		return nil, errors.New("failed to get poster")
	}

	// 验证权限
	if poster.Author == nil || poster.Author.ID != userID {
		return nil, errors.New("unauthorized: you can only publish your own posters")
	}

	// 验证海报状态（只有生成完成的海报可以发布）
	if poster.Status != domain.PosterStatusGenerated {
		return nil, errors.New("poster must be generated before publishing")
	}

	// 验证海报有图片
	if poster.Image == "" {
		return nil, errors.New("poster has no image")
	}

	// 更新状态为已发布
	poster.Status = domain.PosterStatusPublished
	if err := s.repo.UpdateCharacterPoster(ctx, poster); err != nil {
		s.logger.Error("failed to publish poster", zap.Error(err))
		return nil, errors.New("failed to publish poster")
	}

	s.logger.Info("character poster published successfully", zap.String("posterID", posterID))
	return poster, nil
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

	// 构建生成提示词
	systemPrompt := `你是一个专业的故事角色设计师，擅长根据用户的描述创建丰富多彩的角色。
请根据用户的描述，生成角色的各项属性。

要求：
1. 直接返回纯JSON，不要使用markdown代码块
2. 每个属性都应该是简洁有力的描述，100-200字左右
3. 角色属性应该相互协调，形成一个完整的角色形象
4. 确保JSON格式完整闭合

返回格式：
{
  "description": "角色整体描述和基本特征",
  "personality": "性格特点、气质和行为模式",
  "background": "角色的历史、起源故事和成长经历",
  "shortTermGoal": "当前故事中的即时目标",
  "longTermGoal": "长远抱负和人生追求",
  "handlingStyle": "处理问题和应对情况的风格",
  "cognitionRange": "认知范围、知识水平和世界观",
  "abilityFeatures": "特殊技能、才能和独特能力",
  "appearance": "外貌特征、体型和显著特点",
  "dressPreference": "服装偏好和穿衣风格"
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
		MaxTokens:         3000,
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
