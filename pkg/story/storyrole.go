package story

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"

	api "github.com/grapery/common-protoc/gen"
	"github.com/grapery/grapery/models"
	"github.com/grapery/grapery/pkg/client"
	"github.com/grapery/grapery/utils"
	"github.com/grapery/grapery/utils/convert"
	"github.com/grapery/grapery/utils/log"
)

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
	userProfile := &models.UserProfile{
		UserId: int64(req.GetUserId()),
	}
	err = userProfile.IncrementLikedRoleNum()
	if err != nil {
		log.Log().Error("increment liked role num failed", zap.Error(err))
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
	userProfile := &models.UserProfile{
		UserId: int64(req.GetUserId()),
	}
	err = userProfile.DecrementLikedRoleNum()
	if err != nil {
		log.Log().Error("decrement liked role num failed", zap.Error(err))
	}
	return &api.UnLikeStoryRoleResponse{
		Code:    0,
		Message: "OK",
	}, nil
}

func (s *StoryService) FollowStoryRole(ctx context.Context, req *api.FollowStoryRoleRequest) (*api.FollowStoryRoleResponse, error) {
	story, err := models.GetStory(ctx, req.GetStoryId())
	if err != nil {
		log.Log().Error("get story failed", zap.Error(err))
		return nil, err
	}
	isWatch, err := models.GetWatchItemByStoryRoleAndUser(ctx, req.GetRoleId(), int64(req.GetUserId()))
	if err != nil {
		log.Log().Error("get watch item by story role and user failed", zap.Error(err))
		return nil, err
	}
	if isWatch != nil {
		return &api.FollowStoryRoleResponse{
			Code:    0,
			Message: "OK",
		}, nil
	}
	err = models.WatchStoryRole(ctx, int(req.GetUserId()), req.GetStoryId(), req.GetRoleId(), story.GroupID)
	if err != nil {
		log.Log().Error("watch story role failed", zap.Error(err))
		return nil, err
	}
	err = models.IncreaseStoryRoleFollowCount(ctx, req.GetRoleId(), 1)
	if err != nil {
		log.Log().Error("increase story role follow count failed", zap.Error(err))
		return nil, err
	}
	userProfile := &models.UserProfile{
		UserId: int64(req.GetUserId()),
	}
	err = userProfile.IncrementWatchingStoryRoleNum()
	if err != nil {
		log.Log().Error("increment watching story role num failed", zap.Error(err))
	}
	return &api.FollowStoryRoleResponse{
		Code:    0,
		Message: "OK",
	}, nil
}

func (s *StoryService) UnFollowStoryRole(ctx context.Context, req *api.UnFollowStoryRoleRequest) (*api.UnFollowStoryRoleResponse, error) {
	story, err := models.GetStory(ctx, req.GetStoryId())
	if err != nil {
		log.Log().Error("get story failed", zap.Error(err))
		return nil, err
	}
	isWatch, err := models.GetWatchItemByStoryRoleAndUser(ctx, req.GetRoleId(), int64(req.GetUserId()))
	if err != nil {
		log.Log().Error("get watch item by story role and user failed", zap.Error(err))
		return nil, err
	}
	if isWatch == nil {
		return &api.UnFollowStoryRoleResponse{
			Code:    0,
			Message: "OK",
		}, nil
	}
	err = models.UnWatchStoryRole(ctx, int(req.GetUserId()), req.GetStoryId(), req.GetRoleId(), story.GroupID)
	err = models.DecreaseStoryRoleFollowCount(ctx, req.GetRoleId(), 1)
	if err != nil {
		log.Log().Error("decrease story role follow count failed", zap.Error(err))
		return nil, err
	}
	userProfile := &models.UserProfile{
		UserId: int64(req.GetUserId()),
	}
	err = userProfile.DecrementWatchingStoryRoleNum()
	if err != nil {
		log.Log().Error("decrement watching story role num failed", zap.Error(err))
	}
	return &api.UnFollowStoryRoleResponse{
		Code:    0,
		Message: "OK",
	}, nil
}

// 获取用户创建的角色
func (s *StoryService) GetUserCreatedRoles(ctx context.Context, req *api.GetUserCreatedRolesRequest) (*api.GetUserCreatedRolesResponse, error) {
	roles, total, err := models.GetUserCreatedRolesWithStoryId(ctx, int(req.GetUserId()),
		int(req.GetStoryId()), int(req.GetOffset()), int(req.GetPageSize()))
	if err != nil {
		log.Log().Error("get user created roles failed", zap.Error(err))
		return nil, err
	}
	apiRoles := make([]*api.StoryRole, 0)
	for _, role := range roles {
		if role.Status != 1 {
			continue
		}
		if role.Deleted == true {
			continue
		}
		apiRole := convert.ConvertStoryRoleToApiStoryRoleInfo(role)

		roleDetail := &CharacterDetail{}
		err = json.Unmarshal([]byte(role.CharacterDetail), &roleDetail)
		if err != nil {
			log.Log().Error("unmarshal story role character detail failed", zap.Error(err))
		}
		apiRole.CharacterDetail = &api.CharacterDetail{
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
		apiRole.LikeCount = role.LikeCount
		apiRole.FollowCount = role.FollowCount
		apiRole.StoryboardNum = role.StoryboardNum
		apiRole.Ctime = int64(role.CreateAt.Unix())
		apiRole.Mtime = int64(role.UpdateAt.Unix())
		apiRoles = append(apiRoles, apiRole)
	}
	return &api.GetUserCreatedRolesResponse{
		Code:     0,
		Message:  "OK",
		Roles:    apiRoles,
		Total:    total,
		Offset:   int64(req.GetOffset()),
		PageSize: int64(req.GetPageSize()),
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
	newRole.FollowCount = 1
	newRole.LikeCount = 1
	newRole.Status = 1
	newRole.CharacterDetail = "{}"
	roleId, err := models.CreateStoryRole(ctx, newRole)
	if err != nil {
		return nil, err
	}
	userProfille := new(models.UserProfile)
	userProfille.UserId = req.GetUserId()
	err = userProfille.GetByUserId()
	if err != nil {
		log.Log().Error("update user profile error: ", zap.Error(err))
		return nil, err
	}
	err = models.CreateWatchRoleItem(ctx, int(req.GetRole().GetCreatorId()), int64(story.ID), int64(roleId), int64(story.GroupID))
	if err != nil {
		log.Log().Error("create watch story item failed", zap.Error(err))
		return nil, err
	}
	log.Log().Info("create role success", zap.String("role", newRole.String()))
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
	cu, err := s.GetStoryRoleCurrentUserStatus(ctx, int64(role.ID))
	if err != nil {
		log.Log().Error("get story role current user status failed", zap.Error(err))
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
			CurrentUserStatus:    cu,
		},
	}, nil
}

func (s *StoryService) RenderStoryRole(ctx context.Context, req *api.RenderStoryRoleRequest) (*api.RenderStoryRoleResponse, error) {
	role, err := models.GetStoryRoleByID(ctx, req.GetRoleId())
	if err != nil {
		return nil, err
	}
	if role.CreatorID != req.GetUserId() {
		return nil, errors.New("have no permission")
	}
	if role.Status != 1 {
		return nil, errors.New("role is not ready")
	}
	story, err := models.GetStory(ctx, role.StoryID)
	if err != nil {
		return nil, err
	}
	templatePrompt := `
	为故事的角色生成性格描述，穿着描述，以及行为描述、角色的目标。我会提供这个角色参与的故事的背景。同时，也会输入我认为的这个角色的特点。
	故事角色姓名:"""story_role_name"""
	故事背景:"""story_background"""
`

	templatePrompt2 := `
	返回的角色描述信息，请按照json格式返回，以下是返回样例：
	--------
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
	--------
	请不要生成过于色情、暴力、恶心的内容，或者一直重复的内容，请不要出现任何违反法律法规的内容，保证角色贴合故事背景，同时遵循用户的输入的角色性格特点要求。
	`
	prompt := templatePrompt
	prompt = strings.Replace(prompt, "story_role_name", role.CharacterName, -1)
	prompt = strings.Replace(prompt, "story_background", story.ShortDesc, -1)
	if req.GetPrompt() != "" {
		prompt = prompt + `我建议这个角色的特征包括："""` + req.GetPrompt() + `"""。\n`
	}
	prompt = prompt + templatePrompt2
	// 调用生成器
	storyGen := new(models.StoryGen)
	storyGen.Uuid = uuid.New().String()
	storyGen.LLmPlatform = "Zhipu"
	storyGen.NegativePrompt = ""
	storyGen.PositivePrompt = prompt
	storyGen.Regen = 0
	storyGen.Params = req.String()
	storyGen.OriginID = req.GetRoleId()
	storyGen.StartTime = time.Now().Unix()
	storyGen.BoardID = 0
	storyGen.GenType = int(api.RenderType_RENDER_TYPE_STORYCHARACTERS)
	storyGen.TaskType = 3
	storyGen.Status = 1
	_, err = models.CreateStoryGen(ctx, storyGen)
	if err != nil {
		return nil, err
	}
	var (
		ret                   *client.GenStoryRoleInfoResult
		renderStoryRoleParams = &client.GenStoryRoleInfoParams{
			Content: prompt,
		}
	)

	ret, err = s.client.GenStoryRoleInfo(ctx, renderStoryRoleParams)
	if err != nil {
		log.Log().Error("gen story info failed", zap.Error(err))
		return nil, err
	}
	var renderDetail = new(api.RenderStoryRoleDetail)
	result := new(CharacterDetail)
	cleanResult := utils.CleanLLmJsonResult(ret.Content)
	err = json.Unmarshal([]byte(cleanResult), &result)
	if err != nil {
		log.Log().Error("unmarshal gen result failed", zap.Error(err))
		return nil, err
	}
	storyGen.Content = cleanResult
	storyGen.FinishTime = time.Now().Unix()
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
		log.Log().Error("update story gen failed", zap.Error(err))
	}
	return &api.RenderStoryRoleResponse{
		Code:    0,
		Message: "OK",
		Detail:  renderDetail,
	}, nil
}

// 获取角色故事
func (s *StoryService) GetStoryRoleStories(ctx context.Context, req *api.GetStoryRoleStoriesRequest) (*api.GetStoryRoleStoriesResponse, error) {
	return nil, nil
}

// 获取角色故事板
func (s *StoryService) GetStoryRoleStoryboards(ctx context.Context, req *api.GetStoryRoleStoryboardsRequest) (*api.GetStoryRoleStoryboardsResponse, error) {
	boards, err := models.GetStoryBoardsByRoleID(ctx, req.GetRoleId(), int(req.GetOffset()), int(req.GetPageSize()))
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
	if len(boards) == 0 {
		return &api.GetStoryRoleStoryboardsResponse{
			Code:    0,
			Message: "OK",
		}, nil
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
			StoryCover:       "",
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
	return &api.GetStoryRoleStoryboardsResponse{
		Code:              0,
		Message:           "OK",
		Storyboardactives: apiBoards,
		Total:             int64(len(apiBoards)),
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
	chatCtx, err := models.GetChatContextByUserIDAndRoleID(ctx, int64(req.GetUserId()), int64(req.GetRoleId()))
	if err != nil && err != gorm.ErrRecordNotFound {
		log.Log().Error("get user chat context failed", zap.Error(err))
		return nil, err
	}
	if err == gorm.ErrRecordNotFound {
		// 创建聊天上下文
		chatCtx = new(models.ChatContext)
		chatCtx.UserID = int64(req.GetUserId())
		chatCtx.RoleID = int64(req.GetRoleId())
		chatCtx.Title = "聊天消息"
		chatCtx.Content = ""
		chatCtx.Status = 1
		err = models.CreateChatContext(ctx, chatCtx)
		if err != nil {
			log.Log().Error("create story role chat failed", zap.Error(err))
			return nil, err
		}
	}
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
			log.Log().Error("create story role chat message failed", zap.Error(err))
			return nil, err
		}
		reply = append(reply, convert.ConvertChatMessageToApiChatMessage(chatMessage))
		{
			roleInfo, err := models.GetStoryRoleByID(ctx, int64(req.GetRoleId()))
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
	chatCtxs, total, err := models.GetChatContextByUserID(ctx, int64(req.GetUserId()), int(req.GetOffset()), int(req.GetPageSize()))
	if err != nil {
		log.Log().Error("get user chat context failed", zap.Error(err))
		return nil, err
	}
	log.Log().Info("get user chat context success", zap.Any("total", total), zap.Any("chatCtxs", len(chatCtxs)))
	apiChatCtxs := make([]*api.ChatContext, 0)
	for _, chatCtx := range chatCtxs {
		if chatCtx.UserID == 0 || chatCtx.RoleID == 0 {
			log.Log().Error("invalid chat context", zap.Any("chatCtx", chatCtx))
			continue
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
	log.Log().Info("get user with role chat list success", zap.Any("total", total), zap.Any("apiChatCtxs", len(apiChatCtxs)))
	return &api.GetUserWithRoleChatListResponse{
		Code:     0,
		Message:  "OK",
		Chats:    apiChatCtxs,
		Total:    int64(total),
		Offset:   int64(req.GetOffset()),
		PageSize: int64(req.GetPageSize()),
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
		var promptDetail = new(api.RenderStoryRoleDetail)
		err = json.Unmarshal([]byte(req.GetRole().GetCharacterPrompt()), promptDetail)
		if err != nil {
			log.Log().Error("unmarshal character prompt failed", zap.Error(err))
			return nil, err
		}
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

// 根据角色参与的故事板的历史记录，以及和别的角色的冲突，生成角色的性格描述，以及新的角色背景图片和头像图片
func (s *StoryService) RenderStoryRoleContinuously(ctx context.Context, req *api.RenderStoryRoleContinuouslyRequest) (*api.RenderStoryRoleContinuouslyResponse, error) {
	role, err := models.GetStoryRoleByID(ctx, req.GetRoleId())
	if err != nil {
		return nil, err
	}
	if role.CreatorID != req.GetUserId() {
		return nil, errors.New("have no permission")
	}
	if role.Status != 1 {
		return nil, errors.New("role is not ready")
	}
	story, err := models.GetStory(ctx, role.StoryID)
	if err != nil {
		return nil, err
	}
	historyStoryGen, err := models.GetStoryGensByStoryAndRole(ctx, role.StoryID, int64(role.ID))
	if err != nil {
		log.Log().Error("get story gen by story and role failed", zap.Error(err))
	}
	if historyStoryGen != nil && historyStoryGen.GenStatus == 1 {
		return &api.RenderStoryRoleContinuouslyResponse{
			Code:    0,
			Message: "generating",
			Detail:  nil,
		}, nil
	}
	if historyStoryGen != nil && historyStoryGen.GenStatus == 2 && historyStoryGen.CreateAt.Add(time.Hour*12).Before(time.Now()) {
		return &api.RenderStoryRoleContinuouslyResponse{
			Code:    0,
			Message: "role render finished",
			Detail:  nil,
		}, nil
	}

	templatePrompt := `
			为故事的角色生成性格描述，穿着描述，以及行为描述、角色的目标等信息。我会提供这个角色参与的故事的背景。同时，也会输入我认为的这个角色的特点。
			故事角色姓名:"""story_role_name"""
			故事背景:"""story_background"""

	故事中的这个角色按照时间顺序，所经历的故事场景:"""story_history"""
`
	histroryStoryBoardSences, err := models.GetStoryBoardSencesByRoleID(ctx, role.StoryID)
	if err != nil {
		return nil, err
	}
	var historySenceStr = ""
	for _, histrorySence := range histroryStoryBoardSences {
		historySenceStr = historySenceStr + histrorySence.Content + "\n"
	}
	templatePrompt = strings.Replace(templatePrompt, "story_history", historySenceStr, -1)
	templatePrompt = strings.Replace(templatePrompt, "story_background", story.ShortDesc, -1)
	templatePrompt = strings.Replace(templatePrompt, "story_role_name", role.CharacterName, -1)
	templatePrompt2 := `
	返回的角色描述信息，请按照json格式返回，以下是返回样例：
	--------
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
	--------
	请不要生成过于色情、暴力、恶心的内容，或者一直重复的内容，请不要出现任何违反法律法规的内容，保证角色贴合故事背景，同时遵循用户的输入的角色性格特点要求。
	`
	prompt := templatePrompt
	prompt = strings.Replace(prompt, "story_role_name", role.CharacterName, -1)
	prompt = strings.Replace(prompt, "story_background", story.ShortDesc, -1)
	if req.GetPrompt() != "" {
		prompt = prompt + `我建议这个角色的特征包括："""` + req.GetPrompt() + `"""。\n`
	}
	prompt = prompt + templatePrompt2
	// 调用生成器
	storyGen := new(models.StoryGen)
	storyGen.Uuid = uuid.New().String()
	storyGen.LLmPlatform = "Zhipu"
	storyGen.NegativePrompt = ""
	storyGen.PositivePrompt = prompt
	storyGen.Regen = 2
	storyGen.Params = req.String()
	storyGen.OriginID = req.GetRoleId()
	storyGen.StartTime = time.Now().Unix()
	storyGen.BoardID = 0
	storyGen.GenType = int(api.RenderType_RENDER_TYPE_STORYCHARACTERS)
	storyGen.TaskType = 3
	storyGen.Status = 1
	_, err = models.CreateStoryGen(ctx, storyGen)
	if err != nil {
		return nil, err
	}
	var (
		ret                   *client.GenStoryRoleInfoResult
		renderStoryRoleParams = &client.GenStoryRoleInfoParams{
			Content: prompt,
		}
	)

	ret, err = s.client.GenStoryRoleInfo(ctx, renderStoryRoleParams)
	if err != nil {
		log.Log().Error("gen story info failed", zap.Error(err))
		return nil, err
	}
	var renderDetail = new(api.RenderStoryRoleDetail)
	result := new(CharacterDetail)
	cleanResult := utils.CleanLLmJsonResult(ret.Content)
	err = json.Unmarshal([]byte(cleanResult), &result)
	if err != nil {
		log.Log().Error("unmarshal gen result failed", zap.Error(err))
		return nil, err
	}
	storyGen.Content = cleanResult
	storyGen.FinishTime = time.Now().Unix()
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
		log.Log().Error("update story gen failed", zap.Error(err))
	}
	return &api.RenderStoryRoleContinuouslyResponse{
		Code:    0,
		Message: "OK",
		Detail:  renderDetail,
	}, nil
}

func (s *StoryService) GenerateRoleDescription(ctx context.Context, req *api.GenerateRoleDescriptionRequest) (*api.GenerateRoleDescriptionResponse, error) {

	roleinfo, err := models.GetStoryRoleByID(ctx, req.GetRoleId())
	if err != nil {
		return nil, err
	}
	if roleinfo.CreatorID != req.GetUserId() {
		return nil, errors.New("have no permission")
	}

	storyinfo, err := models.GetStory(ctx, roleinfo.StoryID)
	if err != nil {
		return nil, err
	}

	// Get all roles in the story to provide context
	roles, err := models.GetStoryRole(ctx, req.GetStoryId())
	if err != nil {
		return nil, err
	}

	// Build role context information
	var otherRolesInfo strings.Builder
	for _, role := range roles {
		if role.ID != roleinfo.ID {
			otherRolesInfo.WriteString(fmt.Sprintf("角色名称: %s\n角色描述: %s\n\n", role.CharacterName, role.CharacterDescription))
		}
	}

	// Build the prompt template
	promptTemplate := `
		请为一个故事中的角色生成详细的角色设定。以下是相关背景信息：

		故事背景：
		%s

		故事简介：
		%s

		当前角色基本信息：
		角色名称：%s
		%s

		故事中的其他角色：
		%s

		请根据以上信息，生成一个详细的角色设定描述，包含以下方面：
		1. 角色背景
		2. 性格特征
		3. 处事风格
		4. 认知范围
		5. 能力特点
		6. 外貌特征
		7. 穿着喜好
		8. 角色描述
		9. 角色短期目标
		10. 角色长期目标

		请以JSON格式返回，格式如下：
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

		注意：
		1. 描述要符合故事背景和整体设定
		2. 避免矛盾的人设
		3. 确保描述合理且具体
		4. 不要包含任何暴力、色情或违法的内容
		`

	// Format the prompt with actual data
	prompt := fmt.Sprintf(promptTemplate,
		storyinfo.ShortDesc,           // 故事背景
		storyinfo.Params,              // 故事简介
		roleinfo.CharacterName,        // 角色名称
		roleinfo.CharacterDescription, // 当前角色描述
		otherRolesInfo.String(),       // 其他角色信息
	)

	// Call AI client to generate description
	genParams := &client.GenStoryRoleInfoParams{
		Content: prompt,
	}

	result, err := s.client.GenStoryRoleInfo(ctx, genParams)
	if err != nil {
		log.Log().Error("generate role description failed", zap.Error(err))
		return nil, err
	}

	// Clean and parse the AI response
	cleanResult := utils.CleanLLmJsonResult(result.Content)
	log.Log().Info("cleaned LLM result for role description", zap.String("content", cleanResult))
	var genRoleDetail = new(CharacterDetail)
	err = json.Unmarshal([]byte(cleanResult), &genRoleDetail)
	if err != nil {
		return &api.GenerateRoleDescriptionResponse{
			Code:    -1,
			Message: err.Error(),
		}, nil
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
	log.Log().Info("generate role description success", zap.Any("apiCharacterDetail", apiCharacterDetail.String()))
	return &api.GenerateRoleDescriptionResponse{
		Code:            0,
		Message:         "OK",
		CharacterDetail: apiCharacterDetail,
	}, nil
}

func (s *StoryService) UpdateStoryRoleDescriptionDetail(ctx context.Context, req *api.UpdateStoryRoleDescriptionDetailRequest) (*api.UpdateStoryRoleDescriptionDetailResponse, error) {
	roleinfo, err := models.GetStoryRoleByID(ctx, req.GetRoleId())
	if err != nil {
		return nil, err
	}
	if roleinfo == nil {
		return &api.UpdateStoryRoleDescriptionDetailResponse{
			Code:    -1,
			Message: "role not exist",
		}, nil
	}
	if roleinfo.CreatorID != req.GetUserId() {
		return &api.UpdateStoryRoleDescriptionDetailResponse{
			Code:    -1,
			Message: "have no permission",
		}, nil
	}
	descStr, _ := json.Marshal(req.GetCharacterDetail())
	roleinfo.CharacterDetail = string(descStr)
	err = models.UpdateStoryRole(ctx, int64(roleinfo.ID), map[string]interface{}{
		"character_detail": roleinfo.CharacterDetail,
	})
	if err != nil {
		log.Log().Error("update story role description failed", zap.Error(err))
		return nil, err
	}
	return &api.UpdateStoryRoleDescriptionDetailResponse{
		Code:    0,
		Message: "OK",
	}, nil
}

func (s *StoryService) UpdateRoleDescription(ctx context.Context, req *api.UpdateRoleDescriptionRequest) (*api.UpdateRoleDescriptionResponse, error) {
	roleinfo, err := models.GetStoryRoleByID(ctx, req.GetRoleId())
	if err != nil {
		return nil, err
	}
	if roleinfo.CreatorID != req.GetUserId() {
		return nil, errors.New("have no permission")
	}
	if roleinfo.Status != 1 {
		return nil, errors.New("role is not ready")
	}
	if roleinfo.CreatorID != req.GetUserId() {
		return nil, errors.New("have no permission")
	}
	roleinfo.CharacterDescription = req.GetDescription()
	err = models.UpdateStoryRole(ctx, int64(roleinfo.ID), map[string]interface{}{
		"character_description": req.GetDescription(),
	})
	if err != nil {
		return nil, err
	}
	return &api.UpdateRoleDescriptionResponse{
		Code:    0,
		Message: "OK",
	}, nil
}

func (s *StoryService) UpdateStoryRolePrompt(ctx context.Context, req *api.UpdateStoryRolePromptRequest) (*api.UpdateStoryRolePromptResponse, error) {
	roleinfo, err := models.GetStoryRoleByID(ctx, req.GetRoleId())
	if err != nil {
		return nil, err
	}
	if roleinfo.CreatorID != req.GetRoleId() {
		return nil, errors.New("have no permission")
	}
	roleinfo.CharacterPrompt = req.GetPrompt()
	err = models.UpdateStoryRole(ctx, int64(roleinfo.ID), map[string]interface{}{
		"character_prompt": req.GetPrompt(),
	})
	if err != nil {
		return nil, err
	}
	return &api.UpdateStoryRolePromptResponse{
		Code:    0,
		Message: "OK",
	}, nil
}

func (s *StoryService) GenerateRolePrompt(ctx context.Context, req *api.GenerateRolePromptRequest) (*api.GenerateRolePromptResponse, error) {
	storyinfo, err := models.GetStory(ctx, req.GetStoryId())
	if err != nil {
		return nil, err
	}
	roleinfo, err := models.GetStoryRoleByID(ctx, req.GetRoleId())
	if err != nil {
		return nil, err
	}
	if roleinfo.CreatorID != req.GetUserId() {
		return nil, errors.New("have no permission")
	}
	_ = storyinfo
	return nil, nil
}

func (s *StoryService) UpdateRolePrompt(ctx context.Context, req *api.UpdateRolePromptRequest) (*api.UpdateRolePromptResponse, error) {
	roleinfo, err := models.GetStoryRoleByID(ctx, req.GetRoleId())
	if err != nil {
		return nil, err
	}
	if roleinfo.CreatorID != req.GetUserId() {
		return nil, errors.New("have no permission")
	}
	roleinfo.CharacterPrompt = req.GetPrompt()
	err = models.UpdateStoryRole(ctx, int64(roleinfo.ID), map[string]interface{}{
		"character_prompt": req.GetPrompt(),
	})
	if err != nil {
		return nil, err
	}
	return &api.UpdateRolePromptResponse{
		Code:    0,
		Message: "OK",
	}, nil
}

func (s *StoryService) UpdateStoryRolePoster(ctx context.Context, req *api.UpdateStoryRolePosterRequest) (*api.UpdateStoryRolePosterResponse, error) {
	roleinfo, err := models.GetStoryRoleByID(ctx, req.GetRoleId())
	if err != nil {
		log.Log().Error("get story role by id failed", zap.Error(err))
		return nil, err
	}
	if roleinfo.CreatorID != req.GetUserId() {
		log.Log().Error("have no permission", zap.Any("roleinfo", roleinfo))
		return nil, errors.New("have no permission")
	}
	roleinfo.PosterURL = req.GetImageUrl()
	err = models.UpdateStoryRole(ctx, int64(roleinfo.ID), map[string]interface{}{
		"poster_url": req.GetImageUrl(),
	})
	if err != nil {
		log.Log().Error("update story role poster failed", zap.Error(err))
		return nil, err
	}
	log.Log().Info("update story role poster success", zap.Any("roleinfo", roleinfo))
	return &api.UpdateStoryRolePosterResponse{
		Code:    0,
		Message: "OK",
	}, nil
}

func (s *StoryService) GenerateStoryRolePoster(ctx context.Context, req *api.GenerateStoryRolePosterRequest) (*api.GenerateStoryRolePosterResponse, error) {
	roleinfo, err := models.GetStoryRoleByID(ctx, req.GetRoleId())
	if err != nil {
		log.Log().Error("get story role by id failed", zap.Error(err))
		return nil, err
	}
	if roleinfo.CreatorID != req.GetUserId() {
		log.Log().Error("have no permission", zap.Any("roleinfo", roleinfo))
		return nil, errors.New("have no permission")
	}

	return &api.GenerateStoryRolePosterResponse{
		Code:    0,
		Message: "OK",
	}, nil
}
