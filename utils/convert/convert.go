package convert

import (
	api "github.com/grapery/grapery/api"
	"github.com/grapery/grapery/models"
)

func ConvertActiveToApiActiveInfo(ac *models.Active) *api.ActiveInfo {
	return &api.ActiveInfo{
		ActiveType:  ac.ActiveType,
		User:        &api.UserInfo{UserId: ac.UserId},
		ItemInfo:    &api.ItemInfo{},
		ProjectInfo: &api.ProjectInfo{ProjectId: ac.ProjectID},
		GroupInfo:   &api.GroupInfo{GroupId: ac.GroupID},
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

func ConvertGroupToApiGroupInfo(g *models.Group) *api.GroupInfo {
	return &api.GroupInfo{}
}

func ConvertProjectToApiProjectInfo(p *models.Project) *api.ProjectInfo {
	return &api.ProjectInfo{}
}

func ConvertItemToApiItemInfo(i *models.Item) *api.ItemInfo {
	return &api.ItemInfo{}
}

// func ConvertTeamToApiTeamInfo(t *models.Team) *api.TeamInfo {
// 	return &api.TeamInfo{}
// }
