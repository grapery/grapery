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

// Auth 用户认证信息
type Auth struct {
	IDBase
	UID      int64        `gorm:"column:uid;uniqueIndex" json:"uid,omitempty"`     // 用户ID
	Email    string       `gorm:"column:email;uniqueIndex" json:"email,omitempty"` // 邮箱
	Phone    string       `gorm:"column:phone;uniqueIndex" json:"phone,omitempty"` // 手机号
	Password string       `gorm:"column:password" json:"password,omitempty"`       // 密码
	Token    string       `gorm:"column:token" json:"token,omitempty"`             // token
	Salt     string       `gorm:"column:salt" json:"salt,omitempty"`               // 盐
	IsValid  bool         `gorm:"column:is_valid" json:"is_valid,omitempty"`       // 是否有效
	AuthType api.AuthType `gorm:"column:auth_type" json:"auth_type,omitempty"`     // 认证类型
	Expired  int64        `gorm:"column:expired" json:"expired,omitempty"`         // 过期时间
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

// 新增：分页获取Auth列表
func GetAuthList(ctx context.Context, offset, limit int) ([]*Auth, error) {
	var auths []*Auth
	err := DataBase().Model(&Auth{}).
		WithContext(ctx).
		Offset(offset).
		Limit(limit).
		Order("create_at desc").
		Find(&auths).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return nil, err
	}
	return auths, nil
}

// 新增：通过Email唯一查询
func GetAuthByEmailUnique(ctx context.Context, email string) (*Auth, error) {
	auth := &Auth{}
	err := DataBase().Model(auth).
		WithContext(ctx).
		Where("email = ?", email).
		First(auth).Error
	if err != nil {
		return nil, err
	}
	return auth, nil
}
