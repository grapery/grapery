package story

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	api "github.com/grapery/common-protoc/gen"
	"github.com/grapery/grapery/models"
	"github.com/grapery/grapery/pkg/active"
	"github.com/grapery/grapery/pkg/cache"
	"github.com/grapery/grapery/pkg/client"
	"github.com/grapery/grapery/pkg/cloud/aliyun"
	"github.com/grapery/grapery/pkg/cloud/coze"
	llmchatservice "github.com/grapery/grapery/service/llmchat"
	"github.com/grapery/grapery/utils"
	"github.com/grapery/grapery/utils/compliance"
	"github.com/grapery/grapery/utils/convert"
	"github.com/grapery/grapery/utils/log"
	"gorm.io/gorm"
)

var storyboardServer StoryboardServer

func init() {
	storyboardServer = NewStoryboardService()
}

func GetStoryboardServer() StoryboardServer {
	if storyboardServer == nil {
		storyboardServer = NewStoryboardService()
	}
	return storyboardServer
}

func NewStoryboardService() *StoryboardService {
	return &StoryboardService{
		bailianClient: client.NewAliyunClient(),
		doubaoClient:  client.NewDoubaoClient(),
		helper:        GetStoryHelper(),
	}
}

type StoryboardService struct {
	bailianClient *client.AliyunStoryClient
	doubaoClient  *client.DoubaoClient
	cozeClient    *coze.HuoShanCozeClient
	helper        HelperServer
}

type StoryboardServer interface {
	CreateStoryboard(ctx context.Context, req *api.CreateStoryboardRequest) (resp *api.CreateStoryboardResponse, err error)
	GetStoryboard(ctx context.Context, req *api.GetStoryboardRequest) (resp *api.GetStoryboardResponse, err error)
	UpdateStoryboard(ctx context.Context, req *api.UpdateStoryboardRequest) (resp *api.UpdateStoryboardResponse, err error)
	GetStoryboards(ctx context.Context, req *api.GetStoryboardsRequest) (resp *api.GetStoryboardsResponse, err error)
	DelStoryboard(ctx context.Context, req *api.DelStoryboardRequest) (resp *api.DelStoryboardResponse, err error)
	ForkStoryboard(ctx context.Context, req *api.ForkStoryboardRequest) (resp *api.ForkStoryboardResponse, err error)
	LikeStoryboard(ctx context.Context, req *api.LikeStoryboardRequest) (resp *api.LikeStoryboardResponse, err error)
	UnLikeStoryboard(ctx context.Context, req *api.UnLikeStoryboardRequest) (*api.UnLikeStoryboardResponse, error)
	ShareStoryboard(ctx context.Context, req *api.ShareStoryboardRequest) (resp *api.ShareStoryboardResponse, err error)

	RenderStoryboard(ctx context.Context, req *api.RenderStoryboardRequest) (*api.RenderStoryboardResponse, error)
	GenStoryboardImages(ctx context.Context, req *api.GenStoryboardImagesRequest) (*api.GenStoryboardImagesResponse, error)
	GenStoryboardText(ctx context.Context, req *api.GenStoryboardTextRequest) (*api.GenStoryboardTextResponse, error)
	GetStoryBoardRender(ctx context.Context, req *api.GetStoryBoardRenderRequest) (*api.GetStoryBoardRenderResponse, error)

	ContinueRenderStory(ctx context.Context, req *api.ContinueRenderStoryRequest) (*api.ContinueRenderStoryResponse, error)

	GetStoryBoardGenerate(ctx context.Context, req *api.GetStoryBoardGenerateRequest) (*api.GetStoryBoardGenerateResponse, error)
	RenderStoryRoles(ctx context.Context, req *api.RenderStoryRolesRequest) (*api.RenderStoryRolesResponse, error)
	GetStoryRoles(ctx context.Context, req *api.GetStoryRolesRequest) (*api.GetStoryRolesResponse, error)
	GetStoryBoardRoles(ctx context.Context, req *api.GetStoryBoardRolesRequest) (*api.GetStoryBoardRolesResponse, error)

	RestoreStoryboard(ctx context.Context, req *api.RestoreStoryboardRequest) (*api.RestoreStoryboardResponse, error)
	GetUserCreatedStoryboards(ctx context.Context, req *api.GetUserCreatedStoryboardsRequest) (*api.GetUserCreatedStoryboardsResponse, error)

	GetNextStoryboard(ctx context.Context, req *api.GetNextStoryboardRequest) (*api.GetNextStoryboardResponse, error)

	CancelStoryboard(ctx context.Context, req *api.CancelStoryboardRequest) (*api.CancelStoryboardResponse, error)
	PublishStoryboard(ctx context.Context, req *api.PublishStoryboardRequest) (*api.PublishStoryboardResponse, error)

	GetUnPublishStoryboard(ctx context.Context, req *api.GetUnPublishStoryboardRequest) (*api.GetUnPublishStoryboardResponse, error)
	SaveStoryboardCraft(ctx context.Context, req *api.SaveStoryboardCraftRequest) (*api.SaveStoryboardCraftResponse, error)

	GetUserWatchStoryActiveStoryBoards(ctx context.Context, req *api.GetUserWatchStoryActiveStoryBoardsRequest) (*api.GetUserWatchStoryActiveStoryBoardsResponse, error)
	GetUserWatchRoleActiveStoryBoards(ctx context.Context, req *api.GetUserWatchRoleActiveStoryBoardsRequest) (*api.GetUserWatchRoleActiveStoryBoardsResponse, error)

	UserStoryboardDraftlist(ctx context.Context, req *api.UserStoryboardDraftlistRequest) (*api.UserStoryboardDraftlistResponse, error)
	DeleteUserStoryboardDraft(ctx context.Context, req *api.DeleteUserStoryboardDraftRequest) (*api.DeleteUserStoryboardDraftResponse, error)
	UserDraftStoryboardDetail(ctx context.Context, req *api.UserDraftStoryboardDetailRequest) (*api.UserDraftStoryboardDetailResponse, error)
	GetStoryboardGenerationRoadmap(ctx context.Context, req *api.GetStoryboardGenerationRoadmapRequest) (*api.GetStoryboardGenerationRoadmapResponse, error)
}

// 定义结构体用于解析章节生成结果
// StoryboardGenResult 表示整个返回结构
// ChapterSummary: 章节情节简述
// ChapterDetails: 章节详细情节（章节名->章节内容）
type StoryboardGenResult struct {
	ChapterSummary string                       `json:"章节情节简述"`
	ChapterDetails map[string]StoryboardChapter `json:"章节详细情节"`
}

// StoryboardChapter 表示每一章节的详细内容
// Content: 情节内容
// Characters: 参与人物
// ImagePrompt: 图片提示词
type StoryboardChapter struct {
	Content     string `json:"情节内容"`
	Characters  string `json:"参与人物"`
	ImagePrompt string `json:"图片提示词"`
}

func (s *StoryboardService) CreateStoryboard(ctx context.Context, req *api.CreateStoryboardRequest) (resp *api.CreateStoryboardResponse, err error) {
	log.Log().Info("[CreateStoryboard] 开始创建故事板",
		zap.Int64("storyId", req.Board.StoryId),
		zap.Int64("creatorId", req.Board.Creator),
		zap.String("boardTitle", req.Board.Title))
	log.Log().Info("[CreateStoryboard] 获取故事信息:", zap.Any("params", req.String()))
	newStroyBoard := ConvertApiStoryBoardToStoryBoard(req.GetBoard())
	log.Log().Info("[CreateStoryboard] 转换API故事板数据完成")
	isPass, err := compliance.TextCompliance(newStroyBoard.Description)
	if err != nil {
		log.Log().Error("[CreateStory] 简介合规检测失败", zap.Error(err))
		return nil, err
	}
	if !isPass {
		log.Log().Error("[CreateStory] 简介合规检测失败", zap.Error(err))
		return nil, err
	}
	storyInfo, err := models.GetStory(ctx, req.Board.StoryId)
	if err != nil {
		log.Log().Error("[CreateStoryboard] 获取故事信息失败",
			zap.Int64("storyId", req.Board.StoryId),
			zap.Error(err))
		return nil, err
	}
	log.Log().Info("[CreateStoryboard] 获取故事信息成功",
		zap.Int64("storyId", req.Board.StoryId),
		zap.Int("storyStatus", int(storyInfo.Status)),
		zap.Bool("isAIGen", storyInfo.AIGen))

	if storyInfo.IsClose {
		log.Log().Warn("[CreateStoryboard] 故事已关闭，无法创建故事板",
			zap.Int64("storyId", req.Board.StoryId))
		return &api.CreateStoryboardResponse{
			Code:    int32(api.ResponseCode_STORY_ARCHIVED),
			Message: "story is closed",
		}, nil
	}
	for i, role := range req.GetBoard().GetRoles() {
		// 如果角色的头像/描述为空，需要返回创建故事版失败
		if role.CharacterAvatar == "" || role.CharacterDescription == "" {
			log.Log().Error("[CreateStoryboard] 角色头像/描述为空，无法创建故事板",
				zap.Int("roleIndex", i),
				zap.String("characterName", role.CharacterName))
			return &api.CreateStoryboardResponse{
				Code:    int32(api.ResponseCode_ROLE_RENDER_ERROR),
				Message: "role avatar/description is empty",
			}, nil
		}
		if role.CharacterName == "" {
			log.Log().Error("[CreateStoryboard] 角色名称为空，无法创建故事板",
				zap.Int("roleIndex", i),
				zap.String("characterName", role.CharacterName))
			return &api.CreateStoryboardResponse{
				Code:    int32(api.ResponseCode_ROLE_RENDER_ERROR),
				Message: "role name is empty",
			}, nil
		}
		// if role.RoleId != 0 {
		// 	log.Log().Error("[CreateStoryboard] 角色ID为空，无法创建故事板",
		// 		zap.Int("roleIndex", i),
		// 		zap.Int("roleId", int(role.RoleId)))
		// 	return &api.CreateStoryboardResponse{
		// 		Code:    int32(api.ResponseCode_ROLE_RENDER_ERROR),
		// 		Message: "role id is empty",
		// 	}, nil
		// }
	}

	newStroyBoard.IsAiGen = storyInfo.AIGen
	newStroyBoard.StoryID = req.Board.StoryId
	newStroyBoard.CreatorID = req.Board.Creator
	newStroyBoard.ForkAble = true
	newStroyBoard.Status = 1
	newStroyBoard.Seed = rand.Intn(1000000000)
	log.Log().Info("[CreateStoryboard] 设置故事板基础信息完成")

	storyBoardId, err := models.CreateStoryBoard(ctx, newStroyBoard)
	if err != nil {
		log.Log().Error("[CreateStoryboard] 创建故事板失败",
			zap.Int64("storyId", req.Board.StoryId),
			zap.Int64("creatorId", req.Board.Creator),
			zap.Error(err))
		return nil, err
	}
	log.Log().Info("[CreateStoryboard] 创建故事板成功",
		zap.Int64("storyBoardId", storyBoardId),
		zap.Int64("storyId", req.Board.StoryId))
	creatorProfile := &models.UserProfile{UserId: req.Board.Creator}
	if err := creatorProfile.IncrementCreatedBoardNum(ctx); err != nil {
		log.Log().Warn("[CreateStoryboard] 增加用户创建故事板数量失败",
			zap.Int64("storyBoardId", storyBoardId),
			zap.Int64("creatorId", req.Board.Creator),
			zap.Error(err))
	}

	newStroyBoard.ID = uint(storyBoardId)
	if storyInfo.RootBoardID == 0 {
		log.Log().Info("[CreateStoryboard] 更新故事根故事板ID",
			zap.Int64("storyId", req.Board.StoryId),
			zap.Int64("rootBoardId", storyBoardId))
		err = models.UpdateStorySpecColumns(ctx, req.Board.StoryId, map[string]interface{}{
			"root_board_id": storyBoardId,
		})
		if err != nil {
			log.Log().Error("[CreateStoryboard] 更新故事根故事板ID失败",
				zap.Int64("storyId", req.Board.StoryId),
				zap.Int64("rootBoardId", storyBoardId),
				zap.Error(err))
			return nil, err
		}
		log.Log().Info("[CreateStoryboard] 更新故事根故事板ID成功")
	}

	if len(req.GetBoard().GetRoles()) > 0 {
		log.Log().Info("[CreateStoryboard] 开始创建故事板角色",
			zap.Int("roleCount", len(req.GetBoard().GetRoles())))
		for i, role := range req.GetBoard().GetRoles() {
			roleInfo := new(models.StoryBoardRole)
			roleInfo.BoardId = storyBoardId
			roleInfo.RoleId = role.RoleId
			roleInfo.Name = role.CharacterName
			roleInfo.Avatar = role.CharacterAvatar
			roleInfo.StoryId = req.GetBoard().GetStoryId()
			roleInfo.CreatorId = req.GetBoard().GetCreator()
			roleInfo.Status = 1
			roleInfo.IsMain = 1
			roleInfo.IsPublished = 1

			_, err = models.CreateStoryBoardRole(ctx, roleInfo)
			if err != nil {
				log.Log().Error("[CreateStoryboard] 创建故事板角色失败",
					zap.Int64("storyBoardId", storyBoardId),
					zap.Int64("roleId", role.RoleId),
					zap.String("roleName", role.CharacterName),
					zap.Int("roleIndex", i),
					zap.Error(err))
				return nil, err
			}
			newStroyBoard.RoleNum++
		}
		log.Log().Info("[CreateStoryboard] 所有故事板角色创建完成",
			zap.Int64("storyBoardId", storyBoardId),
			zap.Int("totalRoleCount", len(req.GetBoard().GetRoles())))
	}
	err = models.UpdateStoryBoardRoleNum(ctx, storyBoardId, newStroyBoard.RoleNum)
	if err != nil {
		log.Log().Error("[CreateStoryboard] 更新故事板角色数量失败",
			zap.Int64("storyBoardId", storyBoardId),
			zap.Int("roleNum", newStroyBoard.RoleNum),
			zap.Error(err))
	}
	group := &models.Group{}
	group.ID = uint(storyInfo.GroupID)
	err = group.GetByID(ctx)
	if err != nil {
		log.Log().Error("[CreateStoryboard] 获取群组信息失败",
			zap.Uint("groupId", uint(storyInfo.GroupID)),
			zap.Error(err))
		return nil, err
	}
	log.Log().Info("[CreateStoryboard] 获取群组信息成功",
		zap.Uint("groupId", uint(storyInfo.GroupID)))

	active.GetActiveServer().WriteStoryActive(ctx, group, storyInfo, newStroyBoard,
		nil, req.GetBoard().GetCreator(), api.ActiveType_NewStoryBoard)
	log.Log().Info("[CreateStoryboard] 写入故事活动成功",
		zap.Int64("storyBoardId", storyBoardId),
		zap.Int64("creatorId", req.GetBoard().GetCreator()))

	log.Log().Info("[CreateStoryboard] 创建故事板完成",
		zap.Int64("storyBoardId", storyBoardId),
		zap.Int64("storyId", req.Board.StoryId),
		zap.Int64("creatorId", req.Board.Creator))

	// 清除相关缓存，新创建的故事板需要让相关列表缓存失效
	storyBoardCache := cache.GetStoryBoardCache()
	if cacheErr := storyBoardCache.InvalidateStoryRelatedCache(ctx, req.Board.StoryId); cacheErr != nil {
		log.Log().Warn("[CreateStoryboard] 清除故事相关缓存失败",
			zap.Int64("storyId", req.Board.StoryId),
			zap.Error(cacheErr))
	}
	if cacheErr := storyBoardCache.InvalidateUserStoryBoardCache(ctx, req.Board.Creator); cacheErr != nil {
		log.Log().Warn("[CreateStoryboard] 清除用户故事板缓存失败",
			zap.Int64("creatorId", req.Board.Creator),
			zap.Error(cacheErr))
	}

	return &api.CreateStoryboardResponse{
		Code:    0,
		Message: "create storyboard success",
		Data: &api.CreateStoryboardResponse_Data{
			BoardId: storyBoardId,
		},
	}, nil
}

func (s *StoryboardService) GetStoryboard(ctx context.Context, req *api.GetStoryboardRequest) (resp *api.GetStoryboardResponse, err error) {
	log.Log().Info("[GetStoryboard] 开始获取故事板信息",
		zap.Int64("boardId", req.BoardId))

	// 获取故事板缓存管理器
	storyBoardCache := cache.GetStoryBoardCache()

	// 尝试从缓存获取故事板信息
	boardInfo, err := storyBoardCache.GetStoryBoardDetail(ctx, req.BoardId)
	if err != nil {
		log.Log().Debug("[GetStoryboard] 故事板缓存未命中，从数据库获取",
			zap.Int64("boardId", req.BoardId),
			zap.Error(err))

		// 缓存未命中，从数据库获取
		boardInfo, err = models.GetStoryboard(ctx, req.BoardId)
		if err != nil {
			log.Log().Error("[GetStoryboard] 获取故事板信息失败",
				zap.Int64("boardId", req.BoardId),
				zap.Error(err))
			return nil, err
		}

		// 将故事板信息存入缓存
		if cacheErr := storyBoardCache.SetStoryBoardDetail(ctx, req.BoardId, boardInfo); cacheErr != nil {
			log.Log().Warn("[GetStoryboard] 设置故事板缓存失败",
				zap.Int64("boardId", req.BoardId),
				zap.Error(cacheErr))
		}
	} else {
		log.Log().Info("[GetStoryboard] 从缓存获取故事板信息成功",
			zap.Int64("boardId", req.BoardId))
	}
	log.Log().Info("[GetStoryboard] 获取故事板信息成功",
		zap.Int64("boardId", req.BoardId),
		zap.Int64("storyId", boardInfo.StoryID),
		zap.Int64("creatorId", boardInfo.CreatorID),
		zap.Int("status", boardInfo.Status))

	storyInfo, err := models.GetStory(ctx, boardInfo.StoryID)
	if err != nil {
		log.Log().Error("[GetStoryboard] 获取故事信息失败",
			zap.Int64("storyId", boardInfo.StoryID),
			zap.Error(err))
		return nil, err
	}
	log.Log().Info("[GetStoryboard] 获取故事信息成功",
		zap.Int64("storyId", boardInfo.StoryID),
		zap.Int("storyStatus", int(storyInfo.Status)),
		zap.String("storyTitle", storyInfo.Title))

	if storyInfo.IsClose && boardInfo.CreatorID != req.GetBoardId() {
		log.Log().Warn("[GetStoryboard] 故事已关闭且用户非创建者，无法获取故事板",
			zap.Int64("boardId", req.BoardId),
			zap.Int64("storyId", boardInfo.StoryID),
			zap.Int64("requestUserId", req.GetBoardId()),
			zap.Int64("creatorId", boardInfo.CreatorID))
		return &api.GetStoryboardResponse{
			Code:    0,
			Message: "story is closed",
		}, nil
	}

	// 尝试从缓存获取故事板场景
	sences, err := storyBoardCache.GetStoryBoardScenes(ctx, req.BoardId)
	if err != nil {
		log.Log().Debug("[GetStoryboard] 故事板场景缓存未命中，从数据库获取",
			zap.Int64("boardId", req.BoardId),
			zap.Error(err))

		// 缓存未命中，从数据库获取
		sences, err = models.GetStoryBoardScenesByBoard(ctx, req.BoardId)
		if err != nil {
			log.Log().Error("[GetStoryboard] 获取故事板场景列表失败",
				zap.Int64("boardId", req.BoardId),
				zap.Error(err))
		} else {
			log.Log().Info("[GetStoryboard] 从数据库获取故事板场景列表成功",
				zap.Int64("boardId", req.BoardId),
				zap.Int("sceneCount", len(sences)))

			// 将场景信息存入缓存
			if cacheErr := storyBoardCache.SetStoryBoardScenes(ctx, req.BoardId, sences); cacheErr != nil {
				log.Log().Warn("[GetStoryboard] 设置故事板场景缓存失败",
					zap.Int64("boardId", req.BoardId),
					zap.Error(cacheErr))
			}
		}
	} else {
		log.Log().Info("[GetStoryboard] 从缓存获取故事板场景列表成功",
			zap.Int64("boardId", req.BoardId),
			zap.Int("sceneCount", len(sences)))
	}

	board := ConvertStoryBoardToApiStoryBoard(boardInfo)
	log.Log().Info("[GetStoryboard] 转换故事板数据完成",
		zap.Int64("boardId", req.BoardId))

	if len(sences) != 0 {
		board.Sences = new(api.StoryBoardSences)
		for i, scene := range sences {
			sceneData, _ := json.Marshal(scene)
			log.Log().Info("[GetStoryboard] 处理场景数据",
				zap.Int64("boardId", req.BoardId),
				zap.Int("sceneIndex", i),
				zap.String("sceneData", string(sceneData)))
			board.Sences.List = append(board.Sences.List, ConvertStorySceneToApiScene(scene))
		}
		board.Sences.Total = int64(len(board.Sences.List))
		log.Log().Info("[GetStoryboard] 场景数据处理完成",
			zap.Int64("boardId", req.BoardId),
			zap.Int64("totalScenes", board.Sences.Total))
	}

	cu, err := s.helper.GetStoryboardCurrentUserStatus(ctx, req.BoardId)
	if err != nil {
		log.Log().Error("[GetStoryboard] 获取故事板当前用户状态失败",
			zap.Int64("boardId", req.BoardId),
			zap.Error(err))
	} else {
		log.Log().Info("[GetStoryboard] 获取故事板当前用户状态成功",
			zap.Int64("boardId", req.BoardId))
	}

	creator, err := models.GetUserById(ctx, int64(boardInfo.CreatorID))
	if err != nil {
		log.Log().Error("[GetStoryboard] 获取创建者信息失败",
			zap.Int64("creatorId", boardInfo.CreatorID),
			zap.Error(err))
		return nil, err
	}
	log.Log().Info("[GetStoryboard] 获取创建者信息成功",
		zap.Int64("creatorId", boardInfo.CreatorID),
		zap.String("creatorName", creator.Name))

	board.CurrentUserStatus = cu
	roles, err := models.GetStoryBoardRolesByBoard(ctx, req.BoardId)
	if err != nil {
		log.Log().Error("[GetStoryboard] 获取故事板角色失败",
			zap.Int64("boardId", req.BoardId),
			zap.Error(err))
	} else {
		log.Log().Info("[GetStoryboard] 获取故事板角色成功",
			zap.Int64("boardId", req.BoardId),
			zap.Int("roleCount", len(roles)))
	}

	apiRole := make([]*api.StoryBoardActiveRole, 0)
	for _, role := range roles {
		apiRole = append(apiRole, &api.StoryBoardActiveRole{
			RoleId:     int64(role.ID),
			RoleName:   role.Name,
			RoleAvatar: role.Avatar,
		})
	}
	boardActive := &api.StoryBoardActive{
		Storyboard:        board,
		TotalLikeCount:    int64(boardInfo.LikeNum),
		TotalCommentCount: int64(boardInfo.CommentNum),
		TotalShareCount:   int64(boardInfo.ShareNum),
		TotalForkCount:    int64(boardInfo.ForkNum),
		Mtime:             boardInfo.UpdateAt.Unix(),
		Roles:             apiRole,
		Creator: &api.StoryBoardActiveUser{
			UserId:     int64(creator.ID),
			UserName:   creator.Name,
			UserAvatar: creator.Avatar,
		},
		Summary: &api.StorySummaryInfo{
			StoryId:     int64(storyInfo.ID),
			StoryTitle:  storyInfo.Title,
			StoryAvatar: storyInfo.Avatar,
		},
	}
	log.Log().Info("[GetStoryboard] 构建故事板活动数据完成",
		zap.Int64("boardId", req.BoardId),
		zap.Int64("totalLikeCount", int64(boardInfo.LikeNum)),
		zap.Int64("totalCommentCount", int64(boardInfo.CommentNum)),
		zap.Int64("totalShareCount", int64(boardInfo.ShareNum)),
		zap.Int64("totalForkCount", int64(boardInfo.ForkNum)))

	return &api.GetStoryboardResponse{
		Code:    0,
		Message: "OK",
		Data: &api.GetStoryboardResponse_Data{
			BoardInfo: boardActive,
		},
	}, nil
}

func (s *StoryboardService) UpdateStoryboard(ctx context.Context, req *api.UpdateStoryboardRequest) (resp *api.UpdateStoryboardResponse, err error) {
	log.Log().Info("[UpdateStoryboard] 开始更新故事板",
		zap.Int64("boardId", req.BoardId),
		zap.Int64("requestUserId", req.GetBoardId()))

	boardInfo, err := models.GetStoryboard(ctx, req.BoardId)
	if err != nil {
		log.Log().Error("[UpdateStoryboard] 获取故事板信息失败",
			zap.Int64("boardId", req.BoardId),
			zap.Error(err))
		return nil, err
	}
	log.Log().Info("[UpdateStoryboard] 获取故事板信息成功",
		zap.Int64("boardId", req.BoardId),
		zap.Int64("creatorId", boardInfo.CreatorID),
		zap.Int64("requestUserId", req.GetBoardId()))

	if boardInfo.CreatorID != req.GetBoardId() {
		log.Log().Warn("[UpdateStoryboard] 用户非创建者，无法更新故事板",
			zap.Int64("boardId", req.BoardId),
			zap.Int64("creatorId", boardInfo.CreatorID),
			zap.Int64("requestUserId", req.GetBoardId()))
		return &api.UpdateStoryboardResponse{}, nil
	}

	needUpdateData := make(map[string]interface{})
	if req.Params != nil {
		paramsData, _ := json.Marshal(req.Params)
		needUpdateData["params"] = string(paramsData)
		log.Log().Info("[UpdateStoryboard] 解析更新参数",
			zap.Int64("boardId", req.BoardId),
			zap.String("params", string(paramsData)))
	}

	if len(needUpdateData) == 0 {
		log.Log().Warn("[UpdateStoryboard] 无更新数据，跳过更新",
			zap.Int64("boardId", req.BoardId))
		return &api.UpdateStoryboardResponse{}, nil
	}

	err = models.UpdateStoryboardMultiColumn(ctx, req.BoardId, needUpdateData)
	if err != nil {
		log.Log().Error("[UpdateStoryboard] 更新故事板失败",
			zap.Int64("boardId", req.BoardId),
			zap.Error(err))
		return nil, err
	}

	log.Log().Info("[UpdateStoryboard] 更新故事板成功",
		zap.Int64("boardId", req.BoardId),
		zap.Int("updateFieldCount", len(needUpdateData)))

	// 清除相关缓存
	storyBoardCache := cache.GetStoryBoardCache()
	if cacheErr := storyBoardCache.InvalidateStoryBoardCache(ctx, req.BoardId); cacheErr != nil {
		log.Log().Warn("[UpdateStoryboard] 清除故事板缓存失败",
			zap.Int64("boardId", req.BoardId),
			zap.Error(cacheErr))
	}
	if cacheErr := storyBoardCache.InvalidateStoryRelatedCache(ctx, boardInfo.StoryID); cacheErr != nil {
		log.Log().Warn("[UpdateStoryboard] 清除故事相关缓存失败",
			zap.Int64("storyId", boardInfo.StoryID),
			zap.Error(cacheErr))
	}
	if cacheErr := storyBoardCache.InvalidateUserStoryBoardCache(ctx, boardInfo.CreatorID); cacheErr != nil {
		log.Log().Warn("[UpdateStoryboard] 清除用户故事板缓存失败",
			zap.Int64("creatorId", boardInfo.CreatorID),
			zap.Error(cacheErr))
	}

	return &api.UpdateStoryboardResponse{
		Code:    0,
		Message: "OK",
	}, nil
}

func (s *StoryboardService) GetStoryboards(ctx context.Context, req *api.GetStoryboardsRequest) (resp *api.GetStoryboardsResponse, err error) {
	log.Log().Info("[GetStoryboards] 开始获取故事板列表",
		zap.Int64("storyId", req.StoryId),
		zap.Int32("page", req.Page),
		zap.Int32("pageSize", req.PageSize))

	// 获取故事板缓存管理器
	storyBoardCache := cache.GetStoryBoardCache()

	// 尝试从缓存获取故事板列表
	boardList, err := storyBoardCache.GetStoryBoardList(ctx, req.StoryId, int(req.Page), int(req.PageSize))
	if err != nil {
		log.Log().Debug("[GetStoryboards] 故事板列表缓存未命中，从数据库获取",
			zap.Int64("storyId", req.StoryId),
			zap.Int32("page", req.Page),
			zap.Int32("pageSize", req.PageSize),
			zap.Error(err))

		// 缓存未命中，从数据库获取
		boardList, err = models.GetStoryboardsByStoryMultiPage(ctx, req.StoryId, int(req.Page), int(req.PageSize))
		if err != nil {
			log.Log().Error("[GetStoryboards] 获取故事板列表失败",
				zap.Int64("storyId", req.StoryId),
				zap.Int32("page", req.Page),
				zap.Int32("pageSize", req.PageSize),
				zap.Error(err))
			return nil, err
		}

		// 将故事板列表存入缓存
		if cacheErr := storyBoardCache.SetStoryBoardList(ctx, req.StoryId, int(req.Page), int(req.PageSize), boardList); cacheErr != nil {
			log.Log().Warn("[GetStoryboards] 设置故事板列表缓存失败",
				zap.Int64("storyId", req.StoryId),
				zap.Int32("page", req.Page),
				zap.Int32("pageSize", req.PageSize),
				zap.Error(cacheErr))
		}
	} else {
		log.Log().Info("[GetStoryboards] 从缓存获取故事板列表成功",
			zap.Int64("storyId", req.StoryId),
			zap.Int32("page", req.Page),
			zap.Int32("pageSize", req.PageSize))
	}
	log.Log().Info("[GetStoryboards] 获取故事板列表成功",
		zap.Int64("storyId", req.StoryId),
		zap.Int("boardCount", len(boardList)))

	story, err := models.GetStory(ctx, req.StoryId)
	if err != nil {
		log.Log().Error("[GetStoryboards] 获取故事信息失败",
			zap.Int64("storyId", req.StoryId),
			zap.Error(err))
		return nil, err
	}
	srcBoardMap := make(map[int64]*models.StoryBoard)
	apiBoardsActive := make([]*api.StoryBoardActive, 0)
	log.Log().Info("[GetStoryboards] 开始处理故事板数据",
		zap.Int("totalBoardCount", len(boardList)))

	for i, board := range boardList {
		log.Log().Info("[GetStoryboards] 处理故事板",
			zap.Int64("storyId", req.StoryId),
			zap.Int64("boardId", int64(board.ID)),
			zap.Int("boardIndex", i))

		// 尝试从缓存获取故事板场景
		sences, err := storyBoardCache.GetStoryBoardScenes(ctx, int64(board.ID))
		if err != nil {
			log.Log().Debug("[GetStoryboards] 故事板场景缓存未命中，从数据库获取",
				zap.Int64("boardId", int64(board.ID)),
				zap.Error(err))

			// 缓存未命中，从数据库获取
			sences, err = models.GetStoryBoardScenesByBoard(ctx, int64(board.ID))
			if err != nil {
				log.Log().Error("[GetStoryboards] 获取故事板场景失败",
					zap.Int64("boardId", int64(board.ID)),
					zap.Error(err))
			} else {
				log.Log().Info("[GetStoryboards] 从数据库获取故事板场景成功",
					zap.Int64("boardId", int64(board.ID)),
					zap.Int("sceneCount", len(sences)))

				// 将场景信息存入缓存
				if cacheErr := storyBoardCache.SetStoryBoardScenes(ctx, int64(board.ID), sences); cacheErr != nil {
					log.Log().Warn("[GetStoryboards] 设置故事板场景缓存失败",
						zap.Int64("boardId", int64(board.ID)),
						zap.Error(cacheErr))
				}
			}
		} else {
			log.Log().Info("[GetStoryboards] 从缓存获取故事板场景成功",
				zap.Int64("boardId", int64(board.ID)),
				zap.Int("sceneCount", len(sences)))
		}

		srcBoardMap[int64(board.ID)] = board
		boardInfo := ConvertStoryBoardToApiStoryBoard(board)

		if len(sences) != 0 {
			boardInfo.Sences = new(api.StoryBoardSences)
			for _, scene := range sences {
				boardInfo.Sences.List = append(boardInfo.Sences.List, ConvertStorySceneToApiScene(scene))
			}
			boardInfo.Sences.Total = int64(len(boardInfo.Sences.List))
			log.Log().Info("[GetStoryboards] 故事板场景数据处理完成",
				zap.Int64("boardId", int64(board.ID)),
				zap.Int64("totalScenes", boardInfo.Sences.Total))
		}

		cu, err := s.helper.GetStoryboardCurrentUserStatus(ctx, int64(board.ID))
		if err != nil {
			log.Log().Error("[GetStoryboards] 获取故事板当前用户状态失败",
				zap.Int64("boardId", int64(board.ID)),
				zap.Error(err))
		} else {
			log.Log().Info("[GetStoryboards] 获取故事板当前用户状态成功",
				zap.Int64("boardId", int64(board.ID)))
		}
		boardInfo.CurrentUserStatus = cu

		creator, err := models.GetUserById(ctx, board.CreatorID)
		if err != nil {
			log.Log().Error("[GetStoryboards] 获取故事板创建者失败",
				zap.Int64("boardId", int64(board.ID)),
				zap.Int64("creatorId", board.CreatorID),
				zap.Error(err))
		} else {
			log.Log().Info("[GetStoryboards] 获取故事板创建者成功",
				zap.Int64("boardId", int64(board.ID)),
				zap.Int64("creatorId", board.CreatorID),
				zap.String("creatorName", creator.Name))
		}

		roles, err := models.GetStoryBoardRolesByBoard(ctx, int64(board.ID))
		if err != nil {
			log.Log().Error("[GetStoryboards] 获取故事板角色失败",
				zap.Int64("boardId", int64(board.ID)),
				zap.Error(err))
		} else {
			log.Log().Info("[GetStoryboards] 获取故事板角色成功",
				zap.Int64("boardId", int64(board.ID)),
				zap.Int("roleCount", len(roles)))
		}

		apiRole := make([]*api.StoryBoardActiveRole, 0)
		for _, role := range roles {
			apiRole = append(apiRole, &api.StoryBoardActiveRole{
				RoleId:     int64(role.ID),
				RoleName:   role.Name,
				RoleAvatar: role.Avatar,
			})
		}

		apiBoardsActiveItem := &api.StoryBoardActive{
			Storyboard:        boardInfo,
			TotalLikeCount:    int64(srcBoardMap[int64(board.ID)].LikeNum),
			TotalCommentCount: int64(srcBoardMap[int64(board.ID)].CommentNum),
			TotalShareCount:   int64(srcBoardMap[int64(board.ID)].ShareNum),
			TotalForkCount:    int64(srcBoardMap[int64(board.ID)].ForkNum),
			Users:             []*api.StoryBoardActiveUser{},
			Roles:             apiRole,
			Creator: &api.StoryBoardActiveUser{
				UserId:     int64(creator.ID),
				UserName:   creator.Name,
				UserAvatar: creator.Avatar,
			},
			Summary: &api.StorySummaryInfo{
				StoryId:          int64(story.ID),
				StoryTitle:       story.Title,
				StoryAvatar:      story.Avatar,
				StoryDescription: story.Origin,
				CreateTime:       story.CreateAt.Unix(),
				CreateUserId:     story.CreatorID,
			},
			Mtime: boardInfo.Mtime,
		}
		apiBoardsActive = append(apiBoardsActive, apiBoardsActiveItem)
	}

	log.Log().Info("[GetStoryboards] 获取故事板列表完成",
		zap.Int64("storyId", req.StoryId),
		zap.Int("totalBoardCount", len(apiBoardsActive)),
		zap.Int32("page", req.Page),
		zap.Int32("pageSize", req.PageSize))

	return &api.GetStoryboardsResponse{
		Code:    0,
		Message: "OK",
		Data: &api.GetStoryboardsResponse_Data{
			List:  apiBoardsActive,
			Total: int64(len(apiBoardsActive)),
		},
	}, nil
}

func (s *StoryboardService) DelStoryboard(ctx context.Context, req *api.DelStoryboardRequest) (resp *api.DelStoryboardResponse, err error) {
	log.Log().Info("[DelStoryboard] 开始删除故事板",
		zap.Int64("boardId", req.BoardId))
	story, err := models.GetStory(ctx, req.StoryId)
	if err != nil {
		log.Log().Error("[DelStoryboard] 获取故事信息失败",
			zap.Int64("storyId", req.StoryId),
			zap.Error(err))
	}
	log.Log().Info("[DelStoryboard] 获取故事信息成功",
		zap.Int64("storyId", req.StoryId),
		zap.Int64("creatorId", story.CreatorID))
	// 1. Get current storyboard details
	currentBoard, err := models.GetStoryboard(ctx, req.BoardId)
	if err != nil {
		log.Log().Error("[DelStoryboard] 获取故事板信息失败",
			zap.Int64("boardId", req.BoardId),
			zap.Error(err))
		return nil, err
	}
	log.Log().Info("[DelStoryboard] 获取故事板信息成功",
		zap.Int64("boardId", req.BoardId),
		zap.Int64("storyId", currentBoard.StoryID),
		zap.Int64("creatorId", currentBoard.CreatorID),
		zap.Int64("prevId", currentBoard.PrevId))

	// 2. Get boards that have current board as their prevId
	childBoards, err := models.GetStoryboardsByPrevId(ctx, req.BoardId)
	if err != nil {
		log.Log().Error("[DelStoryboard] 获取子故事板列表失败",
			zap.Int64("boardId", req.BoardId),
			zap.Error(err))
		return nil, err
	}
	log.Log().Info("[DelStoryboard] 获取子故事板列表成功",
		zap.Int64("boardId", req.BoardId),
		zap.Int("childBoardCount", len(childBoards)))

	// 3. Update all child boards to point to current board's prevId
	if len(childBoards) > 0 {
		log.Log().Info("[DelStoryboard] 开始更新子故事板的prevId",
			zap.Int64("boardId", req.BoardId),
			zap.Int64("newPrevId", currentBoard.PrevId),
			zap.Int("childBoardCount", len(childBoards)))

		for i, childBoard := range childBoards {
			updateData := map[string]interface{}{
				"prev_id": currentBoard.PrevId,
			}
			err = models.UpdateStoryboardMultiColumn(ctx, int64(childBoard.ID), updateData)
			if err != nil {
				log.Log().Error("[DelStoryboard] 更新子故事板prevId失败",
					zap.Int64("boardId", req.BoardId),
					zap.Int64("childBoardId", int64(childBoard.ID)),
					zap.Int("childIndex", i),
					zap.Error(err))
				return nil, err
			}
		}
		log.Log().Info("[DelStoryboard] 所有子故事板prevId更新完成",
			zap.Int64("boardId", req.BoardId),
			zap.Int("updatedChildCount", len(childBoards)))
	}

	// 4. Mark current board as deleted
	needUpdateData := map[string]interface{}{
		"status": -1,
		"stage":  api.StoryboardStage_STORYBOARD_STAGE_DRAFT,
	}
	err = models.UpdateStoryboardMultiColumn(ctx, req.BoardId, needUpdateData)
	if err != nil {
		log.Log().Error("[DelStoryboard] 标记故事板为删除状态失败",
			zap.Int64("boardId", req.BoardId),
			zap.Error(err))
		return nil, err
	}
	log.Log().Info("[DelStoryboard] 标记故事板为删除状态成功",
		zap.Int64("boardId", req.BoardId))

	userProfile := &models.UserProfile{
		UserId: int64(currentBoard.CreatorID),
	}
	err = userProfile.DecrementCreatedBoardNum(ctx)
	if err != nil {
		log.Log().Error("[DelStoryboard] 减少用户创建故事板数量失败",
			zap.Int64("userId", int64(currentBoard.CreatorID)),
			zap.Error(err))
	} else {
		log.Log().Info("[DelStoryboard] 减少用户创建故事板数量成功",
			zap.Int64("userId", int64(currentBoard.CreatorID)))
	}
	err = models.DecreaseStoryBoardsNum(ctx, req.StoryId, 1)
	if err != nil {
		log.Log().Error("[DelStoryboard] 减少故事板数量失败",
			zap.Int64("storyId", req.StoryId),
			zap.Error(err))
	}

	boardRoles, err := models.GetStoryBoardRolesID(ctx, req.BoardId)
	if err != nil {
		log.Log().Error("[DelStoryboard] 获取故事板角色失败",
			zap.Int64("boardId", req.BoardId),
			zap.Error(err))
		return nil, err
	}
	err = models.BatchUpdateStoryBoardRoles(ctx, boardRoles, false)
	if err != nil {
		log.Log().Error("[DelStoryboard] 批量更新故事板角色状态失败",
			zap.Int64("boardId", req.BoardId),
			zap.Error(err))
		return nil, err
	}

	log.Log().Info("[DelStoryboard] 删除故事板完成",
		zap.Int64("boardId", req.BoardId),
		zap.Int64("storyId", currentBoard.StoryID),
		zap.Int("updatedChildCount", len(childBoards)))

	// 清除相关缓存
	storyBoardCache := cache.GetStoryBoardCache()
	if cacheErr := storyBoardCache.InvalidateStoryBoardCache(ctx, req.BoardId); cacheErr != nil {
		log.Log().Warn("[DelStoryboard] 清除故事板缓存失败",
			zap.Int64("boardId", req.BoardId),
			zap.Error(cacheErr))
	}
	if cacheErr := storyBoardCache.InvalidateStoryRelatedCache(ctx, currentBoard.StoryID); cacheErr != nil {
		log.Log().Warn("[DelStoryboard] 清除故事相关缓存失败",
			zap.Int64("storyId", currentBoard.StoryID),
			zap.Error(cacheErr))
	}
	if cacheErr := storyBoardCache.InvalidateUserStoryBoardCache(ctx, currentBoard.CreatorID); cacheErr != nil {
		log.Log().Warn("[DelStoryboard] 清除用户故事板缓存失败",
			zap.Int64("creatorId", currentBoard.CreatorID),
			zap.Error(cacheErr))
	}

	return &api.DelStoryboardResponse{
		Code:    0,
		Message: "OK",
	}, nil
}

func (s *StoryboardService) ForkStoryboard(ctx context.Context, req *api.ForkStoryboardRequest) (resp *api.ForkStoryboardResponse, err error) {
	log.Log().Info("[ForkStoryboard] 开始复制故事板",
		zap.Int64("prevBoardId", req.PrevBoardId),
		zap.Int64("userId", req.GetUserId()))
	story, err := models.GetStory(ctx, req.StoryId)
	if err != nil {
		log.Log().Error("[ForkStoryboard] 获取故事信息失败",
			zap.Int64("storyId", req.StoryId),
			zap.Error(err))
	}
	if story.IsClose {
		log.Log().Info("[ForkStoryboard] 故事已停止不能复制",
			zap.Int64("storyId", req.StoryId))
		return nil, errors.New("story is stoped")
	}
	log.Log().Info("[ForkStoryboard] 获取故事信息成功",
		zap.Int64("storyId", req.StoryId),
		zap.Int64("creatorId", story.CreatorID))
	originStoryBoard, err := models.GetStoryboard(ctx, req.PrevBoardId)
	if err != nil {
		log.Log().Error("[ForkStoryboard] 获取原始故事板失败",
			zap.Int64("prevBoardId", req.PrevBoardId),
			zap.Error(err))
		return nil, err
	}
	log.Log().Info("[ForkStoryboard] 获取原始故事板成功",
		zap.Int64("prevBoardId", req.PrevBoardId),
		zap.Int64("storyId", originStoryBoard.StoryID),
		zap.Int64("creatorId", originStoryBoard.CreatorID))

	newStoryBoard := new(models.StoryBoard)
	originData, err := json.Marshal(originStoryBoard)
	if err != nil {
		log.Log().Error("[ForkStoryboard] 序列化原始故事板失败",
			zap.Int64("prevBoardId", req.PrevBoardId),
			zap.Error(err))
		return nil, err
	}
	log.Log().Info("[ForkStoryboard] 序列化原始故事板成功",
		zap.Int64("prevBoardId", req.PrevBoardId))

	err = json.Unmarshal(originData, newStoryBoard)
	if err != nil {
		log.Log().Error("[ForkStoryboard] 反序列化原始故事板失败",
			zap.Int64("prevBoardId", req.PrevBoardId),
			zap.Error(err))
		return nil, err
	}
	log.Log().Info("[ForkStoryboard] 反序列化原始故事板成功",
		zap.Int64("prevBoardId", req.PrevBoardId))

	newStoryBoard.ID = 0
	newStoryBoard.CreatorID = req.GetUserId()
	newStoryBoard.CreateAt = time.Now()
	newStoryBoard.UpdateAt = time.Now()
	log.Log().Info("[ForkStoryboard] 设置新故事板基础信息",
		zap.Int64("prevBoardId", req.PrevBoardId),
		zap.Int64("newCreatorId", req.GetUserId()))

	id, err := models.CreateStoryBoard(ctx, newStoryBoard)
	if err != nil {
		log.Log().Error("[ForkStoryboard] 创建新故事板失败",
			zap.Int64("prevBoardId", req.PrevBoardId),
			zap.Int64("userId", req.GetUserId()),
			zap.Error(err))
		return nil, err
	}
	log.Log().Info("[ForkStoryboard] 创建新故事板成功",
		zap.Int64("prevBoardId", req.PrevBoardId),
		zap.Int64("newBoardId", int64(id)),
		zap.Int64("userId", req.GetUserId()))
	if err := (&models.UserProfile{UserId: req.GetUserId()}).IncrementCreatedBoardNum(ctx); err != nil {
		log.Log().Warn("[ForkStoryboard] 增加用户创建故事板数量失败",
			zap.Int64("newBoardId", int64(id)),
			zap.Int64("userId", req.GetUserId()),
			zap.Error(err))
	}
	if err := models.IncrementStoryBoardForkNum(ctx, req.PrevBoardId); err != nil {
		log.Log().Warn("[ForkStoryboard] 增加原故事板 fork 数失败",
			zap.Int64("prevBoardId", req.PrevBoardId),
			zap.Error(err))
	}
	err = models.IncreaseStoryBoardsNum(ctx, req.StoryId, 1)
	if err != nil {
		log.Log().Error("[ForkStoryboard] 增加故事板数量失败",
			zap.Int64("storyId", req.StoryId),
			zap.Error(err))
	}
	group := &models.Group{}
	group.ID = uint(story.GroupID)
	err = group.GetByID(ctx)
	if err != nil {
		log.Log().Error("[ForkStoryboard] 获取群组信息失败",
			zap.Uint("groupId", uint(story.GroupID)),
			zap.Error(err))
		return nil, err
	}
	log.Log().Info("[ForkStoryboard] 获取群组信息成功",
		zap.Uint("groupId", uint(story.GroupID)))

	active.GetActiveServer().WriteStoryActive(ctx, group, story, newStoryBoard,
		nil, req.GetUserId(), api.ActiveType_ForkStory)
	log.Log().Info("[ForkStoryboard] 写入故事活动成功",
		zap.Int64("newBoardId", int64(id)),
		zap.Int64("userId", req.GetUserId()))

	resp = &api.ForkStoryboardResponse{
		Code:    0,
		Message: "OK",
		Data: &api.ForkStoryboardResponse_Data{
			BoardId: int64(id),
		},
	}

	log.Log().Info("[ForkStoryboard] 复制故事板完成",
		zap.Int64("prevBoardId", req.PrevBoardId),
		zap.Int64("newBoardId", int64(id)),
		zap.Int64("userId", req.GetUserId()))

	return resp, nil
}

func (s *StoryboardService) LikeStoryboard(ctx context.Context, req *api.LikeStoryboardRequest) (resp *api.LikeStoryboardResponse, err error) {
	log.Log().Info("[LikeStoryboard] 开始点赞故事板",
		zap.Int64("boardId", req.BoardId),
		zap.Int64("storyId", req.GetStoryId()),
		zap.Int64("userId", req.GetUserId()))

	storyBoard, err := models.GetStoryboard(ctx, req.BoardId)
	if err != nil {
		log.Log().Error("[LikeStoryboard] 获取故事板信息失败",
			zap.Int64("boardId", req.BoardId),
			zap.Error(err))
		return nil, err
	}
	if storyBoard == nil {
		log.Log().Warn("[LikeStoryboard] 故事板不存在",
			zap.Int64("boardId", req.BoardId))
		return &api.LikeStoryboardResponse{
			Code:    int32(api.ResponseCode_OPERATION_FAILED),
			Message: "storyboard not found",
		}, nil
	}
	log.Log().Info("[LikeStoryboard] 获取故事板信息成功",
		zap.Int64("boardId", req.BoardId),
		zap.Int64("storyId", storyBoard.StoryID),
		zap.Int64("creatorId", storyBoard.CreatorID))

	story, err := models.GetStory(ctx, storyBoard.StoryID)
	if err != nil {
		log.Log().Error("[LikeStoryboard] 获取故事信息失败",
			zap.Int64("storyId", storyBoard.StoryID),
			zap.Error(err))
		return nil, err
	}
	if story == nil {
		log.Log().Warn("[LikeStoryboard] 故事不存在",
			zap.Int64("storyId", storyBoard.StoryID))
		return &api.LikeStoryboardResponse{
			Code:    int32(api.ResponseCode_OPERATION_FAILED),
			Message: "story not found",
		}, nil
	}
	log.Log().Info("[LikeStoryboard] 获取故事信息成功",
		zap.Int64("storyId", storyBoard.StoryID),
		zap.String("storyTitle", story.Title),
		zap.Int64("groupId", int64(story.GroupID)))

	created, err := models.LikeStoryboard(ctx, req.GetUserId(), storyBoard.StoryID, req.GetBoardId(), int64(story.GroupID))
	if err != nil {
		log.Log().Error("[LikeStoryboard] 点赞故事板失败",
			zap.Int64("boardId", req.BoardId),
			zap.Int64("userId", req.GetUserId()),
			zap.Error(err))
		return nil, err
	}
	if !created {
		log.Log().Info("[LikeStoryboard] 用户已点赞，无需重复",
			zap.Int64("boardId", req.BoardId),
			zap.Int64("userId", req.GetUserId()))
		return &api.LikeStoryboardResponse{
			Code:    int32(api.ResponseCode_OK),
			Message: "already liked",
		}, nil
	}

	group := &models.Group{}
	group.ID = uint(story.GroupID)
	if err = group.GetByID(ctx); err != nil {
		log.Log().Error("[LikeStoryboard] 获取群组信息失败",
			zap.Uint("groupId", uint(story.GroupID)),
			zap.Error(err))
		return nil, err
	}
	log.Log().Info("[LikeStoryboard] 获取群组信息成功",
		zap.Uint("groupId", uint(story.GroupID)))

	active.GetActiveServer().WriteStoryActive(ctx, group, story, storyBoard,
		nil, req.GetUserId(), api.ActiveType_LikeStory)

	log.Log().Info("[LikeStoryboard] 点赞故事板完成",
		zap.Int64("boardId", req.BoardId),
		zap.Int64("storyId", req.GetStoryId()),
		zap.Int64("userId", req.GetUserId()))

	return &api.LikeStoryboardResponse{
		Code:    int32(api.ResponseCode_OK),
		Message: api.ResponseCode_OK.String(),
	}, nil
}

func (s *StoryboardService) ShareStoryboard(ctx context.Context, req *api.ShareStoryboardRequest) (resp *api.ShareStoryboardResponse, err error) {
	log.Log().Info("[ShareStoryboard] 开始分享故事板",
		zap.Int64("boardId", req.BoardId),
		zap.Int64("userId", req.GetUserId()))
	story, err := models.GetStory(ctx, req.StoryId)
	if err != nil {
		log.Log().Error("[ShareStoryboard] 获取故事信息失败",
			zap.Int64("storyId", req.StoryId),
			zap.Error(err))
		return nil, err
	}
	log.Log().Info("[ShareStoryboard] 获取故事信息成功",
		zap.Int64("storyId", req.StoryId),
		zap.String("storyTitle", story.Title))

	storyBoard, err := models.GetStoryboard(ctx, req.BoardId)
	if err != nil {
		log.Log().Error("[ShareStoryboard] 获取故事板信息失败",
			zap.Int64("boardId", req.BoardId),
			zap.Error(err))
		return nil, err
	}
	log.Log().Info("[ShareStoryboard] 获取故事板信息成功",
		zap.Int64("boardId", req.BoardId),
		zap.Int64("storyId", req.StoryId),
		zap.Int64("creatorId", storyBoard.CreatorID))

	group := &models.Group{}
	group.ID = uint(story.GroupID)
	err = group.GetByID(ctx)
	if err != nil {
		log.Log().Error("[ShareStoryboard] 获取群组信息失败",
			zap.Uint("groupId", uint(story.GroupID)),
			zap.Error(err))
		return nil, err
	}
	log.Log().Info("[ShareStoryboard] 获取群组信息成功",
		zap.Uint("groupId", uint(story.GroupID)))
	if err := models.IncrementStoryBoardShareNum(ctx, req.BoardId); err != nil {
		log.Log().Warn("[ShareStoryboard] 增加故事板分享数失败",
			zap.Int64("boardId", req.BoardId),
			zap.Error(err))
	}
	if err := models.IncreaseStoryShareNum(ctx, req.StoryId, 1); err != nil {
		log.Log().Warn("[ShareStoryboard] 增加故事分享数失败",
			zap.Int64("storyId", req.StoryId),
			zap.Error(err))
	}

	active.GetActiveServer().WriteStoryActive(ctx, group, story, storyBoard,
		nil, req.GetUserId(), api.ActiveType_ShareStoryBoard)

	log.Log().Info("[ShareStoryboard] 分享故事板完成",
		zap.Int64("boardId", req.BoardId),
		zap.Int64("userId", req.GetUserId()))

	return &api.ShareStoryboardResponse{
		Code:    0,
		Message: "OK",
	}, nil
}

// 用于创建离散的故事，故事情节离散，但是故事中的角色是一致的
func (s *StoryboardService) RenderStoryboard(ctx context.Context, req *api.RenderStoryboardRequest) (*api.RenderStoryboardResponse, error) {
	// 函数入口日志
	log.Log().Info("[RenderStoryboard] 开始渲染故事板",
		zap.Int64("boardId", req.GetBoardId()),
		zap.Int64("storyId", req.GetStoryId()),
		zap.Int64("userId", req.GetUserId()),
		zap.String("renderType", req.GetRenderType().String()))

	// 获取故事白板
	log.Log().Info("[RenderStoryboard] 开始获取故事白板", zap.Int64("boardId", req.GetBoardId()))
	board, err := models.GetStoryboard(ctx, req.GetBoardId())
	if err != nil {
		log.Log().Error("[RenderStoryboard] 获取故事白板失败",
			zap.Int64("boardId", req.GetBoardId()),
			zap.Error(err))
		return nil, err
	}
	log.Log().Info("[RenderStoryboard] 成功获取故事白板",
		zap.Uint("boardId", board.ID),
		zap.Int64("storyId", board.StoryID))

	// 获取故事
	log.Log().Info("[RenderStoryboard] 开始获取故事", zap.Int64("storyId", board.StoryID))
	story, err := models.GetStory(ctx, board.StoryID)
	if err != nil {
		log.Log().Error("[RenderStoryboard] 获取故事失败",
			zap.Int64("storyId", board.StoryID),
			zap.Error(err))
		return nil, err
	}
	log.Log().Info("[RenderStoryboard] 成功获取故事",
		zap.Uint("storyId", story.ID),
		zap.String("storyTitle", story.Title),
		zap.Int("status", int(story.Status)))

	if story.IsClose {
		log.Log().Warn("[RenderStoryboard] 故事已关闭，无法处理",
			zap.Uint("storyId", story.ID),
			zap.String("storyTitle", story.Title))
		return &api.RenderStoryboardResponse{
			Code:    -1,
			Message: "story is closed",
		}, nil
	}

	// 获取故事板生成记录
	log.Log().Info("[RenderStoryboard] 开始获取故事板生成记录", zap.Int64("boardId", req.GetBoardId()))
	stroyGen, err := models.GetStoryGensByStoryBoard(ctx, req.GetBoardId(), 1)
	if err != nil {
		log.Log().Error("[RenderStoryboard] 获取故事板生成记录失败",
			zap.Int64("boardId", req.GetBoardId()),
			zap.Error(err))
		return nil, err
	}
	log.Log().Info("[RenderStoryboard] 成功获取故事板生成记录",
		zap.Int64("boardId", req.GetBoardId()),
		zap.Int("genCount", len(stroyGen)))

	// 检查是否正在渲染
	if len(stroyGen) > 0 && stroyGen[0].Status == 1 {
		log.Log().Warn("[RenderStoryboard] 故事板正在渲染中，无法重复渲染",
			zap.Int64("boardId", req.GetBoardId()),
			zap.Int("status", stroyGen[0].Status))
		return &api.RenderStoryboardResponse{
			Code:    0,
			Message: "storyboard is rendering",
		}, nil
	}

	// 解析生成参数
	log.Log().Info("[RenderStoryboard] 开始解析生成参数")
	genParams := new(models.StoryBoardParams)
	genParams.StoryContent = story.Origin
	err = json.Unmarshal([]byte(board.Params), genParams)
	if err != nil {
		log.Log().Error("[RenderStoryboard] 解析故事板生成参数失败",
			zap.String("params", board.Params),
			zap.Error(err))
		return nil, err
	}

	// 创建故事生成记录
	log.Log().Info("[RenderStoryboard] 开始创建故事生成记录")
	storyGen := new(models.StoryGen)
	storyGen.UUID = uuid.New().String()
	storyGenData, _ := json.Marshal(genParams)
	storyParam := new(api.StoryParams)
	json.Unmarshal([]byte(story.Params), &storyParam)

	// 故事全局风格
	imageStyle := story.Style
	if imageStyle == "" {
		imageStyle = "吉卜力风格"
		log.Log().Info("[RenderStoryboard] 使用默认图片风格", zap.String("defaultStyle", imageStyle))
	} else {
		log.Log().Info("[RenderStoryboard] 使用自定义图片风格", zap.String("customStyle", imageStyle))
	}

	// 获取故事角色
	log.Log().Info("[RenderStoryboard] 开始获取故事板角色", zap.Int64("boardId", req.GetBoardId()))
	storyRoles, err := models.GetStoryBoardRoles(ctx, req.GetBoardId())
	if err != nil {
		log.Log().Error("[RenderStoryboard] 获取故事板角色失败",
			zap.Int64("boardId", req.GetBoardId()),
			zap.Error(err))
	}

	roleIds := make([]int64, 0)
	storyRolesStr := ""
	rolesMap := make(map[string]*models.StoryBoardRole)
	if len(storyRoles) != 0 {
		log.Log().Info("[RenderStoryboard] 找到故事板角色",
			zap.Int64("boardId", req.GetBoardId()),
			zap.Int("roleCount", len(storyRoles)))

		for _, role := range storyRoles {
			roleIds = append(roleIds, role.RoleId)
			rolesMap[fmt.Sprintf("%d", role.RoleId)] = role
		}

		roles, err := models.GetStoryRolesByIDs(ctx, roleIds)
		if err != nil {
			log.Log().Error("[RenderStoryboard] 根据ID获取故事角色失败",
				zap.Any("roleIds", roleIds),
				zap.Error(err))
		} else {
			log.Log().Info("[RenderStoryboard] 成功获取故事角色详情", zap.Int("roleCount", len(roles)))
		}

		for _, role := range roles {
			storyRolesStr += "角色id:" + fmt.Sprintf("%d", role.ID) + "," + "角色姓名:" + role.CharacterName + "," + "角色描述:" + role.CharacterDescription + ";\n"
		}
		log.Log().Info("[RenderStoryboard] 构建角色字符串完成", zap.String("rolesStr", storyRolesStr))
	} else {
		log.Log().Warn("[RenderStoryboard] 未找到故事板角色",
			zap.Int64("boardId", req.GetBoardId()),
			zap.Error(err))
	}

	// 构建故事板参数
	log.Log().Info("[RenderStoryboard] 开始构建故事板参数")
	var storyboardParams = coze.CozeStoryboardWriterParams{
		StoryCharacters: storyRolesStr,
		StoryContent:    board.Description,
		StoryBackground: story.Origin,
		StoryChapter:    board.Title,
		Seed:            int64(board.Seed),
		SceneNum:        int(story.SenceMaxNumber),
	}

	// 处理上一章节内容
	if board.PrevId != -1 && board.PrevId != 0 {
		log.Log().Info("[RenderStoryboard] 获取上一章节内容", zap.Int64("prevBoardId", board.PrevId))
		prevBoard, err := models.GetStoryboard(ctx, board.PrevId)
		if err != nil && !strings.Contains(strings.ToLower(err.Error()), "not found") {
			log.Log().Error("[RenderStoryboard] 获取上一章节失败",
				zap.Int64("prevBoardId", board.PrevId),
				zap.Error(err))
			return nil, err
		}
		storyboardParams.PrevContent = prevBoard.Description
		log.Log().Info("[RenderStoryboard] 成功获取上一章节内容",
			zap.Int64("prevBoardId", board.PrevId),
			zap.String("prevContent", prevBoard.Description))
	} else {
		storyboardParams.PrevContent = "暂无上一章节"
		log.Log().Info("[RenderStoryboard] 无上一章节，使用默认内容")
	}

	// 设置故事生成记录参数
	storyGen.LLmPlatform = "coze"
	storyGen.Params = string(storyGenData)
	storyGen.OriginID = int64(story.ID)
	storyGen.StartTime = time.Now().Unix()
	storyGen.BoardID = req.GetBoardId()
	storyGen.TaskType = api.RenderType_RENDER_TYPE_STORYBOARD
	storyGen.UserId = req.GetUserId()
	storyGen.GenStatus = api.StoryGenStatus_STORY_GEN_STATUS_INIT
	storyGen.Seed = board.Seed

	_, err = models.CreateStoryGen(ctx, storyGen)
	if err != nil {
		log.Log().Error("[RenderStoryboard] 创建故事生成记录失败",
			zap.String("uuid", storyGen.UUID),
			zap.Error(err))
		return nil, err
	}
	log.Log().Info("[RenderStoryboard] 成功创建故事生成记录", zap.String("uuid", storyGen.UUID))
	result := new(StoryChapter)
	start := time.Now()
	storyboardParams.ImageStyle = imageStyle
	ret, err := s.cozeClient.StoryboardWriter(ctx, storyboardParams)
	if err != nil {
		log.Log().Error("[RenderStoryboard] LLM生成故事板内容失败",
			zap.String("uuid", storyGen.UUID),
			zap.Error(err))
		return nil, err
	}

	genTime := time.Since(start)
	log.Log().Info("[RenderStoryboard] LLM生成故事板内容成功",
		zap.String("uuid", storyGen.UUID),
		zap.Duration("genTime", genTime),
		zap.String("rawResult", ret))

	// 清理LLM结果
	cleanResult := utils.CleanLLmJsonResult(ret)
	log.Log().Info("[RenderStoryboard] 清理LLM结果完成",
		zap.String("uuid", storyGen.UUID),
		zap.String("cleanResult", cleanResult))

	// 解析生成结果
	err = json.Unmarshal([]byte(cleanResult), &result)
	if err != nil {
		log.Log().Error("[RenderStoryboard] 解析故事生成结果失败",
			zap.String("uuid", storyGen.UUID),
			zap.String("cleanResult", cleanResult),
			zap.Error(err))
		return nil, err
	}
	log.Log().Info("[RenderStoryboard] 成功解析故事生成结果",
		zap.String("uuid", storyGen.UUID),
		zap.String("chapterTitle", result.ChapterSummary.Title))

	// 构建渲染详情
	log.Log().Info("[RenderStoryboard] 开始构建渲染详情")
	renderDetail := new(api.RenderStoryboardDetail)
	renderDetail.RenderType = req.RenderType
	renderDetail.Timecost = int32(genTime.Seconds())
	renderDetail.BoardId = req.BoardId
	renderDetail.StoryId = req.StoryId
	renderDetail.UserId = req.UserId
	renderDetail.Result = new(api.StoryChapter)

	// 转换StoryChapter到api.StoryChapter
	storyChapter := &api.StoryChapter{
		ChapterSummary: &api.ChapterSummary{
			Title:   result.ChapterSummary.Title,
			Content: result.ChapterSummary.Content,
		},
		ChapterDetailInfo: &api.ChapterDetailInformation{
			Details: make([]*api.DetailScene, 0),
		},
	}

	for _, detail := range result.ChapterDetailInfo {
		apiDetail := &api.DetailScene{
			Id:          detail.ID,
			Content:     detail.Content,
			ImagePrompt: detail.ImagePrompt,
			Characters:  make([]*api.Character, 0),
		}

		// 转换角色
		for _, char := range detail.Characters {
			apiChar := &api.Character{
				Id:          char.ID,
				Name:        char.Name,
				Description: char.Description,
			}
			if rolesMap[char.ID] != nil {
				apiChar.AvatarUrl = rolesMap[char.ID].Avatar
			}
			apiDetail.Characters = append(apiDetail.Characters, apiChar)
		}
		storyChapter.ChapterDetailInfo.Details = append(storyChapter.ChapterDetailInfo.Details, apiDetail)
	}

	renderDetail.Result = storyChapter
	// 更新故事生成记录
	renderDetailData, _ := json.Marshal(renderDetail)
	storyGen.Content = string(renderDetailData)
	storyGen.FinishTime = time.Now().Unix()
	storyGen.GenStatus = api.StoryGenStatus_STORY_GEN_STATUS_FINISHED
	storyGen.TaskId = uuid.New().String()

	err = models.UpdateStoryGen(ctx, storyGen)
	if err != nil {
		log.Log().Error("[RenderStoryboard] 更新故事生成记录失败",
			zap.String("uuid", storyGen.UUID),
			zap.Error(err))
	} else {
		log.Log().Info("[RenderStoryboard] 成功更新故事生成记录", zap.String("uuid", storyGen.UUID))
	}

	// 返回成功响应
	log.Log().Info("[RenderStoryboard] 渲染故事板完成",
		zap.Int64("boardId", req.GetBoardId()),
		zap.Int64("storyId", req.GetStoryId()),
		zap.String("uuid", storyGen.UUID),
		zap.Duration("totalTime", time.Since(start)))

	return &api.RenderStoryboardResponse{
		Code:    0,
		Message: "OK",
		Data:    renderDetail,
	}, nil
}

func (s *StoryboardService) GenStoryboardImages(ctx context.Context, req *api.GenStoryboardImagesRequest) (*api.GenStoryboardImagesResponse, error) {
	// 函数入口日志
	log.Log().Info("[GenStoryboardImages] 开始生成故事板图片",
		zap.Int64("boardId", req.GetBoardId()))

	// 获取故事白板
	log.Log().Info("[GenStoryboardImages] 开始获取故事白板", zap.Int64("boardId", req.GetBoardId()))
	board, err := models.GetStoryboard(ctx, req.GetBoardId())
	if err != nil {
		log.Log().Error("[GenStoryboardImages] 获取故事白板失败",
			zap.Int64("boardId", req.GetBoardId()),
			zap.Error(err))
		return nil, err
	}
	log.Log().Info("[GenStoryboardImages] 成功获取故事白板",
		zap.Uint("boardId", board.ID),
		zap.Int64("storyId", board.StoryID))

	// 获取故事
	log.Log().Info("[GenStoryboardImages] 开始获取故事", zap.Int64("storyId", board.StoryID))
	story, err := models.GetStory(ctx, board.StoryID)
	if err != nil {
		log.Log().Error("[GenStoryboardImages] 获取故事失败",
			zap.Int64("storyId", board.StoryID),
			zap.Error(err))
		return nil, err
	}
	log.Log().Info("[GenStoryboardImages] 成功获取故事",
		zap.Uint("storyId", story.ID),
		zap.String("storyTitle", story.Title),
		zap.Bool("isAchieve", story.IsClose),
		zap.Int("status", int(story.Status)))

	// 检查故事状态
	if story.IsClose {
		log.Log().Warn("[GenStoryboardImages] 故事已关闭，无法生成图片",
			zap.Uint("storyId", story.ID),
			zap.String("storyTitle", story.Title))
		return &api.GenStoryboardImagesResponse{
			Code:    int32(api.ResponseCode_STORY_ARCHIVED),
			Message: "story is closed",
		}, nil
	}

	// 获取故事板生成记录
	log.Log().Info("[GenStoryboardImages] 开始获取故事板生成记录", zap.Int64("boardId", req.BoardId))
	stroyboardGen, err := models.GetStoryGensByStoryBoard(ctx, req.BoardId, 1)
	if err != nil {
		log.Log().Error("[GenStoryboardImages] 获取故事板生成记录失败",
			zap.Int64("boardId", req.BoardId),
			zap.Error(err))
		return nil, err
	}
	log.Log().Info("[GenStoryboardImages] 成功获取故事板生成记录",
		zap.Int64("boardId", req.BoardId),
		zap.Int("genCount", len(stroyboardGen)))

	// 检查是否有生成记录
	if len(stroyboardGen) == 0 {
		log.Log().Warn("[GenStoryboardImages] 故事板未渲染，无法生成图片",
			zap.Int64("boardId", req.BoardId))
		return &api.GenStoryboardImagesResponse{
			Code:    int32(api.ResponseCode_STORYBOARD_STATUS_ERROR),
			Message: "storyboard is not rendering",
		}, nil
	}

	// 解析故事板生成结果
	log.Log().Info("[GenStoryboardImages] 开始解析故事板生成结果")
	var result StoryboardGenResult
	err = json.Unmarshal([]byte(stroyboardGen[0].Content), &result)
	if err != nil {
		log.Log().Error("[GenStoryboardImages] 解析故事板生成结果失败",
			zap.String("content", stroyboardGen[0].Content),
			zap.Error(err))
		return nil, err
	}
	log.Log().Info("[GenStoryboardImages] 成功解析故事板生成结果",
		zap.String("chapterSummary", result.ChapterSummary),
		zap.Int("chapterCount", len(result.ChapterDetails)))

	// 遍历每个章节生成图片
	log.Log().Info("[GenStoryboardImages] 开始遍历章节生成图片",
		zap.Int("totalChapters", len(result.ChapterDetails)))

	for chapterName, chapter := range result.ChapterDetails {
		log.Log().Info("[GenStoryboardImages] 开始处理章节",
			zap.String("chapterName", chapterName),
			zap.String("content", chapter.Content),
			zap.String("characters", chapter.Characters),
			zap.String("imagePrompt", chapter.ImagePrompt))

		// 分析角色数量
		charactors := strings.Split(chapter.Characters, ",")
		log.Log().Info("[GenStoryboardImages] 章节角色分析",
			zap.String("chapterName", chapterName),
			zap.String("characters", chapter.Characters),
			zap.Int("characterCount", len(charactors)))

		// 构建模板提示词
		storyboardImagePrompt := fmt.Sprintf("%s,风格为：%s", chapter.ImagePrompt, story.Style)
		templatePrompt := storyboardImagePrompt
		log.Log().Info("[GenStoryboardImages] 构建模板提示词",
			zap.String("chapterName", chapterName),
			zap.String("templatePrompt", templatePrompt))

		// 创建故事生成记录
		log.Log().Info("[GenStoryboardImages] 开始创建故事生成记录", zap.String("chapterName", chapterName))
		imageGen := new(models.ImageGen)
		imageGen.UUID = uuid.New().String()
		imageGen.Prompt = storyboardImagePrompt
		imageGen.StoryId = int64(story.ID)
		imageGen.StartTime = time.Now().Unix()
		imageGen.BoardId = req.GetBoardId()
		imageGen.TaskType = api.RenderType_RENDER_TYPE_STORYSENCE
		imageGen.Seed = int64(board.Seed)

		log.Log().Info("[GenStoryboardImages] 故事生成记录参数",
			zap.String("uuid", imageGen.UUID),
			zap.String("chapterName", chapterName),
			zap.String("llmPlatform", imageGen.LLmPlatform),
			zap.Uint("originId", story.ID),
			zap.Int64("boardId", imageGen.BoardId),
			zap.Int("genType", int(imageGen.TaskType)))

		_, err = models.CreateImageGen(ctx, imageGen)
		if err != nil {
			log.Log().Error("[GenStoryboardImages] 创建故事生成记录失败",
				zap.String("uuid", imageGen.UUID),
				zap.String("chapterName", chapterName),
				zap.Error(err))
			return nil, err
		}
		log.Log().Info("[GenStoryboardImages] 成功创建故事生成记录",
			zap.String("uuid", imageGen.UUID),
			zap.String("chapterName", chapterName))
		renderStoryParams := &client.GenStoryImagesParams{
			Content:  templatePrompt,
			RefImage: "",
			Seed:     int64(imageGen.Seed),
		}

		// 调用图片生成服务
		log.Log().Info("[GenStoryboardImages] 开始调用图片生成服务",
			zap.String("uuid", imageGen.UUID),
			zap.String("chapterName", chapterName))
		ret, err := s.doubaoClient.GenStoryBoardImage(ctx, renderStoryParams)
		if err != nil {
			log.Log().Error("[GenStoryboardImages] 图片生成失败",
				zap.String("uuid", imageGen.UUID),
				zap.String("chapterName", chapterName),
				zap.Error(err))
			return nil, err
		}
		log.Log().Info("[GenStoryboardImages] 图片生成成功",
			zap.String("uuid", imageGen.UUID),
			zap.String("chapterName", chapterName),
			zap.Int("imageCount", len(ret.ImageUrls)))

		// 上传图片到阿里云
		aliyunUrls := make([]string, 0)

		for i, imageUrl := range ret.ImageUrls {
			log.Log().Info("[GenStoryboardImages] 开始上传单张图片",
				zap.String("uuid", imageGen.UUID),
				zap.String("chapterName", chapterName),
				zap.Int("imageIndex", i),
				zap.String("imageUrl", imageUrl))

			aliyunClient := aliyun.GetGlobalClient()
			aliyunUrl, err := aliyunClient.UploadFileFromURL("", imageUrl)
			if err != nil {
				log.Log().Error("[GenStoryboardImages] 上传图片失败",
					zap.String("uuid", imageGen.UUID),
					zap.String("chapterName", chapterName),
					zap.Int("imageIndex", i),
					zap.String("imageUrl", imageUrl),
					zap.Error(err))
				continue
			}
			aliyunUrls = append(aliyunUrls, aliyunUrl)
			log.Log().Info("[GenStoryboardImages] 图片上传成功",
				zap.String("uuid", imageGen.UUID),
				zap.String("chapterName", chapterName),
				zap.Int("imageIndex", i),
				zap.String("aliyunUrl", aliyunUrl))
		}

		// 更新故事生成记录
		log.Log().Info("[GenStoryboardImages] 开始更新故事生成记录",
			zap.String("uuid", imageGen.UUID),
			zap.String("chapterName", chapterName),
			zap.Int("uploadedImageCount", len(aliyunUrls)))

		imageGen.ImageUrl = strings.Join(aliyunUrls, ",")
		imageGen.EndTime = time.Now().Unix()
		imageGen.GenStatus = api.StoryGenStatus_STORY_GEN_STATUS_FINISHED
		imageGen.TaskId = imageGen.UUID

		err = models.UpdateImageGen(ctx, imageGen)
		if err != nil {
			log.Log().Error("[GenStoryboardImages] 更新故事生成记录失败",
				zap.String("uuid", imageGen.UUID),
				zap.String("chapterName", chapterName),
				zap.Error(err))
		} else {
			log.Log().Info("[GenStoryboardImages] 成功更新故事生成记录",
				zap.String("uuid", imageGen.UUID),
				zap.String("chapterName", chapterName),
				zap.String("imageUrls", imageGen.ImageUrl))
		}

		log.Log().Info("[GenStoryboardImages] 章节图片生成完成",
			zap.String("uuid", imageGen.UUID),
			zap.String("chapterName", chapterName),
			zap.Int("totalImages", len(aliyunUrls)))
	}

	// 返回成功响应
	log.Log().Info("[GenStoryboardImages] 故事板图片生成完成",
		zap.Int64("boardId", req.GetBoardId()),
		zap.Int("totalChapters", len(result.ChapterDetails)))

	return &api.GenStoryboardImagesResponse{
		Code:    0,
		Message: "OK",
		Data:    nil,
	}, nil
}

func (s *StoryboardService) GenStoryboardText(ctx context.Context, req *api.GenStoryboardTextRequest) (*api.GenStoryboardTextResponse, error) {
	// 函数入口日志
	log.Log().Info("[GenStoryboardText] 开始生成故事板文本",
		zap.Int64("boardId", req.GetBoardId()))

	// 获取故事白板
	log.Log().Info("[GenStoryboardText] 开始获取故事白板", zap.Int64("boardId", req.GetBoardId()))
	board, err := models.GetStoryboard(ctx, req.GetBoardId())
	if err != nil {
		log.Log().Error("[GenStoryboardText] 获取故事白板失败",
			zap.Int64("boardId", req.GetBoardId()),
			zap.Error(err))
		return nil, err
	}
	story, err := models.GetStory(ctx, board.StoryID)
	if err != nil {
		log.Log().Error("[GenStoryboardText] 获取故事失败",
			zap.Int64("storyId", board.StoryID),
			zap.Error(err))
		return nil, err
	}
	// 检查故事状态
	if story.IsClose {
		log.Log().Warn("[GenStoryboardText] 故事已关闭，无法生成文本",
			zap.Uint("storyId", story.ID),
			zap.String("storyTitle", story.Title))
		return &api.GenStoryboardTextResponse{
			Code:    -1,
			Message: "story is closed",
		}, nil
	}

	// 获取故事板生成记录
	log.Log().Info("[GenStoryboardText] 开始获取故事板生成记录", zap.Int64("boardId", req.GetBoardId()))
	storyGen, err := models.GetStoryGensByStoryBoard(ctx, req.GetBoardId(), 1)
	if err != nil {
		log.Log().Error("[GenStoryboardText] 获取故事板生成记录失败",
			zap.Int64("boardId", req.GetBoardId()),
			zap.Error(err))
		return nil, err
	}

	// 检查是否有生成记录
	if len(storyGen) == 0 {
		log.Log().Warn("[GenStoryboardText] 故事板未渲染，无法生成文本",
			zap.Int64("boardId", req.GetBoardId()))
		return &api.GenStoryboardTextResponse{
			Code:    -1,
			Message: "storyboard is not rendering",
		}, nil
	}

	storyGenContent, err := json.Marshal(storyGen[0].Content)
	if err != nil {
		log.Log().Error("[GenStoryboardText] 序列化故事生成内容失败",
			zap.String("content", storyGen[0].Content),
			zap.Error(err))
		return nil, err
	}

	// 返回成功响应
	log.Log().Info("[GenStoryboardText] 故事板文本生成完成",
		zap.Int64("boardId", req.GetBoardId()),
		zap.Int("contentLength", len(storyGenContent)))

	return &api.GenStoryboardTextResponse{
		Code:    0,
		Message: "OK",
		Data:    nil,
	}, nil
}

func (s *StoryboardService) GetStoryBoardRender(ctx context.Context, req *api.GetStoryBoardRenderRequest) (*api.GetStoryBoardRenderResponse, error) {
	// 函数入口日志
	log.Log().Info("[GetStoryBoardRender] 开始获取故事板渲染结果",
		zap.Int64("boardId", req.GetBoardId()))

	// 获取故事板生成记录
	log.Log().Info("[GetStoryBoardRender] 开始获取故事板生成记录", zap.Int64("boardId", req.GetBoardId()))
	list, err := models.GetStoryGensByStoryBoard(ctx, req.GetBoardId(), 1)
	if err != nil {
		log.Log().Error("[GetStoryBoardRender] 获取故事板生成记录失败",
			zap.Int64("boardId", req.GetBoardId()),
			zap.Error(err))
		return nil, err
	}
	log.Log().Info("[GetStoryBoardRender] 成功获取故事板生成记录",
		zap.Int64("boardId", req.GetBoardId()),
		zap.Int("genCount", len(list)))

	// 检查是否有生成记录
	if len(list) == 0 {
		log.Log().Warn("[GetStoryBoardRender] 故事板未渲染，无法获取渲染结果",
			zap.Int64("boardId", req.GetBoardId()))
		return &api.GetStoryBoardRenderResponse{
			Code:    -1,
			Message: "board is not rendering",
		}, nil
	}

	// 解析渲染详情
	log.Log().Info("[GetStoryBoardRender] 开始解析渲染详情")
	item := new(api.RenderStoryboardDetail)
	err = json.Unmarshal([]byte(list[0].Content), &item)
	if err != nil {
		log.Log().Error("[GetStoryBoardRender] 解析渲染详情失败",
			zap.Int64("boardId", req.GetBoardId()),
			zap.String("content", list[0].Content),
			zap.Error(err))
		return nil, err
	}
	log.Log().Info("[GetStoryBoardRender] 成功解析渲染详情",
		zap.Int64("boardId", req.GetBoardId()),
		zap.Int64("storyId", item.StoryId),
		zap.Int32("timeCost", item.Timecost))

	// 返回成功响应
	log.Log().Info("[GetStoryBoardRender] 获取故事板渲染结果完成",
		zap.Int64("boardId", req.GetBoardId()))

	return &api.GetStoryBoardRenderResponse{
		Code:    0,
		Message: "OK",
		Data: &api.GetStoryBoardRenderResponse_Data{
			List: []*api.RenderStoryboardDetail{
				item,
			},
		},
	}, nil
}

func (s *StoryboardService) InitRenderStoryboard(ctx context.Context, req *api.ContinueRenderStoryRequest) (*api.ContinueRenderStoryResponse, error) {
	// 函数入口日志
	log.Log().Info("[InitRenderStoryboard] 开始初始化渲染故事",
		zap.Int64("storyId", req.GetStoryId()),
		zap.Int64("prevBoardId", req.GetPrevBoardId()),
		zap.String("renderType", req.GetRenderType().String()))

	// 获取故事信息
	story, err := models.GetStory(ctx, req.StoryId)
	if err != nil {
		log.Log().Error("[InitRenderStoryboard] 获取故事失败",
			zap.Int64("storyId", req.GetStoryId()),
			zap.Error(err))
		return nil, err
	}
	if story.RootBoardID > 0 {
		log.Log().Info("[InitRenderStoryboard] 故事有根故事板",
			zap.Int64("storyId", req.GetStoryId()),
			zap.Int("rootBoardId", story.RootBoardID))
		return &api.ContinueRenderStoryResponse{
			Code:    -1,
			Message: "story has root board",
		}, nil
	}
	// 获取角色信息
	roles := req.GetRoles()
	originRoles, err := models.GetStoryRole(ctx, int64(story.ID))
	if err != nil {
		log.Log().Error("[InitRenderStoryboard] 获取故事角色失败",
			zap.Int64("storyId", req.GetStoryId()),
			zap.Error(err))
		return nil, err
	}
	originRolesMap := make(map[int64]*models.StoryRole)
	rolesMap := make(map[int64]*api.StoryRole)
	finalRols := make(map[string]*models.StoryRole)

	for _, role := range originRoles {
		originRolesMap[int64(role.ID)] = role
	}
	for _, role := range roles {
		rolesMap[int64(role.RoleId)] = role
	}

	for _, role := range roles {
		log.Log().Info("[InitRenderStoryboard] 处理角色",
			zap.Int64("storyId", req.GetStoryId()),
			zap.Int64("roleId", role.GetRoleId()),
			zap.String("characterName", role.GetCharacterName()))

		if realRole, ok := originRolesMap[int64(role.RoleId)]; ok {
			finalRols[fmt.Sprintf("%d", role.RoleId)] = realRole
		}
	}

	var rolesPrompt = make([]coze.CozeRoleInfo, 0)
	for _, role := range finalRols {
		rolePrompt := coze.CozeRoleInfo{
			RoleID:          fmt.Sprintf("%d", role.ID),
			RoleName:        role.CharacterName,
			RoleImage:       role.CharacterAvatar,
			RoleDescription: role.CharacterDescription,
		}
		roleDetail := &api.CharacterDetail{}
		if role.CharacterDetail != "" {
			if err := json.Unmarshal([]byte(role.CharacterDetail), &roleDetail); err != nil {
				log.Log().Warn("InitRenderStoryboard: unmarshal character detail failed", zap.Error(err), zap.Int64("role_id", int64(role.ID)))
				rolePrompt.RoleDescription = role.CharacterDescription
			} else {
				rolePrompt.RoleDescription = roleDetail.Appearance + "," + roleDetail.DressPreference + "," + roleDetail.CognitionRange
			}
		}

		rolesPrompt = append(rolesPrompt, rolePrompt)
		log.Log().Info("[InitRenderStoryboard] 添加角色提示",
			zap.Int64("storyId", req.GetStoryId()),
			zap.String("roleId", rolePrompt.RoleID),
			zap.String("roleName", rolePrompt.RoleName))
	}

	storyGen := new(models.StoryGen)
	storyGen.UUID = uuid.New().String()
	var storyboardParams = coze.CozeInitStoryboardParams{
		Title:       story.Title,
		Description: req.GetDescription(),
		Background:  req.GetBackground(),
		Roles:       rolesPrompt,
		Style:       story.Style,
		SceneNum:    int(story.SenceMaxNumber),
	}

	storyGen.LLmPlatform = "coze"
	storyGen.Params = utils.ToJsonString(storyboardParams)
	storyGen.OriginID = req.GetStoryId()
	storyGen.StartTime = time.Now().Unix()
	storyGen.TaskType = api.RenderType_RENDER_TYPE_STORYBOARD
	storyGen.UUID = uuid.New().String()
	storyGen.TaskId = uuid.New().String()
	storyGen.GenStatus = api.StoryGenStatus_STORY_GEN_STATUS_INIT
	storyGen.UserId = req.GetUserId()
	storyGen.OriginID = req.GetStoryId()
	storyGen.UserId = req.GetUserId()

	_, err = models.CreateStoryGen(ctx, storyGen)
	if err != nil {
		log.Log().Error("[InitRenderStoryboard] 创建故事生成记录失败",
			zap.Int64("storyId", req.GetStoryId()),
			zap.String("uuid", storyGen.UUID),
			zap.Error(err))
		return nil, err
	}

	// 调用LLM生成故事板
	log.Log().Info("[InitRenderStoryboard] 开始调用LLM生成故事板",
		zap.Int64("storyId", req.GetStoryId()),
		zap.String("uuid", storyGen.UUID))

	result := new(StoryChapterV2)
	start := time.Now()
	ret, err := s.cozeClient.InitStoryboard(ctx, storyboardParams)
	if err != nil {
		log.Log().Error("[InitRenderStoryboard] LLM生成故事板失败",
			zap.Int64("storyId", req.GetStoryId()),
			zap.String("uuid", storyGen.UUID),
			zap.Error(err))
		return nil, err
	}

	genTime := time.Since(start)
	log.Log().Info("[InitRenderStoryboard] LLM生成故事板成功",
		zap.Int64("storyId", req.GetStoryId()),
		zap.String("uuid", storyGen.UUID),
		zap.Duration("genTime", genTime),
		zap.String("result", ret))

	// 解析生成结果
	log.Log().Info("[InitRenderStoryboard] 开始解析生成结果",
		zap.Int64("storyId", req.GetStoryId()),
		zap.String("uuid", storyGen.UUID))

	cleanResult := utils.CleanLLmJsonResult(ret)
	err = json.Unmarshal([]byte(cleanResult), &result)
	if err != nil {
		log.Log().Error("[InitRenderStoryboard] 解析生成结果失败",
			zap.Int64("storyId", req.GetStoryId()),
			zap.String("uuid", storyGen.UUID),
			zap.Error(err))
		return nil, err
	}
	log.Log().Info("[InitRenderStoryboard] 成功解析生成结果",
		zap.Int64("storyId", req.GetStoryId()),
		zap.String("uuid", storyGen.UUID),
		zap.String("chapterTitle", result.ChapterSummary.Title),
		zap.Int("characterCount", len(result.Characters)))

	// 构建渲染详情
	log.Log().Info("[InitRenderStoryboard] 开始构建渲染详情",
		zap.Int64("storyId", req.GetStoryId()),
		zap.String("uuid", storyGen.UUID))

	renderDetail := new(api.RenderStoryboardDetail)
	renderDetail.RenderType = req.RenderType
	renderDetail.Timecost = int32(time.Since(start).Seconds())
	renderDetail.Result = new(api.StoryChapter)

	// 转换故事章节
	storyChapter := &api.StoryChapter{
		ChapterSummary: &api.ChapterSummary{
			Title:      result.ChapterSummary.Title,
			Content:    result.ChapterSummary.Content,
			Characters: make([]*api.Character, 0),
		},
	}
	for _, character := range result.Characters {
		characterInfo := &api.Character{
			Id:          character.ID,
			Name:        character.Name,
			Description: character.Description,
		}
		if finalRols[characterInfo.Id] != nil {
			characterInfo.AvatarUrl = finalRols[characterInfo.Id].CharacterAvatar
		}
		storyChapter.ChapterSummary.Characters = append(storyChapter.ChapterSummary.Characters, characterInfo)
		log.Log().Info("[InitRenderStoryboard] 添加角色信息",
			zap.Int64("storyId", req.GetStoryId()),
			zap.String("uuid", storyGen.UUID),
			zap.String("characterId", character.ID),
			zap.String("characterName", character.Name))
	}

	renderDetail.Result = storyChapter
	log.Log().Info("[InitRenderStoryboard] 渲染详情构建完成",
		zap.Int64("storyId", req.GetStoryId()),
		zap.String("uuid", storyGen.UUID),
		zap.Int32("timeCost", renderDetail.Timecost),
		zap.Int("characterCount", len(storyChapter.ChapterSummary.Characters)))

	renderDetailData, _ := json.Marshal(renderDetail)
	storyGen.Content = string(renderDetailData)
	storyGen.FinishTime = time.Now().Unix()
	storyGen.GenStatus = api.StoryGenStatus_STORY_GEN_STATUS_FINISHED

	err = models.UpdateStoryGen(ctx, storyGen)
	if err != nil {
		log.Log().Error("[InitRenderStoryboard] 更新故事生成记录失败",
			zap.Int64("storyId", req.GetStoryId()),
			zap.String("uuid", storyGen.UUID),
			zap.Error(err))
	}

	// 返回成功响应
	log.Log().Info("[InitRenderStoryboard] 初始化渲染故事完成",
		zap.Int64("storyId", req.GetStoryId()),
		zap.String("uuid", storyGen.UUID),
		zap.Duration("totalTime", time.Since(start)),
		zap.Int32("timeCost", renderDetail.Timecost))

	return &api.ContinueRenderStoryResponse{
		Code:    0,
		Message: "OK",
		Data:    renderDetail,
	}, nil
}

func (s *StoryboardService) ContinueRenderStory(ctx context.Context, req *api.ContinueRenderStoryRequest) (*api.ContinueRenderStoryResponse, error) {
	// 函数入口日志
	log.Log().Info("[ContinueRenderStory] 开始继续渲染故事",
		zap.Int64("storyId", req.GetStoryId()),
		zap.Int64("prevBoardId", req.GetPrevBoardId()),
		zap.String("renderType", req.GetRenderType().String()))
	story, err := models.GetStory(ctx, req.GetStoryId())
	if err != nil {
		log.Log().Error("[ContinueRenderStory] 获取故事失败",
			zap.Int64("storyId", req.GetStoryId()),
			zap.Error(err))
		return nil, err
	}
	log.Log().Info("[ContinueRenderStory] 继续渲染故事请求",
		zap.Any("req", req))

	if req.PrevBoardId <= 0 {
		log.Log().Info("[ContinueRenderStory] 检测到初始化渲染请求，调用InitRenderStoryboard",
			zap.Int64("storyId", req.GetStoryId()),
			zap.Int64("prevBoardId", req.GetPrevBoardId()))
		return s.InitRenderStoryboard(ctx, req)
	}

	// 获取前一个故事板
	log.Log().Info("[ContinueRenderStory] 开始获取前一个故事板",
		zap.Int64("prevBoardId", req.GetPrevBoardId()))
	board, err := models.GetStoryboard(ctx, req.PrevBoardId)
	if err != nil {
		log.Log().Error("[ContinueRenderStory] 获取前一个故事板失败",
			zap.Int64("prevBoardId", req.GetPrevBoardId()),
			zap.Error(err))
		return nil, err
	}
	log.Log().Info("[ContinueRenderStory] 成功获取前一个故事板",
		zap.Int64("prevBoardId", req.GetPrevBoardId()),
		zap.String("boardTitle", board.Title),
		zap.Int64("prevId", board.PrevId))

	prevBoards := make([]*models.StoryBoard, 0)
	prevBoards = append(prevBoards, board)

	var boardIdtemp int64 = board.PrevId
	boardCount := 1
	for boardIdtemp > 0 {
		prevBoard, err := models.GetStoryboard(ctx, boardIdtemp)
		if err != nil {
			log.Log().Error("[ContinueRenderStory] 获取前一个故事板失败",
				zap.Int64("storyId", req.GetStoryId()),
				zap.Int64("boardId", boardIdtemp),
				zap.Error(err))
			return nil, err
		}
		boardIdtemp = prevBoard.PrevId
		prevBoards = append(prevBoards, prevBoard)
		boardCount++

		if len(prevBoards) > 5 {
			log.Log().Info("[ContinueRenderStory] 达到最大前一个故事板数量限制",
				zap.Int64("storyId", req.GetStoryId()),
				zap.Int("boardCount", len(prevBoards)))
			break
		}
	}

	roles := req.GetRoles()
	originRoles, err := models.GetStoryRole(ctx, int64(story.ID))
	if err != nil {
		log.Log().Error("[ContinueRenderStory] 获取故事角色失败",
			zap.Int64("storyId", req.GetStoryId()),
			zap.Error(err))
		return nil, err
	}

	// 构建角色映射
	originRolesMap := make(map[string]*models.StoryRole)
	rolesMap := make(map[string]*api.StoryRole)
	finalRols := make(map[string]*models.StoryRole)

	for _, role := range originRoles {
		originRolesMap[fmt.Sprintf("%d", role.ID)] = role
	}
	for _, role := range roles {
		rolesMap[fmt.Sprintf("%d", role.RoleId)] = role
	}

	for _, role := range roles {
		if realRole, ok := originRolesMap[fmt.Sprintf("%d", role.RoleId)]; ok {
			finalRols[fmt.Sprintf("%d", role.RoleId)] = realRole
		} else {
			log.Log().Warn("[ContinueRenderStory] 角色不存在",
				zap.Int64("storyId", req.GetStoryId()),
				zap.Int64("roleId", role.GetRoleId()),
				zap.String("characterName", role.GetCharacterName()))
		}
	}
	log.Log().Info("[ContinueRenderStory] 角色映射构建完成",
		zap.Int64("storyId", req.GetStoryId()),
		zap.Int("finalRoleCount", len(finalRols)))
	var rolesPrompt = make([]coze.CozeRoleInfo, 0)
	for _, role := range finalRols {
		rolePrompt := coze.CozeRoleInfo{
			RoleID:          fmt.Sprintf("%d", role.ID),
			RoleName:        role.CharacterName,
			RoleImage:       role.CharacterAvatar,
			RoleDescription: role.CharacterDescription,
		}
		if role.CharacterDetail != "" {
			roleDetail := &api.CharacterDetail{}
			if err := json.Unmarshal([]byte(role.CharacterDetail), &rolePrompt); err != nil {
				log.Log().Warn("ContinueRenderStory: unmarshal character detail failed", zap.Error(err), zap.Int64("role_id", int64(role.ID)))
				rolePrompt.RoleDescription = role.CharacterDescription
			} else {
				rolePrompt.RoleDescription = roleDetail.Appearance + "," + roleDetail.DressPreference + "," + roleDetail.CognitionRange
			}
		}
		rolesPrompt = append(rolesPrompt, rolePrompt)
	}
	log.Log().Info("[ContinueRenderStory] 角色提示信息构建完成",
		zap.Int64("storyId", req.GetStoryId()),
		zap.Int("rolePromptCount", len(rolesPrompt)))

	genParams := new(models.StoryBoardParams)
	genParams.StoryContent = story.Origin
	err = json.Unmarshal([]byte(board.Params), genParams)
	if err != nil {
		log.Log().Error("[ContinueRenderStory] 解析故事板生成参数失败",
			zap.Int64("storyId", req.GetStoryId()),
			zap.Int64("prevBoardId", req.GetPrevBoardId()),
			zap.Error(err))
		return nil, err
	}

	storyGen := new(models.StoryGen)
	storyGen.UUID = uuid.New().String()

	var storyboardParams = coze.CozeStoryboardContinueParams{
		Title:       req.GetTitle(),
		Description: req.GetDescription(),
		Background:  req.GetBackground(),
		Roles:       rolesPrompt,
		Style:       story.Style,
		SceneNum:    int(story.SenceMaxNumber),
	}

	log.Log().Info("[ContinueRenderStory] 故事板继续参数",
		zap.Int64("storyId", req.GetStoryId()),
		zap.String("uuid", storyGen.UUID),
		zap.String("title", storyboardParams.Title),
		zap.String("description", storyboardParams.Description),
		zap.String("background", storyboardParams.Background),
		zap.Int("roleCount", len(storyboardParams.Roles)))

	// 构建前一个故事内容
	log.Log().Info("[ContinueRenderStory] 开始构建前一个故事内容",
		zap.Int64("storyId", req.GetStoryId()),
		zap.Int("prevBoardCount", len(prevBoards)))

	hasPrevContent := false
	story_prev_content := make([]*StoryChapterV2, 0)
	for idx := len(prevBoards) - 1; idx >= 0; idx-- {
		prevBoard := prevBoards[idx]
		log.Log().Info("[ContinueRenderStory] 处理前一个故事板",
			zap.Int64("storyId", req.GetStoryId()),
			zap.Int("boardIndex", idx),
			zap.Uint("boardId", prevBoard.ID),
			zap.String("boardTitle", prevBoard.Title))

		content := new(StoryChapterV2)
		content.ChapterSummary.Title = prevBoard.Title
		content.ChapterSummary.Content = prevBoard.Description
		content.ChapterSummary.Characters = make([]Character, 0)

		storyRoles, err := models.GetStoryBoardRoles(ctx, int64(prevBoard.ID))
		if err != nil {
			log.Log().Error("[ContinueRenderStory] 获取故事板角色失败",
				zap.Int64("storyId", req.GetStoryId()),
				zap.Uint("boardId", prevBoard.ID),
				zap.Error(err))
			return nil, err
		}

		hasPrevContent = true
		for _, role := range storyRoles {
			roleInfo := new(Character)
			roleInfo.ID = fmt.Sprintf("%d", role.RoleId)
			roleInfo.Name = role.Name
			roleInfo.Description = role.Desc
			content.ChapterSummary.Characters = append(content.ChapterSummary.Characters, *roleInfo)
			log.Log().Info("[ContinueRenderStory] 添加前一个故事板角色",
				zap.Int64("storyId", req.GetStoryId()),
				zap.Uint("boardId", prevBoard.ID),
				zap.String("roleId", roleInfo.ID),
				zap.String("roleName", roleInfo.Name))
		}
		story_prev_content = append(story_prev_content, content)
	}

	if hasPrevContent {
		story_prev_content_json, _ := json.Marshal(story_prev_content)
		storyboardParams.StoryPrevContent = string(story_prev_content_json)
		log.Log().Info("[ContinueRenderStory] 设置前一个故事内容",
			zap.Int64("storyId", req.GetStoryId()),
			zap.Int("contentLength", len(story_prev_content_json)))
	} else {
		storyboardParams.StoryPrevContent = "{{暂无上一章节}}"
		log.Log().Info("[ContinueRenderStory] 无前一个故事内容",
			zap.Int64("storyId", req.GetStoryId()))
	}
	// 设置故事生成记录参数
	storyGen.LLmPlatform = "coze"
	storyGen.Params = utils.ToJsonString(storyboardParams)
	storyGen.OriginID = req.GetStoryId()
	storyGen.StartTime = time.Now().Unix()
	storyGen.TaskType = api.RenderType_RENDER_TYPE_STORYBOARD
	storyGen.UUID = uuid.New().String()
	storyGen.TaskId = storyGen.UUID
	storyGen.GenStatus = api.StoryGenStatus_STORY_GEN_STATUS_INIT
	storyGen.UserId = req.GetUserId()

	_, err = models.CreateStoryGen(ctx, storyGen)
	if err != nil {
		log.Log().Error("[ContinueRenderStory] 创建故事生成记录失败",
			zap.Int64("storyId", req.GetStoryId()),
			zap.String("uuid", storyGen.UUID),
			zap.Error(err))
		return nil, err
	}
	log.Log().Info("[ContinueRenderStory] 成功创建故事生成记录",
		zap.Int64("storyId", req.GetStoryId()),
		zap.String("uuid", storyGen.UUID))

	// 调用LLM继续生成故事板
	log.Log().Info("[ContinueRenderStory] 开始调用LLM继续生成故事板",
		zap.Int64("storyId", req.GetStoryId()),
		zap.String("uuid", storyGen.UUID))

	result := new(StoryChapterV2)
	start := time.Now()
	ret, err := s.cozeClient.StoryboardContinue(ctx, storyboardParams)
	if err != nil {
		log.Log().Error("[ContinueRenderStory] LLM继续生成故事板失败",
			zap.Int64("storyId", req.GetStoryId()),
			zap.String("uuid", storyGen.UUID),
			zap.Error(err))
		return nil, err
	}

	genTime := time.Since(start)
	log.Log().Info("[ContinueRenderStory] LLM继续生成故事板成功",
		zap.Int64("storyId", req.GetStoryId()),
		zap.String("uuid", storyGen.UUID),
		zap.Duration("genTime", genTime),
		zap.String("result", ret))

	// 解析生成结果
	log.Log().Info("[ContinueRenderStory] 开始解析生成结果",
		zap.Int64("storyId", req.GetStoryId()),
		zap.String("uuid", storyGen.UUID))

	cleanResult := utils.CleanLLmJsonResult(ret)
	err = json.Unmarshal([]byte(cleanResult), &result)
	if err != nil {
		log.Log().Error("[ContinueRenderStory] 解析生成结果失败",
			zap.Int64("storyId", req.GetStoryId()),
			zap.String("uuid", storyGen.UUID),
			zap.Error(err))
		return nil, err
	}
	log.Log().Info("[ContinueRenderStory] 成功解析生成结果",
		zap.Int64("storyId", req.GetStoryId()),
		zap.String("uuid", storyGen.UUID),
		zap.String("chapterTitle", result.ChapterSummary.Title),
		zap.Int("characterCount", len(result.Characters)))

	renderDetail := new(api.RenderStoryboardDetail)
	renderDetail.RenderType = req.RenderType
	renderDetail.Timecost = int32(time.Since(start).Seconds())
	renderDetail.Result = new(api.StoryChapter)

	// 转换故事章节
	storyChapter := &api.StoryChapter{
		ChapterSummary: &api.ChapterSummary{
			Title:      result.ChapterSummary.Title,
			Content:    result.ChapterSummary.Content,
			Characters: make([]*api.Character, 0),
		},
	}
	for _, character := range result.Characters {
		characterInfo := &api.Character{
			Id:          character.ID,
			Name:        character.Name,
			Description: character.Description,
		}
		if finalRols[characterInfo.Id] != nil {
			characterInfo.AvatarUrl = finalRols[characterInfo.Id].CharacterAvatar
		}
		storyChapter.ChapterSummary.Characters = append(storyChapter.ChapterSummary.Characters, characterInfo)
		log.Log().Info("[ContinueRenderStory] 添加角色信息",
			zap.Int64("storyId", req.GetStoryId()),
			zap.String("uuid", storyGen.UUID),
			zap.String("characterId", character.ID),
			zap.String("characterName", character.Name))
	}

	renderDetail.Result = storyChapter

	renderDetailData, _ := json.Marshal(renderDetail)
	log.Log().Info("[ContinueRenderStory] 渲染详情数据",
		zap.Int64("storyId", req.GetStoryId()),
		zap.String("uuid", storyGen.UUID),
		zap.String("renderDetailData", string(renderDetailData)))

	storyGen.Content = string(renderDetailData)
	storyGen.FinishTime = time.Now().Unix()
	storyGen.GenStatus = api.StoryGenStatus_STORY_GEN_STATUS_FINISHED

	err = models.UpdateStoryGen(ctx, storyGen)
	if err != nil {
		log.Log().Error("[ContinueRenderStory] 更新故事生成记录失败",
			zap.Int64("storyId", req.GetStoryId()),
			zap.String("uuid", storyGen.UUID),
			zap.Error(err))
	} else {
		log.Log().Info("[ContinueRenderStory] 成功更新故事生成记录",
			zap.Int64("storyId", req.GetStoryId()),
			zap.String("uuid", storyGen.UUID))
	}

	// 返回成功响应
	log.Log().Info("[ContinueRenderStory] 继续渲染故事完成",
		zap.Int64("storyId", req.GetStoryId()),
		zap.String("uuid", storyGen.UUID),
		zap.Duration("totalTime", time.Since(start)),
		zap.Int32("timeCost", renderDetail.Timecost))

	return &api.ContinueRenderStoryResponse{
		Code:    0,
		Message: "OK",
		Data:    renderDetail,
	}, nil
}

func (s *StoryboardService) RenderStoryRoles(ctx context.Context, req *api.RenderStoryRolesRequest) (*api.RenderStoryRolesResponse, error) {
	// 函数入口日志
	log.Log().Info("[RenderStoryRoles] 开始渲染故事角色",
		zap.Int64("storyId", req.GetStoryId()))

	// 获取故事信息
	log.Log().Info("[RenderStoryRoles] 开始获取故事信息", zap.Int64("storyId", req.GetStoryId()))
	story, err := models.GetStory(ctx, req.GetStoryId())
	if err != nil {
		log.Log().Error("[RenderStoryRoles] 获取故事失败",
			zap.Int64("storyId", req.GetStoryId()),
			zap.Error(err))
		return nil, err
	}
	if story.IsClose {
		log.Log().Error("[RenderStoryRoles] 故事已关闭",
			zap.Int64("storyId", req.GetStoryId()),
			zap.Int("status", int(story.Status)))
		return &api.RenderStoryRolesResponse{
			Code:    -1,
			Message: "story is closed",
		}, nil
	}
	log.Log().Info("[RenderStoryRoles] 成功获取故事信息",
		zap.Int64("storyId", req.GetStoryId()),
		zap.String("storyTitle", story.Title),
		zap.Int("status", int(story.Status)))

	// 获取角色信息
	log.Log().Info("[RenderStoryRoles] 开始获取角色信息", zap.Int64("storyId", req.GetStoryId()))
	roles, err := models.GetStoryRole(ctx, int64(story.ID))
	if err != nil {
		log.Log().Error("[RenderStoryRoles] 获取故事角色失败",
			zap.Int64("storyId", req.GetStoryId()),
			zap.Error(err))
		return nil, err
	}
	log.Log().Info("[RenderStoryRoles] 成功获取角色信息",
		zap.Int64("storyId", req.GetStoryId()),
		zap.Int("roleCount", len(roles)),
		zap.Any("roles", roles))

	// 返回成功响应
	log.Log().Info("[RenderStoryRoles] 渲染故事角色完成",
		zap.Int64("storyId", req.GetStoryId()),
		zap.Int("roleCount", len(roles)))

	return &api.RenderStoryRolesResponse{
		Code:    0,
		Message: "OK",
	}, nil
}

func (s *StoryboardService) GetStoryRoles(ctx context.Context, req *api.GetStoryRolesRequest) (*api.GetStoryRolesResponse, error) {
	// 函数入口日志
	log.Log().Info("[GetStoryRoles] 开始获取故事角色列表",
		zap.Int64("storyId", req.GetStoryId()))

	// 获取故事
	log.Log().Info("[GetStoryRoles] 开始获取故事", zap.Int64("storyId", req.GetStoryId()))
	story, err := models.GetStory(ctx, req.GetStoryId())
	if err != nil {
		log.Log().Error("[GetStoryRoles] 获取故事失败",
			zap.Int64("storyId", req.GetStoryId()),
			zap.Error(err))
		return nil, err
	}
	log.Log().Info("[GetStoryRoles] 成功获取故事",
		zap.Uint("storyId", story.ID),
		zap.String("storyTitle", story.Title),
		zap.Int("status", int(story.Status)))

	// 检查故事状态
	if story.IsClose {
		log.Log().Warn("[GetStoryRoles] 故事已关闭，无法获取角色列表",
			zap.Uint("storyId", story.ID),
			zap.String("storyTitle", story.Title))
		return &api.GetStoryRolesResponse{
			Code:    -1,
			Message: "story is closed",
		}, nil
	}

	// 获取故事角色列表
	log.Log().Info("[GetStoryRoles] 开始获取故事角色列表", zap.Uint("storyId", story.ID))
	roles, err := models.GetStoryRole(ctx, int64(story.ID))
	if err != nil {
		log.Log().Error("[GetStoryRoles] 获取故事角色列表失败",
			zap.Uint("storyId", story.ID),
			zap.Error(err))
		return nil, err
	}
	log.Log().Info("[GetStoryRoles] 成功获取故事角色列表",
		zap.Uint("storyId", story.ID),
		zap.Int("roleCount", len(roles)))

	// 收集创建者ID
	log.Log().Info("[GetStoryRoles] 开始收集创建者ID")
	creatorIds := make([]int64, 0)
	for _, role := range roles {
		creatorIds = append(creatorIds, role.CreatorID)
	}
	log.Log().Info("[GetStoryRoles] 创建者ID收集完成",
		zap.Uint("storyId", story.ID),
		zap.Int("creatorCount", len(creatorIds)),
		zap.Any("creatorIds", creatorIds))

	// 获取创建者信息
	log.Log().Info("[GetStoryRoles] 开始获取创建者信息")
	creatorsMap, err := models.GetUsersByIdsMap(ctx, creatorIds)
	if err != nil {
		log.Log().Error("[GetStoryRoles] 获取创建者信息失败",
			zap.Uint("storyId", story.ID),
			zap.Any("creatorIds", creatorIds),
			zap.Error(err))
		return nil, err
	}
	log.Log().Info("[GetStoryRoles] 成功获取创建者信息",
		zap.Uint("storyId", story.ID),
		zap.Int("creatorCount", len(creatorsMap)))

	// 构建创建者列表
	log.Log().Info("[GetStoryRoles] 开始构建创建者列表")
	finnalCreators := make([]*api.UserInfo, 0)
	for _, creator := range creatorsMap {
		finnalCreators = append(finnalCreators, &api.UserInfo{
			UserId: int64(creator.ID),
			Name:   creator.Name,
			Avatar: creator.Avatar,
		})
	}
	log.Log().Info("[GetStoryRoles] 创建者列表构建完成",
		zap.Uint("storyId", story.ID),
		zap.Int("creatorCount", len(finnalCreators)))

	// 构建API角色列表
	log.Log().Info("[GetStoryRoles] 开始构建API角色列表",
		zap.Uint("storyId", story.ID),
		zap.Int("roleCount", len(roles)))
	apiRoles := make([]*api.StoryRole, 0)

	for i, role := range roles {
		log.Log().Info("[GetStoryRoles] 开始处理角色",
			zap.Uint("storyId", story.ID),
			zap.Int("roleIndex", i),
			zap.Uint("roleId", role.ID),
			zap.String("characterName", role.CharacterName))

		apiRole := new(api.StoryRole)
		apiRole.RoleId = int64(role.ID)
		apiRole.CharacterName = role.CharacterName
		apiRole.CharacterAvatar = role.CharacterAvatar
		apiRole.CharacterId = role.CharacterID
		apiRole.CharacterType = role.CharacterType
		apiRole.CharacterPrompt = role.CharacterPrompt
		apiRole.CharacterRefImages = strings.Split(role.CharacterRefImages, ",")
		apiRole.CharacterDescription = role.CharacterDescription

		// 获取角色当前用户状态
		log.Log().Info("[GetStoryRoles] 开始获取角色当前用户状态",
			zap.Uint("storyId", story.ID),
			zap.Int("roleIndex", i),
			zap.Uint("roleId", role.ID))
		cu, err := s.helper.GetStoryRoleCurrentUserStatus(ctx, int64(role.ID))
		if err != nil {
			log.Log().Error("[GetStoryRoles] 获取角色当前用户状态失败",
				zap.Uint("storyId", story.ID),
				zap.Int("roleIndex", i),
				zap.Uint("roleId", role.ID),
				zap.Error(err))
		} else {
			log.Log().Info("[GetStoryRoles] 成功获取角色当前用户状态",
				zap.Uint("storyId", story.ID),
				zap.Int("roleIndex", i),
				zap.Uint("roleId", role.ID),
				zap.Any("currentUserStatus", cu))
		}
		apiRole.CurrentUserStatus = cu

		// 设置创建者信息
		apiRole.Creator = &api.UserInfo{
			UserId: int64(role.CreatorID),
			Name:   creatorsMap[int(role.CreatorID)].Name,
			Avatar: creatorsMap[int(role.CreatorID)].Avatar,
		}

		// 设置其他字段
		apiRole.LikeCount = role.LikeCount
		apiRole.FollowCount = role.FollowCount
		apiRole.StoryboardNum = role.StoryboardNum
		apiRole.Ctime = int64(role.CreateAt.Unix())
		apiRole.Mtime = int64(role.UpdateAt.Unix())

		// 解析角色详情
		err = json.Unmarshal([]byte(role.CharacterDetail), &apiRole.CharacterDetail)
		if err != nil {
			log.Log().Error("[GetStoryRoles] 解析角色详情失败",
				zap.Uint("storyId", story.ID),
				zap.Int("roleIndex", i),
				zap.Uint("roleId", role.ID),
				zap.String("characterDetail", role.CharacterDetail),
				zap.Error(err))
		}

		apiRoles = append(apiRoles, apiRole)
		log.Log().Info("[GetStoryRoles] 角色处理完成",
			zap.Uint("storyId", story.ID),
			zap.Int("roleIndex", i),
			zap.Uint("roleId", role.ID),
			zap.String("characterName", role.CharacterName))
	}

	log.Log().Info("[GetStoryRoles] API角色列表构建完成",
		zap.Uint("storyId", story.ID),
		zap.Int("roleCount", len(apiRoles)))

	// 返回成功响应
	log.Log().Info("[GetStoryRoles] 获取故事角色列表完成",
		zap.Int64("storyId", req.GetStoryId()),
		zap.Int("roleCount", len(apiRoles)),
		zap.Int("creatorCount", len(finnalCreators)))

	return &api.GetStoryRolesResponse{
		Code:    0,
		Message: "OK",
		Data: &api.GetStoryRolesResponse_Data{
			List:    apiRoles,
			Creator: finnalCreators,
		},
	}, nil
}

func (s *StoryboardService) GetStoryBoardRoles(ctx context.Context, req *api.GetStoryBoardRolesRequest) (*api.GetStoryBoardRolesResponse, error) {
	// 函数入口日志
	log.Log().Info("[GetStoryBoardRoles] 开始获取故事板角色列表",
		zap.Int64("boardId", req.GetBoardId()))

	// 获取故事白板
	log.Log().Info("[GetStoryBoardRoles] 开始获取故事白板", zap.Int64("boardId", req.GetBoardId()))
	board, err := models.GetStoryboard(ctx, req.GetBoardId())
	if err != nil {
		log.Log().Error("[GetStoryBoardRoles] 获取故事白板失败",
			zap.Int64("boardId", req.GetBoardId()),
			zap.Error(err))
		return nil, err
	}
	log.Log().Info("[GetStoryBoardRoles] 成功获取故事白板",
		zap.Uint("boardId", board.ID),
		zap.Int64("storyId", board.StoryID))

	// 获取故事
	log.Log().Info("[GetStoryBoardRoles] 开始获取故事", zap.Int64("storyId", board.StoryID))
	story, err := models.GetStory(ctx, board.StoryID)
	if err != nil {
		log.Log().Error("[GetStoryBoardRoles] 获取故事失败",
			zap.Int64("storyId", board.StoryID),
			zap.Error(err))
		return nil, err
	}
	log.Log().Info("[GetStoryBoardRoles] 成功获取故事",
		zap.Uint("storyId", story.ID),
		zap.String("storyTitle", story.Title),
		zap.Int("status", int(story.Status)))

	// 检查故事状态
	if story.IsClose {
		log.Log().Warn("[GetStoryBoardRoles] 故事已关闭，无法获取角色列表",
			zap.Uint("storyId", story.ID),
			zap.String("storyTitle", story.Title))
		return &api.GetStoryBoardRolesResponse{
			Code:    -1,
			Message: "story is closed",
		}, nil
	}

	// 获取故事角色列表
	log.Log().Info("[GetStoryBoardRoles] 开始获取故事角色列表", zap.Uint("storyId", story.ID))
	roles, err := models.GetStoryRole(ctx, int64(story.ID))
	if err != nil {
		log.Log().Error("[GetStoryBoardRoles] 获取故事角色列表失败",
			zap.Uint("storyId", story.ID),
			zap.Error(err))
		return nil, err
	}
	log.Log().Info("[GetStoryBoardRoles] 成功获取故事角色列表",
		zap.Uint("storyId", story.ID),
		zap.Int("roleCount", len(roles)))

	// 收集创建者ID
	log.Log().Info("[GetStoryBoardRoles] 开始收集创建者ID")
	creatorIds := make([]int64, 0)
	for _, role := range roles {
		creatorIds = append(creatorIds, role.CreatorID)
	}
	log.Log().Info("[GetStoryBoardRoles] 创建者ID收集完成",
		zap.Uint("storyId", story.ID),
		zap.Int("creatorCount", len(creatorIds)),
		zap.Any("creatorIds", creatorIds))

	// 获取创建者信息
	log.Log().Info("[GetStoryBoardRoles] 开始获取创建者信息")
	creatorsMap, err := models.GetUsersByIdsMap(ctx, creatorIds)
	if err != nil {
		log.Log().Error("[GetStoryBoardRoles] 获取创建者信息失败",
			zap.Uint("storyId", story.ID),
			zap.Any("creatorIds", creatorIds),
			zap.Error(err))
		return nil, err
	}
	log.Log().Info("[GetStoryBoardRoles] 成功获取创建者信息",
		zap.Uint("storyId", story.ID),
		zap.Int("creatorCount", len(creatorsMap)))

	// 构建API角色列表
	log.Log().Info("[GetStoryBoardRoles] 开始构建API角色列表",
		zap.Uint("storyId", story.ID),
		zap.Int("roleCount", len(roles)))
	apiRoles := make([]*api.StoryRole, 0)

	for i, role := range roles {
		log.Log().Info("[GetStoryBoardRoles] 开始处理角色",
			zap.Uint("storyId", story.ID),
			zap.Int("roleIndex", i),
			zap.Uint("roleId", role.ID),
			zap.String("characterName", role.CharacterName))

		apiRole := new(api.StoryRole)
		apiRole.RoleId = int64(role.ID)
		apiRole.CharacterName = role.CharacterName
		apiRole.CharacterAvatar = role.CharacterAvatar
		apiRole.CharacterId = role.CharacterID
		apiRole.CharacterType = role.CharacterType
		apiRole.CharacterPrompt = role.CharacterPrompt
		apiRole.CharacterRefImages = strings.Split(role.CharacterRefImages, ",")
		apiRole.CharacterDescription = role.CharacterDescription

		// 获取角色当前用户状态
		log.Log().Info("[GetStoryBoardRoles] 开始获取角色当前用户状态",
			zap.Uint("storyId", story.ID),
			zap.Int("roleIndex", i),
			zap.Uint("roleId", role.ID))
		cu, err := s.helper.GetStoryRoleCurrentUserStatus(ctx, int64(role.ID))
		if err != nil {
			log.Log().Error("[GetStoryBoardRoles] 获取角色当前用户状态失败",
				zap.Uint("storyId", story.ID),
				zap.Int("roleIndex", i),
				zap.Uint("roleId", role.ID),
				zap.Error(err))
		} else {
			log.Log().Info("[GetStoryBoardRoles] 成功获取角色当前用户状态",
				zap.Uint("storyId", story.ID),
				zap.Int("roleIndex", i),
				zap.Uint("roleId", role.ID),
				zap.Any("currentUserStatus", cu))
		}
		apiRole.CurrentUserStatus = cu

		// 设置其他字段
		apiRole.LikeCount = role.LikeCount
		apiRole.FollowCount = role.FollowCount
		apiRole.StoryboardNum = role.StoryboardNum

		// 设置创建者信息
		apiRole.Creator = &api.UserInfo{
			UserId: int64(role.CreatorID),
			Name:   creatorsMap[int(role.CreatorID)].Name,
			Avatar: creatorsMap[int(role.CreatorID)].Avatar,
		}

		apiRole.Ctime = int64(role.CreateAt.Unix())
		apiRole.Mtime = int64(role.UpdateAt.Unix())
		apiRoles = append(apiRoles, apiRole)

		log.Log().Info("[GetStoryBoardRoles] 角色处理完成",
			zap.Uint("storyId", story.ID),
			zap.Int("roleIndex", i),
			zap.Uint("roleId", role.ID),
			zap.String("characterName", role.CharacterName))
	}

	// 构建创建者列表
	log.Log().Info("[GetStoryBoardRoles] 开始构建创建者列表")
	finnalCreators := make([]*api.UserInfo, 0)
	for _, creator := range creatorsMap {
		finnalCreators = append(finnalCreators, &api.UserInfo{
			UserId: int64(creator.ID),
			Name:   creator.Name,
			Avatar: creator.Avatar,
		})
	}
	log.Log().Info("[GetStoryBoardRoles] 创建者列表构建完成",
		zap.Uint("storyId", story.ID),
		zap.Int("creatorCount", len(finnalCreators)))

	// 返回成功响应
	log.Log().Info("[GetStoryBoardRoles] 获取故事板角色列表完成",
		zap.Int64("boardId", req.GetBoardId()),
		zap.Int("roleCount", len(apiRoles)),
		zap.Int("creatorCount", len(finnalCreators)))

	return &api.GetStoryBoardRolesResponse{
		Code:    0,
		Message: "OK",
		Data: &api.GetStoryBoardRolesResponse_Data{
			List:    apiRoles,
			Creator: finnalCreators,
		},
	}, nil
}

func (s *StoryboardService) UnLikeStoryboard(ctx context.Context, req *api.UnLikeStoryboardRequest) (*api.UnLikeStoryboardResponse, error) {
	// 函数入口日志
	log.Log().Info("[UnLikeStoryboard] 开始取消点赞故事板",
		zap.Int64("boardId", req.GetBoardId()),
		zap.Int64("userId", req.GetUserId()))

	removed, err := models.UnLikeStoryboard(ctx, req.GetUserId(), req.GetBoardId())
	if err != nil {
		log.Log().Error("[UnLikeStoryboard] 取消点赞故事板失败",
			zap.Int64("boardId", req.GetBoardId()),
			zap.Int64("userId", req.GetUserId()),
			zap.Error(err))
		return nil, err
	}

	if !removed {
		log.Log().Info("[UnLikeStoryboard] 用户未点赞该故事板，无需取消",
			zap.Int64("boardId", req.GetBoardId()),
			zap.Int64("userId", req.GetUserId()))
		return &api.UnLikeStoryboardResponse{
			Code:    int32(api.ResponseCode_OK),
			Message: "not liked",
		}, nil
	}

	return &api.UnLikeStoryboardResponse{
		Code:    int32(api.ResponseCode_OK),
		Message: api.ResponseCode_OK.String(),
	}, nil
}

func (s *StoryboardService) GetStoryBoardGenerate(ctx context.Context, req *api.GetStoryBoardGenerateRequest) (*api.GetStoryBoardGenerateResponse, error) {
	// 函数入口日志
	log.Log().Info("[GetStoryBoardGenerate] 开始获取故事板生成状态",
		zap.Int64("boardId", req.GetBoardId()))

	// 获取场景描述
	log.Log().Info("[GetStoryBoardGenerate] 开始获取场景列表", zap.Int64("boardId", req.GetBoardId()))
	scenes, err := models.GetStoryBoardScenesByBoard(ctx, req.GetBoardId())
	if err != nil {
		log.Log().Error("[GetStoryBoardGenerate] 获取故事板场景失败",
			zap.Int64("boardId", req.GetBoardId()),
			zap.Error(err))
		return nil, err
	}
	if len(scenes) == 0 {
		log.Log().Error("[GetStoryBoardGenerate] 场景不存在",
			zap.Int64("boardId", req.GetBoardId()))
		return &api.GetStoryBoardGenerateResponse{
			Code:    -1,
			Message: "scene not found",
		}, nil
	}
	log.Log().Info("[GetStoryBoardGenerate] 成功获取场景列表",
		zap.Int64("boardId", req.GetBoardId()),
		zap.Int("sceneCount", len(scenes)))

	// 统计场景状态
	log.Log().Info("[GetStoryBoardGenerate] 开始统计场景状态",
		zap.Int64("boardId", req.GetBoardId()),
		zap.Int("sceneCount", len(scenes)))

	total := len(scenes)
	generating := 0
	apiScenes := make([]*api.StoryBoardSence, 0)

	for i, scene := range scenes {
		log.Log().Info("[GetStoryBoardGenerate] 检查场景状态",
			zap.Int64("boardId", req.GetBoardId()),
			zap.Int("sceneIndex", i),
			zap.Uint("sceneId", scene.ID),
			zap.Int("status", scene.Status))

		if scene.Status == 1 {
			log.Log().Info("[GetStoryBoardGenerate] 场景正在生成中",
				zap.Int64("boardId", req.GetBoardId()),
				zap.Int("sceneIndex", i),
				zap.Uint("sceneId", scene.ID))
		}
		if scene.Status == 0 {
			generating++
			log.Log().Info("[GetStoryBoardGenerate] 场景未就绪",
				zap.Int64("boardId", req.GetBoardId()),
				zap.Int("sceneIndex", i),
				zap.Uint("sceneId", scene.ID))
		}
		if scene.Status == -1 {
			log.Log().Error("[GetStoryBoardGenerate] 场景已删除",
				zap.Int64("boardId", req.GetBoardId()),
				zap.Int("sceneIndex", i),
				zap.Uint("sceneId", scene.ID))
		}
		apiScenes = append(apiScenes, convert.ConvertStoryBoardSceneToApiStoryBoardScene(scene))
	}

	// 计算生成阶段
	generatingStage := total - generating
	log.Log().Info("[GetStoryBoardGenerate] 场景状态统计完成",
		zap.Int64("boardId", req.GetBoardId()),
		zap.Int("total", total),
		zap.Int("generating", generating),
		zap.Int("generatingStage", generatingStage))

	// 返回成功响应
	log.Log().Info("[GetStoryBoardGenerate] 获取故事板生成状态完成",
		zap.Int64("boardId", req.GetBoardId()),
		zap.Int("sceneCount", len(apiScenes)),
		zap.Int32("generatingStage", int32(generatingStage)))

	return &api.GetStoryBoardGenerateResponse{
		Code:            0,
		Message:         "OK",
		GeneratingStage: int32(generatingStage),
		List:            apiScenes,
	}, nil
}

func (s *StoryboardService) RestoreStoryboard(ctx context.Context, req *api.RestoreStoryboardRequest) (*api.RestoreStoryboardResponse, error) {
	// 函数入口日志
	log.Log().Info("[RestoreStoryboard] 开始恢复故事板",
		zap.Int64("storyId", req.GetStoryId()))

	resp := &api.RestoreStoryboardResponse{}

	// 获取故事
	log.Log().Info("[RestoreStoryboard] 开始获取故事", zap.Int64("storyId", req.GetStoryId()))
	story, err := models.GetStory(ctx, req.GetStoryId())
	if err != nil {
		log.Log().Error("[RestoreStoryboard] 获取故事失败",
			zap.Int64("storyId", req.GetStoryId()),
			zap.Error(err))
		return nil, err
	}
	if story == nil {
		log.Log().Error("[RestoreStoryboard] 故事不存在",
			zap.Int64("storyId", req.GetStoryId()))
		resp.Code = api.ResponseCode_STORY_NOT_FOUND
		resp.Message = "story not found"
		return resp, nil
	}
	log.Log().Info("[RestoreStoryboard] 成功获取故事",
		zap.Int64("storyId", req.GetStoryId()),
		zap.String("storyTitle", story.Title),
		zap.Int("status", int(story.Status)))

	// 检查故事状态
	if story.Status >= 0 {
		log.Log().Info("[RestoreStoryboard] 故事状态正常，无需恢复",
			zap.Int64("storyId", req.GetStoryId()),
			zap.Int("status", int(story.Status)))
		resp.Code = 0
		resp.Message = "story is already normal"
		return resp, nil
	}

	// 恢复故事状态
	log.Log().Info("[RestoreStoryboard] 开始恢复故事状态",
		zap.Int64("storyId", req.GetStoryId()),
		zap.Int("oldStatus", int(story.Status)))

	story.Status = 1
	err = models.UpdateStory(ctx, story)
	if err != nil {
		log.Log().Error("[RestoreStoryboard] 恢复故事状态失败",
			zap.Int64("storyId", req.GetStoryId()),
			zap.Error(err))
		return nil, err
	}
	log.Log().Info("[RestoreStoryboard] 成功恢复故事状态",
		zap.Int64("storyId", req.GetStoryId()),
		zap.Int("newStatus", int(story.Status)))

	// 返回成功响应
	log.Log().Info("[RestoreStoryboard] 故事板恢复完成",
		zap.Int64("storyId", req.GetStoryId()),
		zap.String("storyTitle", story.Title))

	resp.Code = 0
	resp.Message = "OK"
	return resp, nil
}

// 获取用户创建的故事板
func (s *StoryboardService) GetUserCreatedStoryboards(ctx context.Context, req *api.GetUserCreatedStoryboardsRequest) (*api.GetUserCreatedStoryboardsResponse, error) {
	// 函数入口日志
	log.Log().Info("[GetUserCreatedStoryboards] 开始获取用户创建的故事板",
		zap.Int64("userId", req.GetUserId()),
		zap.Int32("storyId", req.GetStoryId()),
		zap.Int64("offset", req.GetOffset()),
		zap.Int64("pageSize", req.GetPageSize()))

	// 获取故事板缓存管理器
	storyBoardCache := cache.GetStoryBoardCache()

	// 尝试从缓存获取用户创建的故事板列表
	storyboards, total, err := storyBoardCache.GetUserCreatedStoryBoards(ctx, req.GetUserId(), int(req.GetOffset()), int(req.GetPageSize()))
	if err != nil {
		log.Log().Debug("[GetUserCreatedStoryboards] 用户故事板列表缓存未命中，从数据库获取",
			zap.Int64("userId", req.GetUserId()),
			zap.Int64("offset", req.GetOffset()),
			zap.Int64("pageSize", req.GetPageSize()),
			zap.Error(err))

		// 缓存未命中，从数据库获取
		storyboards, total, err = models.GetUserCreatedStoryboardsWithStoryId(ctx, int(req.GetUserId()),
			int(req.GetStoryId()), int(req.GetOffset()), int(req.GetPageSize()))
		if err != nil {
			log.Log().Error("[GetUserCreatedStoryboards] 获取用户创建的故事板失败",
				zap.Int64("userId", req.GetUserId()),
				zap.Int32("storyId", req.GetStoryId()),
				zap.Error(err))
			return nil, err
		}

		// 将用户故事板列表存入缓存
		if cacheErr := storyBoardCache.SetUserCreatedStoryBoards(ctx, req.GetUserId(), int(req.GetOffset()), int(req.GetPageSize()), storyboards, total); cacheErr != nil {
			log.Log().Warn("[GetUserCreatedStoryboards] 设置用户故事板列表缓存失败",
				zap.Int64("userId", req.GetUserId()),
				zap.Int64("offset", req.GetOffset()),
				zap.Int64("pageSize", req.GetPageSize()),
				zap.Error(cacheErr))
		}
	} else {
		log.Log().Info("[GetUserCreatedStoryboards] 从缓存获取用户故事板列表成功",
			zap.Int64("userId", req.GetUserId()),
			zap.Int64("offset", req.GetOffset()),
			zap.Int64("pageSize", req.GetPageSize()))
	}
	log.Log().Info("[GetUserCreatedStoryboards] 成功获取用户创建的故事板",
		zap.Int64("userId", req.GetUserId()),
		zap.Int32("storyId", req.GetStoryId()),
		zap.Int("storyboardCount", len(storyboards)),
		zap.Int64("total", total))

	storiesSummary := make(map[int64]*api.StorySummaryInfo)
	storyIds := make([]int64, 0)
	for _, storyboard := range storyboards {
		storyIds = append(storyIds, int64(storyboard.StoryID))
	}
	stories, err := models.GetStoriesByIDs(ctx, storyIds)
	if err != nil {
		log.Log().Error("[GetUserCreatedStoryboards] 获取故事信息失败",
			zap.Int64("userId", req.GetUserId()),
			zap.Int("storyIdCount", len(storyIds)),
			zap.Error(err))
		return nil, err
	}
	for _, story := range stories {
		storySummaryInfo := &api.StorySummaryInfo{
			StoryId:          int64(story.ID),
			StoryTitle:       story.Name,
			StoryDescription: story.ShortDesc,
			StoryCover:       story.Avatar,
			StoryAvatar:      story.Avatar,
		}
		if storySummaryInfo.StoryTitle == "" {
			storySummaryInfo.StoryTitle = story.Title
		}
		storiesSummary[int64(story.ID)] = storySummaryInfo
	}
	apiStoryboards := make([]*api.StoryBoardActive, 0)
	for idx, storyboard := range storyboards {
		log.Log().Info("[GetUserCreatedStoryboards] 处理故事板",
			zap.Int64("userId", req.GetUserId()),
			zap.Int("storyboardIndex", idx),
			zap.Uint("storyboardId", storyboard.ID))

		newApiStoryboard := convert.ConvertStoryBoardToApiStoryBoard(storyboard)
		sences, err := models.GetStoryBoardScenesByBoard(ctx, int64(storyboard.ID))
		if err != nil {
			log.Log().Error("[GetUserCreatedStoryboards] 获取故事板场景失败",
				zap.Uint("storyboardId", storyboard.ID),
				zap.Error(err))
			continue
		}
		newApiStoryboard.Sences = &api.StoryBoardSences{
			List: make([]*api.StoryBoardSence, 0),
		}
		for _, scene := range sences {
			newApiStoryboard.Sences.List = append(newApiStoryboard.Sences.List, convert.ConvertStoryBoardSceneToApiStoryBoardScene(scene))
		}
		cu, err := s.helper.GetStoryboardCurrentUserStatus(ctx, int64(storyboard.ID))
		if err != nil {
			log.Log().Error("[GetUserCreatedStoryboards] 获取故事板当前用户状态失败",
				zap.Uint("storyboardId", storyboard.ID),
				zap.Error(err))
		}
		newApiStoryboard.CurrentUserStatus = cu
		roles, err := models.GetStoryBoardRolesByBoard(ctx, int64(storyboard.ID))
		if err != nil {
			log.Log().Error("[GetUserCreatedStoryboards] 获取故事板角色失败",
				zap.Uint("storyboardId", storyboard.ID),
				zap.Error(err))
			return nil, err
		}
		newApiStoryboard.Roles = make([]*api.StoryRole, 0)
		for _, role := range roles {
			newApiStoryboard.Roles = append(newApiStoryboard.Roles, convert.ConvertSummaryStoryRoleToApiStoryRoleInfo(role))
		}
		apiRoles := make([]*api.StoryBoardActiveRole, 0)
		for _, role := range roles {
			apiRoles = append(apiRoles, &api.StoryBoardActiveRole{
				RoleId:     int64(role.ID),
				RoleName:   role.Name,
				RoleAvatar: role.Avatar,
			})
		}
		apiStoryboards = append(apiStoryboards, &api.StoryBoardActive{
			Storyboard:        newApiStoryboard,
			TotalLikeCount:    int64(storyboard.LikeNum),
			TotalCommentCount: int64(storyboard.CommentNum),
			TotalShareCount:   int64(storyboard.ShareNum),
			TotalForkCount:    int64(storyboard.ForkNum),
			Roles:             apiRoles,
			Mtime:             storyboard.UpdateAt.Unix(),
			Summary:           storiesSummary[int64(storyboard.StoryID)],
		})
	}
	result := &api.GetUserCreatedStoryboardsResponse{
		Code:        0,
		Message:     "OK",
		Storyboards: apiStoryboards,
		Total:       total,
		Offset:      req.GetOffset(),
		PageSize:    req.GetPageSize(),
		HaveMore:    total > int64(req.GetOffset())*int64(req.GetPageSize()),
	}
	log.Log().Info("[GetUserCreatedStoryboards] 获取用户创建的故事板完成",
		zap.Int64("userId", req.GetUserId()),
		zap.Int32("storyId", req.GetStoryId()),
		zap.Int("storyboardCount", len(apiStoryboards)),
		zap.Int64("total", total))
	return result, nil
}

func (s *StoryboardService) GetNextStoryboard(ctx context.Context, req *api.GetNextStoryboardRequest) (*api.GetNextStoryboardResponse, error) {
	// 函数入口日志
	log.Log().Info("[GetNextStoryboard] 开始获取下一个故事板",
		zap.Int64("storyboardId", req.GetStoryboardId()),
		zap.Int64("offset", req.GetOffset()),
		zap.Int64("pageSize", req.GetPageSize()),
		zap.String("orderBy", req.GetOrderBy().String()))

	// 获取当前故事板
	log.Log().Info("[GetNextStoryboard] 开始获取当前故事板", zap.Int64("storyboardId", req.GetStoryboardId()))
	board, err := models.GetStoryboard(ctx, req.GetStoryboardId())
	if err != nil {
		log.Log().Error("[GetNextStoryboard] 获取故事板失败",
			zap.Int64("storyboardId", req.GetStoryboardId()),
			zap.Error(err))
		return nil, err
	}
	log.Log().Info("[GetNextStoryboard] 成功获取当前故事板",
		zap.Int64("storyboardId", req.GetStoryboardId()),
		zap.Int64("storyId", board.StoryID))

	resp := &api.GetNextStoryboardResponse{}

	// 获取下一个故事板列表
	log.Log().Info("[GetNextStoryboard] 开始获取下一个故事板列表",
		zap.Int64("storyboardId", req.GetStoryboardId()),
		zap.Int64("storyId", board.StoryID),
		zap.Int64("offset", req.GetOffset()),
		zap.Int64("pageSize", req.GetPageSize()))

	boards, err := models.GetStoryBoardByStoryAndPrevId(ctx,
		board.StoryID, req.GetStoryboardId(), int(req.GetOffset()),
		int(req.GetPageSize()), req.GetOrderBy().String())
	if err != nil {
		log.Log().Error("[GetNextStoryboard] 获取下一个故事板失败",
			zap.Int64("storyboardId", req.GetStoryboardId()),
			zap.Int64("storyId", board.StoryID),
			zap.Error(err))
		return nil, err
	}

	if len(boards) == 0 {
		log.Log().Info("[GetNextStoryboard] 没有下一个故事板",
			zap.Int64("storyboardId", req.GetStoryboardId()),
			zap.Int64("storyId", board.StoryID))
		resp.Code = 0
		resp.Message = "OK"
		resp.Offset = 0
		resp.Storyboards = make([]*api.StoryBoardActive, 0)
		resp.IsMultiBranch = false
		return resp, nil
	}

	log.Log().Info("[GetNextStoryboard] 成功获取下一个故事板列表",
		zap.Int64("storyboardId", req.GetStoryboardId()),
		zap.Int64("storyId", board.StoryID),
		zap.Int("boardCount", len(boards)))

	apiBoards := make([]*api.StoryBoardActive, 0)
	for i, board := range boards {
		log.Log().Info("[GetNextStoryboard] 处理故事板",
			zap.Int64("storyboardId", req.GetStoryboardId()),
			zap.Int("boardIndex", i),
			zap.Uint("boardId", board.ID),
			zap.Int64("creatorId", board.CreatorID))

		// 获取当前用户状态
		cu, err := s.helper.GetStoryboardCurrentUserStatus(ctx, int64(board.ID))
		if err != nil {
			log.Log().Error("[GetNextStoryboard] 获取故事板当前用户状态失败",
				zap.Uint("boardId", board.ID),
				zap.Error(err))
		}

		boardInfo := convert.ConvertStoryBoardToApiStoryBoard(board)
		boardInfo.CurrentUserStatus = cu

		// 获取场景列表
		log.Log().Info("[GetNextStoryboard] 开始获取故事板场景", zap.Uint("boardId", board.ID))
		sences, err := models.GetStoryBoardScenesByBoard(ctx, int64(board.ID))
		if err != nil {
			log.Log().Error("[GetNextStoryboard] 获取故事板场景失败",
				zap.Uint("boardId", board.ID),
				zap.Error(err))
		}
		if len(sences) != 0 {
			log.Log().Info("[GetNextStoryboard] 成功获取故事板场景",
				zap.Uint("boardId", board.ID),
				zap.Int("sceneCount", len(sences)))
			boardInfo.Sences = new(api.StoryBoardSences)
			for _, scene := range sences {
				boardInfo.Sences.List = append(boardInfo.Sences.List, ConvertStorySceneToApiScene(scene))
			}
			boardInfo.Sences.Total = int64(len(boardInfo.Sences.List))
		} else {
			log.Log().Warn("[GetNextStoryboard] 故事板场景为空", zap.Uint("boardId", board.ID))
		}

		// 获取创建者信息
		log.Log().Info("[GetNextStoryboard] 开始获取创建者信息", zap.Int64("creatorId", board.CreatorID))
		creator, err := models.GetUserById(ctx, int64(board.CreatorID))
		if err != nil {
			log.Log().Error("[GetNextStoryboard] 获取用户失败",
				zap.Int64("creatorId", board.CreatorID),
				zap.Error(err))
			return nil, err
		}
		log.Log().Info("[GetNextStoryboard] 成功获取创建者信息",
			zap.Int64("creatorId", board.CreatorID),
			zap.String("creatorName", creator.Name))

		roles, err := models.GetStoryBoardRolesByBoard(ctx, int64(board.ID))
		if err != nil {
			log.Log().Error("[GetNextStoryboard] 获取故事板角色失败",
				zap.Uint("boardId", board.ID),
				zap.Error(err))
			return nil, err
		}
		log.Log().Info("[GetNextStoryboard] 成功获取故事板角色",
			zap.Uint("boardId", board.ID),
			zap.Int("roleCount", len(roles)))

		boardInfo.Roles = make([]*api.StoryRole, 0)
		for _, role := range roles {
			boardInfo.Roles = append(boardInfo.Roles, convert.ConvertSummaryStoryRoleToApiStoryRoleInfo(role))
		}
		apiRoles := make([]*api.StoryBoardActiveRole, 0)
		for _, role := range roles {
			apiRoles = append(apiRoles, &api.StoryBoardActiveRole{
				RoleId:     int64(role.ID),
				RoleName:   role.Name,
				RoleAvatar: role.Avatar,
			})
		}

		apiBoards = append(apiBoards, &api.StoryBoardActive{
			Storyboard:        boardInfo,
			TotalLikeCount:    int64(board.LikeNum),
			TotalCommentCount: int64(board.CommentNum),
			TotalShareCount:   int64(board.ShareNum),
			TotalForkCount:    int64(board.ForkNum),
			Mtime:             board.UpdateAt.Unix(),
			Roles:             apiRoles,
			Creator: &api.StoryBoardActiveUser{
				UserId:     int64(creator.ID),
				UserName:   creator.Name,
				UserAvatar: creator.Avatar,
			},
		})
	}

	// 返回成功响应
	log.Log().Info("[GetNextStoryboard] 获取下一个故事板完成",
		zap.Int64("storyboardId", req.GetStoryboardId()),
		zap.Int("boardCount", len(apiBoards)))

	resp.Code = 0
	resp.Message = "OK"
	resp.Storyboards = apiBoards
	resp.IsMultiBranch = len(apiBoards) > 1
	return resp, nil
}

func (s *StoryboardService) PublishStoryboard(ctx context.Context, req *api.PublishStoryboardRequest) (*api.PublishStoryboardResponse, error) {
	// 函数入口日志
	log.Log().Info("[PublishStoryboard] 开始发布故事板",
		zap.Int64("storyboardId", req.GetStoryboardId()))

	// 获取故事板
	log.Log().Info("[PublishStoryboard] 开始获取故事板", zap.Int64("storyboardId", req.GetStoryboardId()))
	board, err := models.GetStoryboard(ctx, req.GetStoryboardId())
	if err != nil {
		log.Log().Error("[PublishStoryboard] 获取故事板失败",
			zap.Int64("storyboardId", req.GetStoryboardId()),
			zap.Error(err))
		return nil, err
	}
	if board == nil {
		log.Log().Error("[PublishStoryboard] 故事板不存在",
			zap.Int64("storyboardId", req.GetStoryboardId()))
		return &api.PublishStoryboardResponse{
			Code:    -1,
			Message: "storyboard not found",
		}, nil
	}
	// 更新故事板状态为已发布
	log.Log().Info("[PublishStoryboard] 开始更新故事板状态为已发布",
		zap.Int64("storyboardId", req.GetStoryboardId()),
		zap.Int("oldStage", int(board.Stage)))

	board.Stage = api.StoryboardStage_STORYBOARD_STAGE_PUBLISHED
	err = models.UpdateStoryboard(ctx, board)
	if err != nil {
		log.Log().Error("[PublishStoryboard] 更新故事板状态失败",
			zap.Int64("storyboardId", req.GetStoryboardId()),
			zap.Error(err))
		return nil, err
	}
	log.Log().Info("[PublishStoryboard] 成功更新故事板状态",
		zap.Int64("storyboardId", req.GetStoryboardId()),
		zap.Int("newStage", int(board.Stage)))
	err = models.IncreaseStoryBoardsNum(ctx, board.CreatorID, 1)
	if err != nil {
		log.Log().Error("[PublishStoryboard] 更新用户故事板数量失败",
			zap.Int64("creatorId", board.CreatorID),
			zap.Error(err))
	}
	storyRoles, err := models.GetStoryBoardRolesByBoard(ctx, int64(board.ID))
	if err != nil {
		log.Log().Error("[PublishStoryboard] 获取故事板角色失败",
			zap.Int64("storyboardId", req.GetStoryboardId()),
			zap.Error(err))
	}
	var roleIds []int64
	for _, role := range storyRoles {
		roleIds = append(roleIds, int64(role.ID))
	}
	err = models.IncreaseStoryRoleStoryboardNumBatch(ctx, int64(board.ID), roleIds, 1)
	if err != nil {
		log.Log().Error("[PublishStoryboard] 更新用户角色故事板数量失败",
			zap.Int64("creatorId", board.CreatorID),
			zap.Error(err))
	}
	// 返回成功响应
	log.Log().Info("[PublishStoryboard] 故事板发布完成",
		zap.Int64("storyboardId", req.GetStoryboardId()),
		zap.Int64("creatorId", board.CreatorID))

	// 创建故事发布通知（异步，避免阻塞主流程）
	go func() {
		bgCtx := context.Background()
		// 获取故事信息以便在通知中显示故事标题
		story, err := models.GetStory(bgCtx, board.StoryID)
		if err != nil {
			log.Log().Error("[PublishStoryboard] 获取故事信息失败", zap.Error(err), zap.Int64("storyId", board.StoryID))
			return
		}
		storyID64 := board.StoryID
		storyboardID64 := int64(board.ID)
		storyTitle := "你的故事"
		if story != nil {
			storyTitle = story.Title
		}
		notifErr := llmchatservice.CreateSystemNotification(bgCtx, llmchatservice.CreateNotificationParams{
			UserID:              board.CreatorID,
			Type:                models.SystemNotificationTypeStoryPublished,
			Title:               "故事发布提醒",
			Content:             fmt.Sprintf("你的故事《%s》的故事板已成功发布", storyTitle),
			RelatedStoryID:      &storyID64,
			RelatedStoryBoardID: &storyboardID64,
		})
		if notifErr != nil {
			log.Log().Error("[PublishStoryboard] 创建故事发布通知失败", zap.Error(notifErr), zap.Int64("storyboardId", req.GetStoryboardId()))
		}
	}()

	return &api.PublishStoryboardResponse{
		Code:    0,
		Message: "OK",
	}, nil
}

func (s *StoryboardService) CancelStoryboard(ctx context.Context, req *api.CancelStoryboardRequest) (*api.CancelStoryboardResponse, error) {
	// 函数入口日志
	log.Log().Info("[CancelStoryboard] 开始取消故事板",
		zap.Int64("storyboardId", req.GetStoryboardId()))

	// 获取故事板
	log.Log().Info("[CancelStoryboard] 开始获取故事板", zap.Int64("storyboardId", req.GetStoryboardId()))
	board, err := models.GetStoryboard(ctx, req.GetStoryboardId())
	if err != nil {
		log.Log().Error("[CancelStoryboard] 获取故事板失败",
			zap.Int64("storyboardId", req.GetStoryboardId()),
			zap.Error(err))
		return nil, err
	}
	if board == nil {
		log.Log().Error("[CancelStoryboard] 故事板不存在",
			zap.Int64("storyboardId", req.GetStoryboardId()))
		return &api.CancelStoryboardResponse{
			Code:    -1,
			Message: "storyboard not found",
		}, nil
	}
	if board.Stage == api.StoryboardStage_STORYBOARD_STAGE_PUBLISHED {
		log.Log().Info("[CancelStoryboard] 故事板已发布，不能取消",
			zap.Int64("storyboardId", req.GetStoryboardId()))
		return &api.CancelStoryboardResponse{
			Code:    -1,
			Message: "storyboard is published, cannot cancel",
		}, nil
	}

	// 更新故事板状态为已取消
	log.Log().Info("[CancelStoryboard] 开始更新故事板状态为已取消",
		zap.Int64("storyboardId", req.GetStoryboardId()),
		zap.Int("oldStage", int(board.Stage)))

	board.Stage = api.StoryboardStage_STORYBOARD_STAGE_DRAFT
	err = models.UpdateStoryboard(ctx, board)
	if err != nil {
		log.Log().Error("[CancelStoryboard] 更新故事板状态失败",
			zap.Int64("storyboardId", req.GetStoryboardId()),
			zap.Error(err))
		return nil, err
	}
	log.Log().Info("[CancelStoryboard] 成功更新故事板状态",
		zap.Int64("storyboardId", req.GetStoryboardId()),
		zap.Int("newStage", int(board.Stage)))
	err = models.DecreaseStoryBoardsNum(ctx, board.StoryID, 1)
	if err != nil {
		log.Log().Error("[CancelStoryboard] 更新用户故事板数量失败",
			zap.Int64("storyId", board.StoryID),
			zap.Error(err))
	}
	storyRoles, err := models.GetStoryBoardRolesByBoard(ctx, int64(board.ID))
	if err != nil {
		log.Log().Error("[CancelStoryboard] 获取故事板角色失败",
			zap.Int64("storyboardId", req.GetStoryboardId()),
			zap.Error(err))
	}
	var roleIds []int64
	for _, role := range storyRoles {
		roleIds = append(roleIds, int64(role.ID))
	}
	err = models.DecreaseStoryRoleStoryboardNumBatch(ctx, int64(board.ID), roleIds, 1)
	if err != nil {
		log.Log().Error("[CancelStoryboard] 更新用户角色故事板数量失败",
			zap.Int64("storyboardId", int64(board.ID)),
			zap.Error(err))
	}
	// 返回成功响应
	log.Log().Info("[CancelStoryboard] 故事板取消完成",
		zap.Int64("storyboardId", req.GetStoryboardId()),
		zap.Int64("storyId", board.StoryID))

	return &api.CancelStoryboardResponse{
		Code:    0,
		Message: "OK",
	}, nil
}

func (s *StoryboardService) GetUserWatchStoryActiveStoryBoards(ctx context.Context, req *api.GetUserWatchStoryActiveStoryBoardsRequest) (*api.GetUserWatchStoryActiveStoryBoardsResponse, error) {
	log.Log().Info("[GetUserWatchStoryActiveStoryBoards] 开始获取用户关注的故事活动故事板",
		zap.Int64("userId", req.GetUserId()),
		zap.Int64("offset", req.GetOffset()),
		zap.Int64("pageSize", req.GetPageSize()),
		zap.String("filter", req.GetFilter()))

	stortIds, err := models.GetStoriesIdByUserFollow(ctx, int64(req.GetUserId()))
	if err != nil {
		log.Log().Error("[GetUserWatchStoryActiveStoryBoards] 获取用户关注的故事ID列表失败",
			zap.Int64("userId", req.GetUserId()),
			zap.Error(err))
		return nil, err
	}
	log.Log().Info("[GetUserWatchStoryActiveStoryBoards] 获取用户关注的故事ID列表成功",
		zap.Int64("userId", req.GetUserId()),
		zap.Int("storyCount", len(stortIds)))

	if len(stortIds) == 0 {
		log.Log().Info("[GetUserWatchStoryActiveStoryBoards] 用户未关注任何故事",
			zap.Int64("userId", req.GetUserId()))
		return &api.GetUserWatchStoryActiveStoryBoardsResponse{
			Code:    0,
			Message: "OK",
			Total:   0,
		}, nil
	}

	boards, total, err := models.GetStoryBoardsByStoryIds(ctx, stortIds, int(req.GetOffset()), int(req.GetPageSize()), req.GetFilter())
	if err != nil {
		log.Log().Error("[GetUserWatchStoryActiveStoryBoards] 根据故事ID获取故事板列表失败",
			zap.Int64("userId", req.GetUserId()),
			zap.Int("storyCount", len(stortIds)),
			zap.Int64("offset", req.GetOffset()),
			zap.Int64("pageSize", req.GetPageSize()),
			zap.Error(err))
		return nil, err
	}

	targetStoryIds := make([]int64, 0)
	for _, board := range boards {
		targetStoryIds = append(targetStoryIds, int64(board.StoryID))
	}

	stories, err := models.GetStoriesByIDs(ctx, targetStoryIds)
	if err != nil {
		log.Log().Error("[GetUserWatchStoryActiveStoryBoards] 根据ID获取故事详情失败",
			zap.Int64("userId", req.GetUserId()),
			zap.Int("storyIdCount", len(targetStoryIds)),
			zap.Error(err))
		return nil, err
	}

	storiesSummary := make(map[int64]*api.StorySummaryInfo)
	for _, story := range stories {
		if _, ok := storiesSummary[int64(story.ID)]; ok {
			continue
		}
		storyItem := &api.StorySummaryInfo{
			StoryId:          int64(story.ID),
			StoryTitle:       story.Name,
			StoryDescription: story.ShortDesc,
			StoryCover:       story.Avatar,
			StoryAvatar:      story.Avatar,
		}
		if storyItem.StoryTitle == "" {
			storyItem.StoryTitle = story.Title
		}
		storiesSummary[int64(story.ID)] = storyItem
	}

	apiBoards := make([]*api.StoryBoardActive, 0)
	for _, board := range boards {
		creator, err := models.GetUserById(ctx, int64(board.CreatorID))
		if err != nil {
			log.Log().Error("[GetUserWatchStoryActiveStoryBoards] 获取创建者信息失败",
				zap.Int64("userId", req.GetUserId()),
				zap.Int64("boardId", int64(board.ID)),
				zap.Int64("creatorId", board.CreatorID),
				zap.Error(err))
			return nil, err
		}

		boardsItem := convert.ConvertStoryBoardToApiStoryBoard(board)
		sences, err := models.GetStoryBoardScenesByBoard(ctx, int64(board.ID))
		if err != nil {
			log.Log().Error("[GetUserWatchStoryActiveStoryBoards] 获取故事板场景失败",
				zap.Int64("userId", req.GetUserId()),
				zap.Int64("boardId", int64(board.ID)),
				zap.Error(err))
		} else {
			log.Log().Info("[GetUserWatchStoryActiveStoryBoards] 获取故事板场景成功",
				zap.Int64("userId", req.GetUserId()),
				zap.Int64("boardId", int64(board.ID)),
				zap.Int("sceneCount", len(sences)))
		}

		if len(sences) != 0 {
			boardsItem.Sences = new(api.StoryBoardSences)
			for _, scene := range sences {
				boardsItem.Sences.List = append(boardsItem.Sences.List, ConvertStorySceneToApiScene(scene))
			}
			boardsItem.Sences.Total = int64(len(boardsItem.Sences.List))
			log.Log().Info("[GetUserWatchStoryActiveStoryBoards] 故事板场景处理完成",
				zap.Int64("userId", req.GetUserId()),
				zap.Int64("boardId", int64(board.ID)),
				zap.Int64("totalScenes", boardsItem.Sences.Total))
		} else {
			log.Log().Warn("[GetUserWatchStoryActiveStoryBoards] 故事板场景为空",
				zap.Int64("userId", req.GetUserId()),
				zap.Int64("boardId", int64(board.ID)))
		}

		cu, err := s.helper.GetStoryboardCurrentUserStatus(ctx, int64(board.ID))
		if err != nil {
			log.Log().Error("[GetUserWatchStoryActiveStoryBoards] 获取故事板当前用户状态失败",
				zap.Int64("userId", req.GetUserId()),
				zap.Int64("boardId", int64(board.ID)),
				zap.Error(err))
		} else {
			log.Log().Info("[GetUserWatchStoryActiveStoryBoards] 获取故事板当前用户状态成功",
				zap.Int64("userId", req.GetUserId()),
				zap.Int64("boardId", int64(board.ID)))
		}
		boardsItem.CurrentUserStatus = cu

		roles, err := models.GetStoryBoardRolesByBoard(ctx, int64(board.ID))
		if err != nil {
			log.Log().Error("[GetUserWatchStoryActiveStoryBoards] 获取故事板角色失败",
				zap.Int64("userId", req.GetUserId()),
				zap.Int64("boardId", int64(board.ID)),
				zap.Error(err))
			return nil, err
		}
		log.Log().Info("[GetUserWatchStoryActiveStoryBoards] 获取故事板角色成功",
			zap.Int64("userId", req.GetUserId()),
			zap.Int64("boardId", int64(board.ID)),
			zap.Int("roleCount", len(roles)))

		boardsItem.Roles = make([]*api.StoryRole, 0)
		for _, role := range roles {
			boardsItem.Roles = append(boardsItem.Roles, convert.ConvertSummaryStoryRoleToApiStoryRoleInfo(role))
		}
		apiRoles := make([]*api.StoryBoardActiveRole, 0)
		for _, role := range roles {
			apiRoles = append(apiRoles, &api.StoryBoardActiveRole{
				RoleId:     int64(role.ID),
				RoleName:   role.Name,
				RoleAvatar: role.Avatar,
			})
		}

		apiBoardsItem := &api.StoryBoardActive{
			Storyboard:        boardsItem,
			TotalLikeCount:    int64(board.LikeNum),
			TotalCommentCount: int64(board.CommentNum),
			TotalShareCount:   int64(board.ShareNum),
			TotalForkCount:    int64(board.ForkNum),
			Users:             []*api.StoryBoardActiveUser{},
			Roles:             apiRoles,
			Creator: &api.StoryBoardActiveUser{
				UserId:     int64(creator.ID),
				UserName:   creator.Name,
				UserAvatar: creator.Avatar,
			},
			Summary: storiesSummary[int64(board.StoryID)],
			Isliked: true,
			Mtime:   boardsItem.Mtime,
		}
		apiBoards = append(apiBoards, apiBoardsItem)
	}

	log.Log().Info("[GetUserWatchStoryActiveStoryBoards] 获取用户关注的故事活动故事板完成",
		zap.Int64("userId", req.GetUserId()),
		zap.Int("totalBoardCount", len(apiBoards)),
		zap.Int64("offset", req.GetOffset()),
		zap.Int64("pageSize", req.GetPageSize()),
		zap.Int64("total", total),
		zap.Bool("HaveMore", total > int64(req.GetOffset())*int64(req.GetPageSize())),
	)

	return &api.GetUserWatchStoryActiveStoryBoardsResponse{
		Code:        0,
		Message:     "OK",
		Storyboards: apiBoards,
		Total:       int64(len(apiBoards)),
		Offset:      req.GetOffset(),
		PageSize:    req.GetPageSize(),
		HaveMore:    total > int64(req.GetOffset())*int64(req.GetPageSize()),
	}, nil
}

func (s *StoryboardService) GetUserWatchRoleActiveStoryBoards(ctx context.Context, req *api.GetUserWatchRoleActiveStoryBoardsRequest) (*api.GetUserWatchRoleActiveStoryBoardsResponse, error) {
	rolesIds, err := models.GetStoryRolesIDByUserFollow(ctx, int64(req.GetUserId()))
	if err != nil {
		return nil, err
	}
	if len(rolesIds) == 0 {
		return &api.GetUserWatchRoleActiveStoryBoardsResponse{
			Code:    0,
			Message: "OK",
			Total:   0,
		}, nil
	}
	boards, roleBoardList, err := models.GetStoryBoardsByRolesID(ctx, rolesIds, int(req.GetOffset()), int(req.GetPageSize()), req.GetFilter())
	if err != nil {
		return nil, err
	}
	roleBoardMap := make(map[int64][]*models.StoryBoardRole)
	for _, roleBoard := range roleBoardList {
		roleBoardMap[roleBoard.BoardId] = append(roleBoardMap[roleBoard.BoardId], roleBoard)
	}
	targetStoryIds := make([]int64, 0)
	for _, board := range boards {
		targetStoryIds = append(targetStoryIds, int64(board.StoryID))
	}
	stories, err := models.GetStoriesByIDs(ctx, targetStoryIds)
	if err != nil {
		return nil, err
	}
	storiesSummary := make(map[int64]*api.StorySummaryInfo)
	for _, story := range stories {
		if story.Status != 1 {
			continue
		}
		if story.Deleted == true {
			continue
		}
		if _, ok := storiesSummary[int64(story.ID)]; ok {
			continue
		}
		storyItem := &api.StorySummaryInfo{
			StoryId:          int64(story.ID),
			StoryTitle:       story.Name,
			StoryDescription: story.ShortDesc,
			StoryCover:       utils.DefaultStoryAvatorUrl,
			StoryAvatar:      story.Avatar,
		}
		if storyItem.StoryTitle == "" {
			storyItem.StoryTitle = story.Title
		}
		storiesSummary[int64(story.ID)] = storyItem
	}
	apiBoards := make([]*api.StoryBoardActive, 0)
	for _, board := range boards {
		creator, err := models.GetUserById(ctx, int64(board.CreatorID))
		if err != nil {
			return nil, err
		}
		boardsItem := convert.ConvertStoryBoardToApiStoryBoard(board)
		sences, err := models.GetStoryBoardScenesByBoard(ctx, int64(board.ID))
		if err != nil {
			log.Log().Error("get board sences failed", zap.Error(err))
		}
		if len(sences) != 0 {
			boardsItem.Sences = new(api.StoryBoardSences)
			for _, scene := range sences {
				boardsItem.Sences.List = append(boardsItem.Sences.List, ConvertStorySceneToApiScene(scene))
			}
			boardsItem.Sences.Total = int64(len(boardsItem.Sences.List))
		} else {
			log.Log().Warn("story sences is empty")
		}
		cu, err := s.helper.GetStoryboardCurrentUserStatus(ctx, int64(board.ID))
		if err != nil {
			log.Log().Error("get storyboard current user status failed", zap.Error(err))
		}
		boardsItem.CurrentUserStatus = cu
		for _, role := range roleBoardMap[int64(board.ID)] {
			apiRoles := make([]*api.StoryBoardActiveRole, 0)
			apiRoles = append(apiRoles, &api.StoryBoardActiveRole{
				RoleId:     int64(role.ID),
				RoleName:   role.Name,
				RoleAvatar: role.Avatar,
			})
			apiBoards = append(apiBoards, &api.StoryBoardActive{
				Storyboard:        boardsItem,
				TotalLikeCount:    int64(board.LikeNum),
				TotalCommentCount: int64(board.CommentNum),
				TotalShareCount:   int64(board.ShareNum),
				TotalForkCount:    int64(board.ForkNum),
				Roles:             apiRoles,
				Mtime:             board.UpdateAt.Unix(),
				Creator: &api.StoryBoardActiveUser{
					UserId:     int64(creator.ID),
					UserName:   creator.Name,
					UserAvatar: creator.Avatar,
				},
				Summary: storiesSummary[int64(board.StoryID)],
			})
		}

	}
	return &api.GetUserWatchRoleActiveStoryBoardsResponse{
		Code:        api.ResponseCode_OK,
		Message:     "OK",
		Storyboards: apiBoards,
		Total:       int64(len(boards)),
		Offset:      req.GetOffset(),
		PageSize:    req.GetPageSize(),
	}, nil
}

func (s *StoryboardService) GetUnPublishStoryboard(ctx context.Context, req *api.GetUnPublishStoryboardRequest) (*api.GetUnPublishStoryboardResponse, error) {
	log.Log().Info("[GetUnPublishStoryboard] 开始获取未发布的故事板",
		zap.Int64("userId", req.GetUserId()),
		zap.Int64("offset", req.GetOffset()),
		zap.Int64("pageSize", req.GetPageSize()))

	boards, err := models.GetUnPublishedStoryBoardsByUserId(ctx, req.GetUserId(), int(req.GetOffset()), int(req.GetPageSize()), "create_at desc")
	if err != nil {
		log.Log().Error("[GetUnPublishStoryboard] 获取用户未发布故事板失败",
			zap.Int64("userId", req.GetUserId()),
			zap.Int64("offset", req.GetOffset()),
			zap.Int64("pageSize", req.GetPageSize()),
			zap.Error(err))
		return nil, err
	}
	log.Log().Info("[GetUnPublishStoryboard] 获取用户未发布故事板成功",
		zap.Int64("userId", req.GetUserId()),
		zap.Int("boardCount", len(boards)))

	targetStoryIds := make([]int64, 0)
	for _, board := range boards {
		targetStoryIds = append(targetStoryIds, int64(board.StoryID))
	}

	stories, err := models.GetStoriesByIDs(ctx, targetStoryIds)
	if err != nil {
		log.Log().Error("[GetUnPublishStoryboard] 根据ID获取故事详情失败",
			zap.Int64("userId", req.GetUserId()),
			zap.Int("storyIdCount", len(targetStoryIds)),
			zap.Error(err))
		return nil, err
	}
	log.Log().Info("[GetUnPublishStoryboard] 根据ID获取故事详情成功",
		zap.Int64("userId", req.GetUserId()),
		zap.Int("storyCount", len(stories)))

	storiesSummary := make(map[int64]*api.StorySummaryInfo)
	for _, story := range stories {
		if story.Status != 1 {
			log.Log().Info("[GetUnPublishStoryboard] 跳过状态不为1的故事",
				zap.Int64("userId", req.GetUserId()),
				zap.Int64("storyId", int64(story.ID)),
				zap.Int("storyStatus", int(story.Status)))
			continue
		}
		if story.Deleted == true {
			log.Log().Info("[GetUnPublishStoryboard] 跳过已删除的故事",
				zap.Int64("userId", req.GetUserId()),
				zap.Int64("storyId", int64(story.ID)))
			continue
		}
		if _, ok := storiesSummary[int64(story.ID)]; ok {
			continue
		}
		storyItem := &api.StorySummaryInfo{
			StoryId:          int64(story.ID),
			StoryTitle:       story.Name,
			StoryDescription: story.ShortDesc,
			StoryCover:       utils.DefaultStoryAvatorUrl,
			StoryAvatar:      story.Avatar,
		}
		if storyItem.StoryTitle == "" {
			storyItem.StoryTitle = story.Title
		}
		storiesSummary[int64(story.ID)] = storyItem
	}
	log.Log().Info("[GetUnPublishStoryboard] 构建故事摘要信息完成",
		zap.Int64("userId", req.GetUserId()),
		zap.Int("summaryCount", len(storiesSummary)))

	apiBoards := make([]*api.StoryBoardActive, 0)
	for i, board := range boards {
		log.Log().Info("[GetUnPublishStoryboard] 处理故事板",
			zap.Int64("userId", req.GetUserId()),
			zap.Int("boardIndex", i),
			zap.Int64("boardId", int64(board.ID)),
			zap.Int64("storyId", board.StoryID))

		creator, err := models.GetUserById(ctx, int64(board.CreatorID))
		if err != nil {
			log.Log().Error("[GetUnPublishStoryboard] 获取创建者信息失败",
				zap.Int64("userId", req.GetUserId()),
				zap.Int64("boardId", int64(board.ID)),
				zap.Int64("creatorId", board.CreatorID),
				zap.Error(err))
			return nil, err
		}
		log.Log().Info("[GetUnPublishStoryboard] 获取创建者信息成功",
			zap.Int64("userId", req.GetUserId()),
			zap.Int64("boardId", int64(board.ID)),
			zap.Int64("creatorId", board.CreatorID),
			zap.String("creatorName", creator.Name))

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
				UserName:   creator.Name,
				UserAvatar: creator.Avatar,
			},
			Summary: storiesSummary[int64(board.StoryID)],
		})

		log.Log().Info("[GetUnPublishStoryboard] 故事板处理完成",
			zap.Int64("userId", req.GetUserId()),
			zap.Int("boardIndex", i),
			zap.Int64("boardId", int64(board.ID)),
			zap.Int64("totalLikeCount", int64(board.LikeNum)),
			zap.Int64("totalCommentCount", int64(board.CommentNum)),
			zap.Int64("totalShareCount", int64(board.ShareNum)),
			zap.Int64("totalForkCount", int64(board.ForkNum)))
	}

	log.Log().Info("[GetUnPublishStoryboard] 获取未发布故事板完成",
		zap.Int64("userId", req.GetUserId()),
		zap.Int("totalBoardCount", len(apiBoards)),
		zap.Int64("offset", req.GetOffset()),
		zap.Int64("pageSize", req.GetPageSize()))

	return &api.GetUnPublishStoryboardResponse{
		Code:              api.ResponseCode_OK,
		Message:           "OK",
		Storyboardactives: apiBoards,
		Total:             int64(len(boards)),
		HaveMore:          int64(len(boards)) > int64(req.GetOffset())*int64(req.GetPageSize()),
	}, nil
}

func (s *StoryboardService) SaveStoryboardCraft(ctx context.Context, req *api.SaveStoryboardCraftRequest) (*api.SaveStoryboardCraftResponse, error) {
	log.Log().Info("[SaveStoryboardCraft] 入参", zap.Any("req", req))
	err := models.UpdateStoryboardPublishedState(ctx, req.GetStoryboardId(), api.StoryboardStage_STORYBOARD_STAGE_FINISHED)
	if err != nil {
		log.Log().Error("[SaveStoryboardCraft] 更新分镜发布状态失败", zap.Error(err), zap.Int64("storyboardId", req.GetStoryboardId()))
		return nil, err
	}
	log.Log().Info("[SaveStoryboardCraft] 更新分镜发布状态成功", zap.Int64("storyboardId", req.GetStoryboardId()))
	return &api.SaveStoryboardCraftResponse{
		Code:    api.ResponseCode_OK,
		Message: "OK",
	}, nil
}

func (s *StoryboardService) UserStoryboardDraftlist(ctx context.Context, req *api.UserStoryboardDraftlistRequest) (*api.UserStoryboardDraftlistResponse, error) {
	log.Log().Info("[UserStoryboardDraftlist] 开始获取用户草稿列表",
		zap.Int64("userId", req.GetUserId()),
		zap.Int64("offset", req.GetOffset()),
		zap.Int64("pageSize", req.GetPageSize()),
		zap.Int64("storyId", req.GetStoryId()))

	if req.GetUserId() <= 0 {
		log.Log().Warn("[UserStoryboardDraftlist] 非法用户ID", zap.Int64("userId", req.GetUserId()))
		return &api.UserStoryboardDraftlistResponse{
			Code:    api.ResponseCode_INVALID_PARAMETER,
			Message: "invalid user_id",
		}, nil
	}

	page := int(req.GetOffset())
	if page <= 0 {
		page = 1
	}
	pageSize := int(req.GetPageSize())
	if pageSize <= 0 {
		pageSize = 10
	}
	if pageSize > 100 {
		pageSize = 100
	}

	boards, total, err := models.GetUserStoryboardDrafts(ctx, req.GetUserId(), req.GetStoryId(), page, pageSize)
	if err != nil {
		log.Log().Error("[UserStoryboardDraftlist] 查询草稿失败",
			zap.Int64("userId", req.GetUserId()),
			zap.Error(err))
		return nil, err
	}

	drafts := make([]*api.StoryboardDraftDetail, 0, len(boards))
	for _, board := range boards {
		detail, buildErr := buildStoryboardDraftDetail(ctx, board)
		if buildErr != nil {
			log.Log().Error("[UserStoryboardDraftlist] 构建草稿详情失败",
				zap.Uint("boardId", board.ID),
				zap.Error(buildErr))
			return nil, buildErr
		}
		drafts = append(drafts, detail)
	}

	haveMore := int64(page*pageSize) < total
	resp := &api.UserStoryboardDraftlistResponse{
		Code:     api.ResponseCode_OK,
		Message:  "OK",
		Drafts:   drafts,
		Total:    total,
		HaveMore: haveMore,
	}

	log.Log().Info("[UserStoryboardDraftlist] 成功获取草稿列表",
		zap.Int64("userId", req.GetUserId()),
		zap.Int("draftCount", len(drafts)),
		zap.Int64("total", total),
		zap.Bool("haveMore", haveMore))
	return resp, nil
}

func (s *StoryboardService) DeleteUserStoryboardDraft(ctx context.Context, req *api.DeleteUserStoryboardDraftRequest) (*api.DeleteUserStoryboardDraftResponse, error) {
	log.Log().Info("[DeleteUserStoryboardDraft] 开始删除用户草稿",
		zap.Int64("userId", req.GetUserId()),
		zap.Int64("draftId", req.GetDraftId()),
		zap.Int64("storyId", req.GetStoryId()))

	if req.GetUserId() <= 0 || req.GetDraftId() <= 0 {
		log.Log().Warn("[DeleteUserStoryboardDraft] 非法参数",
			zap.Int64("userId", req.GetUserId()),
			zap.Int64("draftId", req.GetDraftId()))
		return &api.DeleteUserStoryboardDraftResponse{
			Code:    api.ResponseCode_INVALID_PARAMETER,
			Message: "invalid user_id or draft_id",
		}, nil
	}

	board, err := models.GetStoryboard(ctx, req.GetDraftId())
	if err != nil {
		log.Log().Error("[DeleteUserStoryboardDraft] 获取草稿失败",
			zap.Int64("draftId", req.GetDraftId()),
			zap.Error(err))
		return nil, err
	}
	if board == nil {
		log.Log().Warn("[DeleteUserStoryboardDraft] 草稿不存在",
			zap.Int64("draftId", req.GetDraftId()))
		return &api.DeleteUserStoryboardDraftResponse{
			Code:    api.ResponseCode_RESOURCE_NOT_FOUND,
			Message: "draft not found",
		}, nil
	}
	if board.CreatorID != req.GetUserId() {
		log.Log().Warn("[DeleteUserStoryboardDraft] 无权限删除草稿",
			zap.Int64("draftId", req.GetDraftId()),
			zap.Int64("creatorId", board.CreatorID),
			zap.Int64("requestUserId", req.GetUserId()))
		return &api.DeleteUserStoryboardDraftResponse{
			Code:    api.ResponseCode_PERMISSION_DENIED,
			Message: "permission denied",
		}, nil
	}
	if req.GetStoryId() > 0 && board.StoryID != req.GetStoryId() {
		log.Log().Warn("[DeleteUserStoryboardDraft] 故事ID不匹配",
			zap.Int64("draftStoryId", board.StoryID),
			zap.Int64("requestStoryId", req.GetStoryId()))
		return &api.DeleteUserStoryboardDraftResponse{
			Code:    api.ResponseCode_INVALID_PARAMETER,
			Message: "story mismatch",
		}, nil
	}
	if api.StoryboardStage(board.Stage) == api.StoryboardStage_STORYBOARD_STAGE_PUBLISHED {
		log.Log().Warn("[DeleteUserStoryboardDraft] 草稿已发布，无法删除",
			zap.Int64("draftId", req.GetDraftId()))
		return &api.DeleteUserStoryboardDraftResponse{
			Code:    api.ResponseCode_RESOURCE_ALREADY_EXISTS,
			Message: "storyboard already published",
		}, nil
	}

	if err := models.DeleteUserStoryboardDraft(ctx, req.GetUserId(), req.GetDraftId()); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			log.Log().Warn("[DeleteUserStoryboardDraft] 草稿未找到",
				zap.Int64("draftId", req.GetDraftId()))
			return &api.DeleteUserStoryboardDraftResponse{
				Code:    api.ResponseCode_RESOURCE_NOT_FOUND,
				Message: "draft not found",
			}, nil
		}
		log.Log().Error("[DeleteUserStoryboardDraft] 删除草稿失败",
			zap.Int64("draftId", req.GetDraftId()),
			zap.Error(err))
		return nil, err
	}

	resp := &api.DeleteUserStoryboardDraftResponse{
		Code:    api.ResponseCode_OK,
		Message: "OK",
	}
	log.Log().Info("[DeleteUserStoryboardDraft] 删除草稿成功",
		zap.Int64("userId", req.GetUserId()),
		zap.Int64("draftId", req.GetDraftId()))
	return resp, nil
}

func buildStoryboardDraftDetail(ctx context.Context, board *models.StoryBoard) (*api.StoryboardDraftDetail, error) {
	detail := &api.StoryboardDraftDetail{
		DraftId:      int64(board.ID),
		StoryId:      board.StoryID,
		StoryboardId: int64(board.ID),
		Title:        board.Title,
		Content:      board.Description,
		Background:   board.Avatar,
		CreatedAt:    board.CreateAt.Unix(),
		UpdatedAt:    board.UpdateAt.Unix(),
		Stage:        api.StoryboardStage(board.Stage),
		UserId:       board.CreatorID,
	}

	scenes, err := models.GetStoryBoardScenesByBoard(ctx, int64(board.ID))
	if err != nil {
		return nil, err
	}
	apiScenes := make([]*api.StoryBoardSence, 0, len(scenes))
	for _, scene := range scenes {
		apiScenes = append(apiScenes, convert.ConvertStoryBoardSceneToApiStoryBoardScene(scene))
	}
	detail.Sences = &api.StoryBoardSences{List: apiScenes}

	roles, err := models.GetStoryBoardRolesByBoard(ctx, int64(board.ID))
	if err != nil {
		return nil, err
	}
	apiRoles := make([]*api.StoryRole, 0, len(roles))
	for _, role := range roles {
		apiRoles = append(apiRoles, convert.ConvertSummaryStoryRoleToApiStoryRoleInfo(role))
	}
	detail.Roles = apiRoles

	if board.Params != "" {
		var params api.StoryBoardParams
		if err := json.Unmarshal([]byte(board.Params), &params); err != nil {
			log.Log().Warn("[buildStoryboardDraftDetail] 解析故事板参数失败",
				zap.Uint("boardId", board.ID),
				zap.Error(err))
		} else {
			if params.BoardId == 0 {
				params.BoardId = int64(board.ID)
			}
			detail.Params = &params
		}
	}

	return detail, nil
}

func (s *StoryboardService) UserDraftStoryboardDetail(ctx context.Context, req *api.UserDraftStoryboardDetailRequest) (*api.UserDraftStoryboardDetailResponse, error) {
	log.Log().Info("[UserDraftStoryboardDetail] 开始获取草稿详情",
		zap.Int64("userId", req.GetUserId()),
		zap.Int64("draftId", req.GetDraftId()))

	if req.GetUserId() <= 0 || req.GetDraftId() <= 0 {
		log.Log().Warn("[UserDraftStoryboardDetail] 非法参数",
			zap.Int64("userId", req.GetUserId()),
			zap.Int64("draftId", req.GetDraftId()))
		return &api.UserDraftStoryboardDetailResponse{
			Code:    api.ResponseCode_INVALID_PARAMETER,
			Message: "invalid user_id or draft_id",
		}, nil
	}

	board, err := models.GetStoryboard(ctx, req.GetDraftId())
	if err != nil {
		log.Log().Error("[UserDraftStoryboardDetail] 获取草稿失败",
			zap.Int64("draftId", req.GetDraftId()),
			zap.Error(err))
		return nil, err
	}
	if board == nil {
		log.Log().Warn("[UserDraftStoryboardDetail] 草稿不存在",
			zap.Int64("draftId", req.GetDraftId()))
		return &api.UserDraftStoryboardDetailResponse{
			Code:    api.ResponseCode_RESOURCE_NOT_FOUND,
			Message: "draft not found",
		}, nil
	}
	if board.CreatorID != req.GetUserId() {
		log.Log().Warn("[UserDraftStoryboardDetail] 无权限查看草稿",
			zap.Int64("draftId", req.GetDraftId()),
			zap.Int64("creatorId", board.CreatorID),
			zap.Int64("requestUserId", req.GetUserId()))
		return &api.UserDraftStoryboardDetailResponse{
			Code:    api.ResponseCode_PERMISSION_DENIED,
			Message: "permission denied",
		}, nil
	}
	if api.StoryboardStage(board.Stage) == api.StoryboardStage_STORYBOARD_STAGE_PUBLISHED {
		log.Log().Warn("[UserDraftStoryboardDetail] 草稿已发布",
			zap.Int64("draftId", req.GetDraftId()))
		return &api.UserDraftStoryboardDetailResponse{
			Code:    api.ResponseCode_RESOURCE_ALREADY_EXISTS,
			Message: "storyboard already published",
		}, nil
	}

	detail, err := buildStoryboardDraftDetail(ctx, board)
	if err != nil {
		log.Log().Error("[UserDraftStoryboardDetail] 构建草稿详情失败",
			zap.Int64("draftId", req.GetDraftId()),
			zap.Error(err))
		return nil, err
	}

	resp := &api.UserDraftStoryboardDetailResponse{
		Code:    api.ResponseCode_OK,
		Message: "OK",
		Detail:  detail,
	}

	log.Log().Info("[UserDraftStoryboardDetail] 成功获取草稿详情",
		zap.Int64("draftId", req.GetDraftId()))
	return resp, nil
}

func (s *StoryboardService) GetStoryboardGenerationRoadmap(ctx context.Context, req *api.GetStoryboardGenerationRoadmapRequest) (*api.GetStoryboardGenerationRoadmapResponse, error) {
	log.Log().Info("[GetStoryboardGenerationRoadmap] 入参",
		zap.Int64("storyId", req.GetStoryId()),
		zap.Int64("storyboardId", req.GetStoryboardId()),
		zap.Int64("userId", req.GetUserId()))

	if req.GetStoryId() <= 0 || req.GetStoryboardId() <= 0 || req.GetUserId() <= 0 {
		log.Log().Warn("[GetStoryboardGenerationRoadmap] 参数非法",
			zap.Int64("storyId", req.GetStoryId()),
			zap.Int64("storyboardId", req.GetStoryboardId()),
			zap.Int64("userId", req.GetUserId()))
		return &api.GetStoryboardGenerationRoadmapResponse{
			Code:    api.ResponseCode_INVALID_PARAMETER,
			Message: "invalid story_id, storyboard_id or user_id",
		}, nil
	}

	story, err := models.GetStory(ctx, req.GetStoryId())
	if err != nil {
		log.Log().Error("[GetStoryboardGenerationRoadmap] 获取故事失败",
			zap.Int64("storyId", req.GetStoryId()),
			zap.Error(err))
		return nil, err
	}
	if story == nil {
		log.Log().Warn("[GetStoryboardGenerationRoadmap] 故事不存在",
			zap.Int64("storyId", req.GetStoryId()))
		return &api.GetStoryboardGenerationRoadmapResponse{
			Code:    api.ResponseCode_RESOURCE_NOT_FOUND,
			Message: "story not found",
		}, nil
	}

	board, err := models.GetStoryboard(ctx, req.GetStoryboardId())
	if err != nil {
		log.Log().Error("[GetStoryboardGenerationRoadmap] 获取故事板失败",
			zap.Int64("storyboardId", req.GetStoryboardId()),
			zap.Error(err))
		return nil, err
	}
	if board == nil || board.StoryID != req.GetStoryId() {
		log.Log().Warn("[GetStoryboardGenerationRoadmap] 故事板不存在或不属于指定故事",
			zap.Int64("storyboardId", req.GetStoryboardId()),
			zap.Int64("storyId", req.GetStoryId()))
		return &api.GetStoryboardGenerationRoadmapResponse{
			Code:    api.ResponseCode_RESOURCE_NOT_FOUND,
			Message: "storyboard not found",
		}, nil
	}

	if board.CreatorID != req.GetUserId() && story.CreatorID != req.GetUserId() && story.OwnerID != req.GetUserId() {
		log.Log().Warn("[GetStoryboardGenerationRoadmap] 用户无权限访问",
			zap.Int64("requestUserId", req.GetUserId()),
			zap.Int64("boardCreator", board.CreatorID),
			zap.Int64("storyCreator", story.CreatorID))
		return &api.GetStoryboardGenerationRoadmapResponse{
			Code:    api.ResponseCode_PERMISSION_DENIED,
			Message: "permission denied",
		}, nil
	}

	history := &api.StoryGenerationHistory{
		StoryInfo:              convert.ConvertStoryToApiStory(story),
		Roles:                  make([]*api.StoryRole, 0),
		PolishRecords:          make([]*api.AIPolishRecord, 0),
		TranslationRecords:     make([]*api.ChapterTranslationRecord, 0),
		PromptRecords:          make([]*api.GenerationPromptRecord, 0),
		FinalContent:           make([]*api.StoryBoard, 0),
		TotalTokenConsumptions: make([]*api.TokenConsumption, 0),
		ChildStoryboardCount:   0,
		CreatedAt:              board.CreateAt.Unix(),
		UpdatedAt:              board.UpdateAt.Unix(),
	}

	roles, err := models.GetStoryBoardRolesByBoard(ctx, int64(board.ID))
	if err != nil {
		log.Log().Error("[GetStoryboardGenerationRoadmap] 获取角色失败",
			zap.Int64("storyboardId", req.GetStoryboardId()),
			zap.Error(err))
		return nil, err
	}
	for _, role := range roles {
		history.Roles = append(history.Roles, convert.ConvertSummaryStoryRoleToApiStoryRoleInfo(role))
	}

	rootScenes, err := models.GetStoryBoardScenesByBoard(ctx, int64(board.ID))
	if err != nil {
		log.Log().Error("[GetStoryboardGenerationRoadmap] 获取场景失败",
			zap.Int64("storyboardId", req.GetStoryboardId()),
			zap.Error(err))
		return nil, err
	}

	rootBoard := convert.ConvertStoryBoardToApiStoryBoard(board)
	rootBoard.Sences = buildAPIScenes(rootScenes)
	history.FinalContent = append(history.FinalContent, rootBoard)

	promptRecords, translationRecords := buildPromptAndTranslationRecords(rootScenes)
	history.PromptRecords = append(history.PromptRecords, promptRecords...)
	history.TranslationRecords = append(history.TranslationRecords, translationRecords...)

	childBoards, err := models.GetStoryboardsByPrevId(ctx, int64(board.ID))
	if err != nil {
		log.Log().Error("[GetStoryboardGenerationRoadmap] 获取子故事板失败",
			zap.Int64("storyboardId", req.GetStoryboardId()),
			zap.Error(err))
		return nil, err
	}
	history.ChildStoryboardCount = int64(len(childBoards))
	for _, child := range childBoards {
		childScenes, childErr := models.GetStoryBoardScenesByBoard(ctx, int64(child.ID))
		if childErr != nil {
			log.Log().Error("[GetStoryboardGenerationRoadmap] 获取子故事板场景失败",
				zap.Int64("childBoardId", int64(child.ID)),
				zap.Error(childErr))
			return nil, childErr
		}
		childBoard := convert.ConvertStoryBoardToApiStoryBoard(child)
		childBoard.Sences = buildAPIScenes(childScenes)
		history.FinalContent = append(history.FinalContent, childBoard)

		childPrompts, childTranslations := buildPromptAndTranslationRecords(childScenes)
		history.PromptRecords = append(history.PromptRecords, childPrompts...)
		history.TranslationRecords = append(history.TranslationRecords, childTranslations...)
	}

	if polish := buildPolishRecord(story, board); polish != nil {
		history.PolishRecords = append(history.PolishRecords, polish)
	}

	history.ChapterInfo = s.extractStoryChapterInfo(ctx, story, req.GetStoryboardId())

	tokens, err := collectTokenConsumptions(ctx, story, board)
	if err != nil {
		log.Log().Error("[GetStoryboardGenerationRoadmap] 汇总Token信息失败",
			zap.Int64("storyId", req.GetStoryId()),
			zap.Int64("storyboardId", req.GetStoryboardId()),
			zap.Error(err))
		return nil, err
	}
	history.TotalTokenConsumptions = append(history.TotalTokenConsumptions, tokens...)

	if creator, err := models.GetUserById(ctx, story.CreatorID); err != nil {
		log.Log().Error("[GetStoryboardGenerationRoadmap] 获取创建者失败",
			zap.Int64("creatorId", story.CreatorID),
			zap.Error(err))
		return nil, err
	} else if creator != nil {
		history.Creator = convert.ConvertUserToApiUser(creator)
	}

	resp := &api.GetStoryboardGenerationRoadmapResponse{
		Code:    api.ResponseCode_OK,
		Message: "OK",
		Data:    history,
	}

	log.Log().Info("[GetStoryboardGenerationRoadmap] 成功返回结果",
		zap.Int64("storyId", req.GetStoryId()),
		zap.Int64("storyboardId", req.GetStoryboardId()))
	return resp, nil
}

func buildAPIScenes(scenes []*models.StoryBoardScene) *api.StoryBoardSences {
	apiScenes := make([]*api.StoryBoardSence, 0, len(scenes))
	for _, scene := range scenes {
		apiScenes = append(apiScenes, convert.ConvertStoryBoardSceneToApiStoryBoardScene(scene))
	}
	return &api.StoryBoardSences{
		Total: int64(len(apiScenes)),
		List:  apiScenes,
	}
}

func buildPromptAndTranslationRecords(scenes []*models.StoryBoardScene) ([]*api.GenerationPromptRecord, []*api.ChapterTranslationRecord) {
	promptRecords := make([]*api.GenerationPromptRecord, 0, len(scenes))
	translationRecords := make([]*api.ChapterTranslationRecord, 0, len(scenes))
	for _, scene := range scenes {
		record := &api.GenerationPromptRecord{
			ContentType:     "scene",
			ContentId:       int64(scene.ID),
			ImagePrompt:     scene.ImagePrompts,
			VideoPrompt:     scene.VideoPrompts,
			GeneratedImages: filterEmpty(strings.Split(scene.ImageUrl, ",")),
			GeneratedVideo:  scene.VideoUrl,
			GeneratedAt:     scene.UpdateAt.Unix(),
		}
		if record.ImagePrompt != "" || record.VideoPrompt != "" || len(record.GeneratedImages) > 0 || record.GeneratedVideo != "" {
			promptRecords = append(promptRecords, record)
		}

		if scene.GenResult != "" || scene.Content != "" {
			translationRecords = append(translationRecords, &api.ChapterTranslationRecord{
				ChapterId:           strconv.FormatInt(int64(scene.ID), 10),
				ChapterTitle:        scene.Content,
				OriginalScene:       scene.Content,
				TranslatedScene:     scene.GenResult,
				OriginalImageDesc:   scene.ImagePrompts,
				TranslatedImageDesc: scene.GenResult,
				TranslatedAt:        scene.UpdateAt.Unix(),
			})
		}
	}
	return promptRecords, translationRecords
}

func buildPolishRecord(story *models.Story, board *models.StoryBoard) *api.AIPolishRecord {
	original := strings.TrimSpace(story.Origin)
	polished := strings.TrimSpace(board.Description)
	if polished == "" {
		polished = strings.TrimSpace(story.ShortDesc)
	}
	if original == "" || polished == "" || original == polished {
		return nil
	}
	return &api.AIPolishRecord{
		OriginalContent: original,
		PolishedContent: polished,
		PolishedAt:      board.UpdateAt.Unix(),
		PolishType:      "story_description",
	}
}

func (s *StoryboardService) extractStoryChapterInfo(ctx context.Context, story *models.Story, boardID int64) *api.StoryInfo {
	storyGens, err := models.GetStoryGensByStory(ctx, int64(story.ID), 1)
	if err != nil {
		log.Log().Warn("[GetStoryboardGenerationRoadmap] 获取StoryGen列表失败",
			zap.Int64("storyId", int64(story.ID)),
			zap.Error(err))
	} else {
		for _, gen := range storyGens {
			if gen.Content == "" {
				continue
			}
			detail := new(api.RenderStoryDetail)
			if err := json.Unmarshal([]byte(gen.Content), detail); err != nil {
				log.Log().Warn("[GetStoryboardGenerationRoadmap] 解析StoryGen内容失败",
					zap.Int64("storyGenId", int64(gen.ID)),
					zap.Error(err))
				continue
			}
			if detail.GetResult() != nil {
				return detail.GetResult()
			}
		}
	}
	return nil
}

func collectTokenConsumptions(ctx context.Context, story *models.Story, board *models.StoryBoard) ([]*api.TokenConsumption, error) {
	records := make([]*api.TokenConsumption, 0)
	appendRecord := func(tokenNum int64, sourceID int64, consumedAt int64, purpose string) {
		if tokenNum <= 0 {
			return
		}
		records = append(records, &api.TokenConsumption{
			TokenCount: tokenNum,
			SourceType: api.TokenSourceType_TOKEN_SOURCE_PERSONAL,
			SourceId:   sourceID,
			ConsumedAt: consumedAt,
			Purpose:    purpose,
		})
	}

	boardGens, err := models.GetStoryGensByStoryAndBoard(ctx, int64(story.ID), int64(board.ID), 1)
	if err != nil {
		return nil, err
	}
	for _, gen := range boardGens {
		appendRecord(int64(gen.TokenNum), gen.UserId, firstNonZero(gen.FinishTime, gen.UpdateAt.Unix()), fmt.Sprintf("render:%s", gen.TaskType.String()))
	}

	storyGens, err := models.GetStoryGensByStory(ctx, int64(story.ID), 1)
	if err != nil {
		return nil, err
	}
	for _, gen := range storyGens {
		appendRecord(int64(gen.TokenNum), gen.UserId, firstNonZero(gen.FinishTime, gen.UpdateAt.Unix()), fmt.Sprintf("story:%s", gen.TaskType.String()))
	}

	appendRecord(story.TokenNum, story.CreatorID, story.UpdateAt.Unix(), "story:create")
	return records, nil
}

func firstNonZero(values ...int64) int64 {
	for _, v := range values {
		if v > 0 {
			return v
		}
	}
	return time.Now().Unix()
}

func filterEmpty(items []string) []string {
	result := make([]string, 0, len(items))
	for _, item := range items {
		trimmed := strings.TrimSpace(item)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}
