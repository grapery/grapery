package group

import (
	// "net/http"
	"context"

	"connectrpc.com/connect"
	_ "github.com/gin-contrib/sessions"
	_ "github.com/gin-contrib/sessions/redis"

	api "github.com/grapery/common-protoc/gen"
)

type CommentService struct {
}

func (ts *CommentService) CreateStoryComment(ctx context.Context, req *connect.Request[api.CreateStoryCommentRequest]) (*connect.Response[api.CreateStoryCommentResponse], error) {
	return nil, nil
}

func (ts *CommentService) GetStoryComments(ctx context.Context, req *connect.Request[api.GetStoryCommentsRequest]) (*connect.Response[api.GetStoryCommentsResponse], error) {
	return nil, nil
}

func (ts *CommentService) DeleteStoryComment(ctx context.Context, req *connect.Request[api.DeleteStoryCommentRequest]) (*connect.Response[api.DeleteStoryCommentResponse], error) {
	return nil, nil
}

func (ts *CommentService) GetStoryCommentReplies(ctx context.Context, req *connect.Request[api.GetStoryCommentRepliesRequest]) (*connect.Response[api.GetStoryCommentRepliesResponse], error) {
	return nil, nil
}

func (ts *CommentService) CreateStoryCommentReply(ctx context.Context, req *connect.Request[api.CreateStoryCommentReplyRequest]) (*connect.Response[api.CreateStoryCommentReplyResponse], error) {
	return nil, nil
}

func (ts *CommentService) DeleteStoryCommentReply(ctx context.Context, req *connect.Request[api.DeleteStoryCommentReplyRequest]) (*connect.Response[api.DeleteStoryCommentReplyResponse], error) {
	return nil, nil
}

func (ts *CommentService) GetStoryBoardComments(ctx context.Context, req *connect.Request[api.GetStoryBoardCommentsRequest]) (*connect.Response[api.GetStoryBoardCommentsResponse], error) {
	return nil, nil
}

func (ts *CommentService) CreateStoryBoardComment(ctx context.Context, req *connect.Request[api.CreateStoryBoardCommentRequest]) (*connect.Response[api.CreateStoryBoardCommentResponse], error) {
	return nil, nil
}

func (ts *CommentService) DeleteStoryBoardComment(ctx context.Context, req *connect.Request[api.DeleteStoryBoardCommentRequest]) (*connect.Response[api.DeleteStoryBoardCommentResponse], error) {
	return nil, nil
}

func (ts *CommentService) GetStoryBoardCommentReplies(ctx context.Context, req *connect.Request[api.GetStoryBoardCommentRepliesRequest]) (*connect.Response[api.GetStoryBoardCommentRepliesResponse], error) {
	return nil, nil
}

func (ts *CommentService) LikeComment(ctx context.Context, req *connect.Request[api.LikeCommentRequest]) (*connect.Response[api.LikeCommentResponse], error) {
	return nil, nil
}

func (ts *CommentService) DislikeComment(ctx context.Context, req *connect.Request[api.DislikeCommentRequest]) (*connect.Response[api.DislikeCommentResponse], error) {
	return nil, nil
}
