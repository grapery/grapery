package convert

import (
	"encoding/json"
	"strings"

	api "github.com/grapery/common-protoc/gen"
	"github.com/grapery/grapery/models"
)

func ConvertUserToApiUser(user *models.User) *api.UserInfo {
	return &api.UserInfo{
		UserId:   int64(user.ID),
		Name:     user.Name,
		Avatar:   user.Avatar,
		Email:    user.Email,
		Location: user.Location,
		Desc:     user.ShortDesc,
		Ctime:    user.CreateAt.Unix(),
		Mtime:    user.UpdateAt.Unix(),
	}
}

func ConvertGroupToApiGroupInfo(g *models.Group) *api.GroupInfo {
	return &api.GroupInfo{
		GroupId:           int64(g.ID),
		Name:              g.Name,
		Avatar:            g.Avatar,
		Owner:             g.OwnerID,
		Desc:              g.ShortDesc,
		Creator:           g.CreatorID,
		CurrentUserStatus: &api.WhatCurrentUserStatus{},
		Ctime:             g.CreateAt.Unix(),
		Mtime:             g.UpdateAt.Unix(),
	}
}

func ConvertStoryToApiStory(story *models.Story) *api.Story {
	ret := &api.Story{
		Id:           int64(story.ID),
		Name:         story.Name,
		Title:        story.Title,
		Avatar:       story.Avatar,
		CreatorId:    int64(story.CreatorID),
		OwnerId:      int64(story.OwnerID),
		GroupId:      int64(story.GroupID),
		Visable:      story.ScopeType,
		IsClose:      story.IsClose,
		IsAiGen:      story.AIGen,
		Origin:       story.Origin,
		RootBoardId:  int64(story.RootBoardID),
		Desc:         story.ShortDesc,
		Status:       int32(story.Status),
		TotalBoards:  story.TotalBoards,
		TotalRoles:   story.TotalRoles,
		TotalMembers: story.TotalMembers,
		LikeCount:    story.LikeCount,
		CommentCount: story.CommentCount,
		ShareCount:   story.ShareCount,
		FollowCount:  story.FollowCount,
		Style:        story.Style,
		Cover:        story.Cover,
		Ctime:        story.CreateAt.Unix(),
		Mtime:        story.UpdateAt.Unix(),
	}
	if ret.Avatar == "" {
		ret.Avatar = "https://grapery-dev.oss-cn-shanghai.aliyuncs.com/default.jpg"
	}
	if ret.Cover == "" {
		ret.Cover = "https://grapery-dev.oss-cn-shanghai.aliyuncs.com/default.jpg"
	}
	json.Unmarshal([]byte(story.Params), &ret.Params)
	return ret
}

func ConvertGroupProfileToApiGroupProfile(p *models.GroupProfile) *api.GroupProfileInfo {
	return &api.GroupProfileInfo{
		GroupId:          p.GroupID,
		GroupMemberNum:   int32(p.Members),
		Description:      p.Desc,
		GroupFollowerNum: int32(p.Followers),
		GroupStoryNum:    int32(p.StoryCount),
		BackgroudUrl:     p.BackgroundUrl,
		Ctime:            p.CreateAt.Unix(),
		Mtime:            p.UpdateAt.Unix(),
	}
}

func ConvertStoryBoardSceneToApiStoryBoardScene(scene *models.StoryBoardScene) *api.StoryBoardSence {
	characterIds := strings.Split(scene.CharacterIds, ",")
	return &api.StoryBoardSence{
		SenceId:      int64(scene.ID),
		StoryId:      int64(scene.StoryId),
		BoardId:      int64(scene.BoardId),
		CreatorId:    int64(scene.CreatorId),
		CharacterIds: characterIds,
		Content:      scene.Content,
		ImagePrompts: scene.ImagePrompts,
		AudioPrompts: scene.AudioPrompts,
		VideoPrompts: scene.VideoPrompts,
		ImageUrl:     scene.ImageUrl,
		IsGenerating: int32(scene.GenStatus),
		Status:       int32(scene.Status),
		Images:       []string{scene.ImageUrl},
		VideoUrl:     scene.VideoUrl,
		AudioUrl:     scene.AudioUrl,
		Ctime:        scene.CreateAt.Unix(),
		Mtime:        scene.UpdateAt.Unix(),
	}
}

func ConvertApiStoryBoardSceneToStoryBoardScene(scene *api.StoryBoardSence) *models.StoryBoardScene {
	return &models.StoryBoardScene{
		StoryId:      int64(scene.StoryId),
		BoardId:      int64(scene.BoardId),
		CreatorId:    int64(scene.CreatorId),
		Content:      scene.Content,
		ImagePrompts: scene.ImagePrompts,
		AudioPrompts: scene.AudioPrompts,
		VideoPrompts: scene.VideoPrompts,
		ImageUrl:     scene.ImageUrl,
		VideoUrl:     scene.VideoUrl,
		AudioUrl:     scene.AudioUrl,
		GenStatus:    api.StoryGenStatus(scene.IsGenerating),
	}
}

func ConvertStoryRoleToApiStoryRoleInfo(role *models.StoryRole) *api.StoryRole {
	roleDetail := &api.StoryRole{
		RoleId:               int64(role.ID),
		CharacterName:        role.CharacterName,
		CharacterDescription: role.CharacterDescription,
		CharacterAvatar:      role.CharacterAvatar,
		CreatorId:            int64(role.CreatorID),
		Status:               int32(role.Status),
		LikeCount:            int64(role.LikeCount),
		FollowCount:          int64(role.FollowCount),
		StoryboardNum:        int64(role.StoryboardNum),
		Version:              int64(role.Version),
		Ctime:                role.CreateAt.Unix(),
		Mtime:                role.UpdateAt.Unix(),
	}
	return roleDetail
}

func ConvertSummaryStoryRoleToApiStoryRoleInfo(role *models.StoryBoardRole) *api.StoryRole {
	return &api.StoryRole{
		RoleId:               int64(role.ID),
		CharacterName:        role.Name,
		CharacterDescription: role.Desc,
		CharacterAvatar:      role.Avatar,
		CreatorId:            int64(role.CreatorId),
		Ctime:                role.CreateAt.Unix(),
		Mtime:                role.UpdateAt.Unix(),
	}
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
		ForkNum:      int64(storyBoard.ForkNum),
	}
	ret.IsMultiBranch = storyBoard.ForkNum > 1
	_ = json.Unmarshal([]byte(storyBoard.Params), &ret.Params)
	return ret
}

func ConvertChatMessageToApiChatMessage(chatMessage *models.ChatMessage) *api.ChatMessage {
	return &api.ChatMessage{
		Id:        int64(chatMessage.ID),
		ChatId:    int64(chatMessage.ChatContextID),
		UserId:    int64(chatMessage.UserID),
		RoleId:    int64(chatMessage.RoleID),
		Sender:    int32(chatMessage.Sender),
		Message:   chatMessage.Content,
		Timestamp: chatMessage.CreateAt.Unix(),
	}
}

func ConvertApiChatMessageToChatMessage(chatMessage *api.ChatMessage) *models.ChatMessage {
	return &models.ChatMessage{}
}

func ConvertActiveToApiActive(active *models.Active) *api.ActiveInfo {
	return &api.ActiveInfo{
		ActiveId:   int64(active.ID),
		ActiveType: active.ActiveType,
	}
}
