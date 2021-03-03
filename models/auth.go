package models

import (
	"errors"
	"fmt"

	"github.com/jinzhu/gorm"
	log "github.com/sirupsen/logrus"

	api "github.com/grapery/grapery/api"
)

var (
	ErrUserIsExist           = errors.New("user is exist")
	ErrCreateAuthFailed      = errors.New("create auth info failed")
	ErrResetPasswordFailed   = errors.New("reset password failed")
	ErrGetUserAuthInfoFailed = errors.New("get user auth info failed")
	ErrDeleteUserAuthInfo    = errors.New("delete uaer auth info failed")
)

/*
授权/注册/登陆记录表
用做登陆
用做用户注册，注册成功才生成新的用户信息
*/
type Auth struct {
	IDBase
	UID      uint64       `json:"uid,omitempty" gorm:"unique_index"`
	Email    string       `json:"email,omitempty" gorm:"unique_index"`
	Phone    string       `json:"phone,omitempty" gorm:"unique_index"`
	Password string       `json:"-" gorm:"password"`
	Salt     string       `json:"-" gorm:"salt"`
	IsValid  bool         `json:"is_valid,omitempty"`
	AuthType api.AuthType `json:"auth_type,omitempty" gorm:"authtype"`
}

func (a Auth) TableName() string {
	return "auth"
}

func (a *Auth) Create() error {
	database.Table(a.TableName()).Where("phone = ? and deleted = ? and is_valid = ?", a.Phone, 0, true).Find(a)
	var ret *gorm.DB
	if a.IDBase.ID == 0 {
		ret = database.Create(a)
	} else {
		log.Errorf("auth [%s] is exist : ", a.IDBase.ID)
		return ErrUserIsExist
	}
	if ret.Error != nil {
		log.Errorf("create auth [%s] failed [%s] ", a.Phone, ret.Error)
		return ErrCreateAuthFailed
	}
	return nil
}

func (a *Auth) UpdatePwd() error {
	if err := database.Table(a.TableName()).Update("password", a.Password).Where("is_valid = ? and uid = ?", true, a.UID).Error; err != nil {
		log.Errorf("update password failed : [%s]", err.Error())
		return fmt.Errorf("update user [%d] password failed : [%s]", a.UID, err.Error())
	}
	return nil
}

func (a *Auth) GetByEmail() error {
	if err := database.Table(a.TableName()).Where("phone = ? and deleted = ?", a.Email, 0).Find(a).Where("is_valid = ? ", true).Error; err != nil {
		log.Errorf("get auth [%s] info failed : [%s]", a.Email, err)
		return fmt.Errorf("get auth [%s] info failed ", a.Email)
	}
	return nil
}

func (a *Auth) GetByPhone() error {
	if err := database.Table(a.TableName()).Where("phone = ? and deleted = ?", a.Phone, 0).Find(a).Where("is_valid = ? ", true).Error; err != nil {
		log.Errorf("get auth [%s] info failed : [%s]", a.Phone, err)
		return fmt.Errorf("get auth [%s] info failed ", a.Phone)
	}
	return nil
}

func (a *Auth) GetByUID() error {
	if err := database.Table(a.TableName()).Where("uid = ? and deleted = ?", a.UID, 0).Find(a).Where("is_valid = ? ", true).Error; err != nil {
		log.Errorf("get auth [%d] info failed : [%s]", a.UID, err)
		return fmt.Errorf("get auth [%d] info failed ", a.UID)
	}
	return nil
}

func (a *Auth) Delete() error {
	if err := database.Table(a.TableName()).Update("deleted", 1).Where("is_valid = ? ", true); err != nil {
		log.Errorf("update auth [%d] deleted failed ", a.IDBase.ID)
		return fmt.Errorf("deleted auth [%d] failed ", a.IDBase.ID)
	}
	return nil
}
