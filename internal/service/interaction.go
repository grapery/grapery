package service

import (
	"context"
	"fmt"
	"time"

	"github.com/grapestree/fgrapery/grapery/internal/cache"
	"github.com/grapestree/fgrapery/grapery/internal/domain"
	"github.com/grapestree/fgrapery/grapery/internal/utils"
	"go.uber.org/zap"
)

// InteractionService 互动服务接口
type InteractionService interface {
	// 关注相关
	Follow(ctx context.Context, followerID string, followableType domain.FollowableType, followableID string) error
	Unfollow(ctx context.Context, followerID string, followableType domain.FollowableType, followableID string) error
	CheckFollowStatus(ctx context.Context, followerID string, followableType domain.FollowableType, followableID string) (bool, error)
	GetFollowers(ctx context.Context, followableType domain.FollowableType, followableID string, page, pageSize int) ([]*domain.User, int, error)
	GetFollowing(ctx context.Context, userID string, followableType domain.FollowableType, page, pageSize int) ([]*domain.Follow, int, error)
	GetFollowersCount(ctx context.Context, followableType domain.FollowableType, followableID string) (int, error)

	// 点赞相关
	Like(ctx context.Context, userID string, likeableType domain.LikeableType, likeableID string) error
	Unlike(ctx context.Context, userID string, likeableType domain.LikeableType, likeableID string) error
	CheckLikeStatus(ctx context.Context, userID string, likeableType domain.LikeableType, likeableID string) (bool, error)
	GetLikes(ctx context.Context, likeableType domain.LikeableType, likeableID string, page, pageSize int) ([]*domain.User, int, error)
	GetLikesCount(ctx context.Context, likeableType domain.LikeableType, likeableID string) (int, error)

	// 批量检查
	BatchCheckFollowStatus(ctx context.Context, userID string, followableType domain.FollowableType, followableIDs []string) (map[string]bool, error)
	BatchCheckLikeStatus(ctx context.Context, userID string, likeableType domain.LikeableType, likeableIDs []string) (map[string]bool, error)

	// 收藏相关 (Bookmark - StoryCreationAppUI)
	CreateBookmark(ctx context.Context, userID string, bookmarkType domain.BookmarkType, bookmarkID string, collectionName string) (*domain.Bookmark, error)
	DeleteBookmark(ctx context.Context, userID string, bookmarkID string) error
	CheckBookmarkStatus(ctx context.Context, userID string, bookmarkType domain.BookmarkType, bookmarkID string) (bool, error)
	GetBookmarksByUser(ctx context.Context, userID string, bookmarkType domain.BookmarkType) ([]*domain.Bookmark, error)
	GetBookmarksCount(ctx context.Context, bookmarkType domain.BookmarkType, bookmarkID string) (int, error)
}

// interactionService 互动服务实现
type interactionService struct {
	followRepo    domain.FollowRepository
	likeRepo      domain.LikeRepository
	bookmarkRepo  domain.BookmarkRepository
	repo          domain.Repository
	logger        *zap.Logger
}

// NewInteractionService 创建互动服务
func NewInteractionService(
	followRepo domain.FollowRepository,
	likeRepo domain.LikeRepository,
	bookmarkRepo domain.BookmarkRepository,
	repo domain.Repository,
	logger *zap.Logger,
) InteractionService {
	return &interactionService{
		followRepo:   followRepo,
		likeRepo:     likeRepo,
		bookmarkRepo: bookmarkRepo,
		repo:         repo,
		logger:       logger,
	}
}

// Follow 关注
func (s *interactionService) Follow(ctx context.Context, followerID string, followableType domain.FollowableType, followableID string) error {
	s.logger.Info("following",
		zap.String("followerID", followerID),
		zap.String("followableType", string(followableType)),
		zap.String("followableID", followableID))

	// 1. 检查是否已关注
	exists, err := s.followRepo.CheckFollowStatus(ctx, followerID, followableType, followableID)
	if err != nil {
		s.logger.Error("failed to check follow status",
			zap.Error(err),
			zap.String("followerID", followerID),
			zap.String("followableID", followableID))
		return fmt.Errorf("failed to check follow status: %w", err)
	}
	if exists {
		return fmt.Errorf("already following")
	}

	// 2. 检查被关注对象是否存在
	if err := s.checkFollowableExists(ctx, followableType, followableID); err != nil {
		return err
	}

	// 3. 创建关注关系
	follow := &domain.Follow{
		ID:                   utils.GenerateID(),
		FollowerID:           followerID,
		FollowableType:       followableType,
		FollowableID:         followableID,
		NotificationsEnabled: true,
		CreatedAt:            time.Now().Unix(),
	}

	if err := s.followRepo.CreateFollow(ctx, follow); err != nil {
		s.logger.Error("failed to create follow",
			zap.Error(err),
			zap.String("followerID", followerID),
			zap.String("followableID", followableID))
		return fmt.Errorf("failed to create follow: %w", err)
	}

	s.logger.Info("followed successfully",
		zap.String("followerID", followerID),
		zap.String("followableType", string(followableType)),
		zap.String("followableID", followableID))

	return nil
}

// Unfollow 取消关注
func (s *interactionService) Unfollow(ctx context.Context, followerID string, followableType domain.FollowableType, followableID string) error {
	s.logger.Info("unfollowing",
		zap.String("followerID", followerID),
		zap.String("followableType", string(followableType)),
		zap.String("followableID", followableID))

	// 获取关注记录
	follows, err := s.followRepo.GetFollowsByFollower(ctx, followerID, followableType)
	if err != nil {
		s.logger.Error("failed to get follows by follower",
			zap.Error(err),
			zap.String("followerID", followerID))
		return fmt.Errorf("failed to get follows: %w", err)
	}

	for _, f := range follows {
		if f.FollowableID == followableID {
			if err := s.followRepo.DeleteFollow(ctx, f.ID); err != nil {
				s.logger.Error("failed to delete follow",
					zap.Error(err),
					zap.String("followID", f.ID))
				return fmt.Errorf("failed to delete follow: %w", err)
			}
			s.logger.Info("unfollowed successfully",
				zap.String("followerID", followerID),
				zap.String("followableID", followableID))
			return nil
		}
	}

	return fmt.Errorf("not following")
}

// CheckFollowStatus 检查关注状态
func (s *interactionService) CheckFollowStatus(ctx context.Context, followerID string, followableType domain.FollowableType, followableID string) (bool, error) {
	return s.followRepo.CheckFollowStatus(ctx, followerID, followableType, followableID)
}

// GetFollowers 获取粉丝列表
func (s *interactionService) GetFollowers(ctx context.Context, followableType domain.FollowableType, followableID string, page, pageSize int) ([]*domain.User, int, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}

	offset := (page - 1) * pageSize

	// 获取关注记录
	follows, err := s.followRepo.GetFollowsByFollowable(ctx, followableType, followableID)
	if err != nil {
		s.logger.Error("failed to get follows by followable",
			zap.Error(err),
			zap.String("followableType", string(followableType)),
			zap.String("followableID", followableID))
		return nil, 0, fmt.Errorf("failed to get followers: %w", err)
	}

	total := len(follows)

	// 分页
	if offset >= total {
		return []*domain.User{}, total, nil
	}

	end := offset + pageSize
	if end > total {
		end = total
	}
	follows = follows[offset:end]

	// 获取用户信息
	users := make([]*domain.User, 0, len(follows))
	for _, follow := range follows {
		user, err := s.repo.UserByID(ctx, follow.FollowerID)
		if err != nil {
			s.logger.Warn("failed to get follower user info",
				zap.Error(err),
				zap.String("userID", follow.FollowerID))
			continue
		}
		users = append(users, user)
	}

	return users, total, nil
}

// GetFollowing 获取关注列表
func (s *interactionService) GetFollowing(ctx context.Context, userID string, followableType domain.FollowableType, page, pageSize int) ([]*domain.Follow, int, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}

	offset := (page - 1) * pageSize

	// 获取关注记录
	follows, err := s.followRepo.GetFollowsByFollower(ctx, userID, followableType)
	if err != nil {
		s.logger.Error("failed to get follows by follower",
			zap.Error(err),
			zap.String("userID", userID))
		return nil, 0, fmt.Errorf("failed to get following: %w", err)
	}

	total := len(follows)

	// 分页
	if offset >= total {
		return []*domain.Follow{}, total, nil
	}

	end := offset + pageSize
	if end > total {
		end = total
	}
	follows = follows[offset:end]

	return follows, total, nil
}

// GetFollowersCount 获取粉丝数量
func (s *interactionService) GetFollowersCount(ctx context.Context, followableType domain.FollowableType, followableID string) (int, error) {
	return s.followRepo.GetFollowersCount(ctx, followableType, followableID)
}

// Like 点赞
func (s *interactionService) Like(ctx context.Context, userID string, likeableType domain.LikeableType, likeableID string) error {
	s.logger.Info("liking",
		zap.String("userID", userID),
		zap.String("likeableType", string(likeableType)),
		zap.String("likeableID", likeableID))

	// 1. 检查是否已点赞
	exists, err := s.likeRepo.CheckLikeStatus(ctx, userID, likeableType, likeableID)
	if err != nil {
		s.logger.Error("failed to check like status",
			zap.Error(err),
			zap.String("userID", userID),
			zap.String("likeableID", likeableID))
		return fmt.Errorf("failed to check like status: %w", err)
	}
	if exists {
		return fmt.Errorf("already liked")
	}

	// 2. 检查被点赞对象是否存在
	if err := s.checkLikeableExists(ctx, likeableType, likeableID); err != nil {
		return err
	}

	// 3. 创建点赞
	like := &domain.Like{
		ID:           utils.GenerateID(),
		UserID:       userID,
		LikeableType: likeableType,
		LikeableID:   likeableID,
		CreatedAt:    time.Now().Unix(),
	}

	if err := s.likeRepo.CreateLike(ctx, like); err != nil {
		s.logger.Error("failed to create like",
			zap.Error(err),
			zap.String("userID", userID),
			zap.String("likeableID", likeableID))
		return fmt.Errorf("failed to create like: %w", err)
	}

	s.logger.Info("liked successfully",
		zap.String("userID", userID),
		zap.String("likeableType", string(likeableType)),
		zap.String("likeableID", likeableID))

	return nil
}

// Unlike 取消点赞
func (s *interactionService) Unlike(ctx context.Context, userID string, likeableType domain.LikeableType, likeableID string) error {
	s.logger.Info("unliking",
		zap.String("userID", userID),
		zap.String("likeableType", string(likeableType)),
		zap.String("likeableID", likeableID))

	// 获取点赞记录
	likes, err := s.likeRepo.GetLikesByUser(ctx, userID, likeableType)
	if err != nil {
		s.logger.Error("failed to get likes by user",
			zap.Error(err),
			zap.String("userID", userID))
		return fmt.Errorf("failed to get likes: %w", err)
	}

	for _, l := range likes {
		if l.LikeableID == likeableID {
			if err := s.likeRepo.DeleteLike(ctx, l.ID); err != nil {
				s.logger.Error("failed to delete like",
					zap.Error(err),
					zap.String("likeID", l.ID))
				return fmt.Errorf("failed to delete like: %w", err)
			}
			s.logger.Info("unliked successfully",
				zap.String("userID", userID),
				zap.String("likeableID", likeableID))
			return nil
		}
	}

	return fmt.Errorf("not liked")
}

// CheckLikeStatus 检查点赞状态
func (s *interactionService) CheckLikeStatus(ctx context.Context, userID string, likeableType domain.LikeableType, likeableID string) (bool, error) {
	return s.likeRepo.CheckLikeStatus(ctx, userID, likeableType, likeableID)
}

// GetLikes 获取点赞用户列表
func (s *interactionService) GetLikes(ctx context.Context, likeableType domain.LikeableType, likeableID string, page, pageSize int) ([]*domain.User, int, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}

	offset := (page - 1) * pageSize

	// 获取点赞记录
	likes, err := s.likeRepo.GetLikesByLikeable(ctx, likeableType, likeableID)
	if err != nil {
		s.logger.Error("failed to get likes by likeable",
			zap.Error(err),
			zap.String("likeableType", string(likeableType)),
			zap.String("likeableID", likeableID))
		return nil, 0, fmt.Errorf("failed to get likes: %w", err)
	}

	total := len(likes)

	// 分页
	if offset >= total {
		return []*domain.User{}, total, nil
	}

	end := offset + pageSize
	if end > total {
		end = total
	}
	likes = likes[offset:end]

	// 获取用户信息
	users := make([]*domain.User, 0, len(likes))
	for _, like := range likes {
		user, err := s.repo.UserByID(ctx, like.UserID)
		if err != nil {
			s.logger.Warn("failed to get liker user info",
				zap.Error(err),
				zap.String("userID", like.UserID))
			continue
		}
		users = append(users, user)
	}

	return users, total, nil
}

// GetLikesCount 获取点赞数量
func (s *interactionService) GetLikesCount(ctx context.Context, likeableType domain.LikeableType, likeableID string) (int, error) {
	return s.likeRepo.GetLikesCount(ctx, likeableType, likeableID)
}

// BatchCheckFollowStatus 批量检查关注状态
func (s *interactionService) BatchCheckFollowStatus(ctx context.Context, userID string, followableType domain.FollowableType, followableIDs []string) (map[string]bool, error) {
	result := make(map[string]bool, len(followableIDs))

	for _, id := range followableIDs {
		status, err := s.followRepo.CheckFollowStatus(ctx, userID, followableType, id)
		if err != nil {
			s.logger.Warn("failed to check follow status in batch",
				zap.Error(err),
				zap.String("userID", userID),
				zap.String("followableID", id))
			result[id] = false
			continue
		}
		result[id] = status
	}

	return result, nil
}

// BatchCheckLikeStatus 批量检查点赞状态
func (s *interactionService) BatchCheckLikeStatus(ctx context.Context, userID string, likeableType domain.LikeableType, likeableIDs []string) (map[string]bool, error) {
	result := make(map[string]bool, len(likeableIDs))

	for _, id := range likeableIDs {
		status, err := s.likeRepo.CheckLikeStatus(ctx, userID, likeableType, id)
		if err != nil {
			s.logger.Warn("failed to check like status in batch",
				zap.Error(err),
				zap.String("userID", userID),
				zap.String("likeableID", id))
			result[id] = false
			continue
		}
		result[id] = status
	}

	return result, nil
}

// checkFollowableExists 检查被关注对象是否存在
func (s *interactionService) checkFollowableExists(ctx context.Context, followableType domain.FollowableType, followableID string) error {
	switch followableType {
	case domain.FollowableTypeStory:
		_, err := s.repo.StoryByID(ctx, followableID)
		if err != nil {
			return fmt.Errorf("story not found: %w", err)
		}
	case domain.FollowableTypeUser:
		_, err := s.repo.UserByID(ctx, followableID)
		if err != nil {
			return fmt.Errorf("user not found: %w", err)
		}
	case domain.FollowableTypeCharacter:
		_, err := s.repo.CharacterByID(ctx, followableID)
		if err != nil {
			return fmt.Errorf("character not found: %w", err)
		}
	default:
		return fmt.Errorf("invalid followable type: %s", followableType)
	}
	return nil
}

// checkLikeableExists 检查被点赞对象是否存在
func (s *interactionService) checkLikeableExists(ctx context.Context, likeableType domain.LikeableType, likeableID string) error {
	switch likeableType {
	case domain.LikeableTypeStory:
		_, err := s.repo.StoryByID(ctx, likeableID)
		if err != nil {
			return fmt.Errorf("story not found: %w", err)
		}
	case domain.LikeableTypeCharacter:
		_, err := s.repo.CharacterByID(ctx, likeableID)
		if err != nil {
			return fmt.Errorf("character not found: %w", err)
		}
	case domain.LikeableTypeStoryboardNode:
		_, err := s.repo.StoryboardByID(ctx, likeableID)
		if err != nil {
			return fmt.Errorf("storyboard not found: %w", err)
		}
	case domain.LikeableTypeFragment:
		// Fragment 检查需要额外的 repo 方法，这里先跳过具体检查
		// 实际项目中应该添加 FragmentRepository
		return nil
	case domain.LikeableTypeCharacterPoster:
		// CharacterPoster 检查
		return nil
	default:
		return fmt.Errorf("invalid likeable type: %s", likeableType)
	}
	return nil
}

// InvalidateFollowCache 使关注相关缓存失效
func (s *interactionService) InvalidateFollowCache(ctx context.Context, c cache.Cache, followerID string, followableType domain.FollowableType, followableID string) {
	if c == nil {
		return
	}
	// 清除关注者和被关注者的关注列表缓存
	for limit := 20; limit <= 100; limit += 20 {
		for offset := 0; offset < 200; offset += limit {
			_ = c.Delete(ctx, cache.UserFollowingKey(followerID)+fmt.Sprintf(":%d:%d", limit, offset))
			_ = c.Delete(ctx, cache.UserFollowersKey(followableID)+fmt.Sprintf(":%d:%d", limit, offset))
		}
	}
}

// InvalidateLikeCache 使点赞相关缓存失效
func (s *interactionService) InvalidateLikeCache(ctx context.Context, c cache.Cache, likeableType domain.LikeableType, likeableID string) {
	if c == nil {
		return
	}
	// 清除点赞列表缓存
	for limit := 20; limit <= 100; limit += 20 {
		for offset := 0; offset < 200; offset += limit {
			_ = c.Delete(ctx, fmt.Sprintf("likes:%s:%s:%d:%d", likeableType, likeableID, limit, offset))
		}
	}
}

// ========== Bookmark Methods (StoryCreationAppUI) ==========

// CreateBookmark 创建收藏
func (s *interactionService) CreateBookmark(ctx context.Context, userID string, bookmarkType domain.BookmarkType, bookmarkID string, collectionName string) (*domain.Bookmark, error) {
	// 检查是否已收藏
	isBookmarked, err := s.bookmarkRepo.CheckBookmarkStatus(ctx, userID, bookmarkType, bookmarkID)
	if err != nil {
		return nil, fmt.Errorf("failed to check bookmark status: %w", err)
	}
	if isBookmarked {
		return nil, domain.ErrAlreadyExists
	}

	// 创建收藏
	bookmark := &domain.Bookmark{
		UserID:         userID,
		BookmarkType:   bookmarkType,
		BookmarkID:     bookmarkID,
		CollectionName: collectionName,
		CreatedAt:      time.Now().Unix(),
	}

	if err := s.bookmarkRepo.CreateBookmark(ctx, bookmark); err != nil {
		return nil, fmt.Errorf("failed to create bookmark: %w", err)
	}

	// 更新收藏计数
	if err := s.bookmarkRepo.UpdateBookmarksCount(ctx, bookmarkType, bookmarkID, 1); err != nil {
		s.logger.Warn("failed to update bookmarks count", zap.Error(err))
	}

	return bookmark, nil
}

// DeleteBookmark 删除收藏
func (s *interactionService) DeleteBookmark(ctx context.Context, userID string, bookmarkID string) error {
	// 获取收藏信息
	bookmark, err := s.bookmarkRepo.GetBookmarkByID(ctx, bookmarkID)
	if err != nil {
		return fmt.Errorf("failed to get bookmark: %w", err)
	}

	// 验证所有权
	if bookmark.UserID != userID {
		return domain.ErrUnauthorized
	}

	// 删除收藏
	if err := s.bookmarkRepo.DeleteBookmark(ctx, bookmarkID); err != nil {
		return fmt.Errorf("failed to delete bookmark: %w", err)
	}

	// 更新收藏计数
	if err := s.bookmarkRepo.UpdateBookmarksCount(ctx, bookmark.BookmarkType, bookmark.BookmarkID, -1); err != nil {
		s.logger.Warn("failed to update bookmarks count", zap.Error(err))
	}

	return nil
}

// CheckBookmarkStatus 检查收藏状态
func (s *interactionService) CheckBookmarkStatus(ctx context.Context, userID string, bookmarkType domain.BookmarkType, bookmarkID string) (bool, error) {
	return s.bookmarkRepo.CheckBookmarkStatus(ctx, userID, bookmarkType, bookmarkID)
}

// GetBookmarksByUser 获取用户收藏列表
func (s *interactionService) GetBookmarksByUser(ctx context.Context, userID string, bookmarkType domain.BookmarkType) ([]*domain.Bookmark, error) {
	return s.bookmarkRepo.GetBookmarksByUser(ctx, userID, bookmarkType)
}

// GetBookmarksCount 获取收藏数量
func (s *interactionService) GetBookmarksCount(ctx context.Context, bookmarkType domain.BookmarkType, bookmarkID string) (int, error) {
	return s.bookmarkRepo.GetBookmarksCount(ctx, bookmarkType, bookmarkID)
}
