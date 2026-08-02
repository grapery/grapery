package story

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"

	api "github.com/grapery/common-protoc/gen"
	"github.com/grapery/grapery/models"
	"github.com/grapery/grapery/pkg/cache"
	"github.com/grapery/grapery/pkg/client"
	"github.com/grapery/grapery/pkg/cloud/aliyun"
	"github.com/grapery/grapery/pkg/cloud/coze"
	"github.com/grapery/grapery/pkg/cloud/doubao"
	"github.com/grapery/grapery/pkg/llmchat"
	"github.com/grapery/grapery/utils"
	"github.com/grapery/grapery/utils/compliance"
	"github.com/grapery/grapery/utils/convert"
	"github.com/grapery/grapery/utils/log"
)

// 用来管理故事的产场景描述，场景图片，场景参与角色信息
var storyroleServer StoryroleServer

func init() {
	storyroleServer = NewStoryroleService()
}

func GetStoryroleServer() StoryroleServer {
	if storyroleServer == nil {
		storyroleServer = NewStoryroleService()
	}
	return storyroleServer
}

func NewStoryroleService() *StoryroleService {
	return &StoryroleService{
		bailianClient:  client.NewAliyunClient(),
		doubaoClient:   client.NewDoubaoClient(),
		seedreamClient: doubao.NewSeedreamClient(""),
		helper:         GetStoryHelper(),
	}
}

type StoryroleService struct {
	bailianClient  *client.AliyunStoryClient
	doubaoClient   *client.DoubaoClient
	cozeClient     *coze.HuoShanCozeClient
	seedreamClient *doubao.SeedreamClient
	helper         HelperServer
}

type StoryroleServer interface {
	LikeStoryRole(ctx context.Context, req *api.LikeStoryRoleRequest) (*api.LikeStoryRoleResponse, error)
	UnLikeStoryRole(ctx context.Context, req *api.UnLikeStoryRoleRequest) (*api.UnLikeStoryRoleResponse, error)
	FollowStoryRole(ctx context.Context, req *api.FollowStoryRoleRequest) (*api.FollowStoryRoleResponse, error)
	UnFollowStoryRole(ctx context.Context, req *api.UnFollowStoryRoleRequest) (*api.UnFollowStoryRoleResponse, error)
	GetUserCreatedRoles(ctx context.Context, req *api.GetUserCreatedRolesRequest) (*api.GetUserCreatedRolesResponse, error)

	CreateStoryRole(ctx context.Context, req *api.CreateStoryRoleRequest) (*api.CreateStoryRoleResponse, error)
	GetStoryRoleDetail(ctx context.Context, req *api.GetStoryRoleDetailRequest) (*api.GetStoryRoleDetailResponse, error)
	RenderStoryRole(ctx context.Context, req *api.RenderStoryRoleRequest) (*api.RenderStoryRoleResponse, error)
	GetStoryRoleStories(ctx context.Context, req *api.GetStoryRoleStoriesRequest) (*api.GetStoryRoleStoriesResponse, error)
	GetStoryRoleStoryboards(ctx context.Context, req *api.GetStoryRoleStoryboardsRequest) (*api.GetStoryRoleStoryboardsResponse, error)
	CreateStoryRoleChat(ctx context.Context, req *api.CreateStoryRoleChatRequest) (*api.CreateStoryRoleChatResponse, error)
	ChatWithStoryRole(ctx context.Context, req *api.ChatWithStoryRoleRequest) (*api.ChatWithStoryRoleResponse, error)
	UpdateStoryRoleDetail(ctx context.Context, req *api.UpdateStoryRoleDetailRequest) (*api.UpdateStoryRoleDetailResponse, error)
	GetUserWithRoleChatList(ctx context.Context, req *api.GetUserWithRoleChatListRequest) (*api.GetUserWithRoleChatListResponse, error)
	GetUserChatWithRole(ctx context.Context, req *api.GetUserChatWithRoleRequest) (*api.GetUserChatWithRoleResponse, error)
	GetUserChatMessages(ctx context.Context, req *api.GetUserChatMessagesRequest) (*api.GetUserChatMessagesResponse, error)
	RenderStoryRoleContinuously(ctx context.Context, req *api.RenderStoryRoleContinuouslyRequest) (*api.RenderStoryRoleContinuouslyResponse, error)

	GenerateRoleDescription(ctx context.Context, req *api.GenerateRoleDescriptionRequest) (*api.GenerateRoleDescriptionResponse, error)
	UpdateRoleDescription(ctx context.Context, req *api.UpdateRoleDescriptionRequest) (*api.UpdateRoleDescriptionResponse, error)
	GenerateRolePrompt(ctx context.Context, req *api.GenerateRolePromptRequest) (*api.GenerateRolePromptResponse, error)
	UpdateRolePrompt(ctx context.Context, req *api.UpdateRolePromptRequest) (*api.UpdateRolePromptResponse, error)
	UpdateStoryRoleDescriptionDetail(ctx context.Context, req *api.UpdateStoryRoleDescriptionDetailRequest) (*api.UpdateStoryRoleDescriptionDetailResponse, error)
	UpdateStoryRolePrompt(ctx context.Context, req *api.UpdateStoryRolePromptRequest) (*api.UpdateStoryRolePromptResponse, error)

	UpdateStoryRolePoster(ctx context.Context, req *api.UpdateStoryRolePosterRequest) (*api.UpdateStoryRolePosterResponse, error)
	GenerateStoryRolePoster(ctx context.Context, req *api.GenerateStoryRolePosterRequest) (*api.GenerateStoryRolePosterResponse, error)

	GenerateRoleAvatar(ctx context.Context, req *api.GenerateRoleAvatarRequest) (*api.GenerateRoleAvatarResponse, error)
	UpdateStoryRoleAvator(ctx context.Context, req *api.UpdateStoryRoleAvatorRequest) (*api.UpdateStoryRoleAvatorResponse, error)
	UpdateStoryRole(ctx context.Context, req *api.UpdateStoryRoleRequest) (*api.UpdateStoryRoleResponse, error)
	RenderStoryRoleDetail(ctx context.Context, req *api.RenderStoryRoleDetailRequest) (*api.RenderStoryRoleDetailResponse, error)
	GenerateStoryRoleVideo(ctx context.Context, req *api.GenerateStoryRoleVideoRequest) (*api.GenerateStoryRoleVideoResponse, error)

	LikeStoryRolePoster(ctx context.Context, req *api.LikeStoryRolePosterRequest) (*api.LikeStoryRolePosterResponse, error)
	UnLikeStoryRolePoster(ctx context.Context, req *api.UnLikeStoryRolePosterRequest) (*api.UnLikeStoryRolePosterResponse, error)
	GetStoryRolePosterList(ctx context.Context, req *api.GetStoryRolePosterListRequest) (*api.GetStoryRolePosterListResponse, error)
}

func (s *StoryroleService) UpdateStoryRole(ctx context.Context, req *api.UpdateStoryRoleRequest) (*api.UpdateStoryRoleResponse, error) {
	// 函数入口日志
	log.Log().Info("[UpdateStoryRole] 开始更新故事角色",
		zap.Int64("roleId", req.Role.GetRoleId()),
		zap.String("characterName", req.Role.GetCharacterName()))

	// 获取角色信息
	log.Log().Info("[UpdateStoryRole] 开始获取角色信息", zap.Int64("roleId", req.Role.GetRoleId()))
	role, err := models.GetStoryRoleByID(ctx, req.Role.GetRoleId())
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			log.Log().Error("[UpdateStoryRole] 角色不存在",
				zap.Int64("roleId", req.Role.GetRoleId()))
			return &api.UpdateStoryRoleResponse{
				Code:    -1,
				Message: "role not found",
			}, nil
		}
		log.Log().Error("[UpdateStoryRole] 获取角色失败",
			zap.Int64("roleId", req.Role.GetRoleId()),
			zap.Error(err))
		return nil, err
	}
	log.Log().Info("[UpdateStoryRole] 成功获取角色信息",
		zap.Int64("roleId", req.Role.GetRoleId()),
		zap.String("characterName", role.CharacterName),
		zap.String("characterAvatar", role.CharacterAvatar))

	// 构建更新字段
	log.Log().Info("[UpdateStoryRole] 开始构建更新字段",
		zap.Int64("roleId", req.Role.GetRoleId()))

	needUpdateFields := make(map[string]interface{})

	if req.Role.GetCharacterName() != "" {
		needUpdateFields["character_name"] = req.Role.GetCharacterName()
		log.Log().Info("[UpdateStoryRole] 更新角色名称",
			zap.Int64("roleId", req.Role.GetRoleId()),
			zap.String("oldName", role.CharacterName),
			zap.String("newName", req.Role.GetCharacterName()))
	}
	if req.Role.GetCharacterAvatar() != "" {
		needUpdateFields["character_avatar"] = req.Role.GetCharacterAvatar()
		log.Log().Info("[UpdateStoryRole] 更新角色头像",
			zap.Int64("roleId", req.Role.GetRoleId()),
			zap.String("oldAvatar", role.CharacterAvatar),
			zap.String("newAvatar", req.Role.GetCharacterAvatar()))
	}
	if req.Role.GetCharacterId() != "" {
		needUpdateFields["character_id"] = req.Role.GetCharacterId()
		log.Log().Info("[UpdateStoryRole] 更新角色ID",
			zap.Int64("roleId", req.Role.GetRoleId()),
			zap.String("oldCharacterId", role.CharacterID),
			zap.String("newCharacterId", req.Role.GetCharacterId()))
	}
	if req.Role.GetCharacterType() != "" {
		needUpdateFields["character_type"] = req.Role.GetCharacterType()
		log.Log().Info("[UpdateStoryRole] 更新角色类型",
			zap.Int64("roleId", req.Role.GetRoleId()),
			zap.String("oldType", role.CharacterType),
			zap.String("newType", req.Role.GetCharacterType()))
	}
	if req.Role.GetCharacterPrompt() != "" {
		needUpdateFields["character_prompt"] = req.Role.GetCharacterPrompt()
		log.Log().Info("[UpdateStoryRole] 更新角色提示词",
			zap.Int64("roleId", req.Role.GetRoleId()),
			zap.String("oldPrompt", role.CharacterPrompt),
			zap.String("newPrompt", req.Role.GetCharacterPrompt()))
	}
	if len(req.Role.GetCharacterRefImages()) > 0 {
		needUpdateFields["character_ref_images"] = strings.Join(req.Role.GetCharacterRefImages(), ",")
		log.Log().Info("[UpdateStoryRole] 更新角色参考图片",
			zap.Int64("roleId", req.Role.GetRoleId()),
			zap.String("oldRefImages", role.CharacterRefImages),
			zap.String("newRefImages", strings.Join(req.Role.GetCharacterRefImages(), ",")))
	}
	if req.Role.GetCharacterDescription() != "" {
		needUpdateFields["character_description"] = req.Role.GetCharacterDescription()
		log.Log().Info("[UpdateStoryRole] 更新角色描述",
			zap.Int64("roleId", req.Role.GetRoleId()),
			zap.String("oldDescription", role.CharacterDescription),
			zap.String("newDescription", req.Role.GetCharacterDescription()))
	}

	log.Log().Info("[UpdateStoryRole] 更新字段构建完成",
		zap.Int64("roleId", req.Role.GetRoleId()),
		zap.Int("updateFieldCount", len(needUpdateFields)),
		zap.Any("updateFields", needUpdateFields))

	// 检查是否有需要更新的字段
	if len(needUpdateFields) == 0 {
		log.Log().Info("[UpdateStoryRole] 无需更新字段",
			zap.Int64("roleId", req.Role.GetRoleId()))
		return &api.UpdateStoryRoleResponse{
			Code:    0,
			Message: "no fields to update",
		}, nil
	}

	// 更新角色信息
	log.Log().Info("[UpdateStoryRole] 开始更新角色信息",
		zap.Int64("roleId", req.Role.GetRoleId()),
		zap.Int("updateFieldCount", len(needUpdateFields)))

	err = models.UpdateStoryRole(ctx, req.Role.GetRoleId(), needUpdateFields)
	if err != nil {
		log.Log().Error("[UpdateStoryRole] 更新角色信息失败",
			zap.Int64("roleId", req.Role.GetRoleId()),
			zap.Error(err))
		return nil, err
	}
	log.Log().Info("[UpdateStoryRole] 成功更新角色信息",
		zap.Int64("roleId", req.Role.GetRoleId()),
		zap.Int("updateFieldCount", len(needUpdateFields)))

	// 返回成功响应
	log.Log().Info("[UpdateStoryRole] 更新故事角色完成",
		zap.Int64("roleId", req.Role.GetRoleId()),
		zap.String("characterName", req.Role.GetCharacterName()))

	return &api.UpdateStoryRoleResponse{
		Code:    0,
		Message: "OK",
	}, nil
}

func (s *StoryroleService) RenderStoryRoleDetail(ctx context.Context, req *api.RenderStoryRoleDetailRequest) (*api.RenderStoryRoleDetailResponse, error) {
	// 函数入口日志
	log.Log().Info("[RenderStoryRoleDetail] 开始渲染故事角色详情",
		zap.Int64("roleId", req.GetRoleId()))

	// 获取故事角色
	log.Log().Info("[RenderStoryRoleDetail] 开始获取故事角色", zap.Int64("roleId", req.GetRoleId()))
	role, err := models.GetStoryRoleByID(ctx, req.GetRoleId())
	if err != nil {
		log.Log().Error("[RenderStoryRoleDetail] 获取故事角色失败",
			zap.Int64("roleId", req.GetRoleId()),
			zap.Error(err))
		return &api.RenderStoryRoleDetailResponse{
			Code:    -1,
			Message: "get story role failed",
		}, err
	}
	log.Log().Info("[RenderStoryRoleDetail] 成功获取故事角色",
		zap.Uint("roleId", role.ID),
		zap.String("characterName", role.CharacterName),
		zap.Int64("storyId", role.StoryID))

	// 获取故事
	log.Log().Info("[RenderStoryRoleDetail] 开始获取故事", zap.Int64("storyId", role.StoryID))
	story, err := models.GetStory(ctx, int64(role.StoryID))
	if err != nil {
		log.Log().Error("[RenderStoryRoleDetail] 获取故事失败",
			zap.Int64("storyId", role.StoryID),
			zap.Error(err))
		return nil, err
	}
	log.Log().Info("[RenderStoryRoleDetail] 成功获取故事",
		zap.Uint("storyId", story.ID),
		zap.String("storyTitle", story.Title),
		zap.Int("status", int(story.Status)))

	// 检查故事状态
	if story.IsClose {
		log.Log().Warn("[RenderStoryRoleDetail] 故事已关闭，无法渲染角色详情",
			zap.Uint("storyId", story.ID),
			zap.String("storyTitle", story.Title))
		return &api.RenderStoryRoleDetailResponse{
			Code:    -1,
			Message: "story is closed",
		}, nil
	}

	// 构建角色详情参数
	log.Log().Info("[RenderStoryRoleDetail] 开始构建角色详情参数")
	var storyroleParams = coze.CozeStoryRoleDetailParams{
		RoleName:    role.CharacterName,
		Description: role.CharacterDescription,
		StoryDesc:   story.ShortDesc,
		StoryName:   story.Title,
	}
	log.Log().Info("[RenderStoryRoleDetail] 角色详情参数构建完成",
		zap.String("roleName", storyroleParams.RoleName),
		zap.String("storyName", storyroleParams.StoryName))

	// 创建故事生成记录
	log.Log().Info("[RenderStoryRoleDetail] 开始创建故事生成记录")
	storyGen := new(models.StoryGen)
	storyGen.UUID = uuid.New().String()
	reqData, _ := json.Marshal(req)
	storyGenData, _ := json.Marshal(reqData)
	storyGen.LLmPlatform = "doubao"
	storyGen.Params = string(storyGenData)
	storyGen.OriginID = int64(role.StoryID)
	storyGen.StartTime = time.Now().Unix()
	storyGen.BoardID = 0
	storyGen.TaskType = api.RenderType_RENDER_TYPE_STORYCHARACTERS
	storyGen.GenStatus = api.StoryGenStatus_STORY_GEN_STATUS_INIT
	storyGen.TaskId = uuid.New().String()
	storyGen.UserId = req.GetUserId()

	log.Log().Info("[RenderStoryRoleDetail] 故事生成记录参数",
		zap.String("uuid", storyGen.UUID),
		zap.String("llmPlatform", storyGen.LLmPlatform),
		zap.Uint("originId", role.ID),
		zap.Int("genType", int(storyGen.TaskType)))

	_, err = models.CreateStoryGen(ctx, storyGen)
	if err != nil {
		log.Log().Error("[RenderStoryRoleDetail] 创建故事生成记录失败",
			zap.String("uuid", storyGen.UUID),
			zap.Error(err))
		return nil, err
	}
	log.Log().Info("[RenderStoryRoleDetail] 成功创建故事生成记录", zap.String("uuid", storyGen.UUID))

	// 调用LLM生成角色详情
	log.Log().Info("[RenderStoryRoleDetail] 开始调用LLM生成角色详情")
	result := new(CharacterDetail)
	start := time.Now()
	ret, err := s.cozeClient.StoryRoleDetail(ctx, storyroleParams)
	if err != nil {
		log.Log().Error("[RenderStoryRoleDetail] LLM生成角色详情失败",
			zap.String("uuid", storyGen.UUID),
			zap.Error(err))
		return nil, err
	}

	genTime := time.Since(start)
	log.Log().Info("[RenderStoryRoleDetail] LLM生成角色详情成功",
		zap.String("uuid", storyGen.UUID),
		zap.Duration("genTime", genTime),
		zap.String("rawResult", ret))

	// 清理和解析LLM结果
	log.Log().Info("[RenderStoryRoleDetail] 开始清理LLM结果")
	cleanResult := utils.CleanLLmJsonResult(ret)
	log.Log().Info("[RenderStoryRoleDetail] LLM结果清理完成",
		zap.String("uuid", storyGen.UUID),
		zap.String("cleanResult", cleanResult))

	err = json.Unmarshal([]byte(cleanResult), &result)
	if err != nil {
		log.Log().Error("[RenderStoryRoleDetail] 解析角色详情结果失败",
			zap.String("uuid", storyGen.UUID),
			zap.String("cleanResult", cleanResult),
			zap.Error(err))
		return nil, err
	}
	log.Log().Info("[RenderStoryRoleDetail] 成功解析角色详情结果",
		zap.String("uuid", storyGen.UUID),
		zap.String("description", result.Description))

	// 构建API角色详情
	log.Log().Info("[RenderStoryRoleDetail] 开始构建API角色详情")
	apiRoleDetail := new(api.StoryRole)
	apiRoleDetail.RoleId = int64(role.ID)
	apiRoleDetail.CharacterName = role.CharacterName
	apiRoleDetail.CharacterAvatar = role.CharacterAvatar
	apiRoleDetail.CharacterId = role.CharacterID
	apiRoleDetail.CharacterType = role.CharacterType
	apiRoleDetail.CharacterPrompt = role.CharacterPrompt
	apiRoleDetail.CharacterRefImages = strings.Split(role.CharacterRefImages, ",")
	apiRoleDetail.CharacterDescription = result.Description
	apiRoleDetail.CharacterDetail = &api.CharacterDetail{
		Description:   result.Description,
		ShortTermGoal: result.ShortTermGoal,
		LongTermGoal:  result.LongTermGoal,
		Personality:   result.Personality,
		Background:    result.Background,
	}
	log.Log().Info("[RenderStoryRoleDetail] API角色详情构建完成",
		zap.String("uuid", storyGen.UUID),
		zap.String("characterName", apiRoleDetail.CharacterName),
		zap.String("characterId", apiRoleDetail.CharacterId))

	// 更新故事生成记录
	log.Log().Info("[RenderStoryRoleDetail] 开始更新故事生成记录", zap.String("uuid", storyGen.UUID))
	storyGen.Content = cleanResult
	storyGen.FinishTime = time.Now().Unix()
	storyGen.GenStatus = api.StoryGenStatus_STORY_GEN_STATUS_FINISHED

	err = models.UpdateStoryGen(ctx, storyGen)
	if err != nil {
		log.Log().Error("[RenderStoryRoleDetail] 更新故事生成记录失败",
			zap.String("uuid", storyGen.UUID),
			zap.Error(err))
	} else {
		log.Log().Info("[RenderStoryRoleDetail] 成功更新故事生成记录", zap.String("uuid", storyGen.UUID))
	}

	// 返回成功响应
	log.Log().Info("[RenderStoryRoleDetail] 角色详情渲染完成",
		zap.Int64("roleId", req.GetRoleId()),
		zap.String("uuid", storyGen.UUID),
		zap.Duration("totalTime", time.Since(start)))

	return &api.RenderStoryRoleDetailResponse{
		Code:    0,
		Message: "OK",
		Role:    apiRoleDetail,
	}, nil
}

func (s *StoryroleService) UpdateStoryRoleAvator(ctx context.Context, req *api.UpdateStoryRoleAvatorRequest) (*api.UpdateStoryRoleAvatorResponse, error) {
	log.Log().Info("[UpdateStoryRoleAvator] 入参", zap.Any("req", req))
	roleinfo, err := models.GetStoryRoleByID(ctx, req.GetRoleId())
	if err != nil {
		log.Log().Error("[UpdateStoryRoleAvator] 获取角色失败", zap.Error(err), zap.Int64("roleId", req.GetRoleId()))
		return nil, err
	}
	if roleinfo.CreatorID != req.GetUserId() {
		log.Log().Warn("[UpdateStoryRoleAvator] 无权限修改角色头像", zap.Int64("roleCreatorId", roleinfo.CreatorID), zap.Int64("userId", req.GetUserId()))
		//return nil, errors.New("have no permission")
	}
	roleinfo.CharacterAvatar = req.GetAvator()
	err = models.UpdateStoryRole(ctx, int64(roleinfo.ID), map[string]interface{}{
		"character_avatar": req.GetAvator(),
	})
	if err != nil {
		log.Log().Error("[UpdateStoryRoleAvator] 更新角色头像失败", zap.Error(err), zap.Int64("roleId", req.GetRoleId()))
		return nil, err
	}
	aliyunClient := aliyun.GetGlobalClient()
	err = aliyunClient.PersistImages(req.GetAvator())
	if err != nil {
		log.Log().Error("[UpdateStoryRoleAvator] 更新角色头像失败", zap.Error(err), zap.Int64("roleId", req.GetRoleId()))
		return nil, err
	}
	log.Log().Info("[UpdateStoryRoleAvator] 更新角色头像成功", zap.Int64("roleId", req.GetRoleId()))
	return &api.UpdateStoryRoleAvatorResponse{
		Code:    0,
		Message: "OK",
	}, nil
}

func (s *StoryroleService) LikeStoryRole(ctx context.Context, req *api.LikeStoryRoleRequest) (*api.LikeStoryRoleResponse, error) {
	traceId := utils.GetTraceID(ctx)
	log.Log().Info("LikeStoryRole called", zap.String("traceId", traceId), zap.Any("req", req))
	// 日志覆盖：调用LikeStoryRole主流程
	created, err := models.LikeStoryRole(ctx, req.GetUserId(), req.GetStoryId(), req.GetRoleId())
	if err != nil {
		// 日志覆盖：like story role错误分支
		log.Log().Error("LikeStoryRole failed: like story role error", zap.String("traceId", traceId), zap.Error(err))
		return nil, err
	}
	if !created {
		log.Log().Info("LikeStoryRole: already liked", zap.String("traceId", traceId), zap.Int64("userId", req.GetUserId()), zap.Int64("roleId", req.GetRoleId()))
		resp := &api.LikeStoryRoleResponse{
			Code:    0,
			Message: "already liked",
		}
		return resp, nil
	}
	// 日志覆盖：like story role成功分支
	log.Log().Info("LikeStoryRole: like story role success", zap.String("traceId", traceId))
	err = models.IncreaseStoryRoleLikeCount(ctx, req.GetRoleId(), 1)
	if err != nil {
		// 日志覆盖：增加like count错误分支
		log.Log().Error("LikeStoryRole failed: increase like count error", zap.String("traceId", traceId), zap.Error(err))
		return nil, err
	}
	// 日志覆盖：增加like count成功分支
	log.Log().Info("LikeStoryRole: increase like count success", zap.String("traceId", traceId))

	// 清除相关缓存
	storyRoleCache := cache.GetStoryRoleCache()
	if err := storyRoleCache.DeleteStoryRoleDetail(ctx, req.GetRoleId()); err != nil {
		log.Log().Warn("LikeStoryRole: delete story role detail cache failed", zap.String("traceId", traceId), zap.Error(err))
	}
	if err := storyRoleCache.DeleteStoryRoleLikeCount(ctx, req.GetRoleId()); err != nil {
		log.Log().Warn("LikeStoryRole: delete story role like count cache failed", zap.String("traceId", traceId), zap.Error(err))
	}

	userProfile := &models.UserProfile{
		UserId: int64(req.GetUserId()),
	}
	err = userProfile.IncrementLikedRoleNum(ctx)
	if err != nil {
		// 日志覆盖：增加用户喜欢角色数失败分支
		log.Log().Warn("LikeStoryRole: increment liked role num failed", zap.String("traceId", traceId), zap.Error(err))
	} else {
		// 日志覆盖：增加用户喜欢角色数成功分支
		log.Log().Info("LikeStoryRole: increment liked role num success", zap.String("traceId", traceId))
	}
	resp := &api.LikeStoryRoleResponse{
		Code:    0,
		Message: "OK",
	}
	// 日志覆盖：LikeStoryRole最终成功返回分支
	log.Log().Info("LikeStoryRole success", zap.String("traceId", traceId), zap.Any("resp", resp))
	return resp, nil
}

func (s *StoryroleService) UnLikeStoryRole(ctx context.Context, req *api.UnLikeStoryRoleRequest) (*api.UnLikeStoryRoleResponse, error) {
	traceId := utils.GetTraceID(ctx)
	log.Log().Info("UnLikeStoryRole called", zap.String("traceId", traceId), zap.Any("req", req))
	// 日志覆盖：调用UnLikeStoryRole主流程
	removed, err := models.UnLikeStoryRole(ctx, req.GetUserId(), req.GetStoryId(), req.GetRoleId())
	if err != nil {
		// 日志覆盖：unlike story role错误分支
		log.Log().Error("UnLikeStoryRole failed: unlike story role error", zap.String("traceId", traceId), zap.Error(err))
		return nil, err
	}
	if !removed {
		log.Log().Info("UnLikeStoryRole: not liked", zap.String("traceId", traceId), zap.Int64("userId", req.GetUserId()), zap.Int64("roleId", req.GetRoleId()))
		resp := &api.UnLikeStoryRoleResponse{
			Code:    0,
			Message: "not liked",
		}
		return resp, nil
	}
	// 日志覆盖：unlike story role成功分支
	log.Log().Info("UnLikeStoryRole: unlike story role success", zap.String("traceId", traceId))
	err = models.DecreaseStoryRoleLikeCount(ctx, req.GetRoleId(), 1)
	if err != nil {
		// 日志覆盖：减少like count错误分支
		log.Log().Error("UnLikeStoryRole failed: decrease like count error", zap.String("traceId", traceId), zap.Error(err))
		return nil, err
	}
	// 日志覆盖：减少like count成功分支
	log.Log().Info("UnLikeStoryRole: decrease like count success", zap.String("traceId", traceId))

	// 清除相关缓存
	storyRoleCache := cache.GetStoryRoleCache()
	if err := storyRoleCache.DeleteStoryRoleDetail(ctx, req.GetRoleId()); err != nil {
		log.Log().Warn("UnLikeStoryRole: delete story role detail cache failed", zap.String("traceId", traceId), zap.Error(err))
	}
	if err := storyRoleCache.DeleteStoryRoleLikeCount(ctx, req.GetRoleId()); err != nil {
		log.Log().Warn("UnLikeStoryRole: delete story role like count cache failed", zap.String("traceId", traceId), zap.Error(err))
	}

	userProfile := &models.UserProfile{
		UserId: int64(req.GetUserId()),
	}
	err = userProfile.DecrementLikedRoleNum(ctx)
	if err != nil {
		// 日志覆盖：减少用户喜欢角色数失败分支
		log.Log().Warn("UnLikeStoryRole: decrement liked role num failed", zap.String("traceId", traceId), zap.Error(err))
	} else {
		// 日志覆盖：减少用户喜欢角色数成功分支
		log.Log().Info("UnLikeStoryRole: decrement liked role num success", zap.String("traceId", traceId))
	}
	resp := &api.UnLikeStoryRoleResponse{
		Code:    0,
		Message: "OK",
	}
	// 日志覆盖：UnLikeStoryRole最终成功返回分支
	log.Log().Info("UnLikeStoryRole success", zap.String("traceId", traceId), zap.Any("resp", resp))
	return resp, nil
}

func (s *StoryroleService) FollowStoryRole(ctx context.Context, req *api.FollowStoryRoleRequest) (*api.FollowStoryRoleResponse, error) {
	traceId := utils.GetTraceID(ctx)
	log.Log().Info("FollowStoryRole called", zap.String("traceId", traceId), zap.Any("req", req))
	// 日志覆盖：调用FollowStoryRole主流程
	created, err := models.WatchStoryRole(ctx, req.GetUserId(), req.GetStoryId(), req.GetRoleId())
	if err != nil {
		// 日志覆盖：watch story role错误分支
		log.Log().Error("FollowStoryRole failed: watch story role error", zap.String("traceId", traceId), zap.Error(err))
		return nil, err
	}
	if !created {
		// 日志覆盖：已关注分支
		log.Log().Info("FollowStoryRole: already watching", zap.String("traceId", traceId), zap.Int64("userId", req.GetUserId()), zap.Int64("roleId", req.GetRoleId()))
		resp := &api.FollowStoryRoleResponse{
			Code:    0,
			Message: "already following",
		}
		log.Log().Info("FollowStoryRole: already watching, return early", zap.String("traceId", traceId), zap.Any("resp", resp))
		return resp, nil
	}
	// 日志覆盖：watch story role成功分支
	log.Log().Info("FollowStoryRole: watch story role success", zap.String("traceId", traceId))
	err = models.IncreaseStoryRoleFollowCount(ctx, req.GetRoleId(), 1)
	if err != nil {
		// 日志覆盖：增加follow count错误分支
		log.Log().Error("FollowStoryRole failed: increase follow count error", zap.String("traceId", traceId), zap.Error(err))
		return nil, err
	}
	// 日志覆盖：增加follow count成功分支
	log.Log().Info("FollowStoryRole: increase follow count success", zap.String("traceId", traceId))

	// 清除相关缓存
	storyRoleCache := cache.GetStoryRoleCache()
	if err := storyRoleCache.DeleteStoryRoleDetail(ctx, req.GetRoleId()); err != nil {
		log.Log().Warn("FollowStoryRole: delete story role detail cache failed", zap.String("traceId", traceId), zap.Error(err))
	}
	if err := storyRoleCache.DeleteStoryRoleFollowCount(ctx, req.GetRoleId()); err != nil {
		log.Log().Warn("FollowStoryRole: delete story role follow count cache failed", zap.String("traceId", traceId), zap.Error(err))
	}

	userProfile := &models.UserProfile{
		UserId: int64(req.GetUserId()),
	}
	err = userProfile.IncrementWatchingStoryRoleNum(ctx)
	if err != nil {
		// 日志覆盖：增加用户关注角色数失败分支
		log.Log().Warn("FollowStoryRole: increment watching story role num failed", zap.String("traceId", traceId), zap.Error(err))
	} else {
		// 日志覆盖：增加用户关注角色数成功分支
		log.Log().Info("FollowStoryRole: increment watching story role num success", zap.String("traceId", traceId))
	}
	resp := &api.FollowStoryRoleResponse{
		Code:    0,
		Message: "OK",
	}
	// 日志覆盖：FollowStoryRole最终成功返回分支
	log.Log().Info("FollowStoryRole success", zap.String("traceId", traceId), zap.Any("resp", resp))
	return resp, nil
}

func (s *StoryroleService) UnFollowStoryRole(ctx context.Context, req *api.UnFollowStoryRoleRequest) (*api.UnFollowStoryRoleResponse, error) {
	traceId := utils.GetTraceID(ctx)
	log.Log().Info("UnFollowStoryRole called", zap.String("traceId", traceId), zap.Any("req", req))
	// 日志覆盖：调用UnFollowStoryRole主流程
	removed, err := models.UnWatchStoryRole(ctx, req.GetUserId(), req.GetStoryId(), req.GetRoleId())
	if err != nil {
		// 日志覆盖：get watch item错误分支
		log.Log().Error("UnFollowStoryRole failed: unwatch story role error", zap.String("traceId", traceId), zap.Error(err))
		return nil, err
	}
	if !removed {
		// 日志覆盖：未关注分支
		log.Log().Info("UnFollowStoryRole: not watching", zap.String("traceId", traceId), zap.Int64("userId", req.GetUserId()), zap.Int64("roleId", req.GetRoleId()))
		resp := &api.UnFollowStoryRoleResponse{
			Code:    0,
			Message: "not following",
		}
		log.Log().Info("UnFollowStoryRole: not watching, return early", zap.String("traceId", traceId), zap.Any("resp", resp))
		return resp, nil
	}
	err = models.DecreaseStoryRoleFollowCount(ctx, req.GetRoleId(), 1)
	if err != nil {
		// 日志覆盖：减少follow count错误分支
		log.Log().Error("UnFollowStoryRole failed: decrease follow count error", zap.String("traceId", traceId), zap.Error(err))
		return nil, err
	}
	// 日志覆盖：减少follow count成功分支
	log.Log().Info("UnFollowStoryRole: decrease follow count success", zap.String("traceId", traceId))

	// 清除相关缓存
	storyRoleCache := cache.GetStoryRoleCache()
	if err := storyRoleCache.DeleteStoryRoleDetail(ctx, req.GetRoleId()); err != nil {
		log.Log().Warn("UnFollowStoryRole: delete story role detail cache failed", zap.String("traceId", traceId), zap.Error(err))
	}
	if err := storyRoleCache.DeleteStoryRoleFollowCount(ctx, req.GetRoleId()); err != nil {
		log.Log().Warn("UnFollowStoryRole: delete story role follow count cache failed", zap.String("traceId", traceId), zap.Error(err))
	}

	userProfile := &models.UserProfile{
		UserId: int64(req.GetUserId()),
	}
	err = userProfile.DecrementWatchingStoryRoleNum(ctx)
	if err != nil {
		// 日志覆盖：减少用户关注角色数失败分支
		log.Log().Warn("UnFollowStoryRole: decrement watching story role num failed", zap.String("traceId", traceId), zap.Error(err))
	} else {
		// 日志覆盖：减少用户关注角色数成功分支
		log.Log().Info("UnFollowStoryRole: decrement watching story role num success", zap.String("traceId", traceId))
	}
	resp := &api.UnFollowStoryRoleResponse{
		Code:    0,
		Message: "OK",
	}
	// 日志覆盖：UnFollowStoryRole最终成功返回分支
	log.Log().Info("UnFollowStoryRole success", zap.String("traceId", traceId), zap.Any("resp", resp))
	return resp, nil
}

// 获取用户创建的角色
func (s *StoryroleService) GetUserCreatedRoles(ctx context.Context, req *api.GetUserCreatedRolesRequest) (*api.GetUserCreatedRolesResponse, error) {
	traceId := utils.GetTraceID(ctx)
	log.Log().Info("GetUserCreatedRoles called", zap.String("traceId", traceId), zap.Any("req", req))

	// 尝试从缓存获取用户创建的角色
	storyRoleCache := cache.GetStoryRoleCache()
	var roles []*models.StoryRole
	var total int64
	cachedRoles, cachedTotal, err := storyRoleCache.GetUserCreatedRoles(ctx, req.GetUserId(), int64(req.GetStoryId()), int(req.GetOffset()), int(req.GetPageSize()))
	if err == nil && cachedRoles != nil {
		log.Log().Info("GetUserCreatedRoles: get from cache success", zap.String("traceId", traceId), zap.Int64("userId", req.GetUserId()), zap.Int32("storyId", req.GetStoryId()))
		roles = cachedRoles
		total = cachedTotal
	} else {
		// 缓存未命中，从数据库获取
		log.Log().Debug("GetUserCreatedRoles: cache miss, get from database", zap.String("traceId", traceId), zap.Int64("userId", req.GetUserId()), zap.Int32("storyId", req.GetStoryId()))
		roles, total, err = models.GetUserCreatedRolesWithStoryId(ctx, int(req.GetUserId()), int(req.GetStoryId()), int(req.GetOffset()), int(req.GetPageSize()))
		if err != nil {
			log.Log().Error("GetUserCreatedRoles failed: get user created roles error", zap.String("traceId", traceId), zap.Error(err))
			return nil, err
		}

		// 将用户创建的角色存入缓存
		if err := storyRoleCache.SetUserCreatedRoles(ctx, req.GetUserId(), int64(req.GetStoryId()), int(req.GetOffset()), int(req.GetPageSize()), roles, total); err != nil {
			log.Log().Warn("GetUserCreatedRoles: set cache failed", zap.String("traceId", traceId), zap.Error(err))
		}
	}
	user, err := models.GetUserById(ctx, int64(req.GetUserId()))
	if err != nil {
		log.Log().Error("GetUserCreatedRoles failed: get user error", zap.String("traceId", traceId), zap.Error(err))
		return nil, err
	}
	apiRoles := make([]*api.StoryRole, 0)
	for _, role := range roles {
		if role.Status != 1 {
			log.Log().Debug("GetUserCreatedRoles: skip role with status!=1", zap.String("traceId", traceId), zap.Int64("role_id", int64(role.ID)))
			// 日志覆盖：跳过非激活状态角色
			continue
		}
		if role.Deleted == true {
			log.Log().Debug("GetUserCreatedRoles: skip deleted role", zap.String("traceId", traceId), zap.Int64("role_id", int64(role.ID)))
			// 日志覆盖：跳过已删除角色
			continue
		}
		apiRole := convert.ConvertStoryRoleToApiStoryRoleInfo(role)
		if role.CharacterDetail != "" {
			if err := json.Unmarshal([]byte(role.CharacterDetail), &apiRole.CharacterDetail); err != nil {
				log.Log().Warn("GetUserCreatedRoles: unmarshal character detail failed", zap.String("traceId", traceId), zap.Error(err), zap.Int64("role_id", int64(role.ID)))
			}
		}
		apiRole.LikeCount = role.LikeCount
		apiRole.FollowCount = role.FollowCount
		apiRole.StoryboardNum = role.StoryboardNum
		apiRole.Ctime = int64(role.CreateAt.Unix())
		apiRole.Mtime = int64(role.UpdateAt.Unix())
		apiRole.Creator = &api.UserInfo{
			UserId: int64(user.ID),
			Name:   utils.MaskContent(user.Name),
			Avatar: user.Avatar,
		}
		apiRoles = append(apiRoles, apiRole)
	}
	resp := &api.GetUserCreatedRolesResponse{
		Code:     0,
		Message:  "OK",
		Roles:    apiRoles,
		Total:    total,
		Offset:   int64(req.GetOffset()),
		PageSize: int64(req.GetPageSize()),
		HaveMore: total > int64(req.GetOffset()*req.GetPageSize()),
	}
	log.Log().Info("GetUserCreatedRoles success", zap.String("traceId", traceId), zap.Any("resp", resp))
	return resp, nil
}

func (s *StoryroleService) CreateStoryRole(ctx context.Context, req *api.CreateStoryRoleRequest) (*api.CreateStoryRoleResponse, error) {
	traceId := utils.GetTraceID(ctx)
	log.Log().Info("CreateStoryRole called", zap.String("traceId", traceId), zap.Any("req", req))
	story, err := models.GetStory(ctx, req.GetRole().GetStoryId())
	if err != nil {
		log.Log().Error("CreateStoryRole failed: get story error", zap.String("traceId", traceId), zap.Error(err))
		return nil, err
	}
	if story.Status == -1 {
		log.Log().Warn("CreateStoryRole: story is closed", zap.String("traceId", traceId), zap.Int64("story_id", int64(story.ID)))
		resp := &api.CreateStoryRoleResponse{
			Code:    -1,
			Message: "story is closed",
		}
		log.Log().Info("CreateStoryRole: story is closed, return early", zap.String("traceId", traceId), zap.Any("resp", resp))
		return resp, nil
	}
	role, err := models.GetStoryRoleByName(ctx, req.GetRole().GetCharacterName(), int64(story.ID))
	if err != nil && err != gorm.ErrRecordNotFound {
		log.Log().Error("CreateStoryRole failed: get story role by name error", zap.String("traceId", traceId), zap.Error(err))
		return nil, err
	}
	if role != nil {
		log.Log().Warn("CreateStoryRole: role already exists", zap.String("traceId", traceId), zap.String("role_name", utils.MaskContent(req.GetRole().GetCharacterName())))
		resp := &api.CreateStoryRoleResponse{
			Code:    -1,
			Message: "role already exists",
		}
		log.Log().Info("CreateStoryRole: role already exists, return early", zap.String("traceId", traceId), zap.Any("resp", resp))
		return resp, nil
	}
	isPass, err := compliance.TextCompliance(req.GetRole().GetCharacterName())
	if err != nil {
		log.Log().Error("CreateStoryRole: 角色名称合规检测失败", zap.String("traceId", traceId), zap.Error(err))
		return nil, err
	}
	if !isPass {
		log.Log().Warn("CreateStoryRole: 角色名称不符合合规规则", zap.String("traceId", traceId), zap.String("role_name", utils.MaskContent(req.GetRole().GetCharacterName())))
	}
	isPass, err = compliance.TextCompliance(req.GetRole().GetCharacterDescription())
	if err != nil {
		log.Log().Error("CreateStoryRole: 角色描述合规检测失败",
			zap.String("traceId", traceId),
			zap.Error(err))
		return nil, err
	}
	if !isPass {
		log.Log().Warn("CreateStoryRole: 角色描述不符合合规规则",
			zap.String("traceId", traceId),
			zap.String("role_description", utils.MaskContent(req.GetRole().GetCharacterDescription())))
	}
	newRole := new(models.StoryRole)
	newRole.CharacterName = req.GetRole().GetCharacterName()
	newRole.StoryID = int64(story.ID)
	newRole.CreatorID = req.GetRole().GetCreatorId()
	newRole.CharacterDescription = req.GetRole().GetCharacterDescription()
	newRole.CharacterAvatar = req.GetRole().GetCharacterAvatar()
	newRole.CharacterID = req.GetRole().GetCharacterId()
	newRole.CharacterType = req.GetRole().GetCharacterType()
	newRole.CharacterPrompt = req.GetRole().GetCharacterPrompt()
	newRole.CharacterRefImages = strings.Join(req.GetRole().GetCharacterRefImages(), ",")
	newRole.FollowCount = 1
	newRole.LikeCount = 1
	newRole.Status = 1
	newRole.CharacterDetail = "{}"
	roleId, err := models.CreateStoryRole(ctx, newRole)
	if err != nil {
		log.Log().Error("CreateStoryRole failed: create story role error", zap.String("traceId", traceId), zap.Error(err))
		return nil, err
	}
	userProfille := new(models.UserProfile)
	creatorID := req.GetRole().GetCreatorId()
	userProfille.UserId = creatorID
	err = userProfille.GetByUserId(ctx)
	if err != nil {
		log.Log().Error("CreateStoryRole failed: update user profile error", zap.String("traceId", traceId), zap.Error(err))
		return nil, err
	}
	if err := userProfille.IncrementCreatedRoleNum(ctx); err != nil {
		log.Log().Warn("CreateStoryRole: increment created role num failed", zap.String("traceId", traceId), zap.Error(err))
	}
	if err := userProfille.IncrementContributRoleNum(ctx); err != nil {
		log.Log().Warn("CreateStoryRole: increment contribut role num failed", zap.String("traceId", traceId), zap.Error(err))
	}
	err = models.CreateWatchRoleItem(ctx, int(creatorID), int64(story.ID), int64(roleId))
	if err != nil {
		log.Log().Error("CreateStoryRole failed: create watch story item error", zap.String("traceId", traceId), zap.Error(err))
		return nil, err
	}
	if err := userProfille.IncrementWatchingStoryRoleNum(ctx); err != nil {
		log.Log().Warn("CreateStoryRole: increment watching story role num failed", zap.String("traceId", traceId), zap.Error(err))
	}
	err = models.IncreaseStoryRoleNum(ctx, int64(story.ID), int64(roleId), 1)
	if err != nil {
		log.Log().Error("CreateStoryRole failed: increase story role num error", zap.String("traceId", traceId), zap.Error(err))
		return nil, err
	}
	log.Log().Info("CreateStoryRole success", zap.String("traceId", traceId), zap.String("role", newRole.String()))

	// 清除相关缓存
	storyRoleCache := cache.GetStoryRoleCache()
	if err := storyRoleCache.InvalidateStoryRoleListCache(ctx, int64(story.ID)); err != nil {
		log.Log().Warn("CreateStoryRole: invalidate story role list cache failed", zap.String("traceId", traceId), zap.Error(err))
	}
	if err := storyRoleCache.InvalidateUserCreatedRolesCache(ctx, req.GetRole().GetCreatorId()); err != nil {
		log.Log().Warn("CreateStoryRole: invalidate user created roles cache failed", zap.String("traceId", traceId), zap.Error(err))
	}

	roleDetail, err := s.GenerateRoleDescription(ctx, &api.GenerateRoleDescriptionRequest{
		RoleId: int64(roleId),
		UserId: req.GetUserId(),
	})
	if err != nil {
		log.Log().Error("CreateStoryRole failed: generate role description error", zap.String("traceId", traceId), zap.Error(err))
		return nil, err
	}
	s.UpdateStoryRoleDescriptionDetail(ctx, &api.UpdateStoryRoleDescriptionDetailRequest{
		StoryId:         int64(story.ID),
		RoleId:          int64(roleId),
		CharacterDetail: roleDetail.GetCharacterDetail(),
		UserId:          req.GetUserId(),
	})
	llmEngine := llmchat.GetLLMChatEngine()
	botId, err := llmEngine.CreateBot(ctx, req.GetRole().GetCreatorId(), int64(roleId), req.GetRole().GetCharacterName(), req.GetRole().GetCharacterAvatar())
	if err != nil {
		log.Log().Error("CreateStoryRole failed: create bot error", zap.String("traceId", traceId), zap.Error(err))
		return nil, err
	}
	log.Log().Info("CreateStoryRole: create bot success", zap.String("traceId", traceId), zap.String("botId", botId))
	resp := &api.CreateStoryRoleResponse{
		Code:    0,
		Message: "OK",
	}
	botId, err = llmEngine.PublishBot(ctx, req.GetRole().GetCreatorId(), int64(roleId), botId)
	if err != nil {
		log.Log().Error("CreateStoryRole failed: publish bot error", zap.String("traceId", traceId), zap.Error(err))
		return nil, err
	}
	log.Log().Info("CreateStoryRole: publish bot success", zap.String("traceId", traceId), zap.String("botId", botId))
	return resp, nil
}

func (s *StoryroleService) GetStoryRoleDetail(ctx context.Context, req *api.GetStoryRoleDetailRequest) (*api.GetStoryRoleDetailResponse, error) {
	traceId := utils.GetTraceID(ctx)
	log.Log().Info("GetStoryRoleDetail called", zap.String("traceId", traceId), zap.Any("req", req))

	// 尝试从缓存获取角色详情
	storyRoleCache := cache.GetStoryRoleCache()
	cachedRole, err := storyRoleCache.GetStoryRoleDetail(ctx, req.GetRoleId())
	if err == nil && cachedRole != nil {
		log.Log().Info("GetStoryRoleDetail: get from cache success", zap.String("traceId", traceId), zap.Int64("roleId", req.GetRoleId()))
	} else {
		// 缓存未命中，从数据库获取
		log.Log().Debug("GetStoryRoleDetail: cache miss, get from database", zap.String("traceId", traceId), zap.Int64("roleId", req.GetRoleId()))
		role, err := models.GetStoryRoleByID(ctx, req.GetRoleId())
		if err != nil {
			log.Log().Error("GetStoryRoleDetail failed: get story role detail error", zap.String("traceId", traceId), zap.Error(err))
			return nil, err
		}
		cachedRole = role

		// 将角色详情存入缓存
		if err := storyRoleCache.SetStoryRoleDetail(ctx, req.GetRoleId(), role); err != nil {
			log.Log().Warn("GetStoryRoleDetail: set cache failed", zap.String("traceId", traceId), zap.Error(err))
		}
	}

	role := cachedRole
	cu, err := s.helper.GetStoryRoleCurrentUserStatus(ctx, int64(role.ID))
	if err != nil {
		log.Log().Warn("GetStoryRoleDetail: get current user status failed", zap.String("traceId", traceId), zap.Error(err))
	}
	detail := &api.StoryRole{
		RoleId:               int64(role.ID),
		CharacterDescription: utils.MaskContent(role.CharacterDescription),
		CharacterName:        utils.MaskContent(role.CharacterName),
		CharacterAvatar:      role.CharacterAvatar,
		CharacterId:          role.CharacterID,
		StoryId:              int64(role.StoryID),
		CharacterType:        role.CharacterType,
		CharacterPrompt:      utils.MaskContent(role.CharacterPrompt),
		CharacterRefImages:   strings.Split(role.CharacterRefImages, ","),
		Ctime:                role.CreateAt.Unix(),
		Mtime:                role.UpdateAt.Unix(),
		CreatorId:            role.CreatorID,
		FollowCount:          role.FollowCount,
		LikeCount:            role.LikeCount,
		Status:               int32(role.Status),
		StoryboardNum:        role.StoryboardNum,
		CurrentUserStatus:    cu,
	}
	err = json.Unmarshal([]byte(role.CharacterDetail), &detail.CharacterDetail)
	if err != nil {
		log.Log().Error("GetStoryRoleDetail failed: unmarshal character detail error", zap.String("traceId", traceId), zap.Error(err))
		return nil, err
	}
	resp := &api.GetStoryRoleDetailResponse{
		Code:    0,
		Message: "OK",
		Info:    detail,
	}
	log.Log().Info("GetStoryRoleDetail success", zap.String("traceId", traceId), zap.Any("resp", resp))
	return resp, nil
}

func (s *StoryroleService) RenderStoryRole(ctx context.Context, req *api.RenderStoryRoleRequest) (*api.RenderStoryRoleResponse, error) {
	traceId := utils.GetTraceID(ctx)
	log.Log().Info("RenderStoryRole called", zap.String("traceId", traceId), zap.Any("req", req))
	role, err := models.GetStoryRoleByID(ctx, req.GetRoleId())
	if err != nil {
		log.Log().Error("RenderStoryRole failed: get story role by id error", zap.String("traceId", traceId), zap.Error(err))
		return nil, err
	}
	if role.CreatorID != req.GetUserId() {
		log.Log().Warn("RenderStoryRole: no permission", zap.String("traceId", traceId), zap.Int64("creator_id", role.CreatorID), zap.Int64("user_id", req.GetUserId()))
		return nil, errors.New("have no permission")
	}
	if role.Status != 1 {
		log.Log().Warn("RenderStoryRole: role is not ready", zap.String("traceId", traceId), zap.Int32("status", int32(role.Status)))
		return nil, errors.New("role is not ready")
	}
	story, err := models.GetStory(ctx, role.StoryID)
	if err != nil {
		log.Log().Error("RenderStoryRole failed: get story error", zap.String("traceId", traceId), zap.Error(err))
		return nil, err
	}
	roleParams := coze.CozeStoryRoleDetailParams{
		StoryName:   utils.MaskContent(story.Name),
		StoryDesc:   utils.MaskContent(story.ShortDesc),
		RoleName:    utils.MaskContent(role.CharacterName),
		Description: utils.MaskContent(req.GetPrompt()),
	}
	if roleParams.Description == "" {
		roleParams.Description = utils.MaskContent(role.CharacterDescription)
	}
	roleContent, err := s.cozeClient.StoryRoleDetail(ctx, roleParams)
	if err != nil {
		log.Log().Error("RenderStoryRole failed: get story role detail prompt error", zap.String("traceId", traceId), zap.Error(err))
		return nil, err
	}
	storyGen := new(models.StoryGen)
	storyGen.UUID = uuid.New().String()
	storyGen.LLmPlatform = "coze"
	storyGen.Params = req.String()
	storyGen.OriginID = req.GetRoleId()
	storyGen.StartTime = time.Now().Unix()
	storyGen.BoardID = 0
	storyGen.TaskType = api.RenderType_RENDER_TYPE_STORYCHARACTERS
	storyGen.Status = 1
	storyGen.UserId = req.UserId
	storyGen.GenStatus = api.StoryGenStatus_STORY_GEN_STATUS_INIT
	storyGen.OriginID = role.StoryID
	storyGen.TaskId = uuid.New().String()
	_, err = models.CreateStoryGen(ctx, storyGen)
	if err != nil {
		log.Log().Error("RenderStoryRole failed: create story gen error", zap.String("traceId", traceId), zap.Error(err))
		return nil, err
	}
	var renderDetail = new(api.RenderStoryRoleDetail)
	result := new(CharacterDetail)
	cleanResult := utils.CleanLLmJsonResult(roleContent)
	err = json.Unmarshal([]byte(cleanResult), &result)
	if err != nil {
		log.Log().Error("RenderStoryRole failed: unmarshal gen result error", zap.String("traceId", traceId), zap.Error(err))
		return nil, err
	}
	log.Log().Info("RenderStoryRole: cleaned LLM result", zap.String("traceId", traceId), zap.String("content", cleanResult))
	storyGen.Content = cleanResult
	storyGen.FinishTime = time.Now().Unix()
	storyGen.GenStatus = api.StoryGenStatus_STORY_GEN_STATUS_FINISHED
	renderDetail.Background = result.Background
	renderDetail.Appearance = result.Appearance
	renderDetail.Personality = result.Personality
	renderDetail.AbilityFeatures = result.AbilityFeatures
	renderDetail.RoleDescription = result.Description
	renderDetail.RoleGoal = result.LongTermGoal
	renderDetail.RoleBehavior = result.HandlingStyle
	renderDetail.Appearance = result.Appearance
	renderDetail.Personality = result.Personality
	renderDetail.AbilityFeatures = result.AbilityFeatures
	err = models.UpdateStoryGen(ctx, storyGen)
	if err != nil {
		log.Log().Warn("RenderStoryRole: update story gen failed", zap.String("traceId", traceId), zap.Error(err))
	} else {
		log.Log().Info("RenderStoryRole: update story gen success", zap.String("traceId", traceId))
	}
	resp := &api.RenderStoryRoleResponse{
		Code:    0,
		Message: "OK",
		Detail:  renderDetail,
	}
	log.Log().Info("RenderStoryRole success", zap.String("traceId", traceId), zap.Any("resp", resp))
	return resp, nil
}

// 获取角色故事
func (s *StoryroleService) GetStoryRoleStories(ctx context.Context, req *api.GetStoryRoleStoriesRequest) (*api.GetStoryRoleStoriesResponse, error) {
	traceId := utils.GetTraceID(ctx)
	log.Log().Info("GetStoryRoleStories called", zap.String("traceId", traceId), zap.Any("req", req))
	// 当前未实现，直接返回nil
	log.Log().Warn("GetStoryRoleStories: not implemented", zap.String("traceId", traceId))
	return nil, nil
}

// 获取角色故事板
func (s *StoryroleService) GetStoryRoleStoryboards(ctx context.Context, req *api.GetStoryRoleStoryboardsRequest) (*api.GetStoryRoleStoryboardsResponse, error) {
	traceId := utils.GetTraceID(ctx)
	log.Log().Info("GetStoryRoleStoryboards called", zap.String("traceId", traceId), zap.Any("req", req))

	// 尝试从缓存获取角色故事板
	storyRoleCache := cache.GetStoryRoleCache()
	var boards []*models.StoryBoard
	var total int64
	cachedBoards, cachedTotal, err := storyRoleCache.GetStoryRoleStoryboards(ctx, req.GetRoleId(), int(req.GetOffset()), int(req.GetPageSize()))
	if err == nil && cachedBoards != nil {
		log.Log().Info("GetStoryRoleStoryboards: get from cache success", zap.String("traceId", traceId), zap.Int64("roleId", req.GetRoleId()))
		boards = cachedBoards
		total = cachedTotal
	} else {
		// 缓存未命中，从数据库获取
		log.Log().Debug("GetStoryRoleStoryboards: cache miss, get from database", zap.String("traceId", traceId), zap.Int64("roleId", req.GetRoleId()))
		boards, total, err = models.GetStoryBoardsByRoleID(ctx, req.GetRoleId(), int(req.GetOffset()), int(req.GetPageSize()))
		if err != nil && err != gorm.ErrRecordNotFound {
			log.Log().Error("GetStoryRoleStoryboards failed: get storyboards error", zap.String("traceId", traceId), zap.Error(err))
			return nil, err
		}

		// 将角色故事板存入缓存
		if err := storyRoleCache.SetStoryRoleStoryboards(ctx, req.GetRoleId(), int(req.GetOffset()), int(req.GetPageSize()), boards, total); err != nil {
			log.Log().Warn("GetStoryRoleStoryboards: set cache failed", zap.String("traceId", traceId), zap.Error(err))
		}
	}
	if err == gorm.ErrRecordNotFound {
		log.Log().Info("GetStoryRoleStoryboards: no storyboards found", zap.String("traceId", traceId))
		resp := &api.GetStoryRoleStoryboardsResponse{
			Code:    0,
			Message: "OK",
		}
		log.Log().Info("GetStoryRoleStoryboards: no storyboards, return early", zap.String("traceId", traceId), zap.Any("resp", resp))
		return resp, nil
	}
	if len(boards) == 0 {
		log.Log().Info("GetStoryRoleStoryboards: empty storyboards list", zap.String("traceId", traceId))
		resp := &api.GetStoryRoleStoryboardsResponse{
			Code:    0,
			Message: "OK",
		}
		log.Log().Info("GetStoryRoleStoryboards: empty list, return early", zap.String("traceId", traceId), zap.Any("resp", resp))
		return resp, nil
	}
	targetStoryIds := make([]int64, 0)
	for _, board := range boards {
		targetStoryIds = append(targetStoryIds, int64(board.StoryID))
	}
	stories, err := models.GetStoriesByIDs(ctx, targetStoryIds)
	if err != nil {
		log.Log().Error("GetStoryRoleStoryboards failed: get stories by ids error", zap.String("traceId", traceId), zap.Error(err))
		return nil, err
	}
	storiesSummary := make(map[int64]*api.StorySummaryInfo)
	for _, story := range stories {
		if story.Status != 1 {
			log.Log().Debug("GetStoryRoleStoryboards: skip story with status!=1", zap.String("traceId", traceId), zap.Int64("story_id", int64(story.ID)))
			// 日志覆盖：跳过非激活状态故事
			continue
		}
		if story.Deleted == true {
			log.Log().Debug("GetStoryRoleStoryboards: skip deleted story", zap.String("traceId", traceId), zap.Int64("story_id", int64(story.ID)))
			// 日志覆盖：跳过已删除故事
			continue
		}
		if _, ok := storiesSummary[int64(story.ID)]; ok {
			log.Log().Debug("GetStoryRoleStoryboards: skip duplicate story", zap.String("traceId", traceId), zap.Int64("story_id", int64(story.ID)))
			continue
		}
		storyItem := &api.StorySummaryInfo{
			StoryId:          int64(story.ID),
			StoryTitle:       utils.MaskContent(story.Name),
			StoryDescription: utils.MaskContent(story.ShortDesc),
			StoryCover:       "",
			StoryAvatar:      story.Avatar,
		}
		if storyItem.StoryTitle == "" {
			storyItem.StoryTitle = utils.MaskContent(story.Title)
		}
		storiesSummary[int64(story.ID)] = storyItem
	}
	apiBoards := make([]*api.StoryBoardActive, 0)
	for _, board := range boards {
		creator, err := models.GetUserById(ctx, int64(board.CreatorID))
		if err != nil {
			log.Log().Error("GetStoryRoleStoryboards failed: get user by id error", zap.String("traceId", traceId), zap.Error(err))
			return nil, err
		}
		boardsItem := convert.ConvertStoryBoardToApiStoryBoard(board)
		apiBoards = append(apiBoards, &api.StoryBoardActive{
			Storyboard:        boardsItem,
			TotalLikeCount:    int64(board.LikeNum),
			TotalCommentCount: int64(board.CommentNum),
			TotalShareCount:   int64(board.ShareNum),
			TotalForkCount:    int64(board.ForkNum),
			Mtime:             board.UpdateAt.Unix(),
			Creator: &api.StoryBoardActiveUser{
				UserId:     int64(creator.ID),
				UserName:   utils.MaskContent(creator.Name),
				UserAvatar: creator.Avatar,
			},
			Summary: storiesSummary[int64(board.StoryID)],
		})
	}
	resp := &api.GetStoryRoleStoryboardsResponse{
		Code:              0,
		Message:           "OK",
		Storyboardactives: apiBoards,
		Total:             int64(len(apiBoards)),
		HaveMore:          total > int64(req.GetOffset())*int64(req.GetPageSize()),
	}
	log.Log().Info("GetStoryRoleStoryboards success", zap.String("traceId", traceId), zap.Any("resp", resp))
	return resp, nil
}

// 创建角色聊天
func (s *StoryroleService) CreateStoryRoleChat(ctx context.Context, req *api.CreateStoryRoleChatRequest) (*api.CreateStoryRoleChatResponse, error) {
	traceId := utils.GetTraceID(ctx)
	log.Log().Info("CreateStoryRoleChat called", zap.String("traceId", traceId), zap.Any("req", req))
	if req.GetUserId() == 0 || req.GetRoleId() == 0 {
		log.Log().Warn("CreateStoryRoleChat failed: invalid user id or role id", zap.String("traceId", traceId), zap.Any("userId", req.GetUserId()), zap.Any("roleId", req.GetRoleId()))
		return nil, errors.New("invalid user id or role id")
	}
	existChatCtx, err := models.GetChatContextByUserIDAndRoleID(ctx, int64(req.GetUserId()), req.GetRoleId())
	if err != nil && err != gorm.ErrRecordNotFound {
		log.Log().Error("CreateStoryRoleChat failed: get user chat context error", zap.String("traceId", traceId), zap.Error(err))
		return nil, err
	}
	if existChatCtx != nil && existChatCtx.Status == 1 {
		log.Log().Info("CreateStoryRoleChat: chat context already exists", zap.String("traceId", traceId), zap.Int64("chat_id", int64(existChatCtx.ID)))
		resp := &api.CreateStoryRoleChatResponse{
			Code:    0,
			Message: "OK",
			ChatContext: &api.ChatContext{
				ChatId:         int64(existChatCtx.ID),
				UserId:         int64(existChatCtx.UserID),
				RoleId:         int64(existChatCtx.RoleID),
				LastUpdateTime: existChatCtx.UpdateAt.Unix(),
			},
		}
		log.Log().Info("CreateStoryRoleChat: chat context already exists, return early", zap.String("traceId", traceId), zap.Any("resp", resp))
		return resp, nil
	}
	chatContext := new(models.ChatContext)
	chatContext.UserID = int64(req.GetUserId())
	chatContext.RoleID = req.GetRoleId()
	chatContext.Title = "聊天消息"
	chatContext.Content = ""
	chatContext.Status = 1
	err = models.CreateChatContext(ctx, chatContext)
	if err != nil {
		log.Log().Error("CreateStoryRoleChat failed: create chat context error", zap.String("traceId", traceId), zap.Error(err))
		return nil, err
	}
	resp := &api.CreateStoryRoleChatResponse{
		Code:    0,
		Message: "OK",
		ChatContext: &api.ChatContext{
			ChatId:         int64(chatContext.ID),
			UserId:         int64(chatContext.UserID),
			RoleId:         int64(chatContext.RoleID),
			LastUpdateTime: chatContext.UpdateAt.Unix(),
		},
	}
	log.Log().Info("CreateStoryRoleChat success", zap.String("traceId", traceId), zap.Any("resp", resp))
	return resp, nil
}

// 角色聊天
func (s *StoryroleService) ChatWithStoryRole(ctx context.Context, req *api.ChatWithStoryRoleRequest) (*api.ChatWithStoryRoleResponse, error) {
	traceId := utils.GetTraceID(ctx)
	log.Log().Info("ChatWithStoryRole called", zap.String("traceId", traceId), zap.Any("req", req))
	chatCtx, err := models.GetChatContextByUserIDAndRoleID(ctx, int64(req.GetUserId()), int64(req.GetRoleId()))
	if err != nil && err != gorm.ErrRecordNotFound {
		log.Log().Error("ChatWithStoryRole failed: get user chat context error", zap.String("traceId", traceId), zap.Error(err))
		return nil, err
	}
	if err == gorm.ErrRecordNotFound {
		// 创建聊天上下文
		log.Log().Info("ChatWithStoryRole: chat context not found, create new", zap.String("traceId", traceId))
		chatCtx = new(models.ChatContext)
		chatCtx.UserID = int64(req.GetUserId())
		chatCtx.RoleID = int64(req.GetRoleId())
		chatCtx.Title = "聊天消息"
		chatCtx.Content = ""
		chatCtx.Status = 1
		err = models.CreateChatContext(ctx, chatCtx)
		if err != nil {
			log.Log().Error("ChatWithStoryRole failed: create chat context error", zap.String("traceId", traceId), zap.Error(err))
			return nil, err
		}
	}
	log.Log().Info("ChatWithStoryRole: chat context found or created", zap.String("traceId", traceId), zap.Int64("chat_id", int64(chatCtx.ID)))
	fmt.Println("ChatWithStoryRole req ", req.String())
	reply := make([]*api.ChatMessage, 0)
	for _, message := range req.Messages {
		chatMessage := new(models.ChatMessage)
		chatMessage.ChatContextID = int64(chatCtx.ID)
		chatMessage.UserID = int64(message.GetUserId())
		chatMessage.Content = message.GetMessage()
		chatMessage.Status = 1
		chatMessage.RoleID = int64(message.GetRoleId())
		chatMessage.Sender = int64(message.GetSender())
		chatMessage.UUID = message.GetUuid()
		err = models.CreateChatMessage(ctx, chatMessage)
		if err != nil {
			log.Log().Error("ChatWithStoryRole failed: create chat message error", zap.String("traceId", traceId), zap.Error(err))
			return nil, err
		}
		reply = append(reply, convert.ConvertChatMessageToApiChatMessage(chatMessage))
		// AI角色回复
		roleInfo, err := models.GetStoryRoleByID(ctx, int64(req.GetRoleId()))
		if err != nil {
			log.Log().Error("ChatWithStoryRole failed: get story role by id error", zap.String("traceId", traceId), zap.Error(err))
			return nil, err
		}
		characterDetail := &CharacterDetailConverter{}
		if err := json.Unmarshal([]byte(roleInfo.CharacterDetail), characterDetail); err != nil {
			log.Log().Warn("ChatWithStoryRole: unmarshal character detail failed", zap.String("traceId", traceId), zap.Error(err))
		}
		var chatParams = &client.ChatWithRoleParams{
			MessageContent: message.GetMessage(),
			Role:           characterDetail.ToPrompt(),
			SenseDesc:      "", // sence
			RolePositive:   "", // 角色的描述
			RoleNegative:   "",
			RequestId:      message.GetUuid(),
			UserId:         fmt.Sprintf("grapery_chat_ctx_%d_user_%d", chatCtx.ID, chatCtx.UserID),
		}
		chatResp, err := s.bailianClient.ChatWithRole(ctx, chatParams)
		if err != nil {
			log.Log().Error("ChatWithStoryRole failed: chat with role error", zap.String("traceId", traceId), zap.Error(err))
			return nil, err
		}
		roleReplyMessage := new(models.ChatMessage)
		roleReplyMessage.ChatContextID = int64(chatCtx.ID)
		roleReplyMessage.UserID = int64(message.GetUserId())
		roleReplyMessage.Content = chatResp.Content
		roleReplyMessage.Status = 1
		roleReplyMessage.RoleID = int64(message.GetRoleId())
		roleReplyMessage.Sender = int64(message.GetRoleId())
		roleReplyMessage.UUID = message.GetUuid()
		err = models.CreateChatMessage(ctx, roleReplyMessage)
		if err != nil {
			log.Log().Error("ChatWithStoryRole failed: create chat message error", zap.String("traceId", traceId), zap.Error(err))
			return nil, err
		}
		reply = append(reply, convert.ConvertChatMessageToApiChatMessage(roleReplyMessage))
	}
	resp := &api.ChatWithStoryRoleResponse{
		Code:          0,
		Message:       "OK",
		ReplyMessages: reply,
	}
	log.Log().Info("ChatWithStoryRole success", zap.String("traceId", traceId), zap.Any("resp", resp))
	return resp, nil
}

// 获取角色聊天列表
func (s *StoryroleService) GetUserWithRoleChatList(ctx context.Context, req *api.GetUserWithRoleChatListRequest) (*api.GetUserWithRoleChatListResponse, error) {
	traceId := utils.GetTraceID(ctx)
	log.Log().Info("GetUserWithRoleChatList called", zap.String("traceId", traceId), zap.Any("req", req))
	log.Log().Info("get user with role chat list", zap.Any("req", req.String()))
	chatCtxs, total, err := models.GetChatContextByUserID(ctx, int64(req.GetUserId()), int(req.GetOffset()), int(req.GetPageSize()))
	if err != nil {
		log.Log().Error("GetUserWithRoleChatList failed: get user chat context error", zap.String("traceId", traceId), zap.Error(err))
		return nil, err
	}
	log.Log().Info("get user chat context success", zap.Any("total", total), zap.Any("chatCtxs", len(chatCtxs)))
	apiChatCtxs := make([]*api.ChatContext, 0)
	for _, chatCtx := range chatCtxs {
		if chatCtx.UserID == 0 || chatCtx.RoleID == 0 {
			log.Log().Error("GetUserWithRoleChatList failed: invalid chat context", zap.String("traceId", traceId), zap.Any("chatCtx", chatCtx))
			// 日志覆盖：跳过无效聊天上下文
			continue
		}
		user, err := models.GetUserById(ctx, int64(chatCtx.UserID))
		if err != nil {
			log.Log().Error("GetUserWithRoleChatList failed: get user by id error", zap.String("traceId", traceId), zap.Error(err))
			return nil, err
		}
		role, err := models.GetStoryRoleByID(ctx, chatCtx.RoleID)
		if err != nil {
			log.Log().Error("GetUserWithRoleChatList failed: get story role by id error", zap.String("traceId", traceId), zap.Error(err))
			return nil, err
		}
		lastMSg, err := models.GetChatContextLastMessage(ctx, int64(chatCtx.ID))
		if err != nil && err != gorm.ErrRecordNotFound {
			log.Log().Error("GetUserWithRoleChatList failed: get last chat message error", zap.String("traceId", traceId), zap.Error(err))
			return nil, err
		}
		if lastMSg == nil {
			log.Log().Info("GetUserWithRoleChatList: no last message, use empty", zap.String("traceId", traceId), zap.Int64("chat_id", int64(chatCtx.ID)))
			lastMSg = &models.ChatMessage{
				ChatContextID: int64(chatCtx.ID),
				Sender:        0,
			}
		}
		apiChatCtx := &api.ChatContext{
			ChatId:         int64(chatCtx.ID),
			UserId:         int64(chatCtx.UserID),
			RoleId:         int64(chatCtx.RoleID),
			Timestamp:      chatCtx.CreateAt.Unix(),
			LastUpdateTime: chatCtx.UpdateAt.Unix(),
			LastMessage:    convert.ConvertChatMessageToApiChatMessage(lastMSg),
			User:           convert.ConvertUserToApiUser(user),
			Role:           convert.ConvertStoryRoleToApiStoryRoleInfo(role),
		}
		apiChatCtxs = append(apiChatCtxs, apiChatCtx)
	}
	log.Log().Info("get user with role chat list success", zap.Any("total", total), zap.Any("apiChatCtxs", len(apiChatCtxs)))
	resp := &api.GetUserWithRoleChatListResponse{
		Code:     0,
		Message:  "OK",
		Chats:    apiChatCtxs,
		Total:    int64(total),
		Offset:   int64(req.GetOffset()),
		PageSize: int64(req.GetPageSize()),
	}
	log.Log().Info("GetUserWithRoleChatList success", zap.String("traceId", traceId), zap.Any("resp", resp))
	return resp, nil
}

// 更新角色详情
func (s *StoryroleService) UpdateStoryRoleDetail(ctx context.Context, req *api.UpdateStoryRoleDetailRequest) (*api.UpdateStoryRoleDetailResponse, error) {
	traceId := utils.GetTraceID(ctx)
	log.Log().Info("UpdateStoryRoleDetail called", zap.String("traceId", traceId), zap.Any("req", req))
	log.Log().Info("update story role detail", zap.Any("req", req.String()))
	role, err := models.GetStoryRoleByID(ctx, req.GetRoleId())
	if err != nil {
		log.Log().Error("UpdateStoryRoleDetail failed: get story role detail error", zap.String("traceId", traceId), zap.Error(err))
		return nil, err
	}
	if role == nil {
		log.Log().Warn("UpdateStoryRoleDetail: role not found", zap.String("traceId", traceId), zap.Int64("role_id", req.GetRoleId()))
		resp := &api.UpdateStoryRoleDetailResponse{
			Code:    -1,
			Message: "role not found",
		}
		log.Log().Info("UpdateStoryRoleDetail: role not found, return early", zap.String("traceId", traceId), zap.Any("resp", resp))
		return resp, nil
	}
	if role.CreatorID != req.GetUserId() {
		log.Log().Warn("UpdateStoryRoleDetail: no permission", zap.String("traceId", traceId), zap.Int64("creator_id", role.CreatorID), zap.Int64("user_id", req.GetUserId()))
		resp := &api.UpdateStoryRoleDetailResponse{
			Code:    -1,
			Message: "have no permission",
		}
		log.Log().Info("UpdateStoryRoleDetail: no permission, return early", zap.String("traceId", traceId), zap.Any("resp", resp))
		return resp, nil
	}
	updates := make(map[string]interface{})
	if req.GetRole().GetCharacterDescription() != "" {
		updates["character_description"] = req.GetRole().GetCharacterDescription()
	}
	if req.GetRole().GetCharacterAvatar() != "" {
		updates["character_avatar"] = req.GetRole().GetCharacterAvatar()
	}
	if req.GetRole().GetCharacterId() != "" {
		updates["character_id"] = req.GetRole().GetCharacterId()
	}
	if req.GetRole().GetCharacterType() != "" {
		updates["character_type"] = req.GetRole().GetCharacterType()
	}
	if req.GetRole().GetCharacterPrompt() != "" {
		var promptDetail = new(api.RenderStoryRoleDetail)
		err = json.Unmarshal([]byte(req.GetRole().GetCharacterPrompt()), promptDetail)
		if err != nil {
			log.Log().Error("UpdateStoryRoleDetail failed: unmarshal character prompt error", zap.String("traceId", traceId), zap.Error(err))
			return nil, err
		}
		updates["character_prompt"] = req.GetRole().GetCharacterPrompt()
	}
	if len(req.GetRole().GetCharacterRefImages()) > 0 {
		updates["character_ref_images"] = strings.Join(req.GetRole().GetCharacterRefImages(), ",")
	}
	err = models.UpdateStoryRole(ctx, int64(role.ID), updates)
	if err != nil {
		log.Log().Error("UpdateStoryRoleDetail failed: update story role detail error", zap.String("traceId", traceId), zap.Error(err))
		return nil, err
	}

	// 清除相关缓存
	storyRoleCache := cache.GetStoryRoleCache()
	if err := storyRoleCache.DeleteStoryRoleDetail(ctx, req.GetRoleId()); err != nil {
		log.Log().Warn("UpdateStoryRoleDetail: delete story role detail cache failed", zap.String("traceId", traceId), zap.Error(err))
	}

	resp := &api.UpdateStoryRoleDetailResponse{
		Code:    0,
		Message: "OK",
	}
	log.Log().Info("UpdateStoryRoleDetail success", zap.String("traceId", traceId), zap.Any("resp", resp))
	return resp, nil
}

func (s *StoryroleService) GetUserChatWithRole(ctx context.Context, req *api.GetUserChatWithRoleRequest) (*api.GetUserChatWithRoleResponse, error) {
	traceId := utils.GetTraceID(ctx)
	log.Log().Info("GetUserChatWithRole called", zap.String("traceId", traceId), zap.Any("req", req))
	if req.GetUserId() == 0 || req.GetRoleId() == 0 {
		log.Log().Warn("GetUserChatWithRole failed: invalid user id or role id", zap.String("traceId", traceId), zap.Any("userId", req.GetUserId()), zap.Any("roleId", req.GetRoleId()))
		return nil, errors.New("invalid user id or role id")
	}
	chatCtx, err := models.GetChatContextByUserIDAndRoleID(ctx, int64(req.GetUserId()), req.GetRoleId())
	if err != nil && err != gorm.ErrRecordNotFound {
		log.Log().Error("GetUserChatWithRole failed: get user chat context error", zap.String("traceId", traceId), zap.Error(err))
		return nil, err
	}
	if err == gorm.ErrRecordNotFound {
		log.Log().Info("GetUserChatWithRole: chat context not found", zap.String("traceId", traceId), zap.Int64("user_id", req.GetUserId()), zap.Int64("role_id", req.GetRoleId()))
		resp := &api.GetUserChatWithRoleResponse{
			Code:    1,
			Message: "chat context not found",
		}
		log.Log().Info("GetUserChatWithRole: chat context not found, return early", zap.String("traceId", traceId), zap.Any("resp", resp))
		return resp, nil
	}
	if chatCtx == nil {
		log.Log().Info("GetUserChatWithRole: chat context is nil", zap.String("traceId", traceId), zap.Int64("user_id", req.GetUserId()), zap.Int64("role_id", req.GetRoleId()))
		resp := &api.GetUserChatWithRoleResponse{
			Code:    1,
			Message: "chat context not found",
		}
		log.Log().Info("GetUserChatWithRole: chat context is nil, return early", zap.String("traceId", traceId), zap.Any("resp", resp))
		return resp, nil
	}
	user, err := models.GetUserById(ctx, int64(chatCtx.UserID))
	if err != nil {
		log.Log().Error("GetUserChatWithRole failed: get user by id error", zap.String("traceId", traceId), zap.Error(err))
		return nil, err
	}
	role, err := models.GetStoryRoleByID(ctx, chatCtx.RoleID)
	if err != nil {
		log.Log().Error("GetUserChatWithRole failed: get story role by id error", zap.String("traceId", traceId), zap.Error(err))
		return nil, err
	}
	lastMSg, err := models.GetChatContextLastMessage(ctx, int64(chatCtx.ID))
	if err != nil && err != gorm.ErrRecordNotFound {
		log.Log().Error("GetUserChatWithRole failed: get last chat message error", zap.String("traceId", traceId), zap.Error(err))
		return nil, err
	}
	if lastMSg == nil {
		log.Log().Info("GetUserChatWithRole: no last message, use empty", zap.String("traceId", traceId), zap.Int64("chat_id", int64(chatCtx.ID)))
		lastMSg = &models.ChatMessage{
			ChatContextID: int64(chatCtx.ID),
			Sender:        0,
		}
	}
	resp := &api.GetUserChatWithRoleResponse{
		Code:    0,
		Message: "OK",
		ChatContext: &api.ChatContext{
			ChatId:         int64(chatCtx.ID),
			UserId:         int64(chatCtx.UserID),
			RoleId:         int64(chatCtx.RoleID),
			Timestamp:      chatCtx.CreateAt.Unix(),
			LastUpdateTime: chatCtx.UpdateAt.Unix(),
			User:           convert.ConvertUserToApiUser(user),
			Role:           convert.ConvertStoryRoleToApiStoryRoleInfo(role),
			LastMessage:    convert.ConvertChatMessageToApiChatMessage(lastMSg),
		},
	}
	log.Log().Info("GetUserChatWithRole success", zap.String("traceId", traceId), zap.Any("resp", resp))
	return resp, nil
}

func (s *StoryroleService) GetUserChatMessages(ctx context.Context, req *api.GetUserChatMessagesRequest) (*api.GetUserChatMessagesResponse, error) {
	traceId := utils.GetTraceID(ctx)
	log.Log().Info("GetUserChatMessages called", zap.String("traceId", traceId), zap.Any("req", req))
	if req.GetChatId() == 0 && req.GetUserId() == 0 && req.GetRoleId() == 0 {
		log.Log().Warn("GetUserChatMessages failed: invalid chat id or user id or role id", zap.String("traceId", traceId))
		return nil, errors.New("invalid chat id or user id or role id")
	}
	var (
		lastTimestamp int64
		total         int
		err           error
		chatMsgs      []*models.ChatMessage
	)
	if req.GetChatId() == 0 && req.GetUserId() != 0 && req.GetRoleId() == 0 {
		// 获取用户的消息，不区分聊天上下文
		chatMsgs, total, err = models.GetChatMessageByUserID(ctx, int64(req.GetUserId()), 0, 100)
		if err != nil {
			log.Log().Error("GetUserChatMessages failed: get user chat messages error", zap.String("traceId", traceId), zap.Error(err))
			return nil, err
		}
		_ = total
		for _, chatMsg := range chatMsgs {
			if lastTimestamp == 0 || chatMsg.CreateAt.Unix() < lastTimestamp {
				lastTimestamp = chatMsg.CreateAt.Unix()
			}
		}
	} else if req.GetChatId() == 0 && req.GetUserId() == 0 && req.GetRoleId() != 0 {
		// 获取角色的消息，不区分聊天上下文
		chatMsgs, total, err = models.GetChatMessageByRoleID(ctx, req.GetRoleId(), 0, 100)
		if err != nil {
			log.Log().Error("GetUserChatMessages failed: get role chat messages error", zap.String("traceId", traceId), zap.Error(err))
			return nil, err
		}
		_ = total
		for _, chatMsg := range chatMsgs {
			if lastTimestamp == 0 || chatMsg.CreateAt.Unix() < lastTimestamp {
				lastTimestamp = chatMsg.CreateAt.Unix()
			}
		}
	} else if req.GetChatId() != 0 && req.GetUserId() == 0 && req.GetRoleId() == 0 {
		// 获取指定聊天的消息
		chatMsgs, total, err = models.GetChatMessageByChatContextID(ctx, int64(req.GetChatId()), 0, 100)
		if err != nil {
			log.Log().Error("GetUserChatMessages failed: get chat context chat messages error", zap.String("traceId", traceId), zap.Error(err))
			return nil, err
		}
		_ = total
		for _, chatMsg := range chatMsgs {
			if lastTimestamp == 0 || chatMsg.CreateAt.Unix() < lastTimestamp {
				lastTimestamp = chatMsg.CreateAt.Unix()
			}
		}
	}
	apiChatMsgs := make([]*api.ChatMessage, 0)
	for _, chatMsg := range chatMsgs {
		apiChatMsgs = append(apiChatMsgs, convert.ConvertChatMessageToApiChatMessage(chatMsg))
	}
	resp := &api.GetUserChatMessagesResponse{
		Code:      0,
		Message:   "OK",
		Timestamp: lastTimestamp,
		Total:     int64(total),
		Messages:  apiChatMsgs,
	}
	log.Log().Info("GetUserChatMessages success", zap.String("traceId", traceId), zap.Any("resp", resp))
	return resp, nil
}

// 根据角色参与的故事板的历史记录，以及和别的角色的冲突，生成角色的性格描述，以及新的角色背景图片和头像图片
func (s *StoryroleService) RenderStoryRoleContinuously(ctx context.Context, req *api.RenderStoryRoleContinuouslyRequest) (*api.RenderStoryRoleContinuouslyResponse, error) {
	traceId := utils.GetTraceID(ctx)
	log.Log().Info("RenderStoryRoleContinuously called", zap.String("traceId", traceId), zap.Any("req", req))
	role, err := models.GetStoryRoleByID(ctx, req.GetRoleId())
	if err != nil {
		log.Log().Error("RenderStoryRoleContinuously failed: get story role by id error", zap.String("traceId", traceId), zap.Error(err))
		return nil, err
	}
	if role.CreatorID != req.GetUserId() {
		log.Log().Warn("RenderStoryRoleContinuously failed: no permission", zap.String("traceId", traceId), zap.Int64("creator_id", role.CreatorID), zap.Int64("user_id", req.GetUserId()))
		return nil, errors.New("have no permission")
	}
	if role.Status != 1 {
		log.Log().Warn("RenderStoryRoleContinuously failed: role is not ready", zap.String("traceId", traceId), zap.Int32("status", int32(role.Status)))
		return nil, errors.New("role is not ready")
	}
	story, err := models.GetStory(ctx, role.StoryID)
	if err != nil {
		log.Log().Error("RenderStoryRoleContinuously failed: get story error", zap.String("traceId", traceId), zap.Error(err))
		return nil, err
	}
	historyStoryGen, err := models.GetStoryGensByStoryAndRole(ctx, role.StoryID, int64(role.ID))
	if err != nil {
		log.Log().Error("RenderStoryRoleContinuously failed: get story gen by story and role error", zap.String("traceId", traceId), zap.Error(err))
	}
	if historyStoryGen != nil && historyStoryGen.GenStatus == 1 {
		log.Log().Info("RenderStoryRoleContinuously: generating", zap.String("traceId", traceId))
		resp := &api.RenderStoryRoleContinuouslyResponse{
			Code:    0,
			Message: "generating",
			Detail:  nil,
		}
		log.Log().Info("RenderStoryRoleContinuously: generating, return early", zap.String("traceId", traceId), zap.Any("resp", resp))
		return resp, nil
	}
	if historyStoryGen != nil && historyStoryGen.GenStatus == 2 && historyStoryGen.CreateAt.Add(time.Hour*12).Before(time.Now()) {
		log.Log().Info("RenderStoryRoleContinuously: role render finished", zap.String("traceId", traceId))
		resp := &api.RenderStoryRoleContinuouslyResponse{
			Code:    0,
			Message: "role render finished",
			Detail:  nil,
		}
		log.Log().Info("RenderStoryRoleContinuously: role render finished, return early", zap.String("traceId", traceId), zap.Any("resp", resp))
		return resp, nil
	}

	storyroleParams := coze.CozeStoryRoleDetailContinueParams{
		RoleName:    utils.MaskContent(role.CharacterName),
		Description: utils.MaskContent(role.CharacterDescription),
		StoryDesc:   utils.MaskContent(story.ShortDesc),
		StoryName:   utils.MaskContent(story.Title),
		OtherRoles:  "",
		History:     "",
	}

	histroryStoryBoardSences, err := models.GetStoryBoardSencesByRoleID(ctx, role.StoryID)
	if err != nil {
		log.Log().Error("RenderStoryRoleContinuously failed: get story board sences by role id error", zap.String("traceId", traceId), zap.Error(err))
		return nil, err
	}
	if req.GetPrompt() == "" {
		var historySenceStr = ""
		for _, histrorySence := range histroryStoryBoardSences {
			historySenceStr = historySenceStr + histrorySence.Content + "\n"
		}
		storyroleParams.History = historySenceStr
	} else {
		storyroleParams.History = utils.MaskContent(req.GetPrompt())
	}

	// 调用生成器
	storyGen := new(models.StoryGen)
	storyGen.UUID = uuid.New().String()
	storyGen.LLmPlatform = "coze"
	storyGen.Params = req.String()
	storyGen.OriginID = req.GetRoleId()
	storyGen.StartTime = time.Now().Unix()
	storyGen.BoardID = 0
	storyGen.TaskType = api.RenderType_RENDER_TYPE_STORYCHARACTERS
	storyGen.Status = 1
	storyGen.UserId = req.GetUserId()
	storyGen.OriginID = role.StoryID
	storyGen.RoleID = int64(role.ID)
	storyGen.GenStatus = api.StoryGenStatus_STORY_GEN_STATUS_INIT
	storyGen.TaskId = uuid.New().String()
	_, err = models.CreateStoryGen(ctx, storyGen)
	if err != nil {
		log.Log().Error("RenderStoryRoleContinuously failed: create story gen error", zap.String("traceId", traceId), zap.Error(err))
		return nil, err
	}

	ret, err := s.cozeClient.StoryRoleDetailContinue(ctx, storyroleParams)
	if err != nil {
		log.Log().Error("RenderStoryRoleContinuously failed: gen story info error", zap.String("traceId", traceId), zap.Error(err))
		return nil, err
	}
	var renderDetail = new(api.RenderStoryRoleDetail)
	result := new(CharacterDetail)
	cleanResult := utils.CleanLLmJsonResult(ret)
	err = json.Unmarshal([]byte(cleanResult), &result)
	if err != nil {
		log.Log().Error("RenderStoryRoleContinuously failed: unmarshal gen result error", zap.String("traceId", traceId), zap.Error(err))
		return nil, err
	}
	log.Log().Info("RenderStoryRoleContinuously: cleaned LLM result", zap.String("traceId", traceId), zap.String("content", cleanResult))
	storyGen.Content = cleanResult
	storyGen.FinishTime = time.Now().Unix()
	storyGen.GenStatus = api.StoryGenStatus_STORY_GEN_STATUS_FINISHED
	renderDetail.RoleCharacter = result.Description
	renderDetail.RoleDescription = result.DressPreference
	renderDetail.RoleBehavior = result.HandlingStyle
	renderDetail.RoleGoal = result.LongTermGoal
	renderDetail.Background = result.Background
	renderDetail.Appearance = result.Appearance
	renderDetail.Personality = result.Personality
	renderDetail.AbilityFeatures = result.AbilityFeatures
	err = models.UpdateStoryGen(ctx, storyGen)
	if err != nil {
		log.Log().Warn("RenderStoryRoleContinuously failed: update story gen error", zap.String("traceId", traceId), zap.Error(err))
	} else {
		log.Log().Info("RenderStoryRoleContinuously: update story gen success", zap.String("traceId", traceId))
	}
	resp := &api.RenderStoryRoleContinuouslyResponse{
		Code:    0,
		Message: "OK",
		Detail:  renderDetail,
	}
	log.Log().Info("RenderStoryRoleContinuously success", zap.String("traceId", traceId), zap.Any("resp", resp))
	return resp, nil
}

func (s *StoryroleService) GenerateRoleDescription(ctx context.Context, req *api.GenerateRoleDescriptionRequest) (*api.GenerateRoleDescriptionResponse, error) {
	traceId := utils.GetTraceID(ctx)
	log.Log().Info("GenerateRoleDescription called", zap.String("traceId", traceId), zap.Any("req", req))

	roleinfo, err := models.GetStoryRoleByID(ctx, req.GetRoleId())
	if err != nil {
		log.Log().Error("GenerateRoleDescription failed: get story role by id error", zap.String("traceId", traceId), zap.Error(err))
		return nil, err
	}
	if roleinfo.CreatorID != req.GetUserId() {
		log.Log().Warn("GenerateRoleDescription failed: no permission", zap.String("traceId", traceId), zap.Int64("creator_id", roleinfo.CreatorID), zap.Int64("user_id", req.GetUserId()))
		return nil, errors.New("have no permission")
	}

	storyinfo, err := models.GetStory(ctx, roleinfo.StoryID)
	if err != nil {
		log.Log().Error("GenerateRoleDescription failed: get story error", zap.String("traceId", traceId), zap.Error(err))
		return nil, err
	}

	// Get all roles in the story to provide context
	roles, err := models.GetStoryRole(ctx, req.GetStoryId())
	if err != nil {
		log.Log().Error("GenerateRoleDescription failed: get story roles error", zap.String("traceId", traceId), zap.Error(err))
		return nil, err
	}

	// Build role context information
	var otherRolesInfo strings.Builder
	for _, role := range roles {
		if role.ID != roleinfo.ID {
			otherRolesInfo.WriteString(fmt.Sprintf("角色名称: %s\n角色描述: %s\n\n", utils.MaskContent(role.CharacterName), utils.MaskContent(role.CharacterDescription)))
		}
	}
	var storyroleParams = coze.CozeStoryRoleDetailParams{
		RoleName:    utils.MaskContent(roleinfo.CharacterName),
		Description: utils.MaskContent(req.GetDescription()),
		StoryName:   utils.MaskContent(storyinfo.Title),
		StoryDesc:   utils.MaskContent(storyinfo.ShortDesc),
	}
	if storyroleParams.Description == "" {
		storyroleParams.Description = utils.MaskContent(roleinfo.CharacterDescription)
	}
	if len(roles) > 0 {
		storyroleParams.OtherRoles = otherRolesInfo.String()
	} else {
		storyroleParams.OtherRoles = "没有其他角色信息"
	}

	result, err := s.cozeClient.StoryRoleDetail(ctx, storyroleParams)
	if err != nil {
		log.Log().Error("GenerateRoleDescription failed: generate role description error", zap.String("traceId", traceId), zap.Error(err))
		return nil, errors.New("failed to generate role description")
	}

	// Clean and parse the AI response
	cleanResult := utils.CleanLLmJsonResult(result)
	log.Log().Info("GenerateRoleDescription: cleaned LLM result", zap.String("traceId", traceId), zap.String("content", cleanResult))
	var genRoleDetail = new(CharacterDetail)
	err = json.Unmarshal([]byte(cleanResult), &genRoleDetail)
	if err != nil {
		log.Log().Error("GenerateRoleDescription failed: unmarshal gen result error", zap.String("traceId", traceId), zap.Error(err))
		resp := &api.GenerateRoleDescriptionResponse{
			Code:    -1,
			Message: err.Error(),
		}
		log.Log().Info("GenerateRoleDescription: unmarshal error, return early", zap.String("traceId", traceId), zap.Any("resp", resp))
		return resp, nil
	}
	apiCharacterDetail := &api.CharacterDetail{
		Description:     genRoleDetail.Description,
		ShortTermGoal:   genRoleDetail.ShortTermGoal,
		LongTermGoal:    genRoleDetail.LongTermGoal,
		Personality:     genRoleDetail.Personality,
		Background:      genRoleDetail.Background,
		HandlingStyle:   genRoleDetail.HandlingStyle,
		CognitionRange:  genRoleDetail.CognitionRange,
		AbilityFeatures: genRoleDetail.AbilityFeatures,
		Appearance:      genRoleDetail.Appearance,
		DressPreference: genRoleDetail.DressPreference,
	}
	log.Log().Info("GenerateRoleDescription success", zap.String("traceId", traceId), zap.Any("apiCharacterDetail", apiCharacterDetail.String()))
	resp := &api.GenerateRoleDescriptionResponse{
		Code:            0,
		Message:         "OK",
		CharacterDetail: apiCharacterDetail,
	}
	log.Log().Info("GenerateRoleDescription success", zap.String("traceId", traceId), zap.Any("resp", resp))
	return resp, nil
}

func (s *StoryroleService) UpdateStoryRoleDescriptionDetail(ctx context.Context, req *api.UpdateStoryRoleDescriptionDetailRequest) (*api.UpdateStoryRoleDescriptionDetailResponse, error) {
	traceId := utils.GetTraceID(ctx)
	log.Log().Info("UpdateStoryRoleDescriptionDetail called", zap.String("traceId", traceId), zap.Any("req", req))
	roleinfo, err := models.GetStoryRoleByID(ctx, req.GetRoleId())
	if err != nil {
		log.Log().Error("UpdateStoryRoleDescriptionDetail failed: get story role by id error", zap.String("traceId", traceId), zap.Error(err))
		return nil, err
	}
	if roleinfo == nil {
		log.Log().Warn("UpdateStoryRoleDescriptionDetail: role not exist", zap.String("traceId", traceId), zap.Int64("role_id", req.GetRoleId()))
		resp := &api.UpdateStoryRoleDescriptionDetailResponse{
			Code:    -1,
			Message: "role not exist",
		}
		log.Log().Info("UpdateStoryRoleDescriptionDetail: role not exist, return early", zap.String("traceId", traceId), zap.Any("resp", resp))
		return resp, nil
	}
	if roleinfo.CreatorID != req.GetUserId() {
		log.Log().Warn("UpdateStoryRoleDescriptionDetail: no permission", zap.String("traceId", traceId), zap.Int64("creator_id", roleinfo.CreatorID), zap.Int64("user_id", req.GetUserId()))
		resp := &api.UpdateStoryRoleDescriptionDetailResponse{
			Code:    -1,
			Message: "have no permission",
		}
		log.Log().Info("UpdateStoryRoleDescriptionDetail: no permission, return early", zap.String("traceId", traceId), zap.Any("resp", resp))
		return resp, nil
	}
	descStr, _ := json.Marshal(req.GetCharacterDetail())
	roleinfo.CharacterDetail = string(descStr)
	err = models.UpdateStoryRole(ctx, int64(roleinfo.ID), map[string]interface{}{
		"character_detail": roleinfo.CharacterDetail,
	})
	if err != nil {
		log.Log().Error("UpdateStoryRoleDescriptionDetail failed: update story role description error", zap.String("traceId", traceId), zap.Error(err))
		return nil, err
	}
	resp := &api.UpdateStoryRoleDescriptionDetailResponse{
		Code:    0,
		Message: "OK",
	}
	log.Log().Info("UpdateStoryRoleDescriptionDetail success", zap.String("traceId", traceId), zap.Any("resp", resp))
	return resp, nil
}

func (s *StoryroleService) UpdateRoleDescription(ctx context.Context, req *api.UpdateRoleDescriptionRequest) (*api.UpdateRoleDescriptionResponse, error) {
	traceId := utils.GetTraceID(ctx)
	log.Log().Info("UpdateRoleDescription called", zap.String("traceId", traceId), zap.Any("req", req))
	isPass, err := compliance.TextCompliance(req.GetDescription())
	if err != nil {
		log.Log().Error("[UpdateRoleDescription] 简介合规检测失败", zap.Error(err))
		return nil, err
	}
	if !isPass {
		log.Log().Error("[UpdateRoleDescription] 简介合规检测失败", zap.Error(err))
		return nil, errors.New("简介合规检测未通过")
	}
	roleinfo, err := models.GetStoryRoleByID(ctx, req.GetRoleId())
	if err != nil {
		log.Log().Error("UpdateRoleDescription failed: get story role by id error", zap.String("traceId", traceId), zap.Error(err))
		return nil, err
	}
	if roleinfo.CreatorID != req.GetUserId() {
		log.Log().Warn("UpdateRoleDescription failed: no permission", zap.String("traceId", traceId), zap.Int64("creator_id", roleinfo.CreatorID), zap.Int64("user_id", req.GetUserId()))
		return nil, errors.New("have no permission")
	}
	if roleinfo.Status != 1 {
		log.Log().Warn("UpdateRoleDescription failed: role is not ready", zap.String("traceId", traceId), zap.Int32("status", int32(roleinfo.Status)))
		return nil, errors.New("role is not ready")
	}
	roleinfo.CharacterDescription = req.GetDescription()
	err = models.UpdateStoryRole(ctx, int64(roleinfo.ID), map[string]interface{}{
		"character_description": req.GetDescription(),
	})
	if err != nil {
		log.Log().Error("UpdateRoleDescription failed: update story role description error", zap.String("traceId", traceId), zap.Error(err))
		return nil, err
	}
	resp := &api.UpdateRoleDescriptionResponse{
		Code:    0,
		Message: "OK",
	}
	log.Log().Info("UpdateRoleDescription success", zap.String("traceId", traceId), zap.Any("resp", resp))
	return resp, nil
}

func (s *StoryroleService) UpdateStoryRolePrompt(ctx context.Context, req *api.UpdateStoryRolePromptRequest) (*api.UpdateStoryRolePromptResponse, error) {
	traceId := utils.GetTraceID(ctx)
	log.Log().Info("UpdateStoryRolePrompt called", zap.String("traceId", traceId), zap.Any("req", req))
	roleinfo, err := models.GetStoryRoleByID(ctx, req.GetRoleId())
	if err != nil {
		log.Log().Error("UpdateStoryRolePrompt failed: get story role by id error", zap.String("traceId", traceId), zap.Error(err))
		return nil, err
	}
	if roleinfo.CreatorID != req.GetRoleId() {
		log.Log().Warn("UpdateStoryRolePrompt failed: no permission", zap.String("traceId", traceId), zap.Int64("creator_id", roleinfo.CreatorID), zap.Int64("user_id", req.GetUserId()))
		return nil, errors.New("have no permission")
	}
	isPass, err := compliance.TextCompliance(req.GetPrompt())
	if err != nil {
		log.Log().Error("[UpdateStoryRolePrompt] 角色提示合规检测失败", zap.Error(err))
		return nil, err
	}
	if !isPass {
		log.Log().Error("[UpdateStoryRolePrompt] 角色提示合规检测失败", zap.Error(err))
		return nil, errors.New("角色提示合规检测未通过")
	}
	roleinfo.CharacterPrompt = req.GetPrompt()
	err = models.UpdateStoryRole(ctx, int64(roleinfo.ID), map[string]interface{}{
		"character_prompt": req.GetPrompt(),
	})
	if err != nil {
		log.Log().Error("UpdateStoryRolePrompt failed: update story role prompt error", zap.String("traceId", traceId), zap.Error(err))
		return nil, err
	}
	resp := &api.UpdateStoryRolePromptResponse{
		Code:    api.ResponseCode_OK,
		Message: "OK",
	}
	log.Log().Info("UpdateStoryRolePrompt success", zap.String("traceId", traceId), zap.Any("resp", resp))
	return resp, nil
}

func (s *StoryroleService) GenerateRolePrompt(ctx context.Context, req *api.GenerateRolePromptRequest) (*api.GenerateRolePromptResponse, error) {
	traceId := utils.GetTraceID(ctx)
	log.Log().Info("GenerateRolePrompt called", zap.String("traceId", traceId), zap.Any("req", req))
	storyinfo, err := models.GetStory(ctx, req.GetStoryId())
	if err != nil {
		log.Log().Error("GenerateRolePrompt failed: get story error", zap.String("traceId", traceId), zap.Error(err))
		return nil, err
	}
	roleinfo, err := models.GetStoryRoleByID(ctx, req.GetRoleId())
	if err != nil {
		log.Log().Error("GenerateRolePrompt failed: get story role by id error", zap.String("traceId", traceId), zap.Error(err))
		return nil, err
	}
	if roleinfo.CreatorID != req.GetUserId() {
		log.Log().Warn("GenerateRolePrompt failed: no permission", zap.String("traceId", traceId), zap.Int64("creator_id", roleinfo.CreatorID), zap.Int64("user_id", req.GetUserId()))
		return nil, errors.New("have no permission")
	}
	_ = storyinfo
	resp := &api.GenerateRolePromptResponse{
		Code:    api.ResponseCode_OK,
		Message: "OK",
	}
	log.Log().Info("GenerateRolePrompt success", zap.String("traceId", traceId), zap.Any("resp", resp))
	return resp, nil
}

func (s *StoryroleService) UpdateRolePrompt(ctx context.Context, req *api.UpdateRolePromptRequest) (*api.UpdateRolePromptResponse, error) {
	traceId := utils.GetTraceID(ctx)
	log.Log().Info("UpdateRolePrompt called", zap.String("traceId", traceId), zap.Any("req", req))
	roleinfo, err := models.GetStoryRoleByID(ctx, req.GetRoleId())
	if err != nil {
		log.Log().Error("UpdateRolePrompt failed: get story role by id error", zap.String("traceId", traceId), zap.Error(err))
		return nil, err
	}
	if roleinfo.CreatorID != req.GetUserId() {
		log.Log().Warn("UpdateRolePrompt failed: no permission", zap.String("traceId", traceId), zap.Int64("creator_id", roleinfo.CreatorID), zap.Int64("user_id", req.GetUserId()))
		return nil, errors.New("have no permission")
	}
	roleinfo.CharacterPrompt = req.GetPrompt()
	err = models.UpdateStoryRole(ctx, int64(roleinfo.ID), map[string]interface{}{
		"character_prompt": req.GetPrompt(),
	})
	if err != nil {
		log.Log().Error("UpdateRolePrompt failed: update story role prompt error", zap.String("traceId", traceId), zap.Error(err))
		return nil, err
	}
	resp := &api.UpdateRolePromptResponse{
		Code:    0,
		Message: "OK",
	}
	log.Log().Info("UpdateRolePrompt success", zap.String("traceId", traceId), zap.Any("resp", resp))
	return resp, nil
}

func (s *StoryroleService) UpdateStoryRolePoster(ctx context.Context, req *api.UpdateStoryRolePosterRequest) (*api.UpdateStoryRolePosterResponse, error) {
	if req.GetImageUrl() != "" { // 更新url
		err := models.SaveRolePosterUrl(ctx, req.GetPosterId(), req.GetImageUrl())
		if err != nil {
			log.Log().Error("[UpdateStoryRolePoster] 保存故事角色海报url失败",
				zap.Int64("posterId", req.GetPosterId()),
				zap.String("imageUrl", req.GetImageUrl()),
				zap.Error(err))
			return nil, err
		}
	} else { // 发布
		err := models.PublishRolePoster(ctx, req.GetPosterId())
		if err != nil {
			log.Log().Error("[UpdateStoryRolePoster] 发布故事角色海报失败",
				zap.Int64("posterId", req.GetPosterId()),
				zap.Error(err))
			return nil, err
		}
	}
	log.Log().Info("[UpdateStoryRolePoster] 更新故事角色海报成功",
		zap.Int64("posterId", req.GetPosterId()))
	resp := &api.UpdateStoryRolePosterResponse{
		Code:    api.ResponseCode_OK,
		Message: "OK",
	}
	return resp, nil
}

func (s *StoryroleService) GenerateStoryRolePoster(ctx context.Context, req *api.GenerateStoryRolePosterRequest) (*api.GenerateStoryRolePosterResponse, error) {
	traceId := utils.GetTraceID(ctx)
	log.Log().Info("GenerateStoryRolePoster called", zap.String("traceId", traceId), zap.Any("req", req))
	roleInfo, err := models.GetStoryRoleByID(ctx, req.GetRoleId())
	if err != nil {
		log.Log().Error("[GenerateStoryRolePoster] 获取故事角色失败",
			zap.Int64("storyId", req.GetStoryId()),
			zap.Int64("roleId", req.GetRoleId()),
			zap.Error(err))
		return nil, err
	}
	storyinfo, err := models.GetStory(ctx, roleInfo.StoryID)
	if err != nil {
		log.Log().Error("GenerateStoryRolePoster failed: get story error", zap.String("traceId", traceId), zap.Error(err))
		return nil, err
	}
	roleinfo, err := models.GetStoryRoleByID(ctx, req.GetRoleId())
	if err != nil {
		log.Log().Error("GenerateStoryRolePoster failed: get story role by id error", zap.String("traceId", traceId), zap.Error(err))
		return nil, err
	}
	if roleinfo.CreatorID != req.GetUserId() {
		log.Log().Warn("GenerateStoryRolePoster failed: no permission", zap.String("traceId", traceId), zap.Int64("creator_id", roleinfo.CreatorID), zap.Int64("user_id", req.GetUserId()))
		return nil, errors.New("have no permission")
	}
	var roleDetail = new(CharacterDetailConverter)
	err = json.Unmarshal([]byte(roleinfo.CharacterDetail), &roleDetail)
	if err != nil {
		log.Log().Error("GenerateStoryRolePoster failed: unmarshal story role detail error", zap.String("traceId", traceId), zap.Error(err))
		return nil, err
	}
	roleDetail.Description = roleinfo.CharacterDescription
	if err != nil {
		log.Log().Error("GenerateStoryRolePoster failed: marshal story role detail error", zap.String("traceId", traceId), zap.Error(err))
		return nil, err
	}
	log.Log().Info("GenerateStoryRolePoster: marshal story role detail success", zap.String("traceId", traceId))
	// rolePosterParams := coze.CozeStoryRolePosterImageParams{
	// 	RoleName:  roleinfo.CharacterName,
	// 	RoleDesc:  fmt.Sprintln("角色描述:", roleDetail.Description, "性格:", roleDetail.Personality, "外貌:", roleDetail.Appearance, "背景:", roleDetail.Background, "行为风格:", roleDetail.HandlingStyle, "认知范围:", roleDetail.CognitionRange, "能力特征:", roleDetail.AbilityFeatures, "着装偏好:", roleDetail.DressPreference),
	// 	RoleImage: req.Params.GetOriginImageUrl(),
	// 	Style:     req.Params.Style,
	// 	Prompt:    req.Params.TextPrompt,
	// }
	// if len(rolePosterParams.RoleImage) == 0 {
	// 	rolePosterParams.RoleImage = roleInfo.CharacterAvatar
	// }
	// if len(rolePosterParams.Style) == 0 {
	// 	rolePosterParams.Style = storyinfo.Style
	// }
	var allImageParams []string
	if len(req.Params.GetOriginImageUrl()) > 0 {
		allImageParams = append(allImageParams, req.Params.GetOriginImageUrl())
	}
	if len(req.Params.GetAdditionalImageUrls()) > 0 {
		allImageParams = append(allImageParams, req.Params.GetAdditionalImageUrls()...)
	}
	style := req.Params.Style
	if style == "" {
		style = storyinfo.Style
	}
	allImageParamsData, _ := json.Marshal(allImageParams)
	log.Log().Info("GenerateStoryRolePoster: all image params", zap.String("traceId", traceId), zap.String("allImageParams", string(allImageParamsData)), zap.Strings("allImageParamsArr", allImageParams))
	posterPrompt := req.GetParams().TextPrompt +
		"风格: " + style + ",细节描述" + fmt.Sprintln("角色描述:", roleDetail.Description, "性格:", roleDetail.Personality, "外貌:", roleDetail.Appearance, "背景:", roleDetail.Background, "行为风格:", roleDetail.HandlingStyle, "认知范围:", roleDetail.CognitionRange, "能力特征:", roleDetail.AbilityFeatures, "着装偏好:", roleDetail.DressPreference)
	newPosterId, err := models.CreateRolePoster(ctx, req.GetRoleId(), "", req.GetUserId(),
		req.Params.GetOriginImageUrl(), req.Params.TextPrompt, req.Params.Style,
		req.Params.AdditionalImageUrls, 0, uuid.NewString())
	if err != nil {
		log.Log().Error("GenerateStoryRolePoster failed: create role poster error", zap.String("traceId", traceId), zap.Error(err))
		return nil, err
	}
	resp, err := s.seedreamClient.GenerateMultiImageToImage(ctx,
		posterPrompt,
		allImageParams)
	if err != nil {
		log.Log().Error("GenerateStoryRolePoster failed: generate story role poster error", zap.String("traceId", traceId), zap.Error(err))
		return nil, err
	}
	if len(resp.Data) == 0 {
		log.Log().Error("GenerateStoryRolePoster failed: generate story role poster error", zap.String("traceId", traceId), zap.Error(err))
		return nil, errors.New("generate story role poster error")
	}
	respData, _ := json.Marshal(resp)
	log.Log().Info("GenerateStoryRolePoster success", zap.String("traceId", traceId), zap.String("respData", string(respData)))
	imageUrl := resp.Data[0]
	// 上传到aliyun
	aliyunClient := aliyun.GetGlobalClient()
	newUrl, err := aliyunClient.UploadFileFromURL("", imageUrl.URL)
	if err != nil {
		log.Log().Error("GenerateStoryRolePoster failed: upload file from url error", zap.String("traceId", traceId), zap.Error(err))
		return nil, err
	}
	ret := &api.GenerateStoryRolePosterResponse{
		Code:     api.ResponseCode_OK,
		Message:  "OK",
		ImageUrl: newUrl,
		PosterId: newPosterId,
	}
	log.Log().Info("GenerateStoryRolePoster success", zap.String("traceId", traceId), zap.Any("resp", ret))
	return ret, nil
}

func (s *StoryroleService) GenerateRoleAvatar(ctx context.Context, req *api.GenerateRoleAvatarRequest) (*api.GenerateRoleAvatarResponse, error) {
	traceId := utils.GetTraceID(ctx)
	log.Log().Info("GenerateRoleAvatar called", zap.String("traceId", traceId), zap.Any("req", req))
	roleinfo, err := models.GetStoryRoleByID(ctx, req.GetRoleId())
	if err != nil {
		log.Log().Error("GenerateRoleAvatar failed: get story role by id error", zap.String("traceId", traceId), zap.Error(err))
		return nil, err
	}
	log.Log().Info("GenerateRoleAvatar: get story role by id success", zap.String("traceId", traceId), zap.Any("roleinfo", roleinfo))
	var roleDetail = new(CharacterDetailConverter)
	err = json.Unmarshal([]byte(roleinfo.CharacterDetail), &roleDetail)
	if err != nil {
		log.Log().Error("GenerateRoleAvatar failed: marshal story role detail error", zap.String("traceId", traceId), zap.Error(err))
		return nil, err
	}
	log.Log().Info("GenerateRoleAvatar: unmarshal story role detail success", zap.String("traceId", traceId), zap.Any("roleDetail", roleDetail))
	roleAvatarParams := coze.CozeStoryRoleImageParams{
		Description:     req.GetDescription(),
		ShortTermGoal:   roleDetail.ShortTermGoal,
		LongTermGoal:    roleDetail.LongTermGoal,
		Personality:     roleDetail.Personality,
		Background:      roleDetail.Background,
		HandlingStyle:   roleDetail.HandlingStyle,
		CognitionRange:  roleDetail.CognitionRange,
		AbilityFeatures: roleDetail.AbilityFeatures,
		Appearance:      roleDetail.Appearance,
		DressPreference: roleDetail.DressPreference,
		RefImage:        req.GetRefAvatarUrl(),
		Style:           req.GetStyle(),
		Ratio:           req.GetImageRatios(),
	}
	if roleAvatarParams.Style == "" {
		roleAvatarParams.Style = "吉卜力风格"
	}

	imageUrl, err := s.cozeClient.StoryRoleImage(ctx, roleAvatarParams)
	if err != nil {
		log.Log().Error("GenerateRoleAvatar failed: generate story role avatar error", zap.String("traceId", traceId), zap.Error(err))
		return nil, err
	}
	// 上传到aliyun
	aliyunClient := aliyun.GetGlobalClient()
	newUrl, err := aliyunClient.UploadFileFromURL("", imageUrl)
	if err != nil {
		log.Log().Error("GenerateRoleAvatar failed: persist images error", zap.String("traceId", traceId), zap.Error(err))
		return nil, err
	}
	if newUrl == "" {
		log.Log().Error("GenerateRoleAvatar failed: upload file from url error", zap.String("traceId", traceId), zap.Error(err))
		return nil, errors.New("upload file from url error")
	}
	log.Log().Info("GenerateRoleAvatar: generate story role avatar success", zap.String("traceId", traceId), zap.Any("imageUrl", imageUrl))
	resp := &api.GenerateRoleAvatarResponse{
		Code:      api.ResponseCode_OK,
		Message:   api.ResponseCode_OK.String(),
		AvatarUrl: newUrl,
	}
	log.Log().Info("GenerateRoleAvatar success", zap.String("traceId", traceId), zap.Any("resp", resp))
	return resp, nil
}

func (s *StoryroleService) GenerateStoryRoleVideo(ctx context.Context, req *api.GenerateStoryRoleVideoRequest) (*api.GenerateStoryRoleVideoResponse, error) {
	traceId := utils.GetTraceID(ctx)
	log.Log().Info("GenerateStoryRoleVideo called", zap.String("traceId", traceId), zap.Any("req", req))
	roleinfo, err := models.GetStoryRoleByID(ctx, req.GetRoleId())
	if err != nil {
		log.Log().Error("GenerateStoryRoleVideo failed: get story role by id error", zap.String("traceId", traceId), zap.Error(err))
		return nil, err
	}
	if roleinfo.CharacterAvatar == "" {
		log.Log().Error("GenerateStoryRoleVideo failed: character avatar is empty", zap.String("traceId", traceId), zap.Int64("role_id", req.GetRoleId()))
		return nil, errors.New("character avatar is empty")
	}
	var isAllowEdit = false
	// 仅创建者自己编辑
	switch roleinfo.EditScope {
	case 1:
		if roleinfo.CreatorID != req.GetUserId() {
			log.Log().Warn("GenerateStoryRoleVideo failed: no permission", zap.String("traceId", traceId), zap.Int64("creator_id", roleinfo.CreatorID), zap.Int64("user_id", req.GetUserId()))
			return nil, errors.New("have no permission")
		}
		isAllowEdit = true
	case 2: // 角色所在小组内都可以编辑
		storyinfo, err := models.GetStory(ctx, roleinfo.StoryID)
		if err != nil {
			log.Log().Error("GenerateStoryRoleVideo failed: get story error", zap.String("traceId", traceId), zap.Error(err))
			return nil, err
		}
		_, err = models.GetGroupMemberByGroupAndUser(ctx, int64(storyinfo.GroupID), req.GetUserId())
		if err != nil {
			log.Log().Error("GenerateStoryRoleVideo failed: get group member by group and user error", zap.String("traceId", traceId), zap.Error(err))
			return nil, err
		}
		isAllowEdit = true
	default:
		// 其他情况不允许编辑
		log.Log().Warn("GenerateStoryRoleVideo failed: edit scope not allow", zap.String("traceId", traceId))
		return nil, errors.New("have no permission")
	}
	if isAllowEdit != true {
		log.Log().Warn("GenerateStoryRoleVideo failed: no permission", zap.String("traceId", traceId), zap.Int64("creator_id", roleinfo.CreatorID), zap.Int64("user_id", req.GetUserId()))
		return nil, errors.New("have no permission")
	}
	prompt := req.GetTextPrompt()
	refAvatarUrl := req.GetRefAvatarUrl()
	if refAvatarUrl == "" {
		refAvatarUrl = roleinfo.CharacterAvatar
	}

	doubaoClient := doubao.NewVideoClient(client.DoubaoAPIKey)
	videoGenPramas := doubao.VideoGenerationRequest{
		Model: "doubao-seedance-1-0-pro-250528",
		Content: []doubao.ContentItem{
			{
				Type: "text",
				Text: prompt,
			},
			{
				Type: "image_url",
				ImageURL: &doubao.ImageURL{
					URL: refAvatarUrl,
				},
			},
			{
				Type: "image_url",
				ImageURL: &doubao.ImageURL{
					URL: req.GetRefBackgroundUrl(),
				},
			},
		},
	}
	videoGenResult, err := doubaoClient.CreateVideoGenerationTask(ctx, &videoGenPramas)
	if err != nil {
		log.Log().Error("GenerateStoryRoleVideo failed: generate video error", zap.String("traceId", traceId), zap.Error(err))
		return nil, err
	}
	if videoGenResult.ID == "" {
		log.Log().Error("GenerateStoryRoleVideo failed: generate video error", zap.String("traceId", traceId), zap.Error(err))
		return nil, errors.New("generate video error")
	}
	taskUUid := uuid.NewString()
	videoGentask := &models.VideoGen{
		TaskId:    taskUUid,
		UUID:      videoGenResult.ID,
		StoryId:   roleinfo.StoryID,
		BoardId:   0,
		SceneId:   0,
		RoleId:    int64(roleinfo.ID),
		UserID:    req.GetUserId(),
		TaskType:  api.RenderType_RENDER_TYPE_STORYCHARACTERS,
		GenStatus: api.StoryGenStatus_STORY_GEN_STATUS_RUNNING,
		VideoUrl:  "",
		Prompt:    prompt,
		RefImages: roleinfo.CharacterAvatar,
		Seed:      int64(rand.Intn(10000000)),
		Code:      "",
		Message:   "OK",
		Deleted:   0,
		Tokens:    1,
		Provider:  "doubao",
		StartTime: time.Now().Unix(),
		EndTime:   0,
	}
	_, err = models.CreateVideoGen(ctx, videoGentask)
	if err != nil {
		log.Log().Error("GenerateStoryRoleVideo failed: create video gen error", zap.String("traceId", traceId), zap.Error(err))
		return nil, err
	}
	log.Log().Info("GenerateStoryRoleVideo: generate video success", zap.String("traceId", traceId), zap.Any("videoGenResult", videoGenResult))
	return &api.GenerateStoryRoleVideoResponse{
		Code:    api.ResponseCode_OK,
		Message: "OK",
		Detail: &api.GenerateStoryRoleVideoTaskDetail{
			VideoUrl:   "",
			TaskId:     taskUUid,
			TaskStatus: api.StoryGenStatus_STORY_GEN_STATUS_FINISHED,
		},
	}, nil
}

func (s *StoryroleService) LikeStoryRolePoster(ctx context.Context, req *api.LikeStoryRolePosterRequest) (*api.LikeStoryRolePosterResponse, error) {
	traceId := utils.GetTraceID(ctx)
	log.Log().Info("LikeStoryRolePoster called", zap.String("traceId", traceId), zap.Any("req", req))
	role, err := models.GetStoryRoleByID(ctx, req.GetRoleId())
	if err != nil {
		log.Log().Error("LikeStoryRolePoster failed: get story role by id error", zap.String("traceId", traceId), zap.Error(err))
		return nil, err
	}
	created, err := models.LikeRolePoster(ctx, req.GetPosterId(), req.GetUserId())
	if err != nil {
		log.Log().Error("LikeStoryRolePoster failed: like story role poster error", zap.String("traceId", traceId), zap.Error(err))
		return nil, err
	}
	if !created {
		log.Log().Info("LikeStoryRolePoster: already liked", zap.String("traceId", traceId), zap.Int64("role_id", int64(role.ID)))
		return &api.LikeStoryRolePosterResponse{
			Code:    api.ResponseCode_OK,
			Message: "already liked",
		}, nil
	}
	log.Log().Info("LikeStoryRolePoster success", zap.String("traceId", traceId), zap.Int64("role_id", int64(role.ID)))
	return &api.LikeStoryRolePosterResponse{
		Code:    api.ResponseCode_OK,
		Message: api.ResponseCode_OK.String(),
	}, nil
}

func (s *StoryroleService) UnLikeStoryRolePoster(ctx context.Context, req *api.UnLikeStoryRolePosterRequest) (*api.UnLikeStoryRolePosterResponse, error) {
	traceId := utils.GetTraceID(ctx)
	log.Log().Info("UnLikeStoryRolePoster called", zap.String("traceId", traceId), zap.Any("req", req))
	removed, err := models.DislikeRolePoster(ctx, req.GetPosterId(), req.GetUserId())
	if err != nil {
		log.Log().Error("UnLikeStoryRolePoster failed: unlike story role poster error", zap.String("traceId", traceId), zap.Error(err))
		return nil, err
	}
	if !removed {
		log.Log().Info("UnLikeStoryRolePoster: already unliked", zap.String("traceId", traceId), zap.Int64("poster_id", int64(req.GetPosterId())))
		return &api.UnLikeStoryRolePosterResponse{
			Code:    api.ResponseCode_OK,
			Message: "not liked",
		}, nil
	}
	log.Log().Info("UnLikeStoryRolePoster success", zap.String("traceId", traceId), zap.Int64("role_id", req.GetRoleId()))
	return &api.UnLikeStoryRolePosterResponse{
		Code:    api.ResponseCode_OK,
		Message: api.ResponseCode_OK.String(),
	}, nil
}

func (s *StoryroleService) GetStoryRolePosterList(ctx context.Context, req *api.GetStoryRolePosterListRequest) (*api.GetStoryRolePosterListResponse, error) {
	traceId := utils.GetTraceID(ctx)
	log.Log().Info("GetStoryRolePosterList called", zap.String("traceId", traceId), zap.Any("req", req))

	rolePosters, total, hasMore, err := models.GetRolePosterByRoleId(ctx, req.GetRoleId(), req.GetOffset(), req.GetPageSize())
	if err != nil {
		log.Log().Error("GetStoryRolePosterList failed: get role posters by role id error", zap.String("traceId", traceId), zap.Error(err))
		return nil, err
	}
	log.Log().Info("GetStoryRolePosterList: get role posters by role id success", zap.String("traceId", traceId), zap.Int64("role_id", req.GetRoleId()), zap.Int("total", total), zap.Bool("hasMore", hasMore))
	creatorIds := make([]int64, 0, len(rolePosters))
	posterIds := make([]int64, 0, len(rolePosters))
	for _, poster := range rolePosters {
		creatorIds = append(creatorIds, poster.CreatorId)
		posterIds = append(posterIds, int64(poster.ID))
	}
	creatorsMap, err := models.GetUsersByIdsMap(ctx, creatorIds)
	if err != nil {
		log.Log().Error("GetStoryRolePosterList failed: get users by ids map error", zap.String("traceId", traceId), zap.Error(err))
		return nil, err
	}
	userLikedPosters, err := models.GetUserLikedPostersWithPosterIds(ctx, req.GetUserId(), posterIds)
	if err != nil {
		log.Log().Error("GetStoryRolePosterList failed: get user liked posters with poster ids error", zap.String("traceId", traceId), zap.Error(err))
		return nil, err
	}
	userLikedPostersMap := make(map[int64]*models.RolePoster)
	for _, poster := range userLikedPosters {
		userLikedPostersMap[int64(poster.ID)] = poster
	}
	var ret = make([]*api.RolePosterDetail, 0, len(rolePosters))
	for _, poster := range rolePosters {
		ret = append(ret, &api.RolePosterDetail{
			Id:            int64(poster.ID),
			LikeCount:     int64(poster.LikedCount),
			PosterUrl:     poster.PosterURL,
			CreatedAt:     poster.CreateAt.Unix(),
			Prompt:        poster.Prompt,
			IsLikedByUser: userLikedPostersMap[int64(poster.ID)] != nil,
			Creator: &api.UserInfo{
				UserId: poster.CreatorId,
				Name:   creatorsMap[int(poster.CreatorId)].Name,
				Avatar: creatorsMap[int(poster.CreatorId)].Avatar,
			},
		})
	}
	log.Log().Info("GetStoryRolePosterList success", zap.String("traceId", traceId), zap.Int64("role_id", req.GetRoleId()))
	return &api.GetStoryRolePosterListResponse{
		Code:     api.ResponseCode_OK,
		Message:  "OK",
		Posters:  ret,
		Total:    int64(total),
		HaveMore: hasMore,
	}, nil
}
