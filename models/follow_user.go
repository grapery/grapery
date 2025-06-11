package models

import (
	"context"

	"gorm.io/gorm"
)

// FollowUser 用户关注关系
type FollowUser struct {
	IDBase
	UserID     int64 `gorm:"column:user_id" json:"user_id,omitempty"`         // 用户ID
	FollowedID int64 `gorm:"column:followed_id" json:"followed_id,omitempty"` // 被关注用户ID
	Status     int   `gorm:"column:status" json:"status,omitempty"`           // 状态
}

func (f *FollowUser) TableName() string {
	return "user_follow"
}

func (f *FollowUser) Create() error {
	return DataBase().Create(f).Error
}

func (f *FollowUser) GetByUserIDAndFollowID() error {
	err := DataBase().Where("user_id = ? AND follow_id = ?", f.UserID, f.FollowedID).
		First(f).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil
		}
		return err
	}
	return nil
}

func (f *FollowUser) Delete() error {
	return DataBase().Delete(f).Error
}

func GetUserFollowers(offset int, limit int, userId int64) ([]*FollowUser, error) {
	var followers []*FollowUser
	err := DataBase().Where("follow_id = ?", userId).
		Offset(offset).Limit(limit).Find(&followers).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return followers, nil
}

func GetUserFollowing(offset int, limit int, userId int64) ([]*FollowUser, error) {
	var following []*FollowUser
	err := DataBase().Where("user_id = ?", userId).
		Offset(offset).Limit(limit).Find(&following).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return following, nil
}

func GetUserFollowersCount(userId int64) (int64, error) {
	var count int64
	err := DataBase().Model(&FollowUser{}).Where("follow_id = ?", userId).Count(&count).Error
	if err != nil {
		return 0, err
	}
	return count, nil
}

func GetUserFollowingCount(userId int64) (int64, error) {
	var count int64
	err := DataBase().Model(&FollowUser{}).Where("user_id = ?", userId).Count(&count).Error
	if err != nil {
		return 0, err
	}
	return count, nil
}

func NewFollowUser(userId int64, followId int64) error {
	followUser := &FollowUser{
		UserID:     userId,
		FollowedID: followId,
	}
	return followUser.Create()
}

func DeleteFollowUser(userId int64, followId int64) error {
	followUser := &FollowUser{
		UserID:     userId,
		FollowedID: followId,
	}
	return followUser.Delete()
}

// 新增：分页获取FollowUser列表
func GetFollowUserList(ctx context.Context, offset, limit int) ([]*FollowUser, error) {
	var follows []*FollowUser
	err := DataBase().Model(&FollowUser{}).
		WithContext(ctx).
		Offset(offset).
		Limit(limit).
		Order("create_at desc").
		Find(&follows).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return nil, err
	}
	return follows, nil
}

// 新增：通过用户ID和被关注ID唯一查询
func GetFollowUserByIDs(ctx context.Context, userID, followedID int64) (*FollowUser, error) {
	follow := &FollowUser{}
	err := DataBase().Model(follow).
		WithContext(ctx).
		Where("user_id = ? and followed_id = ?", userID, followedID).
		First(follow).Error
	if err != nil {
		return nil, err
	}
	return follow, nil
}
