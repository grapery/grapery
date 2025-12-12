package service

import (
	"context"
	"fmt"
	"time"

	"github.com/grapestree/fgrapery/grapery/internal/domain"
	"go.uber.org/zap"
)

func (s *Service) FollowUser(ctx context.Context, followerID, followeeID string) error {
	if err := s.repo.FollowUser(ctx, followerID, followeeID); err != nil {
		return fmt.Errorf("failed to follow user: %w", err)
	}
	s.logger.Info("user followed", zap.String("follower", followerID), zap.String("followee", followeeID))

	// Create notification
	followee, _ := s.repo.UserByID(ctx, followeeID)
	follower, _ := s.repo.UserByID(ctx, followerID)
	if followee != nil && follower != nil {
		s.NotifyFollow(ctx, followeeID, followerID, follower.DisplayName, follower.Avatar)
	}

	return nil
}

func (s *Service) UnfollowUser(ctx context.Context, followerID, followeeID string) error {
	if err := s.repo.UnfollowUser(ctx, followerID, followeeID); err != nil {
		return fmt.Errorf("failed to unfollow user: %w", err)
	}
	s.logger.Info("user unfollowed", zap.String("follower", followerID), zap.String("followee", followeeID))
	return nil
}

func (s *Service) IsFollowing(ctx context.Context, followerID, followeeID string) (bool, error) {
	return s.repo.IsFollowing(ctx, followerID, followeeID)
}

func (s *Service) GetFollowers(ctx context.Context, userID string, limit, offset int) ([]*domain.User, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	return s.repo.Followers(ctx, userID, limit, offset)
}

func (s *Service) GetFollowing(ctx context.Context, userID string, limit, offset int) ([]*domain.User, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	return s.repo.Following(ctx, userID, limit, offset)
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

// UpdateUserProfile 更新用户资料
func (s *Service) UpdateUserProfile(ctx context.Context, userID string, req *UpdateProfileRequest) (*domain.User, error) {
	user, err := s.repo.UserByID(ctx, userID)
	if err != nil {
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
		s.logger.Error("failed to update user profile", zap.Error(err), zap.String("userId", userID))
		return nil, fmt.Errorf("failed to update user profile: %w", err)
	}

	s.logger.Info("user profile updated", zap.String("userId", userID))
	return user, nil
}

// UpdateUserAvatar 更新用户头像
func (s *Service) UpdateUserAvatar(ctx context.Context, userID, avatarURL string) error {
	user, err := s.repo.UserByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("user not found: %w", err)
	}

	user.Avatar = avatarURL
	if err := s.repo.UpdateUser(ctx, user); err != nil {
		s.logger.Error("failed to update user avatar", zap.Error(err), zap.String("userId", userID))
		return fmt.Errorf("failed to update user avatar: %w", err)
	}

	s.logger.Info("user avatar updated", zap.String("userId", userID))
	return nil
}

// UpdateUserBackground 更新用户背景图
func (s *Service) UpdateUserBackground(ctx context.Context, userID, backgroundURL string) error {
	user, err := s.repo.UserByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("user not found: %w", err)
	}

	user.Background = backgroundURL
	if err := s.repo.UpdateUser(ctx, user); err != nil {
		s.logger.Error("failed to update user background", zap.Error(err), zap.String("userId", userID))
		return fmt.Errorf("failed to update user background: %w", err)
	}

	s.logger.Info("user background updated", zap.String("userId", userID))
	return nil
}

// UserProfile 获取用户资料（对外API）
func (s *Service) UserProfile(ctx context.Context, userID string) (*domain.User, error) {
	user, err := s.repo.UserByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}
	return user, nil
}

// GetUser 获取用户信息（内部使用）
func (s *Service) GetUser(ctx context.Context, userID string) (*domain.User, error) {
	return s.repo.UserByID(ctx, userID)
}

// GetUserStories 获取用户的故事列表
func (s *Service) GetUserStories(ctx context.Context, userID string, limit, offset int) ([]*domain.Story, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	return s.repo.StoriesByAuthor(ctx, userID, limit, offset)
}

// GetUserCharacters 获取用户的角色列表
func (s *Service) GetUserCharacters(ctx context.Context, userID string, limit, offset int) ([]*domain.Character, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	return s.repo.CharactersByAuthor(ctx, userID, limit, offset)
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

// CreateUserActivity 创建用户活动记录
func (s *Service) CreateUserActivity(ctx context.Context, activity *domain.UserActivity) error {
	if err := s.repo.CreateUserActivity(ctx, activity); err != nil {
		s.logger.Error("failed to create user activity", zap.Error(err))
		return fmt.Errorf("failed to create user activity: %w", err)
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
