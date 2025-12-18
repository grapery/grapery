package service

import (
	"context"
	"fmt"
	"time"

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
func (s *Service) UserProfile(ctx context.Context, userID string) (*domain.User, error) {
	s.logger.Debug("getting user profile",
		zap.String("userID", userID))

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
			return &cachedUser, nil
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

	// 从数据库获取
	user, err := s.repo.UserByID(ctx, userID)
	if err != nil {
		s.logger.Error("failed to get user profile",
			zap.String("userID", userID),
			zap.Error(err))
		return nil, fmt.Errorf("user not found: %w", err)
	}

	// 写入缓存
	s.cacheUser(ctx, user)

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

	// 写入缓存
	s.cacheUser(ctx, user)

	return user, nil
}

// GetUserStories 获取用户的故事列表（带缓存）
func (s *Service) GetUserStories(ctx context.Context, userID string, limit, offset int) ([]*domain.Story, error) {
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
			return cachedStories, nil
		} else {
			s.logger.Debug("user stories cache miss",
				zap.String("userID", userID),
				zap.Error(err))
		}
	}

	// 从数据库获取
	stories, err := s.repo.StoriesByAuthor(ctx, userID, limit, offset)
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

	return stories, nil
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
	characters, err := s.repo.CharactersByAuthor(ctx, userID, limit, offset)
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

// GetLikedStories 获取用户点赞的故事
func (s *Service) GetLikedStories(ctx context.Context, userID string, limit, offset int) ([]*domain.Story, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	return s.repo.LikedStories(ctx, userID, limit, offset)
}

// GetLikedCharacters 获取用户点赞（关注）的角色
func (s *Service) GetLikedCharacters(ctx context.Context, userID string, limit, offset int) ([]*domain.Character, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	return s.repo.LikedCharacters(ctx, userID, limit, offset)
}

// GetLikedStoryboards 获取用户点赞的故事板
func (s *Service) GetLikedStoryboards(ctx context.Context, userID string, limit, offset int) ([]*domain.Storyboard, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	return s.repo.LikedStoryboards(ctx, userID, limit, offset)
}

// GetDraftStoryboards 获取用户的草稿故事板（未发布的）
func (s *Service) GetDraftStoryboards(ctx context.Context, userID string, limit, offset int) ([]*domain.Storyboard, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	return s.repo.DraftStoryboardsByCreator(ctx, userID, limit, offset)
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

// GetUserActivityList 获取用户活动列表
func (s *Service) GetUserActivityList(ctx context.Context, userID string, limit, offset int) ([]*domain.UserActivity, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	return s.repo.UserActivitiesByUserID(ctx, userID, limit, offset)
}

// CreateUserActivity 创建用户活动记录（带缓存失效）
func (s *Service) CreateUserActivity(ctx context.Context, activity *domain.UserActivity) error {
	if err := s.repo.CreateUserActivity(ctx, activity); err != nil {
		s.logger.Error("failed to create user activity",
			zap.String("userID", activity.UserID),
			zap.Error(err))
		return fmt.Errorf("failed to create user activity: %w", err)
	}

	// 使用户活动列表缓存失效
	c := s.getCache()
	if c != nil {
		for limit := 20; limit <= 100; limit += 20 {
			for offset := 0; offset < 200; offset += limit {
				_ = c.Delete(ctx, cache.UserActivitiesKey(activity.UserID, limit, offset))
			}
		}
		s.logger.Debug("user activities cache invalidated",
			zap.String("userID", activity.UserID))
	}

	return nil
}

// RecordStoryCreated 记录故事创建活动
func (s *Service) RecordStoryCreated(ctx context.Context, userID, storyID, storyTitle string) {
	activity := &domain.UserActivity{
		UserID:      userID,
		Type:        "story_created",
		TargetID:    storyID,
		TargetType:  "story",
		TargetTitle: storyTitle,
		Message:     "created story",
	}
	_ = s.CreateUserActivity(ctx, activity)
}

// RecordStoryPublished 记录故事发布活动
func (s *Service) RecordStoryPublished(ctx context.Context, userID, storyID, storyTitle string) {
	activity := &domain.UserActivity{
		UserID:      userID,
		Type:        "story_published",
		TargetID:    storyID,
		TargetType:  "story",
		TargetTitle: storyTitle,
		Message:     "published",
	}
	_ = s.CreateUserActivity(ctx, activity)
}

// RecordCharacterCreated 记录角色创建活动
func (s *Service) RecordCharacterCreated(ctx context.Context, userID, characterID, characterName string) {
	activity := &domain.UserActivity{
		UserID:      userID,
		Type:        "character_created",
		TargetID:    characterID,
		TargetType:  "character",
		TargetTitle: characterName,
		Message:     "created character",
	}
	_ = s.CreateUserActivity(ctx, activity)
}

// RecordUserFollowed 记录用户关注活动
func (s *Service) RecordUserFollowed(ctx context.Context, followerID, followeeID, followeeName string) {
	activity := &domain.UserActivity{
		UserID:      followerID,
		Type:        "user_followed",
		TargetID:    followeeID,
		TargetType:  "user",
		TargetTitle: followeeName,
		Message:     "followed",
	}
	_ = s.CreateUserActivity(ctx, activity)
}

// RecordStoryboardCreated 记录故事板创建活动
func (s *Service) RecordStoryboardCreated(ctx context.Context, userID, storyboardID, storyboardTitle string) {
	activity := &domain.UserActivity{
		UserID:      userID,
		Type:        "storyboard_created",
		TargetID:    storyboardID,
		TargetType:  "storyboard",
		TargetTitle: storyboardTitle,
		Message:     "created storyboard",
	}
	_ = s.CreateUserActivity(ctx, activity)
}

// ========== User Activity Heatmap ==========

// GetUserActivitiesWithFilter 获取带过滤条件的用户活动
func (s *Service) GetUserActivitiesWithFilter(ctx context.Context, userID string, timeRange domain.ActivityTimeRange, date string, limit, offset int) ([]*domain.UserActivity, int, error) {
	s.logger.Info("fetching user activities with filter",
		zap.String("userID", userID),
		zap.String("timeRange", string(timeRange)),
		zap.String("date", date),
		zap.Int("limit", limit))

	if limit <= 0 {
		limit = 50
	}
	if limit > 100 {
		limit = 100
	}

	var activities []*domain.UserActivity
	var err error

	// If specific date is provided, filter by that date
	if date != "" {
		activities, err = s.repo.UserActivitiesByDate(ctx, userID, date, limit, offset)
		if err != nil {
			s.logger.Error("failed to fetch user activities by date",
				zap.String("userID", userID),
				zap.String("date", date),
				zap.Error(err))
			return nil, 0, err
		}
	} else {
		// Filter by time range
		startTime, endTime := s.getTimeRangeBounds(timeRange)
		activities, err = s.repo.UserActivitiesByTimeRange(ctx, userID, startTime, endTime, limit, offset)
		if err != nil {
			s.logger.Error("failed to fetch user activities by time range",
				zap.String("userID", userID),
				zap.String("timeRange", string(timeRange)),
				zap.Error(err))
			return nil, 0, err
		}
	}

	s.logger.Info("successfully fetched user activities with filter",
		zap.String("userID", userID),
		zap.Int("count", len(activities)))

	return activities, len(activities), nil
}

// GetUserActivityHeatmap 获取用户活动热力图数据
func (s *Service) GetUserActivityHeatmap(ctx context.Context, userID string, timeRange domain.ActivityTimeRange) (*domain.ActivityHeatmapResponse, error) {
	s.logger.Info("fetching user activity heatmap",
		zap.String("userID", userID),
		zap.String("timeRange", string(timeRange)))

	startTime, endTime := s.getTimeRangeBounds(timeRange)

	heatmapData, err := s.repo.UserActivityHeatmap(ctx, userID, startTime, endTime)
	if err != nil {
		s.logger.Error("failed to fetch user activity heatmap",
			zap.String("userID", userID),
			zap.Error(err))
		return nil, err
	}

	// Calculate total count
	totalCount := 0
	for _, data := range heatmapData {
		totalCount += data.Count
	}

	// Fill in missing dates with zero counts
	heatmapData = s.fillMissingDates(heatmapData, startTime, endTime)

	response := &domain.ActivityHeatmapResponse{
		TimeRange:   timeRange,
		StartDate:   s.formatDate(startTime),
		EndDate:     s.formatDate(endTime),
		HeatmapData: make([]domain.ActivityHeatmapData, len(heatmapData)),
		TotalCount:  totalCount,
	}

	for i, data := range heatmapData {
		response.HeatmapData[i] = *data
	}

	s.logger.Info("successfully fetched user activity heatmap",
		zap.String("userID", userID),
		zap.Int("dataPoints", len(heatmapData)),
		zap.Int("totalCount", totalCount))

	return response, nil
}

// formatDate formats a Unix timestamp to YYYY-MM-DD string using China timezone
func (s *Service) formatDate(timestamp int64) string {
	return s.formatDateFromTime(timestamp)
}

func (s *Service) formatDateFromTime(timestamp int64) string {
	loc := s.getChinaTimezone()
	t := time.Unix(timestamp, 0).In(loc)
	return t.Format("2006-01-02")
}
