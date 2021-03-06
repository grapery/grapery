package models

import (
	"fmt"

	"github.com/jinzhu/gorm"
	log "github.com/sirupsen/logrus"

	api "github.com/grapery/grapery/api"
	"github.com/grapery/grapery/utils/errors"
)

/*
授权/注册/登陆记录表
用做登陆
用做用户注册，注册成功才生成新的用户信息
*/
type Auth struct {
	IDBase
	UID      uint64       `json:"uid,omitempty" gorm:"unique_index,column:"`
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

func (a *Auth) CreateUsePhone() error {
	err := database.Table(a.TableName()).Where("phone = ? and deleted = ? and is_valid = ?", a.Phone, 0, true).Find(a).Error
	if err != nil {
		return err
	}
	var ret *gorm.DB
	if a.IDBase.ID == 0 {
		ret = database.Create(a)
	} else {
		log.Errorf("auth [%s] is exist : ", a.IDBase.ID)
		return errors.ErrUserIsExist
	}
	if ret.Error != nil {
		log.Errorf("create auth [%s] failed [%s] ", a.Phone, ret.Error)
		return errors.ErrCreateAuthFailed
	}
	return nil
}

func (a *Auth) CreateUseEmail() error {
	err := database.Table(a.TableName()).Where("email = ? and is_valid in (?)", a.Email, 0, true).Find(a).Error
	if err != nil {
		return err
	}
	var ret *gorm.DB
	if a.IDBase.ID == 0 {
		ret = database.Create(a)
	} else {
		log.Errorf("auth [%s] is exist : ", a.IDBase.ID)
		return errors.ErrUserIsExist
	}
	if ret.Error != nil {
		log.Errorf("create auth [%s] failed [%s] ", a.Phone, ret.Error)
		return errors.ErrCreateAuthFailed
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
	if err := database.Table(a.TableName()).Find(a).Where("phone = ? and deleted = ? and is_valid = ?", a.Email, 0, true).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return errors.ErrAuthNotFound
		}
		log.Errorf("get auth [%s] info failed : [%s]", a.Email, err)
		return fmt.Errorf("get auth [%s] info failed ", a.Email)
	}
	return nil
}

func (a *Auth) GetByPhone() error {
	if err := database.Table(a.TableName()).Find(a).Where("phone = ? and deleted = ? and is_valid = ?", a.Phone, 0, true).Error; err != nil {
		log.Errorf("get auth [%s] info failed : [%s]", a.Phone, err)
		return fmt.Errorf("get auth [%s] info failed ", a.Phone)
	}
	return nil
}

func (a *Auth) GetByUID() error {
	if err := database.Table(a.TableName()).Find(a).Where("uid = ? and deleted = ? and is_valid = ?", a.UID, 0, true).Error; err != nil {
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
