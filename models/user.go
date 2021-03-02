package models

import (
	_ "database/sql"
	_ "encoding/json"
	"fmt"
	_ "time"

	"github.com/jinzhu/gorm"
	log "github.com/sirupsen/logrus"
)

type User struct {
	IDBase
	Name     string `json:"name,omitempty" gorm:"index"`
	UserType int    `json:"user_type,omitempty"`
	Email    string `json:"email,omitempty" gorm:"index"`
	Phone    string `json:"phone,omitempty" gorm:"index"`
	Gender   int    `json:"gender,omitempty"`
	BioID    uint   `json:"bio,omitempty"`
	Location string `json:"location,omitempty"`

	Avatar string `json:"avatar,omitempty"`
	URL    string `json:"url,omitempty"`
}

func (u User) TableNamse() string {
	return "users"
}

func (u *User) Create() error {
	database.Where("phone  = ? and deleted = ?", u.Phone, 0).Find(u)
	var ret *gorm.DB
	if u.IDBase.ID != 0 {
		ret = database.Create(u)
	} else {
		log.Errorf("user [%s] is exist : ", u.IDBase.ID)
		return fmt.Errorf("user [%s] is exist", u.Phone)
	}
	if ret.Error != nil {
		log.Errorf("create user [%s] failed [%s] ", u.Phone, ret.Error)
		return fmt.Errorf("create user failed")
	}
	return nil
}

func (u *User) UpdateName() error {
	var ret *gorm.DB
	ret = database.Model(u).Update("name", u.Name)
	if ret.Error != nil {
		log.Errorf("update user [%d] name failed ", u.IDBase.ID)
		return fmt.Errorf("update user [%d] name failed ", u.IDBase.ID)
	}
	return nil
}

func (u *User) UpdateBio() error {
	var ret *gorm.DB
	ret = database.Model(u).Update("bio", u.BioID)
	if ret.Error != nil {
		log.Errorf("update user [%d] bio [%d] failed ", u.IDBase.ID, u.BioID)
		return fmt.Errorf("update user [%d] bio failed ", u.IDBase.ID)
	}
	return nil
}

func (u *User) UpdateAvatar() error {
	var ret *gorm.DB
	ret = database.Model(u).Update("avatar", u.Avatar)
	if ret.Error != nil {
		log.Errorf("update user [%d] avatar [%s] failed ", u.IDBase.ID, u.Avatar)
		return fmt.Errorf("update user [%d] avatar failed ", u.IDBase.ID)
	}
	return nil
}

func (u *User) GetById() error {
	var ret *gorm.DB
	ret = database.First(u)
	if ret.Error != nil {
		log.Errorf("get user [%d] info failed : [%s]", u.ID, ret.Error.Error())
		return fmt.Errorf("get user [%d] info failed ", u.ID)
	}
	return nil
}

func (u *User) GetByName() error {
	var ret *gorm.DB
	ret = database.Where("name = ? and deleted = ? ", u.Name, 0).Find(u)
	if ret.Error != nil {
		log.Errorf("get user [%s] info failed : [%s]", u.Name, ret.Error.Error())
		return fmt.Errorf("get user [%s] info failed ", u.Name)
	}
	return nil
}

func (u *User) GetByPhone() error {
	var ret *gorm.DB
	ret = database.Where("phone = ? and deleted = ?", u.Phone, 0).Find(u)
	if ret.Error != nil {
		log.Errorf("get user [%d] info failed : [%s]", u.ID, ret.Error.Error())
		return fmt.Errorf("get user [%s] info failed ", u.Phone)
	}
	return nil
}

func (u *User) GetByEmail() error {
	var ret *gorm.DB
	ret = database.Where("email = ? and deleted = ?", u.Email, 0).Find(u)
	if ret.Error != nil {
		log.Errorf("get user [%d] info failed : [%s]", u.ID, ret.Error.Error())
		return fmt.Errorf("get user [%s] info failed ", u.Email)
	}
	return nil
}

func (u *User) Delete() error {
	var ret *gorm.DB
	ret = database.Model(u).Update("deleted", 1)
	if ret.Error != nil {
		log.Errorf("update user [%d] deleted failed ", u.IDBase.ID)
		return fmt.Errorf("deleted user [%d] failed ", u.IDBase.ID)
	}
	return nil
}
