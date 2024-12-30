package group

import (
	"context"
	"fmt"

	"connectrpc.com/connect"

	"github.com/grapery/common-protoc/gen"
	api "github.com/grapery/common-protoc/gen"
	storyServer "github.com/grapery/grapery/pkg/story"
)

type StoryRoleService struct {
}

func (s *StoryRoleService) RenderStoryRoleContinuouslyCancel(ctx context.Context, req *api.RenderStoryRoleContinuouslyRequest) (*api.RenderStoryRoleContinuouslyResponse, error) {
	return nil, fmt.Errorf("not implemented")
}

func (s *StoryRoleService) RenderStoryRoles(ctx context.Context, req *connect.Request[gen.RenderStoryRolesRequest]) (*connect.Response[gen.RenderStoryRolesResponse], error) {
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

func (s *StoryRoleService) UpdateStoryRole(ctx context.Context, req *connect.Request[gen.UpdateStoryRoleRequest]) (*connect.Response[gen.UpdateStoryRoleResponse], error) {
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

func (s *StoryRoleService) CreateStoryRole(ctx context.Context, req *connect.Request[gen.CreateStoryRoleRequest]) (*connect.Response[gen.CreateStoryRoleResponse], error) {
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

func (s *StoryRoleService) GetStoryRoleDetail(ctx context.Context, req *connect.Request[gen.GetStoryRoleDetailRequest]) (*connect.Response[gen.GetStoryRoleDetailResponse], error) {
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

func (s *StoryRoleService) RenderStoryRole(ctx context.Context, req *connect.Request[gen.RenderStoryRoleRequest]) (*connect.Response[gen.RenderStoryRoleResponse], error) {
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

func (s *StoryRoleService) SearchRoles(ctx context.Context, req *connect.Request[gen.SearchRolesRequest]) (*connect.Response[gen.SearchRolesResponse], error) {
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

func (s *StoryRoleService) FollowStoryRole(ctx context.Context, req *connect.Request[gen.FollowStoryRoleRequest]) (*connect.Response[gen.FollowStoryRoleResponse], error) {
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

func (s *StoryRoleService) LikeStoryRole(ctx context.Context, req *connect.Request[gen.LikeStoryRoleRequest]) (*connect.Response[gen.LikeStoryRoleResponse], error) {
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

func (s *StoryRoleService) UnLikeStoryRole(ctx context.Context, req *connect.Request[gen.UnLikeStoryRoleRequest]) (*connect.Response[gen.UnLikeStoryRoleResponse], error) {
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

// 获取用户创建的角色
func (s *StoryRoleService) GetUserCreatedRoles(ctx context.Context, req *connect.Request[gen.GetUserCreatedRolesRequest]) (*connect.Response[gen.GetUserCreatedRolesResponse], error) {
	ret, err := storyServer.GetStoryServer().GetUserCreatedRoles(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	resp := &gen.GetUserCreatedRolesResponse{
		Code:     ret.Code,
		Message:  "OK",
		Roles:    ret.Roles,
		Total:    ret.Total,
		Offset:   ret.Offset,
		PageSize: ret.PageSize,
	}
	return connect.NewResponse(resp), nil
}

func (s *StoryRoleService) GetStoryRoleStories(ctx context.Context, req *connect.Request[gen.GetStoryRoleStoriesRequest]) (*connect.Response[gen.GetStoryRoleStoriesResponse], error) {
	ret, err := storyServer.GetStoryServer().GetStoryRoleStories(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	resp := &gen.GetStoryRoleStoriesResponse{
		Code:     ret.Code,
		Message:  "OK",
		Stories:  ret.Stories,
		Total:    ret.Total,
		Offset:   ret.Offset,
		PageSize: ret.PageSize,
	}
	return connect.NewResponse(resp), nil
}

func (s *StoryRoleService) GetStoryRoleStoryboards(ctx context.Context, req *connect.Request[gen.GetStoryRoleStoryboardsRequest]) (*connect.Response[gen.GetStoryRoleStoryboardsResponse], error) {
	ret, err := storyServer.GetStoryServer().GetStoryRoleStoryboards(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	resp := &gen.GetStoryRoleStoryboardsResponse{
		Code:        ret.Code,
		Message:     "OK",
		Storyboards: ret.Storyboards,
		Total:       ret.Total,
		Offset:      ret.Offset,
		PageSize:    ret.PageSize,
	}
	return connect.NewResponse(resp), nil
}

func (s *StoryRoleService) CreateStoryRoleChat(ctx context.Context, req *connect.Request[gen.CreateStoryRoleChatRequest]) (*connect.Response[gen.CreateStoryRoleChatResponse], error) {
	ret, err := storyServer.GetStoryServer().CreateStoryRoleChat(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	resp := &gen.CreateStoryRoleChatResponse{
		Code:        ret.Code,
		Message:     "OK",
		ChatContext: ret.ChatContext,
	}
	return connect.NewResponse(resp), nil
}

func (s *StoryRoleService) ChatWithStoryRole(ctx context.Context, req *connect.Request[gen.ChatWithStoryRoleRequest]) (*connect.Response[gen.ChatWithStoryRoleResponse], error) {
	ret, err := storyServer.GetStoryServer().ChatWithStoryRole(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	resp := &gen.ChatWithStoryRoleResponse{
		Code:          ret.Code,
		Message:       "OK",
		ReplyMessages: ret.ReplyMessages,
	}
	return connect.NewResponse(resp), nil
}

func (s *StoryRoleService) UpdateStoryRoleDetail(ctx context.Context, req *connect.Request[gen.UpdateStoryRoleDetailRequest]) (*connect.Response[gen.UpdateStoryRoleDetailResponse], error) {
	ret, err := storyServer.GetStoryServer().UpdateStoryRoleDetail(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	resp := &gen.UpdateStoryRoleDetailResponse{
		Code:    ret.Code,
		Message: "OK",
	}
	return connect.NewResponse(resp), nil
}

func (s *StoryRoleService) GetUserWithRoleChatList(ctx context.Context, req *connect.Request[gen.GetUserWithRoleChatListRequest]) (*connect.Response[gen.GetUserWithRoleChatListResponse], error) {
	ret, err := storyServer.GetStoryServer().GetUserWithRoleChatList(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	resp := &gen.GetUserWithRoleChatListResponse{
		Code:    ret.Code,
		Message: "OK",
		Chats:   ret.Chats,
	}
	return connect.NewResponse(resp), nil
}

func (s *StoryRoleService) GetUserChatWithRole(ctx context.Context, req *connect.Request[gen.GetUserChatWithRoleRequest]) (*connect.Response[gen.GetUserChatWithRoleResponse], error) {
	ret, err := storyServer.GetStoryServer().GetUserChatWithRole(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	resp := &gen.GetUserChatWithRoleResponse{
		Code:        ret.Code,
		Message:     "OK",
		ChatContext: ret.ChatContext,
		Messages:    ret.Messages,
	}
	return connect.NewResponse(resp), nil
}

func (s *StoryRoleService) GetUserChatMessages(ctx context.Context, req *connect.Request[gen.GetUserChatMessagesRequest]) (*connect.Response[gen.GetUserChatMessagesResponse], error) {
	ret, err := storyServer.GetStoryServer().GetUserChatMessages(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(ret), nil
}

func (s *StoryRoleService) RenderStoryRoleContinuously(ctx context.Context, req *connect.Request[gen.RenderStoryRoleContinuouslyRequest]) (*connect.Response[gen.RenderStoryRoleContinuouslyResponse], error) {
	ret, err := storyServer.GetStoryServer().RenderStoryRoleContinuously(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(ret), nil
}