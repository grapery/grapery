package models

import (
	_ "database/sql"
	_ "encoding/json"
	"fmt"
	_ "time"

	log "github.com/sirupsen/logrus"

	"github.com/grapery/grapery/api"
)

/*
普通用户：
机器人用户：
*/
type User struct {
	IDBase
	Name         string         `json:"name,omitempty" gorm:"index"`
	Email        string         `json:"email,omitempty" gorm:"index"`
	Phone        string         `json:"phone,omitempty" gorm:"index"`
	Gender       int            `json:"gender,omitempty"`
	BioID        string         `json:"bio,omitempty"`
	Status       api.UserStatus `json:"status,omitempty"`
	Location     string         `json:"location,omitempty"`
	Emotion      int            `json:"emotion,omitempty"`
	Avatar       string         `json:"avatar,omitempty"`
	URL          string         `json:"url,omitempty"`
	NumFollowers int            `json:"num_followers,omitempty"`
	NumFollowing int            `json:"num_following,omitempty"`
	NumStars     int            `json:"num_stars,omitempty"`
	NumProjects  int            `json:"num_projects,omitempty"`
	NumGroup     int            `json:"num_group,omitempty"`
	NumTeams     int            `json:"num_teams,omitempty"`
	ShortDesc    string         `json:"short_desc,omitempty"`
}

func (u User) TableName() string {
	return "users"
}

func (u *User) Create() error {
	err := database.Create(u).First(u).Error
	if err != nil {
		log.Errorf("create user [%s/%s] failed [%s] ", u.Phone, u.Email, err.Error())
		return fmt.Errorf("create user failed")
	}
	return nil
}

func (u *User) UpdateName() error {
	err := database.Model(u).Update("name", u.Name).Where("id = ?", u.ID).Error
	if err != nil {
		log.Errorf("update user [%d] name failed ", u.ID)
		return fmt.Errorf("update user [%d] name failed ", u.ID)
	}
	return nil
}

func (u *User) UpdateBio() error {
	err := database.Model(u).Update("bio", u.BioID).Where("id = ?", u.ID).Error
	if err != nil {
		log.Errorf("update user [%d] bio [%d] failed ", u.ID, u.BioID)
		return fmt.Errorf("update user [%d] bio failed ", u.ID)
	}
	return nil
}

func (u *User) UpdateAvatar() error {
	err := database.Model(u).Update("avatar", u.Avatar).Where("id = ?", u.ID).Error
	if err != nil {
		log.Errorf("update user [%d] avatar [%s] failed ", u.ID, u.Avatar)
		return fmt.Errorf("update user [%d] avatar failed ", u.ID)
	}
	return nil
}

func (u *User) GetById() error {
	err := database.Model(u).Where("id = ? and deleted = ?", u.ID, 0).First(u).Error
	if err != nil {
		log.Errorf("get user [%d] info failed : [%s]", u.ID, err.Error())
		return fmt.Errorf("get user [%d] info failed ", u.ID)
	}
	return nil
}

func (u *User) GetByName() error {
	err := database.Where("name = ? and deleted = ? ", u.Name, 0).First(u).Error
	if err != nil {
		log.Errorf("get user [%s] info failed : [%s]", u.Name, err.Error())
		return fmt.Errorf("get user [%s] info failed ", u.Name)
	}
	return nil
}

func (u *User) GetByPhone() error {
	err := database.Where("phone = ? and deleted = ?", u.Phone, 0).First(u).Error
	if err != nil {
		log.Errorf("get user [%d] info failed : [%s]", u.ID, err.Error())
		return fmt.Errorf("get user [%s] info failed ", u.Phone)
	}
	return nil
}

func (u *User) GetByEmail() error {
	err := database.Where("email = ? and deleted = ?", u.Email, 0).First(u).Error
	if err != nil {
		log.Errorf("get user [%d] info failed : [%s]", u.ID, err.Error())
		return fmt.Errorf("get user [%s] info failed ", u.Email)
	}
	return nil
}

func (u *User) Delete() error {
	err := database.Model(u).Update("deleted", 1).Where("id = ? ", u.ID).Error
	if err != nil {
		log.Errorf("update user [%d] deleted failed ", u.ID)
		return fmt.Errorf("deleted user [%d] failed ", u.ID)
	}
	return nil
}
