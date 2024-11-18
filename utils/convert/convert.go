package convert

import (
	"encoding/json"

	log "github.com/sirupsen/logrus"

	api "github.com/grapery/common-protoc/gen"
	"github.com/grapery/grapery/models"
)

func ConvertActiveToApiActiveInfo(ac *models.Active) *api.ActiveInfo {
	return &api.ActiveInfo{
		ActiveType:  ac.ActiveType,
		User:        &api.UserInfo{UserId: int64(ac.UserId)},
		ItemInfo:    &api.ItemInfo{},
		ProjectInfo: &api.ProjectInfo{},
	}
}

func ConvertUserToApiUser(user *models.User) *api.UserInfo {
	return &api.UserInfo{
		UserId:   int64(user.ID),
		Name:     user.Name,
		Avatar:   user.Avatar,
		Email:    user.Email,
		Location: user.Location,
		Desc:     user.ShortDesc,
	}
}

func ConvertItemToInfo(item *models.StoryItem) *api.ItemInfo {
	info := new(api.ItemInfo)
	info.UserId = int64(item.UserID)
	info.Content = nil
	info.ProjectId = int64(item.ProjectID)
	info.Itype = item.ItemType
	info.Title = item.Description
	return info
}

func ConvertInfoToItem(info *api.ItemInfo) *models.StoryItem {
	item := new(models.StoryItem)
	item.UserID = int64(info.UserId)
	item.Description = info.Title
	item.ProjectID = int64(info.ProjectId)
	item.ItemType = info.Itype
	return item
}

func ConvertGroupToApiGroupInfo(g *models.Group) *api.GroupInfo {
	return &api.GroupInfo{
		GroupId: int64(g.ID),
		Name:    g.Name,
		Avatar:  g.Avatar,
		Owner:   g.OwnerID,
		Desc:    g.ShortDesc,
		Creator: g.CreatorID,
		Ctime:   g.CreateAt.Unix(),
		Mtime:   g.UpdateAt.Unix(),
	}
}

func ConvertProjectToApiProjectInfo(p *models.Project) *api.ProjectInfo {
	return &api.ProjectInfo{
		ProjectId: uint64(p.ID),
		Name:      p.Name,
		Avatar:    p.Avatar,
		Owner:     p.OwnerID,
	}
}

func ConvertItemToApiItemInfo(i *models.StoryItem) *api.ItemInfo {
	info := &api.ItemInfo{
		ProjectId: int64(i.ProjectID),
		UserId:    int64(i.UserID),
		Title:     i.Title,
		Itype:     i.ItemType,
	}
	var err error
	itemDetail := new(api.ItemDetail)
	switch i.ItemType {
	case api.ItemType_Link:
		shareLink := new(api.ShareDetail)
		err = json.Unmarshal([]byte(i.Content), shareLink)
		itemDetail.Detail = &api.ItemDetail_Share{
			Share: shareLink,
		}
	case api.ItemType_Picture:
		shareLink := new(api.PictureDetail)
		err = json.Unmarshal([]byte(i.Content), shareLink)
		itemDetail.Detail = &api.ItemDetail_Pictures{
			Pictures: shareLink,
		}
	case api.ItemType_ShortWord:
		shareLink := new(api.WordDetail)
		err = json.Unmarshal([]byte(i.Content), shareLink)
		itemDetail.Detail = &api.ItemDetail_Word{
			Word: shareLink,
		}
	case api.ItemType_Video:
		shareLink := new(api.VideoDetail)
		err = json.Unmarshal([]byte(i.Content), shareLink)
		itemDetail.Detail = &api.ItemDetail_Video{
			Video: shareLink,
		}
	}
	if err != nil {
		log.Errorf("convert item failed: %v", err)
		return nil
	}

	return info
}

func ConvertStoryToApiStory(story *models.Story) *api.Story {
	ret := &api.Story{
		Id:          int64(story.ID),
		Name:        story.Name,
		Avatar:      story.Avatar,
		CreatorId:   int64(story.CreatorID),
		OwnerId:     int64(story.OwnerID),
		GroupId:     int64(story.GroupID),
		Visable:     story.Visable,
		IsAchieve:   story.IsAchieve,
		IsClose:     story.IsClose,
		IsAiGen:     story.AIGen,
		Origin:      story.Origin,
		RootBoardId: int64(story.RootBoardID),
		Desc:        story.ShortDesc,
		Status:      int32(story.Status),
		Ctime:       story.CreateAt.Unix(),
		Mtime:       story.UpdateAt.Unix(),
	}
	if ret.Avatar == "" {
		ret.Avatar = "https://grapery-1301865260.cos.ap-shanghai.myqcloud.com/avator/tmp3evp1xxl.png"
	}
	json.Unmarshal([]byte(story.Params), &ret.Params)
	return ret
}

func ConvertGroupProfileToApiGroupProfile(p *models.GroupProfile) *api.GroupProfileInfo {
	return &api.GroupProfileInfo{
		GroupId:          p.GroupID,
		GroupMemberNum:   int32(p.Members),
		GroupProjectNum:  int32(p.StoryCount),
		IsVerified:       p.IsVerified,
		Description:      p.Desc,
		GroupFollowerNum: int32(p.Followers),
		Ctime:            p.CreateAt.Unix(),
		Mtime:            p.UpdateAt.Unix(),
	}
}

func ConvertStoryBoardSceneToApiStoryBoardScene(scene *models.StoryBoardScene) *api.StoryBoardSence {
	return &api.StoryBoardSence{
		SenceId:      int64(scene.ID),
		StoryId:      int64(scene.StoryId),
		BoardId:      int64(scene.BoardId),
		CreatorId:    int64(scene.CreatorId),
		Content:      scene.Content,
		ImagePrompts: scene.ImagePrompts,
		AudioPrompts: scene.AudioPrompts,
		VideoPrompts: scene.VideoPrompts,
		GenResult:    scene.GenResult,
		IsGenerating: int32(scene.IsGenerating),
		Status:       int32(scene.Status),
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
		GenResult:    scene.GenResult,
		IsGenerating: int(scene.IsGenerating),
		Status:       int(scene.Status),
	}
}

func ConvertStoryRoleToApiStoryRoleInfo(role *models.StoryRole) *api.StoryRole {
	return &api.StoryRole{
		RoleId:        int64(role.ID),
		CharacterName: role.CharacterName,
	}
}
