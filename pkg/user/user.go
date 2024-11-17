package user

import (
	"context"
	"fmt"
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
	FetchUserActives(ctx context.Context, req *api.FetchUserActivesRequest) (*api.FetchUserActivesResponse, error)
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
		Info: &api.UserInfo{
			UserId:   int64(u.ID),
			Name:     u.Name,
			Avatar:   u.Avatar,
			Email:    u.Email,
			Location: u.Location,
		},
	}, err
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
		Info: &api.UserInfo{
			UserId:   int64(u.ID),
			Name:     u.Name,
			Avatar:   u.Avatar,
			Email:    u.Email,
			Location: u.Location,
		},
	}, err
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
		List: groups,
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
		List: groups,
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
func (user *UserService) FetchUserActives(ctx context.Context, req *api.FetchUserActivesRequest) (
	*api.FetchUserActivesResponse, error) {
	// TODO: fetch user actives
	actives := make([]*api.ActiveInfo, 0)
	data, err := models.GetActiveByUserID(req.GetUserId())
	if err != nil {
		return nil, err
	}
	for _, active := range *data {
		actives = append(actives, &api.ActiveInfo{
			ItemInfo: &api.ItemInfo{},
			User: &api.UserInfo{
				UserId: active.UserId,
			},
		})
	}
	return &api.FetchUserActivesResponse{
		List: actives,
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
		List: projects,
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
		Info: convertModelUserProfileToApi(profile),
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
