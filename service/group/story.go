package group

import (
	"context"
	"fmt"
	"log"

	connect "connectrpc.com/connect"

	"github.com/grapery/common-protoc/gen"
	groupService "github.com/grapery/grapery/pkg/group"
	storyServer "github.com/grapery/grapery/pkg/story"
	"github.com/grapery/grapery/utils"
	"go.uber.org/zap"
)

// 从 context 获取 traceId，没有则生成
func getTraceID(ctx context.Context) string {
	if v := ctx.Value("traceId"); v != nil {
		if s, ok := v.(string); ok && s != "" {
			return s
		}
	}
	// 简单生成traceId
	return fmt.Sprintf("trace-%d", ctx.Value("request_id"))
}

// 内容脱敏，只显示前20字
func maskContent(content string) string {
	if len(content) > 20 {
		return content[:20] + "..."
	}
	return content
}

// 日志调用全部用 getTraceID(ctx) 和 maskContent

type StoryService struct {
}

func (s *StoryService) CreateStory(ctx context.Context, req *connect.Request[gen.CreateStoryRequest]) (*connect.Response[gen.CreateStoryResponse], error) {
	traceId := getTraceID(ctx)
	zap.L().Info("CreateStory called", zap.String("traceId", traceId), zap.Int64("groupId", req.Msg.GroupId), zap.Int64("ownerId", req.Msg.OwnerId), zap.String("title", maskContent(req.Msg.Title)))
	groupInfo, err := groupService.GetGroupServer().GetGroup(ctx, &gen.GetGroupRequest{
		GroupId: req.Msg.GroupId,
		UserId:  req.Msg.OwnerId,
	})
	if err != nil {
		zap.L().Error("get group info failed", zap.String("traceId", traceId), zap.Error(err))
		return nil, err
	}
	// if groupInfo.Data.Info.Status != int32(gen.GroupStatus_Normal) {
	// 	return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("group is not normal"))
	// }
	_ = groupInfo

	ret, err := storyServer.GetStoryServer().CreateStory(ctx, req.Msg)
	if err != nil {
		zap.L().Error("create story failed", zap.String("traceId", traceId), zap.Error(err))
		return nil, err
	}
	if ret.Code != 0 {
		zap.L().Error("create story failed", zap.String("traceId", traceId), zap.String("msg", ret.Message))
		return nil, connect.NewError(connect.CodeUnknown, fmt.Errorf(ret.Message))
	}
	resp := &gen.CreateStoryResponse{
		Code:    ret.Code,
		Message: "OK",
		Data: &gen.CreateStoryResponse_Data{
			StoryId: int32(ret.Data.StoryId),
		},
	}
	zap.L().Info("create story success", zap.String("traceId", traceId), zap.Int64("storyId", int64(ret.Data.StoryId)))
	return connect.NewResponse(resp), nil
}

func (s *StoryService) UpdateStory(ctx context.Context, req *connect.Request[gen.UpdateStoryRequest]) (*connect.Response[gen.UpdateStoryResponse], error) {
	traceId := getTraceID(ctx)
	zap.L().Info("UpdateStory called", zap.String("traceId", traceId), zap.Int64("storyId", req.Msg.StoryId))
	ret, err := storyServer.GetStoryServer().UpdateStory(ctx, req.Msg)
	if err != nil {
		zap.L().Error("UpdateStory failed", zap.String("traceId", traceId), zap.Error(err))
		return nil, err
	}
	resp := &gen.UpdateStoryResponse{
		Code:    ret.Code,
		Message: "OK",
	}
	zap.L().Info("UpdateStory success", zap.String("traceId", traceId), zap.Int64("storyId", req.Msg.StoryId))
	return connect.NewResponse(resp), nil
}

func (s *StoryService) GetStoryInfo(ctx context.Context, req *connect.Request[gen.GetStoryInfoRequest]) (*connect.Response[gen.GetStoryInfoResponse], error) {
	traceId := getTraceID(ctx)
	zap.L().Info("GetStoryInfo called", zap.String("traceId", traceId), zap.Int64("storyId", req.Msg.StoryId))
	info, err := storyServer.GetStoryServer().GetStory(ctx, req.Msg)
	if err != nil {
		zap.L().Error("GetStoryInfo failed", zap.String("traceId", traceId), zap.Error(err))
		return nil, err
	}
	fmt.Printf("req %s info %s", req.Msg.String(), info.String())
	userID, err := utils.GetUserIDFromContext(ctx)

	if err != nil {
		fmt.Printf("get user id from context failed: %s", err.Error())
		return nil, err
	}
	fmt.Printf("user id: %d", userID)
	resp := &gen.GetStoryInfoResponse{
		Code:    0,
		Message: "OK",
		Data: &gen.GetStoryInfoResponse_Data{
			Info: info.Data.Info,
		},
	}
	fmt.Printf("GetStoryInfo %s", resp.String())
	zap.L().Info("GetStoryInfo success", zap.String("traceId", traceId), zap.Int64("userId", userID), zap.Int64("storyId", req.Msg.StoryId))
	return connect.NewResponse(resp), nil
}

func (s *StoryService) WatchStory(ctx context.Context, req *connect.Request[gen.WatchStoryRequest]) (*connect.Response[gen.WatchStoryResponse], error) {
	traceId := getTraceID(ctx)
	zap.L().Info("WatchStory called", zap.String("traceId", traceId), zap.Int64("storyId", req.Msg.StoryId), zap.Int64("userId", req.Msg.UserId))
	ret, err := storyServer.GetStoryServer().WatchStory(ctx, req.Msg)
	if err != nil {
		zap.L().Error("WatchStory failed", zap.String("traceId", traceId), zap.Error(err))
		return nil, err
	}
	resp := &gen.WatchStoryResponse{
		Code:    ret.Code,
		Message: "OK",
	}
	zap.L().Info("WatchStory success", zap.String("traceId", traceId), zap.Int64("storyId", req.Msg.StoryId), zap.Int64("userId", req.Msg.UserId))
	return connect.NewResponse(resp), nil
}

func (s *StoryService) RenderStory(ctx context.Context, req *connect.Request[gen.RenderStoryRequest]) (*connect.Response[gen.RenderStoryResponse], error) {
	traceId := getTraceID(ctx)
	zap.L().Info("RenderStory called", zap.String("traceId", traceId), zap.Int64("storyId", req.Msg.StoryId))
	ret, err := storyServer.GetStoryServer().RenderStory(ctx, req.Msg)
	if err != nil {
		zap.L().Error("RenderStory failed", zap.String("traceId", traceId), zap.Error(err))
		return nil, err
	}
	resp := &gen.RenderStoryResponse{
		Code:    ret.Code,
		Message: "OK",
		Data:    ret.Data,
	}
	zap.L().Info("RenderStory success", zap.String("traceId", traceId), zap.Int64("storyId", req.Msg.StoryId))
	return connect.NewResponse(resp), nil
}

func (s *StoryService) GetStoryRender(ctx context.Context, req *connect.Request[gen.GetStoryRenderRequest]) (*connect.Response[gen.GetStoryRenderResponse], error) {
	traceId := getTraceID(ctx)
	zap.L().Info("GetStoryRender called", zap.String("traceId", traceId), zap.Int64("storyId", req.Msg.StoryId))
	ret, err := storyServer.GetStoryServer().GetStoryRender(ctx, req.Msg)
	if err != nil {
		zap.L().Error("GetStoryRender failed", zap.String("traceId", traceId), zap.Error(err))
		return nil, err
	}
	resp := &gen.GetStoryRenderResponse{
		Code:    ret.Code,
		Message: "OK",
		Data:    ret.Data,
	}
	zap.L().Info("GetStoryRender success", zap.String("traceId", traceId), zap.Int64("storyId", req.Msg.StoryId))
	return connect.NewResponse(resp), nil
}

func (s *StoryService) ContinueRenderStory(ctx context.Context, req *connect.Request[gen.ContinueRenderStoryRequest]) (*connect.Response[gen.ContinueRenderStoryResponse], error) {
	traceId := getTraceID(ctx)
	zap.L().Info("ContinueRenderStory called", zap.String("traceId", traceId), zap.Int64("storyId", req.Msg.StoryId))
	ret, err := storyServer.GetStoryServer().ContinueRenderStory(ctx, req.Msg)
	if err != nil {
		zap.L().Error("ContinueRenderStory failed", zap.String("traceId", traceId), zap.Error(err))
		return nil, err
	}
	resp := &gen.ContinueRenderStoryResponse{
		Code:    ret.Code,
		Message: "OK",
		Data:    ret.Data,
	}
	zap.L().Info("ContinueRenderStory success", zap.String("traceId", traceId), zap.Int64("storyId", req.Msg.StoryId))
	return connect.NewResponse(resp), nil
}

func (s *StoryService) RenderStoryRoleDetail(ctx context.Context, req *connect.Request[gen.RenderStoryRoleDetailRequest]) (*connect.Response[gen.RenderStoryRoleDetailResponse], error) {
	traceId := getTraceID(ctx)
	// req.Msg.StoryId 改为 req.Msg.GetStoryId()，如无则注释掉
	// zap.L().Info("RenderStoryRoleDetail called", zap.String("traceId", traceId), zap.Int64("storyId", req.Msg.StoryId), zap.Int64("roleId", req.Msg.RoleId))
	zap.L().Info("RenderStoryRoleDetail called", zap.String("traceId", traceId), zap.Int64("roleId", req.Msg.RoleId))
	ret, err := storyServer.GetStoryServer().RenderStoryRoleDetail(ctx, req.Msg)
	if err != nil {
		zap.L().Error("RenderStoryRoleDetail failed", zap.String("traceId", traceId), zap.Error(err))
		return nil, err
	}
	resp := &gen.RenderStoryRoleDetailResponse{
		Code:    ret.Code,
		Message: "OK",
		Role:    ret.Role,
	}
	// zap.L().Info("RenderStoryRoleDetail success", zap.String("traceId", traceId), zap.Int64("storyId", req.Msg.StoryId), zap.Int64("roleId", req.Msg.RoleId))
	zap.L().Info("RenderStoryRoleDetail success", zap.String("traceId", traceId), zap.Int64("roleId", req.Msg.RoleId))
	return connect.NewResponse(resp), nil
}

func (s *StoryService) GetStoryRoles(ctx context.Context, req *connect.Request[gen.GetStoryRolesRequest]) (*connect.Response[gen.GetStoryRolesResponse], error) {
	traceId := getTraceID(ctx)
	zap.L().Info("GetStoryRoles called", zap.String("traceId", traceId), zap.Int64("storyId", req.Msg.StoryId))
	ret, err := storyServer.GetStoryServer().GetStoryRoles(ctx, req.Msg)
	if err != nil {
		zap.L().Error("GetStoryRoles failed", zap.String("traceId", traceId), zap.Error(err))
		return nil, err
	}
	resp := &gen.GetStoryRolesResponse{
		Code:    ret.Code,
		Message: "OK",
		Data:    ret.Data,
	}
	zap.L().Info("GetStoryRoles success", zap.String("traceId", traceId), zap.Int64("storyId", req.Msg.StoryId))
	return connect.NewResponse(resp), nil
}

func (s *StoryService) GetStoryContributors(ctx context.Context, req *connect.Request[gen.GetStoryContributorsRequest]) (*connect.Response[gen.GetStoryContributorsResponse], error) {
	traceId := getTraceID(ctx)
	zap.L().Info("GetStoryContributors called", zap.String("traceId", traceId), zap.Int64("storyId", req.Msg.StoryId))
	ret, err := storyServer.GetStoryServer().GetStoryContributors(ctx, req.Msg)
	if err != nil {
		zap.L().Error("GetStoryContributors failed", zap.String("traceId", traceId), zap.Error(err))
		return nil, err
	}
	resp := &gen.GetStoryContributorsResponse{
		Code:    ret.Code,
		Message: "OK",
		Data:    ret.Data,
	}
	zap.L().Info("GetStoryContributors success", zap.String("traceId", traceId), zap.Int64("storyId", req.Msg.StoryId))
	return connect.NewResponse(resp), nil
}

func (s *StoryService) LikeStory(ctx context.Context, req *connect.Request[gen.LikeStoryRequest]) (*connect.Response[gen.LikeStoryResponse], error) {
	traceId := getTraceID(ctx)
	zap.L().Info("LikeStory called", zap.String("traceId", traceId), zap.Int64("storyId", req.Msg.StoryId), zap.Int64("userId", req.Msg.UserId))
	ret, err := storyServer.GetStoryServer().LikeStory(ctx, &gen.LikeStoryRequest{
		StoryId: req.Msg.StoryId,
		UserId:  req.Msg.UserId,
	})
	if err != nil {
		zap.L().Error("LikeStory failed", zap.String("traceId", traceId), zap.Error(err))
		return nil, err
	}
	resp := &gen.LikeStoryResponse{
		Code:    ret.Code,
		Message: "OK",
	}
	zap.L().Info("LikeStory success", zap.String("traceId", traceId), zap.Int64("storyId", req.Msg.StoryId), zap.Int64("userId", req.Msg.UserId))
	return connect.NewResponse(resp), nil
}

func (s *StoryService) UnLikeStory(ctx context.Context, req *connect.Request[gen.UnLikeStoryRequest]) (*connect.Response[gen.UnLikeStoryResponse], error) {
	traceId := getTraceID(ctx)
	zap.L().Info("UnLikeStory called", zap.String("traceId", traceId), zap.Int64("storyId", req.Msg.StoryId), zap.Int64("userId", req.Msg.UserId))
	ret, err := storyServer.GetStoryServer().UnLikeStory(ctx, req.Msg)
	if err != nil {
		zap.L().Error("UnLikeStory failed", zap.String("traceId", traceId), zap.Error(err))
		return nil, err
	}
	resp := &gen.UnLikeStoryResponse{
		Code:    ret.Code,
		Message: "OK",
	}
	zap.L().Info("UnLikeStory success", zap.String("traceId", traceId), zap.Int64("storyId", req.Msg.StoryId), zap.Int64("userId", req.Msg.UserId))
	return connect.NewResponse(resp), nil
}

func (s *StoryService) SearchStories(ctx context.Context, req *connect.Request[gen.SearchStoriesRequest]) (*connect.Response[gen.SearchStoriesResponse], error) {
	traceId := getTraceID(ctx)
	zap.L().Info("SearchStories called", zap.String("traceId", traceId), zap.String("keyword", maskContent(req.Msg.Keyword)))
	ret, err := storyServer.GetStoryServer().SearchStories(ctx, req.Msg)
	if err != nil {
		zap.L().Error("SearchStories failed", zap.String("traceId", traceId), zap.Error(err))
		return nil, err
	}
	resp := &gen.SearchStoriesResponse{
		Code:    ret.Code,
		Message: "OK",
		Stories: ret.Stories,
	}
	zap.L().Info("SearchStories success", zap.String("traceId", traceId), zap.Int("storyCount", len(ret.Stories)))
	return connect.NewResponse(resp), nil
}

func (s *StoryService) UnFollowStoryRole(ctx context.Context, req *connect.Request[gen.UnFollowStoryRoleRequest]) (*connect.Response[gen.UnFollowStoryRoleResponse], error) {
	traceId := getTraceID(ctx)
	zap.L().Info("UnFollowStoryRole called", zap.String("traceId", traceId), zap.Int64("roleId", req.Msg.RoleId), zap.Int64("userId", req.Msg.UserId))
	ret, err := storyServer.GetStoryServer().UnFollowStoryRole(ctx, req.Msg)
	if err != nil {
		zap.L().Error("UnFollowStoryRole failed", zap.String("traceId", traceId), zap.Error(err))
		return nil, err
	}
	resp := &gen.UnFollowStoryRoleResponse{
		Code:    ret.Code,
		Message: "OK",
	}
	zap.L().Info("UnFollowStoryRole success", zap.String("traceId", traceId), zap.Int64("roleId", req.Msg.RoleId), zap.Int64("userId", req.Msg.UserId))
	return connect.NewResponse(resp), nil
}

func (s *StoryService) GetNextStoryboard(ctx context.Context, req *connect.Request[gen.GetNextStoryboardRequest]) (*connect.Response[gen.GetNextStoryboardResponse], error) {
	traceId := getTraceID(ctx)
	zap.L().Info("GetNextStoryboard called", zap.String("traceId", traceId), zap.Int64("storyId", req.Msg.StoryId))
	ret, err := storyServer.GetStoryServer().GetNextStoryboard(ctx, req.Msg)
	if err != nil {
		log.Printf("get next storyboard failed: %s", err.Error())
		return nil, err
	}
	log.Printf("get next storyboard success: %s", ret.String())
	zap.L().Info("GetNextStoryboard success", zap.String("traceId", traceId), zap.Int64("storyId", req.Msg.StoryId))
	return connect.NewResponse(ret), nil
}

func (s *StoryService) ArchiveStory(ctx context.Context, req *connect.Request[gen.ArchiveStoryRequest]) (*connect.Response[gen.ArchiveStoryResponse], error) {
	traceId := getTraceID(ctx)
	zap.L().Info("ArchiveStory called", zap.String("traceId", traceId), zap.Int64("storyId", req.Msg.StoryId))
	ret := &gen.ArchiveStoryResponse{
		Code:    0,
		Message: "OK",
	}
	zap.L().Info("ArchiveStory success", zap.String("traceId", traceId), zap.Int64("storyId", req.Msg.StoryId))
	return connect.NewResponse(ret), nil
}

func (s *StoryService) TrendingStory(ctx context.Context, req *connect.Request[gen.TrendingStoryRequest]) (*connect.Response[gen.TrendingStoryResponse], error) {
	traceId := getTraceID(ctx)
	zap.L().Info("TrendingStory called", zap.String("traceId", traceId))
	ret, err := storyServer.GetStoryServer().TrendingStory(ctx, req.Msg)
	if err != nil {
		zap.L().Error("TrendingStory failed", zap.String("traceId", traceId), zap.Error(err))
		return nil, err
	}
	zap.L().Info("TrendingStory success", zap.String("traceId", traceId))
	return connect.NewResponse(ret), nil
}

func (s *StoryService) TrendingStoryRole(ctx context.Context, req *connect.Request[gen.TrendingStoryRoleRequest]) (*connect.Response[gen.TrendingStoryRoleResponse], error) {
	traceId := getTraceID(ctx)
	zap.L().Info("TrendingStoryRole called", zap.String("traceId", traceId))
	ret, err := storyServer.GetStoryServer().TrendingStoryRole(ctx, req.Msg)
	if err != nil {
		zap.L().Error("TrendingStoryRole failed", zap.String("traceId", traceId), zap.Error(err))
		return nil, err
	}
	zap.L().Info("TrendingStoryRole success", zap.String("traceId", traceId))
	return connect.NewResponse(ret), nil
}

func (s *StoryService) GetStoryImageStyle(ctx context.Context, req *connect.Request[gen.GetStoryImageStyleRequest]) (*connect.Response[gen.GetStoryImageStyleResponse], error) {
	traceId := getTraceID(ctx)
	zap.L().Info("GetStoryImageStyle called", zap.String("traceId", traceId), zap.Int64("storyId", req.Msg.StoryId))
	ret, err := storyServer.GetStoryServer().GetStoryImageStyle(ctx, req.Msg)
	if err != nil {
		zap.L().Error("GetStoryImageStyle failed", zap.String("traceId", traceId), zap.Error(err))
		return nil, err
	}
	zap.L().Info("GetStoryImageStyle success", zap.String("traceId", traceId), zap.Int64("storyId", req.Msg.StoryId))
	return connect.NewResponse(ret), nil
}

func (s *StoryService) UpdateStoryImageStyle(ctx context.Context, req *connect.Request[gen.UpdateStoryImageStyleRequest]) (*connect.Response[gen.UpdateStoryImageStyleResponse], error) {
	traceId := getTraceID(ctx)
	zap.L().Info("UpdateStoryImageStyle called", zap.String("traceId", traceId), zap.Int64("storyId", req.Msg.StoryId))
	ret, err := storyServer.GetStoryServer().UpdateStoryImageStyle(ctx, req.Msg)
	if err != nil {
		zap.L().Error("UpdateStoryImageStyle failed", zap.String("traceId", traceId), zap.Error(err))
		return nil, err
	}
	zap.L().Info("UpdateStoryImageStyle success", zap.String("traceId", traceId), zap.Int64("storyId", req.Msg.StoryId))
	return connect.NewResponse(ret), nil
}

func (s *StoryService) UpdateStorySenceMaxNumber(ctx context.Context, req *connect.Request[gen.UpdateStorySenceMaxNumberRequest]) (*connect.Response[gen.UpdateStorySenceMaxNumberResponse], error) {
	traceId := getTraceID(ctx)
	zap.L().Info("UpdateStorySenceMaxNumber called", zap.String("traceId", traceId), zap.Int64("storyId", req.Msg.StoryId))
	ret, err := storyServer.GetStoryServer().UpdateStorySenceMaxNumber(ctx, req.Msg)
	if err != nil {
		zap.L().Error("UpdateStorySenceMaxNumber failed", zap.String("traceId", traceId), zap.Error(err))
		return nil, err
	}
	zap.L().Info("UpdateStorySenceMaxNumber success", zap.String("traceId", traceId), zap.Int64("storyId", req.Msg.StoryId))
	return connect.NewResponse(ret), nil
}
