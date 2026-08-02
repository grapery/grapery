package user

import (
	"context"
	"encoding/json"
	"sort"
	"time"

	"go.uber.org/zap"

	api "github.com/grapery/common-protoc/gen"
	"github.com/grapery/grapery/models"
	"github.com/grapery/grapery/pkg/cloud/aliyun"
	llmchatservice "github.com/grapery/grapery/service/llmchat"
	"github.com/grapery/grapery/utils"
	"github.com/grapery/grapery/utils/cache"
	"github.com/grapery/grapery/utils/errors"
)

var (
	logger, _  = zap.NewDevelopment()
	userServer UserServer
)

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

	FollowUser(ctx context.Context, req *api.FollowUserRequest) (*api.FollowUserResponse, error)
	UnfollowUser(ctx context.Context, req *api.UnfollowUserRequest) (*api.UnfollowUserResponse, error)
	GetFollowList(ctx context.Context, req *api.GetFollowListRequest) (*api.GetFollowListResponse, error)
	GetFollowerList(ctx context.Context, req *api.GetFollowerListRequest) (*api.GetFollowerListResponse, error)

	FetchUserGenTaskStatus(ctx context.Context, req *api.FetchUserGenTaskStatusRequest) (*api.FetchUserGenTaskStatusResponse, error)
}

type UserService struct {
}

func (user *UserService) UserInit(ctx context.Context, req *api.UserInitRequest) (*api.UserInitResponse, error) {
	traceId := utils.GetTraceID(ctx)
	logger.Info("UserInit called", zap.String("traceId", traceId), zap.Any("req", req))
	defer func() {
		if re := recover(); re != nil {
			logger.Error("UserInit panic", zap.String("traceId", traceId), zap.Any("recover", re))
		}
	}()
	userProfile := &models.UserProfile{
		UserId: int64(req.GetUserId()),
	}
	err := userProfile.GetByUserId(ctx)
	if err != nil {
		logger.Error("UserInit failed: get user profile error", zap.String("traceId", traceId), zap.Error(err))
		return nil, err
	}
	if userProfile.ID == 0 {
		userProfile.IDBase = models.IDBase{
			Base: models.Base{
				CreateAt: time.Now(),
				UpdateAt: time.Now(),
			},
		}
		userProfile.Background = ""
		userProfile.NumGroup = 0
		userProfile.DefaultGroupID = 0
		userProfile.MinSameGroup = 0
		userProfile.Limit = 0
		userProfile.UsedTokens = 0
		userProfile.CreatedGroupNum = 0
		userProfile.UserId = int64(req.GetUserId())
		err = userProfile.Create(ctx)
		if err != nil {
			logger.Error("UserInit failed: create user profile error", zap.String("traceId", traceId), zap.Error(err))
			return nil, err
		}
	}
	defaultGroup, ok, err := models.GetUserDefaultGroup(ctx, int(req.GetUserId()))
	if err != nil {
		logger.Error("UserInit failed: get user default group error", zap.String("traceId", traceId), zap.Error(err))
		return nil, err
	}
	if ok {
		resp := &api.UserInitResponse{
			Code: api.ResponseCode_OK,
			Msg:  "success",
			Data: &api.UserInitResponse_Data{
				UserId: req.GetUserId(),
				List: []*api.GroupInfo{
					{
						GroupId: int64(defaultGroup.ID),
						Name:    utils.MaskContent(defaultGroup.Name),
						Avatar:  defaultGroup.Avatar,
						Desc:    utils.MaskContent(defaultGroup.Gtype),
						Creator: req.GetUserId(),
						Ctime:   defaultGroup.CreateAt.Unix(),
						Mtime:   defaultGroup.UpdateAt.Unix(),
					},
				},
			},
		}
		logger.Info("UserInit success (default group exists)", zap.String("traceId", traceId), zap.Any("resp", resp))
		return resp, nil
	}
	// user default group is not exist,need create one
	if !ok {
		defaultGroup, ok, err = models.GetUserDefaultGroup(ctx, int(req.GetUserId()))
		if !ok {
			logger.Error("UserInit failed: create default group failed", zap.String("traceId", traceId), zap.Error(err))
			return nil, errors.ErrCreateDefaultGroupFailed
		}
	}
	resp := &api.UserInitResponse{
		Code: api.ResponseCode_OK,
		Msg:  "success",
		Data: &api.UserInitResponse_Data{
			UserId: req.GetUserId(),
			List: []*api.GroupInfo{
				{
					GroupId: int64(defaultGroup.ID),
					Name:    utils.MaskContent(defaultGroup.Name),
					Avatar:  defaultGroup.Avatar,
					Desc:    utils.MaskContent(defaultGroup.Gtype),
					Creator: req.GetUserId(),
					Ctime:   defaultGroup.CreateAt.Unix(),
					Mtime:   defaultGroup.UpdateAt.Unix(),
				},
			},
		},
	}
	logger.Info("UserInit success (default group created)", zap.String("traceId", traceId), zap.Any("resp", resp))
	return resp, nil
}

func (user *UserService) GetUserInfo(ctx context.Context, req *api.UserInfoRequest) (*api.UserInfoResponse, error) {
	traceId := utils.GetTraceID(ctx)
	logger.Info("GetUserInfo called", zap.String("traceId", traceId), zap.Any("req", req))
	var u = new(models.User)
	u.ID = uint(req.GetUserId())
	err := u.GetById(ctx)
	if err != nil {
		logger.Error("GetUserInfo failed: get user by id error", zap.String("traceId", traceId), zap.Error(err))
		return nil, err
	}
	userProfile := &models.UserProfile{
		UserId: int64(u.ID),
	}
	err = userProfile.GetByUserId(ctx)
	if err != nil {
		logger.Error("GetUserInfo failed: get user profile error", zap.String("traceId", traceId), zap.Error(err))
		return nil, err
	}
	resp := &api.UserInfoResponse{
		Code: api.ResponseCode_OK,
		Msg:  "success",
		Data: &api.UserInfoResponse_Data{
			Info: &api.UserInfo{
				UserId:   int64(u.ID),
				Name:     utils.MaskContent(u.Name),
				Avatar:   u.Avatar,
				Email:    utils.MaskContent(u.Email),
				Location: utils.MaskContent(u.Location),
				Desc:     utils.MaskContent(u.ShortDesc),
				Ctime:    u.CreateAt.Unix(),
				Mtime:    u.UpdateAt.Unix(),
			},
			Profile: convertModelUserProfileToApi(userProfile),
		},
	}
	logger.Info("GetUserInfo success", zap.String("traceId", traceId), zap.Any("resp", resp))
	return resp, nil
}

func (user *UserService) UpdateAvator(ctx context.Context, req *api.UpdateUserAvatorRequest) (*api.UpdateUserAvatorResponse, error) {
	traceId := utils.GetTraceID(ctx)
	logger.Info("UpdateAvator called", zap.String("traceId", traceId), zap.Any("req", req))
	var u = new(models.User)
	u.ID = uint(req.GetUserId())
	u.Avatar = req.GetAvatar()
	err := u.UpdateAvatar(ctx)
	if err != nil {
		logger.Error("UpdateAvator failed: update avatar error", zap.String("traceId", traceId), zap.Error(err))
		return nil, err
	}
	aliyunClient := aliyun.GetGlobalClient()
	err = aliyunClient.PersistImages(u.Avatar)
	if err != nil {
		logger.Error("UpdateAvator failed: persist images error", zap.String("traceId", traceId), zap.Error(err))
		return nil, err
	}
	resp := &api.UpdateUserAvatorResponse{
		Code: api.ResponseCode_OK,
		Msg:  "success",
		Data: &api.UpdateUserAvatorResponse_Data{
			Info: &api.UserInfo{
				UserId:   int64(u.ID),
				Name:     utils.MaskContent(u.Name),
				Avatar:   u.Avatar,
				Email:    utils.MaskContent(u.Email),
				Location: utils.MaskContent(u.Location),
			},
		},
	}
	logger.Info("UpdateAvator success", zap.String("traceId", traceId), zap.Any("resp", resp))
	return resp, nil
}

func (user *UserService) GetUserGroup(ctx context.Context, req *api.UserGroupRequest) (*api.UserGroupResponse, error) {
	traceId := utils.GetTraceID(ctx)
	logger.Info("GetUserGroup called", zap.String("traceId", traceId), zap.Any("req", req))
	list, total, _, err := models.GetUserGroups(ctx, int(req.GetUserId()), int(req.GetOffset()), int(req.GetPageSize()))
	if err != nil {
		logger.Error("GetUserGroup failed: get user groups error", zap.String("traceId", traceId), zap.Error(err))
		return nil, err
	}
	if len(list) == 0 {
		logger.Info("GetUserGroup success: no groups", zap.String("traceId", traceId), zap.Int64("user_id", req.GetUserId()))
		return &api.UserGroupResponse{
			Code: api.ResponseCode_OK,
			Msg:  "success",
			Data: &api.UserGroupResponse_Data{
				List:     []*api.GroupInfo{},
				Offset:   req.GetOffset(),
				PageSize: req.GetPageSize(),
				Total:    int32(total),
				HaveMore: false,
			},
		}, nil
	}
	var groups = make([]*api.GroupInfo, len(list), len(list))
	var u = new(models.User)
	u.ID = uint(req.GetUserId())
	err = u.GetById(ctx)
	if err != nil {
		logger.Error("GetUserGroup failed: get user by id error", zap.String("traceId", traceId), zap.Error(err))
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
		logger.Error("GetUserGroup failed: get group profiles error", zap.String("traceId", traceId), zap.Error(err))
		return nil, err
	}
	groupProfileMap := make(map[int64]*models.GroupProfile)
	for _, profile := range profiles {
		groupProfileMap[profile.GroupID] = profile
	}
	groupProfileMapData, _ := json.Marshal(groupProfileMap)
	logger.Debug("GetUserGroup groupProfileMap", zap.String("traceId", traceId), zap.String("groupProfileMap", string(groupProfileMapData)))
	for idx := range list {
		groups[idx] = &api.GroupInfo{}
		groups[idx].Avatar = list[idx].Avatar
		groups[idx].Name = utils.MaskContent(list[idx].Name)
		groups[idx].GroupId = int64(list[idx].ID)
		groups[idx].Desc = utils.MaskContent(list[idx].ShortDesc)
		groups[idx].Owner = info.UserId
		groups[idx].Creator = info.UserId
		if groupProfileMap[int64(list[idx].ID)] != nil {
			groups[idx].Profile = &api.GroupProfileInfo{
				GroupId:          int64(list[idx].ID),
				GroupMemberNum:   int32(groupProfileMap[int64(list[idx].ID)].Members),
				GroupFollowerNum: int32(groupProfileMap[int64(list[idx].ID)].Followers),
				GroupStoryNum:    int32(groupProfileMap[int64(list[idx].ID)].StoryCount),
				Description:      utils.MaskContent(groupProfileMap[int64(list[idx].ID)].Desc),
				BackgroudUrl:     groupProfileMap[int64(list[idx].ID)].BackgroundUrl,
			}
		}
		cu, err := user.GetGroupCurrentUserStatus(ctx, int64(list[idx].ID))
		if err != nil {
			logger.Warn("GetUserGroup: get group current user status failed", zap.String("traceId", traceId), zap.Error(err), zap.Int64("group_id", int64(list[idx].ID)))
			continue
		}
		groups[idx].CurrentUserStatus = cu
		groups[idx].Ctime = list[idx].CreateAt.Unix()
		groups[idx].Mtime = list[idx].UpdateAt.Unix()
	}
	resp := &api.UserGroupResponse{
		Code: api.ResponseCode_OK,
		Msg:  "success",
		Data: &api.UserGroupResponse_Data{
			List:     groups,
			Offset:   req.GetOffset(),
			PageSize: req.GetPageSize(),
			HaveMore: total > int64(req.GetOffset())*int64(req.GetPageSize()),
			Total:    int32(total),
		},
	}
	logger.Info("GetUserGroup success", zap.String("traceId", traceId), zap.Any("resp", resp))
	return resp, nil
}
func (user *UserService) GetUserFollowingGroup(ctx context.Context, req *api.UserFollowingGroupRequest) (
	*api.UserFollowingGroupResponse, error) {
	traceId := utils.GetTraceID(ctx)
	logger.Info("GetUserFollowingGroup called", zap.String("traceId", traceId), zap.Any("req", req))
	list, total, err := models.GetUserJoinedGroups(ctx, int(req.GetUserId()), int(req.GetOffset()), int(req.GetPageSize()))
	if err != nil {
		logger.Error("GetUserFollowingGroup failed: get user joined groups error", zap.String("traceId", traceId), zap.Error(err))
		return nil, err
	}
	if len(list) == 0 {
		logger.Info("GetUserFollowingGroup success: no joined groups", zap.String("traceId", traceId), zap.Int64("user_id", req.GetUserId()))
		return &api.UserFollowingGroupResponse{
			Code: api.ResponseCode_OK,
			Msg:  "success",
			Data: &api.UserFollowingGroupResponse_Data{
				List:     []*api.GroupInfo{},
				HaveMore: false,
			},
		}, nil
	}
	var groups = make([]*api.GroupInfo, len(list), len(list))
	var u = new(models.User)
	u.ID = uint(req.GetUserId())
	err = u.GetById(ctx)
	if err != nil {
		logger.Error("GetUserFollowingGroup failed: get user by id error", zap.String("traceId", traceId), zap.Error(err))
		return nil, err
	}
	info := &api.UserInfo{
		UserId:   int64(u.ID),
		Name:     utils.MaskContent(u.Name),
		Avatar:   u.Avatar,
		Email:    utils.MaskContent(u.Email),
		Location: utils.MaskContent(u.Location),
	}
	for idx := range list {
		groups[idx] = &api.GroupInfo{}
		groups[idx].Avatar = list[idx].Avatar
		groups[idx].Name = utils.MaskContent(list[idx].Name)
		groups[idx].GroupId = int64(list[idx].ID)
		groups[idx].Desc = utils.MaskContent(list[idx].ShortDesc)
		groups[idx].Owner = info.GetUserId()
		groups[idx].Creator = info.GetUserId()
	}
	resp := &api.UserFollowingGroupResponse{
		Code: api.ResponseCode_OK,
		Msg:  "success",
		Data: &api.UserFollowingGroupResponse_Data{
			List:     groups,
			HaveMore: total > int64(req.GetOffset())*int64(req.GetPageSize()),
			Total:    int32(total),
			Offset:   int64(req.GetOffset()),
			PageSize: int64(req.GetPageSize()),
		},
	}
	logger.Info("GetUserFollowingGroup success", zap.String("traceId", traceId), zap.Any("resp", resp))
	return resp, nil
}

func (user *UserService) UpdateUser(ctx context.Context, req *api.UserUpdateRequest) (*api.UserUpdateResponse, error) {
	traceId := utils.GetTraceID(ctx)
	logger.Info("UpdateUser called", zap.String("traceId", traceId), zap.Any("req", req))
	if req.GetUserId() <= 0 {
		logger.Error("UpdateUser failed: invalid user id", zap.String("traceId", traceId), zap.Int64("user_id", req.GetUserId()))
		return nil, errors.ErrInvalidUserID
	}
	u := &models.User{
		ID: uint(req.GetUserId()),
	}
	err := u.GetById(ctx)
	if err != nil {
		logger.Error("UpdateUser failed: get user by id error", zap.String("traceId", traceId), zap.Error(err))
		return nil, err
	}
	u.Avatar = req.GetAvatar()
	u.Name = req.GetNickname()
	u.ShortDesc = req.GetDesc()
	err = u.UpdateAll(ctx)
	if err != nil {
		logger.Error("UpdateUser failed: update user error", zap.String("traceId", traceId), zap.Error(err))
		return nil, err
	}

	resp := &api.UserUpdateResponse{
		Code: api.ResponseCode_OK,
	}
	logger.Info("UpdateUser success", zap.String("traceId", traceId), zap.Any("resp", resp))
	return resp, nil
}

func (user *UserService) FetchActives(ctx context.Context, req *api.FetchActivesRequest) (*api.FetchActivesResponse, error) {
	traceId := utils.GetTraceID(ctx)
	logger.Info("FetchActives called", zap.String("traceId", traceId), zap.Any("req", req))
	if req.GetUserId() <= 0 {
		logger.Error("FetchActives failed: invalid user id", zap.String("traceId", traceId), zap.Int64("user_id", req.GetUserId()))
		return nil, errors.ErrInvalidUserID
	}
	if req.GetAtype() > api.ActiveFlowType_GroupFlowType || req.GetAtype() < api.ActiveFlowType_AllFlowType {
		logger.Error("FetchActives failed: invalid active type", zap.String("traceId", traceId), zap.Int32("atype", int32(req.GetAtype())))
		return nil, errors.ErrInvalidActiveType
	}
	if req.GetAtype() == api.ActiveFlowType_GroupFlowType {
		return user.FetchGroupActives(ctx, req)
	}

	if req.GetAtype() == api.ActiveFlowType_StoryFlowType {
		return user.FetchUserStoryActives(ctx, req)
	}

	if req.GetAtype() == api.ActiveFlowType_AllFlowType {
		resp, err := user.FetchUserAllActives(ctx, req)
		if err != nil {
			logger.Error("FetchActives failed: fetch user all actives error", zap.String("traceId", traceId), zap.Error(err))
			return nil, err
		}
		return resp, nil
	}

	return &api.FetchActivesResponse{
		Code: api.ResponseCode_INVALID_PARAMETER,
		Msg:  api.ResponseCode_INVALID_PARAMETER.String(),
		Data: &api.FetchActivesResponse_Data{
			HaveMore: false,
			Total:    0,
			PageSize: int64(req.GetPageSize()),
			Offset:   int64(req.GetOffset()),
		},
	}, nil
}

func (user *UserService) FetchGroupActives(ctx context.Context, req *api.FetchActivesRequest) (*api.FetchActivesResponse, error) {
	var (
		hasMore  bool
		total    int
		err      error
		groupMap = make(map[int64]*models.Group)
		roleMap  = make(map[int64]*models.StoryRole)
		storyMap = make(map[int64]*models.Story)
		boardMap = make(map[int64]*models.StoryBoard)
		userMap  = make(map[int64]*models.User)

		lasttimeStamp = req.GetTimestamp()
		storiesIds    = make([]int64, 0)
		rolesIds      = make([]int64, 0)
		storyboardIds = make([]int64, 0)
		userIds       = make([]int64, 0)
	)
	var (
		apiActives = make([]*api.ActiveInfo, 0)
		allActives = make([]*models.Active, 0)
		actives    = make([]*models.Active, 0)
	)

	actives, hasMore, total, err = models.GetActiveByFollowingGroupID(req.GetUserId(),
		[]int64{req.GetGroupId()}, int(req.GetOffset()), int(req.GetPageSize()))
	if err != nil {
		logger.Error("get user followed group actives failed", zap.Int64("user_id", req.GetUserId()), zap.Error(err))
		return nil, err
	}
	if len(actives) != 0 {
		allActives = append(allActives, actives...)
	}
	targetGroupIds := make([]int64, 0)
	for _, active := range actives {
		groupMap[active.GroupId] = &models.Group{}
		storyMap[active.StoryId] = &models.Story{}
		boardMap[active.StoryBoardId] = &models.StoryBoard{}
		roleMap[active.StoryRoleId] = &models.StoryRole{}
		userMap[active.UserId] = &models.User{}

		storiesIds = append(storiesIds, active.StoryId)
		rolesIds = append(rolesIds, active.StoryRoleId)
		storyboardIds = append(storyboardIds, active.StoryBoardId)
		userIds = append(userIds, active.UserId)
	}
	targetGroupIds = append(targetGroupIds, req.GetGroupId())
	groups, err := models.GetGroupsByIds(targetGroupIds)
	if err != nil {
		logger.Error("get user followed group failed", zap.Int64("user_id", req.GetUserId()), zap.Error(err))
		return nil, err
	}
	for _, group := range groups {
		groupMap[int64(group.ID)] = group
	}
	stories, err := models.GetStoriesByIDs(ctx, storiesIds)
	if err != nil {
		logger.Error("get user followed stories failed", zap.Int64("user_id", req.GetUserId()), zap.Error(err))
		return nil, err
	}
	for _, story := range stories {
		storyMap[int64(story.ID)] = story
	}
	roles, err := models.GetStoryRolesByIDs(ctx, rolesIds)
	if err != nil {
		logger.Error("get user followed roles failed", zap.Int64("user_id", req.GetUserId()), zap.Error(err))
		return nil, err
	}
	for _, role := range roles {
		roleMap[int64(role.ID)] = role
	}
	boards, err := models.GetStoryBoardsByIds(ctx, storyboardIds)
	if err != nil {
		logger.Error("get user followed storyboards failed", zap.Int64("user_id", req.GetUserId()), zap.Error(err))
		return nil, err
	}
	for _, board := range boards {
		boardMap[int64(board.ID)] = board
	}

	users, err := models.GetUsersByIds(ctx, userIds)
	if err != nil {
		logger.Error("get user followed users failed", zap.Int64("user_id", req.GetUserId()), zap.Error(err))
		return nil, err
	}
	for _, user := range users {
		userMap[int64(user.ID)] = user
	}

	for _, active := range allActives {
		apiActive := &api.ActiveInfo{}
		apiActive.ActiveId = int64(active.ID)
		apiActive.Ctime = active.CreateAt.Unix()
		apiActive.Mtime = active.UpdateAt.Unix()
		apiActive.User = &api.UserInfo{
			UserId: int64(active.UserId),
			Name:   userMap[active.UserId].Name,
			Avatar: userMap[active.UserId].Avatar,
			Ctime:  userMap[active.UserId].CreateAt.Unix(),
			Mtime:  userMap[active.UserId].UpdateAt.Unix(),
		}
		// 根据活动类型判断业务分类，然后使用大的 switch-case 处理数据填充
		switch {
		// 故事相关活动类型
		case active.ActiveType == api.ActiveType_NewStory ||
			active.ActiveType == api.ActiveType_FollowStory ||
			active.ActiveType == api.ActiveType_LikeStory ||
			active.ActiveType == api.ActiveType_ForkStory ||
			active.ActiveType == api.ActiveType_ShareStory:
			// 填充故事信息
			if story, exists := storyMap[active.StoryId]; exists && story != nil {
				apiActive.StoryInfo = &api.Story{
					Id:     int64(active.StoryId),
					Name:   story.Name,
					Title:  story.Title,
					Avatar: story.Avatar,
					Desc:   story.ShortDesc,
					Ctime:  story.CreateAt.Unix(),
					Mtime:  story.UpdateAt.Unix(),
				}
			}
			// 内部小 switch-case 处理 ActiveType 赋值
			switch active.ActiveType {
			case api.ActiveType_NewStory:
				apiActive.ActiveType = api.ActiveType_NewStory
			case api.ActiveType_FollowStory:
				apiActive.ActiveType = api.ActiveType_FollowStory
			case api.ActiveType_LikeStory:
				apiActive.ActiveType = api.ActiveType_LikeStory
			case api.ActiveType_ForkStory:
				apiActive.ActiveType = api.ActiveType_ForkStory
			case api.ActiveType_ShareStory:
				apiActive.ActiveType = api.ActiveType_ShareStory
			}

		// 故事角色相关活动类型
		case active.ActiveType == api.ActiveType_NewRole ||
			active.ActiveType == api.ActiveType_FollowRole ||
			active.ActiveType == api.ActiveType_LikeRole ||
			active.ActiveType == api.ActiveType_ShareRole:
			// 填充角色信息
			if role, exists := roleMap[active.StoryRoleId]; exists && role != nil {
				apiActive.RoleInfo = &api.StoryRole{
					RoleId:               active.StoryRoleId,
					CharacterName:        role.CharacterName,
					CharacterAvatar:      role.CharacterAvatar,
					CharacterDescription: role.CharacterDescription,
					CharacterPrompt:      role.CharacterPrompt,
					Ctime:                role.CreateAt.Unix(),
					Mtime:                int64(role.UpdateAt.Unix()),
					LikeCount:            role.LikeCount,
					FollowCount:          role.FollowCount,
					StoryboardNum:        role.StoryboardNum,
				}
			}
			// 填充故事信息
			if story, exists := storyMap[active.StoryId]; exists && story != nil {
				apiActive.StoryInfo = &api.Story{
					Id:     int64(active.StoryId),
					Name:   story.Name,
					Avatar: story.Avatar,
					Title:  story.Title,
					Desc:   story.ShortDesc,
					Ctime:  story.CreateAt.Unix(),
					Mtime:  story.UpdateAt.Unix(),
				}
			}
			// 内部小 switch-case 处理 ActiveType 赋值
			switch active.ActiveType {
			case api.ActiveType_NewRole:
				apiActive.ActiveType = api.ActiveType_NewRole
			case api.ActiveType_FollowRole:
				apiActive.ActiveType = api.ActiveType_FollowRole
			case api.ActiveType_LikeRole:
				apiActive.ActiveType = api.ActiveType_LikeRole
			case api.ActiveType_ShareRole:
				apiActive.ActiveType = api.ActiveType_ShareRole
			}

		// 故事板相关活动类型
		case active.ActiveType == api.ActiveType_NewStoryBoard ||
			active.ActiveType == api.ActiveType_LikeStoryBoard ||
			active.ActiveType == api.ActiveType_ShareStoryBoard:
			// 填充故事板信息
			if storyboard, exists := boardMap[active.StoryBoardId]; exists && storyboard != nil {
				apiActive.BoardInfo = &api.StoryBoard{
					StoryBoardId: int64(active.StoryBoardId),
					Title:        storyboard.Title,
					Content:      storyboard.Description,
					Creator:      storyboard.CreatorID,
					StoryId:      storyboard.StoryID,
					Stage:        storyboard.Stage,
					ForkNum:      int64(storyboard.ForkNum),
					Ctime:        storyboard.CreateAt.Unix(),
					Mtime:        storyboard.UpdateAt.Unix(),
				}
			}
			// 填充故事信息
			if story, exists := storyMap[active.StoryId]; exists && story != nil {
				apiActive.StoryInfo = &api.Story{
					Id:     int64(active.StoryId),
					Name:   story.Name,
					Title:  story.Title,
					Avatar: story.Avatar,
					Desc:   story.ShortDesc,
					Ctime:  story.CreateAt.Unix(),
					Mtime:  story.UpdateAt.Unix(),
				}
			}
			// 内部小 switch-case 处理 ActiveType 赋值
			switch active.ActiveType {
			case api.ActiveType_NewStoryBoard:
				apiActive.ActiveType = api.ActiveType_NewStoryBoard
			case api.ActiveType_LikeStoryBoard:
				apiActive.ActiveType = api.ActiveType_LikeStoryBoard
			case api.ActiveType_ShareStoryBoard:
				apiActive.ActiveType = api.ActiveType_ShareStoryBoard
			}

		// 小组相关活动类型
		case active.ActiveType == api.ActiveType_JoinGroup ||
			active.ActiveType == api.ActiveType_FollowGroup ||
			active.ActiveType == api.ActiveType_LikeGroup:
			// 填充小组信息
			if group, exists := groupMap[active.GroupId]; exists && group != nil {
				apiActive.GroupInfo = &api.GroupInfo{
					GroupId: active.GroupId,
					Name:    group.Name,
					Avatar:  group.Avatar,
					Desc:    group.ShortDesc,
					Creator: group.CreatorID,
					Owner:   group.OwnerID,
					Ctime:   group.CreateAt.Unix(),
					Mtime:   group.UpdateAt.Unix(),
				}
			}
			// 内部小 switch-case 处理 ActiveType 赋值
			switch active.ActiveType {
			case api.ActiveType_JoinGroup:
				apiActive.ActiveType = api.ActiveType_JoinGroup
			case api.ActiveType_FollowGroup:
				apiActive.ActiveType = api.ActiveType_FollowGroup
			case api.ActiveType_LikeGroup:
				apiActive.ActiveType = api.ActiveType_LikeGroup
			}

		// 默认情况：不公开互动或其他未定义类型
		default:
			apiActive.ActiveType = active.ActiveType
			logger.Warn("未处理的活动类型", zap.Any("activeType", active.ActiveType))
		}

		apiActives = append(apiActives, apiActive)
		if lasttimeStamp > active.CreateAt.Unix() {
			lasttimeStamp = active.CreateAt.Unix()
		}
	}
	return &api.FetchActivesResponse{
		Code: api.ResponseCode_OK,
		Msg:  "success",
		Data: &api.FetchActivesResponse_Data{
			List:      apiActives,
			Timestamp: lasttimeStamp,
			PageSize:  int64(req.GetPageSize()),
			Offset:    int64(req.GetOffset()),
			HaveMore:  hasMore,
			Total:     int64(total),
		},
	}, nil
}

func (user *UserService) FetchUserStoryActives(ctx context.Context, req *api.FetchActivesRequest) (*api.FetchActivesResponse, error) {
	var (
		storyIds      []int64
		err           error
		storyMap      = make(map[int64]*models.Story)
		lasttimeStamp = req.GetTimestamp()
		hasMore       bool
		total         int
		actives       = make([]*models.Active, 0)
		apiActives    = make([]*api.ActiveInfo, 0)
		allActives    = make([]*models.Active, 0)
	)
	if req.GetAtype() == api.ActiveFlowType_StoryFlowType {
		storyIds, err = models.GetUserFollowedStoryIds(ctx, int(req.GetUserId()))
		if err != nil {
			logger.Error("get user followed story ids failed", zap.Int64("user_id", req.GetUserId()), zap.Error(err))
			return nil, err
		}
		if len(storyIds) == 0 {
			logger.Info("user has no followed story", zap.Int64("user_id", req.GetUserId()))
			return &api.FetchActivesResponse{
				Code: api.ResponseCode_OK,
				Msg:  "success",
				Data: &api.FetchActivesResponse_Data{
					List:      nil,
					Timestamp: lasttimeStamp,
					PageSize:  int64(req.GetPageSize()),
					Offset:    int64(req.GetOffset()),
					HaveMore:  false,
				},
			}, nil
		}
	}
	if len(storyIds) != 0 {
		actives, hasMore, total, err = models.GetActiveByFollowingStoryID(req.GetUserId(), storyIds, int(req.GetOffset()), int(req.GetPageSize()))
		if err != nil {
			logger.Error("get user followed story actives failed", zap.Int64("user_id", req.GetUserId()), zap.Error(err))
			return nil, err
		}
		if len(actives) != 0 {
			allActives = append(allActives, actives...)
		}
		targetStoryIds := make([]int64, 0)
		for _, active := range actives {
			storyMap[active.StoryId] = &models.Story{}
			targetStoryIds = append(targetStoryIds, active.StoryId)
		}
		stories, err := models.GetStoriesByIDs(ctx, targetStoryIds)
		if err != nil {
			logger.Error("get user followed story failed", zap.Int64("user_id", req.GetUserId()), zap.Error(err))
			return nil, err
		}
		for _, story := range stories {
			storyMap[int64(story.ID)] = story
		}
	}
	// TODO: fetch user actives

	sort.Sort(models.ActiveList(allActives))
	for _, active := range allActives {
		apiActive := &api.ActiveInfo{}
		apiActive.ActiveId = int64(active.ID)
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
		}

		apiActives = append(apiActives, apiActive)
		if lasttimeStamp > active.CreateAt.Unix() {
			lasttimeStamp = active.CreateAt.Unix()
		}
	}

	return &api.FetchActivesResponse{
		Code: api.ResponseCode_OK,
		Msg:  "success",
		Data: &api.FetchActivesResponse_Data{
			List:      apiActives,
			Timestamp: lasttimeStamp,
			PageSize:  int64(req.GetPageSize()),
			Offset:    int64(req.GetOffset()),
			HaveMore:  hasMore,
			Total:     int64(total),
		},
	}, nil
}

func (user *UserService) FetchUserAllActives(ctx context.Context, req *api.FetchActivesRequest) (*api.FetchActivesResponse, error) {
	var (
		err           error
		groupMap      = make(map[int64]*models.Group)
		storyMap      = make(map[int64]*models.Story)
		roleMap       = make(map[int64]*models.StoryRole)
		storyboardMap = make(map[int64]*models.StoryBoard)
		lasttimeStamp = req.GetTimestamp()
		hasMore       bool
		total         int
		apiActives    = make([]*api.ActiveInfo, 0)
		allActives    = make([]*models.Active, 0)

		targetGroupIds      = make([]int64, 0)
		targetStoryIds      = make([]int64, 0)
		targetRoleIds       = make([]int64, 0)
		targetStoryboardIds = make([]int64, 0)
	)
	allActives, hasMore, total, err = models.GetActiveByUserID(req.GetUserId(), int(req.GetPageSize()), int(req.GetOffset()))
	if err != nil {
		logger.Error("get user followed group actives failed", zap.Int64("user_id", req.GetUserId()), zap.Error(err))
		return nil, err
	}

	if len(allActives) != 0 {
		for _, active := range allActives {
			groupMap[active.GroupId] = &models.Group{}
			storyMap[active.StoryId] = &models.Story{}
			roleMap[active.StoryRoleId] = &models.StoryRole{}
			storyboardMap[active.StoryBoardId] = &models.StoryBoard{}
			targetGroupIds = append(targetGroupIds, active.GroupId)
			targetStoryIds = append(targetStoryIds, active.StoryId)
			targetRoleIds = append(targetRoleIds, active.StoryRoleId)
			targetStoryboardIds = append(targetStoryboardIds, active.StoryBoardId)
		}
	} else {
		logger.Error("get user followed group actives failed", zap.Int64("user_id", req.GetUserId()), zap.Any("no active", allActives))
		return &api.FetchActivesResponse{
			Code: api.ResponseCode_OK,
			Msg:  "success",
			Data: &api.FetchActivesResponse_Data{
				List:      nil,
				Timestamp: lasttimeStamp,
				PageSize:  int64(req.GetPageSize()),
				Offset:    int64(req.GetOffset()),
				HaveMore:  false,
				Total:     int64(total),
			},
		}, nil
	}
	if len(groupMap) != 0 {
		groups, err := models.GetGroupsByIds(targetGroupIds)
		if err != nil {
			logger.Error("get user followed group failed", zap.Int64("user_id", req.GetUserId()), zap.Error(err))
			return nil, err
		}
		for _, group := range groups {
			groupMap[int64(group.ID)] = group
		}
	}
	if len(storyMap) != 0 {
		stories, err := models.GetStoriesByIDs(ctx, targetStoryIds)
		if err != nil {
			logger.Error("get user followed story failed", zap.Int64("user_id", req.GetUserId()), zap.Error(err))
			return nil, err
		}
		for _, story := range stories {
			storyMap[int64(story.ID)] = story
		}
	}
	if len(roleMap) != 0 {
		roles, err := models.GetStoryRolesByIDs(ctx, targetRoleIds)
		if err != nil {
			logger.Error("get user followed role failed", zap.Int64("user_id", req.GetUserId()), zap.Error(err))
			return nil, err
		}
		for _, role := range roles {
			roleMap[int64(role.ID)] = role
		}
	}
	if len(storyboardMap) != 0 {
		// 过滤掉0值的ID
		validStoryboardIds := make([]int64, 0)
		for _, id := range targetStoryboardIds {
			if id > 0 {
				validStoryboardIds = append(validStoryboardIds, id)
			}
		}
		if len(validStoryboardIds) > 0 {
			storyboards, _, err := models.GetStoryBoardsByStoryIds(ctx, validStoryboardIds, 1, len(validStoryboardIds), "create_at desc")
			if err != nil {
				logger.Error("get storyboards failed", zap.Int64("user_id", req.GetUserId()), zap.Error(err))
				return nil, err
			}
			for _, storyboard := range storyboards {
				storyboardMap[int64(storyboard.ID)] = storyboard
			}
		}
	}
	sort.Sort(models.ActiveList(allActives))
	for _, active := range allActives {
		apiActive := &api.ActiveInfo{}
		apiActive.ActiveId = int64(active.ID)
		apiActive.Ctime = active.CreateAt.Unix()
		apiActive.Mtime = active.UpdateAt.Unix()

		// 根据活动类型判断业务分类，然后使用大的 switch-case 处理数据填充
		switch {
		// 故事相关活动类型
		case active.ActiveType == api.ActiveType_NewStory ||
			active.ActiveType == api.ActiveType_FollowStory ||
			active.ActiveType == api.ActiveType_LikeStory ||
			active.ActiveType == api.ActiveType_ForkStory ||
			active.ActiveType == api.ActiveType_ShareStory:
			// 填充故事信息
			if story, exists := storyMap[active.StoryId]; exists && story != nil {
				apiActive.StoryInfo = &api.Story{
					Id:     int64(active.StoryId),
					Name:   story.Name,
					Title:  story.Title,
					Avatar: story.Avatar,
					Desc:   story.ShortDesc,
					Ctime:  story.CreateAt.Unix(),
					Mtime:  story.UpdateAt.Unix(),
				}
			}
			// 内部小 switch-case 处理 ActiveType 赋值
			switch active.ActiveType {
			case api.ActiveType_NewStory:
				apiActive.ActiveType = api.ActiveType_NewStory
			case api.ActiveType_FollowStory:
				apiActive.ActiveType = api.ActiveType_FollowStory
			case api.ActiveType_LikeStory:
				apiActive.ActiveType = api.ActiveType_LikeStory
			case api.ActiveType_ForkStory:
				apiActive.ActiveType = api.ActiveType_ForkStory
			case api.ActiveType_ShareStory:
				apiActive.ActiveType = api.ActiveType_ShareStory
			}

		// 故事角色相关活动类型
		case active.ActiveType == api.ActiveType_NewRole ||
			active.ActiveType == api.ActiveType_FollowRole ||
			active.ActiveType == api.ActiveType_LikeRole ||
			active.ActiveType == api.ActiveType_ShareRole:
			// 填充角色信息
			if role, exists := roleMap[active.StoryRoleId]; exists && role != nil {
				apiActive.RoleInfo = &api.StoryRole{
					RoleId:               active.StoryRoleId,
					CharacterName:        role.CharacterName,
					CharacterAvatar:      role.CharacterAvatar,
					CharacterDescription: role.CharacterDescription,
					CharacterPrompt:      role.CharacterPrompt,
					Ctime:                role.CreateAt.Unix(),
					Mtime:                int64(role.UpdateAt.Unix()),
					LikeCount:            role.LikeCount,
					FollowCount:          role.FollowCount,
					StoryboardNum:        role.StoryboardNum,
				}
			}
			// 填充故事信息
			if story, exists := storyMap[active.StoryId]; exists && story != nil {
				apiActive.StoryInfo = &api.Story{
					Id:     int64(active.StoryId),
					Name:   story.Name,
					Avatar: story.Avatar,
					Title:  story.Title,
					Desc:   story.ShortDesc,
					Ctime:  story.CreateAt.Unix(),
					Mtime:  story.UpdateAt.Unix(),
				}
			}
			// 内部小 switch-case 处理 ActiveType 赋值
			switch active.ActiveType {
			case api.ActiveType_NewRole:
				apiActive.ActiveType = api.ActiveType_NewRole
			case api.ActiveType_FollowRole:
				apiActive.ActiveType = api.ActiveType_FollowRole
			case api.ActiveType_LikeRole:
				apiActive.ActiveType = api.ActiveType_LikeRole
			case api.ActiveType_ShareRole:
				apiActive.ActiveType = api.ActiveType_ShareRole
			}

		// 故事板相关活动类型
		case active.ActiveType == api.ActiveType_NewStoryBoard ||
			active.ActiveType == api.ActiveType_LikeStoryBoard ||
			active.ActiveType == api.ActiveType_ShareStoryBoard:
			// 填充故事板信息
			if storyboard, exists := storyboardMap[active.StoryBoardId]; exists && storyboard != nil {
				apiActive.BoardInfo = &api.StoryBoard{
					StoryBoardId: int64(active.StoryBoardId),
					Title:        storyboard.Title,
					Content:      storyboard.Description,
					Creator:      storyboard.CreatorID,
					StoryId:      storyboard.StoryID,
					Stage:        storyboard.Stage,
					ForkNum:      int64(storyboard.ForkNum),
					Ctime:        storyboard.CreateAt.Unix(),
					Mtime:        storyboard.UpdateAt.Unix(),
				}
			}
			// 填充故事信息
			if story, exists := storyMap[active.StoryId]; exists && story != nil {
				apiActive.StoryInfo = &api.Story{
					Id:     int64(active.StoryId),
					Name:   story.Name,
					Title:  story.Title,
					Avatar: story.Avatar,
					Desc:   story.ShortDesc,
					Ctime:  story.CreateAt.Unix(),
					Mtime:  story.UpdateAt.Unix(),
				}
			}
			// 内部小 switch-case 处理 ActiveType 赋值
			switch active.ActiveType {
			case api.ActiveType_NewStoryBoard:
				apiActive.ActiveType = api.ActiveType_NewStoryBoard
			case api.ActiveType_LikeStoryBoard:
				apiActive.ActiveType = api.ActiveType_LikeStoryBoard
			case api.ActiveType_ShareStoryBoard:
				apiActive.ActiveType = api.ActiveType_ShareStoryBoard
			}

		// 小组相关活动类型
		case active.ActiveType == api.ActiveType_JoinGroup ||
			active.ActiveType == api.ActiveType_FollowGroup ||
			active.ActiveType == api.ActiveType_LikeGroup:
			// 填充小组信息
			if group, exists := groupMap[active.GroupId]; exists && group != nil {
				apiActive.GroupInfo = &api.GroupInfo{
					GroupId: active.GroupId,
					Name:    group.Name,
					Avatar:  group.Avatar,
					Desc:    group.ShortDesc,
					Creator: group.CreatorID,
					Owner:   group.OwnerID,
					Ctime:   group.CreateAt.Unix(),
					Mtime:   group.UpdateAt.Unix(),
				}
			}
			// 内部小 switch-case 处理 ActiveType 赋值
			switch active.ActiveType {
			case api.ActiveType_JoinGroup:
				apiActive.ActiveType = api.ActiveType_JoinGroup
			case api.ActiveType_FollowGroup:
				apiActive.ActiveType = api.ActiveType_FollowGroup
			case api.ActiveType_LikeGroup:
				apiActive.ActiveType = api.ActiveType_LikeGroup
			}

		// 默认情况：不公开互动或其他未定义类型
		default:
			apiActive.ActiveType = active.ActiveType
			logger.Warn("未处理的活动类型", zap.Any("activeType", active.ActiveType))
		}

		apiActives = append(apiActives, apiActive)
		if lasttimeStamp > active.CreateAt.Unix() {
			lasttimeStamp = active.CreateAt.Unix()
		}
	}

	return &api.FetchActivesResponse{
		Code: api.ResponseCode_OK,
		Msg:  "success",
		Data: &api.FetchActivesResponse_Data{
			List:      apiActives,
			Timestamp: lasttimeStamp,
			PageSize:  int64(req.GetPageSize()),
			Offset:    int64(req.GetOffset()),
			HaveMore:  hasMore,
			Total:     int64(total),
		},
	}, nil

}

// 组织内搜索指定用户
func (user *UserService) SearchUser(ctx context.Context, req *api.SearchUserRequest) (
	*api.SearchUserResponse, error) {
	users, total, err := models.GetUsersByName(ctx, req.GetName(), int(req.GetOffset()), int(req.GetPageSize()))
	if err != nil {
		logger.Error("search user failed", zap.Error(err))
		return nil, err
	}
	apiUsers := make([]*api.UserInfo, 0)
	for _, user := range users {
		apiUsers = append(apiUsers, &api.UserInfo{
			UserId:   int64(user.ID),
			Name:     user.Name,
			Avatar:   user.Avatar,
			Email:    user.Email,
			Location: user.Location,
		})
	}
	return &api.SearchUserResponse{
		Code: api.ResponseCode_OK,
		Msg:  "success",
		Data: &api.SearchUserResponse_Data{
			List:     apiUsers,
			Total:    int32(total),
			Offset:   int64(req.GetOffset()),
			PageSize: int64(req.GetPageSize()),
			HaveMore: total > int64(req.GetOffset())*int64(req.GetPageSize()),
		},
	}, nil
}

func (user *UserService) UserWatching(ctx context.Context, req *api.UserWatchingRequest) (
	*api.UserWatchingResponse, error) {
	return nil, errors.ErrFeatureNotImplemented
}

func (user *UserService) GetUserProfile(ctx context.Context, req *api.GetUserProfileRequest) (
	*api.GetUserProfileResponse, error) {
	profile := &models.UserProfile{
		UserId: req.GetUserId(),
	}
	err := profile.GetByUserId(ctx)
	if err != nil {
		logger.Error("get user profile failed", zap.Error(err))
		return nil, err
	}
	return &api.GetUserProfileResponse{
		Code:    api.ResponseCode_OK,
		Message: "success",
		Info:    convertModelUserProfileToApi(profile),
	}, nil
}

func (user *UserService) UpdateUserProfile(ctx context.Context, req *api.UpdateUserProfileRequest) (
	*api.UpdateUserProfileResponse, error) {
	profile := &models.UserProfile{
		UserId: req.GetUserId(),
	}
	err := profile.GetByUserId(ctx)
	if err != nil {
		logger.Error("get user profile failed", zap.Error(err))
		return nil, err
	}
	if req.GetBackgroundImage() != "" {
		profile.UserId = req.GetUserId()
		profile.Background = req.GetBackgroundImage()
		err = profile.Update(ctx)
		if err != nil {
			logger.Error("update user profile backgroud image failed", zap.Error(err))
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
		needUpdates["short_desc"] = req.GetDescription()
	}

	err = models.UpdateUserInfo(ctx, req.GetUserId(), needUpdates)
	if err != nil {
		return &api.UpdateUserProfileResponse{
			Code:    api.ResponseCode_OPERATION_FAILED,
			Message: "user info err:" + err.Error(),
		}, nil
	}

	return &api.UpdateUserProfileResponse{
		Code:    api.ResponseCode_OK,
		Message: "success",
	}, nil
}

func (user *UserService) UpdateUserBackgroundImage(ctx context.Context, req *api.UpdateUserBackgroundImageRequest) (*api.UpdateUserBackgroundImageResponse, error) {
	profile := &models.UserProfile{
		UserId: req.GetUserId(),
	}
	err := profile.GetByUserId(ctx)
	if err != nil {
		logger.Error("get user profile failed", zap.Error(err))
		return nil, err
	}
	profile.Background = req.GetBackgroundImage()
	err = profile.Update(ctx)
	if err != nil {
		return &api.UpdateUserBackgroundImageResponse{
			Code:    api.ResponseCode_OPERATION_FAILED,
			Message: "update user background image failed : " + err.Error(),
		}, nil
	}
	aliyunClient := aliyun.GetGlobalClient()
	err = aliyunClient.PersistImages(req.GetBackgroundImage())
	if err != nil {
		return nil, err
	}
	return &api.UpdateUserBackgroundImageResponse{
		Code:    api.ResponseCode_OK,
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

	// 尝试从缓存获取用户状态
	userStatusCache := cache.GetUserStatusCache()
	cacheKey := userStatusCache.GetGroupUserStatusKey(groupId, int64(userID))
	cachedStatus, err := userStatusCache.GetCachedUserStatus(ctx, cacheKey)
	if err == nil && cachedStatus != nil {
		logger.Info("从缓存获取小组用户状态成功", zap.Int64("groupId", groupId), zap.Int64("userID", int64(userID)))
		return cachedStatus, nil
	}

	// 缓存未命中，从数据库查询
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

	// 将结果缓存
	err = userStatusCache.CacheUserStatus(ctx, cacheKey, cu)
	if err != nil {
		logger.Warn("缓存小组用户状态失败", zap.Error(err), zap.Int64("groupId", groupId), zap.Int64("userID", int64(userID)))
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

	// 尝试从缓存获取用户状态
	userStatusCache := cache.GetUserStatusCache()
	cacheKey := userStatusCache.GetStoryRoleUserStatusKey(roleId, int64(userID))
	cachedStatus, err := userStatusCache.GetCachedUserStatus(ctx, cacheKey)
	if err == nil && cachedStatus != nil {
		logger.Info("从缓存获取故事角色用户状态成功", zap.Int64("roleId", roleId), zap.Int64("userID", int64(userID)))
		return cachedStatus, nil
	}

	// 缓存未命中，从数据库查询
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

	// 将结果缓存
	err = userStatusCache.CacheUserStatus(ctx, cacheKey, cu)
	if err != nil {
		logger.Warn("缓存故事角色用户状态失败", zap.Error(err), zap.Int64("roleId", roleId), zap.Int64("userID", int64(userID)))
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

	// 尝试从缓存获取用户状态
	userStatusCache := cache.GetUserStatusCache()
	cacheKey := userStatusCache.GetStoryUserStatusKey(storyId, int64(userID))
	cachedStatus, err := userStatusCache.GetCachedUserStatus(ctx, cacheKey)
	if err == nil && cachedStatus != nil {
		logger.Info("从缓存获取故事用户状态成功", zap.Int64("storyId", storyId), zap.Int64("userID", int64(userID)))
		return cachedStatus, nil
	}

	// 缓存未命中，从数据库查询
	cu := new(api.WhatCurrentUserStatus)
	// 查询用户是否关注了故事
	follow, err := models.GetWatchItemByStoryAndUser(ctx, storyId, int(userID))
	if err != nil {
		return nil, err
	}
	if follow != nil && follow.Deleted == false {
		cu.IsFollowed = true
	}
	// 查询用户是否点赞了故事
	like, err := models.GetLikeItemByStoryAndUser(ctx, storyId, int(userID))
	if err != nil {
		return nil, err
	}
	if like != nil && like.Deleted == false {
		cu.IsLiked = true
	}

	// 将结果缓存
	err = userStatusCache.CacheUserStatus(ctx, cacheKey, cu)
	if err != nil {
		logger.Warn("缓存故事用户状态失败", zap.Error(err), zap.Int64("storyId", storyId), zap.Int64("userID", int64(userID)))
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

	// 尝试从缓存获取用户状态
	userStatusCache := cache.GetUserStatusCache()
	cacheKey := userStatusCache.GetStoryboardUserStatusKey(storyboardId, int64(userID))
	cachedStatus, err := userStatusCache.GetCachedUserStatus(ctx, cacheKey)
	if err == nil && cachedStatus != nil {
		logger.Info("从缓存获取分镜用户状态成功", zap.Int64("storyboardId", storyboardId), zap.Int64("userID", int64(userID)))
		return cachedStatus, nil
	}

	// 缓存未命中，从数据库查询
	cu := new(api.WhatCurrentUserStatus)
	// 查询用户是否点赞了分镜
	like, err := models.GetLikeItemByStoryBoardAndUser(ctx, storyboardId, int(userID))
	if err != nil {
		return nil, err
	}
	if like != nil && like.Deleted == false {
		cu.IsLiked = true
	}

	// 将结果缓存
	err = userStatusCache.CacheUserStatus(ctx, cacheKey, cu)
	if err != nil {
		logger.Warn("缓存分镜用户状态失败", zap.Error(err), zap.Int64("storyboardId", storyboardId), zap.Int64("userID", int64(userID)))
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
		NumFollowers:      int32(profile.FollowersNum),
		NumFollowing:      int32(profile.FollowingNum),
		Ctime:             profile.CreateAt.Unix(),
		Mtime:             profile.UpdateAt.Unix(),
	}
}

func (user *UserService) FollowUser(ctx context.Context, req *api.FollowUserRequest) (*api.FollowUserResponse, error) {
	traceId := utils.GetTraceID(ctx)
	logger.Info("FollowUser called", zap.String("traceId", traceId), zap.Any("req", req))
	if req.GetUserId() <= 0 || req.GetFollowerId() <= 0 {
		logger.Error("FollowUser failed: invalid user id", zap.String("traceId", traceId), zap.Int64("user_id", req.GetUserId()), zap.Int64("follower_id", req.GetFollowerId()))
		return &api.FollowUserResponse{Code: api.ResponseCode_INVALID_PARAMETER, Message: "invalid user id"}, nil
	}
	if req.GetUserId() == req.GetFollowerId() {
		logger.Error("FollowUser failed: cannot follow self", zap.String("traceId", traceId), zap.Int64("user_id", req.GetUserId()))
		return &api.FollowUserResponse{Code: api.ResponseCode_INVALID_PARAMETER, Message: "cannot follow self"}, nil
	}
	created, err := models.CreateUserFollow(ctx, req.GetUserId(), req.GetFollowerId())
	if err != nil {
		logger.Error("FollowUser failed: create follow error", zap.String("traceId", traceId), zap.Error(err))
		return &api.FollowUserResponse{Code: api.ResponseCode_OPERATION_FAILED, Message: err.Error()}, nil
	}
	if !created {
		logger.Info("FollowUser idempotent: already following", zap.String("traceId", traceId), zap.Int64("user_id", req.GetUserId()), zap.Int64("follower_id", req.GetFollowerId()))
		return &api.FollowUserResponse{Code: api.ResponseCode_OK, Message: "already following"}, nil
	}

	// 创建关注通知（异步，避免阻塞主流程）
	go func() {
		bgCtx := context.Background()
		// followeeID 是被关注者，应该收到通知
		followerID64 := req.GetFollowerId()
		notifErr := llmchatservice.CreateSystemNotification(bgCtx, llmchatservice.CreateNotificationParams{
			UserID:        req.GetUserId(),
			Type:          models.SystemNotificationTypeFollow,
			Title:         "关注提醒",
			Content:       "有人关注了你",
			RelatedUserID: &followerID64,
		})
		if notifErr != nil {
			logger.Error("FollowUser failed: create follow notification error", zap.String("traceId", traceId), zap.Error(notifErr))
		}
	}()

	logger.Info("FollowUser success", zap.String("traceId", traceId), zap.Int64("user_id", req.GetUserId()), zap.Int64("follower_id", req.GetFollowerId()))
	return &api.FollowUserResponse{Code: api.ResponseCode_OK, Message: "success"}, nil
}

func (user *UserService) UnfollowUser(ctx context.Context, req *api.UnfollowUserRequest) (*api.UnfollowUserResponse, error) {
	traceId := utils.GetTraceID(ctx)
	logger.Info("UnfollowUser called", zap.String("traceId", traceId), zap.Any("req", req))
	if req.GetUserId() <= 0 || req.GetFollowerId() <= 0 {
		logger.Error("UnfollowUser failed: invalid user id", zap.String("traceId", traceId), zap.Int64("user_id", req.GetUserId()), zap.Int64("follower_id", req.GetFollowerId()))
		return &api.UnfollowUserResponse{Code: api.ResponseCode_INVALID_PARAMETER, Message: "invalid user id"}, nil
	}
	if req.GetUserId() == req.GetFollowerId() {
		logger.Error("UnfollowUser failed: cannot unfollow self", zap.String("traceId", traceId), zap.Int64("user_id", req.GetUserId()))
		return &api.UnfollowUserResponse{Code: api.ResponseCode_INVALID_PARAMETER, Message: "cannot unfollow self"}, nil
	}
	removed, err := models.DeleteUserFollow(ctx, req.GetUserId(), req.GetFollowerId())
	if err != nil {
		logger.Error("UnfollowUser failed: delete follow error", zap.String("traceId", traceId), zap.Error(err))
		return &api.UnfollowUserResponse{Code: api.ResponseCode_OPERATION_FAILED, Message: err.Error()}, nil
	}
	if !removed {
		logger.Info("UnfollowUser idempotent: not following", zap.String("traceId", traceId), zap.Int64("user_id", req.GetUserId()), zap.Int64("follower_id", req.GetFollowerId()))
		return &api.UnfollowUserResponse{Code: api.ResponseCode_OK, Message: "not following"}, nil
	}
	logger.Info("UnfollowUser success", zap.String("traceId", traceId), zap.Int64("user_id", req.GetUserId()), zap.Int64("follower_id", req.GetFollowerId()))
	return &api.UnfollowUserResponse{Code: api.ResponseCode_OK, Message: "success"}, nil
}

func (user *UserService) GetFollowList(ctx context.Context, req *api.GetFollowListRequest) (*api.GetFollowListResponse, error) {
	traceId := utils.GetTraceID(ctx)
	logger.Info("GetFollowList called", zap.String("traceId", traceId), zap.Any("req", req))
	if req.GetUserId() <= 0 {
		logger.Error("GetFollowList failed: invalid user id", zap.String("traceId", traceId), zap.Int64("user_id", req.GetUserId()))
		return &api.GetFollowListResponse{Code: api.ResponseCode_INVALID_PARAMETER, Message: "invalid user id"}, nil
	}
	users, total, err := models.GetFollowList(req.GetUserId(), int(req.GetOffset()), int(req.GetPageSize()))
	if err != nil {
		logger.Error("GetFollowList failed: get follow list error", zap.String("traceId", traceId), zap.Error(err))
		return &api.GetFollowListResponse{Code: api.ResponseCode_OPERATION_FAILED, Message: err.Error()}, nil
	}
	apiUsers := make([]*api.UserInfo, 0, len(users))
	for _, u := range users {
		apiUsers = append(apiUsers, &api.UserInfo{
			UserId:   int64(u.ID),
			Name:     utils.MaskContent(u.Name),
			Avatar:   u.Avatar,
			Email:    utils.MaskContent(u.Email),
			Location: utils.MaskContent(u.Location),
		})
	}
	logger.Info("GetFollowList success", zap.String("traceId", traceId), zap.Int64("user_id", req.GetUserId()), zap.Int("count", len(apiUsers)))
	return &api.GetFollowListResponse{
		Code:      api.ResponseCode_OK,
		Message:   "success",
		Followers: apiUsers,
		Total:     int64(total),
		HaveMore:  total > int64(req.GetOffset())*int64(req.GetPageSize()),
	}, nil
}

func (user *UserService) GetFollowerList(ctx context.Context, req *api.GetFollowerListRequest) (*api.GetFollowerListResponse, error) {
	traceId := utils.GetTraceID(ctx)
	logger.Info("GetFollowerList called", zap.String("traceId", traceId), zap.Any("req", req))
	if req.GetUserId() <= 0 {
		logger.Error("GetFollowerList failed: invalid user id", zap.String("traceId", traceId), zap.Int64("user_id", req.GetUserId()))
		return &api.GetFollowerListResponse{Code: api.ResponseCode_INVALID_PARAMETER, Message: "invalid user id"}, nil
	}
	users, total, err := models.GetFollowerList(req.GetUserId(), int(req.GetOffset()), int(req.GetPageSize()))
	if err != nil {
		logger.Error("GetFollowerList failed: get follower list error", zap.String("traceId", traceId), zap.Error(err))
		return &api.GetFollowerListResponse{Code: api.ResponseCode_OPERATION_FAILED, Message: err.Error()}, nil
	}
	apiUsers := make([]*api.UserInfo, 0, len(users))
	for _, u := range users {
		apiUsers = append(apiUsers, &api.UserInfo{
			UserId:   int64(u.ID),
			Name:     utils.MaskContent(u.Name),
			Avatar:   u.Avatar,
			Email:    utils.MaskContent(u.Email),
			Location: utils.MaskContent(u.Location),
		})
	}
	logger.Info("GetFollowerList success", zap.String("traceId", traceId), zap.Int64("user_id", req.GetUserId()), zap.Int("count", len(apiUsers)))
	return &api.GetFollowerListResponse{
		Code:      api.ResponseCode_OK,
		Message:   "success",
		Followers: apiUsers,
		Total:     int64(total),
		HaveMore:  total > int64(req.GetOffset())*int64(req.GetPageSize()),
	}, nil
}

func (user *UserService) FetchUserGenTaskStatus(ctx context.Context, req *api.FetchUserGenTaskStatusRequest) (*api.FetchUserGenTaskStatusResponse, error) {
	traceId := utils.GetTraceID(ctx)
	logger.Info("FetchUserGenTaskStatus called", zap.String("traceId", traceId), zap.Any("req", req))
	if req.GetUserId() <= 0 {
		logger.Error("FetchUserGenTaskStatus failed: invalid user id", zap.String("traceId", traceId), zap.Int64("user_id", req.GetUserId()))
		return &api.FetchUserGenTaskStatusResponse{Code: api.ResponseCode_INVALID_PARAMETER, Message: "invalid user id"}, nil
	}

	logger.Info("FetchUserGenTaskStatus success", zap.String("traceId", traceId), zap.Int64("user_id", req.GetUserId()))
	return &api.FetchUserGenTaskStatusResponse{
		Code:     api.ResponseCode_OK,
		Message:  "success",
		Total:    0,
		HaveMore: false,
		Tasks:    nil,
	}, nil
}
