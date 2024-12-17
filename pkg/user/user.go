package user

import (
	"context"
	"fmt"
	"sort"
	"time"

	log "github.com/sirupsen/logrus"

	api "github.com/grapery/common-protoc/gen"
	"github.com/grapery/grapery/models"
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
			},
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
	list, err := models.GetUserGroups(int(req.GetUserId()), 0, 10)
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
	for idx, _ := range list {
		groups[idx] = &api.GroupInfo{}
		groups[idx].Avatar = list[idx].Avatar
		groups[idx].Name = list[idx].Name
		groups[idx].GroupId = int64(list[idx].ID)
		groups[idx].Desc = list[idx].ShortDesc
		groups[idx].Owner = info.UserId
		groups[idx].Creator = info.UserId
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
	if req.GetUserId() <= 0 {
		return nil, fmt.Errorf("invalid user id")
	}
	if req.GetTimestamp() <= 0 {
		return nil, fmt.Errorf("invalid timestamp")
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
			return nil, err
		}
	}

	if req.GetAtype() == api.ActiveFlowType_StoryFlowType {
		storyIds, err = models.GetUserFollowedStoryIds(ctx, int(req.GetUserId()))
		if err != nil {
			return nil, err
		}
	}

	if req.GetAtype() == api.ActiveFlowType_RoleFlowType {
		roleIds, err = models.GetUserFollowedStoryRoleIds(ctx, int(req.GetUserId()))
		if err != nil {
			return nil, err
		}
	}

	// TODO: fetch user actives
	apiActives := make([]*api.ActiveInfo, 0)
	allActives := make([]*models.Active, 0)
	if len(groupIds) != 0 {
		actives, _, err := models.GetActiveByFollowingGroupID(req.GetUserId(), groupIds, int(req.GetOffset()), int(req.GetPageSize()))
		if err != nil {
			return nil, err
		}
		if len(*actives) != 0 {
			allActives = append(allActives, *actives...)
		}
		targetGroupIds := make([]int64, 0)
		for _, active := range *actives {
			groupMap[active.GroupId] = &models.Group{}
			targetGroupIds = append(targetGroupIds, active.GroupId)
		}
		groups, err := models.GetGroupsByIds(targetGroupIds)
		if err != nil {
			return nil, err
		}
		for _, group := range groups {
			groupMap[int64(group.ID)] = group
		}
	}
	if len(storyIds) != 0 {
		actives, _, err := models.GetActiveByFollowingStoryID(req.GetUserId(), storyIds, int(req.GetOffset()), int(req.GetPageSize()))
		if err != nil {
			return nil, err
		}
		if len(*actives) != 0 {
			allActives = append(allActives, *actives...)
		}
		targetStoryIds := make([]int64, 0)
		for _, active := range *actives {
			groupMap[active.GroupId] = &models.Group{}
			targetStoryIds = append(targetStoryIds, active.GroupId)
		}
		stories, err := models.GetStoriesByIDs(ctx, targetStoryIds)
		if err != nil {
			return nil, err
		}
		for _, story := range stories {
			storyMap[int64(story.ID)] = story
		}
	}
	if len(roleIds) != 0 {
		actives, _, err := models.GetActiveByFollowingStoryRoleID(req.GetUserId(), roleIds, int(req.GetOffset()), int(req.GetPageSize()))
		if err != nil {
			return nil, err
		}
		if len(*actives) != 0 {
			allActives = append(allActives, *actives...)
		}
		targetStoryroleIds := make([]int64, 0)
		for _, active := range *actives {
			roleMap[active.GroupId] = &models.StoryRole{}
			targetStoryroleIds = append(targetStoryroleIds, active.GroupId)
		}
		roles, err := models.GetStoryRolesByIDs(ctx, targetStoryroleIds)
		if err != nil {
			return nil, err
		}
		for _, role := range roles {
			roleMap[int64(role.ID)] = role
		}
	}
	sort.Sort(models.ActiveList(allActives))
	for _, active := range allActives {
		apiActive := &api.ActiveInfo{}
		if req.GetAtype() == api.ActiveFlowType_GroupFlowType {
			apiActive.ActiveType = api.ActiveType_FollowGroup
			apiActive.GroupInfo = &api.GroupInfo{
				GroupId: active.GroupId,
				Name:    groupMap[active.GroupId].Name,
				Avatar:  groupMap[active.GroupId].Avatar,
				Desc:    groupMap[active.GroupId].ShortDesc,
				Creator: groupMap[active.GroupId].CreatorID,
				Owner:   groupMap[active.GroupId].OwnerID,
			}
		}
		if req.GetAtype() == api.ActiveFlowType_StoryFlowType {
			apiActive.ActiveType = api.ActiveType_FollowStory
			apiActive.StoryInfo = &api.Story{
				Id:     active.StoryId,
				Name:   storyMap[active.StoryId].Name,
				Avatar: storyMap[active.StoryId].Avatar,
				Desc:   storyMap[active.StoryId].ShortDesc,
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
	profile = convertApiUserProfileInfoToModel(req.GetInfo())
	profile.UserId = req.GetUserId()
	err = profile.Update()
	if err != nil {
		log.Errorf("update user profile failed : %s", err.Error())
		return nil, err
	}
	return &api.UpdateUserProfileResponse{
		Code:    0,
		Message: "success",
	}, nil
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
		Ctime:             profile.CreateAt.Unix(),
		Mtime:             profile.UpdateAt.Unix(),
	}
}
