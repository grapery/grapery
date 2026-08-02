package models

import (
	"context"
)

// 分享故事/角色/故事板，分享时用户会有自己的文字输入内容

type UserShare struct {
	ID uint `gorm:"primary_key,column:id" json:"id,omitempty"`
	IDBase
	UserID       int64  `gorm:"column:user_id" json:"user_id,omitempty"`             // 用户ID
	StoryID      int64  `gorm:"column:story_id" json:"story_id,omitempty"`           // 故事ID
	StoryboardId int64  `gorm:"column:storyboard_id" json:"storyboard_id,omitempty"` // 故事板ID
	RoleID       int64  `gorm:"column:role_id" json:"role_id,omitempty"`             // 角色ID
	Content      string `gorm:"column:content" json:"content,omitempty"`             // 分享内容
	IsValied     bool   `gorm:"column:is_valied" json:"is_valied,omitempty"`         // 分享内容是否有效
	IsExpired    bool   `gorm:"column:is_expired" json:"is_expired,omitempty"`       // 分享内容是否过期
	ExpiredTime  int64  `gorm:"column:expired_time" json:"expired_time,omitempty"`   // 分享内容过期时间
}

func (u *UserShare) TableName() string {
	return "user_share"
}

func CreateNewShare(ctx context.Context, share *UserShare) error {
	return DataBase().WithContext(ctx).Model(&UserShare{}).Create(share).Error
}

func GetShareByID(ctx context.Context, id uint) (*UserShare, error) {
	var share UserShare
	err := DataBase().
		WithContext(ctx).
		Model(&UserShare{}).
		Where("id = ?", id).
		First(&share).Error
	if err != nil {
		return nil, err
	}
	return &share, nil
}

func GetShareByUserID(ctx context.Context, userID int64, offset int64, pageSize int64) ([]*UserShare, error) {
	var shares []*UserShare
	err := DataBase().
		WithContext(ctx).
		Model(&UserShare{}).
		Where("user_id = ?", userID).
		Offset(int((offset - 1) * pageSize)).
		Limit(int(pageSize)).
		Find(&shares).Error
	if err != nil {
		return nil, err
	}
	return shares, nil
}

func BatchSetShareExpired(ctx context.Context, ids []uint) error {
	return DataBase().
		WithContext(ctx).
		Model(&UserShare{}).
		Where("id IN ?", ids).
		Update("is_expired", true).
		Error
}
