package models

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"

	api "github.com/grapery/common-protoc/gen"
	"github.com/grapery/grapery/utils/errors"
	"github.com/grapery/grapery/utils/log"
)

type Auth struct {
	IDBase
	UID      int64        `json:"uid,omitempty" gorm:"unique_index,column:uid"`
	Email    string       `json:"email,omitempty" gorm:"unique_index"`
	Phone    string       `json:"phone,omitempty" gorm:"unique_index"`
	Password string       `json:"password,omitempty" gorm:"password"`
	Token    string       `json:"token,omitempty" gorm:"token"`
	Salt     string       `json:"salt,omitempty" gorm:"salt"`
	IsValid  bool         `json:"is_valid,omitempty"`
	AuthType api.AuthType `json:"auth_type,omitempty" gorm:"authtype"`
	Expired  int64        `json:"expired,omitempty"`
}

func (a Auth) TableName() string {
	return "auth"
}

func (a *Auth) Delete(ctx context.Context) error {
	if err := DataBase().WithContext(ctx).Table(a.TableName()).
		Update("deleted", 1).
		Where("is_valid = ? ", true).Error; err != nil {
		log.Log().WithOptions(logFieldModels).Error(
			fmt.Sprintf("update auth [%d] deleted failed ", a.ID),
		)
		return fmt.Errorf("deleted auth [%d] failed ", a.ID)
	}
	return nil
}

func CreateWithPhone(ctx context.Context, a *Auth) error {
	a.Expired = int64(time.Now().Unix()) + 3600*72
	err := DataBase().WithContext(ctx).Model(a).Create(a).Error
	if err != nil {
		log.Log().WithOptions(logFieldModels).Error(
			fmt.Sprintf("create auth [%s] failed [%s] ", a.Phone, err.Error()),
		)
		return errors.ErrCreateAuthFailed
	}
	return nil
}

func IsUserAuthExist(ctx context.Context, account string) bool {
	var accountInfo = new(Auth)
	err := DataBase().WithContext(ctx).Model(Auth{}).
		Where("email = ? or phone = ?", account, account).
		Order("create_at").
		Limit(1).
		First(&accountInfo).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		log.Log().WithOptions(logFieldModels).Error(
			fmt.Sprintf("check user auth [%s] failed [%s] ", account, err.Error()),
		)
		return true
	}
	if err != nil && err == gorm.ErrRecordNotFound {
		log.Log().WithOptions(logFieldModels).Error(
			fmt.Sprintf("user auth [%s] not exist", account),
		)
		return false
	}
	if accountInfo.CreateAt.Unix() == 0 {
		return false
	}
	if accountInfo.Expired < int64(time.Now().Unix()) {
		return false
	}
	return true
}

func CreateWithEmail(ctx context.Context, a *Auth) error {
	a.Expired = int64(time.Now().Unix()) + 3600*72 // 3 days
	err := DataBase().WithContext(ctx).Model(a).Create(a).Error
	if err != nil {
		log.Log().WithOptions(logFieldModels).Error(
			fmt.Sprintf("create auth [%s] with email failed [%s] ", a.Email, err.Error()))
		return errors.ErrCreateAuthFailed
	}
	return nil
}

func UpdatePwd(ctx context.Context, a *Auth) error {
	if err := DataBase().WithContext(ctx).Model(a).
		Update("password", a.Password).
		Where("is_valid = ? and uid = ?", true, a.UID).Error; err != nil {
		log.Log().WithOptions(logFieldModels).Error(fmt.Sprintf("update password failed : [%s]", err.Error()))
		return fmt.Errorf("update user [%d] password failed : [%s]", a.UID, err.Error())
	}
	return nil
}

func GetByEmail(ctx context.Context, email string) (*Auth, error) {
	var a = new(Auth)
	if err := DataBase().WithContext(ctx).Model(a).Where("email = ?", email).First(a).
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

func GetByPhone(ctx context.Context, phone string) (*Auth, error) {
	var a = new(Auth)
	if err := DataBase().WithContext(ctx).Model(a).
		Where("phone = ?", phone).
		Where("deleted = ?", 0).
		First(a).
		Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.ErrAuthNotFound
		}
		log.Log().WithOptions(logFieldModels).Error(
			fmt.Sprintf("get auth [%s] info failed : [%s]", a.Phone, err),
		)
		return nil, fmt.Errorf("get auth [%s] info failed ", phone)
	}
	return a, nil
}

func GetByUID(ctx context.Context, uid int) (*Auth, error) {
	var a = new(Auth)
	if err := DataBase().WithContext(ctx).Model(a).
		Where("id = ? and deleted = ? and is_valid = ?", uid, 0, true).
		First(a).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.ErrAuthNotFound
		}
		log.Log().WithOptions(logFieldModels).Error(
			fmt.Sprintf("get auth [%d] info failed : [%s]", uid, err),
		)
		return nil, fmt.Errorf("get auth [%d] info failed ", uid)
	}
	return a, nil
}
