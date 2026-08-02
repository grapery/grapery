package common

import (
	// "net/http"
	"context"

	connect "connectrpc.com/connect"

	api "github.com/grapery/common-protoc/gen"
)

type CommonService struct {
}

// default is project
func (cs *CommonService) Explore(ctx context.Context, req *connect.Request[api.ExploreRequest]) (*connect.Response[api.ExploreResponse], error) {
	return nil, nil
}

func (cs *CommonService) UploadImageFile(ctx context.Context, req *connect.Request[api.UploadImageRequest]) (*connect.Response[api.UploadImageResponse], error) {
	return nil, nil
}
