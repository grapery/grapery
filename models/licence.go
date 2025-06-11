package models

import (
	"context"

	"gorm.io/gorm"
)

// Licence 版权/授权信息
type Licence struct {
	IDBase
	Name        string `gorm:"column:name" json:"name,omitempty"`               // 名称
	Description string `gorm:"column:description" json:"description,omitempty"` // 描述
	Content     string `gorm:"column:content" json:"content,omitempty"`         // 内容
	Ref         int64  `gorm:"column:ref" json:"ref,omitempty"`                 // 关联ID
	Avatar      string `gorm:"column:avatar" json:"avatar,omitempty"`           // 头像
	Creator     int64  `gorm:"column:creator" json:"creator,omitempty"`         // 创建者ID
	Status      int    `gorm:"column:status" json:"status,omitempty"`           // 状态
	Version     int    `gorm:"column:version" json:"version,omitempty"`         // 版本
	Apply       int    `gorm:"column:apply" json:"apply,omitempty"`             // 是否应用
}

func (licence Licence) TableName() string {
	return "licence"
}

func CreateLicense(licence *Licence) error {
	err := DataBase().Create(licence).Error
	if err != nil {
		return err
	}
	return nil
}

func GetLicenseById(id int64) (*Licence, error) {
	var licence Licence
	err := DataBase().Where("id = ?", id).First(&licence).Error
	if err != nil {
		return nil, err
	}
	return &licence, nil
}

func GetLicenseByName(name string) (*Licence, error) {
	var licence Licence
	err := DataBase().Where("name = ?", name).First(&licence).Error
	if err != nil {
		return nil, err
	}
	return &licence, nil
}

func GetLicenseByRef(ref int64) (*Licence, error) {
	var licence Licence
	err := DataBase().Where("ref = ?", ref).First(&licence).Error
	if err != nil {
		return nil, err
	}
	return &licence, nil
}

func GetLicenseByCreator(creator int64) ([]*Licence, error) {
	var licences []*Licence
	err := DataBase().Where("creator = ?", creator).Find(&licences).Error
	if err != nil {
		return nil, err
	}
	return licences, nil
}

func UpdateLicense(licence *Licence) error {
	err := DataBase().Save(licence).Error
	if err != nil {
		return err
	}
	return nil
}

func DeleteLicense(licence *Licence) error {
	err := DataBase().Delete(licence).Error
	if err != nil {
		return err
	}
	return nil
}

// 新增：分页获取Licence列表
func GetLicenceList(ctx context.Context, offset, limit int) ([]*Licence, error) {
	var licences []*Licence
	err := DataBase().Model(&Licence{}).
		WithContext(ctx).
		Offset(offset).
		Limit(limit).
		Order("create_at desc").
		Find(&licences).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return nil, err
	}
	return licences, nil
}

// 新增：通过Name唯一查询
func GetLicenceByNameUnique(ctx context.Context, name string) (*Licence, error) {
	lic := &Licence{}
	err := DataBase().Model(lic).
		WithContext(ctx).
		Where("name = ?", name).
		First(lic).Error
	if err != nil {
		return nil, err
	}
	return lic, nil
}
