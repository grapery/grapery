package group

import (
	"context"
	"errors"
	"fmt"

	"go.uber.org/zap"

	connect "connectrpc.com/connect"

	"github.com/grapery/common-protoc/gen"
	api "github.com/grapery/common-protoc/gen"
	storyEngineServer "github.com/grapery/grapery/pkg/story"
	"github.com/grapery/grapery/utils"
	"github.com/grapery/grapery/utils/log"
)

// 移除本文件顶部的 getTraceID 和 maskContent 的重复定义

type StoryRoleService struct {
}

func (s *StoryRoleService) RenderStoryRoleContinuouslyCancel(ctx context.Context, req *api.RenderStoryRoleContinuouslyRequest) (*api.RenderStoryRoleContinuouslyResponse, error) {
	return nil, fmt.Errorf("not implemented")
}

func (s *StoryRoleService) RenderStoryRoles(ctx context.Context, req *connect.Request[gen.RenderStoryRolesRequest]) (*connect.Response[gen.RenderStoryRolesResponse], error) {
	traceId := getTraceID(ctx)
	zap.L().Info("RenderStoryRoles called", zap.String("traceId", traceId), zap.Int64("storyId", req.Msg.GetStoryId()))
	ret, err := storyEngineServer.GetStoryEngine().RenderStoryRoles(ctx, req.Msg)
	if err != nil {
		zap.L().Error("RenderStoryRoles failed", zap.String("traceId", traceId), zap.Error(err))
		return nil, err
	}
	resp := &gen.RenderStoryRolesResponse{
		Code:    ret.Code,
		Message: "OK",
	}
	zap.L().Info("RenderStoryRoles success", zap.String("traceId", traceId), zap.Int64("storyId", req.Msg.GetStoryId()))
	return connect.NewResponse(resp), nil
}

func (s *StoryRoleService) UpdateStoryRole(ctx context.Context, req *connect.Request[gen.UpdateStoryRoleRequest]) (*connect.Response[gen.UpdateStoryRoleResponse], error) {
	ret, err := storyEngineServer.GetStoryEngine().UpdateStoryRole(ctx, req.Msg)
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
	ret, err := storyEngineServer.GetStoryEngine().CreateStoryRole(ctx, req.Msg)
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
	ret, err := storyEngineServer.GetStoryEngine().GetStoryRoleDetail(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	resp := &gen.GetStoryRoleDetailResponse{
		Code:    ret.Code,
		Message: ret.Message,
		Info:    ret.Info,
	}
	return connect.NewResponse(resp), nil
}

func (s *StoryRoleService) RenderStoryRole(ctx context.Context, req *connect.Request[gen.RenderStoryRoleRequest]) (*connect.Response[gen.RenderStoryRoleResponse], error) {
	ret, err := storyEngineServer.GetStoryEngine().RenderStoryRole(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	resp := &gen.RenderStoryRoleResponse{
		Code:    ret.Code,
		Message: ret.Message,
		Detail:  ret.GetDetail(),
	}
	return connect.NewResponse(resp), nil
}

func (s *StoryRoleService) SearchRoles(ctx context.Context, req *connect.Request[gen.SearchRolesRequest]) (*connect.Response[gen.SearchRolesResponse], error) {
	ret, err := storyEngineServer.GetStoryEngine().SearchRoles(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	resp := &gen.SearchRolesResponse{
		Code:     ret.Code,
		Message:  "OK",
		Roles:    ret.Roles,
		Total:    ret.Total,
		HaveMore: ret.HaveMore,
	}
	return connect.NewResponse(resp), nil
}

func (s *StoryRoleService) FollowStoryRole(ctx context.Context, req *connect.Request[gen.FollowStoryRoleRequest]) (*connect.Response[gen.FollowStoryRoleResponse], error) {
	if req.Msg.GetUserId() <= 0 {
		uid, err := utils.GetUserIDFromContext(ctx)
		if err != nil || uid <= 0 {
			return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("unauthenticated"))
		}
		req.Msg.UserId = uid
	}
	ret, err := storyEngineServer.GetStoryEngine().FollowStoryRole(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	resp := &gen.FollowStoryRoleResponse{
		Code:    ret.Code,
		Message: ret.Message,
	}
	return connect.NewResponse(resp), nil
}

func (s *StoryRoleService) LikeStoryRole(ctx context.Context, req *connect.Request[gen.LikeStoryRoleRequest]) (*connect.Response[gen.LikeStoryRoleResponse], error) {
	if req.Msg.GetUserId() <= 0 {
		uid, err := utils.GetUserIDFromContext(ctx)
		if err != nil || uid <= 0 {
			return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("unauthenticated"))
		}
		req.Msg.UserId = uid
	}
	ret, err := storyEngineServer.GetStoryEngine().LikeStoryRole(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	resp := &gen.LikeStoryRoleResponse{
		Code:    ret.Code,
		Message: ret.Message,
	}
	return connect.NewResponse(resp), nil
}

func (s *StoryRoleService) UnLikeStoryRole(ctx context.Context, req *connect.Request[gen.UnLikeStoryRoleRequest]) (*connect.Response[gen.UnLikeStoryRoleResponse], error) {
	if req.Msg.GetUserId() <= 0 {
		uid, err := utils.GetUserIDFromContext(ctx)
		if err != nil || uid <= 0 {
			return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("unauthenticated"))
		}
		req.Msg.UserId = uid
	}
	ret, err := storyEngineServer.GetStoryEngine().UnLikeStoryRole(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	resp := &gen.UnLikeStoryRoleResponse{
		Code:    ret.Code,
		Message: ret.Message,
	}
	return connect.NewResponse(resp), nil
}

// 获取用户创建的角色
func (s *StoryRoleService) GetUserCreatedRoles(ctx context.Context, req *connect.Request[gen.GetUserCreatedRolesRequest]) (*connect.Response[gen.GetUserCreatedRolesResponse], error) {
	ret, err := storyEngineServer.GetStoryEngine().GetUserCreatedRoles(ctx, req.Msg)
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
		HaveMore: ret.HaveMore,
	}
	return connect.NewResponse(resp), nil
}

func (s *StoryRoleService) GetStoryRoleStories(ctx context.Context, req *connect.Request[gen.GetStoryRoleStoriesRequest]) (*connect.Response[gen.GetStoryRoleStoriesResponse], error) {
	ret, err := storyEngineServer.GetStoryEngine().GetStoryRoleStories(ctx, req.Msg)
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
		HaveMore: ret.HaveMore,
	}
	return connect.NewResponse(resp), nil
}

func (s *StoryRoleService) GetStoryRoleStoryboards(ctx context.Context, req *connect.Request[gen.GetStoryRoleStoryboardsRequest]) (*connect.Response[gen.GetStoryRoleStoryboardsResponse], error) {
	ret, err := storyEngineServer.GetStoryEngine().GetStoryRoleStoryboards(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(ret), nil
}

func (s *StoryRoleService) CreateStoryRoleChat(ctx context.Context, req *connect.Request[gen.CreateStoryRoleChatRequest]) (*connect.Response[gen.CreateStoryRoleChatResponse], error) {
	ret, err := storyEngineServer.GetStoryEngine().CreateStoryRoleChat(ctx, req.Msg)
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
	ret, err := storyEngineServer.GetStoryEngine().ChatWithStoryRole(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	resp := &gen.ChatWithStoryRoleResponse{
		Code:          ret.Code,
		Message:       "OK",
		ReplyMessages: ret.ReplyMessages,
		Total:         ret.Total,
		HaveMore:      req.Msg.GetHaveMore(),
	}
	return connect.NewResponse(resp), nil
}

func (s *StoryRoleService) UpdateStoryRoleDetail(ctx context.Context, req *connect.Request[gen.UpdateStoryRoleDetailRequest]) (*connect.Response[gen.UpdateStoryRoleDetailResponse], error) {
	ret, err := storyEngineServer.GetStoryEngine().UpdateStoryRoleDetail(ctx, req.Msg)
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
	ret, err := storyEngineServer.GetStoryEngine().GetUserWithRoleChatList(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	resp := &gen.GetUserWithRoleChatListResponse{
		Code:     ret.Code,
		Message:  "OK",
		Chats:    ret.Chats,
		Total:    ret.Total,
		HaveMore: ret.HaveMore,
	}
	return connect.NewResponse(resp), nil
}

func (s *StoryRoleService) GetUserChatWithRole(ctx context.Context, req *connect.Request[gen.GetUserChatWithRoleRequest]) (*connect.Response[gen.GetUserChatWithRoleResponse], error) {
	ret, err := storyEngineServer.GetStoryEngine().GetUserChatWithRole(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	resp := &gen.GetUserChatWithRoleResponse{
		Code:        ret.Code,
		Message:     "OK",
		ChatContext: ret.ChatContext,
		Messages:    ret.Messages,
		Total:       ret.Total,
		HaveMore:    ret.HaveMore,
	}
	return connect.NewResponse(resp), nil
}

func (s *StoryRoleService) GetUserChatMessages(ctx context.Context, req *connect.Request[gen.GetUserChatMessagesRequest]) (*connect.Response[gen.GetUserChatMessagesResponse], error) {
	ret, err := storyEngineServer.GetStoryEngine().GetUserChatMessages(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(ret), nil
}

func (s *StoryRoleService) RenderStoryRoleContinuously(ctx context.Context, req *connect.Request[gen.RenderStoryRoleContinuouslyRequest]) (*connect.Response[gen.RenderStoryRoleContinuouslyResponse], error) {
	ret, err := storyEngineServer.GetStoryEngine().RenderStoryRoleContinuously(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(ret), nil
}

func (s *StoryRoleService) GenerateRoleDescription(ctx context.Context, req *connect.Request[gen.GenerateRoleDescriptionRequest]) (*connect.Response[gen.GenerateRoleDescriptionResponse], error) {
	ret, err := storyEngineServer.GetStoryEngine().GenerateRoleDescription(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(ret), nil
}

func (s *StoryRoleService) UpdateRoleDescription(ctx context.Context, req *connect.Request[gen.UpdateRoleDescriptionRequest]) (*connect.Response[gen.UpdateRoleDescriptionResponse], error) {
	ret, err := storyEngineServer.GetStoryEngine().UpdateRoleDescription(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(ret), nil
}

func (s *StoryRoleService) GenerateRolePrompt(ctx context.Context, req *connect.Request[gen.GenerateRolePromptRequest]) (*connect.Response[gen.GenerateRolePromptResponse], error) {
	ret, err := storyEngineServer.GetStoryEngine().GenerateRolePrompt(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(ret), nil
}

func (s *StoryRoleService) UpdateRolePrompt(ctx context.Context, req *connect.Request[gen.UpdateRolePromptRequest]) (*connect.Response[gen.UpdateRolePromptResponse], error) {
	ret, err := storyEngineServer.GetStoryEngine().UpdateRolePrompt(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(ret), nil
}

func (s *StoryRoleService) UpdateStoryRoleAvator(ctx context.Context, req *connect.Request[gen.UpdateStoryRoleAvatorRequest]) (*connect.Response[gen.UpdateStoryRoleAvatorResponse], error) {
	ret, err := storyEngineServer.GetStoryEngine().UpdateStoryRoleAvator(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(ret), nil
}

func (s *StoryRoleService) GetStoryRoleList(ctx context.Context, req *connect.Request[gen.GetStoryRoleListRequest]) (*connect.Response[gen.GetStoryRoleListResponse], error) {
	ret, err := storyEngineServer.GetStoryEngine().GetStoryRoleList(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(ret), nil
}

func (s *StoryRoleService) UpdateStoryRolePoster(ctx context.Context, req *connect.Request[gen.UpdateStoryRolePosterRequest]) (*connect.Response[gen.UpdateStoryRolePosterResponse], error) {
	ret, err := storyEngineServer.GetStoryEngine().UpdateStoryRolePoster(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(ret), nil
}

func (s *StoryRoleService) GenerateStoryRolePoster(ctx context.Context, req *connect.Request[gen.GenerateStoryRolePosterRequest]) (*connect.Response[gen.GenerateStoryRolePosterResponse], error) {
	ret, err := storyEngineServer.GetStoryEngine().GenerateStoryRolePoster(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(ret), nil
}

func (s *StoryRoleService) UpdateStoryRoleDescriptionDetail(ctx context.Context, req *connect.Request[gen.UpdateStoryRoleDescriptionDetailRequest]) (*connect.Response[gen.UpdateStoryRoleDescriptionDetailResponse], error) {
	ret, err := storyEngineServer.GetStoryEngine().UpdateStoryRoleDescriptionDetail(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(ret), nil
}

func (s *StoryRoleService) UpdateStoryRolePrompt(ctx context.Context, req *connect.Request[gen.UpdateStoryRolePromptRequest]) (*connect.Response[gen.UpdateStoryRolePromptResponse], error) {
	ret, err := storyEngineServer.GetStoryEngine().UpdateStoryRolePrompt(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(ret), nil
}

func (s *StoryRoleService) GenerateStoryRoleVideo(ctx context.Context, req *connect.Request[api.GenerateStoryRoleVideoRequest]) (*connect.Response[api.GenerateStoryRoleVideoResponse], error) {
	ret, err := storyEngineServer.GetStoryEngine().GenerateStoryRoleVideo(ctx, req.Msg)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&api.GenerateStoryRoleVideoResponse{
		Code:    ret.Code,
		Message: "OK",
		Detail:  ret.Detail,
	}), nil
}

func (s *StoryRoleService) GenerateRoleAvatar(ctx context.Context, req *connect.Request[api.GenerateRoleAvatarRequest]) (*connect.Response[api.GenerateRoleAvatarResponse], error) {
	ret, err := storyEngineServer.GetStoryEngine().GenerateRoleAvatar(ctx, req.Msg)
	if err != nil {
		log.Log().Error("GenerateRoleAvatar failed", zap.Error(err))
		return nil, err
	}
	return connect.NewResponse(ret), nil
}

func (s *StoryRoleService) LikeStoryRolePoster(ctx context.Context, req *connect.Request[api.LikeStoryRolePosterRequest]) (*connect.Response[api.LikeStoryRolePosterResponse], error) {
	if req.Msg.GetUserId() <= 0 {
		uid, err := utils.GetUserIDFromContext(ctx)
		if err != nil || uid <= 0 {
			return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("unauthenticated"))
		}
		req.Msg.UserId = uid
	}
	ret, err := storyEngineServer.GetStoryEngine().LikeStoryRolePoster(ctx, req.Msg)
	if err != nil {
		log.Log().Error("LikeStoryRolePoster failed", zap.Error(err))
		return nil, err
	}
	return connect.NewResponse(ret), nil
}

func (s *StoryRoleService) UnLikeStoryRolePoster(ctx context.Context, req *connect.Request[api.UnLikeStoryRolePosterRequest]) (*connect.Response[api.UnLikeStoryRolePosterResponse], error) {
	if req.Msg.GetUserId() <= 0 {
		uid, err := utils.GetUserIDFromContext(ctx)
		if err != nil || uid <= 0 {
			return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("unauthenticated"))
		}
		req.Msg.UserId = uid
	}
	ret, err := storyEngineServer.GetStoryEngine().UnLikeStoryRolePoster(ctx, req.Msg)
	if err != nil {
		log.Log().Error("UnLikeStoryRolePoster failed", zap.Error(err))
		return nil, err
	}
	return connect.NewResponse(ret), nil
}

func (s *StoryRoleService) GetStoryRolePosterList(ctx context.Context, req *connect.Request[api.GetStoryRolePosterListRequest]) (*connect.Response[api.GetStoryRolePosterListResponse], error) {
	ret, err := storyEngineServer.GetStoryEngine().GetStoryRolePosterList(ctx, req.Msg)
	if err != nil {
		log.Log().Error("GetStoryRolePoster failed", zap.Error(err))
		return nil, err
	}
	return connect.NewResponse(ret), nil
}
