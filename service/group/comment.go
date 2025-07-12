package group

import (
	// "net/http"
	"context"
	"strings"

	"go.uber.org/zap"

	connect "connectrpc.com/connect"
	_ "github.com/gin-contrib/sessions"
	_ "github.com/gin-contrib/sessions/redis"
	api "github.com/grapery/common-protoc/gen"
	commentService "github.com/grapery/grapery/pkg/comment"
	// 注释掉 models 相关代码，避免导入错误
	// import (
	// 	"github.com/grapery/grapery/pkg/models"
	// )
)

type CommentService struct {
}

// 从 context 获取 traceId，没有则生成

func (ts *CommentService) CreateStoryComment(ctx context.Context, req *connect.Request[api.CreateStoryCommentRequest]) (*connect.Response[api.CreateStoryCommentResponse], error) {
	traceId := getTraceID(ctx)
	userId := req.Msg.GetUserId()
	storyId := req.Msg.GetStoryId()
	zap.L().Info("CreateStoryComment called", zap.String("traceId", traceId), zap.Int64("userId", userId), zap.Int64("storyId", storyId), zap.String("content", maskContent(req.Msg.GetContent())))
	if userId <= 0 || storyId <= 0 || strings.TrimSpace(req.Msg.GetContent()) == "" {
		zap.L().Warn("CreateStoryComment param invalid", zap.String("traceId", traceId), zap.Int64("userId", userId), zap.Int64("storyId", storyId))
		return &connect.Response[api.CreateStoryCommentResponse]{
			Msg: &api.CreateStoryCommentResponse{
				Code:    api.ResponseCode_INVALID_PARAMETER,
				Message: "参数错误",
			},
		}, nil
	}
	ret, err := commentService.GetCommentServer().CreateStoryComment(ctx, req.Msg)
	if err != nil {
		zap.L().Error("CreateStoryComment failed", zap.String("traceId", traceId), zap.Error(err))
	}
	return &connect.Response[api.CreateStoryCommentResponse]{
		Msg: ret,
	}, nil
}

func (ts *CommentService) GetStoryComments(ctx context.Context, req *connect.Request[api.GetStoryCommentsRequest]) (*connect.Response[api.GetStoryCommentsResponse], error) {
	traceId := getTraceID(ctx)
	storyId := req.Msg.GetStoryId()
	zap.L().Info("GetStoryComments called", zap.String("traceId", traceId), zap.Int64("storyId", storyId))
	if storyId <= 0 {
		zap.L().Warn("GetStoryComments param invalid", zap.String("traceId", traceId), zap.Int64("storyId", storyId))
		return &connect.Response[api.GetStoryCommentsResponse]{
			Msg: &api.GetStoryCommentsResponse{
				Code:    api.ResponseCode_INVALID_PARAMETER,
				Message: "参数错误",
			},
		}, nil
	}
	ret, err := commentService.GetCommentServer().GetStoryComments(ctx, req.Msg)
	if err != nil {
		zap.L().Error("GetStoryComments failed", zap.String("traceId", traceId), zap.Error(err))
	}
	return &connect.Response[api.GetStoryCommentsResponse]{
		Msg: ret,
	}, nil
}

func (ts *CommentService) DeleteStoryComment(ctx context.Context, req *connect.Request[api.DeleteStoryCommentRequest]) (*connect.Response[api.DeleteStoryCommentResponse], error) {
	traceId := getTraceID(ctx)
	userId := req.Msg.GetUserId()
	commentId := req.Msg.GetCommentId()
	zap.L().Info("DeleteStoryComment called", zap.String("traceId", traceId), zap.Int64("userId", userId), zap.Int64("commentId", commentId))
	if userId <= 0 || commentId <= 0 {
		zap.L().Warn("DeleteStoryComment param invalid", zap.String("traceId", traceId), zap.Int64("userId", userId), zap.Int64("commentId", commentId))
		return &connect.Response[api.DeleteStoryCommentResponse]{
			Msg: &api.DeleteStoryCommentResponse{
				Code:    api.ResponseCode_INVALID_PARAMETER,
				Message: "参数错误",
			},
		}, nil
	}
	// 权限校验暂不做（如需请补充models包）
	ret, err := commentService.GetCommentServer().DeleteStoryComment(ctx, req.Msg)
	if err != nil {
		zap.L().Error("DeleteStoryComment failed", zap.String("traceId", traceId), zap.Error(err))
	}
	return &connect.Response[api.DeleteStoryCommentResponse]{
		Msg: ret,
	}, nil
}

func (ts *CommentService) GetStoryCommentReplies(ctx context.Context, req *connect.Request[api.GetStoryCommentRepliesRequest]) (*connect.Response[api.GetStoryCommentRepliesResponse], error) {
	traceId := getTraceID(ctx)
	commentId := req.Msg.GetCommentId()
	zap.L().Info("GetStoryCommentReplies called", zap.String("traceId", traceId), zap.Int64("commentId", commentId))
	if commentId <= 0 {
		zap.L().Warn("GetStoryCommentReplies param invalid", zap.String("traceId", traceId), zap.Int64("commentId", commentId))
		return &connect.Response[api.GetStoryCommentRepliesResponse]{
			Msg: &api.GetStoryCommentRepliesResponse{
				Code:    api.ResponseCode_INVALID_PARAMETER,
				Message: "参数错误",
			},
		}, nil
	}
	ret, err := commentService.GetCommentServer().GetStoryCommentReplies(ctx, req.Msg)
	if err != nil {
		zap.L().Error("GetStoryCommentReplies failed", zap.String("traceId", traceId), zap.Error(err))
	}
	println("get story comment replies success", ret.String())
	return &connect.Response[api.GetStoryCommentRepliesResponse]{
		Msg: ret,
	}, nil
}

func (ts *CommentService) CreateStoryCommentReply(ctx context.Context, req *connect.Request[api.CreateStoryCommentReplyRequest]) (*connect.Response[api.CreateStoryCommentReplyResponse], error) {
	traceId := getTraceID(ctx)
	userId := req.Msg.GetUserId()
	commentId := req.Msg.GetCommentId()
	replyContent := req.Msg.GetContent()
	zap.L().Info("CreateStoryCommentReply called", zap.String("traceId", traceId), zap.Int64("userId", userId), zap.Int64("commentId", commentId), zap.String("replyContent", maskContent(replyContent)))
	if userId <= 0 || commentId <= 0 || strings.TrimSpace(replyContent) == "" {
		zap.L().Warn("CreateStoryCommentReply param invalid", zap.String("traceId", traceId), zap.Int64("userId", userId), zap.Int64("commentId", commentId))
		return &connect.Response[api.CreateStoryCommentReplyResponse]{
			Msg: &api.CreateStoryCommentReplyResponse{
				Code:    api.ResponseCode_INVALID_PARAMETER,
				Message: "参数错误",
			},
		}, nil
	}
	ret, err := commentService.GetCommentServer().CreateStoryCommentReply(ctx, req.Msg)
	if err != nil {
		zap.L().Error("CreateStoryCommentReply failed", zap.String("traceId", traceId), zap.Error(err))
	}
	return &connect.Response[api.CreateStoryCommentReplyResponse]{
		Msg: ret,
	}, nil
}

func (ts *CommentService) DeleteStoryCommentReply(ctx context.Context, req *connect.Request[api.DeleteStoryCommentReplyRequest]) (*connect.Response[api.DeleteStoryCommentReplyResponse], error) {
	traceId := getTraceID(ctx)
	userId := req.Msg.GetUserId()
	replyId := req.Msg.GetReplyId()
	zap.L().Info("DeleteStoryCommentReply called", zap.String("traceId", traceId), zap.Int64("userId", userId), zap.Int64("replyId", replyId))
	if userId <= 0 || replyId <= 0 {
		zap.L().Warn("DeleteStoryCommentReply param invalid", zap.String("traceId", traceId), zap.Int64("userId", userId), zap.Int64("replyId", replyId))
		return &connect.Response[api.DeleteStoryCommentReplyResponse]{
			Msg: &api.DeleteStoryCommentReplyResponse{
				Code:    api.ResponseCode_INVALID_PARAMETER,
				Message: "参数错误",
			},
		}, nil
	}
	// 权限校验暂不做（如需请补充models包）
	ret, err := commentService.GetCommentServer().DeleteStoryCommentReply(ctx, req.Msg)
	if err != nil {
		zap.L().Error("DeleteStoryCommentReply failed", zap.String("traceId", traceId), zap.Error(err))
	}
	return &connect.Response[api.DeleteStoryCommentReplyResponse]{
		Msg: ret,
	}, nil
}

func (ts *CommentService) GetStoryBoardComments(ctx context.Context, req *connect.Request[api.GetStoryBoardCommentsRequest]) (*connect.Response[api.GetStoryBoardCommentsResponse], error) {
	traceId := getTraceID(ctx)
	boardId := req.Msg.GetBoardId()
	zap.L().Info("GetStoryBoardComments called", zap.String("traceId", traceId), zap.Int64("boardId", boardId))
	if boardId <= 0 {
		zap.L().Warn("GetStoryBoardComments param invalid", zap.String("traceId", traceId), zap.Int64("boardId", boardId))
		return &connect.Response[api.GetStoryBoardCommentsResponse]{
			Msg: &api.GetStoryBoardCommentsResponse{
				Code:    api.ResponseCode_INVALID_PARAMETER,
				Message: "参数错误",
			},
		}, nil
	}
	ret, err := commentService.GetCommentServer().GetStoryBoardComments(ctx, req.Msg)
	if err != nil {
		zap.L().Error("GetStoryBoardComments failed", zap.String("traceId", traceId), zap.Error(err))
	}
	return &connect.Response[api.GetStoryBoardCommentsResponse]{
		Msg: ret,
	}, nil
}

func (ts *CommentService) CreateStoryBoardComment(ctx context.Context, req *connect.Request[api.CreateStoryBoardCommentRequest]) (*connect.Response[api.CreateStoryBoardCommentResponse], error) {
	traceId := getTraceID(ctx)
	userId := req.Msg.GetUserId()
	boardId := req.Msg.GetBoardId()
	commentContent := req.Msg.GetContent()
	zap.L().Info("CreateStoryBoardComment called", zap.String("traceId", traceId), zap.Int64("userId", userId), zap.Int64("boardId", boardId), zap.String("commentContent", maskContent(commentContent)))
	if userId <= 0 || boardId <= 0 || strings.TrimSpace(commentContent) == "" {
		zap.L().Warn("CreateStoryBoardComment param invalid", zap.String("traceId", traceId), zap.Int64("userId", userId), zap.Int64("boardId", boardId))
		return &connect.Response[api.CreateStoryBoardCommentResponse]{
			Msg: &api.CreateStoryBoardCommentResponse{
				Code:    api.ResponseCode_INVALID_PARAMETER,
				Message: "参数错误",
			},
		}, nil
	}
	ret, err := commentService.GetCommentServer().CreateStoryBoardComment(ctx, req.Msg)
	if err != nil {
		zap.L().Error("CreateStoryBoardComment failed", zap.String("traceId", traceId), zap.Error(err))
	}
	return &connect.Response[api.CreateStoryBoardCommentResponse]{
		Msg: ret,
	}, nil
}

func (ts *CommentService) DeleteStoryBoardComment(ctx context.Context, req *connect.Request[api.DeleteStoryBoardCommentRequest]) (*connect.Response[api.DeleteStoryBoardCommentResponse], error) {
	traceId := getTraceID(ctx)
	userId := req.Msg.GetUserId()
	commentId := req.Msg.GetCommentId()
	zap.L().Info("DeleteStoryBoardComment called", zap.String("traceId", traceId), zap.Int64("userId", userId), zap.Int64("commentId", commentId))
	if userId <= 0 || commentId <= 0 {
		zap.L().Warn("DeleteStoryBoardComment param invalid", zap.String("traceId", traceId), zap.Int64("userId", userId), zap.Int64("commentId", commentId))
		return &connect.Response[api.DeleteStoryBoardCommentResponse]{
			Msg: &api.DeleteStoryBoardCommentResponse{
				Code:    api.ResponseCode_INVALID_PARAMETER,
				Message: "参数错误",
			},
		}, nil
	}
	// 权限校验暂不做（如需请补充models包）
	ret, err := commentService.GetCommentServer().DeleteStoryBoardComment(ctx, req.Msg)
	if err != nil {
		zap.L().Error("DeleteStoryBoardComment failed", zap.String("traceId", traceId), zap.Error(err))
	}
	return &connect.Response[api.DeleteStoryBoardCommentResponse]{
		Msg: ret,
	}, nil
}

func (ts *CommentService) GetStoryBoardCommentReplies(ctx context.Context, req *connect.Request[api.GetStoryBoardCommentRepliesRequest]) (*connect.Response[api.GetStoryBoardCommentRepliesResponse], error) {
	traceId := getTraceID(ctx)
	commentId := req.Msg.GetCommentId()
	zap.L().Info("GetStoryBoardCommentReplies called", zap.String("traceId", traceId), zap.Int64("commentId", commentId))
	if commentId <= 0 {
		zap.L().Warn("GetStoryBoardCommentReplies param invalid", zap.String("traceId", traceId), zap.Int64("commentId", commentId))
		return &connect.Response[api.GetStoryBoardCommentRepliesResponse]{
			Msg: &api.GetStoryBoardCommentRepliesResponse{
				Code:    api.ResponseCode_INVALID_PARAMETER,
				Message: "参数错误",
			},
		}, nil
	}
	// 保持原有空实现，避免找不到方法
	return &connect.Response[api.GetStoryBoardCommentRepliesResponse]{
		Msg: &api.GetStoryBoardCommentRepliesResponse{},
	}, nil
}

func (ts *CommentService) LikeComment(ctx context.Context, req *connect.Request[api.LikeCommentRequest]) (*connect.Response[api.LikeCommentResponse], error) {
	traceId := getTraceID(ctx)
	userId := req.Msg.GetUserId()
	commentId := req.Msg.GetCommentId()
	zap.L().Info("LikeComment called", zap.String("traceId", traceId), zap.Int64("userId", userId), zap.Int64("commentId", commentId))
	if userId <= 0 || commentId <= 0 {
		zap.L().Warn("LikeComment param invalid", zap.String("traceId", traceId), zap.Int64("userId", userId), zap.Int64("commentId", commentId))
		return &connect.Response[api.LikeCommentResponse]{
			Msg: &api.LikeCommentResponse{
				Code:    api.ResponseCode_INVALID_PARAMETER,
				Message: "参数错误",
			},
		}, nil
	}
	ret, err := commentService.GetCommentServer().LikeComment(ctx, req.Msg)
	if err != nil {
		zap.L().Error("LikeComment failed", zap.String("traceId", traceId), zap.Error(err))
	}
	return &connect.Response[api.LikeCommentResponse]{
		Msg: ret,
	}, nil
}

func (ts *CommentService) DislikeComment(ctx context.Context, req *connect.Request[api.DislikeCommentRequest]) (*connect.Response[api.DislikeCommentResponse], error) {
	traceId := getTraceID(ctx)
	userId := req.Msg.GetUserId()
	commentId := req.Msg.GetCommentId()
	zap.L().Info("DislikeComment called", zap.String("traceId", traceId), zap.Int64("userId", userId), zap.Int64("commentId", commentId))
	if userId <= 0 || commentId <= 0 {
		zap.L().Warn("DislikeComment param invalid", zap.String("traceId", traceId), zap.Int64("userId", userId), zap.Int64("commentId", commentId))
		return &connect.Response[api.DislikeCommentResponse]{
			Msg: &api.DislikeCommentResponse{
				Code:    api.ResponseCode_INVALID_PARAMETER,
				Message: "参数错误",
			},
		}, nil
	}
	ret, err := commentService.GetCommentServer().DislikeComment(ctx, req.Msg)
	if err != nil {
		zap.L().Error("DislikeComment failed", zap.String("traceId", traceId), zap.Error(err))
	}
	return &connect.Response[api.DislikeCommentResponse]{
		Msg: ret,
	}, nil
}
