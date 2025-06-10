package group

import (
	"context"

	"github.com/bufbuild/connect-go"
	api "github.com/grapery/common-protoc/gen"
)

func (s *StoryBoardService) QueryTaskStatus(ctx context.Context, req *connect.Request[api.QueryTaskStatusRequest]) (*connect.Response[api.QueryTaskStatusResponse], error) {
	return connect.NewResponse(&api.QueryTaskStatusResponse{
		Code:    0,
		Message: "OK",
	}), nil
}
