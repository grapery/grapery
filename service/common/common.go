package common

import (
	// "net/http"
	"context"
	"errors"
	"strings"

	"connectrpc.com/connect"

	api "github.com/grapery/common-protoc/gen"
	tencentcloud "github.com/grapery/grapery/pkg/cloud/tencentcloud"
)

type CommonService struct {
}

// default is project
func (cs *CommonService) Explore(ctx context.Context, req *connect.Request[api.ExploreRequest]) (*connect.Response[api.ExploreResponse], error) {
	return nil, nil
}

func (cs *CommonService) UploadImageFile(ctx context.Context, req *connect.Request[api.UploadImageRequest]) (*connect.Response[api.UploadImageResponse], error) {
	if req.Msg.ImageData == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("image data is empty"))
	}
	if len(req.Msg.ImageData) > 1024*1024*10 {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("image data is too large"))
	}
	var fileFormat string
	if strings.Contains(req.Msg.Filename, ".png") {
		fileFormat = "png"
	} else if strings.Contains(req.Msg.Filename, ".gif") {
		fileFormat = "gif"
	} else if strings.Contains(req.Msg.Filename, ".jpeg") {
		fileFormat = "jpeg"
	} else if strings.Contains(req.Msg.Filename, ".bmp") {
		fileFormat = "bmp"
	} else if strings.Contains(req.Msg.Filename, ".svg") {
		fileFormat = "svg"
	} else if strings.Contains(req.Msg.Filename, ".jpg") {
		fileFormat = "jpg"
	} else if strings.Contains(req.Msg.Filename, ".mp4") {
		fileFormat = "mp4"
	} else if strings.Contains(req.Msg.Filename, ".txt") {
		fileFormat = "txt"
	} else {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("unsupported image format"))
	}
	// TODO: save image data to storage
	filepath, err := tencentcloud.UploadObject(req.Msg.ImageData, fileFormat)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return &connect.Response[api.UploadImageResponse]{
		Msg: &api.UploadImageResponse{
			Code:    0,
			Message: "upload image success",
			Data: &api.UploadImageResponse_Data{
				Url: filepath,
			},
		},
	}, nil
}
