package group

import (
	"context"
	"encoding/json"

	"go.uber.org/zap"

	api "github.com/grapery/common-protoc/gen"
	"github.com/grapery/grapery/models"
	"github.com/grapery/grapery/pkg/active"
	"github.com/grapery/grapery/pkg/cache"
	"github.com/grapery/grapery/pkg/cloud/aliyun"
	"github.com/grapery/grapery/utils"
	"github.com/grapery/grapery/utils/convert"
)

var (
	groupServer GroupServer
	logger, _   = zap.NewDevelopment()
	groupCache  *cache.GroupCache
)

func init() {
	groupServer = NewGroupService()
	groupCache = cache.NewGroupCache()
}

func GetGroupServer() GroupServer {
	return groupServer
}

func NewGroupService() *GroupService {
	return &GroupService{}
}

type GroupServer interface {
	GetGroup(ctx context.Context, req *api.GetGroupRequest) (resp *api.GetGroupResponse, err error)
	GetByName(ctx context.Context, req *api.GetGroupRequest) (resp *api.GetGroupResponse, err error)
	CreateGroup(ctx context.Context, req *api.CreateGroupRequest) (resp *api.CreateGroupResponse, err error)
	DeleteGroup(ctx context.Context, req *api.DeleteGroupRequest) (resp *api.DeleteGroupResponse, err error)
	GetGroupActives(ctx context.Context, req *api.GetGroupActivesRequest) (resp *api.GetGroupActivesResponse, err error)
	UpdateGroupInfo(ctx context.Context, req *api.UpdateGroupInfoRequest) (resp *api.UpdateGroupInfoResponse, err error)
	FetchGroupMembers(ctx context.Context, req *api.FetchGroupMembersRequest) (resp *api.FetchGroupMembersResponse, err error)
	JoinGroup(ctx context.Context, req *api.JoinGroupRequest) (resp *api.JoinGroupResponse, err error)
	LeaveGroup(ctx context.Context, req *api.LeaveGroupRequest) (resp *api.LeaveGroupResponse, err error)
	GetGroupProfile(ctx context.Context, req *api.GetGroupProfileRequest) (resp *api.GetGroupProfileResponse, err error)
	UpdateGroupProfile(ctx context.Context, req *api.UpdateGroupProfileRequest) (resp *api.UpdateGroupProfileResponse, err error)
	SearchGroup(ctx context.Context, req *api.SearchGroupRequest) (resp *api.SearchGroupResponse, err error)
	FetchGroupStorys(ctx context.Context, req *api.FetchGroupStorysRequest) (*api.FetchGroupStorysResponse, error)
}

type GroupService struct {
}

func (g *GroupService) GetGroup(ctx context.Context, req *api.GetGroupRequest) (resp *api.GetGroupResponse, err error) {
	traceId := utils.GetTraceID(ctx)
	logger.Info("GetGroup called", zap.String("traceId", traceId), zap.Any("req", req))
	// 参数校验
	if req.GetGroupId() <= 0 {
		logger.Error("GetGroup failed: invalid group id", zap.String("traceId", traceId), zap.Int64("group_id", req.GetGroupId()))
		return &api.GetGroupResponse{Code: api.ResponseCode_INVALID_PARAMETER, Message: "invalid group id"}, nil
	}
	if req.GetUserId() <= 0 {
		logger.Error("GetGroup failed: invalid user id", zap.String("traceId", traceId), zap.Int64("user_id", req.GetUserId()))
		return &api.GetGroupResponse{Code: api.ResponseCode_INVALID_PARAMETER, Message: "invalid user id"}, nil
	}

	// 尝试从缓存获取群组信息
	group, err := groupCache.GetGroupInfo(ctx, req.GetGroupId())
	if err != nil {
		logger.Debug("get group info from cache failed, fallback to database", zap.String("traceId", traceId), zap.Error(err))
		// 缓存未命中，从数据库获取
		group = &models.Group{}
		group.ID = uint(req.GetGroupId())
		err = group.GetByID(ctx)
		if err != nil {
			logger.Error("get group by id error", zap.String("traceId", traceId), zap.Error(err))
			return nil, err
		}
		// 将群组信息存入缓存
		if err := groupCache.SetGroupInfo(ctx, req.GetGroupId(), group); err != nil {
			logger.Warn("set group info to cache failed", zap.String("traceId", traceId), zap.Error(err))
		}
	} else {
		logger.Debug("get group info from cache success", zap.String("traceId", traceId))
	}
	creator := &models.User{}
	creator.ID = uint(req.GetUserId())
	err = creator.GetById(ctx)
	if err != nil {
		logger.Error("get user info by id failed", zap.String("traceId", traceId), zap.Error(err))
		return nil, err
	}
	// 尝试从缓存获取群组详细信息
	profile, err := groupCache.GetGroupProfile(ctx, int64(group.ID))
	if err != nil {
		logger.Debug("get group profile from cache failed, fallback to database", zap.String("traceId", traceId), zap.Error(err))
		// 缓存未命中，从数据库获取
		profile = &models.GroupProfile{}
		profile.GroupID = int64(group.ID)
		profile, err = models.GetGroupProfile(ctx, profile.GroupID)
		if err != nil {
			logger.Error("get group profile failed", zap.String("traceId", traceId), zap.Error(err))
			return nil, err
		}
		// 将群组详细信息存入缓存
		if profile != nil {
			if err := groupCache.SetGroupProfile(ctx, int64(group.ID), profile); err != nil {
				logger.Warn("set group profile to cache failed", zap.String("traceId", traceId), zap.Error(err))
			}
		}
	} else {
		logger.Debug("get group profile from cache success", zap.String("traceId", traceId))
	}
	var apiProfile *api.GroupProfileInfo
	if profile != nil {
		apiProfile = convert.ConvertGroupProfileToApiGroupProfile(profile)
		apiProfile.GroupId = int64(group.ID)
	}
	likeItems, err := models.GetLikeItemByGroup(ctx, []int64{req.GetGroupId()}, int(req.GetUserId()))
	if err != nil {
		logger.Info("get like item by group failed", zap.String("traceId", traceId), zap.Error(err))
	}
	likeMap := make(map[int64]bool)
	for _, val := range likeItems {
		likeMap[int64(val.GroupID)] = true
	}
	watchMap := make(map[int64]bool)
	watchItem, err := models.GetWatchItemByGroupAndUser(ctx, req.GetGroupId(), req.GetUserId())
	if err != nil {
		logger.Info("get watch item by group and user failed", zap.String("traceId", traceId), zap.Error(err))
	} else {
		watchMap[int64(watchItem.GroupID)] = true
	}
	// 尝试从缓存获取用户群组状态
	isIn, err := groupCache.GetUserGroupStatus(ctx, req.GetUserId(), req.GetGroupId())
	if err != nil {
		logger.Debug("get user group status from cache failed, fallback to database", zap.String("traceId", traceId), zap.Error(err))
		// 缓存未命中，从数据库检查
		groupMember := &models.GroupMember{
			GroupID: req.GetGroupId(),
			UserID:  req.GetUserId(),
		}
		isIn, err = groupMember.IsInGroup(ctx)
		if err != nil {
			logger.Info("get group member by group and user failed", zap.String("traceId", traceId), zap.Error(err))
		}
		// 将用户群组状态存入缓存
		if err := groupCache.SetUserGroupStatus(ctx, req.GetUserId(), req.GetGroupId(), isIn); err != nil {
			logger.Warn("set user group status to cache failed", zap.String("traceId", traceId), zap.Error(err))
		}
	} else {
		logger.Debug("get user group status from cache success", zap.String("traceId", traceId))
	}
	logger.Info("user is in/not in group", zap.String("traceId", traceId), zap.Int64("group_id", req.GetGroupId()), zap.Int64("user_id", req.GetUserId()), zap.Bool("is_in", isIn))
	resp = &api.GetGroupResponse{
		Code:    api.ResponseCode_OK,
		Message: "ok",
		Data: &api.GetGroupResponse_Data{
			Info: &api.GroupInfo{
				GroupId: int64(group.ID),
				Name:    utils.MaskContent(group.Name),
				Avatar:  group.Avatar,
				Desc:    utils.MaskContent(group.ShortDesc),
				Creator: int64(creator.ID),
				Owner:   int64(creator.ID),
				Profile: apiProfile,
				CurrentUserStatus: &api.WhatCurrentUserStatus{
					UserId:     utils.GetUserInfoFromMetadata(ctx),
					IsJoined:   isIn,
					IsFollowed: watchMap[int64(group.ID)],
					IsWatched:  watchMap[int64(group.ID)],
					IsLiked:    likeMap[int64(group.ID)],
				},
			},
		},
	}
	logger.Info("GetGroup success", zap.String("traceId", traceId), zap.Int64("group_id", req.GetGroupId()), zap.Any("resp", resp))
	return resp, nil
}

func (g *GroupService) GetByName(ctx context.Context, req *api.GetGroupRequest) (resp *api.GetGroupResponse, err error) {
	traceId := utils.GetTraceID(ctx)
	logger.Info("GetByName called", zap.String("traceId", traceId), zap.Any("req", req))
	if req.GetName() == "" {
		logger.Error("GetByName failed: name is empty", zap.String("traceId", traceId))
		return &api.GetGroupResponse{Code: api.ResponseCode_INVALID_PARAMETER, Message: "name is empty"}, nil
	}
	group := &models.Group{}
	group.Name = req.GetName()
	err = group.GetByName(ctx)
	if err != nil {
		logger.Error("get group by name error", zap.String("traceId", traceId), zap.Error(err))
		return nil, err
	}
	creator := &models.User{}
	creator.ID = uint(group.CreatorID)
	err = creator.GetById(ctx)
	if err != nil {
		logger.Error("get user info by id failed", zap.String("traceId", traceId), zap.Error(err))
		return nil, err
	}
	resp = &api.GetGroupResponse{
		Code:    api.ResponseCode_OK,
		Message: "ok",
		Data: &api.GetGroupResponse_Data{
			Info: &api.GroupInfo{
				GroupId: int64(group.ID),
				Name:    utils.MaskContent(group.Name),
				Avatar:  group.Avatar,
				Desc:    utils.MaskContent(group.ShortDesc),
				Creator: int64(creator.ID),
				Owner:   int64(creator.ID),
			},
		},
	}
	logger.Info("GetByName success", zap.String("traceId", traceId), zap.Int64("group_id", int64(group.ID)), zap.Any("resp", resp))
	return resp, nil
}

func (g *GroupService) CreateGroup(ctx context.Context, req *api.CreateGroupRequest) (resp *api.CreateGroupResponse, err error) {
	traceId := utils.GetTraceID(ctx)
	logger.Info("CreateGroup called", zap.String("traceId", traceId), zap.Any("req", req))
	if req.GetName() == "" {
		logger.Error("CreateGroup failed: name is empty", zap.String("traceId", traceId))
		return &api.CreateGroupResponse{Code: api.ResponseCode_INVALID_PARAMETER, Message: "name is empty"}, nil
	}
	if req.GetUserId() <= 0 {
		logger.Error("CreateGroup failed: invalid user id", zap.String("traceId", traceId), zap.Int64("user_id", req.GetUserId()))
		return &api.CreateGroupResponse{Code: api.ResponseCode_INVALID_PARAMETER, Message: "invalid user id"}, nil
	}
	group := &models.Group{}
	group.Name = req.Name
	group.CreatorID = req.GetUserId()
	group.OwnerID = req.GetUserId()
	desc := req.GetDescription()
	if desc == "" {
		desc = "这是一个神秘的组织"
	}
	group.ShortDesc = desc
	group.Description = desc
	if req.GetAvatar() != "" {
		group.Avatar = req.GetAvatar()
	} else {
		group.Avatar = utils.DefaultGroupAvatorUrl
	}
	err = group.Create(ctx)
	if err != nil {
		logger.Error("create group error", zap.String("traceId", traceId), zap.Error(err))
		return nil, err
	}
	creator := &models.User{}
	creator.ID = uint(group.CreatorID)
	err = creator.GetById(ctx)
	if err != nil {
		logger.Error("get user info by id failed", zap.String("traceId", traceId), zap.Error(err))
		return nil, err
	}
	logger.Info("create group success", zap.String("traceId", traceId), zap.Uint("group_id", group.ID), zap.String("group_name", utils.MaskContent(group.Name)), zap.Int64("creator_id", group.CreatorID))

	// 将新创建的群组信息存入缓存
	if err := groupCache.SetGroupInfo(ctx, int64(group.ID), group); err != nil {
		logger.Warn("set new group info to cache failed", zap.String("traceId", traceId), zap.Error(err))
	}

	err = models.CreateGroupProfile(ctx,
		int64(group.ID),
		desc,
		0, false, 1)
	if err != nil {
		logger.Error("create group profile failed", zap.String("traceId", traceId), zap.Error(err))
		return nil, err
	}
	err = models.CreateWatchGroupItem(ctx, int(group.CreatorID), int64(group.ID))
	if err != nil {
		logger.Warn("create watch group item failed", zap.String("traceId", traceId), zap.Error(err))
	}
	groupMember := &models.GroupMember{
		GroupID:  int64(group.ID),
		UserID:   int64(group.CreatorID),
		Role:     1,
		Nickname: creator.Name,
		Status:   1,
	}
	err = groupMember.Create(ctx)
	if err != nil {
		logger.Error("create group member failed", zap.String("traceId", traceId), zap.Error(err))
		return nil, err
	}
	userProfile := &models.UserProfile{UserId: req.GetUserId()}
	if err := userProfile.IncrementCreatedGroupNum(ctx); err != nil {
		logger.Warn("CreateGroup warning: increment created group num failed", zap.String("traceId", traceId), zap.Error(err))
	}
	if err := userProfile.IncrementWatchingGroupNum(ctx); err != nil {
		logger.Warn("CreateGroup warning: increment watching group num failed", zap.String("traceId", traceId), zap.Error(err))
	}
	resp = &api.CreateGroupResponse{
		Code:    api.ResponseCode_OK,
		Message: "ok",
		Data: &api.CreateGroupResponse_Data{
			Info: &api.GroupInfo{
				GroupId: int64(group.ID),
				Name:    utils.MaskContent(group.Name),
				Avatar:  group.Avatar,
				Desc:    utils.MaskContent(group.ShortDesc),
				Creator: int64(creator.ID),
				Owner:   int64(creator.ID),
			},
		},
	}
	logger.Info("CreateGroup success", zap.String("traceId", traceId), zap.Uint("group_id", group.ID), zap.Any("resp", resp))
	return resp, nil
}

func (g *GroupService) DeleteGroup(ctx context.Context, req *api.DeleteGroupRequest) (resp *api.DeleteGroupResponse, err error) {
	traceId := utils.GetTraceID(ctx)
	logger.Info("DeleteGroup called", zap.String("traceId", traceId), zap.Any("req", req))
	if req.GetGroupId() <= 0 {
		logger.Error("DeleteGroup failed: invalid group id", zap.String("traceId", traceId), zap.Int64("group_id", req.GetGroupId()))
		return &api.DeleteGroupResponse{Code: api.ResponseCode_INVALID_PARAMETER, Message: "invalid group id"}, nil
	}
	group := &models.Group{}
	group.ID = uint(req.GetGroupId())
	group.GetByID(ctx)
	if err != nil {
		logger.Error("DeleteGroup failed: get group by id error", zap.String("traceId", traceId), zap.Error(err))
		return &api.DeleteGroupResponse{Code: api.ResponseCode_OPERATION_FAILED, Message: err.Error()}, nil
	}
	if group.CreatorID != req.GetUserId() {
		logger.Error("DeleteGroup failed: user is not the creator of the group", zap.String("traceId", traceId), zap.Int64("group_id", req.GetGroupId()), zap.Int64("user_id", req.GetUserId()))
		return &api.DeleteGroupResponse{Code: api.ResponseCode_OPERATION_FAILED, Message: "user is not the creator of the group"}, nil
	}
	if group.IsDefault {
		logger.Error("DeleteGroup failed: default group cannot be deleted", zap.String("traceId", traceId), zap.Int64("group_id", req.GetGroupId()))
		return &api.DeleteGroupResponse{Code: api.ResponseCode_OPERATION_FAILED, Message: "default group cannot be deleted"}, nil
	}
	err = group.Delete(ctx)
	if err != nil {
		logger.Error("DeleteGroup failed: delete group error", zap.String("traceId", traceId), zap.Error(err))
		return &api.DeleteGroupResponse{Code: api.ResponseCode_OPERATION_FAILED, Message: err.Error()}, nil
	}
	err = models.UnWatchGroupItemByGroup(ctx, int64(req.GetGroupId()))
	if err != nil {
		logger.Error("DeleteGroup failed: unwatch group item failed", zap.String("traceId", traceId), zap.Error(err))
		return &api.DeleteGroupResponse{Code: api.ResponseCode_OPERATION_FAILED, Message: err.Error()}, nil
	}
	err = models.DisableGroupMembers(ctx, int64(req.GetGroupId()))
	if err != nil {
		logger.Error("DeleteGroup failed: disable group members failed", zap.String("traceId", traceId), zap.Error(err))
		return &api.DeleteGroupResponse{Code: api.ResponseCode_OPERATION_FAILED, Message: err.Error()}, nil
	}
	userProfile := &models.UserProfile{UserId: int64(group.CreatorID)}
	if err := userProfile.DecrementCreatedGroupNum(ctx); err != nil {
		logger.Warn("DeleteGroup warning: decrement created group num failed", zap.String("traceId", traceId), zap.Error(err))
	}
	if err := userProfile.DecrementWatchingGroupNum(ctx); err != nil {
		logger.Warn("DeleteGroup warning: decrement watching group num failed", zap.String("traceId", traceId), zap.Error(err))
	}

	// 删除群组成功后，清除所有相关缓存
	if err := groupCache.InvalidateGroupCache(ctx, req.GetGroupId()); err != nil {
		logger.Warn("invalidate group cache failed", zap.String("traceId", traceId), zap.Error(err))
	}
	resp = &api.DeleteGroupResponse{
		Code:    api.ResponseCode_OK,
		Message: "ok",
		Data:    &api.DeleteGroupResponse_Data{},
	}
	logger.Info("DeleteGroup success", zap.String("traceId", traceId), zap.Int64("group_id", req.GetGroupId()), zap.Any("resp", resp))
	return resp, nil
}

func (g *GroupService) GetGroupActives(ctx context.Context, req *api.GetGroupActivesRequest) (resp *api.GetGroupActivesResponse, err error) {
	traceId := utils.GetTraceID(ctx)
	logger.Info("GetGroupActives called", zap.String("traceId", traceId), zap.Any("req", req))
	if req.GetGroupId() <= 0 {
		logger.Error("GetGroupActives failed: invalid group id", zap.String("traceId", traceId), zap.Int64("group_id", req.GetGroupId()))
		return &api.GetGroupActivesResponse{Code: api.ResponseCode_INVALID_PARAMETER, Message: "invalid group id"}, nil
	}
	actives, hasMore, total, err := models.GetActiveByGroupID(ctx, req.GetGroupId(), int(req.GetPageSize()), int(req.GetOffset()))
	if err != nil {
		logger.Error("GetGroupActives failed: get active by group id error", zap.String("traceId", traceId), zap.Error(err))
		return nil, err
	}
	var list = make([]*api.ActiveInfo, len(*actives), len(*actives))
	for idx, active := range *actives {
		logger.Debug("active item", zap.String("traceId", traceId), zap.Any("active", active))
		list[idx] = convert.ConvertActiveToApiActive(active)
	}
	resp = &api.GetGroupActivesResponse{
		Code:    api.ResponseCode_OK,
		Message: "ok",
		Data: &api.GetGroupActivesResponse_Data{
			List:     list,
			HaveMore: hasMore,
			Total:    int64(total),
		},
	}
	return resp, nil
}

func (g *GroupService) UpdateGroupInfo(ctx context.Context, req *api.UpdateGroupInfoRequest) (resp *api.UpdateGroupInfoResponse, err error) {
	traceId := utils.GetTraceID(ctx)
	logger.Info("UpdateGroupInfo called", zap.String("traceId", traceId), zap.Any("req", req))
	if req.GetGroupId() <= 0 {
		logger.Error("UpdateGroupInfo failed: invalid group id", zap.String("traceId", traceId), zap.Int64("group_id", req.GetGroupId()))
		return &api.UpdateGroupInfoResponse{Code: api.ResponseCode_INVALID_PARAMETER, Message: "invalid group id"}, nil
	}
	group := new(models.Group)
	group.ID = uint(req.GetGroupId())
	err = group.GetByID(ctx)
	groupData, _ := json.Marshal(group)
	logger.Debug("update group info", zap.String("traceId", traceId), zap.Uint("group_id", group.ID), zap.String("data", string(groupData)))
	if err != nil {
		logger.Error("UpdateGroupInfo failed: group not found", zap.String("traceId", traceId), zap.Error(err))
		return &api.UpdateGroupInfoResponse{Code: api.ResponseCode_GROUP_NOT_FOUND, Message: err.Error()}, err
	}
	logger.Debug("UpdateGroupInfo params", zap.String("traceId", traceId), zap.String("req", req.String()))
	if req.GetInfo().GetAvatar() != "" {
		group.Avatar = req.GetInfo().GetAvatar()
		aliyunCli := aliyun.GetGlobalClient()
		err = aliyunCli.PersistImages(req.GetInfo().GetAvatar())
		if err != nil {
			logger.Error("UpdateGroupInfo failed: persist images error", zap.String("traceId", traceId), zap.Error(err))
			return &api.UpdateGroupInfoResponse{Code: api.ResponseCode_OPERATION_FAILED, Message: err.Error()}, err
		}
	}
	if req.GetInfo().GetDesc() != "" {
		group.ShortDesc = req.GetInfo().GetDesc()
	}
	if req.GetInfo().GetName() != "" {
		group.Name = req.GetInfo().GetName()
	}
	if req.GetInfo().Status != 0 {
		group.Status = int64(req.GetInfo().Status)
	}

	err = group.UpdateAll(ctx, req.GetGroupId())
	if err != nil {
		logger.Error("UpdateGroupInfo failed: update group error", zap.String("traceId", traceId), zap.Error(err))
		return &api.UpdateGroupInfoResponse{Code: api.ResponseCode_OPERATION_FAILED, Message: err.Error()}, err
	}

	// 更新成功后，清除相关缓存
	if err := groupCache.InvalidateGroupCache(ctx, req.GetGroupId()); err != nil {
		logger.Warn("invalidate group cache failed", zap.String("traceId", traceId), zap.Error(err))
	}
	resp = &api.UpdateGroupInfoResponse{
		Code:    api.ResponseCode_OK,
		Message: "ok",
		Data:    &api.UpdateGroupInfoResponse_Data{},
	}
	logger.Info("UpdateGroupInfo success", zap.String("traceId", traceId), zap.Uint("group_id", group.ID), zap.Any("resp", resp))
	return resp, nil
}

func (g *GroupService) FetchGroupMembers(ctx context.Context, req *api.FetchGroupMembersRequest) (resp *api.FetchGroupMembersResponse, err error) {
	traceId := utils.GetTraceID(ctx)
	logger.Info("FetchGroupMembers called", zap.String("traceId", traceId), zap.Any("req", req))
	if req.GetGroupId() <= 0 {
		logger.Error("FetchGroupMembers failed: invalid group id", zap.String("traceId", traceId), zap.Int64("group_id", req.GetGroupId()))
		return &api.FetchGroupMembersResponse{Code: api.ResponseCode_INVALID_PARAMETER, Message: "invalid group id"}, nil
	}

	// 尝试从缓存获取群组成员列表
	users, err := groupCache.GetGroupMembers(ctx, req.GetGroupId(), int(req.GetOffset()), int(req.GetPageSize()))
	if err != nil {
		logger.Debug("get group members from cache failed, fallback to database", zap.String("traceId", traceId), zap.Error(err))
		// 缓存未命中，从数据库获取
		users, err = models.GetGroupMemberInfoList(ctx, int(req.GetGroupId()), int(req.GetOffset()), int(req.GetPageSize()))
		if err != nil {
			logger.Error("FetchGroupMembers failed: get group member info list error", zap.String("traceId", traceId), zap.Error(err))
			return nil, err
		}
		// 将群组成员列表存入缓存
		if err := groupCache.SetGroupMembers(ctx, req.GetGroupId(), int(req.GetOffset()), int(req.GetPageSize()), users); err != nil {
			logger.Warn("set group members to cache failed", zap.String("traceId", traceId), zap.Error(err))
		}
	} else {
		logger.Debug("get group members from cache success", zap.String("traceId", traceId))
	}
	usersInfo := make([]*api.UserInfo, len(users), len(users))
	for idx := range users {
		usersInfo[idx] = convert.ConvertUserToApiUser(users[idx])
	}
	var (
		HasMore = false
	)
	if int64(len(usersInfo)) == req.GetPageSize() {
		HasMore = true
	}
	resp = &api.FetchGroupMembersResponse{
		Code:    api.ResponseCode_OK,
		Message: "ok",
		Data: &api.FetchGroupMembersResponse_Data{
			List:     usersInfo,
			Offset:   req.GetOffset() + int64(len(usersInfo)),
			Total:    int64(len(usersInfo)),
			HaveMore: HasMore,
		},
	}
	logger.Info("FetchGroupMembers success", zap.String("traceId", traceId), zap.Int64("group_id", req.GetGroupId()), zap.Any("resp", resp))
	return resp, nil
}

func (g *GroupService) JoinGroup(ctx context.Context, req *api.JoinGroupRequest) (resp *api.JoinGroupResponse, err error) {
	traceId := utils.GetTraceID(ctx)
	logger.Info("JoinGroup called", zap.String("traceId", traceId), zap.Any("req", req))
	if req.GetGroupId() <= 0 {
		logger.Error("JoinGroup failed: invalid group id", zap.String("traceId", traceId), zap.Int64("group_id", req.GetGroupId()))
		return &api.JoinGroupResponse{Code: api.ResponseCode_INVALID_PARAMETER, Message: "invalid group id"}, nil
	}
	if req.GetUserId() <= 0 {
		logger.Error("JoinGroup failed: invalid user id", zap.String("traceId", traceId), zap.Int64("user_id", req.GetUserId()))
		return &api.JoinGroupResponse{Code: api.ResponseCode_INVALID_PARAMETER, Message: "invalid user id"}, nil
	}
	group := &models.Group{}
	group.ID = uint(req.GetGroupId())
	err = group.GetByID(ctx)
	if err != nil {
		logger.Error("JoinGroup failed: group not found", zap.String("traceId", traceId), zap.Error(err))
		return &api.JoinGroupResponse{Code: api.ResponseCode_GROUP_NOT_FOUND, Message: err.Error()}, nil
	}
	member := &models.GroupMember{
		GroupID: req.GetGroupId(),
		UserID:  req.GetUserId(),
		Role:    api.GroupMemberType_GROUP_MEMBER_TYPE_MEMBER,
		Status:  1,
	}
	created, err := models.JoinGroupMember(ctx, member)
	if err != nil {
		logger.Error("JoinGroup failed: create group member error", zap.String("traceId", traceId), zap.Error(err))
		return &api.JoinGroupResponse{Code: api.ResponseCode_OPERATION_FAILED, Message: err.Error()}, nil
	}
	if !created {
		logger.Info("JoinGroup refused: user already in group", zap.String("traceId", traceId), zap.Int64("group_id", req.GetGroupId()), zap.Int64("user_id", req.GetUserId()))
		return &api.JoinGroupResponse{Code: api.ResponseCode_OK, Message: "user already in group"}, nil
	}

	// 加入群组成功后，清除相关缓存
	if err := groupCache.InvalidateUserGroupCache(ctx, req.GetUserId(), req.GetGroupId()); err != nil {
		logger.Warn("invalidate user group cache failed", zap.String("traceId", traceId), zap.Error(err))
	}
	userProfile := &models.UserProfile{UserId: req.GetUserId()}
	if err := userProfile.IncrementWatchingGroupNum(ctx); err != nil {
		logger.Warn("JoinGroup warning: increment watching group num failed", zap.String("traceId", traceId), zap.Error(err))
	}
	active.GetActiveServer().WriteGroupActive(ctx, group, nil, nil, req.GetUserId(), api.ActiveType_JoinGroup)
	logger.Info("JoinGroup success", zap.String("traceId", traceId), zap.Int64("group_id", req.GetGroupId()), zap.Int64("user_id", req.GetUserId()))
	resp = &api.JoinGroupResponse{Code: api.ResponseCode_OK, Message: "ok"}
	return resp, nil
}

func (g *GroupService) LeaveGroup(ctx context.Context, req *api.LeaveGroupRequest) (resp *api.LeaveGroupResponse, err error) {
	traceId := utils.GetTraceID(ctx)
	logger.Info("LeaveGroup called", zap.String("traceId", traceId), zap.Any("req", req))
	if req.GetGroupId() <= 0 {
		logger.Error("LeaveGroup failed: invalid group id", zap.String("traceId", traceId), zap.Int64("group_id", req.GetGroupId()))
		return &api.LeaveGroupResponse{Code: api.ResponseCode_INVALID_PARAMETER, Message: "invalid group id"}, nil
	}
	if req.GetUserId() <= 0 {
		logger.Error("LeaveGroup failed: invalid user id", zap.String("traceId", traceId), zap.Int64("user_id", req.GetUserId()))
		return &api.LeaveGroupResponse{Code: api.ResponseCode_INVALID_PARAMETER, Message: "invalid user id"}, nil
	}
	removed, err := models.LeaveGroupMember(ctx, req.GetGroupId(), req.GetUserId())
	if err != nil {
		logger.Error("LeaveGroup failed: delete group member error", zap.String("traceId", traceId), zap.Error(err))
		return &api.LeaveGroupResponse{Code: api.ResponseCode_OPERATION_FAILED, Message: err.Error()}, nil
	}
	if !removed {
		logger.Info("LeaveGroup refused: user not in group", zap.String("traceId", traceId), zap.Int64("group_id", req.GetGroupId()), zap.Int64("user_id", req.GetUserId()))
		return &api.LeaveGroupResponse{Code: api.ResponseCode_NOT_GROUP_MEMBER, Message: "user not in group"}, nil
	}

	// 离开群组成功后，清除相关缓存
	if err := groupCache.InvalidateUserGroupCache(ctx, req.GetUserId(), req.GetGroupId()); err != nil {
		logger.Warn("invalidate user group cache failed", zap.String("traceId", traceId), zap.Error(err))
	}
	userProfile := &models.UserProfile{UserId: req.GetUserId()}
	if err := userProfile.DecrementWatchingGroupNum(ctx); err != nil {
		logger.Warn("LeaveGroup warning: decrement watching group num failed", zap.String("traceId", traceId), zap.Error(err))
	}
	logger.Info("LeaveGroup success", zap.String("traceId", traceId), zap.Int64("group_id", req.GetGroupId()), zap.Int64("user_id", req.GetUserId()))
	resp = &api.LeaveGroupResponse{Code: api.ResponseCode_OK, Message: "ok"}
	return resp, nil
}

func (g *GroupService) GetGroupProfile(ctx context.Context, req *api.GetGroupProfileRequest) (resp *api.GetGroupProfileResponse, err error) {
	traceId := utils.GetTraceID(ctx)
	logger.Info("GetGroupProfile called", zap.String("traceId", traceId), zap.Any("req", req))
	if req.GetGroupId() <= 0 {
		logger.Error("GetGroupProfile failed: invalid group id", zap.String("traceId", traceId), zap.Int64("group_id", req.GetGroupId()))
		return &api.GetGroupProfileResponse{Code: api.ResponseCode_INVALID_PARAMETER, Message: "invalid group id"}, nil
	}
	group := &models.Group{}
	group.ID = uint(req.GetGroupId())
	err = group.GetByID(ctx)
	if err != nil {
		logger.Error("GetGroupProfile failed: get group by id error", zap.String("traceId", traceId), zap.Error(err))
		return nil, err
	}
	member, err := models.GetGroupMemberByGroupAndUser(ctx, req.GetGroupId(), req.GetUserId())
	if err != nil {
		logger.Error("GetGroupProfile failed: get group member by group and user error", zap.String("traceId", traceId), zap.Error(err))
		return nil, err
	}
	if member == nil {
		logger.Info("GetGroupProfile: user not in group", zap.String("traceId", traceId), zap.Int64("group_id", req.GetGroupId()), zap.Int64("user_id", req.GetUserId()))
		return &api.GetGroupProfileResponse{Code: api.ResponseCode_NOT_GROUP_MEMBER, Message: "user not in group"}, nil
	}
	profile := &models.GroupProfile{}
	profile.GroupID = req.GetGroupId()
	profile, err = models.GetGroupProfile(ctx, profile.GroupID)
	if err != nil {
		logger.Error("GetGroupProfile failed: get group profile error", zap.String("traceId", traceId), zap.Error(err))
		return nil, err
	}
	if profile == nil {
		logger.Info("group profile is nil", zap.String("traceId", traceId))
		resp = &api.GetGroupProfileResponse{
			Code:    api.ResponseCode_OK,
			Message: "ok",
			Data: &api.GetGroupProfileResponse_Data{
				Info: &api.GroupProfileInfo{
					GroupId:          req.GetGroupId(),
					Description:      "",
					GroupStoryNum:    0,
					GroupFollowerNum: 0,
					GroupMemberNum:   0,
					BackgroudUrl:     group.Avatar,
				},
			},
		}
		logger.Info("GetGroupProfile success (nil profile)", zap.String("traceId", traceId), zap.Any("resp", resp))
		return resp, nil
	}
	resp = &api.GetGroupProfileResponse{
		Code:    api.ResponseCode_OK,
		Message: "ok",
		Data: &api.GetGroupProfileResponse_Data{
			Info: convert.ConvertGroupProfileToApiGroupProfile(profile),
		},
	}
	if resp.Data.Info.BackgroudUrl == "" {
		resp.Data.Info.BackgroudUrl = group.Avatar
	}
	if resp.Data.Info.Description == "" {
		resp.Data.Info.Description = group.Description
	}
	if resp.Data.Info.GroupMemberNum == 0 {
		resp.Data.Info.GroupMemberNum = int32(group.Members)
	}
	logger.Info("GetGroupProfile success", zap.String("traceId", traceId), zap.Any("resp", resp))
	return resp, nil
}

func (g *GroupService) UpdateGroupProfile(ctx context.Context, req *api.UpdateGroupProfileRequest) (resp *api.UpdateGroupProfileResponse, err error) {
	traceId := utils.GetTraceID(ctx)
	logger.Info("UpdateGroupProfile called", zap.String("traceId", traceId), zap.Any("req", req))
	if req.GetGroupId() <= 0 {
		logger.Error("UpdateGroupProfile failed: invalid group id", zap.String("traceId", traceId), zap.Int64("group_id", req.GetGroupId()))
		return &api.UpdateGroupProfileResponse{Code: api.ResponseCode_INVALID_PARAMETER, Message: "invalid group id"}, nil
	}
	profile := req.GetInfo()
	err = models.UpdateGroupProfile(ctx,
		req.GetGroupId(),
		profile.GetDescription(),
		int64(profile.GetGroupFollowerNum()),
	)
	if err != nil {
		logger.Error("UpdateGroupProfile failed: update group profile error", zap.String("traceId", traceId), zap.Error(err))
		return &api.UpdateGroupProfileResponse{Code: api.ResponseCode_OPERATION_FAILED, Message: err.Error()}, err
	}

	// 更新群组详细信息成功后，清除相关缓存
	if err := groupCache.InvalidateGroupCache(ctx, req.GetGroupId()); err != nil {
		logger.Warn("invalidate group cache failed", zap.String("traceId", traceId), zap.Error(err))
	}
	resp = &api.UpdateGroupProfileResponse{Code: api.ResponseCode_OK, Message: "ok"}
	logger.Info("UpdateGroupProfile success", zap.String("traceId", traceId), zap.Any("resp", resp))
	return resp, nil
}

func (g *GroupService) SearchGroup(ctx context.Context, req *api.SearchGroupRequest) (resp *api.SearchGroupResponse, err error) {
	traceId := utils.GetTraceID(ctx)
	logger.Info("SearchGroup called", zap.String("traceId", traceId), zap.Any("req", req))
	if req.GetName() == "" {
		logger.Error("SearchGroup failed: name is empty", zap.String("traceId", traceId))
		return &api.SearchGroupResponse{Code: api.ResponseCode_INVALID_PARAMETER, Message: "name is empty"}, nil
	}
	if req.GetOffset() < 0 || req.GetPageSize() < 0 {
		logger.Error("SearchGroup failed: offset or pageSize < 0", zap.String("traceId", traceId), zap.Int64("offset", req.GetOffset()), zap.Int64("pageSize", req.GetPageSize()))
		return &api.SearchGroupResponse{Code: api.ResponseCode_INVALID_PARAMETER, Message: "offset or pageSize < 0"}, nil
	}

	name := req.GetName()
	// 尝试从缓存获取群组搜索结果
	groups, total, err := groupCache.GetGroupSearch(ctx, name, int(req.GetOffset()), int(req.GetPageSize()))
	if err != nil {
		logger.Debug("get group search from cache failed, fallback to database", zap.String("traceId", traceId), zap.Error(err))
		// 缓存未命中，从数据库搜索
		groups, total, err = models.GetGroupByName(name, int(req.GetOffset()), int(req.GetPageSize()))
		if err != nil {
			logger.Error("SearchGroup failed: get group by name error", zap.String("traceId", traceId), zap.Error(err))
			return nil, err
		}
		// 将群组搜索结果存入缓存
		if err := groupCache.SetGroupSearch(ctx, name, int(req.GetOffset()), int(req.GetPageSize()), groups, total); err != nil {
			logger.Warn("set group search to cache failed", zap.String("traceId", traceId), zap.Error(err))
		}
	} else {
		logger.Debug("get group search from cache success", zap.String("traceId", traceId))
	}
	list := make([]*api.GroupInfo, len(groups), len(groups))
	for idx, val := range groups {
		list[idx] = convert.ConvertGroupToApiGroupInfo(val)
	}
	resp = &api.SearchGroupResponse{
		Code:    api.ResponseCode_OK,
		Message: "ok",
		Data: &api.SearchGroupResponse_Data{
			List:     list,
			Offset:   req.GetOffset() + int64(len(list)),
			Total:    total,
			HaveMore: total > int64(req.GetOffset()+int64(len(list))),
		},
	}
	logger.Info("SearchGroup success", zap.String("traceId", traceId), zap.Any("resp", resp))
	return resp, nil
}

func (g *GroupService) FetchGroupStorys(ctx context.Context, req *api.FetchGroupStorysRequest) (*api.FetchGroupStorysResponse, error) {
	traceId := utils.GetTraceID(ctx)
	logger.Info("FetchGroupStorys called", zap.String("traceId", traceId), zap.Any("req", req))
	if req.GetGroupId() <= 0 {
		logger.Error("FetchGroupStorys failed: invalid group id", zap.String("traceId", traceId), zap.Int64("group_id", req.GetGroupId()))
		return &api.FetchGroupStorysResponse{Code: int32(api.ResponseCode_INVALID_PARAMETER), Message: "invalid group id"}, nil
	}

	// 尝试从缓存获取群组故事列表
	storys, total, hasMore, err := groupCache.GetGroupStories(ctx, req.GetGroupId(), int(req.GetPage()), int(req.GetPageSize()))
	if err != nil {
		logger.Debug("get group stories from cache failed, fallback to database", zap.String("traceId", traceId), zap.Error(err))
		// 缓存未命中，从数据库获取
		storys, total, hasMore, err = models.GetStoryByGroupID(ctx, req.GetGroupId(), int(req.GetPage()), int(req.GetPageSize()))
		if err != nil {
			logger.Error("FetchGroupStorys failed: get story by group id error", zap.String("traceId", traceId), zap.Error(err))
			return nil, err
		}
		// 将群组故事列表存入缓存
		if err := groupCache.SetGroupStories(ctx, req.GetGroupId(), int(req.GetPage()), int(req.GetPageSize()), storys, total, hasMore); err != nil {
			logger.Warn("set group stories to cache failed", zap.String("traceId", traceId), zap.Error(err))
		}
	} else {
		logger.Debug("get group stories from cache success", zap.String("traceId", traceId))
	}

	storysIds := make([]int64, 0)
	for _, val := range storys {
		storysIds = append(storysIds, int64(val.ID))
	}
	likeItems, err := models.GetLikeItemByStoriesAndUser(ctx, storysIds, int(req.GetUserId()))
	if err != nil {
		logger.Warn("FetchGroupStorys: get like item by stories and user failed", zap.String("traceId", traceId), zap.Error(err))
	}
	likeMap := make(map[int64]bool)
	for _, val := range likeItems {
		likeMap[int64(val.StoryID)] = true
	}
	watchItems, err := models.GetWatchItemByStoriesAndUser(ctx, storysIds, int(req.GetUserId()))
	if err != nil {
		logger.Warn("FetchGroupStorys: get watch item by stories and user failed", zap.String("traceId", traceId), zap.Error(err))
	}
	watchMap := make(map[int64]bool)
	for _, val := range watchItems {
		watchMap[int64(val.StoryID)] = true
	}
	list := make([]*api.Story, len(storys), len(storys))
	for idx, val := range storys {
		storyItem := convert.ConvertStoryToApiStory(val)
		storyItem.CurrentUserStatus = &api.WhatCurrentUserStatus{
			UserId:    req.GetUserId(),
			IsLiked:   likeMap[int64(val.ID)],
			IsWatched: watchMap[int64(val.ID)],
		}
		if likeMap[int64(val.ID)] {
			storyItem.Isliked = true
		}
		if watchMap[int64(val.ID)] {
			storyItem.Iswatched = true
		}
		list[idx] = storyItem
	}
	resp := &api.FetchGroupStorysResponse{
		Code:    int32(api.ResponseCode_OK),
		Message: "ok",
		Data: &api.FetchGroupStorysResponse_Data{
			List:     list,
			Total:    total,
			HaveMore: hasMore,
		},
	}
	logger.Info("FetchGroupStorys success", zap.String("traceId", traceId), zap.Any("resp", resp))
	return resp, nil
}
