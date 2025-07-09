package comment

import (
	"context"
	"encoding/json"

	"go.uber.org/zap"

	api "github.com/grapery/common-protoc/gen"
	"github.com/grapery/grapery/models"
)

var logger, _ = zap.NewDevelopment()

var commentServer CommentServer

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
	LikeComment(ctx context.Context, req *api.LikeCommentRequest) (*api.LikeCommentResponse, error)
	DislikeComment(ctx context.Context, req *api.DislikeCommentRequest) (*api.DislikeCommentResponse, error)
}

type CommentService struct {
}

func NewCommentService() *CommentService {
	return &CommentService{}
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
	comment := &models.Comment{
		UserID:       req.GetUserId(),
		StoryID:      req.GetStoryId(),
		Content:      []byte(req.GetContent()),
		CommentType:  models.CommentTypeComment,
		Status:       1, // 建议用常量
		LikeCount:    0,
		DislikeCount: 0,
	}
	err := comment.Create()
	if err != nil {
		logger.Error("CreateStoryComment failed: create comment error", zap.Error(err))
		return &api.CreateStoryCommentResponse{
			Code:    api.ResponseCode_OPERATION_FAILED,
			Message: err.Error(),
		}, nil
	}
	logger.Info("CreateStoryComment: comment created", zap.Any("comment", comment))
	story, err := models.GetStory(ctx, req.GetStoryId())
	if err != nil {
		logger.Error("CreateStoryComment failed: get story error", zap.Error(err))
		return &api.CreateStoryCommentResponse{
			Code:    api.ResponseCode_STORY_NOT_FOUND,
			Message: "get story failed",
		}, nil
	}
	story.CommentCount++
	err = models.UpdateStory(ctx, story)
	if err != nil {
		logger.Error("CreateStoryComment failed: update story comment count error", zap.Error(err))
		return &api.CreateStoryCommentResponse{
			Code:    api.ResponseCode_OPERATION_FAILED,
			Message: "update story comment count failed",
		}, nil
	}
	logger.Info("CreateStoryComment success", zap.Int64("story_id", req.GetStoryId()), zap.Int64("user_id", req.GetUserId()))
	return &api.CreateStoryCommentResponse{
		Code:    api.ResponseCode_OK,
		Message: "success",
	}, nil
}

func (s *CommentService) GetStoryComments(ctx context.Context, req *api.GetStoryCommentsRequest) (*api.GetStoryCommentsResponse, error) {
	comments, err := models.GetCommentByStory(uint64(req.GetStoryId()),
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
	return &api.GetStoryCommentsResponse{
		Code:     api.ResponseCode_OK,
		Message:  "success",
		Total:    int64(len(*comments)),
		Comments: apiComments,
	}, nil
}

func (s *CommentService) DeleteStoryComment(ctx context.Context, req *api.DeleteStoryCommentRequest) (*api.DeleteStoryCommentResponse, error) {
	err := models.DeleteComment(uint64(req.GetCommentId()))
	if err != nil {
		return &api.DeleteStoryCommentResponse{
			Code:    api.ResponseCode_OPERATION_FAILED,
			Message: err.Error(),
		}, nil
	}

	return &api.DeleteStoryCommentResponse{
		Code:    api.ResponseCode_OK,
		Message: "success",
	}, nil
}

func (s *CommentService) GetStoryCommentReplies(ctx context.Context, req *api.GetStoryCommentRepliesRequest) (*api.GetStoryCommentRepliesResponse, error) {
	replies, err := models.GetStoryCommentReplies(uint64(req.GetCommentId()))
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
		IDBase: models.IDBase{
			ID: uint(req.GetCommentId()),
		},
	}
	err := rootComment.GetComment()
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
	err = comment.Create()
	if err != nil {
		logger.Error("CreateStoryCommentReply failed: create reply error", zap.Error(err))
		return &api.CreateStoryCommentReplyResponse{
			Code:    api.ResponseCode_OPERATION_FAILED,
			Message: err.Error(),
		}, nil
	}
	logger.Info("CreateStoryCommentReply: reply created", zap.Any("reply", comment))
	err = models.IncreaseReplyCount(uint64(rootComment.ID))
	if err != nil {
		logger.Error("increase story comment reply count failed", zap.Error(err))
	}
	logger.Info("CreateStoryCommentReply success", zap.Int64("comment_id", req.GetCommentId()), zap.Int64("user_id", req.GetUserId()))
	return &api.CreateStoryCommentReplyResponse{
		Code:    api.ResponseCode_OK,
		Message: "success",
	}, nil
}

func (s *CommentService) DeleteStoryCommentReply(ctx context.Context, req *api.DeleteStoryCommentReplyRequest) (*api.DeleteStoryCommentReplyResponse, error) {
	err := models.DeleteStoryCommentReply(uint64(req.GetReplyId()))
	if err != nil {
		return &api.DeleteStoryCommentReplyResponse{
			Code:    api.ResponseCode_OPERATION_FAILED,
			Message: err.Error(),
		}, nil
	}
	targetComment := &models.Comment{
		IDBase: models.IDBase{
			ID: uint(req.GetReplyId()),
		},
	}
	err = targetComment.GetComment()
	if err != nil {
		return &api.DeleteStoryCommentReplyResponse{
			Code:    api.ResponseCode_COMMENT_NOT_FOUND,
			Message: err.Error(),
		}, nil
	}
	err = models.DecreaseReplyCount(uint64(targetComment.RootCommentID))
	if err != nil {
		logger.Error("decrease story comment reply count failed", zap.Error(err))
	}
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
	err := comment.Create()
	if err != nil {
		logger.Error("CreateStoryBoardComment failed: create comment error", zap.Error(err))
		return &api.CreateStoryBoardCommentResponse{
			Code:    api.ResponseCode_OPERATION_FAILED,
			Message: err.Error(),
		}, nil
	}
	logger.Info("CreateStoryBoardComment: comment created", zap.Any("comment", comment))
	storyBoard, err := models.GetStoryboard(ctx, req.GetBoardId())
	if err != nil {
		logger.Error("CreateStoryBoardComment failed: get storyboard error", zap.Error(err))
		return &api.CreateStoryBoardCommentResponse{
			Code:    api.ResponseCode_STORYBOARD_NOT_FOUND,
			Message: "get storyboard failed",
		}, nil
	}
	storyBoard.CommentNum++
	err = models.UpdateStoryboard(ctx, storyBoard)
	if err != nil {
		logger.Error("CreateStoryBoardComment failed: update storyboard comment count error", zap.Error(err))
		return &api.CreateStoryBoardCommentResponse{
			Code:    api.ResponseCode_OPERATION_FAILED,
			Message: "update storyboard comment count failed",
		}, nil
	}
	logger.Info("CreateStoryBoardComment success", zap.Int64("board_id", req.GetBoardId()), zap.Int64("user_id", req.GetUserId()))
	return &api.CreateStoryBoardCommentResponse{
		Code:    api.ResponseCode_OK,
		Message: "success",
	}, nil
}

func (s *CommentService) DeleteStoryBoardComment(ctx context.Context, req *api.DeleteStoryBoardCommentRequest) (*api.DeleteStoryBoardCommentResponse, error) {
	err := models.DeleteComment(uint64(req.GetCommentId()))
	if err != nil {
		return &api.DeleteStoryBoardCommentResponse{
			Code:    api.ResponseCode_OPERATION_FAILED,
			Message: err.Error(),
		}, nil
	}
	storyBoard, err := models.GetStoryboard(ctx, req.GetBoardId())
	if err != nil {
		return &api.DeleteStoryBoardCommentResponse{
			Code:    api.ResponseCode_STORYBOARD_NOT_FOUND,
			Message: "get storyboard failed",
		}, nil
	}
	storyBoard.CommentNum--
	return &api.DeleteStoryBoardCommentResponse{
		Code:    api.ResponseCode_OK,
		Message: "success",
	}, nil
}
func (s *CommentService) GetStoryBoardComments(ctx context.Context, req *api.GetStoryBoardCommentsRequest) (*api.GetStoryBoardCommentsResponse, error) {
	comments, err := models.GetCommentListByStoryBoard(
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
	logger.Info("get comment list by story board success")
	return &api.GetStoryBoardCommentsResponse{
		Code:     api.ResponseCode_OK,
		Message:  "success",
		Total:    int64(len(*comments)),
		Comments: apiComments,
	}, nil
}

func (s *CommentService) LikeComment(ctx context.Context, req *api.LikeCommentRequest) (*api.LikeCommentResponse, error) {
	err := models.LikeComment(uint64(req.GetCommentId()), uint64(req.GetUserId()))
	if err != nil {
		return &api.LikeCommentResponse{
			Code:    api.ResponseCode_OPERATION_FAILED,
			Message: err.Error(),
		}, nil
	}
	return &api.LikeCommentResponse{
		Code:    api.ResponseCode_OK,
		Message: "success",
	}, nil
}
func (s *CommentService) DislikeComment(ctx context.Context, req *api.DislikeCommentRequest) (*api.DislikeCommentResponse, error) {
	err := models.DislikeComment(uint64(req.GetCommentId()), uint64(req.GetUserId()))
	if err != nil {
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
	replies, err := models.GetStoryBoardCommentReplies(uint64(req.GetCommentId()))
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
	return &api.GetStoryBoardCommentRepliesResponse{
		Code:    api.ResponseCode_OK,
		Message: "success",
		Total:   int64(len(*replies)),
		Replies: apiReplies,
	}, nil
}
