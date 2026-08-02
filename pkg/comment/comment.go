package comment

import (
	"context"
	"encoding/json"
	"fmt"

	"go.uber.org/zap"
	"gorm.io/gorm"

	api "github.com/grapery/common-protoc/gen"
	"github.com/grapery/grapery/models"
	llmchatservice "github.com/grapery/grapery/service/llmchat"
	"github.com/grapery/grapery/utils/cache"
)

var logger, _ = zap.NewDevelopment()

var commentServer CommentServer

// hasPermissionToDelete 检查用户是否有权限删除评论
func (s *CommentService) hasPermissionToDelete(ctx context.Context, userID int64, commentID int64) bool {
	// 获取评论信息
	var comment models.Comment
	if err := models.DataBase().WithContext(ctx).Where("id = ?", commentID).First(&comment).Error; err != nil {
		logger.Error("failed to get comment for permission check", zap.Error(err))
		return false
	}

	// 只有评论作者可以删除自己的评论
	return comment.UserID == userID
}

// HasPermissionToDelete 实现接口方法，供外部调用
func (s *CommentService) HasPermissionToDelete(ctx context.Context, userID int64, commentID int64) bool {
	return s.hasPermissionToDelete(ctx, userID, commentID)
}

func init() {
	commentServer = NewCommentService()
}

func GetCommentServer() CommentServer {
	return commentServer
}

type CommentServer interface {
	CreateStoryComment(ctx context.Context, req *api.CreateStoryCommentRequest) (*api.CreateStoryCommentResponse, error)
	GetStoryComments(ctx context.Context, req *api.GetStoryCommentsRequest) (*api.GetStoryCommentsResponse, error)
	DeleteStoryComment(ctx context.Context, req *api.DeleteStoryCommentRequest) (*api.DeleteStoryCommentResponse, error)
	GetStoryCommentReplies(ctx context.Context, req *api.GetStoryCommentRepliesRequest) (*api.GetStoryCommentRepliesResponse, error)
	CreateStoryCommentReply(ctx context.Context, req *api.CreateStoryCommentReplyRequest) (*api.CreateStoryCommentReplyResponse, error)
	DeleteStoryCommentReply(ctx context.Context, req *api.DeleteStoryCommentReplyRequest) (*api.DeleteStoryCommentReplyResponse, error)
	CreateStoryBoardComment(ctx context.Context, req *api.CreateStoryBoardCommentRequest) (*api.CreateStoryBoardCommentResponse, error)
	DeleteStoryBoardComment(ctx context.Context, req *api.DeleteStoryBoardCommentRequest) (*api.DeleteStoryBoardCommentResponse, error)
	GetStoryBoardComments(ctx context.Context, req *api.GetStoryBoardCommentsRequest) (*api.GetStoryBoardCommentsResponse, error)
	GetStoryBoardCommentReplies(ctx context.Context, req *api.GetStoryBoardCommentRepliesRequest) (*api.GetStoryBoardCommentRepliesResponse, error)
	LikeComment(ctx context.Context, req *api.LikeCommentRequest) (*api.LikeCommentResponse, error)
	DislikeComment(ctx context.Context, req *api.DislikeCommentRequest) (*api.DislikeCommentResponse, error)
	HasPermissionToDelete(ctx context.Context, userID int64, commentID int64) bool
}

type CommentService struct {
	cache *cache.CommentCache
}

func NewCommentService() *CommentService {
	return &CommentService{
		cache: cache.GetCommentCache(),
	}
}

func (s *CommentService) CreateStoryComment(ctx context.Context, req *api.CreateStoryCommentRequest) (*api.CreateStoryCommentResponse, error) {
	logger.Info("CreateStoryComment called", zap.Any("req", req))
	// 参数校验：内容判空，StoryId合法性
	if req.GetContent() == "" {
		logger.Error("CreateStoryComment failed: content is empty")
		return &api.CreateStoryCommentResponse{
			Code:    api.ResponseCode_INVALID_PARAMETER,
			Message: "content is empty",
		}, nil
	}
	if req.GetStoryId() <= 0 {
		logger.Error("CreateStoryComment failed: invalid story id", zap.Int64("story_id", req.GetStoryId()))
		return &api.CreateStoryCommentResponse{
			Code:    api.ResponseCode_INVALID_PARAMETER,
			Message: "invalid story id",
		}, nil
	}

	// 使用事务确保数据一致性
	var commentID uint
	err := models.DataBase().Transaction(func(tx *gorm.DB) error {
		comment := &models.Comment{
			UserID:       req.GetUserId(),
			StoryID:      req.GetStoryId(),
			Content:      []byte(req.GetContent()),
			CommentType:  models.CommentTypeComment,
			Status:       1, // 建议用常量
			LikeCount:    0,
			DislikeCount: 0,
		}

		// 创建评论
		if err := tx.WithContext(ctx).Create(comment).Error; err != nil {
			logger.Error("CreateStoryComment failed: create comment error", zap.Error(err))
			return err
		}
		commentID = comment.ID

		// 更新故事评论计数
		if err := tx.Model(&models.Story{}).
			WithContext(ctx).
			Where("id = ?", req.GetStoryId()).
			Update("comment_count", gorm.Expr("comment_count + 1")).Error; err != nil {
			logger.Error("CreateStoryComment failed: increase story comment count error", zap.Error(err))
			return err
		}

		return nil
	})

	if err != nil {
		logger.Error("CreateStoryComment failed: transaction error", zap.Error(err))
		return &api.CreateStoryCommentResponse{
			Code:    api.ResponseCode_OPERATION_FAILED,
			Message: err.Error(),
		}, nil
	}

	logger.Info("CreateStoryComment success", zap.Uint("comment_id", commentID), zap.Int64("story_id", req.GetStoryId()), zap.Int64("user_id", req.GetUserId()))

	// Invalidate cache for this story's comments
	go func() {
		if cacheErr := s.cache.InvalidateStoryCommentsCache(context.Background(), req.GetStoryId()); cacheErr != nil {
			logger.Error("Failed to invalidate story comments cache", zap.Error(cacheErr))
		}
	}()

	// Create system notification for story author
	go func() {
		bgCtx := context.Background()
		story, err := models.GetStory(bgCtx, req.GetStoryId())
		if err != nil {
			logger.Error("Failed to get story for notification", zap.Error(err))
			return
		}
		// Don't notify if user comments on their own story
		if story.CreatorID == req.GetUserId() {
			return
		}
		commentID64 := int64(commentID)
		storyID64 := req.GetStoryId()
		userID64 := req.GetUserId()
		notifErr := llmchatservice.CreateSystemNotification(bgCtx, llmchatservice.CreateNotificationParams{
			UserID:           story.CreatorID,
			Type:             models.SystemNotificationTypeComment,
			Title:            "新评论提醒",
			Content:          fmt.Sprintf("有人评论了你的故事《%s》", story.Title),
			RelatedUserID:    &userID64,
			RelatedStoryID:   &storyID64,
			RelatedCommentID: &commentID64,
		})
		if notifErr != nil {
			logger.Error("Failed to create comment notification", zap.Error(notifErr))
		}
	}()

	return &api.CreateStoryCommentResponse{
		Code:    api.ResponseCode_OK,
		Message: "success",
	}, nil
}

func (s *CommentService) GetStoryComments(ctx context.Context, req *api.GetStoryCommentsRequest) (*api.GetStoryCommentsResponse, error) {
	// Try to get from cache first
	cachedComments, err := s.cache.GetCachedStoryComments(ctx, req.GetStoryId(), int64(req.GetOffset()), int64(req.GetPageSize()))
	if err == nil && cachedComments != nil {
		logger.Info("GetStoryComments: cache hit", zap.Int64("story_id", req.GetStoryId()))
		return &api.GetStoryCommentsResponse{
			Code:     api.ResponseCode_OK,
			Message:  "success",
			Total:    int64(len(cachedComments)),
			Comments: cachedComments,
		}, nil
	}

	logger.Info("GetStoryComments: cache miss, querying database", zap.Int64("story_id", req.GetStoryId()))
	comments, err := models.GetCommentByStory(ctx, uint64(req.GetStoryId()),
		models.CommentTypeComment, int64(req.GetOffset()), int64(req.GetPageSize()))
	if err != nil {
		return nil, err
	}
	if len(*comments) == 0 {
		return &api.GetStoryCommentsResponse{
			Code:     api.ResponseCode_OK,
			Message:  "success",
			Total:    0,
			Comments: []*api.StoryComment{},
		}, nil
	}
	apiComments := make([]*api.StoryComment, 0)
	for _, comment := range *comments {
		apiComments = append(apiComments, &api.StoryComment{
			CommentId: int64(comment.ID),
			Content:   string(comment.Content),
			CreatedAt: comment.CreateAt.Unix(),
			UpdatedAt: comment.UpdateAt.Unix(),
			UserId:    comment.UserID,
			StoryId:   comment.StoryID,
			LikeCount: comment.LikeCount,
		})
	}

	// Cache the result
	go func() {
		if cacheErr := s.cache.CacheStoryComments(context.Background(), req.GetStoryId(), int64(req.GetOffset()), int64(req.GetPageSize()), apiComments); cacheErr != nil {
			logger.Error("Failed to cache story comments", zap.Error(cacheErr))
		}
	}()

	return &api.GetStoryCommentsResponse{
		Code:     api.ResponseCode_OK,
		Message:  "success",
		Total:    int64(len(*comments)),
		Comments: apiComments,
	}, nil
}

func (s *CommentService) DeleteStoryComment(ctx context.Context, req *api.DeleteStoryCommentRequest) (*api.DeleteStoryCommentResponse, error) {
	// 权限验证：只有评论作者可以删除自己的评论
	if !s.hasPermissionToDelete(ctx, req.GetUserId(), req.GetCommentId()) {
		logger.Warn("DeleteStoryComment failed: permission denied",
			zap.Int64("user_id", req.GetUserId()),
			zap.Int64("comment_id", req.GetCommentId()))
		return &api.DeleteStoryCommentResponse{
			Code:    api.ResponseCode_PERMISSION_DENIED,
			Message: "no permission to delete comment",
		}, nil
	}

	// 使用事务确保数据一致性
	var storyID int64
	err := models.DataBase().Transaction(func(tx *gorm.DB) error {
		// 先获取评论信息
		var comment models.Comment
		if err := tx.WithContext(ctx).Where("id = ?", req.GetCommentId()).First(&comment).Error; err != nil {
			logger.Error("DeleteStoryComment failed: get comment error", zap.Error(err))
			return err
		}
		storyID = comment.StoryID

		// 软删除评论
		if err := tx.WithContext(ctx).Model(&models.Comment{}).
			Where("id = ?", req.GetCommentId()).
			Update("deleted", 1).Error; err != nil {
			logger.Error("DeleteStoryComment failed: delete comment error", zap.Error(err))
			return err
		}

		// 更新故事评论计数
		if err := tx.Model(&models.Story{}).
			WithContext(ctx).
			Where("id = ?", storyID).
			Update("comment_count", gorm.Expr("comment_count - 1")).Error; err != nil {
			logger.Error("DeleteStoryComment failed: decrease story comment count error", zap.Error(err))
			return err
		}

		return nil
	})

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return &api.DeleteStoryCommentResponse{
				Code:    api.ResponseCode_COMMENT_NOT_FOUND,
				Message: "comment not found",
			}, nil
		}
		return &api.DeleteStoryCommentResponse{
			Code:    api.ResponseCode_OPERATION_FAILED,
			Message: err.Error(),
		}, nil
	}

	// 失效相关缓存
	go func() {
		if cacheErr := s.cache.InvalidateStoryCommentsCache(context.Background(), storyID); cacheErr != nil {
			logger.Error("Failed to invalidate story comments cache", zap.Error(cacheErr))
		}
	}()

	return &api.DeleteStoryCommentResponse{
		Code:    api.ResponseCode_OK,
		Message: "success",
	}, nil
}

func (s *CommentService) GetStoryCommentReplies(ctx context.Context, req *api.GetStoryCommentRepliesRequest) (*api.GetStoryCommentRepliesResponse, error) {
	// Try to get from cache first
	cachedReplies, err := s.cache.GetCachedCommentReplies(ctx, req.GetCommentId())
	if err == nil && cachedReplies != nil {
		logger.Info("GetStoryCommentReplies: cache hit", zap.Int64("comment_id", req.GetCommentId()))
		return &api.GetStoryCommentRepliesResponse{
			Code:    api.ResponseCode_OK,
			Message: "success",
			Total:   int64(len(cachedReplies)),
			Replies: cachedReplies,
		}, nil
	}

	logger.Info("GetStoryCommentReplies: cache miss, querying database", zap.Int64("comment_id", req.GetCommentId()))
	replies, err := models.GetStoryCommentReplies(ctx, uint64(req.GetCommentId()))
	if err != nil {
		return nil, err
	}
	if len(*replies) == 0 {
		return &api.GetStoryCommentRepliesResponse{
			Code:    api.ResponseCode_OK,
			Message: "success",
			Total:   0,
			Replies: []*api.StoryComment{},
		}, nil
	}
	createrIds := make([]int64, 0)
	for _, comment := range *replies {
		createrIds = append(createrIds, comment.UserID)
	}
	createrMap, err := models.GetUsersByIdsMap(ctx, createrIds)
	if err != nil {
		logger.Error("get user by ids map error", zap.Error(err))
		return nil, err
	}
	apiReplies := make([]*api.StoryComment, 0)
	for _, reply := range *replies {
		apiReplies = append(apiReplies, &api.StoryComment{
			CommentId:  int64(reply.ID),
			Content:    string(reply.Content),
			CreatedAt:  reply.CreateAt.Unix(),
			UpdatedAt:  reply.UpdateAt.Unix(),
			UserId:     reply.UserID,
			StoryId:    reply.StoryID,
			LikeCount:  reply.LikeCount,
			ReplyCount: reply.ReplyCount,
			Creator: &api.UserInfo{
				UserId: reply.UserID,
				Name:   createrMap[int(reply.UserID)].Name,
				Avatar: createrMap[int(reply.UserID)].Avatar,
			},
		})
	}

	// Cache the result
	go func() {
		if cacheErr := s.cache.CacheCommentReplies(context.Background(), req.GetCommentId(), apiReplies); cacheErr != nil {
			logger.Error("Failed to cache comment replies", zap.Error(cacheErr))
		}
	}()

	return &api.GetStoryCommentRepliesResponse{
		Code:    api.ResponseCode_OK,
		Message: "success",
		Total:   int64(len(*replies)),
		Replies: apiReplies,
	}, nil
}

func (s *CommentService) CreateStoryCommentReply(ctx context.Context, req *api.CreateStoryCommentReplyRequest) (*api.CreateStoryCommentReplyResponse, error) {
	logger.Info("CreateStoryCommentReply called", zap.Any("req", req))
	// 参数校验：内容判空，CommentId合法性
	if req.GetContent() == "" {
		logger.Error("CreateStoryCommentReply failed: content is empty")
		return &api.CreateStoryCommentReplyResponse{
			Code:    api.ResponseCode_INVALID_PARAMETER,
			Message: "content is empty",
		}, nil
	}
	if req.GetCommentId() <= 0 {
		logger.Error("CreateStoryCommentReply failed: invalid comment id", zap.Int64("comment_id", req.GetCommentId()))
		return &api.CreateStoryCommentReplyResponse{
			Code:    api.ResponseCode_INVALID_PARAMETER,
			Message: "invalid comment id",
		}, nil
	}
	rootComment := &models.Comment{
		ID: uint(req.GetCommentId()),
	}
	err := rootComment.GetComment(ctx)
	if err != nil {
		logger.Error("CreateStoryCommentReply failed: root comment not found", zap.Error(err))
		return &api.CreateStoryCommentReplyResponse{
			Code:    api.ResponseCode_COMMENT_NOT_FOUND,
			Message: err.Error(),
		}, nil
	}
	comment := &models.Comment{
		UserID:       req.GetUserId(),
		StoryID:      rootComment.StoryID,
		StoryboardID: rootComment.StoryboardID,
		PreID:        req.GetCommentId(),
		Content:      []byte(req.GetContent()),
		CommentType:  models.CommentTypeReply,
		Status:       1, // 建议用常量
		ReplyCount:   0,
	}
	if rootComment.RootCommentID == 0 {
		comment.RootCommentID = int64(rootComment.ID)
	} else {
		comment.RootCommentID = int64(rootComment.RootCommentID)
	}
	err = comment.Create(ctx)
	if err != nil {
		logger.Error("CreateStoryCommentReply failed: create reply error", zap.Error(err))
		return &api.CreateStoryCommentReplyResponse{
			Code:    api.ResponseCode_OPERATION_FAILED,
			Message: err.Error(),
		}, nil
	}
	logger.Info("CreateStoryCommentReply: reply created", zap.Any("reply", comment))
	err = models.IncreaseReplyCount(ctx, uint64(rootComment.ID))
	if err != nil {
		logger.Error("increase story comment reply count failed", zap.Error(err))
	}
	logger.Info("CreateStoryCommentReply success", zap.Int64("comment_id", req.GetCommentId()), zap.Int64("user_id", req.GetUserId()))

	// Invalidate cache for comment replies
	go func() {
		if cacheErr := s.cache.InvalidateCommentRepliesCache(context.Background(), req.GetCommentId()); cacheErr != nil {
			logger.Error("Failed to invalidate comment replies cache", zap.Error(cacheErr))
		}
		// Also invalidate root comment's replies cache if different
		if rootComment.RootCommentID != 0 && rootComment.RootCommentID != req.GetCommentId() {
			if cacheErr := s.cache.InvalidateCommentRepliesCache(context.Background(), rootComment.RootCommentID); cacheErr != nil {
				logger.Error("Failed to invalidate root comment replies cache", zap.Error(cacheErr))
			}
		}
	}()

	// Create system notification for comment author
	go func() {
		bgCtx := context.Background()
		// Don't notify if user replies to their own comment
		if rootComment.UserID == req.GetUserId() {
			return
		}
		commentID64 := int64(comment.ID)
		storyID64 := comment.StoryID
		userID64 := req.GetUserId()
		rootCommentID64 := req.GetCommentId()
		notifErr := llmchatservice.CreateSystemNotification(bgCtx, llmchatservice.CreateNotificationParams{
			UserID:           rootComment.UserID,
			Type:             models.SystemNotificationTypeComment,
			Title:            "新回复提醒",
			Content:          "有人回复了你的评论",
			RelatedUserID:    &userID64,
			RelatedStoryID:   &storyID64,
			RelatedCommentID: &rootCommentID64,
			ExtraData: map[string]interface{}{
				"reply_id": commentID64,
			},
		})
		if notifErr != nil {
			logger.Error("Failed to create reply notification", zap.Error(notifErr))
		}
	}()

	return &api.CreateStoryCommentReplyResponse{
		Code:    api.ResponseCode_OK,
		Message: "success",
	}, nil
}

func (s *CommentService) DeleteStoryCommentReply(ctx context.Context, req *api.DeleteStoryCommentReplyRequest) (*api.DeleteStoryCommentReplyResponse, error) {
	// 权限验证：只有回复作者可以删除自己的回复
	if !s.hasPermissionToDelete(ctx, req.GetUserId(), req.GetReplyId()) {
		logger.Warn("DeleteStoryCommentReply failed: permission denied",
			zap.Int64("user_id", req.GetUserId()),
			zap.Int64("reply_id", req.GetReplyId()))
		return &api.DeleteStoryCommentReplyResponse{
			Code:    api.ResponseCode_PERMISSION_DENIED,
			Message: "no permission to delete reply",
		}, nil
	}

	// 先获取回复信息（在删除前获取，确保能正确获取到计数信息）
	targetReply := &models.Comment{
		ID: uint(req.GetReplyId()),
	}
	err := targetReply.GetComment(ctx)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return &api.DeleteStoryCommentReplyResponse{
				Code:    api.ResponseCode_COMMENT_NOT_FOUND,
				Message: "reply not found",
			}, nil
		}
		return &api.DeleteStoryCommentReplyResponse{
			Code:    api.ResponseCode_OPERATION_FAILED,
			Message: err.Error(),
		}, nil
	}

	// 使用事务确保删除和计数更新的一致性
	err = models.DataBase().Transaction(func(tx *gorm.DB) error {
		// 软删除回复
		if err := tx.WithContext(ctx).Model(&models.Comment{}).
			Where("id = ?", req.GetReplyId()).
			Update("deleted", 1).Error; err != nil {
			logger.Error("delete story comment reply failed", zap.Error(err))
			return err
		}

		// 减少根评论的回复计数
		if targetReply.RootCommentID > 0 {
			if err := tx.WithContext(ctx).Model(&models.Comment{}).
				Where("id = ?", targetReply.RootCommentID).
				Update("reply_count", gorm.Expr("reply_count - 1")).Error; err != nil {
				logger.Error("decrease reply count failed", zap.Error(err))
				return err
			}
		}

		return nil
	})

	if err != nil {
		logger.Error("DeleteStoryCommentReply transaction failed", zap.Error(err))
		return &api.DeleteStoryCommentReplyResponse{
			Code:    api.ResponseCode_OPERATION_FAILED,
			Message: err.Error(),
		}, nil
	}

	// 失效相关缓存
	go func() {
		if targetReply.RootCommentID > 0 {
			if cacheErr := s.cache.InvalidateCommentRepliesCache(context.Background(), targetReply.RootCommentID); cacheErr != nil {
				logger.Error("Failed to invalidate comment replies cache", zap.Error(cacheErr))
			}
		}
	}()

	return &api.DeleteStoryCommentReplyResponse{
		Code:    api.ResponseCode_OK,
		Message: "success",
	}, nil
}

func (s *CommentService) CreateStoryBoardComment(ctx context.Context, req *api.CreateStoryBoardCommentRequest) (*api.CreateStoryBoardCommentResponse, error) {
	logger.Info("CreateStoryBoardComment called", zap.Any("req", req))
	// 参数校验：内容判空，BoardId合法性
	if req.GetContent() == "" {
		logger.Error("CreateStoryBoardComment failed: content is empty")
		return &api.CreateStoryBoardCommentResponse{
			Code:    api.ResponseCode_INVALID_PARAMETER,
			Message: "content is empty",
		}, nil
	}
	if req.GetBoardId() <= 0 {
		logger.Error("CreateStoryBoardComment failed: invalid board id", zap.Int64("board_id", req.GetBoardId()))
		return &api.CreateStoryBoardCommentResponse{
			Code:    api.ResponseCode_INVALID_PARAMETER,
			Message: "invalid board id",
		}, nil
	}
	comment := &models.Comment{
		UserID:        req.GetUserId(),
		StoryboardID:  req.GetBoardId(),
		Content:       []byte(req.GetContent()),
		CommentType:   models.CommentTypeComment,
		RootCommentID: 0,
		PreID:         0,
		Status:        1, // 建议用常量
		LikeCount:     0,
	}
	err := comment.Create(ctx)
	if err != nil {
		logger.Error("CreateStoryBoardComment failed: create comment error", zap.Error(err))
		return &api.CreateStoryBoardCommentResponse{
			Code:    api.ResponseCode_OPERATION_FAILED,
			Message: err.Error(),
		}, nil
	}
	logger.Info("CreateStoryBoardComment: comment created", zap.Any("comment", comment))
	err = models.IncrementStoryBoardCommentNum(ctx, req.GetBoardId())
	if err != nil {
		logger.Error("CreateStoryBoardComment failed: increase storyboard comment count error", zap.Error(err))
		return &api.CreateStoryBoardCommentResponse{
			Code:    api.ResponseCode_OPERATION_FAILED,
			Message: "increase storyboard comment count failed",
		}, nil
	}
	logger.Info("CreateStoryBoardComment success", zap.Int64("board_id", req.GetBoardId()), zap.Int64("user_id", req.GetUserId()))

	// Invalidate cache for this storyboard's comments
	go func() {
		if cacheErr := s.cache.InvalidateStoryBoardCommentsCache(context.Background(), req.GetBoardId()); cacheErr != nil {
			logger.Error("Failed to invalidate storyboard comments cache", zap.Error(cacheErr))
		}
	}()

	// Create system notification for storyboard author
	go func() {
		bgCtx := context.Background()
		storyboard, err := models.GetStoryboard(bgCtx, req.GetBoardId())
		if err != nil {
			logger.Error("Failed to get storyboard for notification", zap.Error(err))
			return
		}
		// Don't notify if user comments on their own storyboard
		if storyboard.CreatorID == req.GetUserId() {
			return
		}
		commentID64 := int64(comment.ID)
		storyboardID64 := req.GetBoardId()
		userID64 := req.GetUserId()
		notifErr := llmchatservice.CreateSystemNotification(bgCtx, llmchatservice.CreateNotificationParams{
			UserID:              storyboard.CreatorID,
			Type:                models.SystemNotificationTypeComment,
			Title:               "新评论提醒",
			Content:             fmt.Sprintf("有人评论了你的故事板《%s》", storyboard.Title),
			RelatedUserID:       &userID64,
			RelatedStoryBoardID: &storyboardID64,
			RelatedCommentID:    &commentID64,
		})
		if notifErr != nil {
			logger.Error("Failed to create storyboard comment notification", zap.Error(notifErr))
		}
	}()

	return &api.CreateStoryBoardCommentResponse{
		Code:    api.ResponseCode_OK,
		Message: "success",
	}, nil
}

func (s *CommentService) DeleteStoryBoardComment(ctx context.Context, req *api.DeleteStoryBoardCommentRequest) (*api.DeleteStoryBoardCommentResponse, error) {
	// 权限验证：只有评论作者可以删除自己的评论
	if !s.hasPermissionToDelete(ctx, req.GetUserId(), req.GetCommentId()) {
		logger.Warn("DeleteStoryBoardComment failed: permission denied",
			zap.Int64("user_id", req.GetUserId()),
			zap.Int64("comment_id", req.GetCommentId()))
		return &api.DeleteStoryBoardCommentResponse{
			Code:    api.ResponseCode_PERMISSION_DENIED,
			Message: "no permission to delete comment",
		}, nil
	}

	// 先获取评论信息（在删除前获取，确保能正确获取到信息）
	comment := &models.Comment{
		ID: uint(req.GetCommentId()),
	}
	err := comment.GetComment(ctx)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return &api.DeleteStoryBoardCommentResponse{
				Code:    api.ResponseCode_COMMENT_NOT_FOUND,
				Message: "comment not found",
			}, nil
		}
		return &api.DeleteStoryBoardCommentResponse{
			Code:    api.ResponseCode_OPERATION_FAILED,
			Message: err.Error(),
		}, nil
	}

	// 使用事务确保删除和计数更新的一致性
	err = models.DataBase().Transaction(func(tx *gorm.DB) error {
		// 软删除评论
		if err := tx.WithContext(ctx).Model(&models.Comment{}).
			Where("id = ?", req.GetCommentId()).
			Update("deleted", 1).Error; err != nil {
			logger.Error("delete storyboard comment failed", zap.Error(err))
			return err
		}

		// 减少故事板评论计数
		if err := tx.WithContext(ctx).Model(&models.StoryBoard{}).
			Where("id = ?", comment.StoryboardID).
			Update("comment_count", gorm.Expr("comment_count - 1")).Error; err != nil {
			logger.Error("decrease storyboard comment count failed", zap.Error(err))
			return err
		}

		return nil
	})

	if err != nil {
		logger.Error("DeleteStoryBoardComment transaction failed", zap.Error(err))
		return &api.DeleteStoryBoardCommentResponse{
			Code:    api.ResponseCode_OPERATION_FAILED,
			Message: err.Error(),
		}, nil
	}

	// 失效相关缓存
	go func() {
		if cacheErr := s.cache.InvalidateStoryBoardCommentsCache(context.Background(), comment.StoryboardID); cacheErr != nil {
			logger.Error("Failed to invalidate storyboard comments cache", zap.Error(cacheErr))
		}
	}()

	return &api.DeleteStoryBoardCommentResponse{
		Code:    api.ResponseCode_OK,
		Message: "success",
	}, nil
}

func (s *CommentService) GetStoryBoardComments(ctx context.Context, req *api.GetStoryBoardCommentsRequest) (*api.GetStoryBoardCommentsResponse, error) {
	// Try to get from cache first
	cachedComments, err := s.cache.GetCachedStoryBoardComments(ctx, req.GetBoardId(), int64(req.GetOffset()), int64(req.GetPageSize()))
	if err == nil && cachedComments != nil {
		logger.Info("GetStoryBoardComments: cache hit", zap.Int64("board_id", req.GetBoardId()))
		return &api.GetStoryBoardCommentsResponse{
			Code:     api.ResponseCode_OK,
			Message:  "success",
			Total:    int64(len(cachedComments)),
			Comments: cachedComments,
		}, nil
	}

	logger.Info("GetStoryBoardComments: cache miss, querying database", zap.Int64("board_id", req.GetBoardId()))
	comments, err := models.GetCommentListByStoryBoard(ctx,
		uint64(req.GetBoardId()), int64(req.GetOffset()), int64(req.GetPageSize()))
	if err != nil {
		return &api.GetStoryBoardCommentsResponse{
			Code:     api.ResponseCode_DATABASE_ERROR,
			Message:  "get comments error",
			Total:    0,
			Comments: []*api.StoryComment{},
		}, nil
	}
	if len(*comments) == 0 {
		logger.Info("get comment list by story board empty")
		return &api.GetStoryBoardCommentsResponse{
			Code:     api.ResponseCode_OK,
			Message:  "success",
			Total:    0,
			Comments: []*api.StoryComment{},
		}, nil
	}
	createrIds := make([]int64, 0)
	for _, comment := range *comments {
		createrIds = append(createrIds, comment.UserID)
	}
	createrMap, err := models.GetUsersByIdsMap(ctx, createrIds)
	if err != nil {
		logger.Error("get user by ids map error", zap.Error(err))
		return nil, err
	}
	createrMapData, _ := json.Marshal(createrMap)
	logger.Info("get user by ids map success", zap.String("creater_map", string(createrMapData)))
	apiComments := make([]*api.StoryComment, 0)
	for _, comment := range *comments {
		apiComments = append(apiComments, &api.StoryComment{
			CommentId:  int64(comment.ID),
			Content:    string(comment.Content),
			CreatedAt:  comment.CreateAt.Unix(),
			UpdatedAt:  comment.UpdateAt.Unix(),
			UserId:     comment.UserID,
			BoardId:    comment.StoryboardID,
			LikeCount:  comment.LikeCount,
			ReplyCount: comment.ReplyCount,
			Creator: &api.UserInfo{
				UserId: comment.UserID,
				Name:   createrMap[int(comment.UserID)].Name,
				Avatar: createrMap[int(comment.UserID)].Avatar,
			},
		})
	}

	// Cache the result
	go func() {
		if cacheErr := s.cache.CacheStoryBoardComments(context.Background(), req.GetBoardId(), int64(req.GetOffset()), int64(req.GetPageSize()), apiComments); cacheErr != nil {
			logger.Error("Failed to cache storyboard comments", zap.Error(cacheErr))
		}
	}()

	logger.Info("get comment list by story board success")
	return &api.GetStoryBoardCommentsResponse{
		Code:     api.ResponseCode_OK,
		Message:  "success",
		Total:    int64(len(*comments)),
		Comments: apiComments,
	}, nil
}

func (s *CommentService) LikeComment(ctx context.Context, req *api.LikeCommentRequest) (*api.LikeCommentResponse, error) {
	// 检查是否已经点赞
	isLiked, err := models.GetCommentLike(ctx, uint64(req.GetCommentId()), uint64(req.GetUserId()))
	if err != nil {
		logger.Error("check comment like status failed", zap.Error(err))
		return &api.LikeCommentResponse{
			Code:    api.ResponseCode_OPERATION_FAILED,
			Message: "failed to check like status",
		}, nil
	}

	// 如果已经点赞，返回已点赞状态
	if isLiked != nil {
		return &api.LikeCommentResponse{
			Code:    api.ResponseCode_OK,
			Message: "already liked",
		}, nil
	}

	// 使用事务确保点赞操作和计数更新的一致性
	err = models.DataBase().Transaction(func(tx *gorm.DB) error {
		// 创建点赞记录
		commentLike := &models.CommentLike{
			UserID:    req.GetUserId(),
			CommentID: req.GetCommentId(),
		}
		if err := tx.WithContext(ctx).Create(commentLike).Error; err != nil {
			logger.Error("create comment like failed", zap.Error(err))
			return err
		}

		// 更新评论点赞数
		if err := tx.Model(&models.Comment{}).
			WithContext(ctx).
			Where("id = ?", req.GetCommentId()).
			Update("like_count", gorm.Expr("like_count + 1")).Error; err != nil {
			logger.Error("update comment like count failed", zap.Error(err))
			return err
		}

		return nil
	})

	if err != nil {
		logger.Error("like comment failed", zap.Error(err))
		return &api.LikeCommentResponse{
			Code:    api.ResponseCode_OPERATION_FAILED,
			Message: err.Error(),
		}, nil
	}

	// 创建点赞通知（异步，避免阻塞主流程）
	go func() {
		bgCtx := context.Background()
		// 获取评论信息以获取评论作者
		var comment models.Comment
		if err := models.DataBase().WithContext(bgCtx).Where("id = ?", req.GetCommentId()).First(&comment).Error; err != nil {
			logger.Error("failed to get comment for notification", zap.Error(err))
			return
		}
		// 如果点赞的是自己的评论，不需要通知
		if comment.UserID == req.GetUserId() {
			return
		}
		userID64 := req.GetUserId()
		commentID64 := req.GetCommentId()
		notifErr := llmchatservice.CreateSystemNotification(bgCtx, llmchatservice.CreateNotificationParams{
			UserID:        comment.UserID,
			Type:          models.SystemNotificationTypeLike,
			Title:         "点赞提醒",
			Content:       "有人点赞了你的评论",
			RelatedUserID: &userID64,
			RelatedCommentID: &commentID64,
		})
		if notifErr != nil {
			logger.Error("Failed to create comment like notification", zap.Error(notifErr))
		}
	}()

	return &api.LikeCommentResponse{
		Code:    api.ResponseCode_OK,
		Message: "success",
	}, nil
}
func (s *CommentService) DislikeComment(ctx context.Context, req *api.DislikeCommentRequest) (*api.DislikeCommentResponse, error) {
	// 检查是否已经点赞
	isLiked, err := models.GetCommentLike(ctx, uint64(req.GetCommentId()), uint64(req.GetUserId()))
	if err != nil {
		logger.Error("check comment like status failed", zap.Error(err))
		return &api.DislikeCommentResponse{
			Code:    api.ResponseCode_OPERATION_FAILED,
			Message: "failed to check like status",
		}, nil
	}

	// 如果没有点赞，返回未点赞状态
	if isLiked == nil {
		return &api.DislikeCommentResponse{
			Code:    api.ResponseCode_OK,
			Message: "not liked",
		}, nil
	}

	// 使用事务确保取消点赞操作和计数更新的一致性
	err = models.DataBase().Transaction(func(tx *gorm.DB) error {
		// 删除点赞记录
		if err := tx.WithContext(ctx).Delete(isLiked).Error; err != nil {
			logger.Error("delete comment like failed", zap.Error(err))
			return err
		}

		// 更新评论点赞数
		if err := tx.Model(&models.Comment{}).
			WithContext(ctx).
			Where("id = ?", req.GetCommentId()).
			Update("like_count", gorm.Expr("like_count - 1")).Error; err != nil {
			logger.Error("update comment like count failed", zap.Error(err))
			return err
		}

		return nil
	})

	if err != nil {
		logger.Error("dislike comment failed", zap.Error(err))
		return &api.DislikeCommentResponse{
			Code:    api.ResponseCode_OPERATION_FAILED,
			Message: err.Error(),
		}, nil
	}

	return &api.DislikeCommentResponse{
		Code:    api.ResponseCode_OK,
		Message: "success",
	}, nil
}

func (s *CommentService) GetStoryBoardCommentReplies(ctx context.Context, req *api.GetStoryBoardCommentRepliesRequest) (*api.GetStoryBoardCommentRepliesResponse, error) {
	// Try to get from cache first
	cachedReplies, err := s.cache.GetCachedCommentReplies(ctx, req.GetCommentId())
	if err == nil && cachedReplies != nil {
		logger.Info("GetStoryBoardCommentReplies: cache hit", zap.Int64("comment_id", req.GetCommentId()))
		return &api.GetStoryBoardCommentRepliesResponse{
			Code:    api.ResponseCode_OK,
			Message: "success",
			Total:   int64(len(cachedReplies)),
			Replies: cachedReplies,
		}, nil
	}

	logger.Info("GetStoryBoardCommentReplies: cache miss, querying database", zap.Int64("comment_id", req.GetCommentId()))
	replies, err := models.GetStoryBoardCommentReplies(ctx, uint64(req.GetCommentId()))
	if err != nil {
		return nil, err
	}
	if len(*replies) == 0 {
		return &api.GetStoryBoardCommentRepliesResponse{
			Code:    api.ResponseCode_OK,
			Message: "success",
			Total:   0,
			Replies: []*api.StoryComment{},
		}, nil
	}
	createrIds := make([]int64, 0)
	for _, reply := range *replies {
		createrIds = append(createrIds, reply.UserID)
	}
	createrMap, err := models.GetUsersByIdsMap(ctx, createrIds)
	if err != nil {
		logger.Error("get user by ids map error", zap.Error(err))
		return nil, err
	}
	createrMapData, _ := json.Marshal(createrMap)
	logger.Info("get user by ids map success", zap.String("creater_map", string(createrMapData)))
	apiReplies := make([]*api.StoryComment, 0)
	for _, reply := range *replies {
		apiReplies = append(apiReplies, &api.StoryComment{
			CommentId:  int64(reply.ID),
			Content:    string(reply.Content),
			CreatedAt:  reply.CreateAt.Unix(),
			UpdatedAt:  reply.UpdateAt.Unix(),
			UserId:     reply.UserID,
			BoardId:    reply.StoryboardID,
			LikeCount:  reply.LikeCount,
			ReplyCount: reply.ReplyCount,
			Creator: &api.UserInfo{
				UserId: reply.UserID,
				Name:   createrMap[int(reply.UserID)].Name,
				Avatar: createrMap[int(reply.UserID)].Avatar,
			},
		})
	}

	// Cache the result
	go func() {
		if cacheErr := s.cache.CacheCommentReplies(context.Background(), req.GetCommentId(), apiReplies); cacheErr != nil {
			logger.Error("Failed to cache storyboard comment replies", zap.Error(cacheErr))
		}
	}()

	return &api.GetStoryBoardCommentRepliesResponse{
		Code:    api.ResponseCode_OK,
		Message: "success",
		Total:   int64(len(*replies)),
		Replies: apiReplies,
	}, nil
}
