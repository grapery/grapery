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

func (ts *CommentService) CreateComment(ctx context.Context, req *connect.Request[api.CreateCommentReq]) (*connect.Response[api.CreateCommentResp], error) {
	return nil, nil
}

func (ts *CommentService) AppendComment(ctx context.Context, req *connect.Request[api.CreateCommentReq]) (*connect.Response[api.CreateCommentResp], error) {
	return nil, nil
}

func (ts *CommentService) EmojiComment(ctx context.Context, req *connect.Request[api.CreateCommentReq]) (*connect.Response[api.CreateCommentResp], error) {
	return nil, nil
}

func (ts *CommentService) DeleteComment(ctx context.Context, req *connect.Request[api.CreateCommentReq]) (*connect.Response[api.CreateCommentResp], error) {
	return nil, nil
}

func (ts *CommentService) GetItemComment(ctx context.Context, req *connect.Request[api.GetItemsCommentReq]) (*connect.Response[api.GetItemsCommentResp], error) {
	return nil, nil
}
