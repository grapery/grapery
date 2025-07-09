package models

import (
	"time"

	"gorm.io/gorm"
)

type UserFollow struct {
	IDBase
	FollowerID int64     `gorm:"index:idx_follower_id;index:idx_follower_followee,priority:1"` // 关注者ID
	FolloweeID int64     `gorm:"index:idx_followee_id;index:idx_follower_followee,priority:2"` // 被关注者ID
	CreatedAt  time.Time // 关注时间
	Deleted    bool      // 软删除
}

// 增加粉丝数（事务）
func incUserProfileFollowersNumTx(tx *gorm.DB, userId int64) error {
	return tx.Model(&UserProfile{}).Where("user_id = ?", userId).
		UpdateColumn("followers_num", gorm.Expr("CASE WHEN followers_num IS NULL THEN 1 ELSE followers_num + 1 END")).Error
}

// 减少粉丝数（事务）
func decUserProfileFollowersNumTx(tx *gorm.DB, userId int64) error {
	return tx.Model(&UserProfile{}).Where("user_id = ? AND followers_num > 0", userId).
		UpdateColumn("followers_num", gorm.Expr("followers_num - 1")).Error
}

// 增加关注数（事务）
func incUserProfileFollowingNumTx(tx *gorm.DB, userId int64) error {
	return tx.Model(&UserProfile{}).Where("user_id = ?", userId).
		UpdateColumn("following_num", gorm.Expr("CASE WHEN following_num IS NULL THEN 1 ELSE following_num + 1 END")).Error
}

// 减少关注数（事务）
func decUserProfileFollowingNumTx(tx *gorm.DB, userId int64) error {
	return tx.Model(&UserProfile{}).Where("user_id = ? AND following_num > 0", userId).
		UpdateColumn("following_num", gorm.Expr("following_num - 1")).Error
}

// 创建关注关系（如已软删则恢复，否则插入新记录），强一致性事务
func CreateUserFollow(followerID, followeeID int64) error {
	return DataBase().Transaction(func(tx *gorm.DB) error {
		var uf UserFollow
		err := tx.Where("follower_id = ? AND followee_id = ?", followerID, followeeID).First(&uf).Error
		if err == nil {
			if uf.Deleted {
				// 恢复
				err1 := tx.Model(&uf).Updates(map[string]interface{}{"Deleted": false, "CreatedAt": time.Now()}).Error
				if err1 != nil {
					return err1
				}
				if err := incUserProfileFollowingNumTx(tx, followerID); err != nil {
					return err
				}
				if err := incUserProfileFollowersNumTx(tx, followeeID); err != nil {
					return err
				}
				return nil
			}
			return nil // 已存在有效关注
		}
		if err != gorm.ErrRecordNotFound {
			return err
		}
		uf = UserFollow{
			FollowerID: followerID,
			FolloweeID: followeeID,
			CreatedAt:  time.Now(),
			Deleted:    false,
		}
		err = tx.Create(&uf).Error
		if err != nil {
			return err
		}
		if err := incUserProfileFollowingNumTx(tx, followerID); err != nil {
			return err
		}
		if err := incUserProfileFollowersNumTx(tx, followeeID); err != nil {
			return err
		}
		return nil
	})
}

// 取消关注关系（软删除），强一致性事务
func DeleteUserFollow(followerID, followeeID int64) error {
	return DataBase().Transaction(func(tx *gorm.DB) error {
		err := tx.Model(&UserFollow{}).
			Where("follower_id = ? AND followee_id = ? AND deleted = ?", followerID, followeeID, false).
			Update("deleted", true).Error
		if err != nil {
			return err
		}
		if err := decUserProfileFollowingNumTx(tx, followerID); err != nil {
			return err
		}
		if err := decUserProfileFollowersNumTx(tx, followeeID); err != nil {
			return err
		}
		return nil
	})
}

// 判断是否已关注
func IsFollowing(followerID, followeeID int64) (bool, error) {
	var count int64
	err := DataBase().Model(&UserFollow{}).
		Where("follower_id = ? AND followee_id = ? AND deleted = ?", followerID, followeeID, false).
		Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// 获取我关注的用户列表，按时间倒序，分页
func GetFollowList(followerID int64, offset, limit int) ([]*User, error) {
	var follows []UserFollow
	err := DataBase().Where("follower_id = ? AND deleted = ?", followerID, false).
		Order("created_at DESC").Offset(offset).Limit(limit).Find(&follows).Error
	if err != nil {
		return nil, err
	}
	userIDs := make([]int64, 0, len(follows))
	for _, f := range follows {
		userIDs = append(userIDs, f.FolloweeID)
	}
	if len(userIDs) == 0 {
		return []*User{}, nil
	}
	var users []*User
	err = DataBase().Where("id IN ?", userIDs).Find(&users).Error
	return users, err
}

// 获取关注我的用户列表，按时间倒序，分页
func GetFollowerList(followeeID int64, offset, limit int) ([]*User, error) {
	var follows []UserFollow
	err := DataBase().Where("followee_id = ? AND deleted = ?", followeeID, false).
		Order("created_at DESC").Offset(offset).Limit(limit).Find(&follows).Error
	if err != nil {
		return nil, err
	}
	userIDs := make([]int64, 0, len(follows))
	for _, f := range follows {
		userIDs = append(userIDs, f.FollowerID)
	}
	if len(userIDs) == 0 {
		return []*User{}, nil
	}
	var users []*User
	err = DataBase().Where("id IN ?", userIDs).Find(&users).Error
	return users, err
}
