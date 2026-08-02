package models

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type UserFollow struct {
	ID         uint      `gorm:"primaryKey;column:id" json:"id,omitempty"`
	FollowerID int64     `gorm:"column:follower_id;not null;index:idx_follower_id;uniqueIndex:uniq_user_follow,priority:1"`
	FolloweeID int64     `gorm:"column:followee_id;not null;index:idx_followee_id;uniqueIndex:uniq_user_follow,priority:2"`
	CreatedAt  time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at,omitempty"`
	UpdatedAt  time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at,omitempty"`
	Deleted    bool      `gorm:"column:deleted;not null;default:false" json:"deleted,omitempty"`
}

func (UserFollow) TableName() string {
	return "user_follow"
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
func CreateUserFollow(ctx context.Context, followerID, followeeID int64) (bool, error) {
	var created bool
	err := DataBase().WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var uf UserFollow
		err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("follower_id = ? AND followee_id = ?", followerID, followeeID).
			First(&uf).Error
		switch {
		case err == nil:
			if uf.Deleted {
				now := time.Now()
				if err := tx.Model(&UserFollow{}).
					Where("id = ?", uf.ID).
					Updates(map[string]interface{}{
						"deleted":    false,
						"created_at": now,
						"updated_at": now,
					}).Error; err != nil {
					return err
				}
				created = true
			}
		case errors.Is(err, gorm.ErrRecordNotFound):
			uf = UserFollow{
				FollowerID: followerID,
				FolloweeID: followeeID,
			}
			if err := tx.Create(&uf).Error; err != nil {
				return err
			}
			created = true
		default:
			return err
		}

		if !created {
			return nil
		}

		if err := incUserProfileFollowingNumTx(tx, followerID); err != nil {
			return err
		}
		if err := incUserProfileFollowersNumTx(tx, followeeID); err != nil {
			return err
		}
		return nil
	})
	return created, err
}

// 取消关注关系（软删除），强一致性事务
func DeleteUserFollow(ctx context.Context, followerID, followeeID int64) (bool, error) {
	var removed bool
	err := DataBase().WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var uf UserFollow
		err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("follower_id = ? AND followee_id = ?", followerID, followeeID).
			First(&uf).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		if err != nil {
			return err
		}
		if uf.Deleted {
			return nil
		}
		now := time.Now()
		if err := tx.Model(&UserFollow{}).
			Where("id = ?", uf.ID).
			Updates(map[string]interface{}{
				"deleted":    true,
				"updated_at": now,
			}).Error; err != nil {
			return err
		}
		if err := decUserProfileFollowingNumTx(tx, followerID); err != nil {
			return err
		}
		if err := decUserProfileFollowersNumTx(tx, followeeID); err != nil {
			return err
		}
		removed = true
		return nil
	})
	return removed, err
}

// 判断是否已关注
func IsFollowing(ctx context.Context, followerID, followeeID int64) (bool, error) {
	var count int64
	err := DataBase().WithContext(ctx).Model(&UserFollow{}).
		Where("follower_id = ? AND followee_id = ? AND deleted = ?", followerID, followeeID, false).
		Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// 获取我关注的用户列表，按时间倒序，分页
func GetFollowList(followerID int64, offset, limit int) ([]*User, int64, error) {
	var follows []UserFollow
	var total int64
	err := DataBase().Where("follower_id = ? AND deleted = ?", followerID, false).
		Count(&total).Error
	if err != nil {
		return nil, 0, err
	}
	err = DataBase().Where("follower_id = ? AND deleted = ?", followerID, false).
		Order("created_at DESC").
		Offset((offset - 1) * limit).
		Limit(limit).
		Find(&follows).Error
	if err != nil {
		return nil, 0, err
	}
	userIDs := make([]int64, 0, len(follows))
	for _, f := range follows {
		userIDs = append(userIDs, f.FolloweeID)
	}
	if len(userIDs) == 0 {
		return []*User{}, 0, nil
	}
	var users []*User
	err = DataBase().Where("id IN ?", userIDs).Find(&users).Error
	return users, total, err
}

// 获取关注我的用户列表，按时间倒序，分页
func GetFollowerList(followeeID int64, offset, limit int) ([]*User, int64, error) {
	var follows []UserFollow
	var total int64
	err := DataBase().Where("followee_id = ? AND deleted = ?", followeeID, false).
		Count(&total).Error
	if err != nil {
		return nil, 0, err
	}
	err = DataBase().Where("followee_id = ? AND deleted = ?", followeeID, false).
		Order("created_at DESC").Offset((offset - 1) * limit).Limit(limit).Find(&follows).Error
	if err != nil {
		return nil, 0, err
	}
	userIDs := make([]int64, 0, len(follows))
	for _, f := range follows {
		userIDs = append(userIDs, f.FollowerID)
	}
	if len(userIDs) == 0 {
		return []*User{}, 0, nil
	}
	var users []*User
	err = DataBase().Where("id IN ?", userIDs).Find(&users).Error
	return users, total, err
}
