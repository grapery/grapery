package models

import (
	"fmt"

	"gorm.io/gorm"

	api "github.com/grapery/grapery/api"
	"github.com/grapery/grapery/utils/errors"
	"github.com/grapery/grapery/utils/log"
)

type Auth struct {
	IDBase
	UID      uint64       `json:"uid,omitempty" gorm:"unique_index,column:uid"`
	Email    string       `json:"email,omitempty" gorm:"unique_index"`
	Phone    string       `json:"phone,omitempty" gorm:"unique_index"`
	Password string       `json:"-" gorm:"password"`
	Salt     string       `json:"-" gorm:"salt"`
	IsValid  bool         `json:"is_valid,omitempty"`
	AuthType api.AuthType `json:"auth_type,omitempty" gorm:"authtype"`
	Expired  bool         `json:"expired,omitempty"`
}

func (a Auth) TableName() string {
	return "auth"
}

func CreateWithPhone(a *Auth) error {
	err := DataBase().Model(a).Create(a).Error
	if err != nil {
		log.Log().WithOptions(logFieldModels).Error(
			fmt.Sprintf("create auth [%s] failed [%s] ", a.Phone, err.Error()),
		)
		return errors.ErrCreateAuthFailed
	}
	return nil
}

func IsUserAuthExist(account string) bool {
	var count int64
	err := DataBase().Model(Auth{}).
		Where("email = ? or phone = ?", account, account).
		Count(&count).Error
	if err != gorm.ErrRecordNotFound {
		log.Log().WithOptions(logFieldModels).Error(
			fmt.Sprintf("check user auth [%s] failed [%s] ", account, err.Error()),
		)
		return false
	}
	if count == 0 {
		return false
	}
	return true
}

func CreateWithEmail(a *Auth) error {
	err := DataBase().Model(a).Create(a).Error
	if err != nil {
		log.Log().WithOptions(logFieldModels).Error(
			fmt.Sprintf("create auth [%s] with email failed [%s] ", a.Email, err.Error()))
		return errors.ErrCreateAuthFailed
	}
	return nil
}

func UpdatePwd(a *Auth) error {
	if err := DataBase().Model(a).
		Update("password", a.Password).
		Where("is_valid = ? and uid = ?", true, a.UID).Error; err != nil {
		log.Log().WithOptions(logFieldModels).Error(fmt.Sprintf("update password failed : [%s]", err.Error()))
		return fmt.Errorf("update user [%d] password failed : [%s]", a.UID, err.Error())
	}
	return nil
}

func GetByEmail(email string) (*Auth, error) {
	var a = new(Auth)
	if err := DataBase().Model(a).Find(a).
		Where("email = ? and deleted = ? and is_valid = ?", email, 0, true).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.ErrAuthNotFound
		}
		log.Log().WithOptions(logFieldModels).Error(
			fmt.Sprintf("get auth [%s] info failed : [%s]", a.Email, err),
		)
		return nil, fmt.Errorf("get auth [%s] info failed ", email)
	}
	return a, nil
}

func (a *Auth) GetByPhone() error {
	if err := DataBase().Table(a.TableName()).Find(a).
		Where("phone = ? and deleted = ? and is_valid = ?", a.Phone, 0, true).
		First(a).Error; err != nil {
		log.Log().WithOptions(logFieldModels).Error(
			fmt.Sprintf("get auth [%s] info failed : [%s]", a.Phone, err),
		)
		return fmt.Errorf("get auth [%s] info failed ", a.Phone)
	}
	return nil
}

func (a *Auth) GetByUID() error {
	if err := DataBase().Table(a.TableName()).Find(a).
		Where("uid = ? and deleted = ? and is_valid = ?", a.UID, 0, true).
		First(a).Error; err != nil {
		log.Log().WithOptions(logFieldModels).Error(
			fmt.Sprintf("get auth [%d] info failed : [%s]", a.UID, err),
		)
		return fmt.Errorf("get auth [%d] info failed ", a.UID)
	}
	return nil
}

func (a *Auth) Delete() error {
	if err := DataBase().Table(a.TableName()).Update("deleted", 1).
		Where("is_valid = ? ", true); err != nil {
		log.Log().WithOptions(logFieldModels).Error(
			fmt.Sprintf("update auth [%d] deleted failed ", a.ID),
		)
		return fmt.Errorf("deleted auth [%d] failed ", a.ID)
	}
	return nil
}
