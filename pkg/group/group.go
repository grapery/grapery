package group

import (
	"context"
	"encoding/json"

	"go.uber.org/zap"

	api "github.com/grapery/common-protoc/gen"
	"github.com/grapery/grapery/models"
	"github.com/grapery/grapery/pkg/active"
	"github.com/grapery/grapery/utils"
	"github.com/grapery/grapery/utils/convert"
	"github.com/grapery/grapery/utils/errors"
)

var (
	groupServer GroupServer
	logger, _   = zap.NewDevelopment()
)

func init() {
	groupServer = NewGroupService()
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
	group := &models.Group{}
	group.ID = uint(req.GetGroupId())
	err = group.GetByID()
	if err != nil {
		logger.Error("get group by id error", zap.Error(err))
		return nil, err
	}
	creator := &models.User{}
	creator.ID = uint(req.GetUserId())
	err = creator.GetById()
	if err != nil {
		logger.Error("get user info by id failed", zap.Error(err))
		return nil, err
	}
	profile := &models.GroupProfile{}
	profile.GroupID = int64(group.ID)
	profile, err = models.GetGroupProfile(ctx, profile.GroupID)
	if err != nil {
		logger.Error("get group profile failed", zap.Error(err))
		return nil, err
	}
	var apiProfile *api.GroupProfileInfo
	if profile != nil {
		apiProfile = convert.ConvertGroupProfileToApiGroupProfile(profile)
		apiProfile.GroupId = int64(group.ID)
	}
	likeItems, err := models.GetLikeItemByGroup(ctx, []int64{req.GetGroupId()}, int(req.GetUserId()))
	if err != nil {
		logger.Info("get like item by group failed", zap.Error(err))
	}
	likeMap := make(map[int64]bool)
	for _, val := range likeItems {
		likeMap[int64(val.GroupID)] = true
	}
	watchMap := make(map[int64]bool)
	watchItem, err := models.GetWatchItemByGroupAndUser(ctx, req.GetGroupId(), req.GetUserId())
	if err != nil {
		logger.Info("get watch item by group and user failed", zap.Error(err))
	} else {
		watchMap[int64(watchItem.GroupID)] = true
	}
	var isIn bool = false
	groupMember := &models.GroupMember{
		GroupID: req.GetGroupId(),
		UserID:  req.GetUserId(),
	}
	isIn, err = groupMember.IsInGroup()
	if err != nil {
		logger.Info("get group member by group and user failed", zap.Error(err))
	}
	logger.Info("user is in/not in group", zap.Int64("group_id", req.GetGroupId()), zap.Int64("user_id", req.GetUserId()), zap.Bool("is_in", isIn))
	return &api.GetGroupResponse{
		Code:    api.ResponseCode_OK,
		Message: "ok",
		Data: &api.GetGroupResponse_Data{
			Info: &api.GroupInfo{
				GroupId: int64(group.ID),
				Name:    group.Name,
				Avatar:  group.Avatar,
				Desc:    group.ShortDesc,
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
	}, nil
}

func (g *GroupService) GetByName(ctx context.Context, req *api.GetGroupRequest) (resp *api.GetGroupResponse, err error) {
	group := &models.Group{}
	group.Name = req.GetName()
	err = group.GetByName()
	if err != nil {
		logger.Error("get group by name error", zap.Error(err))
		return nil, err
	}
	creator := &models.User{}
	creator.ID = uint(group.CreatorID)
	err = creator.GetById()
	if err != nil {
		logger.Error("get user info by id failed", zap.Error(err))
		return nil, err
	}
	return &api.GetGroupResponse{
		Code:    api.ResponseCode_OK,
		Message: "ok",
		Data: &api.GetGroupResponse_Data{
			Info: &api.GroupInfo{
				GroupId: int64(group.ID),
				Name:    group.Name,
				Avatar:  group.Avatar,
				Desc:    group.ShortDesc,
				Creator: int64(creator.ID),
				Owner:   int64(creator.ID),
			},
		},
	}, nil
}

func (g *GroupService) CreateGroup(ctx context.Context, req *api.CreateGroupRequest) (resp *api.CreateGroupResponse, err error) {
	group := &models.Group{}
	group.Name = req.Name
	group.CreatorID = req.GetUserId()
	group.OwnerID = req.GetUserId()
	group.Members = 1
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
	err = group.Create()
	if err != nil {
		logger.Info("create group error", zap.Error(err))
		return nil, err
	}
	creator := &models.User{}
	creator.ID = uint(group.CreatorID)
	err = creator.GetById()
	if err != nil {
		logger.Info("get user info by id failed", zap.Error(err))
		return nil, err
	}
	logger.Info("create group success", zap.Uint("group_id", group.ID), zap.String("group_name", group.Name), zap.Int64("creator_id", group.CreatorID))
	err = models.CreateGroupProfile(ctx,
		int64(group.ID),
		desc,
		0, false, 1)
	if err != nil {
		return nil, err
	}
	err = models.CreateWatchGroupItem(ctx, int(group.CreatorID), int64(group.ID))
	if err != nil {
		logger.Info("create watch group item failed", zap.Error(err))
	}
	groupMember := &models.GroupMember{
		GroupID:  int64(group.ID),
		UserID:   int64(group.CreatorID),
		Role:     1,
		Nickname: creator.Name,
		Status:   1,
	}
	err = groupMember.Create()
	if err != nil {
		logger.Info("create group member failed", zap.Error(err))
		return nil, err
	}
	return &api.CreateGroupResponse{
		Code:    api.ResponseCode_OK,
		Message: "ok",
		Data: &api.CreateGroupResponse_Data{
			Info: &api.GroupInfo{
				GroupId: int64(group.ID),
				Name:    group.Name,
				Avatar:  group.Avatar,
				Desc:    group.ShortDesc,
				Creator: int64(creator.ID),
				Owner:   int64(creator.ID),
			},
		},
	}, nil
}

func (g *GroupService) DeleteGroup(ctx context.Context, req *api.DeleteGroupRequest) (resp *api.DeleteGroupResponse, err error) {
	group := &models.Group{}
	group.ID = uint(req.GetGroupId())
	err = group.Delete()
	if err != nil {
		return &api.DeleteGroupResponse{Code: api.ResponseCode_OPERATION_FAILED, Message: err.Error()}, nil
	}
	return &api.DeleteGroupResponse{
		Code:    api.ResponseCode_OK,
		Message: "ok",
		Data:    &api.DeleteGroupResponse_Data{},
	}, nil
}

func (g *GroupService) GetGroupActives(ctx context.Context, req *api.GetGroupActivesRequest) (resp *api.GetGroupActivesResponse, err error) {
	actives, err := models.GetActiveByGroupID(req.GetGroupId())
	if err != nil {
		return nil, err
	}
	var list = make([]*api.ActiveInfo, len(*actives), len(*actives))
	for _, active := range *actives {
		logger.Info("active item", zap.Any("active", active))
		//list[idx]....
	}
	return &api.GetGroupActivesResponse{
		Code:    api.ResponseCode_OK,
		Message: "ok",
		Data: &api.GetGroupActivesResponse_Data{
			List: list,
		},
	}, nil
}

func (g *GroupService) UpdateGroupInfo(ctx context.Context, req *api.UpdateGroupInfoRequest) (resp *api.UpdateGroupInfoResponse, err error) {
	group := new(models.Group)
	group.ID = uint(req.GetGroupId())
	err = group.GetByID()
	groupData, _ := json.Marshal(group)
	logger.Debug("update group info", zap.Uint("group_id", group.ID), zap.String("data", string(groupData)))
	if err != nil {
		return &api.UpdateGroupInfoResponse{Code: api.ResponseCode_GROUP_NOT_FOUND, Message: err.Error()}, err
	}
	logger.Debug("UpdateGroupInfo params", zap.String("req", req.String()))
	if req.GetInfo().GetAvatar() != "" {
		group.Avatar = req.GetInfo().GetAvatar()
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

	err = group.UpdateAll(req.GetGroupId())
	if err != nil {
		return &api.UpdateGroupInfoResponse{Code: api.ResponseCode_OPERATION_FAILED, Message: err.Error()}, err
	}
	return &api.UpdateGroupInfoResponse{
		Code:    api.ResponseCode_OK,
		Message: "ok",
		Data:    &api.UpdateGroupInfoResponse_Data{},
	}, nil
}

func (g *GroupService) FetchGroupMembers(ctx context.Context, req *api.FetchGroupMembersRequest) (resp *api.FetchGroupMembersResponse, err error) {
	users, err := models.GetGroupMemberInfoList(int(req.GetGroupId()), int(req.GetOffset()), int(req.GetPageSize()))
	if err != nil {
		return nil, err
	}
	usersInfo := make([]*api.UserInfo, len(users), len(users))
	for idx := range users {
		usersInfo[idx] = convert.ConvertUserToApiUser(users[idx])
	}

	return &api.FetchGroupMembersResponse{
		Code:    api.ResponseCode_OK,
		Message: "ok",
		Data: &api.FetchGroupMembersResponse_Data{
			List:   usersInfo,
			Offset: req.GetOffset() + int64(len(usersInfo)),
			Total:  int64(len(usersInfo)),
		},
	}, nil
}

func (g *GroupService) JoinGroup(ctx context.Context, req *api.JoinGroupRequest) (resp *api.JoinGroupResponse, err error) {
	group := &models.Group{}
	group.ID = uint(req.GetGroupId())
	err = group.GetByID()
	if err != nil {
		return &api.JoinGroupResponse{Code: api.ResponseCode_GROUP_NOT_FOUND, Message: err.Error()}, nil
	}
	groupMember := &models.GroupMember{
		GroupID: req.GetGroupId(),
		UserID:  req.GetUserId(),
	}
	isIn, err := groupMember.IsInGroup()
	if err != nil {
		return &api.JoinGroupResponse{Code: api.ResponseCode_DATABASE_ERROR, Message: err.Error()}, nil
	}
	if isIn {
		return &api.JoinGroupResponse{Code: api.ResponseCode_GROUP_ALREADY_EXISTS, Message: "user already in group"}, nil
	}
	err = groupMember.Create()
	if err != nil {
		return &api.JoinGroupResponse{Code: api.ResponseCode_OPERATION_FAILED, Message: err.Error()}, nil
	}
	err = models.IncGroupProfileMembers(ctx, req.GetGroupId())
	if err != nil {
		return &api.JoinGroupResponse{Code: api.ResponseCode_OPERATION_FAILED, Message: err.Error()}, nil
	}

	active.GetActiveServer().WriteGroupActive(ctx, group, nil, nil, req.GetUserId(), api.ActiveType_JoinGroup)
	return &api.JoinGroupResponse{Code: api.ResponseCode_OK, Message: "ok"}, nil
}

func (g *GroupService) LeaveGroup(ctx context.Context, req *api.LeaveGroupRequest) (resp *api.LeaveGroupResponse, err error) {
	groupMember := &models.GroupMember{
		GroupID: int64(req.GetGroupId()),
		UserID:  req.GetUserId(),
	}
	isIn, err := groupMember.IsInGroup()
	if err != nil {
		return &api.LeaveGroupResponse{Code: api.ResponseCode_DATABASE_ERROR, Message: err.Error()}, nil
	}
	if !isIn {
		return &api.LeaveGroupResponse{Code: api.ResponseCode_NOT_GROUP_MEMBER, Message: "user not in group"}, nil
	}
	err = groupMember.Delete()
	if err != nil {
		return &api.LeaveGroupResponse{Code: api.ResponseCode_OPERATION_FAILED, Message: err.Error()}, nil
	}
	err = models.DecGroupProfileMembers(ctx, req.GetGroupId())
	if err != nil {
		return &api.LeaveGroupResponse{Code: api.ResponseCode_OPERATION_FAILED, Message: err.Error()}, nil
	}
	return &api.LeaveGroupResponse{Code: api.ResponseCode_OK, Message: "ok"}, nil
}

func (g *GroupService) GetGroupProfile(ctx context.Context, req *api.GetGroupProfileRequest) (resp *api.GetGroupProfileResponse, err error) {
	profile := &models.GroupProfile{}
	profile.GroupID = req.GetGroupId()
	profile, err = models.GetGroupProfile(ctx, profile.GroupID)
	if err != nil {
		logger.Info("get group profile failed", zap.Error(err))
		return nil, err
	}
	if profile == nil {
		logger.Info("group profile is nil")
		return &api.GetGroupProfileResponse{
			Code:    api.ResponseCode_OK,
			Message: "ok",
			Data: &api.GetGroupProfileResponse_Data{
				Info: &api.GroupProfileInfo{
					GroupId:          req.GetGroupId(),
					Description:      "",
					GroupStoryNum:    0,
					GroupFollowerNum: 0,
					GroupMemberNum:   0,
				},
			},
		}, nil
	}
	return &api.GetGroupProfileResponse{
		Code:    api.ResponseCode_OK,
		Message: "ok",
		Data: &api.GetGroupProfileResponse_Data{
			Info: convert.ConvertGroupProfileToApiGroupProfile(profile),
		},
	}, nil
}

func (g *GroupService) UpdateGroupProfile(ctx context.Context, req *api.UpdateGroupProfileRequest) (resp *api.UpdateGroupProfileResponse, err error) {
	profile := req.GetInfo()
	err = models.UpdateGroupProfile(ctx,
		req.GetGroupId(),
		profile.GetDescription(),
		int64(profile.GetGroupFollowerNum()),
	)
	if err != nil {
		return &api.UpdateGroupProfileResponse{Code: api.ResponseCode_OPERATION_FAILED, Message: err.Error()}, err
	}
	return &api.UpdateGroupProfileResponse{Code: api.ResponseCode_OK, Message: "ok"}, nil
}

func (g *GroupService) SearchGroup(ctx context.Context, req *api.SearchGroupRequest) (resp *api.SearchGroupResponse, err error) {
	name := req.GetName()
	if name == "" {
		return nil, errors.ErrMissingParameter
	}
	if req.GetOffset() < 0 || req.GetPageSize() < 0 {
		return nil, errors.ErrInvalidParameter
	}
	groups, total, err := models.GetGroupByName(name, int(req.GetOffset()), int(req.GetPageSize()))
	if err != nil {
		return nil, err
	}
	list := make([]*api.GroupInfo, len(groups), len(groups))
	for idx, val := range groups {
		list[idx] = convert.ConvertGroupToApiGroupInfo(val)
	}
	return &api.SearchGroupResponse{
		Code:    api.ResponseCode_OK,
		Message: "ok",
		Data: &api.SearchGroupResponse_Data{
			List:     list,
			Offset:   total - int64(len(list)),
			PageSize: int64(len(list)),
		},
	}, nil
}

func (g *GroupService) FetchGroupStorys(ctx context.Context, req *api.FetchGroupStorysRequest) (*api.FetchGroupStorysResponse, error) {
	// TODO: 实现获取群组的故事列表
	storys, err := models.GetStoryByGroupID(ctx, req.GetGroupId(), int(req.GetPage()), int(req.GetPageSize()))
	if err != nil {
		logger.Info("get story by group id failed", zap.Error(err))
		return nil, err
	}

	storysIds := make([]int64, 0)
	for _, val := range storys {
		storysIds = append(storysIds, int64(val.ID))
	}
	likeItems, err := models.GetLikeItemByStoriesAndUser(ctx, storysIds, int(req.GetUserId()))
	if err != nil {
		logger.Info("get like item by stories and user failed", zap.Error(err))
	}
	likeMap := make(map[int64]bool)
	for _, val := range likeItems {
		likeMap[int64(val.StoryID)] = true
	}
	watchItems, err := models.GetWatchItemByStoriesAndUser(ctx, storysIds, int(req.GetUserId()))
	if err != nil {
		logger.Info("get watch item by stories and user failed", zap.Error(err))
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
	return &api.FetchGroupStorysResponse{
		Code:    int32(api.ResponseCode_OK),
		Message: "ok",
		Data: &api.FetchGroupStorysResponse_Data{
			List: list,
		},
	}, nil
}
