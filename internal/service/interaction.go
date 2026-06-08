package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/grapestree/fgrapery/grapery/internal/cache"
	"github.com/grapestree/fgrapery/grapery/internal/domain"
	"github.com/grapestree/fgrapery/grapery/internal/repository"
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
	GetUserBookmarks(ctx context.Context, ownerID, viewerID string, bookmarkType domain.BookmarkType) ([]*domain.Bookmark, error)
	GetBookmarksByUserPaged(ctx context.Context, userID string, bookmarkType domain.BookmarkType, page, limit int) ([]*domain.Bookmark, int64, bool, error)
	GetUserBookmarksPaged(ctx context.Context, ownerID, viewerID string, bookmarkType domain.BookmarkType, page, limit int) ([]*domain.Bookmark, int64, bool, error)
	GetBookmarksCount(ctx context.Context, bookmarkType domain.BookmarkType, bookmarkID string) (int, error)
}

// interactionService 互动服务实现
type interactionService struct {
	likeRepo          domain.LikeRepository
	bookmarkRepo      domain.BookmarkRepository
	repo              domain.Repository
	social            *Service
	fragmentLikeRepo  *repository.FragmentInteractionRepository
	logger            *zap.Logger
}

// NewInteractionService 创建互动服务
func NewInteractionService(
	likeRepo domain.LikeRepository,
	bookmarkRepo domain.BookmarkRepository,
	repo domain.Repository,
	social *Service,
	fragmentLikeRepo *repository.FragmentInteractionRepository,
	logger *zap.Logger,
) InteractionService {
	return &interactionService{
		likeRepo:         likeRepo,
		bookmarkRepo:     bookmarkRepo,
		repo:             repo,
		social:           social,
		fragmentLikeRepo: fragmentLikeRepo,
		logger:           logger,
	}
}

// Follow 关注
func (s *interactionService) Follow(ctx context.Context, followerID string, followableType domain.FollowableType, followableID string) error {
	s.logger.Info("following",
		zap.String("followerID", followerID),
		zap.String("followableType", string(followableType)),
		zap.String("followableID", followableID))

	if followableType == domain.FollowableTypeUser {
		if err := s.checkFollowableExists(ctx, followableType, followableID); err != nil {
			return err
		}
		if s.social == nil {
			return fmt.Errorf("interaction service: Service dependency required for user follow")
		}
		return s.social.FollowUser(ctx, followerID, followableID)
	}

	if followableType == domain.FollowableTypeStory {
		ok, err := s.repo.IsStoryFollowing(ctx, followerID, followableID)
		if err != nil {
			return fmt.Errorf("failed to check follow status: %w", err)
		}
		if ok {
			return fmt.Errorf("already following")
		}
		if err := s.checkFollowableExists(ctx, followableType, followableID); err != nil {
			return err
		}
		if err := s.repo.FollowStory(ctx, followerID, followableID); err != nil {
			if errors.Is(err, domain.ErrAlreadyExists) {
				return fmt.Errorf("already following")
			}
			return fmt.Errorf("failed to create follow: %w", err)
		}
		s.logger.Info("followed successfully",
			zap.String("followerID", followerID),
			zap.String("followableType", string(followableType)),
			zap.String("followableID", followableID))
		return nil
	}

	if followableType == domain.FollowableTypeCharacter {
		ok, err := s.repo.IsCharacterFollowing(ctx, followerID, followableID)
		if err != nil {
			return fmt.Errorf("failed to check follow status: %w", err)
		}
		if ok {
			return fmt.Errorf("already following")
		}
		if err := s.checkFollowableExists(ctx, followableType, followableID); err != nil {
			return err
		}
		if err := s.repo.FollowCharacter(ctx, followerID, followableID); err != nil {
			if errors.Is(err, domain.ErrAlreadyExists) {
				return fmt.Errorf("already following")
			}
			return fmt.Errorf("failed to create follow: %w", err)
		}
		s.logger.Info("followed successfully",
			zap.String("followerID", followerID),
			zap.String("followableType", string(followableType)),
			zap.String("followableID", followableID))
		return nil
	}

	return fmt.Errorf("invalid followable type: %s", followableType)
}

// Unfollow 取消关注
func (s *interactionService) Unfollow(ctx context.Context, followerID string, followableType domain.FollowableType, followableID string) error {
	s.logger.Info("unfollowing",
		zap.String("followerID", followerID),
		zap.String("followableType", string(followableType)),
		zap.String("followableID", followableID))

	if followableType == domain.FollowableTypeUser {
		if s.social == nil {
			return fmt.Errorf("interaction service: Service dependency required for user unfollow")
		}
		return s.social.UnfollowUser(ctx, followerID, followableID)
	}

	if followableType == domain.FollowableTypeStory {
		if err := s.repo.UnfollowStory(ctx, followerID, followableID); err != nil {
			if errors.Is(err, domain.ErrNotFound) {
				return fmt.Errorf("not following")
			}
			return fmt.Errorf("failed to delete follow: %w", err)
		}
		s.logger.Info("unfollowed successfully",
			zap.String("followerID", followerID),
			zap.String("followableID", followableID))
		return nil
	}

	if followableType == domain.FollowableTypeCharacter {
		if err := s.repo.UnfollowCharacter(ctx, followerID, followableID); err != nil {
			if errors.Is(err, domain.ErrNotFound) {
				return fmt.Errorf("not following")
			}
			return fmt.Errorf("failed to delete follow: %w", err)
		}
		s.logger.Info("unfollowed successfully",
			zap.String("followerID", followerID),
			zap.String("followableID", followableID))
		return nil
	}

	return fmt.Errorf("invalid followable type: %s", followableType)
}

// CheckFollowStatus 检查关注状态
func (s *interactionService) CheckFollowStatus(ctx context.Context, followerID string, followableType domain.FollowableType, followableID string) (bool, error) {
	if followableType == domain.FollowableTypeUser {
		return s.repo.IsFollowing(ctx, followerID, followableID)
	}
	if followableType == domain.FollowableTypeStory {
		return s.repo.IsStoryFollowing(ctx, followerID, followableID)
	}
	if followableType == domain.FollowableTypeCharacter {
		return s.repo.IsCharacterFollowing(ctx, followerID, followableID)
	}
	return false, fmt.Errorf("invalid followable type: %s", followableType)
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

	if followableType == domain.FollowableTypeUser {
		total64, err := s.repo.CountFollowersOfUser(ctx, followableID)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to count user followers: %w", err)
		}
		total := int(total64)
		if offset >= total {
			return []*domain.User{}, total, nil
		}
		users, err := s.repo.Followers(ctx, followableID, pageSize, offset)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to get user followers: %w", err)
		}
		return users, total, nil
	}

	var follows []*domain.Follow
	var total64 int64
	var err error
	switch followableType {
	case domain.FollowableTypeStory:
		total64, err = s.repo.CountFollowersOfStory(ctx, followableID)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to count story followers: %w", err)
		}
		follows, err = s.repo.ListStoryFollowRecordsByStory(ctx, followableID, pageSize, offset)
	case domain.FollowableTypeCharacter:
		total64, err = s.repo.CountFollowersOfCharacter(ctx, followableID)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to count character followers: %w", err)
		}
		follows, err = s.repo.ListCharacterFollowRecordsByCharacter(ctx, followableID, pageSize, offset)
	default:
		return nil, 0, fmt.Errorf("invalid followable type: %s", followableType)
	}
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get followers: %w", err)
	}

	total := int(total64)
	if offset >= total {
		return []*domain.User{}, total, nil
	}

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

	if followableType == domain.FollowableTypeUser {
		total64, err := s.repo.CountFollowingOfUser(ctx, userID)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to count user following: %w", err)
		}
		total := int(total64)
		if offset >= total {
			return []*domain.Follow{}, total, nil
		}
		follows, err := s.repo.ListUserFollowsByFollower(ctx, userID, pageSize, offset)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to get user following list: %w", err)
		}
		return follows, total, nil
	}

	var follows []*domain.Follow
	var total64 int64
	var err error
	switch followableType {
	case domain.FollowableTypeStory:
		total64, err = s.repo.CountStoriesFollowedByUser(ctx, userID)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to count story following: %w", err)
		}
		follows, err = s.repo.ListStoryFollowRecordsByUser(ctx, userID, pageSize, offset)
	case domain.FollowableTypeCharacter:
		total64, err = s.repo.CountCharactersFollowedByUser(ctx, userID)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to count character following: %w", err)
		}
		follows, err = s.repo.ListCharacterFollowRecordsByUser(ctx, userID, pageSize, offset)
	default:
		return nil, 0, fmt.Errorf("invalid followable type: %s", followableType)
	}
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get following: %w", err)
	}

	total := int(total64)
	if offset >= total {
		return []*domain.Follow{}, total, nil
	}

	return follows, total, nil
}

// GetFollowersCount 获取粉丝数量
func (s *interactionService) GetFollowersCount(ctx context.Context, followableType domain.FollowableType, followableID string) (int, error) {
	if followableType == domain.FollowableTypeUser {
		n, err := s.repo.CountFollowersOfUser(ctx, followableID)
		if err != nil {
			return 0, err
		}
		return int(n), nil
	}
	if followableType == domain.FollowableTypeStory {
		n, err := s.repo.CountFollowersOfStory(ctx, followableID)
		if err != nil {
			return 0, err
		}
		return int(n), nil
	}
	if followableType == domain.FollowableTypeCharacter {
		n, err := s.repo.CountFollowersOfCharacter(ctx, followableID)
		if err != nil {
			return 0, err
		}
		return int(n), nil
	}
	return 0, fmt.Errorf("invalid followable type: %s", followableType)
}

// Like 点赞
func (s *interactionService) Like(ctx context.Context, userID string, likeableType domain.LikeableType, likeableID string) error {
	s.logger.Info("liking",
		zap.String("userID", userID),
		zap.String("likeableType", string(likeableType)),
		zap.String("likeableID", likeableID))

	// 故事板点赞走 Service.LikeStoryboard：与 POST /storyboards/:id/like 同逻辑（缓存失效、通知等）
	if likeableType == domain.LikeableTypeStoryboardNode {
		if err := s.checkLikeableExists(ctx, likeableType, likeableID); err != nil {
			return err
		}
		if s.social == nil {
			return fmt.Errorf("interaction service: storyboard like requires Service dependency")
		}
		if err := s.social.LikeStoryboard(ctx, userID, likeableID); err != nil {
			if errors.Is(err, domain.ErrAlreadyLiked) {
				return nil
			}
			return err
		}
		return nil
	}

	if likeableType == domain.LikeableTypeFragment {
		if s.fragmentLikeRepo == nil {
			return fmt.Errorf("interaction service: fragment like requires FragmentInteractionRepository")
		}
		frag, err := s.repo.FragmentByID(ctx, likeableID)
		if err != nil || frag == nil {
			return fmt.Errorf("fragment not found: %w", err)
		}
		exists, err := s.fragmentLikeRepo.IsLiked(ctx, likeableID, userID)
		if err != nil {
			return fmt.Errorf("failed to check fragment like status: %w", err)
		}
		if exists {
			s.logger.Info("already liked fragment (idempotent)",
				zap.String("userID", userID),
				zap.String("likeableID", likeableID))
			return nil
		}
		fl := &domain.FragmentLike{
			ID:         uuid.New().String(),
			FragmentID: likeableID,
			UserID:     userID,
		}
		if err := s.fragmentLikeRepo.CreateLike(ctx, fl); err != nil {
			errL := strings.ToLower(err.Error())
			if strings.Contains(errL, "duplicate") || strings.Contains(errL, "unique") {
				return nil
			}
			return fmt.Errorf("failed to like fragment: %w", err)
		}
		if s.social != nil && frag.UserID != "" && frag.UserID != userID {
			liker, uerr := s.social.GetUser(ctx, userID)
			if uerr == nil && liker != nil {
				if nerr := s.social.NotifyLike(ctx, frag.UserID, userID, liker.DisplayName, liker.Avatar, "fragment", likeableID, ""); nerr != nil {
					s.logger.Warn("fragment polymorphic like notification failed", zap.Error(nerr), zap.String("fragmentId", likeableID))
				}
			}
		}
		s.logger.Info("liked fragment via interaction service",
			zap.String("userID", userID),
			zap.String("likeableID", likeableID))
		return nil
	}

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
		s.logger.Info("already liked (idempotent)",
			zap.String("userID", userID),
			zap.String("likeableType", string(likeableType)),
			zap.String("likeableID", likeableID))
		return nil
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
		errL := strings.ToLower(err.Error())
		if strings.Contains(errL, "duplicate") || strings.Contains(errL, "unique") {
			s.logger.Info("like create duplicate (race), idempotent ok",
				zap.String("userID", userID),
				zap.String("likeableType", string(likeableType)),
				zap.String("likeableID", likeableID))
			return nil
		}
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

	if likeableType == domain.LikeableTypeStoryboardNode {
		if s.social == nil {
			return fmt.Errorf("interaction service: storyboard unlike requires Service dependency")
		}
		if err := s.social.UnlikeStoryboard(ctx, userID, likeableID); err != nil {
			if errors.Is(err, domain.ErrNotFound) {
				return nil
			}
			return err
		}
		return nil
	}

	if likeableType == domain.LikeableTypeFragment {
		if s.fragmentLikeRepo == nil {
			return fmt.Errorf("interaction service: fragment unlike requires FragmentInteractionRepository")
		}
		if err := s.fragmentLikeRepo.DeleteLike(ctx, likeableID, userID); err != nil {
			return fmt.Errorf("failed to unlike fragment: %w", err)
		}
		s.logger.Info("unliked fragment via interaction service",
			zap.String("userID", userID),
			zap.String("likeableID", likeableID))
		return nil
	}

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

	s.logger.Info("not liked (idempotent unlike)",
		zap.String("userID", userID),
		zap.String("likeableType", string(likeableType)),
		zap.String("likeableID", likeableID))
	return nil
}

// CheckLikeStatus 检查点赞状态
func (s *interactionService) CheckLikeStatus(ctx context.Context, userID string, likeableType domain.LikeableType, likeableID string) (bool, error) {
	if likeableType == domain.LikeableTypeStoryboardNode {
		return s.repo.IsStoryboardLiked(ctx, userID, likeableID)
	}
	if likeableType == domain.LikeableTypeFragment {
		if s.fragmentLikeRepo == nil {
			return false, fmt.Errorf("interaction service: fragment like check requires FragmentInteractionRepository")
		}
		return s.fragmentLikeRepo.IsLiked(ctx, likeableID, userID)
	}
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

	if likeableType == domain.LikeableTypeStoryboardNode {
		users, total, err := s.repo.ListStoryboardLikers(ctx, likeableID, pageSize, offset)
		if err != nil {
			s.logger.Error("failed to list storyboard likers",
				zap.Error(err),
				zap.String("likeableID", likeableID))
			return nil, 0, fmt.Errorf("failed to get likes: %w", err)
		}
		return users, total, nil
	}

	if likeableType == domain.LikeableTypeFragment {
		if s.fragmentLikeRepo == nil {
			return nil, 0, fmt.Errorf("interaction service: fragment likes list requires FragmentInteractionRepository")
		}
		flikes, total, err := s.fragmentLikeRepo.GetFragmentLikes(ctx, likeableID, pageSize, offset)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to get fragment likes: %w", err)
		}
		users := make([]*domain.User, 0, len(flikes))
		for _, l := range flikes {
			u, uerr := s.repo.UserByID(ctx, l.UserID)
			if uerr != nil {
				s.logger.Warn("failed to get fragment liker user",
					zap.Error(uerr),
					zap.String("userID", l.UserID))
				continue
			}
			if u != nil {
				users = append(users, u)
			}
		}
		return users, int(total), nil
	}

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
	if likeableType == domain.LikeableTypeStoryboardNode {
		sb, err := s.repo.StoryboardByID(ctx, likeableID)
		if err != nil {
			return 0, err
		}
		return sb.Likes, nil
	}
	if likeableType == domain.LikeableTypeFragment {
		if s.fragmentLikeRepo == nil {
			return 0, fmt.Errorf("interaction service: fragment likes count requires FragmentInteractionRepository")
		}
		n, err := s.fragmentLikeRepo.GetLikesCount(ctx, likeableID)
		if err != nil {
			return 0, err
		}
		return int(n), nil
	}
	return s.likeRepo.GetLikesCount(ctx, likeableType, likeableID)
}

// BatchCheckFollowStatus 批量检查关注状态
func (s *interactionService) BatchCheckFollowStatus(ctx context.Context, userID string, followableType domain.FollowableType, followableIDs []string) (map[string]bool, error) {
	result := make(map[string]bool, len(followableIDs))

	if followableType == domain.FollowableTypeUser {
		for _, id := range followableIDs {
			ok, err := s.repo.IsFollowing(ctx, userID, id)
			if err != nil {
				s.logger.Warn("failed to check user follow status in batch",
					zap.Error(err),
					zap.String("userID", userID),
					zap.String("followableID", id))
				result[id] = false
				continue
			}
			result[id] = ok
		}
		return result, nil
	}

	for _, id := range followableIDs {
		var status bool
		var err error
		switch followableType {
		case domain.FollowableTypeStory:
			status, err = s.repo.IsStoryFollowing(ctx, userID, id)
		case domain.FollowableTypeCharacter:
			status, err = s.repo.IsCharacterFollowing(ctx, userID, id)
		default:
			err = fmt.Errorf("invalid followable type: %s", followableType)
		}
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

	if likeableType == domain.LikeableTypeStoryboardNode {
		for _, id := range likeableIDs {
			ok, err := s.repo.IsStoryboardLiked(ctx, userID, id)
			if err != nil {
				s.logger.Warn("failed to check storyboard like status in batch",
					zap.Error(err),
					zap.String("userID", userID),
					zap.String("likeableID", id))
				result[id] = false
				continue
			}
			result[id] = ok
		}
		return result, nil
	}

	if likeableType == domain.LikeableTypeFragment {
		if s.fragmentLikeRepo == nil {
			return nil, fmt.Errorf("interaction service: fragment batch like requires FragmentInteractionRepository")
		}
		for _, id := range likeableIDs {
			ok, err := s.fragmentLikeRepo.IsLiked(ctx, id, userID)
			if err != nil {
				result[id] = false
				continue
			}
			result[id] = ok
		}
		return result, nil
	}

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
		frag, err := s.repo.FragmentByID(ctx, likeableID)
		if err != nil || frag == nil {
			return fmt.Errorf("fragment not found: %w", err)
		}
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
	if followableType != domain.FollowableTypeUser || s.social == nil {
		return
	}
	_ = c
	s.social.invalidateUserFollowListCaches(ctx, followerID, followableID)
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

// GetBookmarksByUserPaged 获取当前用户收藏列表（分页）
func (s *interactionService) GetBookmarksByUserPaged(ctx context.Context, userID string, bookmarkType domain.BookmarkType, page, limit int) ([]*domain.Bookmark, int64, bool, error) {
	if page <= 0 {
		page = 1
	}
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	offset := (page - 1) * limit

	bookmarks, total, err := s.bookmarkRepo.GetBookmarksByUserPaginated(ctx, userID, bookmarkType, limit, offset)
	if err != nil {
		return nil, 0, false, err
	}
	hasMore := int64(offset+len(bookmarks)) < total
	return bookmarks, total, hasMore, nil
}

// GetUserBookmarks 获取用户主页可见的收藏列表（遵守被访问用户的公开设置）
func (s *interactionService) GetUserBookmarks(ctx context.Context, ownerID, viewerID string, bookmarkType domain.BookmarkType) ([]*domain.Bookmark, error) {
	bookmarks, err := s.bookmarkRepo.GetBookmarksByUser(ctx, ownerID, bookmarkType)
	if err != nil {
		return nil, err
	}

	if viewerID == ownerID {
		return bookmarks, nil
	}

	settings, err := s.repo.UserSettings(ctx, ownerID)
	if err == nil && settings != nil && !settings.ShowPublicBookmarks {
		return []*domain.Bookmark{}, nil
	}

	isFollower := false
	if viewerID != "" {
		if following, followErr := s.repo.IsFollowing(ctx, viewerID, ownerID); followErr == nil {
			isFollower = following
		}
	}

	filtered := make([]*domain.Bookmark, 0, len(bookmarks))
	for _, bm := range bookmarks {
		if bm == nil {
			continue
		}
		visible, visErr := s.isBookmarkVisibleToViewer(ctx, bm, ownerID, viewerID, isFollower)
		if visErr != nil {
			continue
		}
		if visible {
			filtered = append(filtered, bm)
		}
	}
	return filtered, nil
}

// GetUserBookmarksPaged 获取用户主页可见收藏列表（分页）
func (s *interactionService) GetUserBookmarksPaged(ctx context.Context, ownerID, viewerID string, bookmarkType domain.BookmarkType, page, limit int) ([]*domain.Bookmark, int64, bool, error) {
	if page <= 0 {
		page = 1
	}
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	offset := (page - 1) * limit

	bookmarks, total, err := s.bookmarkRepo.GetBookmarksByUserPaginated(ctx, ownerID, bookmarkType, limit, offset)
	if err != nil {
		return nil, 0, false, err
	}
	if viewerID == ownerID {
		hasMore := int64(offset+len(bookmarks)) < total
		return bookmarks, total, hasMore, nil
	}

	settings, err := s.repo.UserSettings(ctx, ownerID)
	if err == nil && settings != nil && !settings.ShowPublicBookmarks {
		return []*domain.Bookmark{}, 0, false, nil
	}

	isFollower := false
	if viewerID != "" {
		if following, followErr := s.repo.IsFollowing(ctx, viewerID, ownerID); followErr == nil {
			isFollower = following
		}
	}

	filtered := make([]*domain.Bookmark, 0, len(bookmarks))
	for _, bm := range bookmarks {
		if bm == nil {
			continue
		}
		visible, visErr := s.isBookmarkVisibleToViewer(ctx, bm, ownerID, viewerID, isFollower)
		if visErr != nil {
			continue
		}
		if visible {
			filtered = append(filtered, bm)
		}
	}
	hasMore := int64(offset+len(bookmarks)) < total
	return filtered, total, hasMore, nil
}

// GetBookmarksCount 获取收藏数量
func (s *interactionService) GetBookmarksCount(ctx context.Context, bookmarkType domain.BookmarkType, bookmarkID string) (int, error) {
	return s.bookmarkRepo.GetBookmarksCount(ctx, bookmarkType, bookmarkID)
}

func (s *interactionService) isBookmarkVisibleToViewer(ctx context.Context, bm *domain.Bookmark, ownerID, viewerID string, isFollower bool) (bool, error) {
	switch bm.BookmarkType {
	case domain.BookmarkTypeStory:
		story, err := s.repo.StoryByID(ctx, bm.BookmarkID)
		if err != nil || story == nil {
			return false, err
		}
		return isStoryVisibleToViewer(story.Visibility, ownerID, viewerID, isFollower), nil
	case domain.BookmarkTypeStoryboard:
		sb, err := s.repo.StoryboardByID(ctx, bm.BookmarkID)
		if err != nil || sb == nil {
			return false, err
		}
		story, err := s.repo.StoryByID(ctx, sb.StoryID)
		if err != nil || story == nil {
			return false, err
		}
		return isStoryVisibleToViewer(story.Visibility, ownerID, viewerID, isFollower), nil
	case domain.BookmarkTypeFragment:
		fragment, err := s.repo.FragmentByID(ctx, bm.BookmarkID)
		if err != nil || fragment == nil {
			return false, err
		}
		return isFragmentVisibleToViewer(fragment.Visibility, ownerID, viewerID, isFollower), nil
	default:
		return false, nil
	}
}

func isStoryVisibleToViewer(visibility, ownerID, viewerID string, isFollower bool) bool {
	if viewerID == ownerID {
		return true
	}
	switch visibility {
	case string(domain.StoryVisibilityPrivate):
		return false
	case string(domain.StoryVisibilityFollowers):
		return isFollower
	default:
		return true
	}
}

func isFragmentVisibleToViewer(visibility, ownerID, viewerID string, isFollower bool) bool {
	if viewerID == ownerID {
		return true
	}
	switch visibility {
	case string(domain.FragmentVisibilityPrivate):
		return false
	case string(domain.FragmentVisibilityFollowers), string(domain.FragmentVisibilityFollowersLegacy):
		return isFollower
	default:
		return true
	}
}
