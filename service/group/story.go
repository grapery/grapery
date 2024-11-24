package group

import (
	"context"
	"fmt"

	"connectrpc.com/connect"

	"github.com/grapery/common-protoc/gen"
	groupService "github.com/grapery/grapery/pkg/group"
	storyServer "github.com/grapery/grapery/pkg/story"
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
	resp := &gen.GetStoryInfoResponse{
		Code:    0,
		Message: "OK",
		Data: &gen.GetStoryInfoResponse_Data{
			Info: info.Data.Info,
		},
	}
	return connect.NewResponse(resp), nil
}

func (s *StoryService) CreateStoryboard(ctx context.Context, req *connect.Request[gen.CreateStoryboardRequest]) (*connect.Response[gen.CreateStoryboardResponse], error) {
	fmt.Println("CreateStoryboard req: ", req.Msg.String())
	ret, err := storyServer.GetStoryServer().CreateStoryboard(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	resp := &gen.CreateStoryboardResponse{
		Code:    ret.Code,
		Message: "OK",
		Data: &gen.CreateStoryboardResponse_Data{
			BoardId: int64(ret.Data.BoardId),
		},
	}
	return connect.NewResponse(resp), nil
}

func (s *StoryService) GetStoryboard(ctx context.Context, req *connect.Request[gen.GetStoryboardRequest]) (*connect.Response[gen.GetStoryboardResponse], error) {
	ret, err := storyServer.GetStoryServer().GetStoryboard(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	resp := &gen.GetStoryboardResponse{
		Code:    ret.Code,
		Message: "OK",
		Data: &gen.GetStoryboardResponse_Data{
			Info: ret.Data.Info,
		},
	}
	return connect.NewResponse(resp), nil
}

func (s *StoryService) GetStoryboards(ctx context.Context, req *connect.Request[gen.GetStoryboardsRequest]) (*connect.Response[gen.GetStoryboardsResponse], error) {
	ret, err := storyServer.GetStoryServer().GetStoryboards(ctx, req.Msg)
	if err != nil {
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

func (s *StoryService) DelStoryboard(ctx context.Context, req *connect.Request[gen.DelStoryboardRequest]) (*connect.Response[gen.DelStoryboardResponse], error) {
	ret, err := storyServer.GetStoryServer().DelStoryboard(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	resp := &gen.DelStoryboardResponse{
		Code:    ret.Code,
		Message: "OK",
	}
	return connect.NewResponse(resp), nil
}

func (s *StoryService) ForkStoryboard(ctx context.Context, req *connect.Request[gen.ForkStoryboardRequest]) (*connect.Response[gen.ForkStoryboardResponse], error) {
	ret, err := storyServer.GetStoryServer().ForkStoryboard(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	resp := &gen.ForkStoryboardResponse{
		Code:    ret.Code,
		Message: "OK",
		Data:    ret.Data,
	}
	return connect.NewResponse(resp), nil
}

func (s *StoryService) LikeStoryboard(ctx context.Context, req *connect.Request[gen.LikeStoryboardRequest]) (*connect.Response[gen.LikeStoryboardResponse], error) {
	ret, err := storyServer.GetStoryServer().LikeStoryboard(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	resp := &gen.LikeStoryboardResponse{
		Code:    ret.Code,
		Message: "OK",
	}
	return connect.NewResponse(resp), nil
}

func (s *StoryService) ShareStoryboard(ctx context.Context, req *connect.Request[gen.ShareStoryboardRequest]) (*connect.Response[gen.ShareStoryboardResponse], error) {
	ret, err := storyServer.GetStoryServer().ShareStoryboard(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	resp := &gen.ShareStoryboardResponse{
		Code:    ret.Code,
		Message: "OK",
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

func (s *StoryService) UpdateStoryboard(ctx context.Context, req *connect.Request[gen.UpdateStoryboardRequest]) (*connect.Response[gen.UpdateStoryboardResponse], error) {
	ret, err := storyServer.GetStoryServer().UpdateStoryboard(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	resp := &gen.UpdateStoryboardResponse{
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

func (s *StoryService) RenderStoryboard(ctx context.Context, req *connect.Request[gen.RenderStoryboardRequest]) (*connect.Response[gen.RenderStoryboardResponse], error) {
	ret, err := storyServer.GetStoryServer().RenderStoryboard(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	resp := &gen.RenderStoryboardResponse{
		Code:    ret.Code,
		Message: "OK",
		Data:    ret.Data,
	}
	return connect.NewResponse(resp), nil
}

func (s *StoryService) GenStoryboardImages(ctx context.Context, req *connect.Request[gen.GenStoryboardImagesRequest]) (*connect.Response[gen.GenStoryboardImagesResponse], error) {
	ret, err := storyServer.GetStoryServer().GenStoryboardImages(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	resp := &gen.GenStoryboardImagesResponse{
		Code:    ret.Code,
		Message: "OK",
	}
	return connect.NewResponse(resp), nil
}

func (s *StoryService) GenStoryboardText(ctx context.Context, req *connect.Request[gen.GenStoryboardTextRequest]) (*connect.Response[gen.GenStoryboardTextResponse], error) {
	ret, err := storyServer.GetStoryServer().GenStoryboardText(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	resp := &gen.GenStoryboardTextResponse{
		Code:    ret.Code,
		Message: "OK",
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

func (s *StoryService) GetStoryBoardRender(ctx context.Context, req *connect.Request[gen.GetStoryBoardRenderRequest]) (*connect.Response[gen.GetStoryBoardRenderResponse], error) {
	ret, err := storyServer.GetStoryServer().GetStoryBoardRender(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	resp := &gen.GetStoryBoardRenderResponse{
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

func (s *StoryService) RenderStoryRoles(ctx context.Context, req *connect.Request[gen.RenderStoryRolesRequest]) (*connect.Response[gen.RenderStoryRolesResponse], error) {
	ret, err := storyServer.GetStoryServer().RenderStoryRoles(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	resp := &gen.RenderStoryRolesResponse{
		Code:    ret.Code,
		Message: "OK",
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

func (s *StoryService) GetStoryBoardRoles(ctx context.Context, req *connect.Request[gen.GetStoryBoardRolesRequest]) (*connect.Response[gen.GetStoryBoardRolesResponse], error) {
	ret, err := storyServer.GetStoryServer().GetStoryBoardRoles(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	resp := &gen.GetStoryBoardRolesResponse{
		Code:    ret.Code,
		Message: "OK",
		Data:    ret.Data,
	}
	return connect.NewResponse(resp), nil
}

func (s *StoryService) UpdateStoryRole(ctx context.Context, req *connect.Request[gen.UpdateStoryRoleRequest]) (*connect.Response[gen.UpdateStoryRoleResponse], error) {
	ret, err := storyServer.GetStoryServer().UpdateStoryRole(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	resp := &gen.UpdateStoryRoleResponse{
		Code:    ret.Code,
		Message: "OK",
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

func (ts *StoryService) CreateStoryRole(ctx context.Context, req *connect.Request[gen.CreateStoryRoleRequest]) (*connect.Response[gen.CreateStoryRoleResponse], error) {
	ret, err := storyServer.GetStoryServer().CreateStoryRole(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	resp := &gen.CreateStoryRoleResponse{
		Code:    ret.Code,
		Message: "OK",
	}
	return connect.NewResponse(resp), nil
}

func (ts *StoryService) GetStoryRoleDetail(ctx context.Context, req *connect.Request[gen.GetStoryRoleDetailRequest]) (*connect.Response[gen.GetStoryRoleDetailResponse], error) {
	ret, err := storyServer.GetStoryServer().GetStoryRoleDetail(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	resp := &gen.GetStoryRoleDetailResponse{
		Code:    ret.Code,
		Message: "OK",
	}
	return connect.NewResponse(resp), nil
}

func (ts *StoryService) RenderStoryRole(ctx context.Context, req *connect.Request[gen.RenderStoryRoleRequest]) (*connect.Response[gen.RenderStoryRoleResponse], error) {
	ret, err := storyServer.GetStoryServer().RenderStoryRole(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	resp := &gen.RenderStoryRoleResponse{
		Code:    ret.Code,
		Message: "OK",
	}
	return connect.NewResponse(resp), nil
}

func (ts *StoryService) LikeStory(ctx context.Context, req *connect.Request[gen.LikeStoryRequest]) (*connect.Response[gen.LikeStoryResponse], error) {
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

func (ts *StoryService) UnLikeStory(ctx context.Context, req *connect.Request[gen.UnLikeStoryRequest]) (*connect.Response[gen.UnLikeStoryResponse], error) {
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

func (ts *StoryService) UnLikeStoryboard(ctx context.Context, req *connect.Request[gen.UnLikeStoryboardRequest]) (*connect.Response[gen.UnLikeStoryboardResponse], error) {
	ret, err := storyServer.GetStoryServer().UnLikeStoryboard(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	resp := &gen.UnLikeStoryboardResponse{
		Code:    ret.Code,
		Message: "OK",
	}
	return connect.NewResponse(resp), nil
}

func (ts *StoryService) GetStoryBoardSences(ctx context.Context, req *connect.Request[gen.GetStoryBoardSencesRequest]) (*connect.Response[gen.GetStoryBoardSencesResponse], error) {
	ret, err := storyServer.GetStoryServer().GetStoryboardScene(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	resp := &gen.GetStoryBoardSencesResponse{
		Code:    ret.Code,
		Message: "OK",
		Data:    ret.Data,
	}
	return connect.NewResponse(resp), nil
}

func (ts *StoryService) CreateStoryBoardSence(ctx context.Context, req *connect.Request[gen.CreateStoryBoardSenceRequest]) (*connect.Response[gen.CreateStoryBoardSenceResponse], error) {
	ret, err := storyServer.GetStoryServer().CreateStoryBoardScene(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	resp := &gen.CreateStoryBoardSenceResponse{
		Code:    ret.Code,
		Message: "OK",
		Data:    ret.Data,
	}
	return connect.NewResponse(resp), nil
}

func (ts *StoryService) UpdateStoryBoardSence(ctx context.Context, req *connect.Request[gen.UpdateStoryBoardSenceRequest]) (*connect.Response[gen.UpdateStoryBoardSenceResponse], error) {
	ret, err := storyServer.GetStoryServer().UpdateStoryBoardSence(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	resp := &gen.UpdateStoryBoardSenceResponse{
		Code:    ret.Code,
		Message: "OK",
	}
	return connect.NewResponse(resp), nil
}

func (ts *StoryService) DeleteStoryBoardSence(ctx context.Context, req *connect.Request[gen.DeleteStoryBoardSenceRequest]) (*connect.Response[gen.DeleteStoryBoardSenceResponse], error) {
	ret, err := storyServer.GetStoryServer().DeleteStoryBoardSence(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	resp := &gen.DeleteStoryBoardSenceResponse{
		Code:    ret.Code,
		Message: "OK",
	}
	return connect.NewResponse(resp), nil
}

func (ts *StoryService) RenderStoryBoardSence(ctx context.Context, req *connect.Request[gen.RenderStoryBoardSenceRequest]) (*connect.Response[gen.RenderStoryBoardSenceResponse], error) {
	ret, err := storyServer.GetStoryServer().RenderStoryBoardSence(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	resp := &gen.RenderStoryBoardSenceResponse{
		Code:    ret.Code,
		Message: "OK",
		Data:    ret.Data,
	}
	return connect.NewResponse(resp), nil
}

func (ts *StoryService) GetStoryBoardSenceGenerate(ctx context.Context, req *connect.Request[gen.GetStoryBoardSenceGenerateRequest]) (*connect.Response[gen.GetStoryBoardSenceGenerateResponse], error) {
	ret, err := storyServer.GetStoryServer().GetStoryBoardSenceGenerate(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	resp := &gen.GetStoryBoardSenceGenerateResponse{
		Code:    ret.Code,
		Message: "OK",
		Data:    ret.Data,
	}
	return connect.NewResponse(resp), nil
}

func (ts *StoryService) GetStoryBoardGenerate(ctx context.Context, req *connect.Request[gen.GetStoryBoardGenerateRequest]) (*connect.Response[gen.GetStoryBoardGenerateResponse], error) {
	ret, err := storyServer.GetStoryServer().GetStoryBoardGenerate(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	resp := &gen.GetStoryBoardGenerateResponse{
		Code:    ret.Code,
		Message: "OK",
		List:    ret.List,
	}
	return connect.NewResponse(resp), nil
}

func (ts *StoryService) RenderStoryBoardSences(ctx context.Context, req *connect.Request[gen.RenderStoryBoardSencesRequest]) (*connect.Response[gen.RenderStoryBoardSencesResponse], error) {
	ret, err := storyServer.GetStoryServer().RenderStoryBoardSences(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	resp := &gen.RenderStoryBoardSencesResponse{
		Code:    ret.Code,
		Message: "OK",
		List:    ret.List,
	}
	return connect.NewResponse(resp), nil
}

func (ts *StoryService) SearchRoles(ctx context.Context, req *connect.Request[gen.SearchRolesRequest]) (*connect.Response[gen.SearchRolesResponse], error) {
	ret, err := storyServer.GetStoryServer().SearchRoles(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	resp := &gen.SearchRolesResponse{
		Code:    ret.Code,
		Message: "OK",
		Roles:   ret.Roles,
		Total:   ret.Total,
	}
	return connect.NewResponse(resp), nil
}

func (ts *StoryService) FollowStoryRole(ctx context.Context, req *connect.Request[gen.FollowStoryRoleRequest]) (*connect.Response[gen.FollowStoryRoleResponse], error) {
	ret, err := storyServer.GetStoryServer().FollowStoryRole(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	resp := &gen.FollowStoryRoleResponse{
		Code:    ret.Code,
		Message: "OK",
	}
	return connect.NewResponse(resp), nil
}

func (ts *StoryService) LikeStoryRole(ctx context.Context, req *connect.Request[gen.LikeStoryRoleRequest]) (*connect.Response[gen.LikeStoryRoleResponse], error) {
	ret, err := storyServer.GetStoryServer().LikeStoryRole(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	resp := &gen.LikeStoryRoleResponse{
		Code:    ret.Code,
		Message: "OK",
	}
	return connect.NewResponse(resp), nil
}

func (ts *StoryService) RestoreStoryboard(ctx context.Context, req *connect.Request[gen.RestoreStoryboardRequest]) (*connect.Response[gen.RestoreStoryboardResponse], error) {
	ret, err := storyServer.GetStoryServer().RestoreStoryboard(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	resp := &gen.RestoreStoryboardResponse{
		Code:    ret.Code,
		Message: "OK",
	}
	return connect.NewResponse(resp), nil
}

func (ts *StoryService) SearchStories(ctx context.Context, req *connect.Request[gen.SearchStoriesRequest]) (*connect.Response[gen.SearchStoriesResponse], error) {
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

func (ts *StoryService) UnFollowStoryRole(ctx context.Context, req *connect.Request[gen.UnFollowStoryRoleRequest]) (*connect.Response[gen.UnFollowStoryRoleResponse], error) {
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

func (ts *StoryService) UnLikeStoryRole(ctx context.Context, req *connect.Request[gen.UnLikeStoryRoleRequest]) (*connect.Response[gen.UnLikeStoryRoleResponse], error) {
	ret, err := storyServer.GetStoryServer().UnLikeStoryRole(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	resp := &gen.UnLikeStoryRoleResponse{
		Code:    ret.Code,
		Message: "OK",
	}
	return connect.NewResponse(resp), nil
}

// 获取用户创建的故事板
func (ts *StoryService) GetUserCreatedStoryboards(ctx context.Context, req *connect.Request[gen.GetUserCreatedStoryboardsRequest]) (*connect.Response[gen.GetUserCreatedStoryboardsResponse], error) {
	ret, err := storyServer.GetStoryServer().GetUserCreatedStoryboards(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	resp := &gen.GetUserCreatedStoryboardsResponse{
		Code:    ret.Code,
		Message: "OK",
	}
	return connect.NewResponse(resp), nil
}

// 获取用户创建的角色
func (ts *StoryService) GetUserCreatedRoles(ctx context.Context, req *connect.Request[gen.GetUserCreatedRolesRequest]) (*connect.Response[gen.GetUserCreatedRolesResponse], error) {
	ret, err := storyServer.GetStoryServer().GetUserCreatedRoles(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	resp := &gen.GetUserCreatedRolesResponse{
		Code:    ret.Code,
		Message: "OK",
	}
	return connect.NewResponse(resp), nil
}
