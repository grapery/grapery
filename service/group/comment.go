package group

import (
	// "net/http"
	"context"

	_ "github.com/gin-contrib/sessions"
	_ "github.com/gin-contrib/sessions/redis"

	api "github.com/grapery/common-protoc/gen"
)

type CommentService struct {
}

func (ts *CommentService) CreateComment(ctx context.Context, req *api.CreateCommentReq) (*api.CreateCommentResp, error) {
	return nil, nil
}

func (ts *CommentService) AppendComment(ctx context.Context, req *api.CreateCommentReq) (*api.CreateCommentResp, error) {
	return nil, nil
}

func (ts *CommentService) EmojiComment(ctx context.Context, req *api.CreateCommentReq) (*api.CreateCommentResp, error) {
	return nil, nil
}

func (ts *CommentService) GetItemComment(ctx context.Context, req *api.GetItemCommentReq) (*api.GetItemCommentResp, error) {
	return nil, nil
}

func (ts *CommentService) DeleteComment(ctx context.Context, req *api.CreateCommentReq) (*api.CreateCommentResp, error) {
	return nil, nil
}
