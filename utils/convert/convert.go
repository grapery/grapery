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
	return &api.GroupInfo{}
}

func ConvertProjectToApiProjectInfo(p *models.Project) *api.ProjectInfo {
	return &api.ProjectInfo{}
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

// func ConvertTeamToApiTeamInfo(t *models.Team) *api.TeamInfo {
// 	return &api.TeamInfo{}
// }
