package common

import (
	// "net/http"
	"context"

	"github.com/grapery/grapery/api"
)

type CommonService struct {
}

// default is project
func (cs *CommonService) Explore(ctx context.Context, req *api.ExploreRequest) (*api.ExploreResponse, error) {
	return &api.ExploreResponse{}, nil
}

// default is project
func (cs *CommonService) Trending(ctx context.Context, req *api.TrendingRequest) (*api.TrendingResponse, error) {
	return &api.TrendingResponse{}, nil
}
