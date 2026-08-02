package story

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	api "github.com/grapery/common-protoc/gen"
	"github.com/grapery/grapery/models"
	"github.com/grapery/grapery/pkg/active"
	"github.com/grapery/grapery/pkg/client"
	"github.com/grapery/grapery/pkg/cloud/aliyun"
	"github.com/grapery/grapery/pkg/cloud/coze"
	llmchatservice "github.com/grapery/grapery/service/llmchat"
	"github.com/grapery/grapery/utils"
	"github.com/grapery/grapery/utils/cache"
	"github.com/grapery/grapery/utils/compliance"
	"github.com/grapery/grapery/utils/convert"
	"github.com/grapery/grapery/utils/log"
)

var storyServer StoryServer

func init() {
	storyServer = NewStoryService()
}

func GetStoryServer() StoryServer {
	if storyServer == nil {
		storyServer = NewStoryService()
	}
	return storyServer
}

type StoryServer interface {
	CreateStory(ctx context.Context, req *api.CreateStoryRequest) (resp *api.CreateStoryResponse, err error)
	GetStory(ctx context.Context, req *api.GetStoryInfoRequest) (resp *api.GetStoryInfoResponse, err error)
	UpdateStory(ctx context.Context, req *api.UpdateStoryRequest) (resp *api.UpdateStoryResponse, err error)
	WatchStory(ctx context.Context, req *api.WatchStoryRequest) (resp *api.WatchStoryResponse, err error)

	RenderStory(ctx context.Context, req *api.RenderStoryRequest) (*api.RenderStoryResponse, error)
	GetStoryRender(ctx context.Context, req *api.GetStoryRenderRequest) (*api.GetStoryRenderResponse, error)

	LikeStory(ctx context.Context, req *api.LikeStoryRequest) (resp *api.LikeStoryResponse, err error)
	UnLikeStory(ctx context.Context, req *api.UnLikeStoryRequest) (resp *api.UnLikeStoryResponse, err error)

	SearchStories(ctx context.Context, req *api.SearchStoriesRequest) (*api.SearchStoriesResponse, error)
	SearchRoles(ctx context.Context, req *api.SearchRolesRequest) (*api.SearchRolesResponse, error)
	GetStoryContributors(ctx context.Context, req *api.GetStoryContributorsRequest) (*api.GetStoryContributorsResponse, error)

	GetStoryRoleList(ctx context.Context, req *api.GetStoryRoleListRequest) (*api.GetStoryRoleListResponse, error)

	TrendingStory(ctx context.Context, req *api.TrendingStoryRequest) (*api.TrendingStoryResponse, error)
	TrendingStoryRole(ctx context.Context, req *api.TrendingStoryRoleRequest) (*api.TrendingStoryRoleResponse, error)

	GetStoryImageStyle(ctx context.Context, req *api.GetStoryImageStyleRequest) (*api.GetStoryImageStyleResponse, error)
	UpdateStoryImageStyle(ctx context.Context, req *api.UpdateStoryImageStyleRequest) (*api.UpdateStoryImageStyleResponse, error)
	UpdateStorySenceMaxNumber(ctx context.Context, req *api.UpdateStorySenceMaxNumberRequest) (*api.UpdateStorySenceMaxNumberResponse, error)

	GetStoryParticipants(ctx context.Context, req *api.GetStoryParticipantsRequest) (*api.GetStoryParticipantsResponse, error)
	UpdateStoryAvatar(ctx context.Context, req *api.UpdateStoryAvatarRequest) (*api.UpdateStoryAvatarResponse, error)
	UpdateStoryCover(ctx context.Context, req *api.UpdateStoryCoverRequest) (*api.UpdateStoryCoverResponse, error)
	UnWatchStory(ctx context.Context, req *api.UnWatchStoryRequest) (*api.UnWatchStoryResponse, error)
}

type StoryService struct {
	bailianClient *client.AliyunStoryClient
	doubaoClient  *client.DoubaoClient
	cozeClient    *coze.HuoShanCozeClient
	helper        HelperServer
}

func NewStoryService() *StoryService {
	return &StoryService{
		bailianClient: client.NewAliyunClient(),
		doubaoClient:  client.NewDoubaoClient(),
		helper:        GetStoryHelper(),
	}
}

func ConvertStoryToApiStory(story *models.Story) *api.Story {
	item := &api.Story{
		Id:           int64(story.ID),
		Title:        story.Title,
		Name:         story.Title,
		Origin:       story.Origin,
		Avatar:       story.Avatar,
		Desc:         story.ShortDesc,
		CreatorId:    story.CreatorID,
		GroupId:      story.GroupID,
		Status:       int32(story.Status),
		IsAiGen:      story.AIGen,
		IsClose:      story.IsClose,
		OwnerId:      story.OwnerID,
		RootBoardId:  int64(story.RootBoardID),
		LikeCount:    story.LikeCount,
		CommentCount: story.CommentCount,
		ShareCount:   story.ShareCount,
		FollowCount:  story.FollowCount,
		TotalBoards:  story.TotalBoards,
		TotalRoles:   story.TotalRoles,
		TotalMembers: story.TotalMembers,
		Cover:        story.Cover,
		SenceNum:     story.SenceMaxNumber,
		Style:        story.Style,
		Ctime:        story.CreateAt.Unix(),
		Mtime:        story.UpdateAt.Unix(),
	}
	_ = json.Unmarshal([]byte(story.Params), &item.Params)
	return item
}

func ConvertApiStoryToStory(apiStory *api.Story) *models.Story {
	item := &models.Story{
		Title:          apiStory.Name,
		Name:           apiStory.Name,
		ShortDesc:      apiStory.Desc,
		CreatorID:      apiStory.CreatorId,
		OwnerID:        apiStory.CreatorId,
		GroupID:        apiStory.GroupId,
		Origin:         apiStory.Origin,
		RootBoardID:    int(apiStory.RootBoardId),
		AIGen:          apiStory.IsAiGen,
		Avatar:         apiStory.Avatar,
		Cover:          apiStory.Cover,
		Style:          apiStory.Style,
		SenceMaxNumber: apiStory.SenceNum,
		Status:         models.StoryStatus(apiStory.Status),
	}
	params, _ := json.Marshal(apiStory.Params)
	item.Params = string(params)
	return item
}

func (s *StoryService) CreateStory(ctx context.Context, req *api.CreateStoryRequest) (resp *api.CreateStoryResponse, err error) {
	log.Log().Info("[CreateStory] 入参", zap.Any("req", req))
	isPass, err := compliance.TextCompliance(req.GetShortDesc())
	if err != nil {
		log.Log().Error("[CreateStory] 简介合规检测失败", zap.Error(err))
		return nil, err
	}
	if !isPass {
		log.Log().Error("[CreateStory] 简介合规检测失败", zap.Error(err))
		return nil, err
	}
	if !req.GetIsAiGen() {
		log.Log().Info("[CreateStory] 非AI生成故事任务", zap.Any("req", req))
		return nil, fmt.Errorf("not AI gen story task")
	}
	if req.GetParams().Background != "" {
		isPass, err := compliance.TextCompliance(req.GetParams().Background)
		if err != nil {
			log.Log().Error("[CreateStory] 背景合规检测失败", zap.Error(err))
			return nil, err
		}
		if !isPass {
			log.Log().Error("[CreateStory] 背景合规检测失败", zap.Error(err))
			return nil, err
		}
	} else {
		log.Log().Info("[CreateStory] 背景为空，使用Origin", zap.String("origin", req.Origin))
		req.Params.Background = req.Origin
	}
	if req.GetParams().StoryDescription != "" {
		isPass, err := compliance.TextCompliance(req.GetParams().StoryDescription)
		if err != nil {
			log.Log().Error("[CreateStory] 故事描述合规检测失败", zap.Error(err))
			return nil, err
		}
		if !isPass {
			log.Log().Error("[CreateStory] 故事描述合规检测失败", zap.Error(err))
			return nil, err
		}
	}
	if req.GetParams().NegativePrompt != "" {
		isPass, err := compliance.TextCompliance(req.GetParams().NegativePrompt)
		if err != nil {
			log.Log().Error("[CreateStory] NegativePrompt合规检测失败", zap.Error(err))
			return nil, err
		}
		if !isPass {
			log.Log().Error("[CreateStory] NegativePrompt合规检测失败", zap.Error(err))
			return nil, err
		}
	}
	group := &models.Group{}
	group.ID = uint(req.GroupId)
	err = group.GetByID(ctx)
	if err != nil {
		log.Log().Error("[CreateStory] 获取小组信息失败", zap.Error(err), zap.Int64("groupId", int64(req.GroupId)))
		return nil, err
	}
	params, _ := json.Marshal(req.Params)
	newStory := &models.Story{
		Title:          req.GetTitle(),
		ShortDesc:      req.GetShortDesc(),
		Origin:         req.GetOrigin(),
		Status:         models.StoryStatus(req.GetStatus()),
		RootBoardID:    0,
		GroupID:        req.GetGroupId(),
		AIGen:          req.GetIsAiGen(),
		CreatorID:      req.GetCreatorId(),
		Params:         string(params),
		FollowCount:    1,
		LikeCount:      1,
		TotalMembers:   1,
		SenceMaxNumber: 5,
		ShareCount:     0,
		CommentCount:   0,
		TotalBoards:    0,
		TotalRoles:     0,
		IsClose:        false,
		Style:          styles[0].Style,
	}
	if !req.GetIsAiGen() {
		newStory.SenceMaxNumber = 9
	}
	if req.GetParams().Style != "" {
		newStory.Style = req.GetParams().Style
	}
	if req.GetParams().SceneCount >= 1 {
		newStory.SenceMaxNumber = int64(req.GetParams().SceneCount)
	}

	storyId, err := models.CreateStory(ctx, newStory)
	if err != nil {
		log.Log().Error("[CreateStory] 创建故事失败", zap.Error(err))
		return &api.CreateStoryResponse{
			Code:    -1,
			Message: fmt.Sprintf("create story failed: %s", err.Error()),
			Data:    nil,
		}, nil
	}
	log.Log().Info("[CreateStory] 故事创建成功", zap.Int64("storyId", storyId))
	err = models.IncGroupProfileStoryCount(ctx, int64(group.ID))
	if err != nil {
		log.Log().Error("[CreateStory] 增加小组故事数失败", zap.Error(err))
	}
	userProfile := &models.UserProfile{
		UserId: req.CreatorId,
	}
	err = userProfile.IncrementCreatedStoryNum(ctx)
	if err != nil {
		log.Log().Error("[CreateStory] 增加用户创建故事数失败", zap.Error(err))
	}
	createdWatch, watchErr := models.CreateWatchStoryItem(ctx, int(req.CreatorId), int64(storyId), int64(group.ID))
	if watchErr != nil {
		log.Log().Error("[CreateStory] 创建关注关系失败", zap.Error(watchErr))
	} else if createdWatch {
		if err := userProfile.IncrementWatchingStoryNum(ctx); err != nil {
			log.Log().Error("[CreateStory] 增加用户关注故事数失败", zap.Error(err))
		}
	}
	newStory.ID = uint(storyId)
	log.Log().Info("[CreateStory] 写入活跃信息", zap.Int64("storyId", storyId), zap.Int64("groupId", int64(group.ID)))
	active.GetActiveServer().WriteStoryActive(ctx, group, newStory, nil, nil, req.GetCreatorId(), api.ActiveType_NewStory)

	// 创建故事创建完成通知（异步，避免阻塞主流程）
	go func() {
		bgCtx := context.Background()
		storyID64 := int64(storyId)
		notifErr := llmchatservice.CreateSystemNotification(bgCtx, llmchatservice.CreateNotificationParams{
			UserID:        req.GetCreatorId(),
			Type:          models.SystemNotificationTypeStoryCreated,
			Title:         "故事创建完成",
			Content:       fmt.Sprintf("你的故事《%s》已创建完成", newStory.Title),
			RelatedStoryID: &storyID64,
		})
		if notifErr != nil {
			log.Log().Error("[CreateStory] 创建故事创建完成通知失败", zap.Error(notifErr), zap.Int64("storyId", storyId))
		}
	}()

	log.Log().Info("[CreateStory] 返回成功响应", zap.Int64("storyId", storyId))
	return &api.CreateStoryResponse{
		Code:    0,
		Message: "create story success",
		Data: &api.CreateStoryResponse_Data{
			StoryId: int32(storyId),
		},
	}, nil
}

func (s *StoryService) GetStory(ctx context.Context, req *api.GetStoryInfoRequest) (resp *api.GetStoryInfoResponse, err error) {
	log.Log().Info("[GetStory] 入参", zap.Any("req", req))
	storyInfo, err := models.GetStory(ctx, req.StoryId)
	if err != nil {
		log.Log().Error("[GetStory] 获取故事信息失败", zap.Error(err), zap.Int64("storyId", req.StoryId))
		return nil, err
	}
	log.Log().Info("[GetStory] 获取故事信息成功", zap.Any("storyInfo", storyInfo))
	cu, err := s.helper.GetStoryCurrentUserStatus(ctx, req.StoryId)
	if err != nil {
		log.Log().Error("[GetStory] 获取当前用户状态失败", zap.Error(err), zap.Int64("storyId", req.StoryId))
	} else {
		log.Log().Info("[GetStory] 获取当前用户状态成功", zap.Any("cu", cu))
	}
	creator, err := models.GetUserById(ctx, storyInfo.CreatorID)
	if err != nil {
		log.Log().Error("[GetStory] 获取故事创建者失败", zap.Error(err), zap.Int64("creatorId", storyInfo.CreatorID))
	} else {
		log.Log().Info("[GetStory] 获取故事创建者成功", zap.Any("creator", creator))
	}
	info := ConvertStoryToApiStory(storyInfo)
	info.CurrentUserStatus = cu
	log.Log().Info("[GetStory] 返回成功响应", zap.Any("info", info))
	return &api.GetStoryInfoResponse{
		Code:    0,
		Message: "OK",
		Data: &api.GetStoryInfoResponse_Data{
			Info:    info,
			Creator: convert.ConvertUserToApiUser(creator),
		},
	}, nil
}

func (s *StoryService) UpdateStory(ctx context.Context, req *api.UpdateStoryRequest) (resp *api.UpdateStoryResponse, err error) {
	log.Log().Info("[UpdateStory] 入参", zap.Any("req", req))
	needUpdateData := make(map[string]interface{})
	if req.GetIsAchieve() {
		needUpdateData["is_achieve"] = req.IsAchieve
		log.Log().Info("[UpdateStory] 需要更新is_achieve", zap.Bool("is_achieve", req.IsAchieve))
	}
	if req.GetShortDesc() != "" {
		needUpdateData["short_desc"] = req.ShortDesc
		log.Log().Info("[UpdateStory] 需要更新short_desc", zap.String("short_desc", req.ShortDesc))
	}
	if req.GetOrigin() != "" {
		needUpdateData["origin"] = req.Origin
		log.Log().Info("[UpdateStory] 需要更新origin", zap.String("origin", req.Origin))
	}
	if req.GetStatus() != 0 {
		needUpdateData["status"] = req.Status
		log.Log().Info("[UpdateStory] 需要更新status", zap.Int32("status", req.Status))
	}
	if req.GetParams() != nil {
		needUpdateData["params"] = req.Params
		log.Log().Info("[UpdateStory] 需要更新params", zap.Any("params", req.Params))
	}
	if req.GetIsAiGen() {
		needUpdateData["aigen"] = req.IsAiGen
		log.Log().Info("[UpdateStory] 需要更新aigen", zap.Bool("aigen", req.IsAiGen))
	}
	if req.GetIsClose() {
		needUpdateData["is_close"] = req.IsClose
		log.Log().Info("[UpdateStory] 需要更新is_close", zap.Bool("is_close", req.IsClose))
	}
	if len(needUpdateData) == 0 {
		log.Log().Warn("[UpdateStory] 无需更新任何字段", zap.Any("req", req))
		return &api.UpdateStoryResponse{}, nil
	}
	err = models.UpdateStorySpecColumns(ctx, req.StoryId, needUpdateData)
	if err != nil {
		log.Log().Error("[UpdateStory] 更新故事字段失败", zap.Error(err), zap.Int64("storyId", req.StoryId), zap.Any("updateData", needUpdateData))
		return nil, err
	}
	log.Log().Info("[UpdateStory] 更新故事字段成功", zap.Int64("storyId", req.StoryId), zap.Any("updateData", needUpdateData))
	return &api.UpdateStoryResponse{
		Code:    0,
		Message: "update story success",
		Data: &api.UpdateStoryResponse_Data{
			StoryId: int32(req.StoryId),
		},
	}, nil
}

func (s *StoryService) WatchStory(ctx context.Context, req *api.WatchStoryRequest) (resp *api.WatchStoryResponse, err error) {
	log.Log().Info("[WatchStory] 入参", zap.Any("req", req))
	storyInfo, err := models.GetStory(ctx, req.StoryId)
	if err != nil {
		log.Log().Error("[WatchStory] 获取故事信息失败", zap.Error(err), zap.Int64("storyId", req.StoryId))
		return nil, err
	}
	if storyInfo == nil {
		log.Log().Warn("[WatchStory] 故事不存在", zap.Int64("storyId", req.StoryId))
		return &api.WatchStoryResponse{
			Code:    int32(api.ResponseCode_OPERATION_FAILED),
			Message: "story not found",
		}, nil
	}
	if storyInfo.IsClose {
		log.Log().Warn("[WatchStory] 故事已关闭", zap.Int64("storyId", req.StoryId))
		return &api.WatchStoryResponse{
			Code:    int32(api.ResponseCode_OPERATION_FAILED),
			Message: "story is closed",
		}, nil
	}
	created, err := models.WatchStory(ctx, req.GetUserId(), req.GetStoryId(), int64(storyInfo.GroupID))
	if err != nil {
		log.Log().Error("[WatchStory] 创建关注关系失败", zap.Error(err), zap.Int64("storyId", req.StoryId), zap.Int64("userId", req.GetUserId()))
		return nil, err
	}
	if !created {
		log.Log().Info("[WatchStory] 已关注，无需重复关注", zap.Int64("storyId", req.StoryId), zap.Int64("userId", req.GetUserId()))
		return &api.WatchStoryResponse{
			Code:    int32(api.ResponseCode_OK),
			Message: "already watching",
		}, nil
	}
	userProfile := &models.UserProfile{UserId: req.GetUserId()}
	if err := userProfile.IncrementWatchingStoryNum(ctx); err != nil {
		log.Log().Error("[WatchStory] 增加用户关注故事数失败", zap.Error(err), zap.Int64("userId", req.GetUserId()))
	}
	userStatusCache := cache.GetUserStatusCache()
	if err := userStatusCache.InvalidateStoryUserStatusCache(ctx, req.GetStoryId(), req.GetUserId()); err != nil {
		log.Log().Warn("[WatchStory] 清除故事用户状态缓存失败", zap.Error(err), zap.Int64("storyId", req.GetStoryId()), zap.Int64("userId", req.GetUserId()))
	}
	log.Log().Info("[WatchStory] 返回成功响应", zap.Int64("storyId", req.StoryId), zap.Int64("userId", req.GetUserId()))
	return &api.WatchStoryResponse{
		Code:    int32(api.ResponseCode_OK),
		Message: "OK",
	}, nil
}

func (s *StoryService) UnWatchStory(ctx context.Context, req *api.UnWatchStoryRequest) (*api.UnWatchStoryResponse, error) {
	log.Log().Info("[UnWatchStory] 入参", zap.Any("req", req))
	removed, err := models.UnWatchStory(ctx, req.GetUserId(), req.GetStoryId())
	if err != nil {
		log.Log().Error("[UnWatchStory] 取消关注故事失败", zap.Error(err), zap.Int64("storyId", req.GetStoryId()), zap.Int64("userId", req.GetUserId()))
		return nil, err
	}
	if !removed {
		log.Log().Info("[UnWatchStory] 未关注，无需取消", zap.Int64("storyId", req.GetStoryId()), zap.Int64("userId", req.GetUserId()))
		return &api.UnWatchStoryResponse{
			Code:    int32(api.ResponseCode_OK),
			Message: "not watching",
		}, nil
	}
	userStatusCache := cache.GetUserStatusCache()
	if err := userStatusCache.InvalidateStoryUserStatusCache(ctx, req.GetStoryId(), req.GetUserId()); err != nil {
		log.Log().Warn("[UnWatchStory] 清除故事用户状态缓存失败", zap.Error(err), zap.Int64("storyId", req.GetStoryId()), zap.Int64("userId", req.GetUserId()))
	}
	log.Log().Info("[UnWatchStory] 返回成功响应", zap.Int64("storyId", req.GetStoryId()), zap.Int64("userId", req.GetUserId()))
	return &api.UnWatchStoryResponse{
		Code:    int32(api.ResponseCode_OK),
		Message: "OK",
	}, nil
}

func (s *StoryService) RenderStory(ctx context.Context, req *api.RenderStoryRequest) (*api.RenderStoryResponse, error) {
	log.Log().Info("[RenderStory] 入参", zap.Any("req", req))
	story, err := models.GetStory(ctx, req.StoryId)
	if err != nil {
		log.Log().Error("[RenderStory] 获取故事失败", zap.Error(err), zap.Int64("storyId", req.StoryId))
		return nil, err
	}
	if story.IsClose {
		log.Log().Info("[RenderStory] 故事已关闭", zap.Int64("storyId", req.StoryId))
		return &api.RenderStoryResponse{
			Code:    -1,
			Message: "story is closed",
		}, nil
	}
	genParams := new(models.StoryParams)
	if story.Params == "" {
		log.Log().Error("[RenderStory] story.Params为空", zap.Int64("storyId", req.StoryId))
		return &api.RenderStoryResponse{
			Code:    -1,
			Message: "story params is empty",
		}, nil
	}
	err = json.Unmarshal([]byte(story.Params), &genParams)
	if err != nil {
		log.Log().Error("[RenderStory] 解析story.Params失败", zap.Error(err), zap.Int64("storyId", req.StoryId))
		return nil, err
	}
	log.Log().Info("[RenderStory] 解析story.Params成功", zap.Any("genParams", genParams))
	storyGen := new(models.StoryGen)
	storyGenData, _ := json.Marshal(genParams)
	storyGen.LLmPlatform = "coze"
	storyGen.Params = string(storyGenData)
	storyGen.OriginID = req.StoryId
	storyGen.BoardID = 0
	storyGen.StartTime = time.Now().Unix()
	storyGen.TaskType = api.RenderType_RENDER_TYPE_STORYBOARD_TEXT
	storyGen.Status = 1
	storyGen.UserId = req.UserId
	storyGen.GenStatus = api.StoryGenStatus_STORY_GEN_STATUS_INIT
	storyGen.BoardID = req.BoardId
	storyGen.OriginID = req.StoryId
	storyGen.TaskId = uuid.New().String()
	exist, _ := models.GetStoryGensByStory(ctx, req.StoryId, 1)
	if len(exist) > 0 {
		existGen := new(api.RenderStoryDetail)
		json.Unmarshal([]byte(exist[0].Content), existGen)
		log.Log().Info("[RenderStory] 已有渲染任务，直接返回", zap.Any("existGen", existGen))
		return &api.RenderStoryResponse{
			Code:    0,
			Message: "story is rendering",
			Data:    existGen,
		}, nil
	}
	_, err = models.CreateStoryGen(ctx, storyGen)
	if err != nil {
		log.Log().Error("[RenderStory] 创建StoryGen失败", zap.Error(err))
		return nil, err
	}
	log.Log().Info("[RenderStory] 创建StoryGen成功", zap.String("uuid", storyGen.UUID))
	if err := (&models.UserProfile{UserId: req.UserId}).IncrementCreatedGenNum(ctx); err != nil {
		log.Log().Warn("[RenderStory] 增加用户生成次数失败", zap.Error(err), zap.Int64("userId", req.UserId))
	}
	renderDetail := new(api.RenderStoryDetail)
	renderStoryParams := coze.CozeStoryWriteParams{
		StoryTitle: story.Title,
		StoryDesc:  story.Origin,
	}
	start := time.Now()
	var (
		resp         = &api.RenderStoryResponse{}
		storyContent string
	)
	if req.RenderType == api.RenderType_RENDER_TYPE_TEXT_UNSPECIFIED {
		renderDetail.StoryId = req.StoryId
		renderDetail.BoardId = req.BoardId
		storyContent, err = s.cozeClient.StoryWrite(ctx, renderStoryParams)
		if err != nil {
			log.Log().Error("[RenderStory] 生成故事内容失败", zap.Error(err))
			return nil, err
		}
		log.Log().Info("[RenderStory] 生成故事内容成功", zap.String("storyContent", storyContent))
	} else if req.RenderType == api.RenderType_RENDER_TYPE_STORYSENCE {
		renderDetail.StoryId = req.StoryId
		renderDetail.BoardId = req.BoardId
		log.Log().Info("[RenderStory] 渲染类型为STORYSENCE")
	} else if req.RenderType == api.RenderType_RENDER_TYPE_STORYCHARACTERS {
		renderDetail.StoryId = req.StoryId
		renderDetail.BoardId = req.BoardId
		log.Log().Info("[RenderStory] 渲染类型为STORYCHARACTERS")
	}
	result := new(StoryInfo)
	cleanResult := utils.CleanLLmJsonResult(storyContent)
	err = json.Unmarshal([]byte(cleanResult), &result)
	if err != nil {
		log.Log().Error("[RenderStory] 解析生成结果失败", zap.Error(err))
		return nil, err
	}
	log.Log().Info("[RenderStory] 解析生成结果成功", zap.Any("result", result))
	renderDetail.Text = storyContent
	renderDetail.RenderType = req.RenderType
	renderDetail.Timecost = int32(time.Since(start).Seconds())
	renderDetail.Result = new(api.StoryInfo)
	storyInfo := &api.StoryInfo{
		StoryNameAndTheme: &api.StoryNameAndTheme{},
		StoryChapters:     make([]*api.ChapterInfo, 0),
	}
	if result.StoryNameAndTheme.Name != "" {
		storyInfo.StoryNameAndTheme.Name = result.StoryNameAndTheme.Name
	}
	if result.StoryNameAndTheme.Theme != "" {
		storyInfo.StoryNameAndTheme.Theme = result.StoryNameAndTheme.Theme
	}
	if result.StoryNameAndTheme.Description != "" {
		storyInfo.StoryNameAndTheme.Description = result.StoryNameAndTheme.Description
	}
	for _, chapter := range result.StoryChapters {
		apiChapter := &api.ChapterInfo{
			Id:      chapter.ID,
			Title:   chapter.Title,
			Content: chapter.Content,
		}
		storyInfo.StoryChapters = append(storyInfo.StoryChapters, apiChapter)
	}
	renderDetail.Result = storyInfo
	renderDetailData, _ := json.Marshal(renderDetail)
	storyGen.Content = string(renderDetailData)
	storyGen.GenStatus = api.StoryGenStatus_STORY_GEN_STATUS_FINISHED
	storyGen.FinishTime = time.Now().Unix()
	err = models.UpdateStoryGen(ctx, storyGen)
	if err != nil {
		log.Log().Error("[RenderStory] 更新StoryGen失败", zap.Error(err))
	}
	resp.Code = 0
	resp.Message = "OK"
	resp.Data = renderDetail
	log.Log().Info("[RenderStory] 返回成功响应", zap.Any("resp", resp))
	return resp, nil
}

func (s *StoryService) GetStoryRender(ctx context.Context, req *api.GetStoryRenderRequest) (*api.GetStoryRenderResponse, error) {
	log.Log().Info("[GetStoryRender] 入参", zap.Any("req", req))
	list, err := models.GetStoryGensByStory(ctx, req.GetStoryId(), 1)
	if err != nil {
		log.Log().Error("[GetStoryRender] 获取StoryGen列表失败", zap.Error(err), zap.Int64("storyId", req.GetStoryId()))
		return nil, err
	}
	if len(list) == 0 {
		log.Log().Warn("[GetStoryRender] 没有渲染任务", zap.Int64("storyId", req.GetStoryId()))
		return &api.GetStoryRenderResponse{
			Code:    -1,
			Message: "story is not rendering",
		}, nil
	}
	item := new(api.RenderStoryDetail)
	err = json.Unmarshal([]byte(list[0].Content), &item)
	if err != nil {
		log.Log().Error("[GetStoryRender] 解析渲染内容失败", zap.Error(err))
		return nil, err
	}
	log.Log().Info("[GetStoryRender] 返回成功响应", zap.Any("item", item))
	return &api.GetStoryRenderResponse{
		Code:    0,
		Message: "OK",
		Data: &api.GetStoryRenderResponse_Data{
			List: []*api.RenderStoryDetail{
				item,
			},
		},
	}, nil
}

func (s *StoryService) GetStoryContributors(ctx context.Context, req *api.GetStoryContributorsRequest) (*api.GetStoryContributorsResponse, error) {
	log.Log().Info("[GetStoryContributors] 入参", zap.Any("req", req))
	story, err := models.GetStory(ctx, req.GetStoryId())
	if err != nil {
		log.Log().Error("[GetStoryContributors] 获取故事失败", zap.Error(err), zap.Int64("storyId", req.GetStoryId()))
		return nil, err
	}
	if story.IsClose {
		log.Log().Warn("[GetStoryContributors] 故事已关闭", zap.Int64("storyId", req.GetStoryId()))
		return &api.GetStoryContributorsResponse{
			Code:    -1,
			Message: "story is closed",
		}, nil
	}
	contributors, err := models.GetStoryContributors(ctx, int64(story.ID))
	if err != nil {
		log.Log().Error("[GetStoryContributors] 获取贡献者失败", zap.Error(err), zap.Int64("storyId", req.GetStoryId()))
		return nil, err
	}
	apiContributors := make([]*api.StoryContributor, 0)
	for _, contributor := range contributors {
		apiContributor := new(api.StoryContributor)
		apiContributor.UserId = int64(contributor.ID)
		apiContributor.Username = contributor.Name
		apiContributor.Avatar = contributor.Avatar
		apiContributors = append(apiContributors, apiContributor)
	}
	log.Log().Info("[GetStoryContributors] 返回成功响应", zap.Any("contributors", apiContributors))
	return &api.GetStoryContributorsResponse{
		Code:    0,
		Message: "OK",
		Data: &api.GetStoryContributorsResponse_Data{
			List: apiContributors,
		},
	}, nil
}

func (s *StoryService) LikeStory(ctx context.Context, req *api.LikeStoryRequest) (*api.LikeStoryResponse, error) {
	log.Log().Info("[LikeStory] 入参", zap.Any("req", req))
	story, err := models.GetStory(ctx, req.GetStoryId())
	if err != nil {
		log.Log().Error("[LikeStory] 获取故事失败", zap.Error(err), zap.Int64("storyId", req.GetStoryId()))
		return nil, err
	}
	if story == nil {
		log.Log().Warn("[LikeStory] 故事不存在", zap.Int64("storyId", req.GetStoryId()))
		return &api.LikeStoryResponse{
			Code:    api.ResponseCode_OPERATION_FAILED,
			Message: "story not found",
		}, nil
	}
	created, err := models.LikeStory(ctx, req.GetUserId(), req.GetStoryId())
	if err != nil {
		log.Log().Error("[LikeStory] 创建点赞记录失败", zap.Error(err), zap.Int64("storyId", req.GetStoryId()), zap.Int64("userId", req.GetUserId()))
		return nil, err
	}
	if !created {
		log.Log().Info("[LikeStory] 已点赞，无需重复点赞", zap.Int64("storyId", req.GetStoryId()), zap.Int64("userId", req.GetUserId()))
		return &api.LikeStoryResponse{
			Code:    api.ResponseCode_OK,
			Message: "already liked",
		}, nil
	}
	story.LikeCount++
	userProfile := &models.UserProfile{UserId: int64(req.GetUserId())}
	if err := userProfile.IncrementLikedStoryNum(ctx); err != nil {
		log.Log().Error("[LikeStory] 增加用户点赞故事数失败", zap.Error(err), zap.Int64("userId", req.GetUserId()))
	}
	group := &models.Group{}
	group.ID = uint(story.GroupID)
	if err := group.GetByID(ctx); err != nil {
		log.Log().Error("[LikeStory] 获取小组信息失败", zap.Error(err), zap.Int64("groupId", int64(story.GroupID)))
		return nil, err
	}
	log.Log().Info("[LikeStory] 获取小组信息成功", zap.Any("group", group))
	active.GetActiveServer().WriteStoryActive(ctx, group, story, nil, nil, req.UserId, api.ActiveType_LikeStory)
	userStatusCache := cache.GetUserStatusCache()
	if err := userStatusCache.InvalidateStoryUserStatusCache(ctx, req.GetStoryId(), req.GetUserId()); err != nil {
		log.Log().Warn("[LikeStory] 清除故事用户状态缓存失败", zap.Error(err), zap.Int64("storyId", req.GetStoryId()), zap.Int64("userId", req.GetUserId()))
	}

	// 创建点赞通知（异步，避免阻塞主流程）
	go func() {
		bgCtx := context.Background()
		// 如果点赞的是自己的故事，不需要通知
		if story.CreatorID == req.GetUserId() {
			return
		}
		userID64 := req.GetUserId()
		storyID64 := req.GetStoryId()
		notifErr := llmchatservice.CreateSystemNotification(bgCtx, llmchatservice.CreateNotificationParams{
			UserID:        story.CreatorID,
			Type:          models.SystemNotificationTypeLike,
			Title:         "点赞提醒",
			Content:       fmt.Sprintf("有人点赞了你的故事《%s》", story.Title),
			RelatedUserID: &userID64,
			RelatedStoryID: &storyID64,
		})
		if notifErr != nil {
			log.Log().Error("[LikeStory] 创建点赞通知失败", zap.Error(notifErr), zap.Int64("storyId", req.GetStoryId()), zap.Int64("userId", req.GetUserId()))
		}
	}()

	log.Log().Info("[LikeStory] 返回成功响应", zap.Int64("storyId", req.GetStoryId()), zap.Int64("userId", req.GetUserId()))
	return &api.LikeStoryResponse{
		Code:    api.ResponseCode_OK,
		Message: "OK",
	}, nil
}

func (s *StoryService) UnLikeStory(ctx context.Context, req *api.UnLikeStoryRequest) (*api.UnLikeStoryResponse, error) {
	log.Log().Info("[UnLikeStory] 入参", zap.Any("req", req))
	removed, err := models.UnLikeStory(ctx, req.GetUserId(), req.GetStoryId())
	if err != nil {
		log.Log().Error("[UnLikeStory] 取消点赞故事失败", zap.Error(err), zap.Int64("storyId", req.GetStoryId()), zap.Int64("userId", req.GetUserId()))
		return nil, err
	}
	if !removed {
		log.Log().Warn("[UnLikeStory] 未点赞，无需取消", zap.Int64("storyId", req.GetStoryId()), zap.Int64("userId", req.GetUserId()))
		return &api.UnLikeStoryResponse{
			Code:    api.ResponseCode_OK,
			Message: "not liked",
		}, nil
	}
	userProfile := &models.UserProfile{UserId: int64(req.GetUserId())}
	if err := userProfile.DecrementLikedStoryNum(ctx); err != nil {
		log.Log().Error("[UnLikeStory] 减少用户点赞故事数失败", zap.Error(err), zap.Int64("userId", req.GetUserId()))
	}
	userStatusCache := cache.GetUserStatusCache()
	if err := userStatusCache.InvalidateStoryUserStatusCache(ctx, req.GetStoryId(), req.GetUserId()); err != nil {
		log.Log().Warn("[UnLikeStory] 清除故事用户状态缓存失败", zap.Error(err), zap.Int64("storyId", req.GetStoryId()), zap.Int64("userId", req.GetUserId()))
	}

	log.Log().Info("[UnLikeStory] 返回成功响应", zap.Int64("storyId", req.GetStoryId()), zap.Int64("userId", req.GetUserId()))
	return &api.UnLikeStoryResponse{
		Code:    api.ResponseCode_OK,
		Message: api.ResponseCode_OK.String(),
	}, nil
}

func (s *StoryService) SearchStories(ctx context.Context, req *api.SearchStoriesRequest) (*api.SearchStoriesResponse, error) {
	log.Log().Info("[SearchStories] 入参", zap.Any("req", req))
	stories, total, err := models.GetStoriesByName(ctx, req.GetKeyword(), int(req.GetOffset()), int(req.GetPageSize()))
	if err != nil {
		log.Log().Error("[SearchStories] 获取故事列表失败", zap.Error(err), zap.String("keyword", req.GetKeyword()))
		return nil, err
	}
	log.Log().Info("[SearchStories] 获取故事列表成功", zap.Int("count", len(stories)), zap.Int64("total", total))
	apiStories := make([]*api.Story, 0)
	for _, story := range stories {
		info := convert.ConvertStoryToApiStory(story)
		info.CurrentUserStatus, err = s.helper.GetStoryCurrentUserStatus(ctx, int64(story.ID))
		if err != nil {
			log.Log().Error("[SearchStories] 获取当前用户状态失败", zap.Error(err), zap.Int64("storyId", int64(story.ID)))
		}
		apiStories = append(apiStories, info)
	}
	log.Log().Info("[SearchStories] 返回成功响应", zap.Int("count", len(apiStories)), zap.Int64("total", total))
	return &api.SearchStoriesResponse{
		Code:     0,
		Message:  "OK",
		Stories:  apiStories,
		Total:    total,
		HaveMore: total > int64(req.GetOffset())*int64(req.GetPageSize()),
	}, nil
}

func (s *StoryService) SearchRoles(ctx context.Context, req *api.SearchRolesRequest) (*api.SearchRolesResponse, error) {
	log.Log().Info("[SearchRoles] 入参", zap.Any("req", req))
	roles, total, err := models.GetStoryRolesByName(ctx, req.GetKeyword(), req.GetStoryId(), int(req.GetOffset()), int(req.GetPageSize()))
	if err != nil {
		log.Log().Error("[SearchRoles] 获取角色列表失败", zap.Error(err), zap.String("keyword", req.GetKeyword()))
		return nil, err
	}
	log.Log().Info("[SearchRoles] 获取角色列表成功", zap.Int("count", len(roles)), zap.Int64("total", total))
	apiRoles := make([]*api.StoryRole, 0)
	for _, role := range roles {
		info := convert.ConvertStoryRoleToApiStoryRoleInfo(role)
		if role.CharacterDetail != "" {
			roleDetail := &CharacterDetailConverter{}
			err = json.Unmarshal([]byte(role.CharacterDetail), &roleDetail)
			if err != nil {
				log.Log().Error("[SearchRoles] 解析角色详情失败", zap.Error(err), zap.Int64("roleId", int64(role.ID)))
			}
			info.CharacterDetail = &api.CharacterDetail{
				Description:     roleDetail.Description,
				ShortTermGoal:   roleDetail.ShortTermGoal,
				LongTermGoal:    roleDetail.LongTermGoal,
				Personality:     roleDetail.Personality,
				Background:      roleDetail.Background,
				HandlingStyle:   roleDetail.HandlingStyle,
				CognitionRange:  roleDetail.CognitionRange,
				AbilityFeatures: roleDetail.AbilityFeatures,
				Appearance:      roleDetail.Appearance,
				DressPreference: roleDetail.DressPreference,
			}
		}
		info.CurrentUserStatus, err = s.helper.GetStoryRoleCurrentUserStatus(ctx, int64(role.ID))
		if err != nil {
			log.Log().Error("[SearchRoles] 获取当前用户状态失败", zap.Error(err), zap.Int64("roleId", int64(role.ID)))
		}
		info.LikeCount = role.LikeCount
		info.FollowCount = role.FollowCount
		info.StoryboardNum = role.StoryboardNum
		apiRoles = append(apiRoles, info)
	}
	log.Log().Info("[SearchRoles] 返回成功响应", zap.Int("count", len(apiRoles)), zap.Int64("total", total))
	return &api.SearchRolesResponse{
		Code:     0,
		Message:  "OK",
		Roles:    apiRoles,
		Total:    total,
		HaveMore: total > int64(req.GetOffset())*int64(req.GetPageSize()),
	}, nil
}

func (s *StoryService) GetStoryRoleList(ctx context.Context, req *api.GetStoryRoleListRequest) (*api.GetStoryRoleListResponse, error) {
	log.Log().Info("[GetStoryRoleList] 入参", zap.Any("req", req))
	roles, total, err := models.GetStoryRolesByName(ctx, req.GetSearchKey(), req.GetStoryId(), int(req.GetOffset()), int(req.GetPageSize()))
	if err != nil {
		log.Log().Error("[GetStoryRoleList] 获取角色列表失败", zap.Error(err), zap.String("searchKey", req.GetSearchKey()))
		return nil, err
	}
	log.Log().Info("[GetStoryRoleList] 获取角色列表成功", zap.Int("count", len(roles)))
	apiRoles := make([]*api.StoryRole, 0)
	for _, role := range roles {
		info := convert.ConvertStoryRoleToApiStoryRoleInfo(role)
		if role.CharacterDetail != "" {
			roleDetail := &CharacterDetailConverter{}
			err = json.Unmarshal([]byte(role.CharacterDetail), &roleDetail)
			if err != nil {
				log.Log().Error("[GetStoryRoleList] 解析角色详情失败", zap.Error(err), zap.Int64("roleId", int64(role.ID)))
			}
			info.CharacterDetail = &api.CharacterDetail{
				Description:     roleDetail.Description,
				ShortTermGoal:   roleDetail.ShortTermGoal,
				LongTermGoal:    roleDetail.LongTermGoal,
				Personality:     roleDetail.Personality,
				Background:      roleDetail.Background,
				HandlingStyle:   roleDetail.HandlingStyle,
				CognitionRange:  roleDetail.CognitionRange,
				AbilityFeatures: roleDetail.AbilityFeatures,
				Appearance:      roleDetail.Appearance,
				DressPreference: roleDetail.DressPreference,
			}
		}
		info.LikeCount = role.LikeCount
		info.FollowCount = role.FollowCount
		info.StoryboardNum = role.StoryboardNum
		info.CurrentUserStatus, err = s.helper.GetStoryRoleCurrentUserStatus(ctx, int64(role.ID))
		if err != nil {
			log.Log().Error("[GetStoryRoleList] 获取当前用户状态失败", zap.Error(err), zap.Int64("roleId", int64(role.ID)))
		}
		apiRoles = append(apiRoles, info)
	}
	log.Log().Info("[GetStoryRoleList] 返回成功响应", zap.Int("count", len(apiRoles)))
	return &api.GetStoryRoleListResponse{
		Code:     0,
		Message:  "OK",
		Roles:    apiRoles,
		Total:    total,
		HaveMore: total > int64(req.GetOffset())*int64(req.GetPageSize()),
	}, nil
}

func (s *StoryService) TrendingStory(ctx context.Context, req *api.TrendingStoryRequest) (*api.TrendingStoryResponse, error) {
	log.Log().Info("[TrendingStory] 入参", zap.Any("req", req))
	stories, total, err := models.GetTrendingStories(ctx, int(req.GetPageNumber()), int(req.GetPageSize()), req.GetStart(), req.GetEnd())
	if err != nil {
		log.Log().Error("[TrendingStory] 获取热门故事失败", zap.Error(err))
		return nil, err
	}
	if len(stories) == 0 {
		log.Log().Info("[TrendingStory] 无热门故事")
		return &api.TrendingStoryResponse{
			Code:    0,
			Message: "OK",
		}, nil
	}
	apiStories := make([]*api.Story, 0)
	for _, story := range stories {
		info := convert.ConvertStoryToApiStory(story)
		info.CurrentUserStatus, err = s.helper.GetStoryCurrentUserStatus(ctx, int64(story.ID))
		if err != nil {
			log.Log().Error("[TrendingStory] 获取当前用户状态失败", zap.Error(err), zap.Int64("storyId", int64(story.ID)))
		}
		apiStories = append(apiStories, info)
	}
	log.Log().Info("[TrendingStory] 返回成功响应", zap.Int("count", len(apiStories)))
	return &api.TrendingStoryResponse{
		Code:    0,
		Message: "OK",
		Data: &api.TrendingStoryResponse_Data{
			List:       apiStories,
			PageSize:   req.GetPageSize(),
			PageNumber: req.GetPageNumber(),
			HaveMore:   total > int64(req.GetPageNumber())*int64(req.GetPageSize()),
			Total:      total,
		},
	}, nil
}

func (s *StoryService) TrendingStoryRole(ctx context.Context, req *api.TrendingStoryRoleRequest) (*api.TrendingStoryRoleResponse, error) {
	log.Log().Info("[TrendingStoryRole] 入参", zap.Any("req", req))
	roles, total, err := models.GetTrendingStoryRoles(ctx, int(req.GetPageNumber()), int(req.GetPageSize()), req.GetStart(), req.GetEnd())
	if err != nil {
		log.Log().Error("[TrendingStoryRole] 获取热门角色失败", zap.Error(err))
		return nil, err
	}
	if len(roles) == 0 {
		log.Log().Info("[TrendingStoryRole] 无热门角色")
		return &api.TrendingStoryRoleResponse{
			Code:    0,
			Message: "OK",
		}, nil
	}
	apiRoles := make([]*api.StoryRole, 0)
	for _, role := range roles {
		info := convert.ConvertStoryRoleToApiStoryRoleInfo(role)
		info.CurrentUserStatus, err = s.helper.GetStoryRoleCurrentUserStatus(ctx, int64(role.ID))
		if err != nil {
			log.Log().Error("[TrendingStoryRole] 获取当前用户状态失败", zap.Error(err), zap.Int64("roleId", int64(role.ID)))
		}
		log.Log().Info("[TrendingStoryRole] 角色详情", zap.Any("role", role.CharacterDetail))
		if role.CharacterDetail != "" {
			roleDetail := &CharacterDetailConverter{}
			err = json.Unmarshal([]byte(role.CharacterDetail), &roleDetail)
			if err != nil {
				log.Log().Error("[TrendingStoryRole] 解析角色详情失败", zap.Error(err), zap.Int64("roleId", int64(role.ID)))
			}
			info.CharacterDetail = &api.CharacterDetail{
				Description:     roleDetail.Description,
				ShortTermGoal:   roleDetail.ShortTermGoal,
				LongTermGoal:    roleDetail.LongTermGoal,
				Personality:     roleDetail.Personality,
				Background:      roleDetail.Background,
				HandlingStyle:   roleDetail.HandlingStyle,
				CognitionRange:  roleDetail.CognitionRange,
				AbilityFeatures: roleDetail.AbilityFeatures,
				Appearance:      roleDetail.Appearance,
				DressPreference: roleDetail.DressPreference,
			}
		}
		info.LikeCount = role.LikeCount
		info.FollowCount = role.FollowCount
		info.StoryboardNum = role.StoryboardNum
		apiRoles = append(apiRoles, info)
	}
	resp := &api.TrendingStoryRoleResponse{
		Code:    0,
		Message: "OK",
		Data: &api.TrendingStoryRoleResponse_Data{
			List:       apiRoles,
			PageSize:   req.GetPageSize(),
			PageNumber: req.GetPageNumber(),
			HaveMore:   total > int64(req.GetPageNumber())*int64(req.GetPageSize()),
			Total:      total,
		},
	}
	log.Log().Info("[TrendingStoryRole] 返回成功响应", zap.Int("count", len(apiRoles)))
	return resp, nil
}

func (s *StoryService) GetStoryImageStyle(ctx context.Context, req *api.GetStoryImageStyleRequest) (*api.GetStoryImageStyleResponse, error) {
	log.Log().Info("[GetStoryImageStyle] 入参", zap.Any("req", req))
	return &api.GetStoryImageStyleResponse{
		Code:    0,
		Message: "OK",
		Style:   styles,
	}, nil
}

func (s *StoryService) UpdateStoryImageStyle(ctx context.Context, req *api.UpdateStoryImageStyleRequest) (*api.UpdateStoryImageStyleResponse, error) {
	log.Log().Info("[UpdateStoryImageStyle] 入参", zap.Any("req", req))
	story, err := models.GetStory(ctx, req.GetStoryId())
	if err != nil {
		log.Log().Error("[UpdateStoryImageStyle] 获取故事失败", zap.Error(err), zap.Int64("storyId", req.GetStoryId()))
		return nil, err
	}
	if story.CreatorID != req.GetUserId() {
		log.Log().Warn("[UpdateStoryImageStyle] 非创建者无权修改", zap.Int64("creatorId", story.CreatorID), zap.Int64("userId", req.GetUserId()))
		return nil, errors.New("not allowed")
	}
	if story.Style == req.GetStyle() {
		log.Log().Info("[UpdateStoryImageStyle] 样式未变，无需更新", zap.String("style", req.GetStyle()))
		return &api.UpdateStoryImageStyleResponse{
			Code:    0,
			Message: "OK",
		}, nil
	}
	story.Style = req.GetStyle()
	err = models.UpdateStoryStyle(ctx, req.GetStoryId(), req.GetStyle())
	if err != nil {
		log.Log().Error("[UpdateStoryImageStyle] 更新样式失败", zap.Error(err), zap.Int64("storyId", req.GetStoryId()))
		return nil, err
	}
	log.Log().Info("[UpdateStoryImageStyle] 更新样式成功", zap.String("style", req.GetStyle()))
	return &api.UpdateStoryImageStyleResponse{
		Code:    0,
		Message: "OK",
	}, nil
}

func (s *StoryService) UpdateStorySenceMaxNumber(ctx context.Context, req *api.UpdateStorySenceMaxNumberRequest) (*api.UpdateStorySenceMaxNumberResponse, error) {
	log.Log().Info("[UpdateStorySenceMaxNumber] 入参", zap.Any("req", req))
	story, err := models.GetStory(ctx, req.GetStoryId())
	if err != nil {
		log.Log().Error("[UpdateStorySenceMaxNumber] 获取故事失败", zap.Error(err), zap.Int64("storyId", req.GetStoryId()))
		return nil, err
	}
	if story.CreatorID != req.GetUserId() {
		log.Log().Warn("[UpdateStorySenceMaxNumber] 非创建者无权修改", zap.Int64("creatorId", story.CreatorID), zap.Int64("userId", req.GetUserId()))
		return nil, errors.New("not allowed")
	}
	if story.SenceMaxNumber == int64(req.GetMaxNumber()) {
		log.Log().Info("[UpdateStorySenceMaxNumber] 场景数未变，无需更新", zap.Int64("maxNumber", int64(req.GetMaxNumber())))
		return &api.UpdateStorySenceMaxNumberResponse{
			Code:    0,
			Message: "OK",
		}, nil
	}
	story.SenceMaxNumber = int64(req.GetMaxNumber())
	err = models.UpdateStorySenceMaxNumber(ctx, req.GetStoryId(), int64(req.GetMaxNumber()))
	if err != nil {
		log.Log().Error("[UpdateStorySenceMaxNumber] 更新场景数失败", zap.Error(err), zap.Int64("storyId", req.GetStoryId()))
		return nil, err
	}
	log.Log().Info("[UpdateStorySenceMaxNumber] 更新场景数成功", zap.Int64("maxNumber", int64(req.GetMaxNumber())))
	return &api.UpdateStorySenceMaxNumberResponse{
		Code:    0,
		Message: "OK",
	}, nil
}

func (s *StoryService) GetStoryParticipants(ctx context.Context, req *api.GetStoryParticipantsRequest) (*api.GetStoryParticipantsResponse, error) {
	log.Log().Info("[GetStoryParticipants] 入参", zap.Any("req", req))
	users, HaveMore, total, err := models.GetStoryParticipants(ctx, req.GetStoryId(), int(req.GetOffset()), int(req.GetPageSize()))
	if err != nil {
		log.Log().Error("[GetStoryParticipants] 获取参与者失败", zap.Error(err), zap.Int64("storyId", req.GetStoryId()))
		return nil, err
	}
	participantsUsers := make([]*api.UserInfo, 0)
	for _, user := range users {
		participantsUsers = append(participantsUsers, &api.UserInfo{
			UserId: int64(user.ID),
			Name:   user.Name,
			Avatar: user.Avatar,
			Desc:   user.ShortDesc,
		})
	}
	log.Log().Info("[GetStoryParticipants] 返回成功响应", zap.Int("count", len(participantsUsers)), zap.Bool("HaveMore", HaveMore), zap.Int64("total", total))
	return &api.GetStoryParticipantsResponse{
		Code:         0,
		Message:      "OK",
		Participants: participantsUsers,
		HaveMore:     HaveMore,
		Total:        total,
	}, nil
}

func (s *StoryService) UpdateStoryAvatar(ctx context.Context, req *api.UpdateStoryAvatarRequest) (*api.UpdateStoryAvatarResponse, error) {
	log.Log().Info("[UpdateStoryAvatar] 入参", zap.Any("req", req))
	story, err := models.GetStory(ctx, req.GetStoryId())
	if err != nil {
		log.Log().Error("[UpdateStoryAvatar] 获取故事失败", zap.Error(err), zap.Int64("storyId", req.GetStoryId()))
		return nil, err
	}
	if story.CreatorID != req.GetUserId() {
		log.Log().Warn("[UpdateStoryAvatar] 非创建者无权修改", zap.Int64("creatorId", story.CreatorID), zap.Int64("userId", req.GetUserId()))
		return nil, errors.New("not allowed")
	}
	story.Avatar = req.GetAvatarUrl()
	err = models.UpdateStoryAvatar(ctx, req.GetStoryId(), req.GetAvatarUrl())
	if err != nil {
		log.Log().Error("[UpdateStoryAvatar] 更新头像失败", zap.Error(err), zap.Int64("storyId", req.GetStoryId()))
		return nil, err
	}
	aliyunClient := aliyun.GetGlobalClient()
	err = aliyunClient.PersistImages(req.GetAvatarUrl())
	if err != nil {
		log.Log().Error("[UpdateStoryAvatar] 更新头像失败", zap.Error(err), zap.Int64("storyId", req.GetStoryId()))
		return nil, err
	}
	log.Log().Info("[UpdateStoryAvatar] 更新头像成功", zap.String("avatar", req.GetAvatarUrl()))
	return &api.UpdateStoryAvatarResponse{
		Code:    0,
		Message: "OK",
	}, nil
}

func (s *StoryService) UpdateStoryCover(ctx context.Context, req *api.UpdateStoryCoverRequest) (*api.UpdateStoryCoverResponse, error) {
	log.Log().Info("[UpdateStoryCover] 入参", zap.Any("req", req))
	story, err := models.GetStory(ctx, req.GetStoryId())
	if err != nil {
		log.Log().Error("[UpdateStoryCover] 获取故事失败", zap.Error(err), zap.Int64("storyId", req.GetStoryId()))
		return nil, err
	}
	if story.CreatorID != req.GetUserId() {
		log.Log().Warn("[UpdateStoryCover] 非创建者无权修改", zap.Int64("creatorId", story.CreatorID), zap.Int64("userId", req.GetUserId()))
		return nil, errors.New("not allowed")
	}
	story.Cover = req.GetCoverUrl()
	err = models.UpdateStoryCover(ctx, req.GetStoryId(), req.GetCoverUrl())
	if err != nil {
		log.Log().Error("[UpdateStoryCover] 更新封面失败", zap.Error(err), zap.Int64("storyId", req.GetStoryId()))
		return nil, err
	}
	aliyunClient := aliyun.GetGlobalClient()
	err = aliyunClient.PersistImages(req.GetCoverUrl())
	if err != nil {
		log.Log().Error("[UpdateStoryCover] 更新封面失败", zap.Error(err), zap.Int64("storyId", req.GetStoryId()))
		return nil, err
	}
	log.Log().Info("[UpdateStoryCover] 更新封面成功", zap.String("cover", req.GetCoverUrl()))
	return &api.UpdateStoryCoverResponse{
		Code:    0,
		Message: "OK",
	}, nil
}
