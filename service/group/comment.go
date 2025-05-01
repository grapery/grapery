package group

import (
	// "net/http"
	"context"

	"connectrpc.com/connect"
	_ "github.com/gin-contrib/sessions"
	_ "github.com/gin-contrib/sessions/redis"
	api "github.com/grapery/common-protoc/gen"
	commentService "github.com/grapery/grapery/pkg/comment"
)

type CommentService struct {
}

func (ts *CommentService) CreateStoryComment(ctx context.Context, req *connect.Request[api.CreateStoryCommentRequest]) (*connect.Response[api.CreateStoryCommentResponse], error) {
	ret, err := commentService.GetCommentServer().CreateStoryComment(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	return &connect.Response[api.CreateStoryCommentResponse]{
		Msg: ret,
	}, nil
}

func (ts *CommentService) GetStoryComments(ctx context.Context, req *connect.Request[api.GetStoryCommentsRequest]) (*connect.Response[api.GetStoryCommentsResponse], error) {
	ret, err := commentService.GetCommentServer().GetStoryComments(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	return &connect.Response[api.GetStoryCommentsResponse]{
		Msg: ret,
	}, nil
}

func (ts *CommentService) DeleteStoryComment(ctx context.Context, req *connect.Request[api.DeleteStoryCommentRequest]) (*connect.Response[api.DeleteStoryCommentResponse], error) {
	ret, err := commentService.GetCommentServer().DeleteStoryComment(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	return &connect.Response[api.DeleteStoryCommentResponse]{
		Msg: ret,
	}, nil
}

func (ts *CommentService) GetStoryCommentReplies(ctx context.Context, req *connect.Request[api.GetStoryCommentRepliesRequest]) (*connect.Response[api.GetStoryCommentRepliesResponse], error) {
	ret, err := commentService.GetCommentServer().GetStoryCommentReplies(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	println("get story comment replies success", ret.String())
	return &connect.Response[api.GetStoryCommentRepliesResponse]{
		Msg: ret,
	}, nil
}

func (ts *CommentService) CreateStoryCommentReply(ctx context.Context, req *connect.Request[api.CreateStoryCommentReplyRequest]) (*connect.Response[api.CreateStoryCommentReplyResponse], error) {
	ret, err := commentService.GetCommentServer().CreateStoryCommentReply(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	return &connect.Response[api.CreateStoryCommentReplyResponse]{
		Msg: ret,
	}, nil
}

func (ts *CommentService) DeleteStoryCommentReply(ctx context.Context, req *connect.Request[api.DeleteStoryCommentReplyRequest]) (*connect.Response[api.DeleteStoryCommentReplyResponse], error) {
	ret, err := commentService.GetCommentServer().DeleteStoryCommentReply(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	return &connect.Response[api.DeleteStoryCommentReplyResponse]{
		Msg: ret,
	}, nil
}

func (ts *CommentService) GetStoryBoardComments(ctx context.Context, req *connect.Request[api.GetStoryBoardCommentsRequest]) (*connect.Response[api.GetStoryBoardCommentsResponse], error) {
	ret, err := commentService.GetCommentServer().GetStoryBoardComments(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	println("get story board comments success", ret.String())
	return &connect.Response[api.GetStoryBoardCommentsResponse]{
		Msg: ret,
	}, nil
}

func (ts *CommentService) CreateStoryBoardComment(ctx context.Context, req *connect.Request[api.CreateStoryBoardCommentRequest]) (*connect.Response[api.CreateStoryBoardCommentResponse], error) {
	ret, err := commentService.GetCommentServer().CreateStoryBoardComment(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	return &connect.Response[api.CreateStoryBoardCommentResponse]{
		Msg: ret,
	}, nil
}

func (ts *CommentService) DeleteStoryBoardComment(ctx context.Context, req *connect.Request[api.DeleteStoryBoardCommentRequest]) (*connect.Response[api.DeleteStoryBoardCommentResponse], error) {
	ret, err := commentService.GetCommentServer().DeleteStoryBoardComment(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	return &connect.Response[api.DeleteStoryBoardCommentResponse]{
		Msg: ret,
	}, nil
}

func (ts *CommentService) GetStoryBoardCommentReplies(ctx context.Context, req *connect.Request[api.GetStoryBoardCommentRepliesRequest]) (*connect.Response[api.GetStoryBoardCommentRepliesResponse], error) {
	return &connect.Response[api.GetStoryBoardCommentRepliesResponse]{
		Msg: &api.GetStoryBoardCommentRepliesResponse{},
	}, nil
}

func (ts *CommentService) LikeComment(ctx context.Context, req *connect.Request[api.LikeCommentRequest]) (*connect.Response[api.LikeCommentResponse], error) {
	ret, err := commentService.GetCommentServer().LikeComment(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	return &connect.Response[api.LikeCommentResponse]{
		Msg: ret,
	}, nil
}

func (ts *CommentService) DislikeComment(ctx context.Context, req *connect.Request[api.DislikeCommentRequest]) (*connect.Response[api.DislikeCommentResponse], error) {
	ret, err := commentService.GetCommentServer().DislikeComment(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	return &connect.Response[api.DislikeCommentResponse]{
		Msg: ret,
	}, nil
}
