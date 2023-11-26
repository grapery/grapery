package common

import (
	// "net/http"
	"context"

	"connectrpc.com/connect"

	api "github.com/grapery/common-protoc/gen"
)

type CommonService struct {
}

// default is project
func (cs *CommonService) Explore(ctx context.Context, req *connect.Request[api.ExploreRequest]) (*connect.Response[api.ExploreResponse], error) {
	return nil, nil
}

// default is project
func (cs *CommonService) Trending(ctx context.Context, req *connect.Request[api.TrendingRequest]) (*connect.Response[api.TrendingResponse], error) {
	return nil, nil
}
