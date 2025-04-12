package group

import (
	"context"
	"fmt"

	"connectrpc.com/connect"

	"github.com/grapery/common-protoc/gen"
	groupService "github.com/grapery/grapery/pkg/group"
	storyServer "github.com/grapery/grapery/pkg/story"
	"github.com/grapery/grapery/utils"
)

type StoryService struct {
}

func (s *StoryService) CreateStory(ctx context.Context, req *connect.Request[gen.CreateStoryRequest]) (*connect.Response[gen.CreateStoryResponse], error) {
	groupInfo, err := groupService.GetGroupServer().GetGroup(ctx, &gen.GetGroupRequest{
		GroupId: req.Msg.GroupId,
		UserId:  req.Msg.OwnerId,
	})
	if err != nil {
		return nil, err
	}
	// if groupInfo.Data.Info.Status != int32(gen.GroupStatus_Normal) {
	// 	return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("group is not normal"))
	// }
	_ = groupInfo

	ret, err := storyServer.GetStoryServer().CreateStory(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	if ret.Code != 0 {
		return nil, connect.NewError(connect.CodeUnknown, fmt.Errorf(ret.Message))
	}
	resp := &gen.CreateStoryResponse{
		Code:    ret.Code,
		Message: "OK",
		Data: &gen.CreateStoryResponse_Data{
			StoryId: int32(ret.Data.StoryId),
		},
	}
	return connect.NewResponse(resp), nil
}

func (s *StoryService) UpdateStory(ctx context.Context, req *connect.Request[gen.UpdateStoryRequest]) (*connect.Response[gen.UpdateStoryResponse], error) {
	ret, err := storyServer.GetStoryServer().UpdateStory(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	resp := &gen.UpdateStoryResponse{
		Code:    ret.Code,
		Message: "OK",
	}
	return connect.NewResponse(resp), nil
}

func (s *StoryService) GetStoryInfo(ctx context.Context, req *connect.Request[gen.GetStoryInfoRequest]) (*connect.Response[gen.GetStoryInfoResponse], error) {
	info, err := storyServer.GetStoryServer().GetStory(ctx, req.Msg)
	if err != nil {
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
	return connect.NewResponse(resp), nil
}

func (s *StoryService) WatchStory(ctx context.Context, req *connect.Request[gen.WatchStoryRequest]) (*connect.Response[gen.WatchStoryResponse], error) {
	ret, err := storyServer.GetStoryServer().WatchStory(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	resp := &gen.WatchStoryResponse{
		Code:    ret.Code,
		Message: "OK",
	}
	return connect.NewResponse(resp), nil
}

func (s *StoryService) RenderStory(ctx context.Context, req *connect.Request[gen.RenderStoryRequest]) (*connect.Response[gen.RenderStoryResponse], error) {
	ret, err := storyServer.GetStoryServer().RenderStory(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	resp := &gen.RenderStoryResponse{
		Code:    ret.Code,
		Message: "OK",
		Data:    ret.Data,
	}
	return connect.NewResponse(resp), nil
}

func (s *StoryService) GetStoryRender(ctx context.Context, req *connect.Request[gen.GetStoryRenderRequest]) (*connect.Response[gen.GetStoryRenderResponse], error) {
	ret, err := storyServer.GetStoryServer().GetStoryRender(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	resp := &gen.GetStoryRenderResponse{
		Code:    ret.Code,
		Message: "OK",
		Data:    ret.Data,
	}
	return connect.NewResponse(resp), nil
}

func (s *StoryService) ContinueRenderStory(ctx context.Context, req *connect.Request[gen.ContinueRenderStoryRequest]) (*connect.Response[gen.ContinueRenderStoryResponse], error) {
	ret, err := storyServer.GetStoryServer().ContinueRenderStory(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	resp := &gen.ContinueRenderStoryResponse{
		Code:    ret.Code,
		Message: "OK",
		Data:    ret.Data,
	}
	return connect.NewResponse(resp), nil
}

func (s *StoryService) RenderStoryRoleDetail(ctx context.Context, req *connect.Request[gen.RenderStoryRoleDetailRequest]) (*connect.Response[gen.RenderStoryRoleDetailResponse], error) {
	ret, err := storyServer.GetStoryServer().RenderStoryRoleDetail(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	resp := &gen.RenderStoryRoleDetailResponse{
		Code:    ret.Code,
		Message: "OK",
		Role:    ret.Role,
	}
	return connect.NewResponse(resp), nil
}

func (s *StoryService) GetStoryRoles(ctx context.Context, req *connect.Request[gen.GetStoryRolesRequest]) (*connect.Response[gen.GetStoryRolesResponse], error) {
	ret, err := storyServer.GetStoryServer().GetStoryRoles(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	resp := &gen.GetStoryRolesResponse{
		Code:    ret.Code,
		Message: "OK",
		Data:    ret.Data,
	}
	return connect.NewResponse(resp), nil
}

func (s *StoryService) GetStoryContributors(ctx context.Context, req *connect.Request[gen.GetStoryContributorsRequest]) (*connect.Response[gen.GetStoryContributorsResponse], error) {
	ret, err := storyServer.GetStoryServer().GetStoryContributors(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	resp := &gen.GetStoryContributorsResponse{
		Code:    ret.Code,
		Message: "OK",
		Data:    ret.Data,
	}
	return connect.NewResponse(resp), nil
}

func (s *StoryService) LikeStory(ctx context.Context, req *connect.Request[gen.LikeStoryRequest]) (*connect.Response[gen.LikeStoryResponse], error) {
	ret, err := storyServer.GetStoryServer().LikeStory(ctx, &gen.LikeStoryRequest{
		StoryId: req.Msg.StoryId,
		UserId:  req.Msg.UserId,
	})
	if err != nil {
		return nil, err
	}
	resp := &gen.LikeStoryResponse{
		Code:    ret.Code,
		Message: "OK",
	}
	return connect.NewResponse(resp), nil
}

func (s *StoryService) UnLikeStory(ctx context.Context, req *connect.Request[gen.UnLikeStoryRequest]) (*connect.Response[gen.UnLikeStoryResponse], error) {
	ret, err := storyServer.GetStoryServer().UnLikeStory(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	resp := &gen.UnLikeStoryResponse{
		Code:    ret.Code,
		Message: "OK",
	}
	return connect.NewResponse(resp), nil
}

func (s *StoryService) SearchStories(ctx context.Context, req *connect.Request[gen.SearchStoriesRequest]) (*connect.Response[gen.SearchStoriesResponse], error) {
	ret, err := storyServer.GetStoryServer().SearchStories(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	resp := &gen.SearchStoriesResponse{
		Code:    ret.Code,
		Message: "OK",
		Stories: ret.Stories,
	}
	return connect.NewResponse(resp), nil
}

func (s *StoryService) UnFollowStoryRole(ctx context.Context, req *connect.Request[gen.UnFollowStoryRoleRequest]) (*connect.Response[gen.UnFollowStoryRoleResponse], error) {
	ret, err := storyServer.GetStoryServer().UnFollowStoryRole(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	resp := &gen.UnFollowStoryRoleResponse{
		Code:    ret.Code,
		Message: "OK",
	}
	return connect.NewResponse(resp), nil
}

func (s *StoryService) GetNextStoryboard(ctx context.Context, req *connect.Request[gen.GetNextStoryboardRequest]) (*connect.Response[gen.GetNextStoryboardResponse], error) {
	ret, err := storyServer.GetStoryServer().GetNextStoryboard(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(ret), nil
}

func (s *StoryService) ArchiveStory(ctx context.Context, req *connect.Request[gen.ArchiveStoryRequest]) (*connect.Response[gen.ArchiveStoryResponse], error) {
	ret := &gen.ArchiveStoryResponse{
		Code:    0,
		Message: "OK",
	}
	return connect.NewResponse(ret), nil
}
