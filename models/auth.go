package models

import (
	"fmt"

	"github.com/jinzhu/gorm"
	log "github.com/sirupsen/logrus"
)

type RegisterType uint32

const (
	_                 RegisterType = iota
	RegisterWithPhone RegisterType = 1
	RegisterWithEmail RegisterType = 2
)

func RegisterTypeName(rtype RegisterType) string {
	switch rtype {
	case RegisterWithPhone:
		return "phone"
	case RegisterWithEmail:
		return "email"
	}
	return "unknown"

}

type Auth struct {
	IDBase
	UID      uint64       `json:"uid,omitempty" gorm:"unique_index"`
	Email    string       `json:"email,omitempty" gorm:"unique_index"`
	Phone    string       `json:"phone,omitempty" gorm:"unique_index"`
	Password string       `json:"-" gorm:"password"`
	Salt     string       `json:"-" gorm:"salt"`
	AuthType RegisterType `json:"auth_type,omitempty" gorm:"authtype"`
}

func (a Auth) TableNamse() string {
	return "auth"
}

func (a *Auth) Create() error {
	database.Where("phone = ? and deleted = ?", a.Phone, 0).Find(a)
	var ret *gorm.DB
	if a.IDBase.ID != 0 {
		ret = database.Create(a)
	} else {
		log.Errorf("auth [%s] is exist : ", a.IDBase.ID)
		return fmt.Errorf("auth [%s] is exist", a.Phone)
	}
	if ret.Error != nil {
		log.Errorf("create auth [%s] failed [%s] ", a.Phone, ret.Error)
		return fmt.Errorf("create auth failed")
	}
	return nil
}

func (a *Auth) UpdatePwd() error {
	if err := database.Model(a).Update("password", a.Password).Error; err != nil {
		log.Errorf("update password failed : [%s]", err.Error())
		return fmt.Errorf("update user [%d] password failed : [%s]", a.UID, err.Error())
	}
	return nil
}

func (a *Auth) GetByEmail() error {
	if err := database.Where("phone = ? and deleted = ?", a.Email, 0).Find(a).Error; err != nil {
		log.Errorf("get auth [%s] info failed : [%s]", a.Email, err)
		return fmt.Errorf("get auth [%s] info failed ", a.Email)
	}
	return nil
}

func (a *Auth) GetByPhone() error {
	if err := database.Where("phone = ? and deleted = ?", a.Phone, 0).Find(a).Error; err != nil {
		log.Errorf("get auth [%s] info failed : [%s]", a.Phone, err)
		return fmt.Errorf("get auth [%s] info failed ", a.Phone)
	}
	return nil
}

func (a *Auth) GetByUID() error {
	if err := database.Where("uid = ? and deleted = ?", a.UID, 0).Find(a).Error; err != nil {
		log.Errorf("get auth [%d] info failed : [%s]", a.UID, err)
		return fmt.Errorf("get auth [%d] info failed ", a.UID)
	}
	return nil
}

func (a *Auth) Delete() error {
	if err := database.Model(a).Update("deleted", 1); err != nil {
		log.Errorf("update auth [%d] deleted failed ", a.IDBase.ID)
		return fmt.Errorf("deleted auth [%d] failed ", a.IDBase.ID)
	}
	return nil
}
