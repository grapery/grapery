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
	"github.com/grapery/common-protoc/gen"
	api "github.com/grapery/common-protoc/gen"
	"github.com/grapery/grapery/models"
	"github.com/grapery/grapery/pkg/asynctask"
	"github.com/grapery/grapery/pkg/cache"
	"github.com/grapery/grapery/pkg/client"
	"github.com/grapery/grapery/pkg/cloud/aliyun"
	"github.com/grapery/grapery/pkg/cloud/coze"
	"github.com/grapery/grapery/utils/convert"
	"github.com/grapery/grapery/utils/log"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// 用来管理故事的产场景描述，场景图片，场景参与角色信息
var storyboardSenceServer StoryboardSenceServer

func init() {
	storyboardSenceServer = NewStoryboardSenceService()
}

func GetStoryboardSenceServer() StoryboardSenceServer {
	if storyboardSenceServer == nil {
		storyboardSenceServer = NewStoryboardSenceService()
	}
	return storyboardSenceServer
}

func NewStoryboardSenceService() *StoryboardSenceService {
	return &StoryboardSenceService{
		bailianClient: client.NewAliyunClient(),
		doubaoClient:  client.NewDoubaoClient(),
	}
}

type StoryboardSenceService struct {
	bailianClient *client.AliyunStoryClient
	doubaoClient  *client.DoubaoClient
	cozeClient    *coze.HuoShanCozeClient
}

type GenerateStoryboardSenceBackgroundParams struct {
	BackgroundImage string `json:"background_image"`
	BackgroundColor string `json:"background_color"`
}

type GenerateStoryboardSenceBackgroundResult struct {
	BackgroundImageURL string `json:"background_image_url"`
	BackgroundColor    string `json:"background_color"`
}

type StoryboardSenceServer interface {
	GetStoryboardScene(ctx context.Context, req *api.GetStoryBoardSencesRequest) (*api.GetStoryBoardSencesResponse, error)
	CreateStoryBoardScene(ctx context.Context, req *api.CreateStoryBoardSenceRequest) (*api.CreateStoryBoardSenceResponse, error)
	UpdateStoryBoardSence(ctx context.Context, req *api.UpdateStoryBoardSenceRequest) (*api.UpdateStoryBoardSenceResponse, error)
	DeleteStoryBoardSence(ctx context.Context, req *api.DeleteStoryBoardSenceRequest) (*api.DeleteStoryBoardSenceResponse, error)
	RenderStoryBoardSence(ctx context.Context, req *api.RenderStoryBoardSenceRequest) (*api.RenderStoryBoardSenceResponse, error)
	GetStoryBoardSenceGenerate(ctx context.Context, req *api.GetStoryBoardSenceGenerateRequest) (*api.GetStoryBoardSenceGenerateResponse, error)
	RenderStoryBoardSences(ctx context.Context, req *api.RenderStoryBoardSencesRequest) (*api.RenderStoryBoardSencesResponse, error)
	GenerateStorySceneVideo(ctx context.Context, req *api.GenerateStorySceneVideoRequest) (*api.GenerateStorySceneVideoResponse, error)
	GenerateStoryboardSenceBackground(ctx context.Context, req *GenerateStoryboardSenceBackgroundParams) (*GenerateStoryboardSenceBackgroundResult, error)
}

func (s *StoryboardSenceService) GenerateStoryboardSenceBackground(ctx context.Context, req *GenerateStoryboardSenceBackgroundParams) (*GenerateStoryboardSenceBackgroundResult, error) {
	return nil, nil
}

func (s *StoryboardSenceService) GenerateStorySceneVideo(ctx context.Context, req *api.GenerateStorySceneVideoRequest) (*api.GenerateStorySceneVideoResponse, error) {
	log.Log().Info("[GenerateStorySceneVideo] 入参", zap.Any("req", req))
	story, err := models.GetStory(ctx, req.GetStoryId())
	if err != nil {
		log.Log().Error("[GenerateStorySceneVideo] 获取故事失败", zap.Error(err), zap.Int64("storyId", req.GetStoryId()))
		return nil, err
	}
	if story.CreatorID != req.GetUserId() {
		log.Log().Warn("[GenerateStorySceneVideo] 故事私有", zap.Int64("storyId", req.GetStoryId()))
		return nil, errors.New("故事私有")
	}
	storyboard, err := models.GetStoryboard(ctx, req.GetBoardId())
	if err != nil {
		log.Log().Error("[GenerateStorySceneVideo] 获取分镜失败", zap.Error(err), zap.Int64("storyboardId", req.GetBoardId()))
		return nil, err
	}
	if storyboard.CreatorID != req.GetUserId() {
		log.Log().Warn("[GenerateStorySceneVideo] 非创建者无权生成", zap.Int64("creatorId", storyboard.CreatorID), zap.Int64("userId", req.GetUserId()))
		return nil, errors.New("not allowed")
	}
	scene, err := models.GetStoryBoardScene(ctx, req.GetSenceId())
	if err != nil {
		log.Log().Error("[GenerateStorySceneVideo] 获取场景失败", zap.Error(err), zap.Int64("boardId", req.GetBoardId()))
		return nil, err
	}
	if scene.GenStatus != gen.StoryGenStatus_STORY_GEN_STATUS_FINISHED {
		log.Log().Info("[GenerateStorySceneVideo] 跳过正在生成的场景", zap.Int64("sceneId", int64(scene.ID)))
		return nil, errors.New("scene is generating")
	}
	if scene.VideoUrl != "" {
		log.Log().Info("[GenerateStorySceneVideo] 跳过已生成的场景", zap.Int64("sceneId", int64(scene.ID)))
	}

	// ===== 改为异步模式：提交任务到队列 =====

	// 获取场景图片用于参考
	var currentSceneImages []string
	if scene.GenResult != "" {
		err = json.Unmarshal([]byte(scene.GenResult), &currentSceneImages)
		if err != nil {
			log.Log().Error("[GenerateStorySceneVideo] 解析场景图片失败", zap.Error(err))
			return nil, err
		}
	}

	// 获取所有场景以确定结束图片
	scenes, err := models.GetStoryBoardScenesByBoard(ctx, req.GetBoardId())
	if err != nil {
		log.Log().Error("[GenerateStorySceneVideo] 获取场景列表失败", zap.Error(err))
		return nil, err
	}

	var startRefImage, endRefImage string
	currentSceneIndex := -1
	for i, s := range scenes {
		if s.ID == scene.ID {
			currentSceneIndex = i
			break
		}
	}

	if len(currentSceneImages) > 0 {
		startRefImage = currentSceneImages[0]
	}

	if currentSceneIndex >= 0 && currentSceneIndex+1 < len(scenes) {
		nextScene := scenes[currentSceneIndex+1]
		var nextSceneImages []string
		if nextScene.GenResult != "" {
			err = json.Unmarshal([]byte(nextScene.GenResult), &nextSceneImages)
			if err == nil && len(nextSceneImages) > 0 {
				endRefImage = nextSceneImages[0]
			}
		}
	}

	platform := "coze"
	prompt := strings.TrimSpace(req.GetPrompt())
	if prompt == "" {
		prompt = scene.Content
	}

	taskID := uuid.NewString()
	if err := models.UpdateStoryBoardSceneMultiColumn(ctx, int64(scene.ID), map[string]interface{}{
		"task_id":    taskID,
		"gen_status": gen.StoryGenStatus_STORY_GEN_STATUS_RUNNING,
	}); err != nil {
		log.Log().Error("[GenerateStorySceneVideo] 更新场景状态失败", zap.Error(err), zap.Int64("sceneId", int64(scene.ID)))
		return nil, err
	}

	now := time.Now().Unix()
	videoRecord := &models.VideoGen{
		StoryId:   int64(story.ID),
		BoardId:   int64(storyboard.ID),
		SceneId:   int64(scene.ID),
		TaskType:  gen.RenderType(api.RenderType_RENDER_TYPE_STORYSENCE),
		TaskId:    taskID,
		Prompt:    prompt,
		RefImages: strings.Join([]string{startRefImage, endRefImage}, ","),
		UserID:    req.GetUserId(),

		GenStatus: gen.StoryGenStatus_STORY_GEN_STATUS_INIT,
		Provider:  platform,
		StartTime: now,
		EndTime:   0,
		Code:      "",
		Message:   "任务已提交，正在异步处理",
		Deleted:   0,
		Tokens:    0,
		Seed:      int64(rand.Intn(10000000)),
	}
	videoID, err := models.CreateVideoGen(ctx, videoRecord)
	if err != nil {
		log.Log().Error("[GenerateStorySceneVideo] 创建视频生成记录失败", zap.Error(err))
		if errUpdate := models.UpdateStoryBoardSceneMultiColumn(ctx, int64(scene.ID), map[string]interface{}{
			"gen_status": gen.StoryGenStatus_STORY_GEN_STATUS_FINISHED,
		}); errUpdate != nil {
			log.Log().Error("[GenerateStorySceneVideo] 回滚场景状态失败", zap.Error(errUpdate))
		}
		return nil, err
	}

	metadata := map[string]interface{}{
		"story_id": int64(story.ID),
		"board_id": int64(storyboard.ID),
		"scene_id": int64(scene.ID),
		"user_id":  req.GetUserId(),
	}
	if startRefImage != "" {
		metadata["start_ref_image"] = startRefImage
	}
	if endRefImage != "" {
		metadata["end_ref_image"] = endRefImage
	}
	if len(currentSceneImages) > 0 {
		metadata["scene_images"] = currentSceneImages
	}

	info, err := asynctask.SubmitVideoGenerationTask(ctx, &asynctask.VideoGeneratePayload{
		VideoGenID:    videoID,
		TaskID:        taskID,
		UserID:        req.GetUserId(),
		Platform:      platform,
		Prompt:        prompt,
		Metadata:      metadata,
		StartRefImage: startRefImage,
		EndRefImage:   endRefImage,
		BoardID:       int64(storyboard.ID),
		SceneID:       int64(scene.ID),
		StoryID:       int64(story.ID),
	})
	if err != nil {
		log.Log().Error("[GenerateStorySceneVideo] 异步任务提交失败", zap.Error(err))
		_ = models.UpdateVideoGenFields(ctx, videoID, map[string]interface{}{
			"gen_status": gen.StoryGenStatus_STORY_GEN_STATUS_ERROR,
			"message":    err.Error(),
			"end_time":   time.Now().Unix(),
		})
		if errUpdate := models.UpdateStoryBoardSceneMultiColumn(ctx, int64(scene.ID), map[string]interface{}{
			"gen_status": gen.StoryGenStatus_STORY_GEN_STATUS_ERROR,
		}); errUpdate != nil {
			log.Log().Error("[GenerateStorySceneVideo] 更新场景失败", zap.Error(errUpdate))
		}
		return nil, err
	}

	log.Log().Info("[GenerateStorySceneVideo] 异步任务提交成功",
		zap.String("taskID", taskID),
		zap.Int64("sceneId", int64(scene.ID)),
		zap.String("queue", info.Queue))

	detail := &api.GenerateStorySceneVideoTaskDetail{
		VideoUrl:   "",
		TaskId:     taskID,
		TaskStatus: api.StoryGenStatus_STORY_GEN_STATUS_RUNNING,
	}
	return &api.GenerateStorySceneVideoResponse{
		Code:    0,
		Message: "任务已提交，正在异步处理",
		Detail:  detail,
	}, nil
}

// GenerateStorySceneVideoSync 同步生成场景视频（保留用于特殊场景）
func (s *StoryboardSenceService) GenerateStorySceneVideoSync(ctx context.Context, req *api.GenerateStorySceneVideoRequest) (*api.GenerateStorySceneVideoResponse, error) {
	log.Log().Info("[GenerateStorySceneVideoSync] 入参", zap.Any("req", req))
	story, err := models.GetStory(ctx, req.GetStoryId())
	if err != nil {
		log.Log().Error("[GenerateStorySceneVideoSync] 获取故事失败", zap.Error(err), zap.Int64("storyId", req.GetStoryId()))
		return nil, err
	}
	if story.CreatorID != req.GetUserId() {
		log.Log().Warn("[GenerateStorySceneVideoSync] 故事私有", zap.Int64("storyId", req.GetStoryId()))
		return nil, errors.New("故事私有")
	}
	storyboard, err := models.GetStoryboard(ctx, req.GetBoardId())
	if err != nil {
		log.Log().Error("[GenerateStorySceneVideoSync] 获取分镜失败", zap.Error(err), zap.Int64("storyboardId", req.GetBoardId()))
		return nil, err
	}
	if storyboard.CreatorID != req.GetUserId() {
		log.Log().Warn("[GenerateStorySceneVideoSync] 非创建者无权生成", zap.Int64("creatorId", storyboard.CreatorID), zap.Int64("userId", req.GetUserId()))
		return nil, errors.New("not allowed")
	}
	scene, err := models.GetStoryBoardScene(ctx, req.GetSenceId())
	if err != nil {
		log.Log().Error("[GenerateStorySceneVideoSync] 获取场景失败", zap.Error(err), zap.Int64("boardId", req.GetBoardId()))
		return nil, err
	}
	if scene.GenStatus != gen.StoryGenStatus_STORY_GEN_STATUS_FINISHED {
		log.Log().Info("[GenerateStorySceneVideoSync] 跳过正在生成的场景", zap.Int64("sceneId", int64(scene.ID)))
		return nil, errors.New("scene is generating")
	}
	if scene.VideoUrl != "" {
		log.Log().Info("[GenerateStorySceneVideoSync] 跳过已生成的场景", zap.Int64("sceneId", int64(scene.ID)))
	}
	videoGenPramas := coze.CozeStoryboardVideoParams{}
	videoGenPramas.Prompt = req.GetPrompt()
	videoGenPramas.Style = story.Style

	// 获取故事板的所有场景
	scenes, err := models.GetStoryBoardScenesByBoard(ctx, req.GetBoardId())
	if err != nil {
		log.Log().Error("[GenerateStorySceneVideoSync] 获取场景列表失败", zap.Error(err))
		return nil, err
	}

	// 找到当前场景在场景数组中的位置
	currentSceneIndex := -1
	for i, s := range scenes {
		if s.ID == scene.ID {
			currentSceneIndex = i
			break
		}
	}

	if currentSceneIndex == -1 {
		log.Log().Error("[GenerateStorySceneVideoSync] 未找到当前场景在故事板中的位置", zap.Int64("sceneId", int64(scene.ID)))
		return nil, errors.New("scene not found in storyboard")
	}

	// 解析当前场景的图片结果
	currentSceneImages := []string{}
	err = json.Unmarshal([]byte(scene.GenResult), &currentSceneImages)
	if err != nil {
		log.Log().Error("[GenerateStorySceneVideoSync] 解析当前场景生成结果失败", zap.Error(err))
		return nil, err
	}

	// 根据场景位置设置开始和结束图片
	if currentSceneIndex+1 < len(scenes) {
		// 有下一个场景：使用当前场景的图片作为开始图片，下一个场景的图片作为结束图片
		if len(currentSceneImages) > 0 {
			videoGenPramas.StartRefImage = currentSceneImages[0]
		}

		// 解析下一个场景的图片作为结束图片
		nextScene := scenes[currentSceneIndex+1]
		nextSceneImages := []string{}
		err = json.Unmarshal([]byte(nextScene.GenResult), &nextSceneImages)
		if err == nil && len(nextSceneImages) > 0 {
			videoGenPramas.EndRefImage = nextSceneImages[0]
		}
	} else {
		// 最后一个场景：只使用当前场景的图片作为开始图片，没有结束图片
		if len(currentSceneImages) > 0 {
			videoGenPramas.StartRefImage = currentSceneImages[0]
		}
		// EndRefImage 保持为空字符串
	}

	// 如果没有设置开始图片，使用当前场景的图片作为默认值
	if videoGenPramas.StartRefImage == "" && len(currentSceneImages) > 0 {
		videoGenPramas.StartRefImage = currentSceneImages[0]
	}

	// 记录图片选择结果
	log.Log().Info("[GenerateStorySceneVideoSync] 场景图片选择完成",
		zap.Int64("sceneId", int64(scene.ID)),
		zap.Int("sceneIndex", currentSceneIndex),
		zap.Int("totalScenes", len(scenes)),
		zap.Bool("isLastScene", currentSceneIndex+1 >= len(scenes)),
		zap.String("startRefImage", videoGenPramas.StartRefImage),
		zap.String("endRefImage", videoGenPramas.EndRefImage))
	ret, err := s.cozeClient.GenerateStoryboardVideo(ctx, videoGenPramas)
	if err != nil {
		log.Log().Error("[GenerateStorySceneVideoSync] 生成视频失败", zap.Error(err))
		return nil, err
	}
	log.Log().Info("[GenerateStorySceneVideoSync] 生成视频成功", zap.Any("result", ret))
	detail := &api.GenerateStorySceneVideoTaskDetail{
		VideoUrl:   ret,
		TaskId:     scene.TaskId,
		TaskStatus: api.StoryGenStatus_STORY_GEN_STATUS_FINISHED,
	}
	return &api.GenerateStorySceneVideoResponse{
		Code:    0,
		Message: "OK",
		Detail:  detail,
	}, nil
}

func (s *StoryboardSenceService) GetStoryboardScene(ctx context.Context, req *api.GetStoryBoardSencesRequest) (*api.GetStoryBoardSencesResponse, error) {
	// 函数入口日志
	log.Log().Info("[GetStoryboardScene] 开始获取故事板场景列表",
		zap.Int64("boardId", req.GetBoardId()))

	// 获取故事板场景列表
	log.Log().Info("[GetStoryboardScene] 开始获取故事板场景列表", zap.Int64("boardId", req.GetBoardId()))
	scenes, err := models.GetStoryBoardScenesByBoard(ctx, req.GetBoardId())
	if err != nil && err != gorm.ErrRecordNotFound {
		log.Log().Error("[GetStoryboardScene] 获取故事板场景列表失败",
			zap.Int64("boardId", req.GetBoardId()),
			zap.Error(err))
		return nil, err
	}
	log.Log().Info("[GetStoryboardScene] 成功获取故事板场景列表",
		zap.Int64("boardId", req.GetBoardId()),
		zap.Int("sceneCount", len(scenes)))

	// 检查是否有场景
	if len(scenes) == 0 {
		log.Log().Info("[GetStoryboardScene] 故事板无场景",
			zap.Int64("boardId", req.GetBoardId()))
		return &api.GetStoryBoardSencesResponse{
			Code:    0,
			Message: "no scenes",
		}, nil
	}

	// 构建API场景列表
	log.Log().Info("[GetStoryboardScene] 开始构建API场景列表",
		zap.Int64("boardId", req.GetBoardId()),
		zap.Int("sceneCount", len(scenes)))
	apiScenes := make([]*api.StoryBoardSence, 0)

	for _, scene := range scenes {

		apiScene := new(api.StoryBoardSence)
		apiScene.SenceId = int64(scene.ID)
		apiScene.Content = scene.Content
		apiScene.CharacterIds = strings.Split(scene.CharacterIds, ",")
		apiScene.CreatorId = scene.CreatorId
		apiScene.StoryId = int64(scene.StoryId)
		apiScene.BoardId = int64(scene.BoardId)
		apiScene.ImagePrompts = scene.ImagePrompts
		apiScene.AudioPrompts = scene.AudioPrompts
		apiScene.VideoPrompts = scene.VideoPrompts
		apiScene.IsGenerating = int32(scene.GenStatus)
		apiScene.ImageUrl = scene.ImageUrl
		apiScene.VideoUrl = scene.VideoUrl
		apiScene.AudioUrl = scene.AudioUrl
		apiScene.Status = int32(scene.Status)
		apiScene.Ctime = scene.CreateAt.Unix()
		apiScene.Mtime = scene.UpdateAt.Unix()
		apiScenes = append(apiScenes, apiScene)
	}

	// 返回成功响应
	log.Log().Info("[GetStoryboardScene] 获取故事板场景列表完成",
		zap.Int64("boardId", req.GetBoardId()),
		zap.Int("sceneCount", len(apiScenes)))

	return &api.GetStoryBoardSencesResponse{
		Code:    0,
		Message: "OK",
		Data: &api.GetStoryBoardSencesResponse_Data{
			List: apiScenes,
		},
	}, nil
}

func (s *StoryboardSenceService) CreateStoryBoardScene(ctx context.Context, req *api.CreateStoryBoardSenceRequest) (*api.CreateStoryBoardSenceResponse, error) {
	// 函数入口日志
	log.Log().Info("[CreateStoryBoardScene] 开始创建故事板场景",
		zap.Int64("boardId", req.Sence.GetBoardId()),
		zap.Int64("storyId", req.Sence.GetStoryId()),
		zap.Int64("creatorId", req.Sence.GetCreatorId()))
	board, err := models.GetStoryboard(ctx, req.Sence.GetBoardId())
	if board.Status == 0 {
		log.Log().Info("[CreateStoryBoardScene] 故事板已经失效",
			zap.Int64("boardId", req.Sence.GetBoardId()),
			zap.Int64("storyId", req.Sence.GetStoryId()))
		return nil, errors.New("故事板已经失效")
	}
	// 创建新场景
	log.Log().Info("[CreateStoryBoardScene] 开始创建新场景")
	newScene := new(models.StoryBoardScene)
	newScene.BoardId = req.Sence.GetBoardId()
	newScene.StoryId = req.Sence.GetStoryId()
	newScene.CreatorId = req.Sence.GetCreatorId()
	newScene.Content = req.Sence.GetContent()
	newScene.CharacterIds = strings.Join(req.Sence.GetCharacterIds(), ",")
	newScene.ImagePrompts = req.Sence.GetImagePrompts()
	newScene.AudioPrompts = req.Sence.GetAudioPrompts()
	newScene.VideoPrompts = req.Sence.GetVideoPrompts()
	newScene.Status = 1
	newScene.GenStatus = api.StoryGenStatus_STORY_GEN_STATUS_INIT
	newScene.ImageUrl = req.Sence.GetImageUrl()
	newScene.VideoUrl = req.Sence.GetVideoUrl()
	newScene.AudioUrl = req.Sence.GetAudioUrl()
	newScene.Seed = board.Seed

	log.Log().Info("[CreateStoryBoardScene] 新场景参数",
		zap.Int64("boardId", newScene.BoardId),
		zap.Int64("storyId", newScene.StoryId),
		zap.Int64("creatorId", newScene.CreatorId),
		zap.String("content", newScene.Content),
		zap.String("characterIds", newScene.CharacterIds),
		zap.Int("status", newScene.Status),
		zap.Int("genStatus", int(newScene.GenStatus)))

	_, err = models.CreateStoryBoardScene(ctx, newScene)
	if err != nil {
		log.Log().Error("[CreateStoryBoardScene] 创建故事板场景失败",
			zap.Int64("boardId", newScene.BoardId),
			zap.Int64("storyId", newScene.StoryId),
			zap.Int64("creatorId", newScene.CreatorId),
			zap.Error(err))
		return nil, err
	}
	profile := &models.UserProfile{UserId: newScene.CreatorId}
	if err := profile.IncrementContributStoryNum(ctx); err != nil {
		log.Log().Warn("[CreateStoryBoardScene] 增加用户贡献故事数失败",
			zap.Int64("userId", newScene.CreatorId),
			zap.Int64("storyId", newScene.StoryId),
			zap.Error(err))
	}

	// 记录创建成功的场景信息
	newSceneData, _ := json.Marshal(newScene)
	log.Log().Info("[CreateStoryBoardScene] 场景创建成功",
		zap.Uint("sceneId", newScene.ID),
		zap.Int64("boardId", newScene.BoardId),
		zap.String("sceneData", string(newSceneData)))
	boardCache := cache.GetStoryBoardCache()
	if cacheErr := boardCache.InvalidateStoryBoardScenes(ctx, newScene.BoardId); cacheErr != nil {
		log.Log().Warn("[CreateStoryBoardScene] 清除故事板场景缓存失败",
			zap.Int64("boardId", newScene.BoardId),
			zap.Error(cacheErr))
	}

	return &api.CreateStoryBoardSenceResponse{
		Code:    0,
		Message: "OK",
		Data: &api.CreateStoryBoardSenceResponse_Data{
			SenceId: int64(newScene.ID),
		},
	}, nil
}

func (s *StoryboardSenceService) UpdateStoryBoardSence(ctx context.Context, req *api.UpdateStoryBoardSenceRequest) (*api.UpdateStoryBoardSenceResponse, error) {
	// 函数入口日志
	log.Log().Info("[UpdateStoryBoardSence] 开始更新故事板场景",
		zap.Int64("sceneId", req.Sence.GetSenceId()),
		zap.Int64("boardId", req.Sence.GetBoardId()))

	// 获取场景
	log.Log().Info("[UpdateStoryBoardSence] 开始获取场景", zap.Int64("sceneId", req.Sence.GetSenceId()))
	scene, err := models.GetStoryBoardScene(ctx, req.Sence.GetSenceId())
	if err != nil && err != gorm.ErrRecordNotFound {
		log.Log().Error("[UpdateStoryBoardSence] 获取故事板场景失败",
			zap.Int64("sceneId", req.Sence.GetSenceId()),
			zap.Error(err))
		return nil, err
	}

	// 检查场景是否存在
	if scene == nil {
		log.Log().Warn("[UpdateStoryBoardSence] 场景不存在",
			zap.Int64("sceneId", req.Sence.GetSenceId()))
		return &api.UpdateStoryBoardSenceResponse{
			Code:    -1,
			Message: "scene not found",
		}, nil
	}
	log.Log().Info("[UpdateStoryBoardSence] 成功获取场景",
		zap.Int64("sceneId", req.Sence.GetSenceId()),
		zap.Int64("boardId", scene.BoardId),
		zap.String("content", scene.Content))

	scene.Content = req.Sence.GetContent()
	scene.ImagePrompts = req.Sence.GetImagePrompts()
	scene.AudioPrompts = req.Sence.GetAudioPrompts()
	scene.VideoPrompts = req.Sence.GetVideoPrompts()
	scene.Status = int(req.Sence.GetStatus())
	scene.GenStatus = api.StoryGenStatus(req.Sence.GetIsGenerating())
	scene.ImageUrl = req.Sence.GetImageUrl()
	scene.VideoUrl = req.Sence.GetVideoUrl()
	scene.AudioUrl = req.Sence.GetAudioUrl()

	log.Log().Info("[UpdateStoryBoardSence] 场景信息更新完成",
		zap.Int64("sceneId", req.Sence.GetSenceId()),
		zap.String("content", scene.Content),
		zap.Int("status", scene.Status),
		zap.Int("genStatus", int(scene.GenStatus)))

	err = models.UpdateStoryBoardScene(ctx, scene)
	if err != nil {
		log.Log().Error("[UpdateStoryBoardSence] 更新故事板场景失败",
			zap.Int64("sceneId", req.Sence.GetSenceId()),
			zap.Error(err))
		return nil, err
	}
	// 返回成功响应
	log.Log().Info("[UpdateStoryBoardSence] 更新故事板场景完成",
		zap.Int64("sceneId", req.Sence.GetSenceId()),
		zap.Int64("boardId", req.Sence.GetBoardId()))

	return &api.UpdateStoryBoardSenceResponse{
		Code:    0,
		Message: "OK",
	}, nil
}

func (s *StoryboardSenceService) DeleteStoryBoardSence(ctx context.Context, req *api.DeleteStoryBoardSenceRequest) (*api.DeleteStoryBoardSenceResponse, error) {
	// 函数入口日志
	log.Log().Info("[DeleteStoryBoardSence] 开始删除故事板场景",
		zap.Int64("sceneId", req.GetSenceId()),
		zap.Int64("userId", req.GetUserId()))
	if req.GetSenceId() <= 0 {
		log.Log().Warn("[DeleteStoryBoardSence] 场景ID无效",
			zap.Int64("sceneId", req.GetSenceId()))
		return &api.DeleteStoryBoardSenceResponse{
			Code:    -1,
			Message: "invalid scene id",
		}, nil
	}

	if err := models.DeleteStoryBoardScene(ctx, req.GetSenceId(), req.GetUserId()); err != nil {
		log.Log().Error("[DeleteStoryBoardSence] 删除故事板场景失败",
			zap.Int64("sceneId", req.GetSenceId()),
			zap.Error(err))
		return nil, err
	}

	boardCache := cache.GetStoryBoardCache()
	if cacheErr := boardCache.InvalidateStoryBoardScenes(ctx, req.GetBoardId()); cacheErr != nil {
		log.Log().Warn("[DeleteStoryBoardSence] 清除故事板场景缓存失败", zap.Int64("boardId", req.GetBoardId()), zap.Error(cacheErr))
	}
	log.Log().Info("[DeleteStoryBoardSence] 删除故事板场景完成", zap.Int64("sceneId", req.GetSenceId()))

	return &api.DeleteStoryBoardSenceResponse{
		Code:    0,
		Message: "OK",
	}, nil
}

// 通过生成的场景描述，生成每个场景的图片
func (s *StoryboardSenceService) RenderStoryBoardSence(ctx context.Context, req *api.RenderStoryBoardSenceRequest) (*api.RenderStoryBoardSenceResponse, error) {
	// 函数入口日志
	log.Log().Info("[RenderStoryBoardSence] 开始渲染故事板场景",
		zap.Int64("sceneId", req.GetSenceId()),
		zap.Int32("boardId", req.GetBoardId()))

	// 参数验证
	if req.GetSenceId() <= 0 {
		log.Log().Error("[RenderStoryBoardSence] 场景ID无效",
			zap.Int64("sceneId", req.GetSenceId()))
		return &api.RenderStoryBoardSenceResponse{
			Code:    -1,
			Message: "sence id is not valid",
		}, nil
	}
	if req.GetBoardId() <= 0 {
		log.Log().Error("[RenderStoryBoardSence] 故事板ID无效",
			zap.Int32("boardId", req.GetBoardId()))
		return &api.RenderStoryBoardSenceResponse{
			Code:    -1,
			Message: "board id is not valid",
		}, nil
	}

	// 获取故事板
	log.Log().Info("[RenderStoryBoardSence] 开始获取故事板", zap.Int32("boardId", req.GetBoardId()))
	board, err := models.GetStoryboard(ctx, int64(req.GetBoardId()))
	if err != nil {
		log.Log().Error("[RenderStoryBoardSence] 获取故事板失败",
			zap.Int32("boardId", req.GetBoardId()),
			zap.Error(err))
		return nil, err
	}
	if board == nil {
		log.Log().Error("[RenderStoryBoardSence] 故事板不存在",
			zap.Int32("boardId", req.GetBoardId()))
		return &api.RenderStoryBoardSenceResponse{
			Code:    -1,
			Message: "board not found",
		}, nil
	}
	log.Log().Info("[RenderStoryBoardSence] 成功获取故事板",
		zap.Int32("boardId", req.GetBoardId()),
		zap.Int64("storyId", board.StoryID))

	story, err := models.GetStory(ctx, int64(board.StoryID))
	if err != nil {
		log.Log().Error("[RenderStoryBoardSence] 获取故事失败",
			zap.Int64("storyId", board.StoryID),
			zap.Error(err))
		return nil, err
	}
	if story.Deleted {
		log.Log().Error("[RenderStoryBoardSence] 故事已删除",
			zap.Int64("storyId", board.StoryID),
			zap.Bool("deleted", story.Deleted))
		return &api.RenderStoryBoardSenceResponse{
			Code:    -1,
			Message: "story is deleted",
		}, nil
	}
	scene, err := models.GetStoryBoardScene(ctx, req.GetSenceId())
	if err != nil {
		log.Log().Error("[RenderStoryBoardSence] 获取故事板场景失败",
			zap.Int64("sceneId", req.GetSenceId()),
			zap.Error(err))
		return nil, err
	}
	if scene == nil {
		log.Log().Error("[RenderStoryBoardSence] 场景不存在",
			zap.Int64("sceneId", req.GetSenceId()))
		return &api.RenderStoryBoardSenceResponse{
			Code:    -1,
			Message: "scene not found",
		}, nil
	}
	if scene.Deleted {
		log.Log().Error("[RenderStoryBoardSence] 场景已删除",
			zap.Int64("sceneId", req.GetSenceId()),
			zap.Bool("deleted", scene.Deleted))
		return &api.RenderStoryBoardSenceResponse{
			Code:    -1,
			Message: "scene is deleted",
		}, nil
	}

	scene.GenStatus = api.StoryGenStatus_STORY_GEN_STATUS_INIT
	err = models.UpdateStoryBoardScene(ctx, scene)
	if err != nil {
		log.Log().Error("[RenderStoryBoardSence] 更新场景状态失败",
			zap.Int64("sceneId", req.GetSenceId()),
			zap.Error(err))
		return nil, err
	}

	// 生成指定场景的图片
	log.Log().Info("[RenderStoryBoardSence] 开始生成场景图片",
		zap.Int64("sceneId", req.GetSenceId()))
	templatePrompt := scene.ImagePrompts
	storyboardImagePrompt := fmt.Sprintf("%s,风格为：%s", templatePrompt, story.Style)
	templatePrompt = storyboardImagePrompt + ",人物: " + scene.CharacterIds

	log.Log().Info("[RenderStoryBoardSence] 图片生成参数",
		zap.Int64("sceneId", req.GetSenceId()),
		zap.String("originalPrompt", scene.ImagePrompts),
		zap.String("templatePrompt", templatePrompt),
		zap.String("characterIds", scene.CharacterIds))

	renderStoryParams := &client.GenStoryImagesParams{
		Content: templatePrompt,
		Seed:    int64(board.Seed),
	}

	// 调用图片生成服务
	log.Log().Info("[RenderStoryBoardSence] 开始调用图片生成服务",
		zap.Int64("sceneId", req.GetSenceId()),
		zap.String("content", scene.Content),
		zap.String("prompt", templatePrompt))

	start := time.Now()
	ret, err := s.doubaoClient.GenStoryBoardImage(ctx, renderStoryParams)
	if err != nil {
		log.Log().Error("[RenderStoryBoardSence] 图片生成失败",
			zap.Int64("sceneId", req.GetSenceId()),
			zap.Error(err))
		return nil, err
	}

	genTime := time.Since(start)
	log.Log().Info("[RenderStoryBoardSence] 图片生成成功",
		zap.Int64("sceneId", req.GetSenceId()),
		zap.Duration("genTime", genTime),
		zap.Int("imageCount", len(ret.ImageUrls)))

	aliyunUrls := make([]string, 0)
	for i, imageUrl := range ret.ImageUrls {
		log.Log().Info("[RenderStoryBoardSence] 开始上传单张图片",
			zap.Int64("sceneId", req.GetSenceId()),
			zap.Int("imageIndex", i),
			zap.String("imageUrl", imageUrl))

		aliyunClient := aliyun.GetGlobalClient()
		aliyunUrl, err := aliyunClient.UploadFileFromURL("", imageUrl)
		if err != nil {
			log.Log().Error("[RenderStoryBoardSence] 上传图片失败",
				zap.Int64("sceneId", req.GetSenceId()),
				zap.Int("imageIndex", i),
				zap.String("imageUrl", imageUrl),
				zap.Error(err))
			continue
		}
		aliyunUrls = append(aliyunUrls, aliyunUrl)
	}

	retData, _ := json.Marshal(aliyunUrls)
	scene.GenResult = string(retData)
	scene.Status = 1
	scene.GenStatus = gen.StoryGenStatus_STORY_GEN_STATUS_FINISHED

	err = models.UpdateStoryBoardScene(ctx, scene)
	if err != nil {
		log.Log().Error("[RenderStoryBoardSence] 更新场景生成结果失败",
			zap.Int64("sceneId", req.GetSenceId()),
			zap.Error(err))
		return nil, err
	}
	log.Log().Info("[RenderStoryBoardSence] 成功更新场景生成结果",
		zap.Int64("sceneId", req.GetSenceId()),
		zap.String("genResult", scene.GenResult))

	return &api.RenderStoryBoardSenceResponse{
		Code:    0,
		Message: "OK",
		Data:    convert.ConvertStoryBoardSceneToApiStoryBoardScene(scene),
	}, nil
}

func (s *StoryboardSenceService) RenderStoryBoardSences(ctx context.Context, req *api.RenderStoryBoardSencesRequest) (*api.RenderStoryBoardSencesResponse, error) {
	// 函数入口日志
	log.Log().Info("[RenderStoryBoardSences] 开始渲染故事板场景列表",
		zap.Int32("boardId", req.GetBoardId()))

	// 参数校验
	if req.GetBoardId() <= 0 {
		log.Log().Error("[RenderStoryBoardSences] 故事板ID无效",
			zap.Int32("boardId", req.GetBoardId()))
		return &api.RenderStoryBoardSencesResponse{
			Code:    -1,
			Message: "board id is 0",
		}, nil
	}

	// 获取故事板
	log.Log().Info("[RenderStoryBoardSences] 开始获取故事板", zap.Int32("boardId", req.GetBoardId()))
	board, err := models.GetStoryboard(ctx, int64(req.GetBoardId()))
	if err != nil {
		log.Log().Error("[RenderStoryBoardSences] 获取故事板失败",
			zap.Int32("boardId", req.GetBoardId()),
			zap.Error(err))
		return nil, err
	}
	if board == nil {
		log.Log().Error("[RenderStoryBoardSences] 故事板不存在",
			zap.Int32("boardId", req.GetBoardId()))
		return &api.RenderStoryBoardSencesResponse{
			Code:    -1,
			Message: "board not found",
		}, nil
	}

	// 获取故事
	log.Log().Info("[RenderStoryBoardSences] 开始获取故事", zap.Int64("storyId", board.StoryID))
	story, err := models.GetStory(ctx, int64(board.StoryID))
	if err != nil {
		log.Log().Error("[RenderStoryBoardSences] 获取故事失败",
			zap.Int64("storyId", board.StoryID),
			zap.Error(err))
		return nil, err
	}
	if story.Deleted {
		log.Log().Error("[RenderStoryBoardSences] 故事已删除",
			zap.Int64("storyId", board.StoryID),
			zap.Bool("deleted", story.Deleted))
		return &api.RenderStoryBoardSencesResponse{
			Code:    -1,
			Message: "story is deleted",
		}, nil
	}

	// 获取所有场景
	log.Log().Info("[RenderStoryBoardSences] 开始获取所有场景", zap.Int32("boardId", req.GetBoardId()))
	scenes, err := models.GetStoryBoardScenesByBoard(ctx, int64(req.GetBoardId()))
	if err != nil {
		log.Log().Error("[RenderStoryBoardSences] 获取故事板场景失败",
			zap.Int32("boardId", req.GetBoardId()),
			zap.Error(err))
		return nil, err
	}
	if len(scenes) == 0 {
		log.Log().Error("[RenderStoryBoardSences] 场景不存在",
			zap.Int32("boardId", req.GetBoardId()))
		return &api.RenderStoryBoardSencesResponse{
			Code:    -1,
			Message: "scene not found",
		}, nil
	}

	// 检查场景状态
	for i, scene := range scenes {

		if scene.Deleted {
			log.Log().Error("[RenderStoryBoardSences] 场景已删除",
				zap.Int32("boardId", req.GetBoardId()),
				zap.Int("sceneIndex", i),
				zap.Uint("sceneId", scene.ID))
			return &api.RenderStoryBoardSencesResponse{
				Code:    -1,
				Message: "scene is deleted",
			}, nil
		}
		if scene.GenStatus != gen.StoryGenStatus_STORY_GEN_STATUS_FINISHED {
			log.Log().Error("[RenderStoryBoardSences] 场景未生成完成",
				zap.Int32("boardId", req.GetBoardId()),
				zap.Int("sceneIndex", i),
				zap.Uint("sceneId", scene.ID))
			return &api.RenderStoryBoardSencesResponse{
				Code:    0,
				Message: "scene is generating",
			}, nil
		}
	}

	apiScenes := make([]*api.StoryBoardSence, 0)

	// 顺序遍历每个场景，依次发起图片生成任务
	for i, scene := range scenes {
		log.Log().Info("[RenderStoryBoardSences] 开始处理场景",
			zap.Int32("boardId", req.GetBoardId()),
			zap.Int("sceneIndex", i),
			zap.Uint("sceneId", scene.ID),
			zap.String("content", scene.Content))

		// 生成图片prompt
		templatePrompt := scene.ImagePrompts + ",人物角色: " + scene.CharacterIds

		log.Log().Info("[RenderStoryBoardSences] 图片生成参数",
			zap.Int32("boardId", req.GetBoardId()),
			zap.Int("sceneIndex", i),
			zap.Uint("sceneId", scene.ID),
			zap.String("originalPrompt", scene.ImagePrompts),
			zap.String("templatePrompt", templatePrompt),
			zap.String("characterIds", scene.CharacterIds))

		// 调用GenStoryBoardImages，获取task_id
		renderStoryParams := &client.GenStoryImagesParams{
			Content: templatePrompt,
			Seed:    int64(board.Seed),
		}

		log.Log().Info("[RenderStoryBoardSences] 开始调用图片生成服务",
			zap.Int32("boardId", req.GetBoardId()),
			zap.Int("sceneIndex", i),
			zap.Uint("sceneId", scene.ID),
			zap.String("content", scene.Content),
			zap.String("prompt", templatePrompt))

		start := time.Now()
		ret, err := s.doubaoClient.GenStoryBoardImage(ctx, renderStoryParams)
		if err != nil {
			log.Log().Error("[RenderStoryBoardSences] 图片生成失败",
				zap.Int32("boardId", req.GetBoardId()),
				zap.Int("sceneIndex", i),
				zap.Uint("sceneId", scene.ID),
				zap.Error(err))
			return nil, err
		}

		genTime := time.Since(start)
		log.Log().Info("[RenderStoryBoardSences] 图片生成成功",
			zap.Int32("boardId", req.GetBoardId()),
			zap.Int("sceneIndex", i),
			zap.Uint("sceneId", scene.ID),
			zap.Duration("genTime", genTime),
			zap.Int("imageCount", len(ret.ImageUrls)))

		// 上传图片到阿里云

		aliyunUrls := make([]string, 0)
		for j, imageUrl := range ret.ImageUrls {

			aliyunClient := aliyun.GetGlobalClient()
			aliyunUrl, err := aliyunClient.UploadFileFromURL("", imageUrl)
			if err != nil {
				log.Log().Error("[RenderStoryBoardSences] 上传图片失败",
					zap.Int32("boardId", req.GetBoardId()),
					zap.Int("sceneIndex", i),
					zap.Uint("sceneId", scene.ID),
					zap.Int("imageIndex", j),
					zap.String("imageUrl", imageUrl),
					zap.Error(err))
				continue
			}
			aliyunUrls = append(aliyunUrls, aliyunUrl)
		}

		retData, _ := json.Marshal(aliyunUrls)
		scene.GenResult = string(retData)
		scene.Status = 1
		scene.GenStatus = gen.StoryGenStatus_STORY_GEN_STATUS_FINISHED

		err = models.UpdateStoryBoardScene(ctx, scene)
		if err != nil {
			log.Log().Error("[RenderStoryBoardSences] 更新场景生成结果失败",
				zap.Int32("boardId", req.GetBoardId()),
				zap.Int("sceneIndex", i),
				zap.Uint("sceneId", scene.ID),
				zap.Error(err))
			return nil, err
		}

		apiScenes = append(apiScenes, convert.ConvertStoryBoardSceneToApiStoryBoardScene(scene))
	}

	// 返回成功响应
	log.Log().Info("[RenderStoryBoardSences] 故事板场景列表渲染完成",
		zap.Int32("boardId", req.GetBoardId()),
		zap.Int("sceneCount", len(apiScenes)))

	return &api.RenderStoryBoardSencesResponse{
		Code:    0,
		Message: "OK",
		List:    apiScenes,
	}, nil
}

func (s *StoryboardSenceService) GetStoryBoardSenceGenerate(ctx context.Context, req *api.GetStoryBoardSenceGenerateRequest) (*api.GetStoryBoardSenceGenerateResponse, error) {
	// 函数入口日志
	log.Log().Info("[GetStoryBoardSenceGenerate] 开始获取故事板场景生成状态",
		zap.Int64("sceneId", req.GetSenceId()))

	// 获取场景描述
	log.Log().Info("[GetStoryBoardSenceGenerate] 开始获取场景描述", zap.Int64("sceneId", req.GetSenceId()))
	scene, err := models.GetStoryBoardScene(ctx, req.GetSenceId())
	if err != nil {
		log.Log().Error("[GetStoryBoardSenceGenerate] 获取故事板场景失败",
			zap.Int64("sceneId", req.GetSenceId()),
			zap.Error(err))
		return nil, err
	}
	if scene == nil {
		log.Log().Error("[GetStoryBoardSenceGenerate] 场景不存在",
			zap.Int64("sceneId", req.GetSenceId()))
		return &api.GetStoryBoardSenceGenerateResponse{
			Code:    -1,
			Message: "scene not found",
		}, nil
	}

	// 检查场景状态
	if scene.GenStatus == gen.StoryGenStatus_STORY_GEN_STATUS_RUNNING {
		log.Log().Info("[GetStoryBoardSenceGenerate] 场景正在生成中",
			zap.Int64("sceneId", req.GetSenceId()),
			zap.Int("status", scene.Status))
		return &api.GetStoryBoardSenceGenerateResponse{
			Code:    0,
			Message: "scene is already generating",
		}, nil
	}
	if scene.GenStatus == gen.StoryGenStatus_STORY_GEN_STATUS_ERROR {
		log.Log().Error("[GetStoryBoardSenceGenerate] 场景已删除",
			zap.Int64("sceneId", req.GetSenceId()),
			zap.Int("status", scene.Status))
		return &api.GetStoryBoardSenceGenerateResponse{
			Code:    -1,
			Message: "scene is deleted",
		}, nil
	}

	// 返回成功响应
	log.Log().Info("[GetStoryBoardSenceGenerate] 获取故事板场景生成状态完成",
		zap.Int64("sceneId", req.GetSenceId()),
		zap.Int("status", scene.Status),
		zap.Int("genStatus", int(scene.GenStatus)))

	return &api.GetStoryBoardSenceGenerateResponse{
		Code:    0,
		Message: "OK",
		Data:    convert.ConvertStoryBoardSceneToApiStoryBoardScene(scene),
	}, nil
}
