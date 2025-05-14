package story

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"

	api "github.com/grapery/common-protoc/gen"
	"github.com/grapery/grapery/models"
	"github.com/grapery/grapery/pkg/active"
	"github.com/grapery/grapery/pkg/client"
	"github.com/grapery/grapery/pkg/cloud/aliyun"
	"github.com/grapery/grapery/utils"
	"github.com/grapery/grapery/utils/convert"
	"github.com/grapery/grapery/utils/log"
	"github.com/grapery/grapery/utils/prompt"
)

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
	log.Log().Info("create storyboard success", zap.Int64("storyBoardId", storyBoardId))
	newStroyBoard.ID = uint(storyBoardId)
	if storyInfo.RootBoardID == 0 {
		err = models.UpdateStorySpecColumns(ctx, req.Board.StoryId, map[string]interface{}{
			"root_board_id": storyBoardId,
		})
		if err != nil {
			return nil, err
		}
	}
	if len(req.GetBoard().GetRoles()) > 0 {
		for _, role := range req.GetBoard().GetRoles() {
			roleInfo := new(models.StoryBoardRole)
			roleInfo.BoardId = storyBoardId
			roleInfo.RoleId = role.RoleId
			roleInfo.Name = role.CharacterName
			roleInfo.Avatar = role.CharacterAvatar
			roleInfo.StoryId = req.GetBoard().GetStoryId()
			roleInfo.CreatorId = req.GetBoard().GetCreator()
			roleInfo.Status = 1
			roleInfo.IsMain = 0
			roleInfo.IsPublished = 0
			_, err = models.CreateStoryBoardRole(ctx, roleInfo)
			if err != nil {
				return nil, err
			}
		}
	}
	userProfile := &models.UserProfile{
		UserId: int64(req.GetBoard().GetCreator()),
	}
	err = userProfile.IncrementCreatedBoardNum()
	if err != nil {
		log.Log().Error("increment created board num failed", zap.Error(err))
	}
	group := &models.Group{}
	group.ID = uint(storyInfo.GroupID)
	err = group.GetByID()
	if err != nil {
		return nil, err
	} else {
		active.GetActiveServer().WriteStoryActive(ctx, group, storyInfo, newStroyBoard,
			nil, req.GetBoard().GetCreator(), api.ActiveType_NewStoryBoard)
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

	cu, err := s.GetStoryboardCurrentUserStatus(ctx, req.BoardId)
	if err != nil {
		log.Log().Error("get storyboard current user status failed", zap.Error(err))
	}
	creator, err := models.GetUserById(ctx, int64(boardInfo.CreatorID))
	if err != nil {
		return nil, err
	}
	board.CurrentUserStatus = cu
	boardActive := &api.StoryBoardActive{
		Storyboard:        board,
		TotalLikeCount:    int64(boardInfo.LikeNum),
		TotalCommentCount: int64(boardInfo.CommentNum),
		TotalShareCount:   int64(boardInfo.ShareNum),
		TotalForkCount:    int64(boardInfo.ForkNum),
		Mtime:             boardInfo.UpdateAt.Unix(),
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
	return &api.GetStoryboardResponse{
		Code:    0,
		Message: "OK",
		Data: &api.GetStoryboardResponse_Data{
			BoardInfo: boardActive,
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
	return &api.UpdateStoryboardResponse{
		Code:    0,
		Message: "OK",
	}, nil
}

func (s *StoryService) GetStoryboards(ctx context.Context, req *api.GetStoryboardsRequest) (resp *api.GetStoryboardsResponse, err error) {
	boardList, err := models.GetStoryboardsByStoryMultiPage(ctx, req.StoryId, int(req.Page), int(req.PageSize))
	if err != nil {
		log.Log().Error("get storyboards by story multi page failed", zap.Error(err))
		return nil, err
	}
	story, err := models.GetStory(ctx, req.StoryId)
	if err != nil {
		log.Log().Error("get story failed", zap.Error(err))
		return nil, err
	}
	srcBoardMap := make(map[int64]*models.StoryBoard)
	apiBoardsActive := make([]*api.StoryBoardActive, 0)
	for _, board := range boardList {
		sences, err := models.GetStoryBoardScenesByBoard(ctx, int64(board.ID))
		if err != nil {
			log.Log().Error("get board sences failed", zap.Error(err))
		}
		srcBoardMap[int64(board.ID)] = board
		boardInfo := ConvertStoryBoardToApiStoryBoard(board)
		if len(sences) != 0 {
			boardInfo.Sences = new(api.StoryBoardSences)
			for _, scene := range sences {
				boardInfo.Sences.List = append(boardInfo.Sences.List, ConvertStorySceneToApiScene(scene))
			}
			boardInfo.Sences.Total = int64(len(boardInfo.Sences.List))
		}
		cu, err := s.GetStoryboardCurrentUserStatus(ctx, int64(board.ID))
		if err != nil {
			log.Log().Error("get storyboard current user status failed", zap.Error(err))
		}
		boardInfo.CurrentUserStatus = cu
		creator, err := models.GetUserById(ctx, board.CreatorID)
		if err != nil {
			log.Log().Error("get story creator failed", zap.Error(err))
		}
		roles, err := models.GetStoryBoardRolesByBoard(ctx, int64(board.ID))
		if err != nil {
			log.Log().Error("get storyboard roles failed", zap.Error(err))
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
			Isliked: true,
			Mtime:   boardInfo.Mtime,
		}
		apiBoardsActive = append(apiBoardsActive, apiBoardsActiveItem)
	}
	return &api.GetStoryboardsResponse{
		Code:    0,
		Message: "OK",
		Data: &api.GetStoryboardsResponse_Data{
			List:  apiBoardsActive,
			Total: int64(len(apiBoardsActive)),
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
	userProfile := &models.UserProfile{
		UserId: int64(currentBoard.CreatorID),
	}
	err = userProfile.DecrementCreatedBoardNum()
	if err != nil {
		log.Log().Error("decrement created board num failed", zap.Error(err))
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
	newStoryBoard.ID = 0
	newStoryBoard.CreatorID = req.GetUserId()
	newStoryBoard.CreateAt = time.Now()
	newStoryBoard.UpdateAt = time.Now()
	id, err := models.CreateStoryBoard(ctx, newStoryBoard)
	if err != nil {
		log.Log().Error("create new story board failed", zap.Error(err))
		return nil, err
	}
	story, err := models.GetStory(ctx, originStoryBoard.StoryID)
	if err != nil {
		return nil, err
	}
	group := &models.Group{}
	group.ID = uint(story.GroupID)
	err = group.GetByID()
	if err != nil {
		return nil, err
	} else {
		active.GetActiveServer().WriteStoryActive(ctx, group, story, newStoryBoard,
			nil, req.GetUserId(), api.ActiveType_ForkStory)
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
	storyBoard, err := models.GetStoryboard(ctx, req.BoardId)
	if err != nil {
		return nil, err
	}
	story, err := models.GetStory(ctx, storyBoard.StoryID)
	if err != nil {
		return nil, err
	}
	item := new(models.LikeItem)
	item.UserID = req.GetUserId()
	item.GroupID = int64(story.GroupID)
	item.StoryID = req.GetStoryId()
	item.StoryboardId = req.GetBoardId()
	item.LikeItemType = models.LikeItemTypeStoryboard
	item.LikeType = models.LikeTypeLike
	err = models.CreateLikeStoryBoardItem(ctx, item)
	if err != nil {
		log.Log().Error("create like item failed", zap.Error(err))
		return nil, err
	}

	group := &models.Group{}
	group.ID = uint(story.GroupID)
	err = group.GetByID()
	if err != nil {
		return nil, err
	} else {
		active.GetActiveServer().WriteStoryActive(ctx, group, story, storyBoard,
			nil, req.GetUserId(), api.ActiveType_LikeStory)
	}
	storyBoard.LikeNum++
	err = models.UpdateStoryboard(ctx, storyBoard)
	if err != nil {
		return nil, err
	}
	userProfile := &models.UserProfile{
		UserId: int64(storyBoard.CreatorID),
	}
	err = userProfile.IncrementLikedStoryNum()
	if err != nil {
		log.Log().Error("increment liked story num failed", zap.Error(err))
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

func (s *StoryService) RenderStoryboard(ctx context.Context, req *api.RenderStoryboardRequest) (*api.RenderStoryboardResponse, error) {
	// 获取故事白板
	board, err := models.GetStoryboard(ctx, req.GetBoardId())
	if err != nil {
		return nil, err
	}
	// 获取故事
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
	// 获取故事板生成记录
	stroyGen, err := models.GetStoryGensByStoryBoard(ctx, req.GetBoardId(), 1)
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
	// 故事全局风格
	imageStyle := storyParam.Style
	if imageStyle == "" {
		imageStyle = "Ghibli style"
	}

	storyRoles, err := models.GetStoryBoardRoles(ctx, req.GetBoardId())
	if err != nil {
		log.Log().Error("get story board roles failed", zap.Error(err))
	}
	roleIds := make([]int64, 0)

	storyRolesStr := ""
	if len(storyRoles) != 0 {
		for _, role := range storyRoles {
			roleIds = append(roleIds, role.RoleId)
		}
		roles, err := models.GetStoryRolesByIDs(ctx, roleIds)
		if err != nil {
			log.Log().Error("get story roles failed", zap.Error(err))
		}
		for _, role := range roles {
			storyRolesStr += "角色id:" + fmt.Sprintf("%d", role.ID) + "," + "角色姓名:" + role.CharacterName + "," + "角色描述:" + role.CharacterDescription + ";\n"
		}
	} else {
		log.Log().Error("get story board roles failed", zap.Error(err))
	}
	storyCharacters := strings.Replace(`参与人物为: """story_characters"""。`, "story_characters", storyRolesStr, -1)

	templatePrompt := `
	为故事章节的 """story_chapter""" 章节的生成详细故事情节细节，请参考故事剧情: """story_content"""。
	故事背景为: """story_background"""。`
	if len(storyRoles) != 0 {
		templatePrompt = storyCharacters + templatePrompt
	}
	templatePrompt = templatePrompt +
		`同时衔接前后章节的情节,上一章节的故事情节为: """story_content"""，生成符合上下文的、合理的、更详细的情节，
	可以生成4-6个故事的细节，以及生成可以展示这些故事剧情的图片 prompt 提示词，图片提示词的风格统一为"""imageStyle"""。
	以json格式返回格式可以参考如下例子:
	--------
		{
			"章节情节简述": {
				"章节题目": "地球生存环境恶化",
				"章节内容": "地球资源日益枯竭，人类将目光投向了火星。我国成功组建了一支马克为首的精英宇航员队伍，肩负起在火星建立基地的重任，为地球移民做准备"
			},
			"章节详细情节": [
				{
					"情节id": "1",
					"情节内容": "气候变化，温室效应加剧，全球平均气温上升超过2摄氏度，极端天气事件频发，如飓风、干旱、洪水等",
					"参与人物": [
						{
							"角色id": "1",
							"角色姓名": "马克",
							"角色描述": "马克是一名经验丰富的宇航员，曾多次执行太空任务，对火星环境有深入了解。"
						},
						{
							"角色id": "2",
							"角色姓名": "飞云",
							"角色描述": "飞云是一名经验丰富的宇航员，曾多次执行太空任务，对火星环境有深入了解。"
						},
					],
					"图片提示词": "一个城市被严重的雾霾笼罩，天空灰暗，远处的高楼大厦若隐若现，人们戴着口罩匆匆行走，街道上的车辆行驶缓慢，整个场景透露出压抑和不安。"
				},
				{
					"情节id": "2",
					"情节内容": "资源枯竭，可耕地减少，粮食产量下降，粮食危机日益严重；淡资源匮乏，多地出现用水紧张状况；矿产资源开采难度加大，能源供应紧张。",
					"参与人物": [
						{
							"角色id": "1",
							"角色姓名": "马克",
							"角色描述": "马克是一名经验丰富的宇航员，曾多次执行太空任务，对火星环境有深入了解。"
						},
					],
					"图片提示词": "一片荒芜的农田，土壤干裂，庄稼枯萎，农民面露愁容地看着土地，天空中没有云彩，烈日炎炎，展现出粮食危机的严峻景象"
				}
			]
		}
	--------
	请保证故事的连贯，以及故事中的各个人物的角色前后一致，同时和故事背景契合，人物的描述清晰，情节人物的性格明显，场景描述详细，图片提示词准确。
	请注意，如果没有要求 ‘参与人物’ 的输出要求，可以不用输出 ‘参与的人物’ 的 json 数据结构。
	有些章节是没有故事角色的，可以不用为这个章节关联人物。所以要遵守这一条，只有当输入有 ‘参与人物’ 的列表时
	再输出这些 ’参与人物‘ 信息，这一点很重要，不要自作主张的引入角色或者故事人物。
	`
	templatePrompt = strings.Replace(templatePrompt, "story_chapter", board.Title, -1)
	templatePrompt = strings.Replace(templatePrompt, "imageStyle", imageStyle, -1)
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
	result := new(StoryChapter)
	start := time.Now()
	ret, err := s.client.GenStoryBoardInfo(ctx, renderStoryParams)
	if err != nil {
		log.Log().Error("gen storyboard info failed", zap.Error(err))
		return nil, err
	}
	cleanResult := utils.CleanLLmJsonResult(ret.Content)
	fmt.Println("render storyboard cleanResult: ", cleanResult)
	// 保存生成的故事板

	err = json.Unmarshal([]byte(cleanResult), &result)
	if err != nil {
		log.Log().Error("unmarshal story gen result failed", zap.Error(err))
		return nil, err
	}
	// 渲染剧情
	renderDetail := new(api.RenderStoryboardDetail)
	renderDetail.RenderType = req.RenderType
	renderDetail.Timecost = int32(time.Since(start).Seconds())
	renderDetail.BoardId = req.BoardId
	renderDetail.StoryId = req.StoryId
	renderDetail.UserId = req.UserId
	renderDetail.Result = new(api.StoryChapter)

	// Convert from StoryChapter to api.StoryChapter
	storyChapter := &api.StoryChapter{
		ChapterSummary: &api.ChapterSummary{
			Title:   result.ChapterSummary.Title,
			Content: result.ChapterSummary.Content,
		},
		ChapterDetailInfo: &api.ChapterDetailInformation{
			Details: make([]*api.DetailScene, 0),
		},
	}

	// Convert each detail scene
	for _, detail := range result.ChapterDetailInfo {
		apiDetail := &api.DetailScene{
			Id:          detail.ID,
			Content:     detail.Content,
			ImagePrompt: detail.ImagePrompt,
			Characters:  make([]*api.Character, 0),
		}

		// Convert characters
		for _, char := range detail.Characters {
			apiChar := &api.Character{
				Id:          char.ID,
				Name:        char.Name,
				Description: char.Description,
			}
			apiDetail.Characters = append(apiDetail.Characters, apiChar)
		}
		storyChapter.ChapterDetailInfo.Details = append(storyChapter.ChapterDetailInfo.Details, apiDetail)
	}

	renderDetail.Result = storyChapter

	renderDetailData, _ := json.Marshal(renderDetail)
	storyGen.Content = string(renderDetailData)
	storyGen.FinishTime = time.Now().Unix()
	err = models.UpdateStoryGen(ctx, storyGen)
	if err != nil {
		log.Log().Error("update story gen failed", zap.Error(err))
	}
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
						preDefineTemplate := strings.Replace(models.PreDefineTemplateEnVersion[1].Prompt, "prompt", subva.(string), -1)
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
						aliyunUrls := make([]string, 0)
						for _, imageUrl := range ret.ImageUrls {
							aliyunClient := aliyun.GetGlobalClient()
							aliyunUrl, err := aliyunClient.UploadFileFromURL("", imageUrl)
							if err != nil {
								log.Log().Error("upload file from url failed", zap.Error(err))
								continue
							}
							// aliyunThumbnailUrl, err := aliyunClient.GenerateThumbnailV2(aliyunUrl, 200)
							// if err != nil {
							// 	log.Log().Error("generate thumbnail failed", zap.Error(err))
							// 	continue
							// }
							// aliyunUrls = append(aliyunUrls, aliyunUrl, aliyunThumbnailUrl)
							aliyunUrls = append(aliyunUrls, aliyunUrl)
						}
						storyGen.ImageUrls = strings.Join(aliyunUrls, ",")
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
	board, err := models.GetStoryboard(ctx, req.GetBoardId())
	if err != nil {
		return nil, err
	}
	story, err := models.GetStory(ctx, board.StoryID)
	if err != nil {
		return nil, err
	}
	if story.Status == -1 {
		return &api.GenStoryboardTextResponse{
			Code:    -1,
			Message: "story is closed",
		}, nil
	}
	storyGen, err := models.GetStoryGensByStoryBoard(ctx, req.GetBoardId(), 1)
	if err != nil {
		return nil, err
	}
	if len(storyGen) == 0 {
		return &api.GenStoryboardTextResponse{
			Code:    -1,
			Message: "storyboard is not rendering",
		}, nil
	}
	storyGenContent, err := json.Marshal(storyGen[0].Content)
	if err != nil {
		return nil, err
	}
	_ = storyGenContent
	return &api.GenStoryboardTextResponse{
		Code:    0,
		Message: "OK",
		Data:    nil,
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

func (s *StoryService) InitRenderStory(ctx context.Context, req *api.ContinueRenderStoryRequest) (*api.ContinueRenderStoryResponse, error) {
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
	roles := req.GetRoles()
	originRoles, err := models.GetStoryRole(ctx, int64(story.ID))
	if err != nil {
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
		log.Log().Sugar().Info("role: %+v", role)
		if realRole, ok := originRolesMap[int64(role.RoleId)]; ok {
			finalRols[role.CharacterName] = realRole
		}
	}
	var rolesPrompt = make([]Character, 0)
	for _, role := range finalRols {
		rolePrompt := Character{
			ID:          fmt.Sprintf("%d", role.ID),
			Name:        role.CharacterName,
			Description: role.CharacterDescription,
		}
		rolesPrompt = append(rolesPrompt, rolePrompt)
	}
	storyGen := new(models.StoryGen)
	storyGen.Uuid = uuid.New().String()
	boardRequire := make(map[string]interface{})
	boardRequire["章节题目要求"] = req.GetTitle()
	boardRequire["章节内容要求"] = req.GetDescription()
	boardRequire["章节背景简介"] = req.GetBackground()
	if len(finalRols) != 0 {
		boardRequire["章节的角色信息"] = rolesPrompt
	}
	templatePrompt := `生成故事 story_name 的第一个章节,故事内容用中文描述,以json格式返回		
		选择的人员角色，不要超过 章节的角色信息' 规定的角色id和角色名称范围。
		一定要遵守 章节的角色信息' 的要求，其中'角色id'不要超出'章节的角色信息'里的范围。
		如果没有 '章节的角色信息'，或者 '章节的角色信息' 里没有角色id和角色名称的信息，那么就不要生成角色信息，直接返回故事内容，这一点很重要，不要自作主张的引入角色或者故事人物。
		参考如下格式生成内容：
		--------
		{
			"章节情节简述": {
				"章节题目": "地球生存环境恶化",
				"章节内容": "地球资源日益枯竭，人类将目光投向了火星。我国成功组建了一支马克为首的精英宇航员队伍，肩负起在火星建立基地的重任，为地球移民做准备"
			},
			"参与人物": [
					{
						"角色id": "1",
						"角色姓名": "马克",
						"角色描述": "马克是一名经验丰富的宇航员，曾多次执行太空任务，对火星环境有深入了解。"
					},
					{
						"角色id": "2",
						"角色姓名": "飞云",
						"角色描述": "飞云是一名经验丰富的宇航员，曾多次执行太空任务，对火星环境有深入了解。"
					},
			]
		}
		--------
		`
	boardRequireJson, _ := json.Marshal(boardRequire)
	if len(req.GetDescription()) > 0 ||
		len(req.GetBackground()) > 0 ||
		len(rolesPrompt) > 0 {
		templatePrompt = templatePrompt + `请一定要按照章节要求：
		-------- \n"` +
			string(boardRequireJson) +
			`\n --------\n`
	}
	templatePrompt = templatePrompt + `请保证故事的连贯，以及故事中的各个人物的角色前后一致,行为描述一致。输出的数据结构和输入的保持一致`
	templatePrompt = strings.Replace(templatePrompt, "story_name", story.Title, -1)
	storyGen.LLmPlatform = "Zhipu"
	storyGen.NegativePrompt = ""
	storyGen.PositivePrompt = templatePrompt
	storyGen.Regen = 1
	storyGen.Params = ""
	storyGen.OriginID = req.GetStoryId()
	storyGen.StartTime = time.Now().Unix()
	storyGen.BoardID = req.GetPrevBoardId()
	storyGen.GenType = int(req.GetRenderType())
	storyGen.TaskType = 1
	_, err = models.CreateStoryGen(ctx, storyGen)
	if err != nil {
		log.Log().Error("create storyboard gen failed", zap.Error(err))
		return nil, err
	}
	log.Log().Sugar().Info("gen storyboard prompt: ", templatePrompt)
	renderStoryParams := &client.StoryInfoParams{
		Content: templatePrompt,
	}
	result := new(StoryChapterV2)
	start := time.Now()
	ret, err := s.client.GenStoryInfo(ctx, renderStoryParams)
	if err != nil {
		log.Log().Error("gen storyboard info failed", zap.Error(err))
		return nil, err
	}
	// 保存生成的故事板
	cleanResult := utils.CleanLLmJsonResult(ret.Content)
	err = json.Unmarshal([]byte(cleanResult), &result)
	if err != nil {
		log.Log().Error("unmarshal story gen result failed", zap.Error(err))
		return nil, err
	}
	// 渲染剧情
	renderDetail := new(api.RenderStoryboardDetail)
	renderDetail.RenderType = req.RenderType
	renderDetail.Timecost = int32(time.Since(start).Seconds())
	renderDetail.Result = new(api.StoryChapter)

	// Convert from StoryChapter to api.StoryChapter
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
		storyChapter.ChapterSummary.Characters = append(storyChapter.ChapterSummary.Characters, characterInfo)
	}

	renderDetail.Result = storyChapter

	renderDetailData, _ := json.Marshal(renderDetail)
	storyGen.Content = string(renderDetailData)
	storyGen.FinishTime = time.Now().Unix()
	err = models.UpdateStoryGen(ctx, storyGen)
	if err != nil {
		log.Log().Error("update story gen failed", zap.Error(err))
	}
	return &api.ContinueRenderStoryResponse{
		Code:    0,
		Message: "OK",
		Data:    renderDetail,
	}, nil
}

func (s *StoryService) ContinueRenderStory(ctx context.Context, req *api.ContinueRenderStoryRequest) (*api.ContinueRenderStoryResponse, error) {
	log.Log().Sugar().Info("continue render story", zap.Any("req", req))
	if req.PrevBoardId <= 0 {
		log.Log().Sugar().Info("continue render story is init")
		return s.InitRenderStory(ctx, req)
	}
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
	roles := req.GetRoles()
	originRoles, err := models.GetStoryRole(ctx, int64(story.ID))
	if err != nil {
		return nil, err
	}
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
			finalRols[role.CharacterName] = realRole
		}
	}
	originRolesMapData, _ := json.Marshal(originRolesMap)
	fmt.Printf("originRolesMap roles: %v \n", string(originRolesMapData))
	rolesMapData, _ := json.Marshal(rolesMap)
	fmt.Printf("rolesMap roles: %v \n", string(rolesMapData))
	finalRolsData, err := json.Marshal(finalRols)
	fmt.Printf("finalRols roles: %v \n", string(finalRolsData))
	var rolesPrompt = make([]Character, 0)
	for _, role := range finalRols {
		rolePrompt := Character{
			ID:          fmt.Sprintf("%d", role.ID),
			Name:        role.CharacterName,
			Description: role.CharacterDescription,
		}
		rolesPrompt = append(rolesPrompt, rolePrompt)
	}
	//

	var boardIdtemp int64 = board.PrevId
	for boardIdtemp > 0 {
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
	if len(finalRols) != 0 {
		boardRequire["章节的角色信息"] = rolesPrompt
	}

	boardRequireJson, _ := json.Marshal(boardRequire)
	fmt.Printf("boardRequireJson: %v \n", string(boardRequireJson))
	templatePrompt := `生成故事 story_name 的下一个章节,故事内容用中文描述,以json格式返回		
		之前的故事章节:
		--------
		story_prev_content
		--------
		请参考以上故事章节，生成故事的下一个章节。所选择的人员角色，不要超过 '章节的角色信息' 规定的范围。
		章节要求：
		--------
		story_chapter_require
		--------
		一定要遵守 '章节的角色信息' 的要求，其中'角色id'要符合'章节的角色信息'里的限制。
		如果没有 '章节的角色信息'，或者 '章节的角色信息' 里没有角色id和角色名称的信息，就不要生成角色信息，只需要返回故事内容.
		参考如下格式生成内容：
		--------
		{
			"章节情节简述": {
				"章节题目": "地球生存环境恶化",
				"章节内容": "地球资源日益枯竭，人类将目光投向了火星。我国成功组建了一支马克为首的精英宇航员队伍，肩负起在火星建立基地的重任，为地球移民做准备"
			},
			"参与人物": [
					{
						"角色id": "1",
						"角色姓名": "马克",
						"角色描述": "马克是一名经验丰富的宇航员，曾多次执行太空任务，对火星环境有深入了解。"
					},
					{
						"角色id": "2",
						"角色姓名": "飞云",
						"角色描述": "飞云是一名经验丰富的宇航员，曾多次执行太空任务，对火星环境有深入了解。"
					},
			]
		}
		--------
		以上输出的json结果中，参与人物的 ""角色id"" 一定要在 '章节的角色信息' 中存在，如果不存在，在这个故事章节中，这个角色不应该参与生成。
		不要擅自引入新的故事人物和角色，故事的人物和角色，只能通过 '章节的角色信息' 参数中获取。
	`
	templatePrompt = strings.Replace(templatePrompt, "story_chapter_require", string(boardRequireJson), -1)

	templatePrompt = templatePrompt + `请保证故事的连贯，以及故事中的各个人物的角色前后一致,行为描述一致。输出的数据结构和输入的保持一致`
	story_prev_content := make([]*StoryChapterV2, 0)
	for idx := len(prevBoards) - 1; idx >= 0; idx-- {
		prevBoard := prevBoards[idx]
		content := new(StoryChapterV2)
		content.ChapterSummary.Title = prevBoard.Title
		content.ChapterSummary.Content = prevBoard.Description
		content.ChapterSummary.Characters = make([]Character, 0)
		storyRoles, err := models.GetStoryBoardRoles(ctx, int64(prevBoard.ID))
		if err != nil {
			log.Log().Error("get story board roles failed", zap.Error(err))
			return nil, err
		}
		for _, role := range storyRoles {
			roleInfo := new(Character)
			roleInfo.ID = fmt.Sprintf("%d", role.RoleId)
			roleInfo.Name = role.Name
			roleInfo.Description = role.Desc
			content.ChapterSummary.Characters = append(content.ChapterSummary.Characters, *roleInfo)
		}
		story_prev_content = append(story_prev_content, content)
	}
	story_prev_content_json, _ := json.Marshal(story_prev_content)
	templatePrompt = strings.Replace(templatePrompt, "story_name", story.Name, -1)
	templatePrompt = strings.Replace(templatePrompt, "story_prev_content", string(story_prev_content_json), -1)

	storyGen.LLmPlatform = "Zhipu"
	storyGen.NegativePrompt = ""
	storyGen.PositivePrompt = templatePrompt
	storyGen.Regen = 1
	storyGen.Params = string(storyGenData)
	storyGen.OriginID = req.GetStoryId()
	storyGen.StartTime = time.Now().Unix()
	storyGen.BoardID = req.GetPrevBoardId()
	storyGen.GenType = int(req.GetRenderType())
	storyGen.TaskType = 1
	_, err = models.CreateStoryGen(ctx, storyGen)
	if err != nil {
		log.Log().Error("create storyboard gen failed", zap.Error(err))
		return nil, err
	}
	log.Log().Sugar().Info("gen storyboard prompt: ", templatePrompt)
	renderStoryParams := &client.StoryInfoParams{
		Content: templatePrompt,
	}
	result := new(StoryChapterV2)
	start := time.Now()
	ret, err := s.client.GenStoryInfo(ctx, renderStoryParams)
	if err != nil {
		log.Log().Error("gen storyboard info failed", zap.Error(err))
		return nil, err
	}
	fmt.Printf("ret.Content: %s\n", ret.Content)
	// 保存生成的故事板
	cleanResult := utils.CleanLLmJsonResult(ret.Content)
	err = json.Unmarshal([]byte(cleanResult), &result)
	if err != nil {
		log.Log().Error("unmarshal story gen result failed", zap.Error(err))
		return nil, err
	}
	// 渲染剧情
	renderDetail := new(api.RenderStoryboardDetail)
	renderDetail.RenderType = req.RenderType
	renderDetail.Timecost = int32(time.Since(start).Seconds())
	renderDetail.Result = new(api.StoryChapter)
	resultData, err := json.Marshal(result)
	log.Log().Sugar().Info("result: ", string(resultData))
	// Convert from StoryChapter to api.StoryChapter
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
		storyChapter.ChapterSummary.Characters = append(storyChapter.ChapterSummary.Characters, characterInfo)
	}

	renderDetail.Result = storyChapter

	renderDetailData, _ := json.Marshal(renderDetail)
	log.Log().Sugar().Info("renderDetailData: ", string(renderDetailData))
	storyGen.Content = string(renderDetailData)
	storyGen.FinishTime = time.Now().Unix()
	err = models.UpdateStoryGen(ctx, storyGen)
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
	story, err := models.GetStory(ctx, req.GetStoryId())
	if err != nil {
		return nil, err
	}
	if story.Status == -1 {
		return &api.RenderStoryRolesResponse{
			Code:    -1,
			Message: "story is closed",
		}, nil
	}
	roles, err := models.GetStoryRole(ctx, int64(story.ID))
	if err != nil {
		return nil, err
	}
	log.Log().Sugar().Infof("story [%d] roles: %v", story.ID, roles)

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

	role, err := models.GetStoryRoleByID(ctx, req.GetRoleId())
	if err != nil {
		log.Log().Error("get story role failed", zap.Error(err))
		return &api.RenderStoryRoleDetailResponse{
			Code:    -1,
			Message: "get story role failed",
		}, err
	}
	story, err := models.GetStory(ctx, int64(role.StoryID))
	if err != nil {
		log.Log().Error("get story failed", zap.Error(err))
		return nil, err
	}
	if story.Status == -1 {
		return &api.RenderStoryRoleDetailResponse{
			Code:    -1,
			Message: "story is closed",
		}, nil
	}
	// 根据角色参与的故事背景，以及这个角色的描述，使用AI生成一个角色描述
	roleRequirePrompt := `生成故事 story_name 的角色,故事内容用中文描述,以json格式返回		
		角色名称:` + role.CharacterName + `
		角色描述:` + role.CharacterDescription + `
		故事背景:` + story.ShortDesc + `
		请参考以上输入,生成[角色描述,角色短期目标,角色长期目标,角色性格,角色背景],返回格式如下：
		---
		{
			"角色背景": "xxxxxx",
			"性格特征": "xxxxxx",
			"处事风格": "xxxxxx",
			"认知范围": "xxxxxx",
			"能力特点": "xxxxxx",
			"外貌特征": "xxxxxx",
			"穿着喜好": "xxxxxx",
			"角色描述": "xxxxxx",
			"角色短期目标": "xxxxxx",
			"角色长期目标": "xxxxxx"
		}
		---
		请返回json格式，不要返回其他内容。角色的描述，短期目标，长期目标，性格，背景，请用中文描述。
		`
	storyGen := new(models.StoryGen)
	storyGen.Uuid = uuid.New().String()
	reqData, _ := json.Marshal(req)
	storyGenData, _ := json.Marshal(reqData)
	storyGen.LLmPlatform = "Zhipu"
	storyGen.NegativePrompt = ""
	storyGen.PositivePrompt = roleRequirePrompt
	storyGen.Regen = 1
	storyGen.Params = string(storyGenData)
	storyGen.OriginID = int64(role.ID)
	storyGen.StartTime = time.Now().Unix()
	storyGen.BoardID = 0
	storyGen.GenType = int(api.RenderType_RENDER_TYPE_TEXT_UNSPECIFIED)
	storyGen.TaskType = int(api.RenderType_RENDER_TYPE_STORYCHARACTERS)
	_, err = models.CreateStoryGen(ctx, storyGen)
	if err != nil {
		log.Log().Error("create storyboard gen failed", zap.Error(err))
		return nil, err
	}
	log.Log().Sugar().Info("gen storyboard prompt: ", roleRequirePrompt)
	renderStoryParams := &client.StoryInfoParams{
		Content: roleRequirePrompt,
	}
	result := new(CharacterDetail)
	ret, err := s.client.GenStoryInfo(ctx, renderStoryParams)
	if err != nil {
		log.Log().Error("gen storyboard info failed", zap.Error(err))
		return nil, err
	}
	// 保存生成的故事板
	cleanResult := utils.CleanLLmJsonResult(ret.Content)
	err = json.Unmarshal([]byte(cleanResult), &result)
	if err != nil {
		log.Log().Error("unmarshal story gen result failed", zap.Error(err))
		return nil, err
	}
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
	storyGen.Content = cleanResult
	storyGen.FinishTime = time.Now().Unix()
	err = models.UpdateStoryGen(ctx, storyGen)
	if err != nil {
		log.Log().Error("update story gen failed", zap.Error(err))
	}
	return &api.RenderStoryRoleDetailResponse{
		Code:    0,
		Message: "OK",
		Role:    apiRoleDetail,
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
	creatorIds := make([]int64, 0)
	for _, role := range roles {
		creatorIds = append(creatorIds, role.CreatorID)
	}
	creatorsMap, err := models.GetUsersByIdsMap(ctx, creatorIds)
	if err != nil {
		return nil, err
	}
	finnalCreators := make([]*api.UserInfo, 0)
	for _, creator := range creatorsMap {
		finnalCreators = append(finnalCreators, &api.UserInfo{
			UserId: int64(creator.ID),
			Name:   creator.Name,
			Avatar: creator.Avatar,
		})
	}
	log.Log().Info("get story roles creators", zap.Any("creators", creatorsMap))
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
		cu, err := s.GetStoryRoleCurrentUserStatus(ctx, int64(role.ID))
		if err != nil {
			log.Log().Error("get story role current user status failed", zap.Error(err))
		}
		apiRole.CurrentUserStatus = cu
		apiRole.Creator = &api.UserInfo{
			UserId: int64(role.CreatorID),
			Name:   creatorsMap[int(role.CreatorID)].Name,
			Avatar: creatorsMap[int(role.CreatorID)].Avatar,
		}
		apiRole.LikeCount = role.LikeCount
		apiRole.FollowCount = role.FollowCount
		apiRole.StoryboardNum = role.StoryboardNum
		apiRole.Ctime = int64(role.CreateAt.Unix())
		apiRole.Mtime = int64(role.UpdateAt.Unix())
		apiRole.PosterImageUrl = role.PosterURL
		apiRoles = append(apiRoles, apiRole)
	}
	log.Log().Info("get story roles success", zap.Any("apiRoles", apiRoles))
	return &api.GetStoryRolesResponse{
		Code:    0,
		Message: "OK",
		Data: &api.GetStoryRolesResponse_Data{
			List:    apiRoles,
			Creator: finnalCreators,
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
	creatorIds := make([]int64, 0)
	for _, role := range roles {
		creatorIds = append(creatorIds, role.CreatorID)
	}
	creatorsMap, err := models.GetUsersByIdsMap(ctx, creatorIds)
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
		cu, err := s.GetStoryRoleCurrentUserStatus(ctx, int64(role.ID))
		if err != nil {
			log.Log().Error("get story role current user status failed", zap.Error(err))
		}
		apiRole.CurrentUserStatus = cu
		apiRole.LikeCount = role.LikeCount
		apiRole.FollowCount = role.FollowCount
		apiRole.StoryboardNum = role.StoryboardNum
		apiRole.Creator = &api.UserInfo{
			UserId: int64(role.CreatorID),
			Name:   creatorsMap[int(role.CreatorID)].Name,
			Avatar: creatorsMap[int(role.CreatorID)].Avatar,
		}
		apiRole.Ctime = int64(role.CreateAt.Unix())
		apiRole.Mtime = int64(role.UpdateAt.Unix())
		apiRole.PosterImageUrl = role.PosterURL
		apiRoles = append(apiRoles, apiRole)
	}
	finnalCreators := make([]*api.UserInfo, 0)
	for _, creator := range creatorsMap {
		finnalCreators = append(finnalCreators, &api.UserInfo{
			UserId: int64(creator.ID),
			Name:   creator.Name,
			Avatar: creator.Avatar,
		})
	}
	return &api.GetStoryBoardRolesResponse{
		Code:    0,
		Message: "OK",
		Data: &api.GetStoryBoardRolesResponse_Data{
			List:    apiRoles,
			Creator: finnalCreators,
		},
	}, nil
}

func (s *StoryService) UnLikeStoryboard(ctx context.Context, req *api.UnLikeStoryboardRequest) (*api.UnLikeStoryboardResponse, error) {
	likeItem, err := models.GetLikeItemByStoryBoardAndUser(ctx, req.GetBoardId(), int(req.GetUserId()))
	if err != nil {
		return nil, err
	}
	if likeItem == nil {
		return &api.UnLikeStoryboardResponse{
			Code:    0,
			Message: "OK",
		}, nil
	}
	err = models.DeleteLikeItem(ctx, int64(likeItem.ID))
	if err != nil {
		return nil, err
	}
	storyBoard, err := models.GetStoryboard(ctx, req.GetBoardId())
	if err != nil {
		return nil, err
	}
	storyBoard.LikeNum--
	err = models.UpdateStoryboard(ctx, storyBoard)
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
	newScene.GenResult = req.Sence.GetGenResult()
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
	preDefineTemplate := strings.Replace(models.PreDefineTemplateEnVersion[1].Prompt, "prompt", templatePrompt, -1)
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
	aliyunUrls := make([]string, 0)
	for _, imageUrl := range ret.ImageUrls {
		aliyunClient := aliyun.GetGlobalClient()
		aliyunUrl, err := aliyunClient.UploadFileFromURL("", imageUrl)
		if err != nil {
			log.Log().Error("upload file from url failed", zap.Error(err))
			continue
		}
		// aliyunThumbnailUrl, err := aliyunClient.GenerateThumbnailV2(aliyunUrl, 200)
		// log.Log().Sugar().Infof("aliyunThumbnailUrl: %s", aliyunThumbnailUrl)
		// if err != nil {
		// 	log.Log().Error("generate thumbnail failed", zap.Error(err))
		// 	continue
		// }
		// aliyunUrls = append(aliyunUrls, aliyunUrl, aliyunThumbnailUrl)
		aliyunUrls = append(aliyunUrls, aliyunUrl)
	}
	retData, _ := json.Marshal(aliyunUrls)
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
		preDefineTemplate := strings.Replace(models.PreDefineTemplateEnVersion[1].Prompt, "prompt", templatePrompt, -1)
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
		aliyunUrls := make([]string, 0)
		for _, imageUrl := range ret.ImageUrls {
			aliyunClient := aliyun.GetGlobalClient()
			aliyunUrl, err := aliyunClient.UploadFileFromURL("", imageUrl)
			if err != nil {
				log.Log().Error("upload file from url failed", zap.Error(err))
				continue
			}
			// aliyunThumbnailUrl, err := aliyunClient.GenerateThumbnailV2(aliyunUrl, 200)
			// if err != nil {
			// 	log.Log().Error("generate thumbnail failed", zap.Error(err))
			// 	continue
			// }
			// aliyunUrls = append(aliyunUrls, aliyunUrl, aliyunThumbnailUrl)
			aliyunUrls = append(aliyunUrls, aliyunUrl)
		}
		retData, _ := json.Marshal(aliyunUrls)
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
		resp.Code = 0
		resp.Message = "storyboard is already published"
		return resp, nil
	}
	switch storyboard.Stage {
	case int(api.StoryboardStage_STORYBOARD_STAGE_CREATED):
		// 创建完故事剧情(故事板)，但是没有渲染剧情
		board, err := models.GetStoryboard(ctx, req.GetStoryboardId())
		if err != nil {
			log.Log().Error("get storyboard failed", zap.Error(err))
			return nil, err
		}
		resp.Store = &api.StoryboardStageStore{
			Storyboard:     convert.ConvertStoryBoardToApiStoryBoard(board),
			Stage:          api.StoryboardStage_STORYBOARD_STAGE_CREATED,
			LastUpdateTime: board.UpdateAt.Unix(),
			Version:        board.UpdateAt.Unix(),
			UserId:         int64(board.CreatorID),
		}
	case int(api.StoryboardStage_STORYBOARD_STAGE_RENDERED):
		// 创建完故事剧情，但是没有生成图片,正常剧情渲染
		board, err := models.GetStoryboard(ctx, req.GetStoryboardId())
		if err != nil {
			log.Log().Error("get storyboard failed", zap.Error(err))
			return nil, err
		}
		resp.Store = &api.StoryboardStageStore{
			Storyboard:     convert.ConvertStoryBoardToApiStoryBoard(board),
			Stage:          api.StoryboardStage_STORYBOARD_STAGE_RENDERED,
			LastUpdateTime: board.UpdateAt.Unix(),
			Version:        board.UpdateAt.Unix(),
			UserId:         int64(board.CreatorID),
		}
		sences, err := models.GetStoryBoardScenesByBoard(ctx, req.GetStoryboardId())
		if err != nil {
			log.Log().Error("get storyboard scenes failed", zap.Error(err))
			return nil, err
		}
		if len(sences) == 0 {
			resp.Code = 0
			resp.Message = "storyboard has no scenes"
			return resp, nil
		}
		var apiScenes []*api.StoryBoardSence
		for _, scene := range sences {
			apiScenes = append(apiScenes, convert.ConvertStoryBoardSceneToApiStoryBoardScene(scene))
		}
		resp.Store.Storyboard.Sences = &api.StoryBoardSences{
			List: apiScenes,
		}
		resp.Store.Sences = &api.StoryBoardSences{
			List:  apiScenes,
			Total: int64(len(apiScenes)),
		}
	case int(api.StoryboardStage_STORYBOARD_STAGE_GEN_IMAGE):
		// 创建完故事剧情以及场景，已经渲染完图片，没有确认完成
		board, err := models.GetStoryboard(ctx, req.GetStoryboardId())
		if err != nil {
			log.Log().Error("get storyboard failed", zap.Error(err))
			return nil, err
		}
		resp.Store = &api.StoryboardStageStore{
			Storyboard:     convert.ConvertStoryBoardToApiStoryBoard(board),
			Stage:          api.StoryboardStage_STORYBOARD_STAGE_RENDERED,
			LastUpdateTime: board.UpdateAt.Unix(),
			Version:        board.UpdateAt.Unix(),
			UserId:         int64(board.CreatorID),
		}
		sences, err := models.GetStoryBoardScenesByBoard(ctx, req.GetStoryboardId())
		if err != nil {
			log.Log().Error("get storyboard scenes failed", zap.Error(err))
			return nil, err
		}
		if len(sences) == 0 {
			resp.Code = 0
			resp.Message = "storyboard has no scenes"
			return resp, nil
		}
		var apiScenes []*api.StoryBoardSence
		for _, scene := range sences {
			apiScenes = append(apiScenes, convert.ConvertStoryBoardSceneToApiStoryBoardScene(scene))
		}
		resp.Store.Storyboard.Sences = &api.StoryBoardSences{
			List: apiScenes,
		}
		resp.Store.Sences = &api.StoryBoardSences{
			List:  apiScenes,
			Total: int64(len(apiScenes)),
		}
	case int(api.StoryboardStage_STORYBOARD_STAGE_GEN_VIDEO):
		// 创建完故事剧情以及场景，但是没有生成视频，建议只有点赞高的、关注多的角色、付费用户使用
	case int(api.StoryboardStage_STORYBOARD_STAGE_GEN_AUDIO):
		// 创建完故事剧情以及场景，但是没有生成音频，建议只有旁白使用
	case int(api.StoryboardStage_STORYBOARD_STAGE_RENDER_SCENE):
		// 创建完故事剧情，但是没有创建场景描述
	case int(api.StoryboardStage_STORYBOARD_STAGE_FINISHED):
		// 已经创建完所有，但是没有发布
		board, err := models.GetStoryboard(ctx, req.GetStoryboardId())
		if err != nil {
			log.Log().Error("get storyboard failed", zap.Error(err))
			return nil, err
		}
		resp.Store = &api.StoryboardStageStore{
			Storyboard:     convert.ConvertStoryBoardToApiStoryBoard(board),
			Stage:          api.StoryboardStage_STORYBOARD_STAGE_RENDERED,
			LastUpdateTime: board.UpdateAt.Unix(),
			Version:        board.UpdateAt.Unix(),
			UserId:         int64(board.CreatorID),
		}
		sences, err := models.GetStoryBoardScenesByBoard(ctx, req.GetStoryboardId())
		if err != nil {
			log.Log().Error("get storyboard scenes failed", zap.Error(err))
			return nil, err
		}
		if len(sences) == 0 {
			resp.Code = 0
			resp.Message = "storyboard has no scenes"
			return resp, nil
		}
		var apiScenes []*api.StoryBoardSence
		for _, scene := range sences {
			apiScenes = append(apiScenes, convert.ConvertStoryBoardSceneToApiStoryBoardScene(scene))
		}
		resp.Store.Storyboard.Sences = &api.StoryBoardSences{
			List: apiScenes,
		}
		resp.Store.Sences = &api.StoryBoardSences{
			List:  apiScenes,
			Total: int64(len(apiScenes)),
		}
	}

	return resp, nil
}

// 获取用户创建的故事板
func (s *StoryService) GetUserCreatedStoryboards(ctx context.Context, req *api.GetUserCreatedStoryboardsRequest) (*api.GetUserCreatedStoryboardsResponse, error) {
	storyboards, total, err := models.GetUserCreatedStoryboardsWithStoryId(ctx, int(req.GetUserId()),
		int(req.GetStoryId()), int(req.GetOffset()), int(req.GetPageSize()))
	if err != nil {
		log.Log().Error("get user created storyboards failed", zap.Error(err))
		return nil, err
	}
	log.Log().Info("get user created storyboards", zap.Int("total", len(storyboards)))
	storiesSummary := make(map[int64]*api.StorySummaryInfo)
	storyIds := make([]int64, 0)
	for _, storyboard := range storyboards {
		storyIds = append(storyIds, int64(storyboard.StoryID))
	}
	stories, err := models.GetStoriesByIDs(ctx, storyIds)
	if err != nil {
		log.Log().Error("get stories by ids failed", zap.Error(err))
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
		cu, err := s.GetStoryboardCurrentUserStatus(ctx, int64(storyboard.ID))
		if err != nil {
			log.Log().Error("get storyboard current user status failed", zap.Error(err))
		}
		newApiStoryboard.CurrentUserStatus = cu
		roles, err := models.GetStoryBoardRolesByBoard(ctx, int64(storyboard.ID))
		if err != nil {
			log.Log().Error("get storyboard roles failed", zap.Error(err))
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
	}
	log.Log().Info("get user created storyboards", zap.Any("apiStoryboards length", len(apiStoryboards)))
	return result, nil
}

func (s *StoryService) GetNextStoryboard(ctx context.Context, req *api.GetNextStoryboardRequest) (*api.GetNextStoryboardResponse, error) {
	board, err := models.GetStoryboard(ctx, req.GetStoryboardId())
	if err != nil {
		log.Log().Error("get storyboard failed", zap.Error(err))
		return nil, err
	}
	resp := &api.GetNextStoryboardResponse{}
	boards, err := models.GetStoryBoardByStoryAndPrevId(ctx,
		board.StoryID, req.GetStoryboardId(), int(req.GetOffset()),
		int(req.GetPageSize()), req.GetOrderBy().String())
	if err != nil {
		log.Log().Error("get next storyboard failed", zap.Error(err))
		return nil, err
	}
	if len(boards) == 0 {
		log.Log().Info("no next storyboard")
		resp.Code = 0
		resp.Message = "OK"
		resp.Offset = 0
		resp.Storyboards = make([]*api.StoryBoardActive, 0)
		resp.IsMultiBranch = true
		return resp, nil
	}
	apiBoards := make([]*api.StoryBoardActive, 0)
	log.Log().Info("get next storyboard", zap.Int("total", len(boards)))
	for _, board := range boards {
		cu, err := s.GetStoryboardCurrentUserStatus(ctx, int64(board.ID))
		if err != nil {
			log.Log().Error("get storyboard current user status failed", zap.Error(err))
		}
		boardInfo := convert.ConvertStoryBoardToApiStoryBoard(board)
		boardInfo.CurrentUserStatus = cu
		sences, err := models.GetStoryBoardScenesByBoard(ctx, int64(board.ID))
		if err != nil {
			log.Log().Error("get board sences failed", zap.Error(err))
		}
		if len(sences) != 0 {
			boardInfo.Sences = new(api.StoryBoardSences)
			for _, scene := range sences {
				boardInfo.Sences.List = append(boardInfo.Sences.List, ConvertStorySceneToApiScene(scene))
			}
			boardInfo.Sences.Total = int64(len(boardInfo.Sences.List))
		} else {
			log.Log().Warn("story sences is empty")
		}
		creator, err := models.GetUserById(ctx, int64(board.CreatorID))
		if err != nil {
			log.Log().Error("get user failed", zap.Error(err))
			return nil, err
		}
		roles, err := models.GetStoryBoardRolesByBoard(ctx, int64(board.ID))
		if err != nil {
			log.Log().Error("get board roles failed", zap.Error(err))
			return nil, err
		}
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
	resp.Storyboards = apiBoards
	resp.IsMultiBranch = true
	resp.Offset = 0
	resp.Total = int64(len(boards))
	resp.PageSize = int64(len(boards))
	log.Log().Info("get next storyboard", zap.Any("result", resp.String()))
	return resp, nil
}

func (s *StoryService) PublishStoryboard(ctx context.Context, req *api.PublishStoryboardRequest) (*api.PublishStoryboardResponse, error) {
	storyboard, err := models.GetStoryboard(ctx, req.GetStoryboardId())
	if err != nil {
		return nil, err
	}
	preBoardId := storyboard.PrevId
	storyboard.Stage = int(api.StoryboardStage_STORYBOARD_STAGE_PUBLISHED)
	models.UpdateStoryboard(ctx, storyboard)
	if preBoardId > 0 {
		err = models.IncrementStoryBoardForkNum(ctx, preBoardId)
		if err != nil {
			return nil, err
		}
	}
	return &api.PublishStoryboardResponse{
		Code:    0,
		Message: "OK",
	}, nil
}

func (s *StoryService) CancelStoryboard(ctx context.Context, req *api.CancelStoryboardRequest) (*api.CancelStoryboardResponse, error) {
	storyboard, err := models.GetStoryboard(ctx, req.GetStoryboardId())
	if err != nil {
		return nil, err
	}
	storyboard.Stage = int(api.StoryboardStage_STORYBOARD_STAGE_UNSPECIFIED)
	models.UpdateStoryboard(ctx, storyboard)
	return &api.CancelStoryboardResponse{
		Code:    0,
		Message: "OK",
	}, nil
}

func (s *StoryService) GetUserWatchStoryActiveStoryBoards(ctx context.Context, req *api.GetUserWatchStoryActiveStoryBoardsRequest) (*api.GetUserWatchStoryActiveStoryBoardsResponse, error) {
	fmt.Println("GetUserWatchStoryActiveStoryBoards: ", req.String())
	stortIds, err := models.GetStoriesIdByUserFollow(ctx, int64(req.GetUserId()))
	if err != nil {
		return nil, err
	}
	if len(stortIds) == 0 {
		return &api.GetUserWatchStoryActiveStoryBoardsResponse{
			Code:    0,
			Message: "OK",
			Total:   0,
		}, nil
	}
	boards, err := models.GetStoryBoardsByStoryIds(ctx, stortIds, int(req.GetOffset()), int(req.GetPageSize()), req.GetFilter())
	if err != nil {
		return nil, err
	}
	log.Log().Info("get user watch story active story boards", zap.Any("boards", len(boards)))
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
		// if story.Status != 1 {
		// 	continue
		// }
		// if story.Deleted == true {
		// 	continue
		// }
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
		cu, err := s.GetStoryboardCurrentUserStatus(ctx, int64(board.ID))
		if err != nil {
			log.Log().Error("get storyboard current user status failed", zap.Error(err))
		}
		boardsItem.CurrentUserStatus = cu

		roles, err := models.GetStoryBoardRolesByBoard(ctx, int64(board.ID))
		if err != nil {
			return nil, err
		}
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
		fmt.Printf("storiesSummary : %+v \n", storiesSummary[int64(board.StoryID)])
	}
	log.Log().Info("get user watch story active story boards", zap.Any("boards", len(apiBoards)))
	resp := &api.GetUserWatchStoryActiveStoryBoardsResponse{
		Code:        0,
		Message:     "OK",
		Storyboards: apiBoards,
		Total:       int64(len(boards)),
		Offset:      req.GetOffset(),
		PageSize:    req.GetPageSize(),
	}
	return resp, nil
}

func (s *StoryService) GetUserWatchRoleActiveStoryBoards(ctx context.Context, req *api.GetUserWatchRoleActiveStoryBoardsRequest) (*api.GetUserWatchRoleActiveStoryBoardsResponse, error) {
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
		cu, err := s.GetStoryboardCurrentUserStatus(ctx, int64(board.ID))
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
		Code:        0,
		Message:     "OK",
		Storyboards: apiBoards,
		Total:       int64(len(boards)),
		Offset:      req.GetOffset(),
		PageSize:    req.GetPageSize(),
	}, nil
}

func (s *StoryService) GetUnPublishStoryboard(ctx context.Context, req *api.GetUnPublishStoryboardRequest) (*api.GetUnPublishStoryboardResponse, error) {
	boards, err := models.GetUnPublishedStoryBoardsByUserId(ctx, req.GetUserId(), int(req.GetOffset()), int(req.GetPageSize()), "create_at desc")
	if err != nil {
		return nil, err
	}
	targetStoryIds := make([]int64, 0)
	for _, board := range boards {
		targetStoryIds = append(targetStoryIds, int64(board.StoryID))
	}
	stories, err := models.GetStoriesByIDs(ctx, targetStoryIds)
	log.Log().Info("stories: ", zap.Any("stories", stories))
	if err != nil {
		log.Log().Error("get stories by ids failed", zap.Error(err))
		return nil, err
	}
	storiesSummary := make(map[int64]*api.StorySummaryInfo)
	for _, story := range stories {
		if story.Status != 1 {
			log.Log().Info("story status is not 1", zap.Any("story", story))
			continue
		}
		if story.Deleted == true {
			log.Log().Info("story is deleted", zap.Any("story", story))
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

	}
	return &api.GetUnPublishStoryboardResponse{
		Code:              0,
		Message:           "OK",
		Storyboardactives: apiBoards,
		Total:             int64(len(boards)),
	}, nil
}
