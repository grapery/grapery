package convert

import (
	"encoding/json"

	log "github.com/sirupsen/logrus"

	api "github.com/grapery/grapery/api"
	"github.com/grapery/grapery/models"
)

func ConvertActiveToApiActiveInfo(ac *models.Active) *api.ActiveInfo {
	return &api.ActiveInfo{
		ActiveType:  ac.ActiveType,
		User:        &api.UserInfo{UserId: ac.UserId},
		ItemInfo:    &api.ItemInfo{},
		ProjectInfo: &api.ProjectInfo{ProjectId: ac.ProjectID},
	}
}

func ConvertUserToApiUser(user *models.User) *api.UserInfo {
	return &api.UserInfo{
		UserId:   uint64(user.ID),
		Name:     user.Name,
		Avatar:   user.Avatar,
		Email:    user.Email,
		Location: user.Location,
		Desc:     user.ShortDesc,
	}
}

func ConvertItemToInfo(item *models.Item) *api.ItemInfo {
	info := new(api.ItemInfo)
	info.UserId = item.UserID
	info.Content = nil
	info.ProjectId = item.ProjectID
	info.Itype = item.ItemType
	info.Title = item.Description
	return info
}

func ConvertInfoToItem(info *api.ItemInfo) *models.Item {
	item := new(models.Item)
	item.UserID = info.UserId
	item.Description = info.Title
	item.ProjectID = info.ProjectId
	item.ItemType = info.Itype
	return item
}

func ConvertGroupToApiGroupInfo(g *models.Group) *api.GroupInfo {
	return &api.GroupInfo{}
}

func ConvertProjectToApiProjectInfo(p *models.Project) *api.ProjectInfo {
	return &api.ProjectInfo{}
}

func ConvertItemToApiItemInfo(i *models.Item) *api.ItemInfo {
	info := &api.ItemInfo{
		ProjectId: i.ProjectID,
		UserId:    i.UserID,
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
	case api.ItemType_Location:
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
