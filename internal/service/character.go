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

	// 确定海报创建权限
	posterPerm := req.PosterCreationPermission
	if posterPerm == "" {
		// 默认值：如果是公开角色则为 anyone，否则为 creator_only
		if req.IsPublic {
			posterPerm = "anyone"
		} else {
			posterPerm = "creator_only"
		}
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
		Followers:                0,
		Stories:                  0,
		PosterCreationPermission: posterPerm,
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

	// 记录用户活动
	go s.RecordCharacterCreated(context.Background(), userID, character.ID, character.Name)

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

	// 根据 PosterCreationPermission 检查权限
	perm := character.PosterCreationPermission
	if perm == "" {
		perm = "creator_only" // 默认权限
	}

	s.logger.Debug("checking poster creation permission",
		zap.String("userID", userID),
		zap.String("characterID", characterID),
		zap.String("permission", perm),
	)

	switch perm {
	case "creator_only":
		// 仅角色创建者可创建海报
		if userID != character.UserID {
			s.logger.Warn("unauthorized poster creation attempt",
				zap.String("userID", userID),
				zap.String("characterID", characterID),
				zap.String("characterUserID", character.UserID),
				zap.String("permission", perm),
			)
			return nil, errors.New("unauthorized: only character creator can create posters")
		}

	case "anyone":
		// 任何人都可以创建，不需要额外检查
		s.logger.Debug("anyone can create poster for this character",
			zap.String("characterID", characterID),
			zap.String("userID", userID),
		)

	default:
		// 默认使用 creator_only
		s.logger.Warn("unknown poster creation permission, defaulting to creator_only",
			zap.String("characterID", characterID),
			zap.String("permission", perm),
		)
		if userID != character.UserID {
			return nil, errors.New("unauthorized: only character creator can create posters")
		}
	}

	s.logger.Debug("poster creation permission verified",
		zap.String("userID", userID),
		zap.String("characterID", characterID),
		zap.String("permission", perm),
	)

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
	Provider    string `json:"provider"`    // huoshan (default), nana, banana
}

// GeneratePosterResult 生成海报结果
type GeneratePosterResult struct {
	Poster              *domain.CharacterPoster `json:"poster"`
	ConceptGenerationID string                  `json:"conceptGenerationId"` // Step 1 AI record
	ImageGenerationID   string                  `json:"imageGenerationId"`   // Step 2 AI record
}

// PosterConcept LLM生成的海报概念结构
// PosterConcept AI返回的海报概念JSON结构（snake_case匹配AI输出）
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

// convertToPosterConceptDetails 将AI返回的PosterConcept转换为domain.PosterConceptDetails
func (s *Service) convertToPosterConceptDetails(concept *PosterConcept) *domain.PosterConceptDetails {
	if concept == nil {
		return nil
	}
	return &domain.PosterConceptDetails{
		VisualConcept: &domain.PosterVisualConcept{
			VisualSubject:      concept.PosterConcept.VisualSubject,
			SceneEnvironment:   concept.PosterConcept.SceneEnvironment,
			CompositionCamera:  concept.PosterConcept.CompositionCamera,
			LightingAtmosphere: concept.PosterConcept.LightingAtmosphere,
			ArtStyle:           concept.PosterConcept.ArtStyle,
		},
		Typography: &domain.PosterTypography{
			TitleContent:    concept.TypographyInstruction.TitleContent,
			TitleStyle:      concept.TypographyInstruction.TitleStyle,
			TitlePosition:   concept.TypographyInstruction.TitlePosition,
			SubtitleContent: concept.TypographyInstruction.SubtitleContent,
			SubtitleStyle:   concept.TypographyInstruction.SubtitleStyle,
		},
	}
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
	conceptStartTime := time.Now()
	conceptResult, err := s.generatePosterConcept(ctx, userID, poster, character, plotContext)
	conceptDuration := time.Since(conceptStartTime)
	if err != nil {
		s.logger.Error("failed to generate poster concept", zap.Error(err))
		poster.Status = domain.PosterStatusFailed
		poster.ErrorMessage = "Failed to generate poster concept: " + err.Error()
		_ = s.repo.UpdateCharacterPoster(ctx, poster)
		// Record metrics: concept generation failed
		if s.metrics != nil {
			s.metrics.RecordCharacterPosterConceptTime("failed", conceptDuration)
			s.metrics.RecordCharacterPosterGeneration("failed", plotContext != "")
			s.metrics.RecordCharacterPosterError("concept_error")
		}
		return nil, errors.New("failed to generate poster concept: " + err.Error())
	}
	// Record metrics: concept generation completed
	if s.metrics != nil {
		s.metrics.RecordCharacterPosterConceptTime("completed", conceptDuration)
		if conceptResult.TokensUsed > 0 {
			s.metrics.RecordCharacterPosterTokenConsumed("concept", float64(conceptResult.TokensUsed))
		}
	}

	poster.ConceptGenerationID = conceptResult.RecordID
	poster.PosterConceptJSON = conceptResult.ConceptJSON
	poster.PosterConcept = s.convertToPosterConceptDetails(conceptResult.Concept)

	// 10. Step 2: 组装最终提示词并生成图像
	imageStartTime := time.Now()
	imageResult, err := s.generatePosterImage(ctx, userID, poster, conceptResult.Concept, req.AspectRatio, req.Provider)
	imageDuration := time.Since(imageStartTime)
	if err != nil {
		s.logger.Error("failed to generate poster image", zap.Error(err))
		poster.Status = domain.PosterStatusFailed
		poster.ErrorMessage = "Failed to generate poster image: " + err.Error()
		_ = s.repo.UpdateCharacterPoster(ctx, poster)
		// Record metrics: image generation failed
		if s.metrics != nil {
			s.metrics.RecordCharacterPosterImageTime("failed", imageDuration)
			s.metrics.RecordCharacterPosterGeneration("failed", plotContext != "")
			s.metrics.RecordCharacterPosterError("image_error")
		}
		return nil, errors.New("failed to generate poster image: " + err.Error())
	}
	// Record metrics: image generation completed
	if s.metrics != nil {
		s.metrics.RecordCharacterPosterImageTime("completed", imageDuration)
		// Note: Token consumption for image generation is tracked in aiGenService
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

	// Record metrics: poster generation completed
	if s.metrics != nil {
		s.metrics.RecordCharacterPosterGenerationTime("concept", conceptDuration)
		s.metrics.RecordCharacterPosterGenerationTime("image", imageDuration)
		s.metrics.RecordCharacterPosterGeneration("completed", plotContext != "")
		// Record AI generation
		s.metrics.RecordAIGeneration("gemini", "poster")
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
		MaxTokens:         4000, // 增加 token 限制以确保完整的 JSON 响应
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
func (s *Service) generatePosterImage(ctx context.Context, userID string, poster *domain.CharacterPoster, concept *PosterConcept, aspectRatio string, provider string) (*imageGenerationResult, error) {
	// 组装最终图像提示词
	finalPrompt := s.assembleFinalImagePrompt(concept, poster)

	// 设置默认宽高比
	if aspectRatio == "" {
		aspectRatio = "16:9"
	}

	// 设置默认 provider 为 huoshan，支持 nana、banana 等特殊指定
	if provider == "" {
		provider = "huoshan"
	}

	// 根据 provider 构建不同的请求参数
	imageReq := &GenerateImageRequest{
		UserID:            userID,
		Prompt:            finalPrompt,
		Provider:          provider,
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

// aspectRatioToSize 将宽高比转换为具体尺寸（用于 huoshan provider）
// huoshan 要求图片至少 3686400 像素 (约 1920x1920)
func aspectRatioToSize(aspectRatio string) string {
	switch aspectRatio {
	case "16:9":
		return "2560x1440" // 3686400 pixels
	case "9:16":
		return "1440x2560" // 3686400 pixels
	case "1:1":
		return "1920x1920" // 3686400 pixels
	case "4:3":
		return "2220x1665" // ~3696300 pixels
	case "3:4":
		return "1665x2220" // ~3696300 pixels
	case "2:3":
		return "1600x2400" // 3840000 pixels (portrait format, meets huoshan minimum)
	case "3:2":
		return "2400x1600" // 3840000 pixels (landscape format, meets huoshan minimum)
	default:
		return "1600x2400" // 默认 2:3 竖版人像
	}
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

// GenerateCharacterViews 生成角色三视图 (front/side/back)
// StoryCreationAppUI Design: AI generates three-view images for character
func (s *Service) GenerateCharacterViews(ctx context.Context, userID, characterID string, req domain.GenerateCharacterViewsRequest) (*domain.GenerateCharacterViewsResponse, error) {
	s.logger.Info("generating character views",
		zap.String("userID", userID),
		zap.String("characterID", characterID))

	// 获取角色
	character, err := s.repo.CharacterByID(ctx, characterID)
	if err != nil {
		if err == domain.ErrNotFound {
			return nil, errors.New("character not found")
		}
		return nil, errors.New("failed to get character")
	}

	// 权限检查：只有角色创建者可以生成视图
	if character.UserID != userID {
		return nil, errors.New("unauthorized: only character creator can generate views")
	}

	// 确定要生成的视图类型
	viewTypes := req.ViewTypes
	if len(viewTypes) == 0 {
		viewTypes = []domain.CharacterViewType{
			domain.CharacterViewFront,
			domain.CharacterViewSide,
			domain.CharacterViewBack,
		}
	}

	// 构建AI生成提示词
	basePrompt := req.CustomPrompt
	if basePrompt == "" {
		// 从角色属性构建提示词
		basePrompt = s.buildCharacterViewPrompt(character)
	}

	var generatedViews []domain.CharacterView

	for _, viewType := range viewTypes {
		// 创建视图记录
		viewID := uuid.New().String()
		view := domain.CharacterView{
			CharacterID:   characterID,
			ViewType:      viewType,
			ImageURL:      "", // 将在AI生成后更新
			IsAIGenerated: true,
			Prompt:        s.buildViewSpecificPrompt(basePrompt, viewType),
			Status:        domain.CharacterViewStatusGenerating,
		}
		view.ID = viewID

		// 保存到数据库
		if err := s.repo.CreateCharacterView(ctx, &view); err != nil {
			s.logger.Error("failed to create character view",
				zap.Error(err),
				zap.String("viewType", string(viewType)))
			continue
		}

		// 异步调用AI生成
		go s.generateViewImage(characterID, viewID, view.Prompt, viewType)

		generatedViews = append(generatedViews, view)
	}

	s.logger.Info("character view generation started",
		zap.String("characterID", characterID),
		zap.Int("viewCount", len(generatedViews)))

	estimatedTime := 30 // 估计30秒
	return &domain.GenerateCharacterViewsResponse{
		Views:         generatedViews,
		TaskID:        uuid.New().String(),
		EstimatedTime: estimatedTime,
	}, nil
}

// GetCharacterViews 获取角色三视图
func (s *Service) GetCharacterViews(ctx context.Context, characterID string) ([]domain.CharacterView, error) {
	views, err := s.repo.GetCharacterViewsByCharacterID(ctx, characterID)
	if err != nil {
		s.logger.Error("failed to get character views",
			zap.Error(err),
			zap.String("characterID", characterID))
		return nil, errors.New("failed to get character views")
	}
	return views, nil
}

// buildCharacterViewPrompt 构建角色视图生成提示词
func (s *Service) buildCharacterViewPrompt(character *domain.Character) string {
	var promptParts []string

	if character.Name != "" {
		promptParts = append(promptParts, "character named "+character.Name)
	}

	if character.Appearance != "" {
		promptParts = append(promptParts, character.Appearance)
	}

	if character.DressPreference != "" {
		promptParts = append(promptParts, "wearing "+character.DressPreference)
	}

	if len(promptParts) == 0 {
		return "character portrait, high quality, detailed"
	}

	return strings.Join(promptParts, ", ") + ", character design, high quality, detailed"
}

// buildViewSpecificPrompt 为特定视图构建提示词
func (s *Service) buildViewSpecificPrompt(basePrompt string, viewType domain.CharacterViewType) string {
	var viewAngle string
	switch viewType {
	case domain.CharacterViewFront:
		viewAngle = "front view, facing camera"
	case domain.CharacterViewSide:
		viewAngle = "side view, profile"
	case domain.CharacterViewBack:
		viewAngle = "back view, from behind"
	default:
		viewAngle = "character view"
	}

	return basePrompt + ", " + viewAngle + ", full body, white background, character reference sheet style"
}

// generateViewImage 异步生成视图图片
func (s *Service) generateViewImage(characterID, viewID, prompt string, viewType domain.CharacterViewType) {
	ctx := context.Background()

	// 调用AI服务生成图片
	imageReq := &GenerateImageRequest{
		Prompt: prompt,
		Size:   "1024x1024",
	}
	result, err := s.aiGenService.GenerateImage(ctx, imageReq)
	if err != nil {
		s.logger.Error("failed to generate character view image",
			zap.Error(err),
			zap.String("viewID", viewID))
		// 更新状态为失败
		_ = s.repo.UpdateCharacterViewStatus(ctx, viewID, domain.CharacterViewStatusFailed, "")
		return
	}

	// 获取生成的图片URL
	var imageURL string
	if len(result.ImageURLs) > 0 {
		imageURL = result.ImageURLs[0]
	}

	// 更新视图记录
	if err := s.repo.UpdateCharacterViewStatus(ctx, viewID, domain.CharacterViewStatusCompleted, imageURL); err != nil {
		s.logger.Error("failed to update character view",
			zap.Error(err),
			zap.String("viewID", viewID))
		return
	}

	s.logger.Info("character view generated successfully",
		zap.String("viewID", viewID),
		zap.String("viewType", string(viewType)))
}
