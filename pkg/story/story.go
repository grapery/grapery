package story

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"

	api "github.com/grapery/common-protoc/gen"
	"github.com/grapery/grapery/models"
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
		FollowCount: 0,
		LikeCount:   0,
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
	return &api.GetStoryInfoResponse{
		Code:    0,
		Message: "OK",
		Data: &api.GetStoryInfoResponse_Data{
			Info: ConvertStoryToApiStory(storyInfo),
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

func (s *StoryService) CreateStoryboard(ctx context.Context, req *api.CreateStoryboardRequest) (resp *api.CreateStoryboardResponse, err error) {
	newStroyBoard := ConvertApiStoryBoardToStoryBoard(req.GetBoard())

	storyInfo, err := models.GetStory(ctx, req.Board.StoryId)
	if err != nil {
		return nil, err
	}
	if storyInfo.Status == -1 {
		return &api.CreateStoryboardResponse{
			Code:    0,
			Message: "story is closed",
		}, nil
	}
	newStroyBoard.IsAiGen = storyInfo.AIGen
	newStroyBoard.StoryID = req.Board.StoryId
	newStroyBoard.CreatorID = req.Board.Creator
	newStroyBoard.ForkAble = true
	newStroyBoard.Status = 1
	storyBoardId, err := models.CreateStoryBoard(ctx, newStroyBoard)
	if err != nil {
		return nil, err
	}
	if storyInfo.RootBoardID == 0 {
		err = models.UpdateStorySpecColumns(ctx, req.Board.StoryId, map[string]interface{}{
			"root_board_id": storyBoardId,
		})
		if err != nil {
			return nil, err
		}
	}
	return &api.CreateStoryboardResponse{
		Code:    0,
		Message: "create storyboard success",
		Data: &api.CreateStoryboardResponse_Data{
			BoardId: storyBoardId,
		},
	}, nil
}

func (s *StoryService) GetStoryboard(ctx context.Context, req *api.GetStoryboardRequest) (resp *api.GetStoryboardResponse, err error) {
	boardInfo, err := models.GetStoryboard(ctx, req.BoardId)
	if err != nil {
		return nil, err
	}
	storyInfo, err := models.GetStory(ctx, boardInfo.StoryID)
	if err != nil {
		return nil, err
	}
	if storyInfo.Status == -1 && boardInfo.CreatorID != req.GetBoardId() {
		return &api.GetStoryboardResponse{
			Code:    0,
			Message: "story is closed",
		}, nil
	}
	sences, err := models.GetStoryBoardScenesByBoard(ctx, req.BoardId)
	if err != nil {
		log.Log().Error("get board sences failed", zap.Error(err))
	}

	boardInfoData, _ := json.Marshal(boardInfo)
	log.Log().Info("get storyboard success", zap.String("board", string(boardInfoData)))
	board := ConvertStoryBoardToApiStoryBoard(boardInfo)
	if len(sences) != 0 {
		board.Sences = new(api.StoryBoardSences)
		for _, scene := range sences {
			sceneData, _ := json.Marshal(scene)
			log.Log().Info("get scene success", zap.String("scene", string(sceneData)))
			board.Sences.List = append(board.Sences.List, ConvertStorySceneToApiScene(scene))
		}
		board.Sences.Total = int64(len(board.Sences.List))
	}
	return &api.GetStoryboardResponse{
		Code:    0,
		Message: "OK",
		Data: &api.GetStoryboardResponse_Data{
			Info: board,
		},
	}, nil
}

func (s *StoryService) UpdateStoryboard(ctx context.Context, req *api.UpdateStoryboardRequest) (resp *api.UpdateStoryboardResponse, err error) {
	boardInfo, err := models.GetStoryboard(ctx, req.BoardId)
	if err != nil {
		return nil, err
	}
	if boardInfo.CreatorID != req.GetBoardId() {
		return &api.UpdateStoryboardResponse{}, nil
	}
	needUpdateData := make(map[string]interface{})
	if req.Params != nil {
		paramsData, _ := json.Marshal(req.Params)
		needUpdateData["params"] = string(paramsData)
	}
	if len(needUpdateData) == 0 {
		return &api.UpdateStoryboardResponse{}, nil
	}
	err = models.UpdateStoryboardMultiColumn(ctx, req.BoardId, needUpdateData)
	if err != nil {
		return nil, err
	}
	return &api.UpdateStoryboardResponse{}, nil
}

func (s *StoryService) GetStoryboards(ctx context.Context, req *api.GetStoryboardsRequest) (resp *api.GetStoryboardsResponse, err error) {
	boardList, err := models.GetStoryboardsByStory(ctx, req.StoryId)
	if err != nil {
		return nil, err
	}
	datas := make([]*api.StoryBoard, 0)
	for _, board := range boardList {
		fmt.Println("board: ", ConvertStoryBoardToApiStoryBoard(board).String())
		sences, err := models.GetStoryBoardScenesByBoard(ctx, int64(board.ID))
		if err != nil {
			log.Log().Error("get board sences failed", zap.Error(err))
		}
		boardInfo := ConvertStoryBoardToApiStoryBoard(board)
		if len(sences) != 0 {
			boardInfo.Sences = new(api.StoryBoardSences)
			for _, scene := range sences {
				sceneData, _ := json.Marshal(scene)
				log.Log().Info("get scene success", zap.String("scene", string(sceneData)))
				boardInfo.Sences.List = append(boardInfo.Sences.List, ConvertStorySceneToApiScene(scene))
			}
			boardInfo.Sences.Total = int64(len(boardInfo.Sences.List))
		}
		datas = append(datas, boardInfo)
	}
	return &api.GetStoryboardsResponse{
		Code:    0,
		Message: "OK",
		Data: &api.GetStoryboardsResponse_Data{
			List:  datas,
			Total: int32(len(datas)),
		},
	}, nil
}

func (s *StoryService) DelStoryboard(ctx context.Context, req *api.DelStoryboardRequest) (resp *api.DelStoryboardResponse, err error) {
	// 1. Get current storyboard details
	currentBoard, err := models.GetStoryboard(ctx, req.BoardId)
	if err != nil {
		return nil, err
	}

	// 2. Get boards that have current board as their prevId
	childBoards, err := models.GetStoryboardsByPrevId(ctx, req.BoardId)
	if err != nil {
		return nil, err
	}

	// 3. Update all child boards to point to current board's prevId
	for _, childBoard := range childBoards {
		updateData := map[string]interface{}{
			"prev_id": currentBoard.PrevId,
		}
		err = models.UpdateStoryboardMultiColumn(ctx, int64(childBoard.ID), updateData)
		if err != nil {
			return nil, err
		}
	}

	// 4. Mark current board as deleted
	needUpdateData := map[string]interface{}{
		"status": -1,
	}
	err = models.UpdateStoryboardMultiColumn(ctx, req.BoardId, needUpdateData)
	if err != nil {
		return nil, err
	}

	return &api.DelStoryboardResponse{
		Code:    0,
		Message: "OK",
	}, nil
}

func (s *StoryService) ForkStoryboard(ctx context.Context, req *api.ForkStoryboardRequest) (resp *api.ForkStoryboardResponse, err error) {
	originStoryBoard, err := models.GetStoryboard(ctx, req.PrevBoardId)
	if err != nil {
		log.Log().Error("get origin story board failed", zap.Error(err))
		return nil, err
	}
	newStoryBoard := new(models.StoryBoard)
	originData, err := json.Marshal(originStoryBoard)
	if err != nil {
		log.Log().Error("marshal origin story board failed", zap.Error(err))
		return nil, err
	}
	err = json.Unmarshal(originData, newStoryBoard)
	if err != nil {
		log.Log().Error("unmarshal origin story board failed", zap.Error(err))
		return nil, err
	}
	id, err := models.CreateStoryBoard(ctx, originStoryBoard)
	if err != nil {
		log.Log().Error("create new story board failed", zap.Error(err))
		return nil, err
	}
	resp = &api.ForkStoryboardResponse{
		Code:    0,
		Message: "OK",
		Data: &api.ForkStoryboardResponse_Data{
			BoardId: int64(id),
		},
	}
	log.Log().Info("fork storyboard success", zap.Int64("new_board_id", id))
	return resp, nil
}

func (s *StoryService) LikeStoryboard(ctx context.Context, req *api.LikeStoryboardRequest) (resp *api.LikeStoryboardResponse, err error) {
	item := new(models.LikeItem)
	err = models.CreateLikeStoryBoardItem(ctx, item)
	if err != nil {
		log.Log().Error("create like item failed", zap.Error(err))
		return nil, err
	}
	resp = &api.LikeStoryboardResponse{
		Code:    0,
		Message: "OK",
	}
	log.Log().Info("like storyboard success", zap.Int64("item_id", int64(item.ID)))
	return resp, nil
}

func (s *StoryService) ShareStoryboard(ctx context.Context, req *api.ShareStoryboardRequest) (resp *api.ShareStoryboardResponse, err error) {
	return &api.ShareStoryboardResponse{
		Code:    0,
		Message: "NOT IMPLEMENTED",
	}, nil
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
    
    "故事章节": {
        "第1章": {
            "章节题目": "火星上的孤岛",
            "章节内容": "马克在火星表面执行任务时，遭遇了一场突如其来的沙尘暴。他与同伴们在撤离过程中不幸与团队失去了联系。马克在沙尘暴中迷失方向，被火星表面的沙丘覆盖，最终昏迷。醒来后，他发现自己成为了火星上的孤岛求生者。"
        },
        "第2章": {
            "章节题目": "生存挑战",
            "章节内容": "马克意识到自己必须生存下去。他利用有限的资源，包括宇航服、食物和水，开始寻找生存的方法。他尝试修复通讯设备，但收到的只有静默。马克在火星上种菜、收集雨水，并研究如何利用太阳能来延长他的生存时间。"
        },
        "第3章": {
            "章节题目": "生存挑战",
            "章节内容": "马克意识到自己必须生存下去。他利用有限的资源，包括宇航服、食物和水，开始寻找生存的方法。他尝试修复通讯设备，但收到的只有静默。马克在火星上种菜、收集雨水，并研究如何利用太阳能来延长他的生存时间。"
        },
        "第4章": {
            "章节题目": "火星救援行动",
            "章节内容": "地球上接收到马克发出的信号后，立刻组织了救援行动。由于距离遥远，救援需要数月时间。马克在这期间不断改善自己的生存条件，甚至尝试与地球上的科学家进行通讯，寻求他们的帮助。"
        },
		"第x章": {
            "章节题目": "最终救援",
            "章节内容": "在漫长的等待中，马克终于等来了救援团队。他们利用火星漫游车抵达了马克的藏身之处。在地球上团队的努力下，马克被成功救回。"
        }
    }
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
	fmt.Println("storyGen.Params: ", req.String())
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
	result := make(map[string]map[string]interface{})
	cleanResult := utils.CleanLLmJsonResult(ret.Content)
	fmt.Println("cleanResult: ", cleanResult)
	err = json.Unmarshal([]byte(cleanResult), &result)
	if err != nil {
		log.Log().Error("unmarshal story gen result failed", zap.Error(err))
		return nil, err
	}
	renderDetail.Text = ret.Content
	renderDetail.RenderType = req.RenderType
	renderDetail.Timecost = int32(time.Since(start).Seconds())
	renderDetail.Result = make(map[string]*api.RenderStoryStructure)
	renderDetail.Result["story"] = &api.RenderStoryStructure{
		Text: ret.Content,
		Data: make(map[string]*api.RenderStoryStructureValue),
	}
	// 转换
	for key, val := range result {
		if key == "故事名称和主题" {
			for chapter, va := range val {
				if chapter == "故事名称" {
					renderDetail.Result["story"].Data[chapter] = &api.RenderStoryStructureValue{
						Text: va.(string),
					}
				} else if chapter == "故事主题" {
					renderDetail.Result["story"].Data[chapter] = &api.RenderStoryStructureValue{
						Text: va.(string),
					}
				} else if chapter == "故事简介" {
					renderDetail.Result["story"].Data[chapter] = &api.RenderStoryStructureValue{
						Text: va.(string),
					}
				}
			}
		} else if key == "故事章节" {
			for chapter, va := range val {
				renderDetail.Result[chapter] = &api.RenderStoryStructure{
					Text: "",
					Data: make(map[string]*api.RenderStoryStructureValue),
				}
				for subchapter, subva := range va.(map[string]interface{}) {
					if subchapter == "章节题目" {
						renderDetail.Result[chapter].Data[subchapter] = &api.RenderStoryStructureValue{
							Text: subva.(string),
						}
					} else if subchapter == "章节内容" {
						renderDetail.Result[chapter].Data[subchapter] = &api.RenderStoryStructureValue{
							Text: subva.(string),
						}
					}
				}
			}
		}
	}
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

func (s *StoryService) RenderStoryboard(ctx context.Context, req *api.RenderStoryboardRequest) (*api.RenderStoryboardResponse, error) {
	board, err := models.GetStoryboard(ctx, req.GetBoardId())
	if err != nil {
		return nil, err
	}
	story, err := models.GetStory(ctx, board.StoryID)
	if err != nil {
		return nil, err
	}
	if story.IsAchieve {
		return &api.RenderStoryboardResponse{
			Code:    -1,
			Message: "story is achieve",
		}, nil

	}

	if story.Status == -1 {
		return &api.RenderStoryboardResponse{
			Code:    -1,
			Message: "story is closed",
		}, nil
	}
	stroyGen, err := models.GetStoryGensByStoryBoard(ctx, req.StoryId, 1)
	if err != nil {
		return nil, err
	}
	if len(stroyGen) > 0 && stroyGen[0].Status == 1 {
		return &api.RenderStoryboardResponse{
			Code:    -1,
			Message: "storyboard is rendering",
		}, nil
	}
	genParams := new(models.StoryBoardParams)
	genParams.StoryContent = story.Origin
	err = json.Unmarshal([]byte(board.Params), genParams)
	if err != nil {
		log.Log().Error("unmarshal storyboard gen params failed", zap.Error(err))
		return nil, err
	}
	storyGen := new(models.StoryGen)
	storyGen.Uuid = uuid.New().String()
	storyGenData, _ := json.Marshal(genParams)
	storyParam := new(api.StoryParams)
	json.Unmarshal([]byte(story.Params), &storyParam)

	templatePrompt := `
	为故事章节的 """story_chapter""" 章节的生成详细故事情节细节，请参考故事剧情: """story_content"""。
	故事背景为: """story_background"""。
	同时衔接前后章节的情节,上一章节的故事情节为: """story_backgroup"""，生成符合上下文的、合理的、更详细的情节，
	可以生成4-6个故事的细节，以及生成可以展示这些故事剧情的图片 prompt 提示词。
	以json格式返回格式可以参考如下例子:
	--------
		{
			"章节情节简述": {
				"章节题目": "地球生存环境恶化",
				"章节内容": "地球资源日益枯竭，人类将目光投向了火星。我国成功组建了一支马克为首的精英宇航员队伍，肩负起在火星建立基地的重任，为地球移民做准备"
			},
			"章节详细情节": {
				"详细情节-1": {
					"情节内容": "气候变化，温室效应加剧，全球平均气温上升超过2摄氏度，极端天气事件频发，如飓风、干旱、洪水等",
					"参与人物": "",
					"图片提示词": "一个城市被严重的雾霾笼罩，天空灰暗，远处的高楼大厦若隐若现，人们戴着口罩匆匆行走，街道上的车辆行驶缓慢，整个场景透露出压抑和不安。"
				},
				"详细情节-2": {
					"情节内容": "资源枯竭，可耕地减少，粮食产量下降，粮食危机日益严重；淡资源匮乏，多地出现用水紧张状况；矿产资源开采难度加大，能源供应紧张。",
					"参与人物": "",
					"图片提示词": "一片荒芜的农田，土壤干裂，庄稼枯萎，农民面露愁容地看着土地，天空中没有云彩，烈日炎炎，展现出粮食危机的严峻景象"
				}
			}
		}
	--------
	请保证故事的连贯，以及故事中的各个人物的角色前后一致，同时和故事背景契合，人物的描述清晰，情节人物的性格明显，场景描述详细，图片提示词准确。
	`
	templatePrompt = strings.Replace(templatePrompt, "story_chapter", board.Title, -1)
	if storyParam.Background != "" {
		templatePrompt = strings.Replace(templatePrompt, "story_background", storyParam.Background, -1)
	} else {
		templatePrompt = strings.Replace(templatePrompt, "story_background", story.Origin, -1)
	}
	templatePrompt = strings.Replace(templatePrompt, "story_content", board.Description, -1)
	var storyBackgroup string
	if board.PrevId != -1 && board.PrevId != 0 {
		prevBoard, err := models.GetStoryboard(ctx, board.PrevId)
		if err != nil && !strings.Contains(strings.ToLower(err.Error()), "not found") {
			log.Log().Error("get prev storyboard failed", zap.Error(err))
			return nil, err
		}
		storyBackgroup = prevBoard.Description
		templatePrompt = strings.Replace(templatePrompt, "story_backgroup ", storyBackgroup, -1)
	} else {
		templatePrompt = strings.Replace(templatePrompt, ",上一章节的故事情节为: story_backgroup ", "", -1)
	}
	storyGen.LLmPlatform = "Zhipu"
	storyGen.NegativePrompt = ""
	storyGen.PositivePrompt = templatePrompt
	storyGen.Regen = 0
	storyGen.Params = string(storyGenData)
	storyGen.OriginID = int64(story.ID)
	storyGen.StartTime = time.Now().Unix()
	storyGen.BoardID = req.GetBoardId()
	storyGen.GenType = int(req.GetRenderType())
	storyGen.TaskType = 2
	_, err = models.CreateStoryGen(ctx, storyGen)
	if err != nil {
		log.Log().Error("create storyboard gen failed", zap.Error(err))
		return nil, err
	}

	log.Log().Sugar().Info("gen storyboard prompt: ", templatePrompt)
	renderStoryParams := &client.StoryInfoParams{
		Content: templatePrompt,
	}
	result := make(map[string]map[string]interface{})
	start := time.Now()
	ret, err := s.client.GenStoryBoardInfo(ctx, renderStoryParams)
	if err != nil {
		log.Log().Error("gen storyboard info failed", zap.Error(err))
		return nil, err
	}
	retData, _ := json.Marshal(ret)
	log.Log().Sugar().Infof("gen storyboard info success, req: %s, data:%s", req.String(), string(retData))
	// 保存生成的故事板
	cleanResult := utils.CleanLLmJsonResult(ret.Content)
	fmt.Println("cleanResult: ", cleanResult)
	err = json.Unmarshal([]byte(cleanResult), &result)
	if err != nil {
		log.Log().Error("unmarshal story gen result failed", zap.Error(err))
		return nil, err
	}
	// 渲染剧情
	renderDetail := new(api.RenderStoryboardDetail)
	renderDetail.RenderType = req.RenderType
	renderDetail.Timecost = int32(time.Since(start).Seconds())
	renderDetail.Result = make(map[string]*api.RenderStoryStructure)
	renderDetail.Result["storyboard"] = &api.RenderStoryStructure{
		Text: ret.Content,
		Data: make(map[string]*api.RenderStoryStructureValue),
	}
	// 转换
	resultData, _ := json.Marshal(result)
	log.Log().Sugar().Info("gen storyboard result: ", string(resultData))
	for key, val := range result {
		if key == "章节情节简述" {
			for chapter, va := range val {
				if chapter == "章节题目" {
					renderDetail.Result["storyboard"].Data[chapter] = &api.RenderStoryStructureValue{
						Text: va.(string),
					}
				} else if chapter == "章节内容" {
					renderDetail.Result["storyboard"].Data[chapter] = &api.RenderStoryStructureValue{
						Text: va.(string),
					}
				}
			}
		} else if key == "章节详细情节" {
			for chapter, va := range val {
				renderDetail.Result[chapter] = &api.RenderStoryStructure{
					Text: "",
					Data: make(map[string]*api.RenderStoryStructureValue),
				}
				for subchapter, subva := range va.(map[string]interface{}) {
					if subchapter == "情节内容" {
						renderDetail.Result[chapter].Data[subchapter] = &api.RenderStoryStructureValue{
							Text: subva.(string),
						}
					} else if subchapter == "参与人物" {
						renderDetail.Result[chapter].Data[subchapter] = &api.RenderStoryStructureValue{
							Text: subva.(string),
						}
					} else if subchapter == "图片提示词" {
						renderDetail.Result[chapter].Data[subchapter] = &api.RenderStoryStructureValue{
							Text: subva.(string),
						}
					}
				}
			}
		}
	}
	renderDetailData, _ := json.Marshal(renderDetail)
	storyGen.Content = string(renderDetailData)
	storyGen.FinishTime = time.Now().Unix()
	err = models.UpdateStoryGen(ctx, storyGen)
	if err != nil {
		log.Log().Error("update story gen failed", zap.Error(err))
	}
	renderDetailData, _ = json.Marshal(renderDetail)
	log.Log().Sugar().Info("gen storyboard result: ", string(renderDetailData))
	// 渲染剧情板
	return &api.RenderStoryboardResponse{
		Code:    0,
		Message: "OK",
		Data:    renderDetail,
	}, nil
}
func (s *StoryService) GenStoryboardImages(ctx context.Context, req *api.GenStoryboardImagesRequest) (*api.GenStoryboardImagesResponse, error) {
	board, err := models.GetStoryboard(ctx, req.GetBoardId())
	if err != nil {
		return nil, err
	}
	story, err := models.GetStory(ctx, board.StoryID)
	if err != nil {
		return nil, err
	}
	if story.IsAchieve {
		return &api.GenStoryboardImagesResponse{
			Code:    -1,
			Message: "story is achieve",
		}, nil
	}
	if story.Status == -1 {
		return &api.GenStoryboardImagesResponse{
			Code:    -1,
			Message: "story is closed",
		}, nil
	}
	stroyboardGen, err := models.GetStoryGensByStoryBoard(ctx, req.BoardId, 1)
	if err != nil {
		return nil, err
	}

	if len(stroyboardGen) == 0 {
		return &api.GenStoryboardImagesResponse{
			Code:    -1,
			Message: "storyboard is not rendering",
		}, nil
	}

	genParams := new(models.StoryBoardParams)
	genParams.StoryContent = story.Origin
	err = json.Unmarshal([]byte(board.Params), genParams)
	if err != nil {
		log.Log().Error("unmarshal storyboard gen params failed", zap.Error(err))
		return nil, err
	}

	result := make(map[string]map[string]interface{})
	err = json.Unmarshal([]byte(stroyboardGen[0].Content), &result)
	if err != nil {
		log.Log().Error("unmarshal storyboard gen result failed", zap.Error(err))
		return nil, err
	}
	for key, value := range result {
		if key == "章节情节简述" {
			log.Log().Sugar().Info("chapter: ", value)
		} else if key == "章节详细情节" {
			for chapter, va := range value {
				log.Log().Sugar().Info("章节详细情节: ", chapter)
				charactorNum := 0
				for subchapter, subva := range va.(map[string]interface{}) {
					if subchapter == "情节内容" {
						log.Log().Sugar().Info("情节内容: ", subva.(string))
					} else if subchapter == "参与人物" {
						charactors := strings.Split(subva.(string), ",")
						charactorNum = len(charactors)
						log.Log().Sugar().Info("参与人物: ", subva.(string))
					}
				}
				for subchapter, subva := range va.(map[string]interface{}) {
					if subchapter == "图片提示词" {
						preDefineTemplate := strings.Replace(models.PreDefineTemplate[1].Prompt, "prompt", subva.(string), -1)
						templatePrompt := preDefineTemplate + ",人物数量:" + strconv.Itoa(charactorNum)
						storyGen := new(models.StoryGen)
						storyGen.Uuid = uuid.New().String()
						storyGenData, _ := json.Marshal(genParams)
						storyGen.LLmPlatform = "Zhipu"
						storyGen.NegativePrompt = prompt.ZhipuNegativePrompt
						storyGen.PositivePrompt = templatePrompt
						storyGen.Regen = 0
						storyGen.Params = string(storyGenData)
						storyGen.OriginID = int64(story.ID)
						storyGen.StartTime = time.Now().Unix()
						storyGen.BoardID = req.GetBoardId()
						storyGen.GenType = int(api.RenderType_RENDER_TYPE_STORYSENCE)
						_, err = models.CreateStoryGen(ctx, storyGen)
						if err != nil {
							log.Log().Error("create storyboard gen failed", zap.Error(err))
							return nil, err
						}

						renderStoryParams := &client.GenStoryImagesParams{
							Content: templatePrompt,
						}

						ret, err := s.client.GenStoryBoardImages(ctx, renderStoryParams)
						if err != nil {
							log.Log().Error("gen storyboard info failed", zap.Error(err))
							return nil, err
						}
						storyGen.ImageUrls = strings.Join(ret.ImageUrls, ",")
						storyGen.Content = ""
						storyGen.FinishTime = time.Now().Unix()
						err = models.UpdateStoryGen(ctx, storyGen)
						if err != nil {
							log.Log().Error("update storyboard image gen failed", zap.Error(err))
						}
					}
				}
			}
		}
	}

	return &api.GenStoryboardImagesResponse{
		Code:    0,
		Message: "OK",
		Data:    nil,
	}, nil
}

func (s *StoryService) GenStoryboardText(ctx context.Context, req *api.GenStoryboardTextRequest) (*api.GenStoryboardTextResponse, error) {
	return &api.GenStoryboardTextResponse{}, nil
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

func (s *StoryService) GetStoryBoardRender(ctx context.Context, req *api.GetStoryBoardRenderRequest) (*api.GetStoryBoardRenderResponse, error) {
	list, err := models.GetStoryGensByStoryBoard(ctx, req.GetBoardId(), 1)
	if err != nil {
		return nil, err
	}
	if len(list) == 0 {
		return &api.GetStoryBoardRenderResponse{
			Code:    -1,
			Message: "board is not rendering",
		}, nil
	}

	item := new(api.RenderStoryboardDetail)
	err = json.Unmarshal([]byte(list[0].Content), &item)
	if err != nil {
		return nil, err
	}
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

func (s *StoryService) ContinueRenderStory(ctx context.Context, req *api.ContinueRenderStoryRequest) (*api.ContinueRenderStoryResponse, error) {
	fmt.Println("board.GetPrevBoardId: ", req.String())
	board, err := models.GetStoryboard(ctx, req.PrevBoardId)
	if err != nil {
		return nil, err
	}
	story, err := models.GetStory(ctx, req.StoryId)
	if err != nil {
		return nil, err
	}
	if story.Status == -1 {
		return &api.ContinueRenderStoryResponse{
			Code:    -1,
			Message: "story is closed",
		}, nil
	}
	prevBoards := make([]*models.StoryBoard, 0)
	prevBoards = append(prevBoards, board)

	var boardIdtemp int64 = board.PrevId
	for boardIdtemp != 0 {
		prevBoard, err := models.GetStoryboard(ctx, boardIdtemp)
		if err != nil {
			return nil, err
		}
		boardIdtemp = prevBoard.PrevId
		prevBoards = append(prevBoards, prevBoard)
		if len(prevBoards) > 5 {
			break
		}
	}

	//  fork 时 据用户和 story 的id来检查是否重复
	genParams := new(models.StoryBoardParams)
	genParams.StoryContent = story.Origin
	err = json.Unmarshal([]byte(board.Params), genParams)
	if err != nil {
		log.Log().Error("unmarshal storyboard gen params failed", zap.Error(err))
		return nil, err
	}
	storyGen := new(models.StoryGen)
	storyGen.Uuid = uuid.New().String()
	storyGenData, _ := json.Marshal(genParams)
	boardRequire := make(map[string]interface{})
	boardRequire["章节题目要求"] = req.GetTitle()
	boardRequire["章节内容要求"] = req.GetDescription()
	boardRequire["章节背景简介"] = req.GetBackground()
	boardRequire["章节参与的角色要求"] = req.GetRoles()
	boardRequireJson, _ := json.Marshal(boardRequire)

	templatePrompt := `生成故事 story_name 的下一个章节,故事内容用中文描述,以json格式返回		
		之前的故事章节:
		--------
		story_prev_content
		--------
		请参考以上输入，生成故事的下一个章节。只生成新的章节的章节内容，章节题目，章节背景简介，章节参与的角色。请参考如下格式：
		{
			"章节内容": "xxxxxx......",
			"章节题目": "xxxxxx......",
			"章节参与的角色": "xxx,xxx,xxx,......",
			"章节背景简介": "xxxxxx......"
		}
		`
	if len(req.GetTitle()) > 0 || len(req.GetDescription()) > 0 || len(req.GetBackground()) > 0 || len(req.GetRoles()) > 0 {
		templatePrompt = templatePrompt + `章节要求：
		-------- ` + "\n" +
			string(boardRequireJson) +
			"\n" + ` --------`
	}
	templatePrompt = templatePrompt + `请保证故事的连贯，以及故事中的各个人物的角色前后一致。输出的数据结构和输入的保持一致`
	story_prev_content := make(map[string]map[string]interface{})
	storyName := make(map[string]interface{})
	storyName["故事名称"] = story.Name
	storyName["故事简介"] = story.Origin
	storyName["故事主题"] = story.Title
	story_prev_content["故事角色"] = make(map[string]interface{})
	story_prev_content["故事名称和主题"] = storyName
	story_prev_content["故事章节"] = make(map[string]interface{})
	for idx := len(prevBoards) - 1; idx >= 0; idx-- {
		prevBoard := prevBoards[idx]
		content := make(map[string]interface{})
		content["章节题目"] = prevBoard.Title
		content["章节内容"] = prevBoard.Description
		story_prev_content["故事章节"][fmt.Sprintf("第%d章", idx+1)] = content
	}
	story_prev_content_json, _ := json.Marshal(story_prev_content)
	templatePrompt = strings.Replace(templatePrompt, "story_name", story.Name, -1)
	templatePrompt = strings.Replace(templatePrompt, "story_prev_content", string(story_prev_content_json), -1)

	storyGen.LLmPlatform = "Zhipu"
	storyGen.NegativePrompt = ""
	storyGen.PositivePrompt = templatePrompt
	storyGen.Regen = 2
	storyGen.Params = string(storyGenData)
	storyGen.OriginID = req.GetStoryId()
	storyGen.StartTime = time.Now().Unix()
	storyGen.BoardID = req.GetPrevBoardId()
	storyGen.GenType = int(req.GetRenderType())
	storyGen.TaskType = 2
	_, err = models.CreateStoryGen(ctx, storyGen)
	if err != nil {
		log.Log().Error("create storyboard gen failed", zap.Error(err))
		return nil, err
	}
	log.Log().Sugar().Info("gen storyboard prompt: ", templatePrompt)
	renderStoryParams := &client.StoryInfoParams{
		Content: templatePrompt,
	}
	result := make(map[string]string)
	start := time.Now()
	ret, err := s.client.GenStoryInfo(ctx, renderStoryParams)
	if err != nil {
		log.Log().Error("gen storyboard info failed", zap.Error(err))
		return nil, err
	}
	retData, _ := json.Marshal(ret)
	log.Log().Sugar().Infof("gen storyboard info success, req: %s, data:%s", req.String(), string(retData))
	// 保存生成的故事板
	cleanResult := utils.CleanLLmJsonResult(ret.Content)
	fmt.Println("cleanResult: ", cleanResult)
	err = json.Unmarshal([]byte(cleanResult), &result)
	if err != nil {
		log.Log().Error("unmarshal story gen result failed", zap.Error(err))
		return nil, err
	}
	// 渲染剧情
	renderDetail := new(api.RenderStoryDetail)
	renderDetail.RenderType = req.RenderType
	renderDetail.Timecost = int32(time.Since(start).Seconds())
	renderDetail.Result = make(map[string]*api.RenderStoryStructure)
	chapter := "新的故事章节"
	renderDetail.Result[chapter] = &api.RenderStoryStructure{
		Text: "",
		Data: make(map[string]*api.RenderStoryStructureValue),
	}
	for key, val := range result {
		if key == "章节内容" {
			renderDetail.Result[chapter].Data[key] = &api.RenderStoryStructureValue{
				Text: val,
			}
		} else if key == "章节题目" {
			renderDetail.Result[chapter].Data[key] = &api.RenderStoryStructureValue{
				Text: val,
			}
		} else if key == "章节参与的角色" {
			renderDetail.Result[chapter].Data[key] = &api.RenderStoryStructureValue{
				Text: val,
			}
		} else if key == "章节背景简介" {
			renderDetail.Result[chapter].Data[key] = &api.RenderStoryStructureValue{
				Text: val,
			}
		}
	}
	renderDetailData, _ := json.Marshal(renderDetail)
	storyGen.Content = string(renderDetailData)
	storyGen.FinishTime = time.Now().Unix()
	err = models.UpdateStoryGen(ctx, storyGen)
	if err != nil {
		log.Log().Error("update story gen failed", zap.Error(err))
	}
	board.ForkNum = board.ForkNum + 1
	err = models.UpdateStoryboard(ctx, board)
	if err != nil {
		log.Log().Error("update story gen failed", zap.Error(err))
	}
	return &api.ContinueRenderStoryResponse{
		Code:    0,
		Message: "OK",
		Data:    renderDetail,
	}, nil
}

func (s *StoryService) RenderStoryRoles(ctx context.Context, req *api.RenderStoryRolesRequest) (*api.RenderStoryRolesResponse, error) {
	return &api.RenderStoryRolesResponse{
		Code:    0,
		Message: "OK",
	}, nil
}

func (s *StoryService) UpdateStoryRole(ctx context.Context, req *api.UpdateStoryRoleRequest) (*api.UpdateStoryRoleResponse, error) {
	role, err := models.GetStoryRoleByID(ctx, req.Role.GetRoleId())
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return &api.UpdateStoryRoleResponse{
				Code:    -1,
				Message: "role not found",
			}, nil
		}
		return nil, err
	}
	needUpdateFields := make(map[string]interface{})
	if req.Role.GetCharacterName() != "" {
		needUpdateFields["character_name"] = req.Role.GetCharacterName()
	}
	if req.Role.GetCharacterAvatar() != "" {
		needUpdateFields["character_avatar"] = req.Role.GetCharacterAvatar()
	}
	if req.Role.GetCharacterId() != "" {
		needUpdateFields["character_id"] = req.Role.GetCharacterId()
	}
	if req.Role.GetCharacterType() != "" {
		needUpdateFields["character_type"] = req.Role.GetCharacterType()
	}
	if req.Role.GetCharacterPrompt() != "" {
		needUpdateFields["character_prompt"] = req.Role.GetCharacterPrompt()
	}
	if len(req.Role.GetCharacterRefImages()) > 0 {
		needUpdateFields["character_ref_images"] = strings.Join(req.Role.GetCharacterRefImages(), ",")
	}
	if req.Role.GetCharacterDescription() != "" {
		needUpdateFields["character_description"] = req.Role.GetCharacterDescription()
	}

	err = models.UpdateStoryRole(ctx, int64(role.ID), needUpdateFields)
	if err != nil {
		return nil, err
	}
	return &api.UpdateStoryRoleResponse{
		Code:    0,
		Message: "OK",
	}, nil
}

func (s *StoryService) RenderStoryRoleDetail(ctx context.Context, req *api.RenderStoryRoleDetailRequest) (*api.RenderStoryRoleDetailResponse, error) {
	return &api.RenderStoryRoleDetailResponse{
		Code:    0,
		Message: "OK",
	}, nil
}
func (s *StoryService) GetStoryRoles(ctx context.Context, req *api.GetStoryRolesRequest) (*api.GetStoryRolesResponse, error) {
	story, err := models.GetStory(ctx, req.GetStoryId())
	if err != nil {
		return nil, err
	}
	if story.Status == -1 {
		return &api.GetStoryRolesResponse{
			Code:    -1,
			Message: "story is closed",
		}, nil
	}
	roles, err := models.GetStoryRole(ctx, int64(story.ID))
	if err != nil {
		return nil, err
	}
	apiRoles := make([]*api.StoryRole, 0)
	for _, role := range roles {
		apiRole := new(api.StoryRole)
		apiRole.RoleId = int64(role.ID)
		apiRole.CharacterName = role.CharacterName
		apiRole.CharacterAvatar = role.CharacterAvatar
		apiRole.CharacterId = role.CharacterID
		apiRole.CharacterType = role.CharacterType
		apiRole.CharacterPrompt = role.CharacterPrompt
		apiRole.CharacterRefImages = strings.Split(role.CharacterRefImages, ",")
		apiRole.CharacterDescription = role.CharacterDescription
		apiRoles = append(apiRoles, apiRole)
	}

	return &api.GetStoryRolesResponse{
		Code:    0,
		Message: "OK",
		Data: &api.GetStoryRolesResponse_Data{
			List: apiRoles,
		},
	}, nil
}
func (s *StoryService) GetStoryBoardRoles(ctx context.Context, req *api.GetStoryBoardRolesRequest) (*api.GetStoryBoardRolesResponse, error) {
	board, err := models.GetStoryboard(ctx, req.GetBoardId())
	if err != nil {
		return nil, err
	}
	story, err := models.GetStory(ctx, board.StoryID)
	if err != nil {
		return nil, err
	}
	if story.Status == -1 {
		return &api.GetStoryBoardRolesResponse{
			Code:    -1,
			Message: "story is closed",
		}, nil
	}
	roles, err := models.GetStoryRole(ctx, int64(story.ID))
	if err != nil {
		return nil, err
	}
	apiRoles := make([]*api.StoryRole, 0)
	for _, role := range roles {
		apiRole := new(api.StoryRole)
		apiRole.RoleId = int64(role.ID)
		apiRole.CharacterName = role.CharacterName
		apiRole.CharacterAvatar = role.CharacterAvatar
		apiRole.CharacterId = role.CharacterID
		apiRole.CharacterType = role.CharacterType
		apiRole.CharacterPrompt = role.CharacterPrompt
		apiRole.CharacterRefImages = strings.Split(role.CharacterRefImages, ",")
		apiRole.CharacterDescription = role.CharacterDescription
		apiRoles = append(apiRoles, apiRole)
	}

	return &api.GetStoryBoardRolesResponse{
		Code:    0,
		Message: "OK",
		Data: &api.GetStoryBoardRolesResponse_Data{
			List: apiRoles,
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
	err = models.DeleteLikeItem(ctx, int64(likeItem.ID))
	if err != nil {
		return nil, err
	}
	return &api.UnLikeStoryResponse{
		Code:    0,
		Message: "OK",
	}, nil
}

func (s *StoryService) UnLikeStoryboard(ctx context.Context, req *api.UnLikeStoryboardRequest) (*api.UnLikeStoryboardResponse, error) {
	likeItem, err := models.GetLikeItemByStoryBoardAndUser(ctx, req.GetStoryId(), req.GetBoardId(), int(req.GetUserId()))
	if err != nil {
		return nil, err
	}
	if likeItem == nil {
		return &api.UnLikeStoryboardResponse{
			Code:    -1,
			Message: "not liked",
		}, nil
	}
	err = models.DeleteLikeItem(ctx, int64(likeItem.ID))
	if err != nil {
		return nil, err
	}
	return &api.UnLikeStoryboardResponse{
		Code:    0,
		Message: "OK",
	}, nil
}

func (s *StoryService) GetStoryboardScene(ctx context.Context, req *api.GetStoryBoardSencesRequest) (*api.GetStoryBoardSencesResponse, error) {
	scenes, err := models.GetStoryBoardScenesByBoard(ctx, req.GetBoardId())
	if err != nil && err != gorm.ErrRecordNotFound {
		return nil, err
	}
	if len(scenes) == 0 {
		return &api.GetStoryBoardSencesResponse{
			Code:    0,
			Message: "no scenes",
		}, nil
	}
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
		apiScene.IsGenerating = int32(scene.IsGenerating)
		apiScene.GenResult = scene.GenResult
		apiScene.Status = int32(scene.Status)
		apiScene.Ctime = scene.CreateAt.Unix()
		apiScene.Mtime = scene.UpdateAt.Unix()
		apiScenes = append(apiScenes, apiScene)
	}
	return &api.GetStoryBoardSencesResponse{
		Code:    0,
		Message: "OK",
		Data: &api.GetStoryBoardSencesResponse_Data{
			List: apiScenes,
		},
	}, nil
}
func (s *StoryService) CreateStoryBoardScene(ctx context.Context, req *api.CreateStoryBoardSenceRequest) (*api.CreateStoryBoardSenceResponse, error) {
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
	newScene.IsGenerating = 0
	newScene.GenResult = ""
	_, err := models.CreateStoryBoardScene(ctx, newScene)
	if err != nil {
		log.Log().Error("create storyboard scene failed", zap.Error(err))
		return nil, err
	}
	newSceneData, _ := json.Marshal(newScene)
	log.Log().Sugar().Infof("create storyboard scene success, scene: %s", string(newSceneData))
	return &api.CreateStoryBoardSenceResponse{
		Code:    0,
		Message: "OK",
		Data: &api.CreateStoryBoardSenceResponse_Data{
			SenceId: int64(newScene.ID),
		},
	}, nil
}

func (s *StoryService) UpdateStoryBoardSence(ctx context.Context, req *api.UpdateStoryBoardSenceRequest) (*api.UpdateStoryBoardSenceResponse, error) {
	scene, err := models.GetStoryBoardScene(ctx, req.Sence.GetSenceId())
	if err != nil && err != gorm.ErrRecordNotFound {
		log.Log().Error("get storyboard scene failed", zap.Error(err))
		return nil, err
	}
	if scene == nil {
		log.Log().Error("scene not found")
		return &api.UpdateStoryBoardSenceResponse{
			Code:    -1,
			Message: "scene not found",
		}, nil
	}
	scene.Content = req.Sence.GetContent()
	scene.ImagePrompts = req.Sence.GetImagePrompts()
	scene.AudioPrompts = req.Sence.GetAudioPrompts()
	scene.VideoPrompts = req.Sence.GetVideoPrompts()
	scene.Status = int(req.Sence.GetStatus())
	scene.IsGenerating = int(req.Sence.GetIsGenerating())
	scene.GenResult = req.Sence.GetGenResult()
	err = models.UpdateStoryBoardScene(ctx, scene)
	if err != nil {
		log.Log().Error("update storyboard scene failed", zap.Error(err))
		return nil, err
	}
	log.Log().Sugar().Infof("update storyboard scene success, scene: %s", req.Sence.String())
	return &api.UpdateStoryBoardSenceResponse{
		Code:    0,
		Message: "OK",
	}, nil
}

func (s *StoryService) DeleteStoryBoardSence(ctx context.Context, req *api.DeleteStoryBoardSenceRequest) (*api.DeleteStoryBoardSenceResponse, error) {
	err := models.UpdateStoryBoardSceneStatus(ctx, req.GetSenceId(), -1)
	if err != nil {
		log.Log().Error("delete storyboard scene failed", zap.Error(err))
		return nil, err
	}
	log.Log().Sugar().Infof("delete storyboard scene success, scene: %d", req.GetSenceId())
	return &api.DeleteStoryBoardSenceResponse{
		Code:    0,
		Message: "OK",
	}, nil
}

// 通过生成的场景描述，生成每个场景的图片
func (s *StoryService) RenderStoryBoardSence(ctx context.Context, req *api.RenderStoryBoardSenceRequest) (*api.RenderStoryBoardSenceResponse, error) {
	if req.GetSenceId() <= 0 {
		log.Log().Error("sence id is 0")
		return &api.RenderStoryBoardSenceResponse{
			Code:    -1,
			Message: "sence id is 0",
		}, nil
	}
	if req.GetBoardId() <= 0 {
		log.Log().Error("board id is 0")
		return &api.RenderStoryBoardSenceResponse{
			Code:    -1,
			Message: "board id is 0",
		}, nil
	}
	board, err := models.GetStoryboard(ctx, int64(req.GetBoardId()))
	if err != nil {
		log.Log().Error("get storyboard failed", zap.Error(err))
		return nil, err
	}
	if board == nil {
		log.Log().Error("board not found")
		return &api.RenderStoryBoardSenceResponse{
			Code:    -1,
			Message: "board not found",
		}, nil
	}
	story, err := models.GetStory(ctx, int64(board.StoryID))
	if err != nil {
		log.Log().Error("get story failed", zap.Error(err))
		return nil, err
	}
	if story.Status < 0 {
		log.Log().Error("story is deleted")
		return &api.RenderStoryBoardSenceResponse{
			Code:    -1,
			Message: "story is deleted",
		}, nil
	}
	// 1. 获取场景描述
	scene, err := models.GetStoryBoardScene(ctx, req.GetSenceId())
	if err != nil {
		log.Log().Error("get storyboard scene failed", zap.Error(err))
		return nil, err
	}
	if scene == nil {
		log.Log().Error("scene not found")
		return &api.RenderStoryBoardSenceResponse{
			Code:    -1,
			Message: "scene not found",
		}, nil
	}
	if scene.Status == -1 {
		log.Log().Error("scene is deleted")
		return &api.RenderStoryBoardSenceResponse{
			Code:    -1,
			Message: "scene is deleted",
		}, nil
	}
	if scene.Status == 0 || scene.Status == 2 {
		log.Log().Error("scene is generating")
		return &api.RenderStoryBoardSenceResponse{
			Code:    0,
			Message: "scene is generating",
		}, nil
	}
	scene.IsGenerating = 1
	scene.Status = 2
	_ = models.UpdateStoryBoardScene(ctx, scene)
	// 2. 生成指定场景的图片
	templatePrompt := scene.ImagePrompts
	preDefineTemplate := strings.Replace(models.PreDefineTemplate[1].Prompt, "prompt", templatePrompt, -1)
	templatePrompt = preDefineTemplate + ",人物: " + scene.CharacterIds
	renderStoryParams := &client.GenStoryImagesParams{
		Content: templatePrompt,
	}
	log.Log().Sugar().Infof("render storyboard scene, scene: %s, prompt: %s", scene.Content, templatePrompt)
	ret, err := s.client.GenStoryBoardImages(ctx, renderStoryParams)
	if err != nil {
		log.Log().Error("gen storyboard info failed", zap.Error(err))
		return nil, err
	}
	retData, _ := json.Marshal(ret.ImageUrls)
	scene.GenResult = string(retData)
	scene.IsGenerating = 0
	scene.Status = 1
	err = models.UpdateStoryBoardScene(ctx, scene)
	if err != nil {
		log.Log().Error("update storyboard scene failed", zap.Error(err))
		return nil, err
	}
	log.Log().Sugar().Infof("render storyboard scene success, scene: %s", scene.GenResult)
	// 3. 返回生成结果
	return &api.RenderStoryBoardSenceResponse{
		Code:    0,
		Message: "OK",
		Data:    convert.ConvertStoryBoardSceneToApiStoryBoardScene(scene),
	}, nil
}

func (s *StoryService) RenderStoryBoardSences(ctx context.Context, req *api.RenderStoryBoardSencesRequest) (*api.RenderStoryBoardSencesResponse, error) {
	if req.GetBoardId() <= 0 {
		log.Log().Error("board id is 0")
		return &api.RenderStoryBoardSencesResponse{
			Code:    -1,
			Message: "board id is 0",
		}, nil
	}
	board, err := models.GetStoryboard(ctx, int64(req.GetBoardId()))
	if err != nil {
		log.Log().Error("get storyboard failed", zap.Error(err))
		return nil, err
	}
	if board == nil {
		log.Log().Error("board not found")
		return &api.RenderStoryBoardSencesResponse{
			Code:    -1,
			Message: "board not found",
		}, nil
	}

	story, err := models.GetStory(ctx, int64(board.StoryID))
	if err != nil {
		log.Log().Error("get story failed", zap.Error(err))
		return nil, err
	}
	if story.Status < 0 {
		log.Log().Error("story is deleted")
		return &api.RenderStoryBoardSencesResponse{
			Code:    -1,
			Message: "story is deleted",
		}, nil
	}
	// 1. 获取场景描述
	scenes, err := models.GetStoryBoardScenesByBoard(ctx, int64(req.GetBoardId()))
	if err != nil {
		log.Log().Error("get storyboard scene failed", zap.Error(err))
		return nil, err
	}
	if len(scenes) == 0 {
		log.Log().Error("scene not found")
		return &api.RenderStoryBoardSencesResponse{
			Code:    -1,
			Message: "scene not found",
		}, nil
	}
	for _, scene := range scenes {
		if scene.Status == -1 {
			log.Log().Error("scene is deleted")
			return &api.RenderStoryBoardSencesResponse{
				Code:    -1,
				Message: "scene is deleted",
			}, nil
		}
		if scene.Status == 0 || scene.Status == 2 {
			log.Log().Error("scene is generating")
			return &api.RenderStoryBoardSencesResponse{
				Code:    0,
				Message: "scene is generating",
			}, nil
		}
	}
	// 2. 生成每个场景的图片
	apiScenes := make([]*api.StoryBoardSence, 0)
	for _, scene := range scenes {
		templatePrompt := scene.ImagePrompts
		preDefineTemplate := strings.Replace(models.PreDefineTemplate[1].Prompt, "prompt", templatePrompt, -1)
		templatePrompt = preDefineTemplate + ",人物: " + scene.CharacterIds
		renderStoryParams := &client.GenStoryImagesParams{
			Content: templatePrompt,
		}
		log.Log().Sugar().Infof("render storyboard scene, scene: %s, prompt: %s", scene.Content, templatePrompt)
		ret, err := s.client.GenStoryBoardImages(ctx, renderStoryParams)
		if err != nil {
			log.Log().Error("gen storyboard info failed", zap.Error(err))
			return nil, err
		}
		retData, _ := json.Marshal(ret.ImageUrls)
		scene.GenResult = string(retData)
		scene.IsGenerating = 0
		scene.Status = 1
		err = models.UpdateStoryBoardScene(ctx, scene)
		if err != nil {
			log.Log().Error("update storyboard scene failed", zap.Error(err))
			return nil, err
		}
		apiScenes = append(apiScenes, convert.ConvertStoryBoardSceneToApiStoryBoardScene(scene))
	}
	return &api.RenderStoryBoardSencesResponse{
		Code:    0,
		Message: "OK",
		List:    apiScenes,
	}, nil
}

func (s *StoryService) GetStoryBoardSenceGenerate(ctx context.Context, req *api.GetStoryBoardSenceGenerateRequest) (*api.GetStoryBoardSenceGenerateResponse, error) {
	// 1. 获取场景描述
	scene, err := models.GetStoryBoardScene(ctx, req.GetSenceId())
	if err != nil {
		log.Log().Error("get storyboard scene failed", zap.Error(err))
		return nil, err
	}
	if scene == nil {
		log.Log().Error("scene not found")
		return &api.GetStoryBoardSenceGenerateResponse{
			Code:    -1,
			Message: "scene not found",
		}, nil
	}
	if scene.Status == 1 {
		log.Log().Error("scene is already generating")
		return &api.GetStoryBoardSenceGenerateResponse{
			Code:    0,
			Message: "scene is already generating",
		}, nil
	}
	if scene.Status == -1 {
		log.Log().Error("scene is deleted")
		return &api.GetStoryBoardSenceGenerateResponse{
			Code:    -1,
			Message: "scene is deleted",
		}, nil
	}
	return &api.GetStoryBoardSenceGenerateResponse{
		Code:    0,
		Message: "OK",
		Data:    convert.ConvertStoryBoardSceneToApiStoryBoardScene(scene),
	}, nil
}

func (s *StoryService) GetStoryBoardGenerate(ctx context.Context, req *api.GetStoryBoardGenerateRequest) (*api.GetStoryBoardGenerateResponse, error) {
	// 1. 获取场景描述
	scenes, err := models.GetStoryBoardScenesByBoard(ctx, req.GetBoardId())
	if err != nil {
		log.Log().Error("get storyboard scene failed", zap.Error(err))
		return nil, err
	}
	if len(scenes) == 0 {
		log.Log().Error("scene not found")
		return &api.GetStoryBoardGenerateResponse{
			Code:    -1,
			Message: "scene not found",
		}, nil
	}
	total := len(scenes)
	generating := 0
	apiScenes := make([]*api.StoryBoardSence, 0)
	for _, scene := range scenes {
		if scene.Status == 1 {
			log.Log().Error("scene is already generating")
		}
		if scene.Status == 0 || scene.Status == 2 {
			generating++
		}
		if scene.Status == -1 {
			log.Log().Error("scene is deleted")
		}
		apiScenes = append(apiScenes, convert.ConvertStoryBoardSceneToApiStoryBoardScene(scene))
	}
	return &api.GetStoryBoardGenerateResponse{
		Code:            0,
		Message:         "OK",
		GeneratingStage: int32(total - generating),
		List:            apiScenes,
	}, nil
}

func (s *StoryService) LikeStoryRole(ctx context.Context, req *api.LikeStoryRoleRequest) (*api.LikeStoryRoleResponse, error) {
	err := models.LikeStoryRole(ctx, int(req.GetUserId()), req.GetStoryId(), req.GetRoleId())
	if err != nil {
		log.Log().Error("like story role failed", zap.Error(err))
		return nil, err
	}
	err = models.IncreaseStoryRoleLikeCount(ctx, req.GetRoleId(), 1)
	if err != nil {
		log.Log().Error("increase story role like count failed", zap.Error(err))
		return nil, err
	}
	return &api.LikeStoryRoleResponse{
		Code:    0,
		Message: "OK",
	}, nil
}

func (s *StoryService) UnLikeStoryRole(ctx context.Context, req *api.UnLikeStoryRoleRequest) (*api.UnLikeStoryRoleResponse, error) {
	err := models.UnLikeStoryRole(ctx, int(req.GetUserId()), req.GetStoryId(), req.GetRoleId())
	if err != nil {
		log.Log().Error("unlike story role failed", zap.Error(err))
		return nil, err
	}
	err = models.DecreaseStoryRoleLikeCount(ctx, req.GetRoleId(), 1)
	if err != nil {
		log.Log().Error("decrease story role like count failed", zap.Error(err))
		return nil, err
	}
	return &api.UnLikeStoryRoleResponse{
		Code:    0,
		Message: "OK",
	}, nil
}

func (s *StoryService) FollowStoryRole(ctx context.Context, req *api.FollowStoryRoleRequest) (*api.FollowStoryRoleResponse, error) {
	err := models.WatchStoryRole(ctx, int(req.GetUserId()), req.GetStoryId(), req.GetRoleId())
	if err != nil {
		log.Log().Error("follow story role failed", zap.Error(err))
		return nil, err
	}
	err = models.IncreaseStoryRoleFollowCount(ctx, req.GetRoleId(), 1)
	if err != nil {
		log.Log().Error("increase story role follow count failed", zap.Error(err))
		return nil, err
	}
	return &api.FollowStoryRoleResponse{
		Code:    0,
		Message: "OK",
	}, nil
}

func (s *StoryService) UnFollowStoryRole(ctx context.Context, req *api.UnFollowStoryRoleRequest) (*api.UnFollowStoryRoleResponse, error) {
	err := models.UnWatchStoryRole(ctx, int(req.GetUserId()), req.GetStoryId(), req.GetRoleId())
	if err != nil {
		log.Log().Error("unfollow story role failed", zap.Error(err))
		return nil, err
	}
	err = models.DecreaseStoryRoleFollowCount(ctx, req.GetRoleId(), 1)
	if err != nil {
		log.Log().Error("decrease story role follow count failed", zap.Error(err))
		return nil, err
	}
	return &api.UnFollowStoryRoleResponse{
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
		apiStories = append(apiStories, convert.ConvertStoryToApiStory(story))
	}
	return &api.SearchStoriesResponse{
		Code:    0,
		Message: "OK",
		Stories: apiStories,
		Total:   total,
	}, nil
}

func (s *StoryService) SearchRoles(ctx context.Context, req *api.SearchRolesRequest) (*api.SearchRolesResponse, error) {
	roles, total, err := models.GetStoryRolesByName(ctx, req.GetKeyword(), int(req.GetOffset()), int(req.GetPageSize()))
	if err != nil {
		log.Log().Error("get story roles failed", zap.Error(err))
		return nil, err
	}
	apiRoles := make([]*api.StoryRole, 0)
	for _, role := range roles {
		apiRoles = append(apiRoles, convert.ConvertStoryRoleToApiStoryRoleInfo(role))
	}
	return &api.SearchRolesResponse{
		Code:    0,
		Message: "OK",
		Roles:   apiRoles,
		Total:   total,
	}, nil
}

func (s *StoryService) RestoreStoryboard(ctx context.Context, req *api.RestoreStoryboardRequest) (*api.RestoreStoryboardResponse, error) {
	resp := &api.RestoreStoryboardResponse{}
	story, err := models.GetStory(ctx, req.GetStoryId())
	if err != nil {
		log.Log().Error("get story failed", zap.Error(err))
		return nil, err
	}
	if story == nil {
		log.Log().Error("story not found")
		resp.Code = -1
		resp.Message = "story not found"
		return resp, nil
	}

	storyboard, err := models.GetStoryboard(ctx, req.GetStoryboardId())
	if err != nil {
		log.Log().Error("get storyboard failed", zap.Error(err))
		return nil, err
	}
	if storyboard == nil {
		log.Log().Error("storyboard not found")
		resp.Code = -1
		resp.Message = "storyboard not found"
		return resp, nil
	}
	if storyboard.Stage == int(api.StoryboardStage_STORYBOARD_STAGE_PUBLISHED) {
		resp.Code = -1
		resp.Message = "storyboard is already published"
		return resp, nil
	}
	switch storyboard.Stage {
	case int(api.StoryboardStage_STORYBOARD_STAGE_CREATED):
		// 创建完故事剧情(故事板)，但是没有渲染剧情
	case int(api.StoryboardStage_STORYBOARD_STAGE_RENDERED):
		// 创建完故事剧情，但是没有渲染场景
		sences, err := models.GetStoryBoardScenesByBoard(ctx, req.GetStoryboardId())
		if err != nil {
			log.Log().Error("get storyboard scenes failed", zap.Error(err))
			return nil, err
		}
		if len(sences) < 0 {
			resp.Code = -1
			resp.Message = "storyboard has scenes"
			return resp, nil
		}
	case int(api.StoryboardStage_STORYBOARD_STAGE_GEN_IMAGE):
		// 创建完故事剧情以及场景，但是没有生成图片,正常剧情渲染
	case int(api.StoryboardStage_STORYBOARD_STAGE_GEN_VIDEO):
		// 创建完故事剧情以及场景，但是没有生成音频，建议只有点赞高的、关注多的角色、付费用户使用
	case int(api.StoryboardStage_STORYBOARD_STAGE_GEN_AUDIO):
		// 创建完故事剧情以及场景，但是没有生成音频，建议只有旁白使用
	case int(api.StoryboardStage_STORYBOARD_STAGE_GEN_TEXT):
		// 创建完故事剧情，但是没有创建场景描述
	case int(api.StoryboardStage_STORYBOARD_STAGE_FINISHED):
		// 已经创建完所有，但是没有发布
	}

	return resp, nil
}

// 获取用户创建的故事板
func (s *StoryService) GetUserCreatedStoryboards(ctx context.Context, req *api.GetUserCreatedStoryboardsRequest) (*api.GetUserCreatedStoryboardsResponse, error) {
	storyboards, total, err := models.GetUserCreatedStoryboardsWithStoryId(ctx, int(req.GetUserId()), int(req.GetStoryId()), int(req.GetOffset()), int(req.GetPageSize()))
	if err != nil {
		log.Log().Error("get user created storyboards failed", zap.Error(err))
		return nil, err
	}
	log.Log().Info("get user created storyboards", zap.Int("total", len(storyboards)))
	apiStoryboards := make([]*api.StoryBoard, 0)
	for idx, storyboard := range storyboards {
		log.Log().Info("get user created storyboard", zap.Int64("id", int64(storyboard.ID)), zap.Int("index", idx))
		newApiStoryboard := convert.ConvertStoryBoardToApiStoryBoard(storyboard)
		sences, err := models.GetStoryBoardScenesByBoard(ctx, int64(storyboard.ID))
		if err != nil {
			log.Log().Error("get storyboard scenes failed", zap.Error(err))
			continue
		}
		newApiStoryboard.Sences = &api.StoryBoardSences{
			List: make([]*api.StoryBoardSence, 0),
		}
		for _, scene := range sences {
			newApiStoryboard.Sences.List = append(newApiStoryboard.Sences.List, convert.ConvertStoryBoardSceneToApiStoryBoardScene(scene))
		}
		apiStoryboards = append(apiStoryboards, newApiStoryboard)
	}
	return &api.GetUserCreatedStoryboardsResponse{
		Code:        0,
		Message:     "OK",
		Storyboards: apiStoryboards,
		Total:       total,
	}, nil
}

// 获取用户创建的角色
func (s *StoryService) GetUserCreatedRoles(ctx context.Context, req *api.GetUserCreatedRolesRequest) (*api.GetUserCreatedRolesResponse, error) {
	roles, total, err := models.GetUserCreatedRolesWithStoryId(ctx, int(req.GetUserId()), int(req.GetStoryId()), int(req.GetOffset()), int(req.GetPageSize()))
	if err != nil {
		log.Log().Error("get user created roles failed", zap.Error(err))
		return nil, err
	}
	apiRoles := make([]*api.StoryRole, 0)
	for _, role := range roles {
		apiRoles = append(apiRoles, convert.ConvertStoryRoleToApiStoryRoleInfo(role))
	}
	return &api.GetUserCreatedRolesResponse{
		Code:    0,
		Message: "OK",
		Roles:   apiRoles,
		Total:   total,
	}, nil
}

func (s *StoryService) CreateStoryRole(ctx context.Context, req *api.CreateStoryRoleRequest) (*api.CreateStoryRoleResponse, error) {
	story, err := models.GetStory(ctx, req.GetRole().GetStoryId())
	if err != nil {
		return nil, err
	}
	if story.Status == -1 {
		return &api.CreateStoryRoleResponse{
			Code:    -1,
			Message: "story is closed",
		}, nil
	}
	role, err := models.GetStoryRoleByName(ctx, req.GetRole().GetCharacterName(), int64(story.ID))
	if err != nil && err != gorm.ErrRecordNotFound {
		return nil, err
	}
	if role != nil {
		return &api.CreateStoryRoleResponse{
			Code:    -1,
			Message: "role already exists",
		}, nil
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
	newRole.Status = 1
	_, err = models.CreateStoryRole(ctx, newRole)
	if err != nil {
		return nil, err
	}
	return &api.CreateStoryRoleResponse{
		Code:    0,
		Message: "OK",
	}, nil
}

func (s *StoryService) GetStoryRoleDetail(ctx context.Context, req *api.GetStoryRoleDetailRequest) (*api.GetStoryRoleDetailResponse, error) {
	role, err := models.GetStoryRoleByID(ctx, req.GetRoleId())
	if err != nil {
		log.Log().Error("get story role detail failed", zap.Error(err))
		return nil, err
	}
	return &api.GetStoryRoleDetailResponse{
		Code:    0,
		Message: "OK",
		Info: &api.StoryRole{
			RoleId:               int64(role.ID),
			CharacterDescription: role.CharacterDescription,
			CharacterName:        role.CharacterName,
			CharacterAvatar:      role.CharacterAvatar,
			CharacterId:          role.CharacterID,
			StoryId:              int64(role.StoryID),
			CharacterType:        role.CharacterType,
			CharacterPrompt:      role.CharacterPrompt,
			CharacterRefImages:   strings.Split(role.CharacterRefImages, ","),
			Ctime:                role.CreateAt.Unix(),
			Mtime:                role.UpdateAt.Unix(),
			CreatorId:            role.CreatorID,
			FollowCount:          role.FollowCount,
			LikeCount:            role.LikeCount,
			Status:               int32(role.Status),
			StoryboardNum:        role.StoryboardNum,
		},
	}, nil
}

func (s *StoryService) RenderStoryRole(ctx context.Context, req *api.RenderStoryRoleRequest) (*api.RenderStoryRoleResponse, error) {

	return nil, nil
}

// 获取角色故事
func (s *StoryService) GetStoryRoleStories(ctx context.Context, req *api.GetStoryRoleStoriesRequest) (*api.GetStoryRoleStoriesResponse, error) {
	return nil, nil
}

// 获取角色故事板
func (s *StoryService) GetStoryRoleStoryboards(ctx context.Context, req *api.GetStoryRoleStoryboardsRequest) (*api.GetStoryRoleStoryboardsResponse, error) {
	boards, err := models.GetStoryBoardsByRoleID(ctx, req.GetRoleId())
	if err != nil && err != gorm.ErrRecordNotFound {
		log.Log().Error("get story role storyboards failed", zap.Error(err))
		return nil, err
	}
	if err == gorm.ErrRecordNotFound {
		return &api.GetStoryRoleStoryboardsResponse{
			Code:    0,
			Message: "OK",
		}, nil
	}
	apiBoards := make([]*api.StoryBoard, 0)
	for _, board := range boards {
		apiBoards = append(apiBoards, convert.ConvertStoryBoardToApiStoryBoard(board))
	}
	return &api.GetStoryRoleStoryboardsResponse{
		Code:        0,
		Message:     "OK",
		Storyboards: apiBoards,
		Total:       int64(len(apiBoards)),
	}, nil
}

// 创建角色聊天
func (s *StoryService) CreateStoryRoleChat(ctx context.Context, req *api.CreateStoryRoleChatRequest) (*api.CreateStoryRoleChatResponse, error) {
	if req.GetUserId() == 0 || req.GetRoleId() == 0 {
		return nil, errors.New("invalid user id or role id")
	}
	existChatCtx, err := models.GetChatContextByUserIDAndRoleID(ctx, int64(req.GetUserId()), req.GetRoleId())
	if err != nil && err != gorm.ErrRecordNotFound {
		log.Log().Error("get user chat context failed", zap.Error(err))
		return nil, err
	}
	if existChatCtx != nil && existChatCtx.Status == 1 {
		return &api.CreateStoryRoleChatResponse{
			Code:    0,
			Message: "OK",
			ChatContext: &api.ChatContext{
				ChatId:         int64(existChatCtx.ID),
				UserId:         int64(existChatCtx.UserID),
				RoleId:         int64(existChatCtx.RoleID),
				LastUpdateTime: existChatCtx.UpdateAt.Unix(),
			},
		}, nil
	}
	chatContext := new(models.ChatContext)
	chatContext.UserID = int64(req.GetUserId())
	chatContext.RoleID = req.GetRoleId()
	chatContext.Title = "聊天消息"
	chatContext.Content = ""
	chatContext.Status = 1
	err = models.CreateChatContext(ctx, chatContext)
	if err != nil {
		log.Log().Error("create story role chat failed", zap.Error(err))
		return nil, err
	}
	return &api.CreateStoryRoleChatResponse{
		Code:    0,
		Message: "OK",
		ChatContext: &api.ChatContext{
			ChatId:         int64(chatContext.ID),
			UserId:         int64(chatContext.UserID),
			RoleId:         int64(chatContext.RoleID),
			LastUpdateTime: chatContext.UpdateAt.Unix(),
		},
	}, nil
}

// 角色聊天
func (s *StoryService) ChatWithStoryRole(ctx context.Context, req *api.ChatWithStoryRoleRequest) (*api.ChatWithStoryRoleResponse, error) {
	chatCtx, err := models.GetChatContextByUserIDAndRoleID(ctx, int64(req.Messages[0].GetUserId()), int64(req.Messages[0].GetRoleId()))
	if err != nil && err != gorm.ErrRecordNotFound {
		log.Log().Error("get user chat context failed", zap.Error(err))
		return nil, err
	}
	if err == gorm.ErrRecordNotFound {
		// 创建聊天上下文
		chatCtx = new(models.ChatContext)
		chatCtx.UserID = int64(req.Messages[0].GetUserId())
		chatCtx.RoleID = int64(req.Messages[0].GetRoleId())
		chatCtx.Title = "聊天消息"
		chatCtx.Content = ""
		chatCtx.Status = 1
		err = models.CreateChatContext(ctx, chatCtx)
		if err != nil {
			log.Log().Error("create story role chat failed", zap.Error(err))
			return nil, err
		}
	}
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
			log.Log().Error("create story role chat message failed", zap.Error(err))
			return nil, err
		}
		reply = append(reply, convert.ConvertChatMessageToApiChatMessage(chatMessage))
		{
			roleInfo, err := models.GetStoryRoleByID(ctx, int64(message.GetRoleId()))
			if err != nil {
				log.Log().Error("get story role by id failed", zap.Error(err))
				return nil, err
			}
			var chatParams = &client.ChatWithRoleParams{
				MessageContent: message.GetMessage(),
				Background:     roleInfo.CharacterDescription,
				SenseDesc:      "", // sence
				RolePositive:   "", // 角色的描述
				RoleNegative:   "",
				RequestId:      message.GetUuid(),
				UserId:         fmt.Sprintf("grapery_chat_ctx_%d_user_%d", chatCtx.ID, chatCtx.UserID),
			}
			chatResp, err := s.client.ChatWithRole(ctx, chatParams)
			if err != nil {
				log.Log().Error("chat with role failed", zap.Error(err))
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
				log.Log().Error("create story role chat message failed", zap.Error(err))
				return nil, err
			}
			reply = append(reply, convert.ConvertChatMessageToApiChatMessage(roleReplyMessage))
		}
	}
	return &api.ChatWithStoryRoleResponse{
		Code:          0,
		Message:       "OK",
		ReplyMessages: reply,
	}, nil
}

// 获取角色聊天列表
func (s *StoryService) GetUserWithRoleChatList(ctx context.Context, req *api.GetUserWithRoleChatListRequest) (*api.GetUserWithRoleChatListResponse, error) {
	log.Log().Info("get user with role chat list", zap.Any("req", req.String()))
	chatCtxs, total, err := models.GetChatContextByUserID(ctx, int64(req.GetUserId()), 0, 100)
	if err != nil {
		log.Log().Error("get user chat context failed", zap.Error(err))
		return nil, err
	}
	_ = total
	apiChatCtxs := make([]*api.ChatContext, 0)
	for _, chatCtx := range chatCtxs {
		user, err := models.GetUserById(ctx, int64(chatCtx.UserID))
		if err != nil {
			log.Log().Error("get user by id failed", zap.Error(err))
			return nil, err
		}
		role, err := models.GetStoryRoleByID(ctx, chatCtx.RoleID)
		if err != nil {
			log.Log().Error("get story role by id failed", zap.Error(err))
			return nil, err
		}
		lastMSg, err := models.GetChatContextLastMessage(ctx, int64(chatCtx.ID))
		if err != nil && err != gorm.ErrRecordNotFound {
			log.Log().Error("get last chat message failed", zap.Error(err))
			return nil, err
		}
		if lastMSg == nil {
			lastMSg = &models.ChatMessage{
				ChatContextID: int64(chatCtx.ID),
				Sender:        0,
			}
		}
		chatCtx := &api.ChatContext{
			ChatId:         int64(chatCtx.ID),
			UserId:         int64(chatCtx.UserID),
			RoleId:         int64(chatCtx.RoleID),
			Timestamp:      chatCtx.CreateAt.Unix(),
			LastUpdateTime: chatCtx.UpdateAt.Unix(),
			LastMessage:    convert.ConvertChatMessageToApiChatMessage(lastMSg),
			User:           convert.ConvertUserToApiUser(user),
			Role:           convert.ConvertStoryRoleToApiStoryRoleInfo(role),
		}
		apiChatCtxs = append(apiChatCtxs, chatCtx)
	}
	return &api.GetUserWithRoleChatListResponse{
		Code:    0,
		Message: "OK",
		Chats:   apiChatCtxs,
	}, nil
}

// 更新角色详情
func (s *StoryService) UpdateStoryRoleDetail(ctx context.Context, req *api.UpdateStoryRoleDetailRequest) (*api.UpdateStoryRoleDetailResponse, error) {
	log.Log().Info("update story role detail", zap.Any("req", req.String()))
	role, err := models.GetStoryRoleByID(ctx, req.GetRoleId())
	if err != nil {
		log.Log().Error("get story role detail failed", zap.Error(err))
		return nil, err
	}
	if role == nil {
		return &api.UpdateStoryRoleDetailResponse{
			Code:    -1,
			Message: "role not found",
		}, nil
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
		updates["character_prompt"] = req.GetRole().GetCharacterPrompt()
	}
	if len(req.GetRole().GetCharacterRefImages()) > 0 {
		updates["character_ref_images"] = strings.Join(req.GetRole().GetCharacterRefImages(), ",")
	}
	err = models.UpdateStoryRole(ctx, int64(role.ID), updates)
	if err != nil {
		log.Log().Error("update story role detail failed", zap.Error(err))
		return nil, err
	}
	return nil, nil
}

func (s *StoryService) GetUserChatWithRole(ctx context.Context, req *api.GetUserChatWithRoleRequest) (*api.GetUserChatWithRoleResponse, error) {
	if req.GetUserId() == 0 || req.GetRoleId() == 0 {
		return nil, errors.New("invalid user id or role id")
	}
	chatCtx, err := models.GetChatContextByUserIDAndRoleID(ctx, int64(req.GetUserId()), req.GetRoleId())
	if err != nil && err != gorm.ErrRecordNotFound {
		log.Log().Error("get user chat context failed", zap.Error(err))
		return nil, err
	}
	if err == gorm.ErrRecordNotFound {
		return &api.GetUserChatWithRoleResponse{
			Code:    1,
			Message: "chat context not found",
		}, nil
	}
	if chatCtx == nil {
		return &api.GetUserChatWithRoleResponse{
			Code:    1,
			Message: "chat context not found",
		}, nil
	}
	user, err := models.GetUserById(ctx, int64(chatCtx.UserID))
	if err != nil {
		log.Log().Error("get user by id failed", zap.Error(err))
		return nil, err
	}
	role, err := models.GetStoryRoleByID(ctx, chatCtx.RoleID)
	if err != nil {
		log.Log().Error("get story role by id failed", zap.Error(err))
		return nil, err
	}
	lastMSg, err := models.GetChatContextLastMessage(ctx, int64(chatCtx.ID))
	if err != nil && err != gorm.ErrRecordNotFound {
		log.Log().Error("get last chat message failed", zap.Error(err))
		return nil, err
	}
	if lastMSg == nil {
		lastMSg = &models.ChatMessage{
			ChatContextID: int64(chatCtx.ID),
			Sender:        0,
		}
	}
	return &api.GetUserChatWithRoleResponse{
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
	}, nil
}

func (s *StoryService) GetUserChatMessages(ctx context.Context, req *api.GetUserChatMessagesRequest) (*api.GetUserChatMessagesResponse, error) {
	if req.GetChatId() == 0 && req.GetUserId() == 0 && req.GetRoleId() == 0 {
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
			log.Log().Error("get user chat messages failed", zap.Error(err))
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
			log.Log().Error("get role chat messages failed", zap.Error(err))
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
			log.Log().Error("get chat context chat messages failed", zap.Error(err))
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
	return &api.GetUserChatMessagesResponse{
		Code:      0,
		Message:   "OK",
		Timestamp: lastTimestamp,
		Total:     int64(total),
		Messages:  apiChatMsgs,
	}, nil
}
