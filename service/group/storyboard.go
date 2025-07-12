package group

import (
	"context"
	"encoding/json"
	"log"

	"go.uber.org/zap"

	connect "connectrpc.com/connect"

	"github.com/grapery/common-protoc/gen"
	storyServer "github.com/grapery/grapery/pkg/story"
)

type StoryBoardService struct {
}

func (s *StoryBoardService) CreateStoryboard(ctx context.Context, req *connect.Request[gen.CreateStoryboardRequest]) (*connect.Response[gen.CreateStoryboardResponse], error) {
	traceId := getTraceID(ctx)
	zap.L().Info("CreateStoryboard called", zap.String("traceId", traceId), zap.String("title", maskContent(req.Msg.GetBoard().GetTitle())))
	ret, err := storyServer.GetStoryServer().CreateStoryboard(ctx, req.Msg)
	if err != nil {
		zap.L().Error("CreateStoryboard failed", zap.String("traceId", traceId), zap.Error(err))
		return nil, err
	}
	resp := &gen.CreateStoryboardResponse{
		Code:    ret.Code,
		Message: "OK",
		Data: &gen.CreateStoryboardResponse_Data{
			BoardId: int64(ret.Data.BoardId),
		},
	}
	zap.L().Info("CreateStoryboard success", zap.String("traceId", traceId), zap.Int64("boardId", int64(ret.Data.BoardId)))
	return connect.NewResponse(resp), nil
}

func (s *StoryBoardService) GetStoryboard(ctx context.Context, req *connect.Request[gen.GetStoryboardRequest]) (*connect.Response[gen.GetStoryboardResponse], error) {
	traceId := getTraceID(ctx)
	zap.L().Info("GetStoryboard called", zap.String("traceId", traceId), zap.Int64("boardId", req.Msg.BoardId))
	ret, err := storyServer.GetStoryServer().GetStoryboard(ctx, req.Msg)
	if err != nil {
		zap.L().Error("GetStoryboard failed", zap.String("traceId", traceId), zap.Error(err))
		return nil, err
	}
	retData, _ := json.Marshal(ret)
	zap.L().Info("GetStoryboard result", zap.String("traceId", traceId), zap.String("data", string(retData)))
	resp := &gen.GetStoryboardResponse{
		Code:    ret.Code,
		Message: "OK",
		Data: &gen.GetStoryboardResponse_Data{
			BoardInfo: ret.Data.BoardInfo,
		},
	}
	zap.L().Info("GetStoryboard success", zap.String("traceId", traceId), zap.Int64("boardId", req.Msg.BoardId))
	return connect.NewResponse(resp), nil
}

func (s *StoryBoardService) GetStoryboards(ctx context.Context, req *connect.Request[gen.GetStoryboardsRequest]) (*connect.Response[gen.GetStoryboardsResponse], error) {
	traceId := getTraceID(ctx)
	zap.L().Info("GetStoryboards called", zap.String("traceId", traceId), zap.String("userId", req.Msg.String()))
	ret, err := storyServer.GetStoryServer().GetStoryboards(ctx, req.Msg)
	if err != nil {
		zap.L().Error("GetStoryboards failed", zap.String("traceId", traceId), zap.Error(err))
		return nil, err
	}
	resp := &gen.GetStoryboardsResponse{
		Code:    ret.Code,
		Message: "OK",
		Data: &gen.GetStoryboardsResponse_Data{
			List:  ret.Data.List,
			Total: ret.Data.Total,
		},
	}
	return connect.NewResponse(resp), nil
}

func (s *StoryBoardService) DelStoryboard(ctx context.Context, req *connect.Request[gen.DelStoryboardRequest]) (*connect.Response[gen.DelStoryboardResponse], error) {
	traceId := getTraceID(ctx)
	zap.L().Info("DelStoryboard called", zap.String("traceId", traceId), zap.Int64("boardId", req.Msg.BoardId))
	ret, err := storyServer.GetStoryServer().DelStoryboard(ctx, req.Msg)
	if err != nil {
		zap.L().Error("DelStoryboard failed", zap.String("traceId", traceId), zap.Error(err))
		return nil, err
	}
	resp := &gen.DelStoryboardResponse{
		Code:    ret.Code,
		Message: "OK",
	}
	zap.L().Info("DelStoryboard success", zap.String("traceId", traceId), zap.Int64("boardId", req.Msg.BoardId))
	return connect.NewResponse(resp), nil
}

func (s *StoryBoardService) ForkStoryboard(ctx context.Context, req *connect.Request[gen.ForkStoryboardRequest]) (*connect.Response[gen.ForkStoryboardResponse], error) {
	traceId := getTraceID(ctx)
	zap.L().Info("ForkStoryboard called", zap.String("traceId", traceId), zap.String("req", req.Msg.String()))
	ret, err := storyServer.GetStoryServer().ForkStoryboard(ctx, req.Msg)
	if err != nil {
		zap.L().Error("ForkStoryboard failed", zap.String("traceId", traceId), zap.Error(err))
		return nil, err
	}
	resp := &gen.ForkStoryboardResponse{
		Code:    ret.Code,
		Message: "OK",
		Data:    ret.Data,
	}
	return connect.NewResponse(resp), nil
}

func (s *StoryBoardService) LikeStoryboard(ctx context.Context, req *connect.Request[gen.LikeStoryboardRequest]) (*connect.Response[gen.LikeStoryboardResponse], error) {
	traceId := getTraceID(ctx)
	zap.L().Info("LikeStoryboard called", zap.String("traceId", traceId), zap.String("req", req.Msg.String()))
	ret, err := storyServer.GetStoryServer().LikeStoryboard(ctx, req.Msg)
	if err != nil {
		zap.L().Error("LikeStoryboard failed", zap.String("traceId", traceId), zap.Error(err))
		return nil, err
	}
	resp := &gen.LikeStoryboardResponse{
		Code:    ret.Code,
		Message: "OK",
	}
	return connect.NewResponse(resp), nil
}

func (s *StoryBoardService) ShareStoryboard(ctx context.Context, req *connect.Request[gen.ShareStoryboardRequest]) (*connect.Response[gen.ShareStoryboardResponse], error) {
	traceId := getTraceID(ctx)
	zap.L().Info("ShareStoryboard called", zap.String("traceId", traceId), zap.String("req", req.Msg.String()))
	ret, err := storyServer.GetStoryServer().ShareStoryboard(ctx, req.Msg)
	if err != nil {
		zap.L().Error("ShareStoryboard failed", zap.String("traceId", traceId), zap.Error(err))
		return nil, err
	}
	resp := &gen.ShareStoryboardResponse{
		Code:    ret.Code,
		Message: "OK",
	}
	return connect.NewResponse(resp), nil
}

func (s *StoryBoardService) UpdateStoryboard(ctx context.Context, req *connect.Request[gen.UpdateStoryboardRequest]) (*connect.Response[gen.UpdateStoryboardResponse], error) {
	traceId := getTraceID(ctx)
	zap.L().Info("UpdateStoryboard called", zap.String("traceId", traceId), zap.String("req", req.Msg.String()))
	ret, err := storyServer.GetStoryServer().UpdateStoryboard(ctx, req.Msg)
	if err != nil {
		zap.L().Error("UpdateStoryboard failed", zap.String("traceId", traceId), zap.Error(err))
		return nil, err
	}
	resp := &gen.UpdateStoryboardResponse{
		Code:    ret.Code,
		Message: "OK",
	}
	zap.L().Info("UpdateStoryboard success", zap.String("traceId", traceId), zap.Int64("boardId", req.Msg.BoardId))
	return connect.NewResponse(resp), nil
}

func (s *StoryBoardService) RenderStoryboard(ctx context.Context, req *connect.Request[gen.RenderStoryboardRequest]) (*connect.Response[gen.RenderStoryboardResponse], error) {
	traceId := getTraceID(ctx)
	zap.L().Info("RenderStoryboard called", zap.String("traceId", traceId), zap.Int64("boardId", req.Msg.BoardId))
	ret, err := storyServer.GetStoryServer().RenderStoryboard(ctx, req.Msg)
	if err != nil {
		zap.L().Error("RenderStoryboard failed", zap.String("traceId", traceId), zap.Error(err))
		return nil, err
	}
	resp := &gen.RenderStoryboardResponse{
		Code:    ret.Code,
		Message: "OK",
		Data:    ret.Data,
	}
	zap.L().Info("RenderStoryboard success", zap.String("traceId", traceId), zap.Int64("boardId", req.Msg.BoardId))
	return connect.NewResponse(resp), nil
}

func (s *StoryBoardService) GenStoryboardImages(ctx context.Context, req *connect.Request[gen.GenStoryboardImagesRequest]) (*connect.Response[gen.GenStoryboardImagesResponse], error) {
	traceId := getTraceID(ctx)
	zap.L().Info("GenStoryboardImages called", zap.String("traceId", traceId), zap.Int64("boardId", req.Msg.BoardId))
	ret, err := storyServer.GetStoryServer().GenStoryboardImages(ctx, req.Msg)
	if err != nil {
		zap.L().Error("GenStoryboardImages failed", zap.String("traceId", traceId), zap.Error(err))
		return nil, err
	}
	resp := &gen.GenStoryboardImagesResponse{
		Code:    ret.Code,
		Message: "OK",
	}
	zap.L().Info("GenStoryboardImages success", zap.String("traceId", traceId), zap.Int64("boardId", req.Msg.BoardId))
	return connect.NewResponse(resp), nil
}

func (s *StoryBoardService) GenStoryboardText(ctx context.Context, req *connect.Request[gen.GenStoryboardTextRequest]) (*connect.Response[gen.GenStoryboardTextResponse], error) {
	traceId := getTraceID(ctx)
	zap.L().Info("GenStoryboardText called", zap.String("traceId", traceId), zap.Int64("boardId", req.Msg.BoardId))
	ret, err := storyServer.GetStoryServer().GenStoryboardText(ctx, req.Msg)
	if err != nil {
		zap.L().Error("GenStoryboardText failed", zap.String("traceId", traceId), zap.Error(err))
		return nil, err
	}
	resp := &gen.GenStoryboardTextResponse{
		Code:    ret.Code,
		Message: "OK",
	}
	zap.L().Info("GenStoryboardText success", zap.String("traceId", traceId), zap.Int64("boardId", req.Msg.BoardId))
	return connect.NewResponse(resp), nil
}

func (s *StoryBoardService) GetStoryBoardRender(ctx context.Context, req *connect.Request[gen.GetStoryBoardRenderRequest]) (*connect.Response[gen.GetStoryBoardRenderResponse], error) {
	traceId := getTraceID(ctx)
	zap.L().Info("GetStoryBoardRender called", zap.String("traceId", traceId), zap.Int64("boardId", req.Msg.BoardId))
	ret, err := storyServer.GetStoryServer().GetStoryBoardRender(ctx, req.Msg)
	if err != nil {
		zap.L().Error("GetStoryBoardRender failed", zap.String("traceId", traceId), zap.Error(err))
		return nil, err
	}
	resp := &gen.GetStoryBoardRenderResponse{
		Code:    ret.Code,
		Message: "OK",
		Data:    ret.Data,
	}
	zap.L().Info("GetStoryBoardRender success", zap.String("traceId", traceId), zap.Int64("boardId", req.Msg.BoardId))
	return connect.NewResponse(resp), nil
}

func (s *StoryBoardService) GetStoryBoardRoles(ctx context.Context, req *connect.Request[gen.GetStoryBoardRolesRequest]) (*connect.Response[gen.GetStoryBoardRolesResponse], error) {
	traceId := getTraceID(ctx)
	zap.L().Info("GetStoryBoardRoles called", zap.String("traceId", traceId), zap.Int64("boardId", req.Msg.BoardId))
	ret, err := storyServer.GetStoryServer().GetStoryBoardRoles(ctx, req.Msg)
	if err != nil {
		zap.L().Error("GetStoryBoardRoles failed", zap.String("traceId", traceId), zap.Error(err))
		return nil, err
	}
	resp := &gen.GetStoryBoardRolesResponse{
		Code:    ret.Code,
		Message: "OK",
		Data:    ret.Data,
	}
	zap.L().Info("GetStoryBoardRoles success", zap.String("traceId", traceId), zap.Int64("boardId", req.Msg.BoardId))
	return connect.NewResponse(resp), nil
}

func (s *StoryBoardService) UnLikeStoryboard(ctx context.Context, req *connect.Request[gen.UnLikeStoryboardRequest]) (*connect.Response[gen.UnLikeStoryboardResponse], error) {
	traceId := getTraceID(ctx)
	zap.L().Info("UnLikeStoryboard called", zap.String("traceId", traceId), zap.Int64("boardId", req.Msg.BoardId), zap.Int64("userId", req.Msg.UserId))
	ret, err := storyServer.GetStoryServer().UnLikeStoryboard(ctx, req.Msg)
	if err != nil {
		zap.L().Error("UnLikeStoryboard failed", zap.String("traceId", traceId), zap.Error(err))
		return nil, err
	}
	resp := &gen.UnLikeStoryboardResponse{
		Code:    ret.Code,
		Message: "OK",
	}
	zap.L().Info("UnLikeStoryboard success", zap.String("traceId", traceId), zap.Int64("boardId", req.Msg.BoardId))
	return connect.NewResponse(resp), nil
}

func (s *StoryBoardService) GetStoryBoardSences(ctx context.Context, req *connect.Request[gen.GetStoryBoardSencesRequest]) (*connect.Response[gen.GetStoryBoardSencesResponse], error) {
	traceId := getTraceID(ctx)
	zap.L().Info("GetStoryBoardSences called", zap.String("traceId", traceId), zap.Int64("boardId", req.Msg.BoardId))
	ret, err := storyServer.GetStoryServer().GetStoryboardScene(ctx, req.Msg)
	if err != nil {
		zap.L().Error("GetStoryBoardSences failed", zap.String("traceId", traceId), zap.Error(err))
		return nil, err
	}
	resp := &gen.GetStoryBoardSencesResponse{
		Code:    ret.Code,
		Message: "OK",
		Data:    ret.Data,
	}
	zap.L().Info("GetStoryBoardSences success", zap.String("traceId", traceId), zap.Int64("boardId", req.Msg.BoardId))
	return connect.NewResponse(resp), nil
}

func (s *StoryBoardService) CreateStoryBoardSence(ctx context.Context, req *connect.Request[gen.CreateStoryBoardSenceRequest]) (*connect.Response[gen.CreateStoryBoardSenceResponse], error) {
	traceId := getTraceID(ctx)
	zap.L().Info("CreateStoryBoardSence called", zap.String("traceId", traceId), zap.String("req", req.Msg.String()))
	ret, err := storyServer.GetStoryServer().CreateStoryBoardScene(ctx, req.Msg)
	if err != nil {
		zap.L().Error("CreateStoryBoardSence failed", zap.String("traceId", traceId), zap.Error(err))
		return nil, err
	}
	resp := &gen.CreateStoryBoardSenceResponse{
		Code:    ret.Code,
		Message: "OK",
		Data:    ret.Data,
	}
	return connect.NewResponse(resp), nil
}

func (s *StoryBoardService) UpdateStoryBoardSence(ctx context.Context, req *connect.Request[gen.UpdateStoryBoardSenceRequest]) (*connect.Response[gen.UpdateStoryBoardSenceResponse], error) {
	traceId := getTraceID(ctx)
	zap.L().Info("UpdateStoryBoardSence called", zap.String("traceId", traceId), zap.String("req", req.Msg.String()))
	ret, err := storyServer.GetStoryServer().UpdateStoryBoardSence(ctx, req.Msg)
	if err != nil {
		zap.L().Error("UpdateStoryBoardSence failed", zap.String("traceId", traceId), zap.Error(err))
		return nil, err
	}
	resp := &gen.UpdateStoryBoardSenceResponse{
		Code:    ret.Code,
		Message: "OK",
	}
	return connect.NewResponse(resp), nil
}

func (s *StoryBoardService) DeleteStoryBoardSence(ctx context.Context, req *connect.Request[gen.DeleteStoryBoardSenceRequest]) (*connect.Response[gen.DeleteStoryBoardSenceResponse], error) {
	traceId := getTraceID(ctx)
	zap.L().Info("DeleteStoryBoardSence called", zap.String("traceId", traceId), zap.String("req", req.Msg.String()))
	ret, err := storyServer.GetStoryServer().DeleteStoryBoardSence(ctx, req.Msg)
	if err != nil {
		zap.L().Error("DeleteStoryBoardSence failed", zap.String("traceId", traceId), zap.Error(err))
		return nil, err
	}
	resp := &gen.DeleteStoryBoardSenceResponse{
		Code:    ret.Code,
		Message: "OK",
	}
	return connect.NewResponse(resp), nil
}

func (s *StoryBoardService) RenderStoryBoardSence(ctx context.Context, req *connect.Request[gen.RenderStoryBoardSenceRequest]) (*connect.Response[gen.RenderStoryBoardSenceResponse], error) {
	traceId := getTraceID(ctx)
	zap.L().Info("RenderStoryBoardSence called", zap.String("traceId", traceId), zap.String("req", req.Msg.String()))
	ret, err := storyServer.GetStoryServer().RenderStoryBoardSence(ctx, req.Msg)
	if err != nil {
		zap.L().Error("RenderStoryBoardSence failed", zap.String("traceId", traceId), zap.Error(err))
		return nil, err
	}
	resp := &gen.RenderStoryBoardSenceResponse{
		Code:    ret.Code,
		Message: "OK",
		Data:    ret.Data,
	}
	return connect.NewResponse(resp), nil
}

func (s *StoryBoardService) GetStoryBoardSenceGenerate(ctx context.Context, req *connect.Request[gen.GetStoryBoardSenceGenerateRequest]) (*connect.Response[gen.GetStoryBoardSenceGenerateResponse], error) {
	traceId := getTraceID(ctx)
	zap.L().Info("GetStoryBoardSenceGenerate called", zap.String("traceId", traceId), zap.String("req", req.Msg.String()))
	ret, err := storyServer.GetStoryServer().GetStoryBoardSenceGenerate(ctx, req.Msg)
	if err != nil {
		zap.L().Error("GetStoryBoardSenceGenerate failed", zap.String("traceId", traceId), zap.Error(err))
		return nil, err
	}
	resp := &gen.GetStoryBoardSenceGenerateResponse{
		Code:    ret.Code,
		Message: "OK",
		Data:    ret.Data,
	}
	return connect.NewResponse(resp), nil
}

func (s *StoryBoardService) GetStoryBoardGenerate(ctx context.Context, req *connect.Request[gen.GetStoryBoardGenerateRequest]) (*connect.Response[gen.GetStoryBoardGenerateResponse], error) {
	traceId := getTraceID(ctx)
	zap.L().Info("GetStoryBoardGenerate called", zap.String("traceId", traceId), zap.String("req", req.Msg.String()))
	ret, err := storyServer.GetStoryServer().GetStoryBoardGenerate(ctx, req.Msg)
	if err != nil {
		zap.L().Error("GetStoryBoardGenerate failed", zap.String("traceId", traceId), zap.Error(err))
		return nil, err
	}
	resp := &gen.GetStoryBoardGenerateResponse{
		Code:    ret.Code,
		Message: "OK",
		List:    ret.List,
	}
	return connect.NewResponse(resp), nil
}

func (s *StoryBoardService) RenderStoryBoardSences(ctx context.Context, req *connect.Request[gen.RenderStoryBoardSencesRequest]) (*connect.Response[gen.RenderStoryBoardSencesResponse], error) {
	traceId := getTraceID(ctx)
	zap.L().Info("RenderStoryBoardSences called", zap.String("traceId", traceId), zap.String("req", req.Msg.String()))
	ret, err := storyServer.GetStoryServer().RenderStoryBoardSences(ctx, req.Msg)
	if err != nil {
		zap.L().Error("RenderStoryBoardSences failed", zap.String("traceId", traceId), zap.Error(err))
		return nil, err
	}
	resp := &gen.RenderStoryBoardSencesResponse{
		Code:    ret.Code,
		Message: "OK",
		List:    ret.List,
	}
	return connect.NewResponse(resp), nil
}

func (s *StoryBoardService) RestoreStoryboard(ctx context.Context, req *connect.Request[gen.RestoreStoryboardRequest]) (*connect.Response[gen.RestoreStoryboardResponse], error) {
	traceId := getTraceID(ctx)
	zap.L().Info("RestoreStoryboard called", zap.String("traceId", traceId), zap.String("req", req.Msg.String()))
	ret, err := storyServer.GetStoryServer().RestoreStoryboard(ctx, req.Msg)
	if err != nil {
		zap.L().Error("RestoreStoryboard failed", zap.String("traceId", traceId), zap.Error(err))
		return nil, err
	}
	resp := &gen.RestoreStoryboardResponse{
		Code:    ret.Code,
		Message: "OK",
	}
	return connect.NewResponse(resp), nil
}

// 获取用户创建的故事板
func (s *StoryBoardService) GetUserCreatedStoryboards(ctx context.Context, req *connect.Request[gen.GetUserCreatedStoryboardsRequest]) (*connect.Response[gen.GetUserCreatedStoryboardsResponse], error) {
	traceId := getTraceID(ctx)
	zap.L().Info("GetUserCreatedStoryboards called", zap.String("traceId", traceId), zap.String("req", req.Msg.String()))
	ret, err := storyServer.GetStoryServer().GetUserCreatedStoryboards(ctx, req.Msg)
	if err != nil {
		zap.L().Error("GetUserCreatedStoryboards failed", zap.String("traceId", traceId), zap.Error(err))
		return nil, err
	}
	resp := &gen.GetUserCreatedStoryboardsResponse{
		Code:        ret.Code,
		Message:     "OK",
		Total:       ret.Total,
		Offset:      ret.Offset,
		PageSize:    ret.PageSize,
		Storyboards: ret.Storyboards,
	}
	return connect.NewResponse(resp), nil
}

func (s *StoryBoardService) PublishStoryboard(ctx context.Context, req *connect.Request[gen.PublishStoryboardRequest]) (*connect.Response[gen.PublishStoryboardResponse], error) {
	traceId := getTraceID(ctx)
	zap.L().Info("PublishStoryboard called", zap.String("traceId", traceId), zap.String("req", req.Msg.String()))
	ret, err := storyServer.GetStoryServer().PublishStoryboard(ctx, req.Msg)
	if err != nil {
		zap.L().Error("PublishStoryboard failed", zap.String("traceId", traceId), zap.Error(err))
		return nil, err
	}
	return connect.NewResponse(ret), nil
}

func (s *StoryBoardService) CancelStoryboard(ctx context.Context, req *connect.Request[gen.CancelStoryboardRequest]) (*connect.Response[gen.CancelStoryboardResponse], error) {
	traceId := getTraceID(ctx)
	zap.L().Info("CancelStoryboard called", zap.String("traceId", traceId), zap.String("req", req.Msg.String()))
	ret, err := storyServer.GetStoryServer().CancelStoryboard(ctx, req.Msg)
	if err != nil {
		zap.L().Error("CancelStoryboard failed", zap.String("traceId", traceId), zap.Error(err))
		return nil, err
	}
	return connect.NewResponse(ret), nil
}

func (s *StoryBoardService) GetUserWatchStoryActiveStoryBoards(ctx context.Context, req *connect.Request[gen.GetUserWatchStoryActiveStoryBoardsRequest]) (*connect.Response[gen.GetUserWatchStoryActiveStoryBoardsResponse], error) {
	traceId := getTraceID(ctx)
	zap.L().Info("GetUserWatchStoryActiveStoryBoards called", zap.String("traceId", traceId), zap.String("req", req.Msg.String()))
	ret, err := storyServer.GetStoryServer().GetUserWatchStoryActiveStoryBoards(ctx, req.Msg)
	if err != nil {
		zap.L().Error("GetUserWatchStoryActiveStoryBoards failed", zap.String("traceId", traceId), zap.Error(err))
		return nil, err
	}
	return connect.NewResponse(ret), nil
}

func (s *StoryBoardService) GetUserWatchRoleActiveStoryBoards(ctx context.Context, req *connect.Request[gen.GetUserWatchRoleActiveStoryBoardsRequest]) (*connect.Response[gen.GetUserWatchRoleActiveStoryBoardsResponse], error) {
	traceId := getTraceID(ctx)
	zap.L().Info("GetUserWatchRoleActiveStoryBoards called", zap.String("traceId", traceId), zap.String("req", req.Msg.String()))
	ret, err := storyServer.GetStoryServer().GetUserWatchRoleActiveStoryBoards(ctx, req.Msg)
	if err != nil {
		zap.L().Error("GetUserWatchRoleActiveStoryBoards failed", zap.String("traceId", traceId), zap.Error(err))
		return nil, err
	}
	log.Printf("GetUserWatchRoleActiveStoryBoards: %v", ret.String())
	return connect.NewResponse(ret), nil
}

func (s *StoryBoardService) GetUnPublishStoryboard(ctx context.Context, req *connect.Request[gen.GetUnPublishStoryboardRequest]) (*connect.Response[gen.GetUnPublishStoryboardResponse], error) {
	traceId := getTraceID(ctx)
	zap.L().Info("GetUnPublishStoryboard called", zap.String("traceId", traceId), zap.String("req", req.Msg.String()))
	ret, err := storyServer.GetStoryServer().GetUnPublishStoryboard(ctx, req.Msg)
	if err != nil {
		zap.L().Error("GetUnPublishStoryboard failed", zap.String("traceId", traceId), zap.Error(err))
		return nil, err
	}
	retData, _ := json.Marshal(ret)
	log.Printf("GetUnPublishStoryboard: %v", string(retData))
	return connect.NewResponse(ret), nil
}
