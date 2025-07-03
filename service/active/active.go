package active

import (
	"context"

	connect "connectrpc.com/connect"

	api "github.com/grapery/common-protoc/gen"
)

type ActiveService struct {
}

// 获取用户/别的用户的活动
func (ts *ActiveService) FetchActives(ctx context.Context, req *connect.Request[api.FetchActivesRequest]) (*connect.Response[api.FetchActivesResponse], error) {
	return nil, nil
}
