package story

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	api "github.com/grapery/common-protoc/gen"
	"github.com/grapery/grapery/models"
	"github.com/grapery/grapery/pkg/active"
	"github.com/grapery/grapery/pkg/client"
	"github.com/grapery/grapery/utils"
	"github.com/grapery/grapery/utils/compliance"
	"github.com/grapery/grapery/utils/convert"
	"github.com/grapery/grapery/utils/log"
	"github.com/grapery/grapery/utils/prompt"
)

var storyServer StoryServer

func init() {
	storyServer = NewStoryService()
}

func GetStoryServer() StoryServer {
	return storyServer
}

type StoryServer interface {
	CreateStory(ctx context.Context, req *api.CreateStoryRequest) (resp *api.CreateStoryResponse, err error)
	GetStory(ctx context.Context, req *api.GetStoryInfoRequest) (resp *api.GetStoryInfoResponse, err error)
	UpdateStory(ctx context.Context, req *api.UpdateStoryRequest) (resp *api.UpdateStoryResponse, err error)
	WatchStory(ctx context.Context, req *api.WatchStoryRequest) (resp *api.WatchStoryResponse, err error)
	CreateStoryboard(ctx context.Context, req *api.CreateStoryboardRequest) (resp *api.CreateStoryboardResponse, err error)
	GetStoryboard(ctx context.Context, req *api.GetStoryboardRequest) (resp *api.GetStoryboardResponse, err error)
	UpdateStoryboard(ctx context.Context, req *api.UpdateStoryboardRequest) (resp *api.UpdateStoryboardResponse, err error)
	GetStoryboards(ctx context.Context, req *api.GetStoryboardsRequest) (resp *api.GetStoryboardsResponse, err error)
	DelStoryboard(ctx context.Context, req *api.DelStoryboardRequest) (resp *api.DelStoryboardResponse, err error)
	ForkStoryboard(ctx context.Context, req *api.ForkStoryboardRequest) (resp *api.ForkStoryboardResponse, err error)
	LikeStoryboard(ctx context.Context, req *api.LikeStoryboardRequest) (resp *api.LikeStoryboardResponse, err error)
	ShareStoryboard(ctx context.Context, req *api.ShareStoryboardRequest) (resp *api.ShareStoryboardResponse, err error)
	LikeStory(ctx context.Context, req *api.LikeStoryRequest) (resp *api.LikeStoryResponse, err error)
	UnLikeStory(ctx context.Context, req *api.UnLikeStoryRequest) (resp *api.UnLikeStoryResponse, err error)
	UnLikeStoryboard(ctx context.Context, req *api.UnLikeStoryboardRequest) (resp *api.UnLikeStoryboardResponse, err error)

	RenderStory(ctx context.Context, req *api.RenderStoryRequest) (*api.RenderStoryResponse, error)
	RenderStoryboard(ctx context.Context, req *api.RenderStoryboardRequest) (*api.RenderStoryboardResponse, error)
	GenStoryboardImages(ctx context.Context, req *api.GenStoryboardImagesRequest) (*api.GenStoryboardImagesResponse, error)
	GenStoryboardText(ctx context.Context, req *api.GenStoryboardTextRequest) (*api.GenStoryboardTextResponse, error)
	GetStoryRender(ctx context.Context, req *api.GetStoryRenderRequest) (*api.GetStoryRenderResponse, error)
	GetStoryBoardRender(ctx context.Context, req *api.GetStoryBoardRenderRequest) (*api.GetStoryBoardRenderResponse, error)

	ContinueRenderStory(ctx context.Context, req *api.ContinueRenderStoryRequest) (*api.ContinueRenderStoryResponse, error)

	GetStoryboardScene(ctx context.Context, req *api.GetStoryBoardSencesRequest) (*api.GetStoryBoardSencesResponse, error)
	CreateStoryBoardScene(ctx context.Context, req *api.CreateStoryBoardSenceRequest) (*api.CreateStoryBoardSenceResponse, error)
	UpdateStoryBoardSence(ctx context.Context, req *api.UpdateStoryBoardSenceRequest) (*api.UpdateStoryBoardSenceResponse, error)
	DeleteStoryBoardSence(ctx context.Context, req *api.DeleteStoryBoardSenceRequest) (*api.DeleteStoryBoardSenceResponse, error)
	RenderStoryBoardSence(ctx context.Context, req *api.RenderStoryBoardSenceRequest) (*api.RenderStoryBoardSenceResponse, error)
	GetStoryBoardSenceGenerate(ctx context.Context, req *api.GetStoryBoardSenceGenerateRequest) (*api.GetStoryBoardSenceGenerateResponse, error)
	GetStoryBoardGenerate(ctx context.Context, req *api.GetStoryBoardGenerateRequest) (*api.GetStoryBoardGenerateResponse, error)
	RenderStoryBoardSences(ctx context.Context, req *api.RenderStoryBoardSencesRequest) (*api.RenderStoryBoardSencesResponse, error)

	LikeStoryRole(ctx context.Context, req *api.LikeStoryRoleRequest) (*api.LikeStoryRoleResponse, error)
	UnLikeStoryRole(ctx context.Context, req *api.UnLikeStoryRoleRequest) (*api.UnLikeStoryRoleResponse, error)
	FollowStoryRole(ctx context.Context, req *api.FollowStoryRoleRequest) (*api.FollowStoryRoleResponse, error)
	UnFollowStoryRole(ctx context.Context, req *api.UnFollowStoryRoleRequest) (*api.UnFollowStoryRoleResponse, error)
	SearchRoles(ctx context.Context, req *api.SearchRolesRequest) (*api.SearchRolesResponse, error)
	RestoreStoryboard(ctx context.Context, req *api.RestoreStoryboardRequest) (*api.RestoreStoryboardResponse, error)
	SearchStories(ctx context.Context, req *api.SearchStoriesRequest) (*api.SearchStoriesResponse, error)
	GetUserCreatedStoryboards(ctx context.Context, req *api.GetUserCreatedStoryboardsRequest) (*api.GetUserCreatedStoryboardsResponse, error)
	GetUserCreatedRoles(ctx context.Context, req *api.GetUserCreatedRolesRequest) (*api.GetUserCreatedRolesResponse, error)

	RenderStoryRoles(ctx context.Context, req *api.RenderStoryRolesRequest) (*api.RenderStoryRolesResponse, error)
	UpdateStoryRole(ctx context.Context, req *api.UpdateStoryRoleRequest) (*api.UpdateStoryRoleResponse, error)
	RenderStoryRoleDetail(ctx context.Context, req *api.RenderStoryRoleDetailRequest) (*api.RenderStoryRoleDetailResponse, error)
	GetStoryRoles(ctx context.Context, req *api.GetStoryRolesRequest) (*api.GetStoryRolesResponse, error)
	GetStoryBoardRoles(ctx context.Context, req *api.GetStoryBoardRolesRequest) (*api.GetStoryBoardRolesResponse, error)
	GetStoryContributors(ctx context.Context, req *api.GetStoryContributorsRequest) (*api.GetStoryContributorsResponse, error)
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
	GetNextStoryboard(ctx context.Context, req *api.GetNextStoryboardRequest) (*api.GetNextStoryboardResponse, error)
	RenderStoryRoleContinuously(ctx context.Context, req *api.RenderStoryRoleContinuouslyRequest) (*api.RenderStoryRoleContinuouslyResponse, error)

	CancelStoryboard(ctx context.Context, req *api.CancelStoryboardRequest) (*api.CancelStoryboardResponse, error)
	PublishStoryboard(ctx context.Context, req *api.PublishStoryboardRequest) (*api.PublishStoryboardResponse, error)

	GetUserWatchStoryActiveStoryBoards(ctx context.Context, req *api.GetUserWatchStoryActiveStoryBoardsRequest) (*api.GetUserWatchStoryActiveStoryBoardsResponse, error)
	GetUserWatchRoleActiveStoryBoards(ctx context.Context, req *api.GetUserWatchRoleActiveStoryBoardsRequest) (*api.GetUserWatchRoleActiveStoryBoardsResponse, error)
	GetUnPublishStoryboard(ctx context.Context, req *api.GetUnPublishStoryboardRequest) (*api.GetUnPublishStoryboardResponse, error)

	GenerateRoleDescription(ctx context.Context, req *api.GenerateRoleDescriptionRequest) (*api.GenerateRoleDescriptionResponse, error)
	UpdateRoleDescription(ctx context.Context, req *api.UpdateRoleDescriptionRequest) (*api.UpdateRoleDescriptionResponse, error)
	GenerateRolePrompt(ctx context.Context, req *api.GenerateRolePromptRequest) (*api.GenerateRolePromptResponse, error)
	UpdateRolePrompt(ctx context.Context, req *api.UpdateRolePromptRequest) (*api.UpdateRolePromptResponse, error)
	UpdateStoryRoleAvator(ctx context.Context, req *api.UpdateStoryRoleAvatorRequest) (*api.UpdateStoryRoleAvatorResponse, error)
	GetStoryRoleList(ctx context.Context, req *api.GetStoryRoleListRequest) (*api.GetStoryRoleListResponse, error)

	TrendingStory(ctx context.Context, req *api.TrendingStoryRequest) (*api.TrendingStoryResponse, error)
	TrendingStoryRole(ctx context.Context, req *api.TrendingStoryRoleRequest) (*api.TrendingStoryRoleResponse, error)

	UpdateStoryRolePoster(ctx context.Context, req *api.UpdateStoryRolePosterRequest) (*api.UpdateStoryRolePosterResponse, error)
	GenerateStoryRolePoster(ctx context.Context, req *api.GenerateStoryRolePosterRequest) (*api.GenerateStoryRolePosterResponse, error)

	UpdateStoryRoleDescriptionDetail(ctx context.Context, req *api.UpdateStoryRoleDescriptionDetailRequest) (*api.UpdateStoryRoleDescriptionDetailResponse, error)
	UpdateStoryRolePrompt(ctx context.Context, req *api.UpdateStoryRolePromptRequest) (*api.UpdateStoryRolePromptResponse, error)
}

type StoryService struct {
	client *client.StoryClient
}

func NewStoryService() *StoryService {
	return &StoryService{
		client: client.NewStoryClient(
			client.PlatformZhipu,
		),
	}
}

func ConvertStoryToApiStory(story *models.Story) *api.Story {
	item := &api.Story{
		Id:        int64(story.ID),
		Name:      story.Title,
		Origin:    story.Origin,
		Avatar:    story.Avatar,
		Desc:      story.ShortDesc,
		CreatorId: story.CreatorID,
		GroupId:   story.GroupID,
		Status:    int32(story.Status),
		IsAiGen:   story.AIGen,
		IsClose:   story.IsClose,
		Ctime:     story.CreateAt.Unix(),
		Mtime:     story.UpdateAt.Unix(),
	}
	fmt.Print("item: ", item.String())
	_ = json.Unmarshal([]byte(story.Params), &item.Params)
	return item
}

func ConvertApiStoryToStory(apiStory *api.Story) *models.Story {
	item := &models.Story{
		Title:       apiStory.Name,
		Name:        apiStory.Name,
		ShortDesc:   apiStory.Desc,
		CreatorID:   apiStory.CreatorId,
		OwnerID:     apiStory.CreatorId,
		GroupID:     apiStory.GroupId,
		Origin:      apiStory.Origin,
		RootBoardID: int(apiStory.RootBoardId),
		AIGen:       apiStory.IsAiGen,
		Avatar:      apiStory.Avatar,
		Status:      models.StoryStatus(apiStory.Status),
	}
	params, _ := json.Marshal(apiStory.Params)
	item.Params = string(params)
	return item
}

func (s *StoryService) CreateStory(ctx context.Context, req *api.CreateStoryRequest) (resp *api.CreateStoryResponse, err error) {
	err = compliance.GetComplianceTool().TextCompliance(req.GetShortDesc())
	if err != nil {
		return nil, err
	}
	if !req.GetIsAiGen() {
		log.Log().Info("not AI gen story task")
		return nil, fmt.Errorf("not AI gen story task")
	}
	if req.GetParams().Background != "" {
		err = compliance.GetComplianceTool().TextCompliance(req.GetParams().Background)
		if err != nil {
			return nil, err
		}
	} else {
		req.Params.Background = req.Origin
	}
	if req.GetParams().StoryDescription != "" {
		err = compliance.GetComplianceTool().TextCompliance(req.GetParams().StoryDescription)
		if err != nil {
			return nil, err
		}
	}
	if req.GetParams().NegativePrompt != "" {
		err = compliance.GetComplianceTool().TextCompliance(req.GetParams().NegativePrompt)
		if err != nil {
			return nil, err
		}
	} else {
		req.Params.NegativePrompt = models.NegativePrompt
	}
	group := &models.Group{}
	group.ID = uint(req.GroupId)
	err = group.GetByID()
	if err != nil {
		log.Log().Error("get group by id failed", zap.Error(err))
		return nil, err
	}
	params, _ := json.Marshal(req.Params)
	newStory := &models.Story{
		Title:       req.Title,
		ShortDesc:   req.ShortDesc,
		Origin:      req.Origin,
		Status:      models.StoryStatus(req.Status),
		RootBoardID: 0,
		GroupID:     req.GroupId,
		AIGen:       req.GetIsAiGen(),
		CreatorID:   req.CreatorId,
		Params:      string(params),
		FollowCount: 1,
		LikeCount:   1,
	}
	storyId, err := models.CreateStory(ctx, newStory)
	if err != nil {
		log.Log().Error("create story failed")
		return &api.CreateStoryResponse{
			Code:    -1,
			Message: fmt.Sprintf("create story failed: %s", err.Error()),
			Data:    nil,
		}, nil
	}
	// 更新当前小组的故事数量
	err = models.IncGroupProfileStoryCount(ctx, int64(group.ID))
	if err != nil {
		log.Log().Error("inc group profile story count failed", zap.Error(err))
	}
	userProfile := &models.UserProfile{
		UserId: req.CreatorId,
	}
	err = userProfile.IncrementCreatedStoryNum()
	if err != nil {
		log.Log().Error("increment created story num failed", zap.Error(err))
	}
	err = userProfile.IncrementWatchingStoryNum()
	if err != nil {
		log.Log().Error("increment watching story num failed", zap.Error(err))
	}
	err = models.CreateWatchStoryItem(ctx, int(req.CreatorId), int64(storyId), int64(group.ID))
	if err != nil {
		log.Log().Error("watch story failed", zap.Error(err))
	}
	newStory.ID = uint(storyId)
	active.GetActiveServer().WriteStoryActive(ctx, group, newStory, nil, nil, req.GetCreatorId(), api.ActiveType_NewStory)
	return &api.CreateStoryResponse{
		Code:    0,
		Message: "create story success",
		Data: &api.CreateStoryResponse_Data{
			StoryId: int32(storyId),
		},
	}, nil
}

func (s *StoryService) GetStory(ctx context.Context, req *api.GetStoryInfoRequest) (resp *api.GetStoryInfoResponse, err error) {
	storyInfo, err := models.GetStory(ctx, req.StoryId)
	if err != nil {
		return nil, err
	}
	cu, err := s.GetStoryCurrentUserStatus(ctx, req.StoryId)
	if err != nil {
		log.Log().Error("get story current user status failed", zap.Error(err))
	}
	creator, err := models.GetUserById(ctx, storyInfo.CreatorID)
	if err != nil {
		log.Log().Error("get story creator failed", zap.Error(err))
	}
	info := ConvertStoryToApiStory(storyInfo)
	info.CurrentUserStatus = cu
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

	needUpdateData := make(map[string]interface{})
	if req.GetIsAchieve() {
		needUpdateData["is_achieve"] = req.IsAchieve
	}
	if req.GetShortDesc() != "" {
		needUpdateData["short_desc"] = req.ShortDesc
	}
	if req.GetOrigin() != "" {
		needUpdateData["origin"] = req.Origin
	}
	if req.GetStatus() != 0 {
		needUpdateData["status"] = req.Status
	}
	if req.GetParams() != nil {
		needUpdateData["params"] = req.Params
	}
	if req.GetIsAiGen() {
		needUpdateData["aigen"] = req.IsAiGen
	}
	if req.GetIsClose() {
		needUpdateData["is_close"] = req.IsClose
	}

	if len(needUpdateData) == 0 {
		return &api.UpdateStoryResponse{}, nil
	}
	err = models.UpdateStorySpecColumns(ctx, req.StoryId, needUpdateData)
	if err != nil {
		return nil, err
	}

	return &api.UpdateStoryResponse{
		Code:    0,
		Message: "update story success",
		Data: &api.UpdateStoryResponse_Data{
			StoryId: int32(req.StoryId),
		},
	}, nil
}

func (s *StoryService) WatchStory(ctx context.Context, req *api.WatchStoryRequest) (resp *api.WatchStoryResponse, err error) {

	storyInfo, err := models.GetStory(ctx, req.StoryId)
	if err != nil {
		return nil, err
	}
	if storyInfo.Status == -1 {
		return &api.WatchStoryResponse{}, nil
	}
	storyInfo.FollowCount += 1
	err = models.UpdateStorySpecColumns(ctx, req.StoryId, map[string]interface{}{
		"follow_count": storyInfo.FollowCount,
	})
	if err != nil {
		return nil, err
	}
	watchInfo, err := models.GetWatchItemByStoryAndUser(ctx, req.GetStoryId(), int(req.GetUserId()))
	if err != nil {
		return nil, err
	}
	// 如果已经关注，不再重复关注
	if watchInfo != nil {
		return &api.WatchStoryResponse{
			Code:    0,
			Message: "OK",
		}, nil
	} else {
		err = models.CreateWatchStoryItem(ctx, int(req.GetUserId()), req.GetStoryId(), 0)
		if err != nil {
			return nil, err
		}
	}
	userProfile := &models.UserProfile{
		UserId: req.GetUserId(),
	}
	err = userProfile.IncrementWatchingStoryNum()
	if err != nil {
		log.Log().Error("increment watching story num failed", zap.Error(err))
	}

	return &api.WatchStoryResponse{
		Code:    0,
		Message: "OK",
	}, nil
}

func ConvertApiStoryBoardToStoryBoard(apiStoryBoard *api.StoryBoard) *models.StoryBoard {
	board := &models.StoryBoard{
		StoryID:     apiStoryBoard.StoryId,
		CreatorID:   apiStoryBoard.Creator,
		PrevId:      apiStoryBoard.PrevBoardId,
		Title:       apiStoryBoard.Title,
		Description: apiStoryBoard.Content,
		IsAiGen:     true,
	}
	params, _ := json.Marshal(apiStoryBoard.Params)
	board.Params = string(params)
	return board
}

func ConvertStoryBoardToApiStoryBoard(storyBoard *models.StoryBoard) *api.StoryBoard {
	ret := &api.StoryBoard{
		StoryId:      storyBoard.StoryID,
		StoryBoardId: int64(storyBoard.ID),
		Creator:      storyBoard.CreatorID,
		Title:        storyBoard.Title,
		Content:      storyBoard.Description,
		PrevBoardId:  storyBoard.PrevId,
		IsAiGen:      storyBoard.IsAiGen,
		Ctime:        storyBoard.CreateAt.Unix(),
		Mtime:        storyBoard.UpdateAt.Unix(),
	}
	_ = json.Unmarshal([]byte(storyBoard.Params), &ret.Params)
	return ret
}

func ConvertStorySceneToApiScene(scene *models.StoryBoardScene) *api.StoryBoardSence {
	ret := &api.StoryBoardSence{
		SenceId:      int64(scene.ID),
		Content:      scene.Content,
		CharacterIds: strings.Split(scene.CharacterIds, ","),
		CreatorId:    scene.CreatorId,
		StoryId:      int64(scene.StoryId),
		BoardId:      int64(scene.BoardId),
		ImagePrompts: scene.ImagePrompts,
		AudioPrompts: scene.AudioPrompts,
		VideoPrompts: scene.VideoPrompts,
		IsGenerating: int32(scene.IsGenerating),
		GenResult:    scene.GenResult,
		Status:       int32(scene.Status),
		Ctime:        scene.CreateAt.Unix(),
		Mtime:        scene.UpdateAt.Unix(),
	}
	return ret
}

func (s *StoryService) RenderStory(ctx context.Context, req *api.RenderStoryRequest) (*api.RenderStoryResponse, error) {
	story, err := models.GetStory(ctx, req.StoryId)
	if err != nil {
		log.Log().Error("get story failed", zap.Error(err))
		return nil, err
	}
	if story.Status == -1 {
		log.Log().Info("story is closed")
		return &api.RenderStoryResponse{
			Code:    -1,
			Message: "story is closed",
		}, nil
	}

	genParams := new(models.StoryParams)
	if story.Params == "" {
		log.Log().Error("story params is empty")
		return &api.RenderStoryResponse{
			Code:    -1,
			Message: "story params is empty",
		}, nil
	}
	err = json.Unmarshal([]byte(story.Params), &genParams)
	if err != nil {
		log.Log().Error("unmarshal story gen params failed", zap.Error(err))
		return nil, err
	}
	storyGen := new(models.StoryGen)
	storyGen.Uuid = uuid.New().String()
	storyGenData, _ := json.Marshal(genParams)
	storyGen.LLmPlatform = "Zhipu"
	storyGen.NegativePrompt = prompt.ZhipuNegativePrompt
	storyGen.PositivePrompt = prompt.ZhipuPositivePrompt
	storyGen.Regen = 0
	storyGen.Params = string(storyGenData)
	storyGen.OriginID = req.StoryId
	storyGen.BoardID = 0
	storyGen.StartTime = time.Now().Unix()
	storyGen.GenType = int(req.GetRenderType())
	storyGen.TaskType = 1
	storyGen.Status = 1
	exist, _ := models.GetStoryGensByStory(ctx, req.StoryId, 1)
	if len(exist) > 0 {
		existGen := new(api.RenderStoryDetail)
		json.Unmarshal([]byte(exist[0].Content), existGen)
		return &api.RenderStoryResponse{
			Code:    0,
			Message: "story is rendering",
			Data:    existGen,
		}, nil
	}
	_, err = models.CreateStoryGen(ctx, storyGen)
	if err != nil {
		log.Log().Error("create story gen failed", zap.Error(err))
		return nil, err
	}
	renderDetail := new(api.RenderStoryDetail)
	renderStoryParams := &client.StoryInfoParams{
		Content: `生成一个 story_name 的故事,故事内容用中文描述,以json格式返回
		可以参考如下例子：
		--------
		{
			"故事名称和主题":{
				"故事名称": "火星绿洲",
				"故事主题": "人类在火星上的生存",
				"故事简介": "在2023年，国际火星探索任务成功地将首批人类送至火星。美国宇航员马克·沃特斯（Mark Watney）作为唯一的幸存者，面临着生死存亡的挑战。以下是他在火星上的求生记。"
			},
			
			"故事章节": [
				{
					"章节ID": "1",
					"章节题目": "火星上的孤岛",
					"章节内容": "马克在火星表面执行任务时，遭遇了一场突如其来的沙尘暴。他与同伴们在撤离过程中不幸与团队失去了联系。马克在沙尘暴中迷失方向，被火星表面的沙丘覆盖，最终昏迷。醒来后，他发现自己成为了火星上的孤岛求生者。"
				},
				{
					"章节ID": "2",
					"章节题目": "生存挑战",
					"章节内容": "马克意识到自己必须生存下去。他利用有限的资源，包括宇航服、食物和水，开始寻找生存的方法。他尝试修复通讯设备，但收到的只有静默。马克在火星上种菜、收集雨水，并研究如何利用太阳能来延长他的生存时间。"
				},
				{
					"章节ID": "3",
					"章节题目": "生存挑战",
					"章节内容": "马克意识到自己必须生存下去。他利用有限的资源，包括宇航服、食物和水，开始寻找生存的方法。他尝试修复通讯设备，但收到的只有静默。马克在火星上种菜、收集雨水，并研究如何利用太阳能来延长他的生存时间。"
				},
				{
					"章节ID": "4",
					"章节题目": "火星救援行动",
					"章节内容": "地球上接收到马克发出的信号后，立刻组织了救援行动。由于距离遥远，救援需要数月时间。马克在这期间不断改善自己的生存条件，甚至尝试与地球上的科学家进行通讯，寻求他们的帮助。"
				},
				{
					"章节ID": "x",
					"章节题目": "最终救援",
					"章节内容": "在漫长的等待中，马克终于等来了救援团队。他们利用火星漫游车抵达了马克的藏身之处。在地球上团队的努力下，马克被成功救回。"
				}
			]
		}
		--------
		请保证故事的连贯，以及故事中的各个人物的角色前后一致
		`,
	}
	start := time.Now()
	renderStoryParams.Content = strings.Replace(renderStoryParams.Content, "story_name", story.Origin, -1)
	var (
		ret  *client.StoryInfoResult
		resp = &api.RenderStoryResponse{}
	)
	if req.RenderType == api.RenderType_RENDER_TYPE_TEXT_UNSPECIFIED {
		renderDetail.StoryId = req.StoryId
		renderDetail.BoardId = req.BoardId
		ret, err = s.client.GenStoryInfo(ctx, renderStoryParams)
		if err != nil {
			log.Log().Error("gen story info failed", zap.Error(err))
			return nil, err
		}
	} else if req.RenderType == api.RenderType_RENDER_TYPE_STORYSENCE {
		renderDetail.StoryId = req.StoryId
		renderDetail.BoardId = req.BoardId
	} else if req.RenderType == api.RenderType_RENDER_TYPE_STORYCHARACTERS {
		renderDetail.StoryId = req.StoryId
		renderDetail.BoardId = req.BoardId
	} else if req.RenderType == api.RenderType_RENDER_TYPE_STORYACTION {
		renderDetail.StoryId = req.StoryId
		renderDetail.BoardId = req.BoardId
	} else if req.RenderType == api.RenderType_RENDER_TYPE_STORYSETTING {
		renderDetail.StoryId = req.StoryId
		renderDetail.BoardId = req.BoardId
	} else if req.RenderType == api.RenderType_RENDER_TYPE_STORYENDING {
		renderDetail.StoryId = req.StoryId
		renderDetail.BoardId = req.BoardId
	}

	// 渲染剧情
	result := new(StoryInfo)
	cleanResult := utils.CleanLLmJsonResult(ret.Content)
	err = json.Unmarshal([]byte(cleanResult), &result)
	if err != nil {
		log.Log().Error("unmarshal story gen result failed", zap.Error(err))
		return nil, err
	}
	renderDetail.Text = ret.Content
	renderDetail.RenderType = req.RenderType
	renderDetail.Timecost = int32(time.Since(start).Seconds())
	renderDetail.Result = new(api.StoryInfo)
	storyInfo := &api.StoryInfo{
		StoryNameAndTheme: &api.StoryNameAndTheme{},
		StoryChapters:     make([]*api.ChapterInfo, 0),
	}

	// 转换
	if result.StoryNameAndTheme.Name != "" {
		storyInfo.StoryNameAndTheme.Name = result.StoryNameAndTheme.Name
	}
	if result.StoryNameAndTheme.Theme != "" {
		storyInfo.StoryNameAndTheme.Theme = result.StoryNameAndTheme.Theme
	}
	if result.StoryNameAndTheme.Description != "" {
		storyInfo.StoryNameAndTheme.Description = result.StoryNameAndTheme.Description
	}

	// 处理章节信息
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
	storyGen.FinishTime = time.Now().Unix()
	err = models.UpdateStoryGen(ctx, storyGen)
	if err != nil {
		log.Log().Error("update story gen failed", zap.Error(err))
	}
	resp.Code = 0
	resp.Message = "OK"
	resp.Data = renderDetail
	return resp, nil
}

func (s *StoryService) GetStoryRender(ctx context.Context, req *api.GetStoryRenderRequest) (*api.GetStoryRenderResponse, error) {
	list, err := models.GetStoryGensByStory(ctx, req.GetStoryId(), 1)
	if err != nil {
		return nil, err
	}
	if len(list) == 0 {
		return &api.GetStoryRenderResponse{
			Code:    -1,
			Message: "story is not rendering",
		}, nil
	}

	item := new(api.RenderStoryDetail)
	err = json.Unmarshal([]byte(list[0].Content), &item)
	if err != nil {
		return nil, err
	}
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
	story, err := models.GetStory(ctx, req.GetStoryId())
	if err != nil {
		return nil, err
	}
	if story.Status == -1 {
		return &api.GetStoryContributorsResponse{
			Code:    -1,
			Message: "story is closed",
		}, nil
	}
	contributors, err := models.GetStoryContributors(ctx, int64(story.ID))
	if err != nil {
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

	return &api.GetStoryContributorsResponse{
		Code:    0,
		Message: "OK",
		Data: &api.GetStoryContributorsResponse_Data{
			List: apiContributors,
		},
	}, nil
}

func (s *StoryService) LikeStory(ctx context.Context, req *api.LikeStoryRequest) (*api.LikeStoryResponse, error) {
	// 检查是否已经点赞
	likeItem, err := models.GetLikeItemByStoryAndUser(ctx, req.GetStoryId(), int(req.GetUserId()))
	if err != nil {
		return nil, err
	}
	if likeItem != nil {
		return &api.LikeStoryResponse{
			Code:    0,
			Message: "already liked",
		}, nil
	}
	// 点赞
	newLike := new(models.LikeItem)
	newLike.StoryID = req.GetStoryId()
	newLike.UserID = int64(req.GetUserId())
	newLike.LikeItemType = models.LikeItemTypeStory
	err = models.CreateLikeStoryItem(ctx, newLike)
	if err != nil {
		return nil, err
	}
	// 更新故事的点赞数
	story, err := models.GetStory(ctx, req.GetStoryId())
	if err != nil {
		return nil, err
	}
	story.LikeCount = story.LikeCount + 1
	err = models.UpdateStory(ctx, story)
	if err != nil {
		return nil, err
	}
	userProfile := &models.UserProfile{
		UserId: int64(req.GetUserId()),
	}
	err = userProfile.IncrementLikedStoryNum()
	if err != nil {
		log.Log().Error("increment liked story num failed", zap.Error(err))
	}
	group := &models.Group{}
	group.ID = uint(story.GroupID)
	err = group.GetByID()
	if err != nil {
		return nil, err
	} else {
		active.GetActiveServer().WriteStoryActive(ctx, group, story, nil, nil, req.UserId, api.ActiveType_LikeStory)
	}
	return &api.LikeStoryResponse{
		Code:    0,
		Message: "OK",
	}, nil
}

func (s *StoryService) UnLikeStory(ctx context.Context, req *api.UnLikeStoryRequest) (*api.UnLikeStoryResponse, error) {
	likeItem, err := models.GetLikeItemByStoryAndUser(ctx, req.GetStoryId(), int(req.GetUserId()))
	if err != nil {
		return nil, err
	}
	if likeItem == nil {
		return &api.UnLikeStoryResponse{
			Code:    -1,
			Message: "not liked",
		}, nil
	}
	// 更新故事的点赞数
	story, err := models.GetStory(ctx, req.GetStoryId())
	if err != nil {
		return nil, err
	}
	story.LikeCount = story.LikeCount - 1
	err = models.UpdateStory(ctx, story)
	if err != nil {
		return nil, err
	}
	err = models.DeleteLikeItem(ctx, int64(likeItem.ID))
	if err != nil {
		return nil, err
	}
	userProfile := &models.UserProfile{
		UserId: int64(req.GetUserId()),
	}
	err = userProfile.DecrementLikedStoryNum()
	if err != nil {
		log.Log().Error("decrement liked story num failed", zap.Error(err))
	}
	return &api.UnLikeStoryResponse{
		Code:    0,
		Message: "OK",
	}, nil
}

func (s *StoryService) SearchStories(ctx context.Context, req *api.SearchStoriesRequest) (*api.SearchStoriesResponse, error) {
	stories, total, err := models.GetStoriesByName(ctx, req.GetKeyword(), int(req.GetOffset()), int(req.GetPageSize()))
	if err != nil {
		log.Log().Error("get story roles failed", zap.Error(err))
		return nil, err
	}
	apiStories := make([]*api.Story, 0)
	for _, story := range stories {
		info := convert.ConvertStoryToApiStory(story)
		info.CurrentUserStatus, err = s.GetStoryCurrentUserStatus(ctx, int64(story.ID))
		if err != nil {
			log.Log().Error("get story current user status failed", zap.Error(err))
		}
		apiStories = append(apiStories, info)
	}
	return &api.SearchStoriesResponse{
		Code:    0,
		Message: "OK",
		Stories: apiStories,
		Total:   total,
	}, nil
}

func (s *StoryService) SearchRoles(ctx context.Context, req *api.SearchRolesRequest) (*api.SearchRolesResponse, error) {
	roles, total, err := models.GetStoryRolesByName(ctx, req.GetKeyword(), req.GetStoryId(), int(req.GetOffset()), int(req.GetPageSize()))
	if err != nil {
		log.Log().Error("get story roles failed", zap.Error(err))
		return nil, err
	}
	apiRoles := make([]*api.StoryRole, 0)
	for _, role := range roles {
		info := convert.ConvertStoryRoleToApiStoryRoleInfo(role)
		if role.CharacterDetail != "" {
			roleDetail := &CharacterDetailConverter{}
			err = json.Unmarshal([]byte(role.CharacterDetail), &roleDetail)
			if err != nil {
				log.Log().Error("unmarshal story role character detail failed", zap.Error(err))
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
		info.CurrentUserStatus, err = s.GetStoryRoleCurrentUserStatus(ctx, int64(role.ID))
		if err != nil {
			log.Log().Error("get story role current user status failed", zap.Error(err))
		}
		info.LikeCount = role.LikeCount
		info.FollowCount = role.FollowCount
		info.StoryboardNum = role.StoryboardNum
		apiRoles = append(apiRoles, info)
	}
	return &api.SearchRolesResponse{
		Code:    0,
		Message: "OK",
		Roles:   apiRoles,
		Total:   total,
	}, nil
}

func (s *StoryService) UpdateStoryRoleAvator(ctx context.Context, req *api.UpdateStoryRoleAvatorRequest) (*api.UpdateStoryRoleAvatorResponse, error) {
	roleinfo, err := models.GetStoryRoleByID(ctx, req.GetRoleId())
	if err != nil {
		fmt.Errorf("get story role failed", zap.Error(err))
		return nil, err
	}
	if roleinfo.CreatorID != req.GetUserId() {
		fmt.Errorf("have no permission", roleinfo.CreatorID, req.GetUserId())
		//return nil, errors.New("have no permission")
	}
	roleinfo.CharacterAvatar = req.GetAvator()
	err = models.UpdateStoryRole(ctx, int64(roleinfo.ID), map[string]interface{}{
		"character_avatar": req.GetAvator(),
	})
	if err != nil {
		fmt.Errorf("update story role failed", zap.Error(err))
		return nil, err
	}
	return &api.UpdateStoryRoleAvatorResponse{
		Code:    0,
		Message: "OK",
	}, nil
}

func (s *StoryService) GetStoryRoleList(ctx context.Context, req *api.GetStoryRoleListRequest) (*api.GetStoryRoleListResponse, error) {
	roles, _, err := models.GetStoryRolesByName(ctx, req.GetSearchKey(), req.GetStoryId(), int(req.GetOffset()), int(req.GetPageSize()))
	if err != nil {
		log.Log().Error("get story roles failed", zap.Error(err))
		return nil, err
	}
	apiRoles := make([]*api.StoryRole, 0)
	for _, role := range roles {
		info := convert.ConvertStoryRoleToApiStoryRoleInfo(role)
		if role.CharacterDetail != "" {
			roleDetail := &CharacterDetailConverter{}
			err = json.Unmarshal([]byte(role.CharacterDetail), &roleDetail)
			if err != nil {
				log.Log().Error("unmarshal story role character detail failed", zap.Error(err))
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
	return &api.GetStoryRoleListResponse{
		Code:    0,
		Message: "OK",
		Roles:   apiRoles,
	}, nil
}

func (s *StoryService) TrendingStory(ctx context.Context, req *api.TrendingStoryRequest) (*api.TrendingStoryResponse, error) {
	stories, err := models.GetTrendingStories(ctx, int(req.GetPageNumber()), int(req.GetPageSize()), req.GetStart(), req.GetEnd())
	if err != nil {
		return nil, err
	}
	if len(stories) == 0 {
		return &api.TrendingStoryResponse{
			Code:    0,
			Message: "OK",
		}, nil
	}
	apiStories := make([]*api.Story, 0)
	for _, story := range stories {
		info := convert.ConvertStoryToApiStory(story)
		info.CurrentUserStatus, err = s.GetStoryCurrentUserStatus(ctx, int64(story.ID))
		if err != nil {
			log.Log().Error("get story current user status failed", zap.Error(err))
		}
		apiStories = append(apiStories, info)
	}
	return &api.TrendingStoryResponse{
		Code:    0,
		Message: "OK",
		Data: &api.TrendingStoryResponse_Data{
			List:       apiStories,
			PageSize:   req.GetPageSize(),
			PageNumber: req.GetPageNumber(),
		},
	}, nil
}

func (s *StoryService) TrendingStoryRole(ctx context.Context, req *api.TrendingStoryRoleRequest) (*api.TrendingStoryRoleResponse, error) {
	roles, err := models.GetTrendingStoryRoles(ctx, int(req.GetPageNumber()), int(req.GetPageSize()), req.GetStart(), req.GetEnd())
	if err != nil {
		return nil, err
	}
	if len(roles) == 0 {
		return &api.TrendingStoryRoleResponse{
			Code:    0,
			Message: "OK",
		}, nil
	}
	apiRoles := make([]*api.StoryRole, 0)
	for _, role := range roles {
		info := convert.ConvertStoryRoleToApiStoryRoleInfo(role)
		info.CurrentUserStatus, err = s.GetStoryRoleCurrentUserStatus(ctx, int64(role.ID))
		if err != nil {
			log.Log().Error("get story role current user status failed", zap.Error(err))
		}
		log.Log().Info("trending story role", zap.Any("role", role.CharacterDetail))
		if role.CharacterDetail != "" {
			roleDetail := &CharacterDetailConverter{}
			err = json.Unmarshal([]byte(role.CharacterDetail), &roleDetail)
			if err != nil {
				log.Log().Error("unmarshal story role character detail failed", zap.Error(err))
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
		},
	}
	log.Log().Info("trending story role", zap.Any("resp", resp))
	return resp, nil
}
