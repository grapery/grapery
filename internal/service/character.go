package service

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

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
		StoryID:                  req.StoryID,
		UserID:                   author.ID,
		Name:                     req.Name,
		Description:              req.Description,
		Avatar:                   req.Avatar,
		Poster:                   "",
		Portrait:                 "",
		NeedsPortrait:            req.NeedsPortrait,
		ReferenceImage:           req.ReferenceImage,
		PortraitGenerationStatus: portraitStatus,
		Author:                   author,
		Personality:              req.Personality,
		Background:               req.Background,
		ShortTermGoal:            req.ShortTermGoal,
		LongTermGoal:             req.LongTermGoal,
		HandlingStyle:            req.HandlingStyle,
		CognitionRange:           req.CognitionRange,
		AbilityFeatures:          req.AbilityFeatures,
		Appearance:               req.Appearance,
		DressPreference:          req.DressPreference,
		IsPublic:                 req.IsPublic,
		SourceType:               sourceType,
		SourcePrompt:             req.SourcePrompt,
		SourceImage:              req.SourceImage,
		CreatedBy:                userID,
		LastEditedBy:             userID,
		Likes:                    0,
		Comments:                 0,
		Shares:                   0,
		Followers: 0,
		Stories:   0,
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

	// 构建生成提示词
	systemPrompt := `你是一个专业的故事角色设计师，擅长根据用户的描述创建丰富多彩的角色。
请根据用户的描述，生成角色的各项属性。

要求：
1. 直接返回纯JSON，不要使用markdown代码块
2. 角色描述(description)字段总长度不超过420字，确保输出完整的JSON数据结构
3. 每个属性都应该是简洁有力的描述，100-200字左右
4. 角色属性应该相互协调，形成一个完整的角色形象
5. 确保JSON格式完整闭合

返回格式：
{
  "description": "角色整体描述和基本特征（不超过420字）",
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
