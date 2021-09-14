package models

import (
	_ "database/sql"
	_ "encoding/json"
	"fmt"
	_ "time"

	log "github.com/sirupsen/logrus"

	"github.com/grapery/grapery/api"
)

type ChatSetting int

const (
	NoLimit           ChatSetting = 0
	AtleastOneGroup   ChatSetting = 1
	AtleastThreeGroup ChatSetting = 2
	Forbiden          ChatSetting = 4
)

/*
 */
type User struct {
	IDBase   `json:"id_base,omitempty"`
	Name     string         `json:"name,omitempty" gorm:"index"`
	Email    string         `json:"email,omitempty" gorm:"index"`
	Phone    string         `json:"phone,omitempty" gorm:"index"`
	Gender   int            `json:"gender,omitempty"`
	BioID    string         `json:"bio,omitempty"`
	Status   api.UserStatus `json:"status,omitempty"`
	Location string         `json:"location,omitempty"`
	Emotion  int            `json:"emotion,omitempty"`
	Avatar   string         `json:"avatar,omitempty"`

	URL          string `json:"url,omitempty"`
	NumFollowing int    `json:"num_following,omitempty"`
	NumProjects  int    `json:"num_projects,omitempty"`
	NumGroup     int    `json:"num_group,omitempty"`
	NumTeams     int    `json:"num_teams,omitempty"`
	ShortDesc    string `json:"short_desc,omitempty"`
}

func (u User) TableName() string {
	return "users"
}

func (u *User) Create() error {
	err := database.Model(u).Create(u).First(u).Error
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

func (u *User) UpdateAll() error {
	err := database.Model(u).
		Update("avatar", u.Avatar).
		Update("short_desc", u.ShortDesc).
		Update("name", u.Name).
		Where("id = ?", u.ID).Error
	if err != nil {
		log.Errorf("update user [%d] all [%s] failed ", u.ID, u.Avatar)
		return fmt.Errorf("update user [%d] all failed ", u.ID)
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
	err := database.Model(u).Where("name = ? and deleted = ? ", u.Name, 0).First(u).Error
	if err != nil {
		log.Errorf("get user [%s] info failed : [%s]", u.Name, err.Error())
		return fmt.Errorf("get user [%s] info failed ", u.Name)
	}
	return nil
}

func (u *User) GetByPhone() error {
	err := database.Model(u).Where("phone = ? and deleted = ?", u.Phone, 0).First(u).Error
	if err != nil {
		log.Errorf("get user [%d] info failed : [%s]", u.ID, err.Error())
		return fmt.Errorf("get user [%s] info failed ", u.Phone)
	}
	return nil
}

func (u *User) GetByEmail() error {
	err := database.Model(u).Where("email = ? and deleted = ?", u.Email, 0).First(u).Error
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

func GetUsersByIds(ids []int) (users []*User, err error) {
	if len(ids) == 0 {
		return nil, nil
	}
	err = database.Model(User{}).Where("id in (?)", ids).Scan(&users).Error
	if err != nil {
		return nil, err
	}
	return users, nil
}
