package user

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"time"

	log "github.com/sirupsen/logrus"

	api "github.com/grapery/common-protoc/gen"
	"github.com/grapery/grapery/models"
	"github.com/grapery/grapery/utils"
)

var userServer UserServer

func init() {
	userServer = NewUserSerivce()
}

func GetUserServer() UserServer {
	return userServer
}

func NewUserSerivce() *UserService {
	return &UserService{}
}

type UserServer interface {
	GetUserInfo(ctx context.Context, req *api.UserInfoRequest) (*api.UserInfoResponse, error)
	UpdateAvator(ctx context.Context, req *api.UpdateUserAvatorRequest) (*api.UpdateUserAvatorResponse, error)
	GetUserGroup(ctx context.Context, req *api.UserGroupRequest) (*api.UserGroupResponse, error)
	GetUserFollowingGroup(ctx context.Context, req *api.UserFollowingGroupRequest) (*api.UserFollowingGroupResponse, error)
	UpdateUser(ctx context.Context, req *api.UserUpdateRequest) (*api.UserUpdateResponse, error)
	FetchActives(ctx context.Context, req *api.FetchActivesRequest) (*api.FetchActivesResponse, error)
	SearchUser(ctx context.Context, req *api.SearchUserRequest) (*api.SearchUserResponse, error)
	UserWatching(ctx context.Context, req *api.UserWatchingRequest) (*api.UserWatchingResponse, error)
	UserInit(ctx context.Context, req *api.UserInitRequest) (*api.UserInitResponse, error)
	GetUserProfile(ctx context.Context, req *api.GetUserProfileRequest) (*api.GetUserProfileResponse, error)
	UpdateUserProfile(ctx context.Context, req *api.UpdateUserProfileRequest) (*api.UpdateUserProfileResponse, error)
	UpdateUserBackgroundImage(ctx context.Context, req *api.UpdateUserBackgroundImageRequest) (*api.UpdateUserBackgroundImageResponse, error)
}

type UserService struct {
}

func (user *UserService) UserInit(ctx context.Context, req *api.UserInitRequest) (*api.UserInitResponse, error) {
	defer func() {
		if re := recover(); re != nil {
			fmt.Println("re: ", re)
		}
	}()
	defaultGroup, ok, err := models.GetUserDefaultGroup(int(req.GetUserId()))
	if err != nil {
		return nil, err
	}
	if ok {
		if defaultGroup.OwnerID != req.GetUserId() {
			return nil, fmt.Errorf("user %d default group info not match", req.GetUserId())
		}
		return &api.UserInitResponse{
			Code: 0,
			Msg:  "success",
			Data: &api.UserInitResponse_Data{
				UserId: req.GetUserId(),
				List: []*api.GroupInfo{
					{
						GroupId: int64(defaultGroup.ID),
						Name:    defaultGroup.Name,
						Avatar:  defaultGroup.Avatar,
						Desc:    defaultGroup.Gtype,
						Creator: req.GetUserId(),
						Ctime:   defaultGroup.CreateAt.Unix(),
						Mtime:   defaultGroup.UpdateAt.Unix(),
					},
				},
			},
		}, nil
	}
	// user default group is not exist,need create one
	if !ok {
		defaultGroup, ok, err = models.GetUserDefaultGroup(int(req.GetUserId()))
		if !ok {
			return nil, fmt.Errorf("create default group failed: %+v", err)
		}
	}
	if defaultGroup.OwnerID != req.GetUserId() {
		return nil, fmt.Errorf("user %d default group info not match", req.GetUserId())
	}
	return &api.UserInitResponse{
		Code: 0,
		Msg:  "success",
		Data: &api.UserInitResponse_Data{
			UserId: req.GetUserId(),
			List: []*api.GroupInfo{
				{
					GroupId: int64(defaultGroup.ID),
					Name:    defaultGroup.Name,
					Avatar:  defaultGroup.Avatar,
					Desc:    defaultGroup.Gtype,
					Creator: req.GetUserId(),
					Ctime:   defaultGroup.CreateAt.Unix(),
					Mtime:   defaultGroup.UpdateAt.Unix(),
				},
			},
		},
	}, nil
}

func (user *UserService) GetUserInfo(ctx context.Context, req *api.UserInfoRequest) (*api.UserInfoResponse, error) {
	var u = new(models.User)
	u.ID = uint(req.GetUserId())
	err := u.GetById()
	if err != nil {
		log.Errorf("get user failed : %s", err.Error())
		return nil, err
	}
	userProfile := &models.UserProfile{
		UserId: int64(u.ID),
	}
	err = userProfile.GetByUserId()
	if err != nil {
		log.Errorf("get user profile failed : %s", err.Error())
		return nil, err
	}
	return &api.UserInfoResponse{
		Code: 0,
		Msg:  "success",
		Data: &api.UserInfoResponse_Data{
			Info: &api.UserInfo{
				UserId:   int64(u.ID),
				Name:     u.Name,
				Avatar:   u.Avatar,
				Email:    u.Email,
				Location: u.Location,
				Desc:     u.ShortDesc,
				Ctime:    u.CreateAt.Unix(),
				Mtime:    u.UpdateAt.Unix(),
			},
			Profile: convertModelUserProfileToApi(userProfile),
		},
	}, nil
}

func (user *UserService) UpdateAvator(ctx context.Context, req *api.UpdateUserAvatorRequest) (
	*api.UpdateUserAvatorResponse, error) {
	var u = new(models.User)
	u.ID = uint(req.GetUserId())
	u.Avatar = req.GetAvatar()
	err := u.UpdateAvatar()
	if err != nil {
		log.Errorf("get user failed : %s", err.Error())
		return nil, err
	}
	return &api.UpdateUserAvatorResponse{
		Code: 0,
		Msg:  "success",
		Data: &api.UpdateUserAvatorResponse_Data{
			Info: &api.UserInfo{
				UserId:   int64(u.ID),
				Name:     u.Name,
				Avatar:   u.Avatar,
				Email:    u.Email,
				Location: u.Location,
			},
		},
	}, nil
}

func (user *UserService) GetUserGroup(ctx context.Context, req *api.UserGroupRequest) (*api.UserGroupResponse, error) {
	list, err := models.GetUserGroups(int(req.GetUserId()), int(req.GetOffset()), int(req.GetPageSize()))
	if err != nil {
		return nil, err
	}
	if len(list) == 0 {
		return &api.UserGroupResponse{}, nil
	}
	var groups = make([]*api.GroupInfo, len(list), len(list))
	var u = new(models.User)
	u.ID = uint(req.GetUserId())
	err = u.GetById()
	if err != nil {
		log.Errorf("get user failed : %s", err.Error())
		return nil, err
	}
	info := &api.UserInfo{
		UserId: int64(u.ID),
	}
	groupIds := make([]int64, 0)
	for _, group := range list {
		groupIds = append(groupIds, int64(group.ID))
	}
	profiles, err := models.GetGroupProfiles(ctx, groupIds)
	if err != nil {
		log.Errorf("get group profiles failed : %s", err.Error())
		return nil, err
	}
	groupProfileMap := make(map[int64]*models.GroupProfile)
	for _, profile := range profiles {
		groupProfileMap[profile.GroupID] = profile
	}
	groupProfileMapData, _ := json.Marshal(groupProfileMap)
	log.Infof("groupProfileMap: %s", string(groupProfileMapData))
	for idx, _ := range list {
		groups[idx] = &api.GroupInfo{}
		groups[idx].Avatar = list[idx].Avatar
		groups[idx].Name = list[idx].Name
		groups[idx].GroupId = int64(list[idx].ID)
		groups[idx].Desc = list[idx].ShortDesc
		groups[idx].Owner = info.UserId
		groups[idx].Creator = info.UserId
		if groupProfileMap[int64(list[idx].ID)] != nil {
			groups[idx].Profile = &api.GroupProfileInfo{
				GroupId:          int64(list[idx].ID),
				GroupMemberNum:   int32(groupProfileMap[int64(list[idx].ID)].Members),
				GroupFollowerNum: int32(groupProfileMap[int64(list[idx].ID)].Followers),
				GroupStoryNum:    int32(groupProfileMap[int64(list[idx].ID)].StoryCount),
				Description:      groupProfileMap[int64(list[idx].ID)].Desc,
				BackgroudUrl:     groupProfileMap[int64(list[idx].ID)].BackgroundUrl,
			}
		}
		cu, err := user.GetGroupCurrentUserStatus(ctx, int64(list[idx].ID))
		if err != nil {
			return nil, err
		}
		groups[idx].CurrentUserStatus = cu

		groups[idx].Ctime = list[idx].CreateAt.Unix()
		groups[idx].Mtime = list[idx].UpdateAt.Unix()
	}
	return &api.UserGroupResponse{
		Code: 0,
		Msg:  "success",
		Data: &api.UserGroupResponse_Data{
			List: groups,
		},
	}, nil
}
func (user *UserService) GetUserFollowingGroup(ctx context.Context, req *api.UserFollowingGroupRequest) (
	*api.UserFollowingGroupResponse, error) {
	list, err := models.GetUserJoinedGroups(int(req.GetUserId()), 0, 10)
	if err != nil {
		return nil, err
	}
	var groups = make([]*api.GroupInfo, 0, len(list))
	var u = new(models.User)
	u.ID = uint(req.GetUserId())
	err = u.GetById()
	if err != nil {
		log.Errorf("get user failed : %s", err.Error())
		return nil, err
	}
	info := &api.UserInfo{
		UserId:   int64(u.ID),
		Name:     u.Name,
		Avatar:   u.Avatar,
		Email:    u.Email,
		Location: u.Location,
	}
	for idx, _ := range list {
		groups[idx] = &api.GroupInfo{}
		groups[idx].Avatar = list[idx].Avatar
		groups[idx].Name = list[idx].Name
		groups[idx].GroupId = int64(list[idx].ID)
		groups[idx].Desc = list[idx].ShortDesc
		groups[idx].Owner = info.GetUserId()
		groups[idx].Creator = info.GetUserId()
	}
	return &api.UserFollowingGroupResponse{
		Code: 0,
		Msg:  "success",
		Data: &api.UserFollowingGroupResponse_Data{
			List: groups,
		},
	}, nil
}

func (user *UserService) UpdateUser(ctx context.Context, req *api.UserUpdateRequest) (
	*api.UserUpdateResponse, error) {
	u := &models.User{
		Avatar:    req.GetAvatar(),
		Name:      req.GetNickname(),
		ShortDesc: req.GetDesc(),
	}
	err := u.UpdateAvatar()
	if err != nil {
		return nil, err
	}
	return &api.UserUpdateResponse{}, nil
}

func (user *UserService) FetchActives(ctx context.Context, req *api.FetchActivesRequest) (
	*api.FetchActivesResponse, error) {
	// TODO: fetch user actives
	log.Println("FetchActives req: ", req.String())
	if req.GetUserId() <= 0 {
		return nil, fmt.Errorf("invalid user id")
	}
	if req.GetAtype() > api.ActiveFlowType_GroupFlowType || req.GetAtype() < api.ActiveFlowType_AllFlowType {
		return nil, fmt.Errorf("invalid active type %d", req.GetAtype())
	}
	var (
		groupIds, storyIds, roleIds []int64
		err                         error
		groupMap                    = make(map[int64]*models.Group)
		storyMap                    = make(map[int64]*models.Story)
		roleMap                     = make(map[int64]*models.StoryRole)
		lasttimeStamp               = req.GetTimestamp()
	)
	if req.GetAtype() == api.ActiveFlowType_GroupFlowType {
		groupIds, _, err = models.GetUserFollowedGroupIds(ctx, int(req.GetUserId()))
		if err != nil {
			log.Errorf("get user [%d] followed group ids failed: %s", req.GetUserId(), err.Error())
			return nil, err
		}
		if len(groupIds) == 0 {
			log.Infof("user [%d] has no followed group", req.GetUserId())
			return &api.FetchActivesResponse{
				Code: 0,
				Msg:  "success",
				Data: &api.FetchActivesResponse_Data{
					List:      nil,
					Timestamp: lasttimeStamp,
					PageSize:  int64(req.GetPageSize()),
					Offset:    int64(req.GetOffset()),
				},
			}, nil
		}
	}

	if req.GetAtype() == api.ActiveFlowType_StoryFlowType {
		storyIds, err = models.GetUserFollowedStoryIds(ctx, int(req.GetUserId()))
		if err != nil {
			log.Errorf("get user [%d] followed story ids failed: %s", req.GetUserId(), err.Error())
			return nil, err
		}
		if len(storyIds) == 0 {
			log.Infof("user [%d] has no followed story", req.GetUserId())
			return &api.FetchActivesResponse{
				Code: 0,
				Msg:  "success",
				Data: &api.FetchActivesResponse_Data{
					List:      nil,
					Timestamp: lasttimeStamp,
					PageSize:  int64(req.GetPageSize()),
					Offset:    int64(req.GetOffset()),
				},
			}, nil
		}
	}

	if req.GetAtype() == api.ActiveFlowType_RoleFlowType {
		roleIds, err = models.GetUserFollowedStoryRoleIds(ctx, int(req.GetUserId()))
		if err != nil {
			log.Errorf("get user [%d] followed story role ids failed: %s", req.GetUserId(), err.Error())
			return nil, err
		}
		if len(roleIds) == 0 {
			log.Infof("user [%d] has no followed story role", req.GetUserId())
			return &api.FetchActivesResponse{
				Code: 0,
				Msg:  "success",
				Data: &api.FetchActivesResponse_Data{
					List:      nil,
					Timestamp: lasttimeStamp,
					PageSize:  int64(req.GetPageSize()),
					Offset:    int64(req.GetOffset()),
				},
			}, nil
		}
	}

	// TODO: fetch user actives
	apiActives := make([]*api.ActiveInfo, 0)
	allActives := make([]*models.Active, 0)
	if len(groupIds) != 0 {
		actives, _, err := models.GetActiveByFollowingGroupID(req.GetUserId(), groupIds, int(req.GetOffset()), int(req.GetPageSize()))
		if err != nil {
			log.Errorf("get user [%d] followed group actives failed: %s", req.GetUserId(), err.Error())
			return nil, err
		}
		if len(actives) != 0 {
			allActives = append(allActives, actives...)
		}
		targetGroupIds := make([]int64, 0)
		for _, active := range actives {
			groupMap[active.GroupId] = &models.Group{}
			targetGroupIds = append(targetGroupIds, active.GroupId)
		}
		groups, err := models.GetGroupsByIds(targetGroupIds)
		if err != nil {
			log.Errorf("get user [%d] followed group failed: %s", req.GetUserId(), err.Error())
			return nil, err
		}
		for _, group := range groups {
			groupMap[int64(group.ID)] = group
		}
	}
	if len(storyIds) != 0 {
		actives, _, err := models.GetActiveByFollowingStoryID(req.GetUserId(), storyIds, int(req.GetOffset()), int(req.GetPageSize()))
		if err != nil {
			log.Errorf("get user [%d] followed story actives failed: %s", req.GetUserId(), err.Error())
			return nil, err
		}
		if len(*actives) != 0 {
			allActives = append(allActives, *actives...)
		}
		targetStoryIds := make([]int64, 0)
		for _, active := range *actives {
			storyMap[active.StoryId] = &models.Story{}
			targetStoryIds = append(targetStoryIds, active.StoryId)
		}
		stories, err := models.GetStoriesByIDs(ctx, targetStoryIds)
		if err != nil {
			log.Errorf("get user [%d] followed story failed: %s", req.GetUserId(), err.Error())
			return nil, err
		}
		for _, story := range stories {
			storyMap[int64(story.ID)] = story
		}
	}
	if len(roleIds) != 0 {
		actives, _, err := models.GetActiveByFollowingStoryRoleID(req.GetUserId(), roleIds, int(req.GetOffset()), int(req.GetPageSize()))
		if err != nil {
			log.Errorf("get user [%d] followed story role failed: %s", req.GetUserId(), err.Error())
			return nil, err
		}
		if len(*actives) != 0 {
			allActives = append(allActives, *actives...)
		}
		targetStoryroleIds := make([]int64, 0)
		for _, active := range *actives {
			roleMap[active.StoryRoleId] = &models.StoryRole{}
			targetStoryroleIds = append(targetStoryroleIds, active.StoryRoleId)
		}
		roles, err := models.GetStoryRolesByIDs(ctx, targetStoryroleIds)
		if err != nil {
			log.Errorf("get user [%d] followed story role failed: %s", req.GetUserId(), err.Error())
			return nil, err
		}
		for _, role := range roles {
			roleMap[int64(role.ID)] = role
		}
	}
	activeUsers := make(map[int64]*models.User)
	userIds := make([]int64, 0)
	for _, active := range allActives {
		activeUsers[active.UserId] = &models.User{}
		userIds = append(userIds, active.UserId)
	}
	users, err := models.GetUsersByIds(userIds)
	if err != nil {
		log.Errorf("get user [%d] followed story role failed: %s", req.GetUserId(), err.Error())
		return nil, err
	}
	if len(users) != 0 {
		for _, user := range users {
			activeUsers[int64(user.ID)] = user
		}
	} else {
		log.Infof("user [%d] has no followed story role", req.GetUserId())
		return &api.FetchActivesResponse{
			Code: api.ResponseCode_USER_PROFILE_INCOMPLETE,
			Msg:  "failed",
		}, nil
	}
	sort.Sort(models.ActiveList(allActives))
	for _, active := range allActives {
		apiActive := &api.ActiveInfo{}
		apiActive.ActiveId = int64(active.ID)
		if req.GetAtype() == api.ActiveFlowType_GroupFlowType {
			apiActive.ActiveType = api.ActiveType_FollowGroup
			apiActive.GroupInfo = &api.GroupInfo{
				GroupId: active.GroupId,
				Name:    groupMap[active.GroupId].Name,
				Avatar:  groupMap[active.GroupId].Avatar,
				Desc:    groupMap[active.GroupId].ShortDesc,
				Creator: groupMap[active.GroupId].CreatorID,
				Owner:   groupMap[active.GroupId].OwnerID,
				Ctime:   groupMap[active.GroupId].CreateAt.Unix(),
				Mtime:   groupMap[active.GroupId].UpdateAt.Unix(),
			}
			apiActive.User = &api.UserInfo{
				UserId:   int64(activeUsers[active.UserId].ID),
				Name:     activeUsers[active.UserId].Name,
				Avatar:   activeUsers[active.UserId].Avatar,
				Email:    activeUsers[active.UserId].Email,
				Location: activeUsers[active.UserId].Location,
			}
		}
		if req.GetAtype() == api.ActiveFlowType_StoryFlowType {
			apiActive.ActiveType = api.ActiveType_FollowStory
			apiActive.StoryInfo = &api.Story{
				Id:     int64(active.StoryId),
				Name:   storyMap[active.StoryId].Name,
				Avatar: storyMap[active.StoryId].Avatar,
				Desc:   storyMap[active.StoryId].ShortDesc,
				Ctime:  storyMap[active.StoryId].CreateAt.Unix(),
				Mtime:  storyMap[active.StoryId].UpdateAt.Unix(),
			}
			apiActive.User = &api.UserInfo{
				UserId:   int64(activeUsers[active.UserId].ID),
				Name:     activeUsers[active.UserId].Name,
				Avatar:   activeUsers[active.UserId].Avatar,
				Email:    activeUsers[active.UserId].Email,
				Location: activeUsers[active.UserId].Location,
			}
		}
		if req.GetAtype() == api.ActiveFlowType_RoleFlowType {
			apiActive.ActiveType = api.ActiveType_FollowRole
			apiActive.RoleInfo = &api.StoryRole{
				RoleId:               active.StoryRoleId,
				CharacterName:        roleMap[active.StoryRoleId].CharacterName,
				CharacterAvatar:      roleMap[active.StoryRoleId].CharacterAvatar,
				CharacterDescription: roleMap[active.StoryRoleId].CharacterDescription,
				CharacterPrompt:      roleMap[active.StoryRoleId].CharacterPrompt,
				Ctime:                roleMap[active.StoryRoleId].CreateAt.Unix(),
				Mtime:                int64(roleMap[active.StoryRoleId].UpdateAt.Unix()),
				LikeCount:            roleMap[active.StoryRoleId].LikeCount,
				FollowCount:          roleMap[active.StoryRoleId].FollowCount,
				StoryboardNum:        roleMap[active.StoryRoleId].StoryboardNum,
			}
			apiActive.User = &api.UserInfo{
				UserId:   int64(activeUsers[active.UserId].ID),
				Name:     activeUsers[active.UserId].Name,
				Avatar:   activeUsers[active.UserId].Avatar,
				Email:    activeUsers[active.UserId].Email,
				Location: activeUsers[active.UserId].Location,
			}
		}
		apiActives = append(apiActives, apiActive)
		if lasttimeStamp > active.CreateAt.Unix() {
			lasttimeStamp = active.CreateAt.Unix()
		}
	}

	return &api.FetchActivesResponse{
		Code: 0,
		Msg:  "success",
		Data: &api.FetchActivesResponse_Data{
			List:      apiActives,
			Timestamp: lasttimeStamp,
			PageSize:  int64(req.GetPageSize()),
			Offset:    int64(req.GetOffset()),
		},
	}, nil
}

// 组织内搜索指定用户
func (user *UserService) SearchUser(ctx context.Context, req *api.SearchUserRequest) (
	*api.SearchUserResponse, error) {
	return nil, nil
}

func (user *UserService) UserWatching(ctx context.Context, req *api.UserWatchingRequest) (
	*api.UserWatchingResponse, error) {
	list, err := models.GetUserWatchingProjects(int64(req.GetUserId()), 0, 10)
	if err != nil {
		log.Errorf("get user [%d] watching projects failed: %s", req.GetUserId(), err.Error())
		return nil, err
	}
	var projects = make([]*api.ProjectInfo, 0, len(list))
	var u = new(models.User)
	u.ID = uint(req.GetUserId())
	err = u.GetById()
	if err != nil {
		log.Errorf("get user failed : %s", err.Error())
		return nil, err
	}
	info := &api.UserInfo{
		UserId:   int64(u.ID),
		Name:     u.Name,
		Avatar:   u.Avatar,
		Email:    u.Email,
		Location: u.Location,
	}
	for idx, _ := range list {
		projects[idx] = &api.ProjectInfo{}
		projects[idx].Avatar = list[idx].Avatar
		projects[idx].Name = list[idx].Name
		projects[idx].ProjectId = uint64(list[idx].ID)
		projects[idx].Owner = info.GetUserId()
		projects[idx].Creator = info.GetUserId()
	}
	return &api.UserWatchingResponse{
		Code: 0,
		Msg:  "success",
		Data: &api.UserWatchingResponse_Data{
			List: projects,
		},
	}, nil
}

func (user *UserService) GetUserProfile(ctx context.Context, req *api.GetUserProfileRequest) (
	*api.GetUserProfileResponse, error) {
	profile := &models.UserProfile{
		UserId: req.GetUserId(),
	}
	err := profile.GetByUserId()
	if err != nil {
		log.Errorf("get user profile failed : %s", err.Error())
		return nil, err
	}
	return &api.GetUserProfileResponse{
		Code:    0,
		Message: "success",
		Info:    convertModelUserProfileToApi(profile),
	}, nil
}

func (user *UserService) UpdateUserProfile(ctx context.Context, req *api.UpdateUserProfileRequest) (
	*api.UpdateUserProfileResponse, error) {
	profile := &models.UserProfile{
		UserId: req.GetUserId(),
	}
	err := profile.GetByUserId()
	if err != nil {
		log.Errorf("get user profile failed : %s", err.Error())
		return nil, err
	}
	if req.GetBackgroundImage() != "" {
		profile.UserId = req.GetUserId()
		profile.Background = req.GetBackgroundImage()
		err = profile.Update()
		if err != nil {
			log.Errorf("update user profile backgroud image failed : %s", err.Error())
			return nil, err
		}
	}
	needUpdates := make(map[string]interface{}, 0)
	if req.GetAvatar() != "" {
		needUpdates["avatar"] = req.GetAvatar()
	}
	if req.GetName() != "" {
		needUpdates["name"] = req.GetName()
	}
	if req.GetLocation() != "" {
		needUpdates["location"] = req.GetLocation()
	}
	if req.GetEmail() != "" {
		needUpdates["email"] = req.GetEmail()
	}
	if req.GetDescription() != "" {
		needUpdates["description"] = req.GetDescription()
	}

	err = models.UpdateUserInfo(ctx, req.GetUserId(), needUpdates)
	if err != nil {
		return &api.UpdateUserProfileResponse{
			Code:    -1,
			Message: "user info err:" + err.Error(),
		}, nil
	}

	return &api.UpdateUserProfileResponse{
		Code:    0,
		Message: "success",
	}, nil
}

func (user *UserService) UpdateUserBackgroundImage(ctx context.Context, req *api.UpdateUserBackgroundImageRequest) (*api.UpdateUserBackgroundImageResponse, error) {
	profile := &models.UserProfile{
		UserId: req.GetUserId(),
	}
	err := profile.GetByUserId()
	if err != nil {
		log.Errorf("get user profile failed : %s", err.Error())
		return nil, err
	}
	profile.Background = req.GetBackgroundImage()
	err = profile.Update()
	if err != nil {
		return &api.UpdateUserBackgroundImageResponse{
			Code:    -1,
			Message: "update user background image failed : " + err.Error(),
		}, nil
	}
	return &api.UpdateUserBackgroundImageResponse{
		Code:    0,
		Message: "OK",
	}, nil
}

func (user *UserService) GetGroupCurrentUserStatus(ctx context.Context, groupId int64) (*api.WhatCurrentUserStatus, error) {
	// 查询用户ID
	if groupId == 0 {
		return nil, nil
	}
	userID, err := utils.GetUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}
	cu := new(api.WhatCurrentUserStatus)
	// 查询用户是否关注了小组
	follow, err := models.GetWatchItemByGroupAndUser(ctx, groupId, int64(userID))
	if err != nil {
		return nil, err
	}
	if follow != nil && follow.Deleted == false {
		cu.IsFollowed = true
	}
	// 查询用户是否加入了小组
	join, err := models.GetGroupMemberByGroupAndUser(ctx, groupId, userID)
	if err != nil {
		return nil, err
	}
	if join != nil && join.Deleted == false {
		cu.IsJoined = true
	}
	return cu, nil
}

func (user *UserService) GetStoryRoleCurrentUserStatus(ctx context.Context, roleId int64) (*api.WhatCurrentUserStatus, error) {
	// 查询用户ID
	if roleId == 0 {
		return nil, nil
	}
	userID, err := utils.GetUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}
	cu := new(api.WhatCurrentUserStatus)
	// 查询用户是否关注了角色
	follow, err := models.GetWatchItemByStoryRoleAndUser(ctx, roleId, int64(userID))
	if err != nil {
		return nil, err
	}
	if follow != nil && follow.Deleted == false {
		cu.IsFollowed = true
	}
	// 查询用户是否点赞了角色
	like, err := models.GetLikeItemByStoryRoleAndUser(ctx, roleId, int(userID))
	if err != nil {
		return nil, err
	}
	if like != nil && like.Deleted == false {
		cu.IsLiked = true
	}
	return cu, nil
}

func (user *UserService) GetStoryCurrentUserStatus(ctx context.Context, storyId int64) (*api.WhatCurrentUserStatus, error) {
	// 查询用户ID
	if storyId == 0 {
		return nil, nil
	}
	userID, err := utils.GetUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}
	cu := new(api.WhatCurrentUserStatus)
	// 查询用户是否关注了角色
	follow, err := models.GetWatchItemByStoryAndUser(ctx, storyId, int(userID))
	if err != nil {
		return nil, err
	}
	if follow != nil && follow.Deleted == false {
		cu.IsFollowed = true
	}
	// 查询用户是否点赞了角色
	like, err := models.GetLikeItemByStoryAndUser(ctx, storyId, int(userID))
	if err != nil {
		return nil, err
	}
	if like != nil && like.Deleted == false {
		cu.IsLiked = true
	}
	return cu, nil
}

func (user *UserService) GetStoryboardCurrentUserStatus(ctx context.Context, storyboardId int64) (*api.WhatCurrentUserStatus, error) {
	// 查询用户ID
	if storyboardId == 0 {
		return nil, nil
	}
	userID, err := utils.GetUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}
	cu := new(api.WhatCurrentUserStatus)
	// 查询用户是否点赞了角色
	like, err := models.GetLikeItemByStoryBoardAndUser(ctx, storyboardId, int(userID))
	if err != nil {
		return nil, err
	}
	if like != nil && like.Deleted == false {
		cu.IsLiked = true
	}
	return cu, nil
}

func convertApiUserProfileInfoToModel(info *api.UserProfileInfo) *models.UserProfile {
	return &models.UserProfile{
		UserId:            info.UserId,
		CreatedGroupNum:   int(info.CreatedGroupNum),
		CreatedStoryNum:   int(info.CreatedStoryNum),
		CreatedRoleNum:    int(info.CreatedRoleNum),
		WatchingStoryNum:  int(info.WatchingStoryNum),
		ContributStoryNum: int(info.ContributStoryNum),
		ContributRoleNum:  int(info.ContributRoleNum),
		NumGroup:          int(info.NumGroup),
		DefaultGroupID:    int64(info.DefaultGroupId),
		MinSameGroup:      int(info.MinSameGroup),
		Limit:             int(info.Limit),
		UsedTokens:        int(info.UsedTokens),
		Status:            int(info.Status),
		IDBase: models.IDBase{
			Base: models.Base{
				CreateAt: time.Unix(info.Ctime, 0),
				UpdateAt: time.Unix(info.Mtime, 0),
			},
		},
	}
}

func convertModelUserProfileToApi(profile *models.UserProfile) *api.UserProfileInfo {
	return &api.UserProfileInfo{
		UserId:            profile.UserId,
		CreatedGroupNum:   int32(profile.CreatedGroupNum),
		CreatedStoryNum:   int32(profile.CreatedStoryNum),
		CreatedRoleNum:    int32(profile.CreatedRoleNum),
		WatchingStoryNum:  int32(profile.WatchingStoryNum),
		ContributStoryNum: int32(profile.ContributStoryNum),
		ContributRoleNum:  int32(profile.ContributRoleNum),
		NumGroup:          int32(profile.NumGroup),
		DefaultGroupId:    int64(profile.DefaultGroupID),
		MinSameGroup:      int32(profile.MinSameGroup),
		Limit:             int32(profile.Limit),
		UsedTokens:        int32(profile.UsedTokens),
		Status:            int32(profile.Status),
		BackgroundImage:   profile.Background,
		Ctime:             profile.CreateAt.Unix(),
		Mtime:             profile.UpdateAt.Unix(),
	}
}
