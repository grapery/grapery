package active

import (
	"context"

	connect "connectrpc.com/connect"

	api "github.com/grapery/common-protoc/gen"
)

type ActiveService struct {
}

// 1.获取指定小组的活动流
// 2.获取所用小组的活动流，混合展示
// 3.获取指定用户的活动流，前提是指定的用户是公开的
// 4.角色暂时没有活动流
// 活动流是展示小组内或者用户的活动情况，让用户、角色能够快速的发现有用的信息
// 获取用户/别的用户的活动
func (ts *ActiveService) FetchActives(ctx context.Context, req *connect.Request[api.FetchActivesRequest]) (*connect.Response[api.FetchActivesResponse], error) {
	return nil, nil
}
