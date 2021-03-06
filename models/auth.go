package models

import (
	"fmt"

	log "github.com/sirupsen/logrus"
	"gorm.io/gorm"

	api "github.com/grapery/grapery/api"
	"github.com/grapery/grapery/utils/errors"
)

type Auth struct {
	IDBase
	UID       uint64       `json:"uid,omitempty" gorm:"unique_index,column:uid"`
	Email     string       `json:"email,omitempty" gorm:"unique_index"`
	Phone     string       `json:"phone,omitempty" gorm:"unique_index"`
	Password  string       `json:"-" gorm:"password"`
	Salt      string       `json:"-" gorm:"salt"`
	IsValid   bool         `json:"is_valid,omitempty"`
	AuthType  api.AuthType `json:"auth_type,omitempty" gorm:"authtype"`
	Confirmed bool         `json:"confirmed,omitempty"`
}

func (a Auth) TableName() string {
	return "auth"
}

func (a *Auth) CreateWithPhone() error {
	err := database.Table(Auth{}.TableName()).Create(a).Error
	if err != nil {
		log.Errorf("create auth [%s] failed [%s] ", a.Phone, err.Error())
		return errors.ErrCreateAuthFailed
	}
	return nil
}

func IsUserAuthExist(account string) bool {
	var count int64
	err := database.Table(Auth{}.TableName()).Where("email = ? or phone = ?", account, account).Count(&count).Error
	if err != gorm.ErrRecordNotFound {
		return false
	}
	if count == 0 {
		return false
	}
	return true
}

func (a *Auth) CreateWithEmail() error {
	err := database.Table(Auth{}.TableName()).Create(a).Error
	if err != nil {
		log.Errorf("create auth [%s] failed [%s] ", a.Email, err.Error())
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
	if err := database.Table(a.TableName()).Find(a).Where("email = ? and deleted = ? and is_valid = ?", a.Email, 0, true).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return errors.ErrAuthNotFound
		}
		log.Errorf("get auth [%s] info failed : [%s]", a.Email, err)
		return fmt.Errorf("get auth [%s] info failed ", a.Email)
	}
	return nil
}

func (a *Auth) GetByPhone() error {
	if err := database.Table(a.TableName()).Find(a).Where("phone = ? and deleted = ? and is_valid = ?", a.Phone, 0, true).First(a).Error; err != nil {
		log.Errorf("get auth [%s] info failed : [%s]", a.Phone, err)
		return fmt.Errorf("get auth [%s] info failed ", a.Phone)
	}
	return nil
}

func (a *Auth) GetByUID() error {
	if err := database.Table(a.TableName()).Find(a).Where("uid = ? and deleted = ? and is_valid = ?", a.UID, 0, true).First(a).Error; err != nil {
		log.Errorf("get auth [%d] info failed : [%s]", a.UID, err)
		return fmt.Errorf("get auth [%d] info failed ", a.UID)
	}
	return nil
}

func (a *Auth) ConfirmAuth() error {
	if err := database.Table(a.TableName()).Update("confirmed", a.Confirmed).Where("uid = ? and deleted = ? ", a.UID, true).Error; err != nil {
		log.Errorf("update confirmed failed : [%s]", err.Error())
		return fmt.Errorf("update user [%d] confirmed failed : [%s]", a.UID, err.Error())
	}
	return nil
}

func (a *Auth) Delete() error {
	if err := database.Table(a.TableName()).Update("deleted", 1).Where("is_valid = ? ", true); err != nil {
		log.Errorf("update auth [%d] deleted failed ", a.ID)
		return fmt.Errorf("deleted auth [%d] failed ", a.ID)
	}
	return nil
}
