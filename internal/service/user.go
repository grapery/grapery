package service

import (
	"context"
	"fmt"

	"github.com/grapestree/fgrapery/grapery/internal/cache"
	"github.com/grapestree/fgrapery/grapery/internal/domain"
	"go.uber.org/zap"
)

func (s *Service) FollowUser(ctx context.Context, followerID, followeeID string) error {
	s.logger.Info("following user",
		zap.String("followerID", followerID),
		zap.String("followeeID", followeeID))

	if err := s.repo.FollowUser(ctx, followerID, followeeID); err != nil {
		s.logger.Error("failed to follow user",
			zap.String("followerID", followerID),
			zap.String("followeeID", followeeID),
			zap.Error(err))
		return fmt.Errorf("failed to follow user: %w", err)
	}

	// 使相关缓存失效
	c := s.getCache()
	if c != nil {
		_ = c.Delete(ctx, cache.UserKey(followerID))
		_ = c.Delete(ctx, cache.UserKey(followeeID))
		// 清除关注者和被关注者的关注列表缓存
		for limit := 20; limit <= 100; limit += 20 {
			for offset := 0; offset < 200; offset += limit {
				_ = c.Delete(ctx, cache.UserFollowingKey(followerID)+fmt.Sprintf(":%d:%d", limit, offset))
				_ = c.Delete(ctx, cache.UserFollowersKey(followeeID)+fmt.Sprintf(":%d:%d", limit, offset))
			}
		}
		s.logger.Debug("follow/unfollow cache invalidated",
			zap.String("followerID", followerID),
			zap.String("followeeID", followeeID))
	}

	s.logger.Info("user followed successfully",
		zap.String("followerID", followerID),
		zap.String("followeeID", followeeID))

	// Create notification
	followee, _ := s.repo.UserByID(ctx, followeeID)
	follower, _ := s.repo.UserByID(ctx, followerID)
	if followee != nil && follower != nil {
		s.NotifyFollow(ctx, followeeID, followerID, follower.DisplayName, follower.Avatar)
	}

	return nil
}

func (s *Service) UnfollowUser(ctx context.Context, followerID, followeeID string) error {
	s.logger.Info("unfollowing user",
		zap.String("followerID", followerID),
		zap.String("followeeID", followeeID))

	if err := s.repo.UnfollowUser(ctx, followerID, followeeID); err != nil {
		s.logger.Error("failed to unfollow user",
			zap.String("followerID", followerID),
			zap.String("followeeID", followeeID),
			zap.Error(err))
		return fmt.Errorf("failed to unfollow user: %w", err)
	}

	// 使相关缓存失效
	c := s.getCache()
	if c != nil {
		_ = c.Delete(ctx, cache.UserKey(followerID))
		_ = c.Delete(ctx, cache.UserKey(followeeID))
		// 清除关注者和被关注者的关注列表缓存
		for limit := 20; limit <= 100; limit += 20 {
			for offset := 0; offset < 200; offset += limit {
				_ = c.Delete(ctx, cache.UserFollowingKey(followerID)+fmt.Sprintf(":%d:%d", limit, offset))
				_ = c.Delete(ctx, cache.UserFollowersKey(followeeID)+fmt.Sprintf(":%d:%d", limit, offset))
			}
		}
		s.logger.Debug("follow/unfollow cache invalidated",
			zap.String("followerID", followerID),
			zap.String("followeeID", followeeID))
	}

	s.logger.Info("user unfollowed successfully",
		zap.String("followerID", followerID),
		zap.String("followeeID", followeeID))
	return nil
}

func (s *Service) IsFollowing(ctx context.Context, followerID, followeeID string) (bool, error) {
	return s.repo.IsFollowing(ctx, followerID, followeeID)
}

func (s *Service) GetFollowers(ctx context.Context, userID string, limit, offset int) ([]*domain.User, error) {
	s.logger.Debug("getting followers",
		zap.String("userID", userID),
		zap.Int("limit", limit),
		zap.Int("offset", offset))

	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	// 尝试从缓存获取
	c := s.getCache()
	if c != nil {
		cacheKey := cache.UserFollowersKey(userID) + fmt.Sprintf(":%d:%d", limit, offset)
		var cachedFollowers []*domain.User
		if err := c.Get(ctx, cacheKey, &cachedFollowers); err == nil {
			s.logger.Debug("followers cache hit",
				zap.String("userID", userID),
				zap.Int("count", len(cachedFollowers)))
			return cachedFollowers, nil
		} else {
			s.logger.Debug("followers cache miss",
				zap.String("userID", userID),
				zap.Error(err))
		}
	}

	// 从数据库获取
	followers, err := s.repo.Followers(ctx, userID, limit, offset)
	if err != nil {
		s.logger.Error("failed to get followers",
			zap.String("userID", userID),
			zap.Error(err))
		return nil, err
	}

	// 写入缓存
	if c != nil && len(followers) > 0 {
		cacheKey := cache.UserFollowersKey(userID) + fmt.Sprintf(":%d:%d", limit, offset)
		if err := c.Set(ctx, cacheKey, followers, listCacheTTL); err != nil {
			s.logger.Warn("failed to cache followers",
				zap.String("userID", userID),
				zap.Error(err))
		} else {
			s.logger.Debug("followers cached",
				zap.String("userID", userID),
				zap.Int("count", len(followers)))
		}
	}

	return followers, nil
}

func (s *Service) GetFollowing(ctx context.Context, userID string, limit, offset int) ([]*domain.User, error) {
	s.logger.Debug("getting following",
		zap.String("userID", userID),
		zap.Int("limit", limit),
		zap.Int("offset", offset))

	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	// 尝试从缓存获取
	c := s.getCache()
	if c != nil {
		cacheKey := cache.UserFollowingKey(userID) + fmt.Sprintf(":%d:%d", limit, offset)
		var cachedFollowing []*domain.User
		if err := c.Get(ctx, cacheKey, &cachedFollowing); err == nil {
			s.logger.Debug("following cache hit",
				zap.String("userID", userID),
				zap.Int("count", len(cachedFollowing)))
			return cachedFollowing, nil
		} else {
			s.logger.Debug("following cache miss",
				zap.String("userID", userID),
				zap.Error(err))
		}
	}

	// 从数据库获取
	following, err := s.repo.Following(ctx, userID, limit, offset)
	if err != nil {
		s.logger.Error("failed to get following",
			zap.String("userID", userID),
			zap.Error(err))
		return nil, err
	}

	// 写入缓存
	if c != nil && len(following) > 0 {
		cacheKey := cache.UserFollowingKey(userID) + fmt.Sprintf(":%d:%d", limit, offset)
		if err := c.Set(ctx, cacheKey, following, listCacheTTL); err != nil {
			s.logger.Warn("failed to cache following",
				zap.String("userID", userID),
				zap.Error(err))
		} else {
			s.logger.Debug("following cached",
				zap.String("userID", userID),
				zap.Int("count", len(following)))
		}
	}

	return following, nil
}

// GetFollowersWithFollowStatus 获取粉丝列表，并包含当前用户的关注状态
func (s *Service) GetFollowersWithFollowStatus(ctx context.Context, userID, currentUserID string, limit, offset int) ([]*domain.User, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	followers, err := s.repo.Followers(ctx, userID, limit, offset)
	if err != nil {
		return nil, err
	}

	// 如果当前用户已登录，检查关注状态
	if currentUserID != "" {
		for _, user := range followers {
			isFollowing, err := s.repo.IsFollowing(ctx, currentUserID, user.ID)
			if err == nil {
				user.IsFollowing = &isFollowing
			}
		}
	}

	return followers, nil
}

// GetFollowingWithFollowStatus 获取关注列表，并包含当前用户的关注状态
func (s *Service) GetFollowingWithFollowStatus(ctx context.Context, userID, currentUserID string, limit, offset int) ([]*domain.User, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	following, err := s.repo.Following(ctx, userID, limit, offset)
	if err != nil {
		return nil, err
	}

	// 如果当前用户已登录，检查关注状态
	if currentUserID != "" {
		for _, user := range following {
			isFollowing, err := s.repo.IsFollowing(ctx, currentUserID, user.ID)
			if err == nil {
				user.IsFollowing = &isFollowing
			}
		}
	}

	return following, nil
}

// UpdateUserProfile 更新用户资料（带缓存失效）
func (s *Service) UpdateUserProfile(ctx context.Context, userID string, req *UpdateProfileRequest) (*domain.User, error) {
	s.logger.Info("updating user profile",
		zap.String("userID", userID))

	user, err := s.repo.UserByID(ctx, userID)
	if err != nil {
		s.logger.Error("failed to get user for update",
			zap.String("userID", userID),
			zap.Error(err))
		return nil, fmt.Errorf("user not found: %w", err)
	}

	// 更新字段
	if req.DisplayName != nil {
		user.DisplayName = *req.DisplayName
	}
	if req.Bio != nil {
		user.Bio = *req.Bio
	}
	if req.Avatar != nil {
		user.Avatar = *req.Avatar
	}
	if req.Background != nil {
		user.Background = *req.Background
	}
	if req.Location != nil {
		user.Location = *req.Location
	}
	if req.Website != nil {
		user.Website = *req.Website
	}
	if req.AIPromptPreferences != nil {
		user.AIPromptPreferences = *req.AIPromptPreferences
	}

	if err := s.repo.UpdateUser(ctx, user); err != nil {
		s.logger.Error("failed to update user profile",
			zap.String("userID", userID),
			zap.Error(err))
		return nil, fmt.Errorf("failed to update user profile: %w", err)
	}

	// 使缓存失效并重新缓存
	s.invalidateUserCache(ctx, userID)
	s.cacheUser(ctx, user)

	// 使相关列表缓存失效
	c := s.getCache()
	if c != nil {
		// 清除用户故事列表缓存（简化实现，实际应该更精确）
		for limit := 20; limit <= 100; limit += 20 {
			for offset := 0; offset < 100; offset += limit {
				_ = c.Delete(ctx, cache.UserStoriesListKey(userID, limit, offset))
			}
		}
		// 清除用户角色列表缓存
		for limit := 20; limit <= 100; limit += 20 {
			for offset := 0; offset < 100; offset += limit {
				_ = c.Delete(ctx, cache.UserCharactersListKey(userID, limit, offset))
			}
		}
	}

	s.logger.Info("user profile updated",
		zap.String("userID", userID))
	return user, nil
}

// UpdateUserAvatar 更新用户头像（带缓存失效）
func (s *Service) UpdateUserAvatar(ctx context.Context, userID, avatarURL string) error {
	s.logger.Info("updating user avatar",
		zap.String("userID", userID))

	user, err := s.repo.UserByID(ctx, userID)
	if err != nil {
		s.logger.Error("failed to get user for avatar update",
			zap.String("userID", userID),
			zap.Error(err))
		return fmt.Errorf("user not found: %w", err)
	}

	user.Avatar = avatarURL
	if err := s.repo.UpdateUser(ctx, user); err != nil {
		s.logger.Error("failed to update user avatar",
			zap.String("userID", userID),
			zap.Error(err))
		return fmt.Errorf("failed to update user avatar: %w", err)
	}

	// 使缓存失效并重新缓存
	s.invalidateUserCache(ctx, userID)
	s.cacheUser(ctx, user)

	s.logger.Info("user avatar updated",
		zap.String("userID", userID))
	return nil
}

// UpdateUserBackground 更新用户背景图（带缓存失效）
func (s *Service) UpdateUserBackground(ctx context.Context, userID, backgroundURL string) error {
	s.logger.Info("updating user background",
		zap.String("userID", userID))

	user, err := s.repo.UserByID(ctx, userID)
	if err != nil {
		s.logger.Error("failed to get user for background update",
			zap.String("userID", userID),
			zap.Error(err))
		return fmt.Errorf("user not found: %w", err)
	}

	user.Background = backgroundURL
	if err := s.repo.UpdateUser(ctx, user); err != nil {
		s.logger.Error("failed to update user background",
			zap.String("userID", userID),
			zap.Error(err))
		return fmt.Errorf("failed to update user background: %w", err)
	}

	// 使缓存失效并重新缓存
	s.invalidateUserCache(ctx, userID)
	s.cacheUser(ctx, user)

	s.logger.Info("user background updated",
		zap.String("userID", userID))
	return nil
}

// UserProfile 获取用户资料（对外API，带缓存）
func (s *Service) UserProfile(ctx context.Context, userID, viewerID string) (*domain.User, error) {
	s.logger.Debug("getting user profile",
		zap.String("userID", userID))

	var user *domain.User

	// 尝试从缓存获取
	c := s.getCache()
	if c != nil {
		key := cache.UserKey(userID)
		var cachedUser domain.User
		if err := c.Get(ctx, key, &cachedUser); err == nil {
			s.logger.Debug("user profile cache hit",
				zap.String("userID", userID))
			// Record cache hit
			if s.metrics != nil {
				s.metrics.RecordCacheHit("user")
			}
			user = &cachedUser
		} else {
			s.logger.Debug("user profile cache miss",
				zap.String("userID", userID),
				zap.Error(err))
			// Record cache miss
			if s.metrics != nil {
				s.metrics.RecordCacheMiss("user")
			}
		}
	}

	if user == nil {
		// 从数据库获取
		var err error
		user, err = s.repo.UserByID(ctx, userID)
		if err != nil {
			s.logger.Error("failed to get user profile",
				zap.String("userID", userID),
				zap.Error(err))
			return nil, fmt.Errorf("user not found: %w", err)
		}

		// 写入缓存
		s.cacheUser(ctx, user)
	}

	// Attach profile visibility toggles for client-side tab rendering.
	if settings, err := s.repo.UserSettings(ctx, userID); err == nil && settings != nil {
		user.ShowPublicStories = &settings.ShowPublicStories
		user.ShowPublicFragments = &settings.ShowPublicFragments
		user.ShowPublicBookmarks = &settings.ShowPublicBookmarks
	}

	// Keep follow state on profile response for convenient UI updates.
	if viewerID != "" && viewerID != userID {
		if isFollowing, err := s.repo.IsFollowing(ctx, viewerID, userID); err == nil {
			user.IsFollowing = &isFollowing
		}
	}

	return user, nil
}

// GetUser 获取用户信息（内部使用，带缓存）
func (s *Service) GetUser(ctx context.Context, userID string) (*domain.User, error) {
	s.logger.Debug("getting user",
		zap.String("userID", userID))

	// 尝试从缓存获取
	c := s.getCache()
	if c != nil {
		key := cache.UserKey(userID)
		var cachedUser domain.User
		if err := c.Get(ctx, key, &cachedUser); err == nil {
			s.logger.Debug("user cache hit",
				zap.String("userID", userID))
			_ = s.AttachPhoneVerificationRequirement(ctx, &cachedUser)
			return &cachedUser, nil
		} else {
			s.logger.Debug("user cache miss",
				zap.String("userID", userID),
				zap.Error(err))
		}
	}

	// 从数据库获取
	user, err := s.repo.UserByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, domain.ErrNotFound
	}

	_ = s.AttachPhoneVerificationRequirement(ctx, user)

	// 写入缓存
	s.cacheUser(ctx, user)

	return user, nil
}

// GetUserStories 获取用户的故事列表（带缓存）
func (s *Service) GetUserStories(ctx context.Context, userID, viewerID string, limit, offset int) ([]*domain.Story, error) {
	s.logger.Debug("getting user stories",
		zap.String("userID", userID),
		zap.Int("limit", limit),
		zap.Int("offset", offset))

	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	// 尝试从缓存获取
	c := s.getCache()
	if c != nil {
		cacheKey := cache.UserStoriesListKey(userID, limit, offset)
		var cachedStories []*domain.Story
		if err := c.Get(ctx, cacheKey, &cachedStories); err == nil {
			s.logger.Debug("user stories cache hit",
				zap.String("userID", userID),
				zap.Int("count", len(cachedStories)))
			return s.filterStoriesForViewer(ctx, cachedStories, userID, viewerID)
		} else {
			s.logger.Debug("user stories cache miss",
				zap.String("userID", userID),
				zap.Error(err))
		}
	}

	// 从数据库获取
	stories, err := s.repo.StoriesByUser(ctx, userID, limit, offset)
	if err != nil {
		s.logger.Error("failed to get user stories",
			zap.String("userID", userID),
			zap.Error(err))
		return nil, err
	}

	// 写入缓存
	if c != nil && len(stories) > 0 {
		cacheKey := cache.UserStoriesListKey(userID, limit, offset)
		if err := c.Set(ctx, cacheKey, stories, listCacheTTL); err != nil {
			s.logger.Warn("failed to cache user stories",
				zap.String("userID", userID),
				zap.Error(err))
		} else {
			s.logger.Debug("user stories cached",
				zap.String("userID", userID),
				zap.Int("count", len(stories)))
		}
	}

	return s.filterStoriesForViewer(ctx, stories, userID, viewerID)
}

// GetUserCharacters 获取用户的角色列表（带缓存）
func (s *Service) GetUserCharacters(ctx context.Context, userID string, limit, offset int) ([]*domain.Character, error) {
	s.logger.Debug("getting user characters",
		zap.String("userID", userID),
		zap.Int("limit", limit),
		zap.Int("offset", offset))

	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	// 尝试从缓存获取
	c := s.getCache()
	if c != nil {
		cacheKey := cache.UserCharactersListKey(userID, limit, offset)
		var cachedCharacters []*domain.Character
		if err := c.Get(ctx, cacheKey, &cachedCharacters); err == nil {
			s.logger.Debug("user characters cache hit",
				zap.String("userID", userID),
				zap.Int("count", len(cachedCharacters)))
			return cachedCharacters, nil
		} else {
			s.logger.Debug("user characters cache miss",
				zap.String("userID", userID),
				zap.Error(err))
		}
	}

	// 从数据库获取
	characters, err := s.repo.CharactersByUser(ctx, userID, limit, offset)
	if err != nil {
		s.logger.Error("failed to get user characters",
			zap.String("userID", userID),
			zap.Error(err))
		return nil, err
	}

	// 写入缓存
	if c != nil && len(characters) > 0 {
		cacheKey := cache.UserCharactersListKey(userID, limit, offset)
		if err := c.Set(ctx, cacheKey, characters, listCacheTTL); err != nil {
			s.logger.Warn("failed to cache user characters",
				zap.String("userID", userID),
				zap.Error(err))
		} else {
			s.logger.Debug("user characters cached",
				zap.String("userID", userID),
				zap.Int("count", len(characters)))
		}
	}

	return characters, nil
}

// GetLikedStories 获取用户点赞的故事 (V2 social feature)
func (s *Service) GetLikedStories(ctx context.Context, userID, viewerID string, limit, offset int) ([]*domain.Story, error) {
	s.logger.Debug("getting liked stories",
		zap.String("userID", userID),
		zap.Int("limit", limit),
		zap.Int("offset", offset))

	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	// Get story IDs that the user has liked from interaction repository
	storyIDs, err := s.repo.GetLikedStoryIDs(ctx, userID, limit, offset)
	if err != nil {
		s.logger.Error("failed to get liked story IDs",
			zap.String("userID", userID),
			zap.Error(err))
		return nil, err
	}

	if len(storyIDs) == 0 {
		return []*domain.Story{}, nil
	}

	// Fetch the actual stories
	stories := make([]*domain.Story, 0, len(storyIDs))
	for _, storyID := range storyIDs {
		story, err := s.repo.StoryByID(ctx, storyID)
		if err != nil {
			s.logger.Warn("failed to get liked story",
				zap.String("storyID", storyID),
				zap.Error(err))
			continue
		}
		stories = append(stories, story)
	}

	s.logger.Debug("got liked stories",
		zap.String("userID", userID),
		zap.Int("count", len(stories)))

	return s.filterStoriesForViewer(ctx, stories, userID, viewerID)
}

// GetLikedCharacters 获取用户点赞（关注）的角色 (V2 social feature)
func (s *Service) GetLikedCharacters(ctx context.Context, userID string, limit, offset int) ([]*domain.Character, error) {
	s.logger.Debug("getting liked characters",
		zap.String("userID", userID),
		zap.Int("limit", limit),
		zap.Int("offset", offset))

	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	// Get character IDs that the user has liked from interaction repository
	characterIDs, err := s.repo.GetLikedCharacterIDs(ctx, userID, limit, offset)
	if err != nil {
		s.logger.Error("failed to get liked character IDs",
			zap.String("userID", userID),
			zap.Error(err))
		return nil, err
	}

	if len(characterIDs) == 0 {
		return []*domain.Character{}, nil
	}

	// Fetch the actual characters
	characters := make([]*domain.Character, 0, len(characterIDs))
	for _, characterID := range characterIDs {
		character, err := s.repo.CharacterByID(ctx, characterID)
		if err != nil {
			s.logger.Warn("failed to get liked character",
				zap.String("characterID", characterID),
				zap.Error(err))
			continue
		}
		characters = append(characters, character)
	}

	s.logger.Debug("got liked characters",
		zap.String("userID", userID),
		zap.Int("count", len(characters)))

	return characters, nil
}

// GetLikedStoryboards 获取用户点赞的故事板 (V2 social feature)
func (s *Service) GetLikedStoryboards(ctx context.Context, userID, viewerID string, limit, offset int) ([]*domain.Storyboard, error) {
	s.logger.Debug("getting liked storyboards",
		zap.String("userID", userID),
		zap.Int("limit", limit),
		zap.Int("offset", offset))

	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	// Get storyboard IDs that the user has liked from interaction repository
	storyboardIDs, err := s.repo.GetLikedStoryboardIDs(ctx, userID, limit, offset)
	if err != nil {
		s.logger.Error("failed to get liked storyboard IDs",
			zap.String("userID", userID),
			zap.Error(err))
		return nil, err
	}

	if len(storyboardIDs) == 0 {
		return []*domain.Storyboard{}, nil
	}

	// Fetch the actual storyboards
	storyboards := make([]*domain.Storyboard, 0, len(storyboardIDs))
	for _, storyboardID := range storyboardIDs {
		storyboard, err := s.repo.StoryboardByID(ctx, storyboardID)
		if err != nil {
			s.logger.Warn("failed to get liked storyboard",
				zap.String("storyboardID", storyboardID),
				zap.Error(err))
			continue
		}
		storyboards = append(storyboards, storyboard)
	}

	s.logger.Debug("got liked storyboards",
		zap.String("userID", userID),
		zap.Int("count", len(storyboards)))

	return s.filterStoryboardsForViewer(ctx, storyboards, userID, viewerID)
}

// REMOVED: GetDraftStoryboards - not in StoryCreationAppUI design
// REMOVED: GetUserDrafts - not in StoryCreationAppUI design

// GetUserStoryboards 获取用户的故事板列表（已发布的）
func (s *Service) GetUserStoryboards(ctx context.Context, userID, viewerID string, limit, offset int) ([]*domain.Storyboard, error) {
	s.logger.Debug("getting user storyboards",
		zap.String("userID", userID),
		zap.Int("limit", limit),
		zap.Int("offset", offset))

	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	// 尝试从缓存获取
	c := s.getCache()
	if c != nil {
		cacheKey := cache.UserStoryboardsListKey(userID, limit, offset)
		var cachedStoryboards []*domain.Storyboard
		if err := c.Get(ctx, cacheKey, &cachedStoryboards); err == nil {
			s.logger.Debug("user storyboards cache hit",
				zap.String("userID", userID),
				zap.Int("count", len(cachedStoryboards)))
			return s.filterStoryboardsForViewer(ctx, cachedStoryboards, userID, viewerID)
		} else {
			s.logger.Debug("user storyboards cache miss",
				zap.String("userID", userID),
				zap.Error(err))
		}
	}

	// 从数据库获取
	storyboards, err := s.repo.StoryboardsByCreator(ctx, userID, limit, offset)
	if err != nil {
		s.logger.Error("failed to get user storyboards",
			zap.String("userID", userID),
			zap.Error(err))
		return nil, err
	}

	// 写入缓存
	if c != nil && len(storyboards) > 0 {
		cacheKey := cache.UserStoryboardsListKey(userID, limit, offset)
		if err := c.Set(ctx, cacheKey, storyboards, listCacheTTL); err != nil {
			s.logger.Warn("failed to cache user storyboards",
				zap.String("userID", userID),
				zap.Error(err))
		} else {
			s.logger.Debug("user storyboards cached",
				zap.String("userID", userID),
				zap.Int("count", len(storyboards)))
		}
	}

	return s.filterStoryboardsForViewer(ctx, storyboards, userID, viewerID)
}

// ListDashboardStoryboards returns storyboards created by the authenticated user (all workflow states),
// ordered by created_at DESC, for merged drafts UI and creator tools. Not cached.
func (s *Service) ListDashboardStoryboards(ctx context.Context, userID string, limit, offset int) ([]*domain.Storyboard, int64, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	total, err := s.repo.CountStoryboardsByCreator(ctx, userID)
	if err != nil {
		return nil, 0, err
	}
	storyboards, err := s.repo.StoryboardsByCreator(ctx, userID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	return storyboards, total, nil
}

func (s *Service) isUserContentPublicEnabled(ctx context.Context, ownerID string) (stories bool, fragments bool, bookmarks bool) {
	stories, fragments, bookmarks = true, true, true
	settings, err := s.repo.UserSettings(ctx, ownerID)
	if err != nil || settings == nil {
		return
	}
	return settings.ShowPublicStories, settings.ShowPublicFragments, settings.ShowPublicBookmarks
}

func (s *Service) isFollowerOf(ctx context.Context, viewerID, ownerID string) bool {
	if viewerID == "" || viewerID == ownerID {
		return false
	}
	isFollowing, err := s.repo.IsFollowing(ctx, viewerID, ownerID)
	return err == nil && isFollowing
}

func (s *Service) canViewerSeeStory(ctx context.Context, story *domain.Story, ownerID, viewerID string, storiesPublic bool, isFollower bool) bool {
	if story == nil {
		return false
	}
	if viewerID == ownerID {
		return true
	}
	if !storiesPublic {
		return false
	}
	switch story.Visibility {
	case string(domain.StoryVisibilityPrivate):
		return false
	case string(domain.StoryVisibilityFollowers):
		return isFollower
	default:
		return true
	}
}

func (s *Service) filterStoriesForViewer(ctx context.Context, stories []*domain.Story, ownerID, viewerID string) ([]*domain.Story, error) {
	if viewerID == ownerID {
		return stories, nil
	}
	storiesPublic, _, _ := s.isUserContentPublicEnabled(ctx, ownerID)
	if !storiesPublic {
		return []*domain.Story{}, nil
	}
	isFollower := s.isFollowerOf(ctx, viewerID, ownerID)
	filtered := make([]*domain.Story, 0, len(stories))
	for _, st := range stories {
		if s.canViewerSeeStory(ctx, st, ownerID, viewerID, storiesPublic, isFollower) {
			filtered = append(filtered, st)
		}
	}
	return filtered, nil
}

func (s *Service) filterStoryboardsForViewer(ctx context.Context, storyboards []*domain.Storyboard, ownerID, viewerID string) ([]*domain.Storyboard, error) {
	if viewerID == ownerID {
		return storyboards, nil
	}
	storiesPublic, _, _ := s.isUserContentPublicEnabled(ctx, ownerID)
	if !storiesPublic {
		return []*domain.Storyboard{}, nil
	}
	isFollower := s.isFollowerOf(ctx, viewerID, ownerID)
	filtered := make([]*domain.Storyboard, 0, len(storyboards))
	for _, sb := range storyboards {
		if sb == nil || sb.StoryID == "" {
			continue
		}
		story, err := s.repo.StoryByID(ctx, sb.StoryID)
		if err != nil {
			continue
		}
		if s.canViewerSeeStory(ctx, story, ownerID, viewerID, storiesPublic, isFollower) {
			filtered = append(filtered, sb)
		}
	}
	return filtered, nil
}

// REMOVED: GetUserDrafts - not in StoryCreationAppUI design
// REMOVED: GetUserActivityList - not in StoryCreationAppUI design
// REMOVED: GetUserActivitiesWithFilter - not in StoryCreationAppUI design
// REMOVED: CreateUserActivity - not in StoryCreationAppUI design
// REMOVED: RecordStoryCreated, RecordStoryPublished, RecordCharacterCreated, RecordUserFollowed, RecordStoryboardCreated - not in StoryCreationAppUI design

// BlockUser blocks a user
func (s *Service) BlockUser(ctx context.Context, blockerID, blockedID string) error {
	s.logger.Info("blocking user",
		zap.String("blockerID", blockerID),
		zap.String("blockedID", blockedID))

	if blockerID == blockedID {
		return fmt.Errorf("cannot block yourself")
	}

	if err := s.repo.BlockUser(ctx, blockerID, blockedID); err != nil {
		s.logger.Error("failed to block user",
			zap.String("blockerID", blockerID),
			zap.String("blockedID", blockedID),
			zap.Error(err))
		return fmt.Errorf("failed to block user: %w", err)
	}

	// Also unfollow if following
	_ = s.repo.UnfollowUser(ctx, blockerID, blockedID)
	_ = s.repo.UnfollowUser(ctx, blockedID, blockerID)

	s.logger.Info("user blocked successfully",
		zap.String("blockerID", blockerID),
		zap.String("blockedID", blockedID))
	return nil
}

// UnblockUser unblocks a user
func (s *Service) UnblockUser(ctx context.Context, blockerID, blockedID string) error {
	s.logger.Info("unblocking user",
		zap.String("blockerID", blockerID),
		zap.String("blockedID", blockedID))

	if err := s.repo.UnblockUser(ctx, blockerID, blockedID); err != nil {
		s.logger.Error("failed to unblock user",
			zap.String("blockerID", blockerID),
			zap.String("blockedID", blockedID),
			zap.Error(err))
		return fmt.Errorf("failed to unblock user: %w", err)
	}

	s.logger.Info("user unblocked successfully",
		zap.String("blockerID", blockerID),
		zap.String("blockedID", blockedID))
	return nil
}

// IsBlocked checks if a user is blocked
func (s *Service) IsBlocked(ctx context.Context, blockerID, blockedID string) (bool, error) {
	return s.repo.IsBlocked(ctx, blockerID, blockedID)
}

// ReportUser reports a user for inappropriate behavior
func (s *Service) ReportUser(ctx context.Context, reporterID, reportedID string, reason string) error {
	s.logger.Info("reporting user",
		zap.String("reporterID", reporterID),
		zap.String("reportedID", reportedID),
		zap.String("reason", reason))

	if reporterID == reportedID {
		return fmt.Errorf("cannot report yourself")
	}

	if err := s.repo.ReportUser(ctx, reporterID, reportedID, reason); err != nil {
		s.logger.Error("failed to report user",
			zap.String("reporterID", reporterID),
			zap.String("reportedID", reportedID),
			zap.Error(err))
		return fmt.Errorf("failed to report user: %w", err)
	}

	s.logger.Info("user reported successfully",
		zap.String("reporterID", reporterID),
		zap.String("reportedID", reportedID))
	return nil
}

// UpdateProfileRequest 更新资料请求
type UpdateProfileRequest struct {
	DisplayName         *string `json:"displayName"`
	Bio                 *string `json:"bio"`
	Avatar              *string `json:"avatar"`
	Background          *string `json:"background"`
	Location            *string `json:"location"`
	Website             *string `json:"website"`
	AIPromptPreferences *string `json:"aiPromptPreferences"`
}
