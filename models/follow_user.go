package models

import "gorm.io/gorm"

type UserFollow struct {
	IDBase
	UserID   int64 `json:"user_id"`
	FollowID int64 `json:"follow_id"`
}

func (f *UserFollow) TableName() string {
	return "user_follow"
}

func (f *UserFollow) Create() error {
	return DataBase().Create(f).Error
}

func (f *UserFollow) GetByUserIDAndFollowID() error {
	err := DataBase().Where("user_id = ? AND follow_id = ?", f.UserID, f.FollowID).
		First(f).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil
		}
		return err
	}
	return nil
}

func (f *UserFollow) Delete() error {
	return DataBase().Delete(f).Error
}

func GetUserFollowers(offset int, limit int, userId int64) ([]*UserFollow, error) {
	var followers []*UserFollow
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

func GetUserFollowing(offset int, limit int, userId int64) ([]*UserFollow, error) {
	var following []*UserFollow
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
	err := DataBase().Model(&UserFollow{}).Where("follow_id = ?", userId).Count(&count).Error
	if err != nil {
		return 0, err
	}
	return count, nil
}

func GetUserFollowingCount(userId int64) (int64, error) {
	var count int64
	err := DataBase().Model(&UserFollow{}).Where("user_id = ?", userId).Count(&count).Error
	if err != nil {
		return 0, err
	}
	return count, nil
}

func FollowUser(userId int64, followId int64) error {
	followUser := &UserFollow{
		UserID:   userId,
		FollowID: followId,
	}
	return followUser.Create()
}

func UnfollowUser(userId int64, followId int64) error {
	followUser := &UserFollow{
		UserID:   userId,
		FollowID: followId,
	}
	return followUser.Delete()
}
