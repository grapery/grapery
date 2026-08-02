package models

import (
	"context"
	"errors"

	log2 "github.com/grapery/grapery/utils/log"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type LikeEventType int64

const (
	LikeEventTypeUnknown LikeEventType = iota
	LikeEventTypeLike
	LikeEventTypeDislike
)

type LikeItemType int64

const (
	LikeItemTypeUnknown LikeItemType = iota
	LikeItemTypeGroup
	LikeItemTypeTimeline
	LikeItemTypeStory
	LikeItemTypeStoryboard
	LikeItemTypeRole
	LikeItemTypeComment
	LikeItemTypeShare
)

// LikeItem 点赞/点踩/收藏等行为记录
type LikeItem struct {
	ID uint `gorm:"primary_key,column:id" json:"id,omitempty"`
	IDBase
	UserID       int64         `gorm:"column:user_id;index;uniqueIndex:uniq_like_item,priority:1" json:"user_id,omitempty"`         // 用户ID
	GroupID      int64         `gorm:"column:group_id;uniqueIndex:uniq_like_item,priority:4" json:"group_id,omitempty"`             // 群组ID
	TimelineID   int64         `gorm:"column:timeline_id;uniqueIndex:uniq_like_item,priority:5" json:"timeline_id,omitempty"`       // 时间线ID
	StoryID      int64         `gorm:"column:story_id;uniqueIndex:uniq_like_item,priority:6" json:"story_id,omitempty"`             // 故事ID
	StoryboardId int64         `gorm:"column:storyboard_id;uniqueIndex:uniq_like_item,priority:7" json:"storyboard_id,omitempty"`   // 故事板ID
	RoleID       int64         `gorm:"column:role_id;uniqueIndex:uniq_like_item,priority:8" json:"role_id,omitempty"`               // 角色ID
	LikeType     LikeEventType `gorm:"column:like_type" json:"like_type,omitempty"`                                                 // 点赞类型
	LikeItemType LikeItemType  `gorm:"column:like_item_type;uniqueIndex:uniq_like_item,priority:2" json:"like_item_type,omitempty"` // 点赞对象类型
}

func (l LikeItem) TableName() string {
	return "like_item"
}

func CreateLikeStoryItem(ctx context.Context, item *LikeItem) error {
	_, err := LikeStory(ctx, item.UserID, item.StoryID)
	return err
}

func LikeStory(ctx context.Context, userId int64, storyId int64) (bool, error) {
	log2.Log().Info("[LikeStory] 尝试点赞故事", zap.Int64("userId", userId), zap.Int64("storyId", storyId))
	var created bool
	err := DataBase().WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var item LikeItem
		err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("user_id = ? AND story_id = ? AND like_item_type = ?", userId, storyId, LikeItemTypeStory).
			First(&item).Error
		switch {
		case err == nil:
			if !item.Deleted && item.LikeType == LikeEventTypeLike {
				log2.Log().Info("[LikeStory] 已点赞，无需重复", zap.Int64("userId", userId), zap.Int64("storyId", storyId))
				return nil
			}
			updates := map[string]interface{}{
				"deleted":   false,
				"like_type": LikeEventTypeLike,
				"story_id":  storyId,
			}
			if err := tx.Model(&LikeItem{}).
				Where("id = ?", item.ID).
				Updates(updates).Error; err != nil {
				return err
			}
			created = true
		case errors.Is(err, gorm.ErrRecordNotFound):
			newItem := &LikeItem{
				UserID:       userId,
				StoryID:      storyId,
				LikeType:     LikeEventTypeLike,
				LikeItemType: LikeItemTypeStory,
			}
			if err := tx.Create(newItem).Error; err != nil {
				return err
			}
			created = true
		default:
			return err
		}

		if !created {
			return nil
		}

		return tx.Model(&Story{}).
			Where("id = ?", storyId).
			Update("like_count", gorm.Expr("CASE WHEN like_count IS NULL THEN 1 ELSE like_count + 1 END")).Error
	})
	if err != nil {
		log2.Log().Error("[LikeStory] 点赞故事失败", zap.Error(err), zap.Int64("userId", userId), zap.Int64("storyId", storyId))
		return false, err
	}
	if created {
		log2.Log().Info("[LikeStory] 点赞故事成功", zap.Int64("userId", userId), zap.Int64("storyId", storyId))
	}
	return created, nil
}

func UnLikeStory(ctx context.Context, userId int64, storyId int64) (bool, error) {
	log2.Log().Info("[UnLikeStory] 尝试取消点赞故事", zap.Int64("userId", userId), zap.Int64("storyId", storyId))
	var removed bool
	err := DataBase().WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var item LikeItem
		err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("user_id = ? AND story_id = ? AND like_item_type = ?", userId, storyId, LikeItemTypeStory).
			First(&item).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			log2.Log().Info("[UnLikeStory] 未找到点赞记录", zap.Int64("userId", userId), zap.Int64("storyId", storyId))
			return nil
		}
		if err != nil {
			return err
		}
		if item.Deleted {
			log2.Log().Info("[UnLikeStory] 点赞记录已删除", zap.Int64("userId", userId), zap.Int64("storyId", storyId))
			return nil
		}

		if err := tx.Model(&LikeItem{}).
			Where("id = ?", item.ID).
			Updates(map[string]interface{}{"deleted": true}).Error; err != nil {
			return err
		}

		if err := tx.Model(&Story{}).
			Where("id = ?", storyId).
			Update("like_count", gorm.Expr("CASE WHEN like_count > 0 THEN like_count - 1 ELSE 0 END")).Error; err != nil {
			return err
		}

		removed = true
		return nil
	})
	if err != nil {
		log2.Log().Error("[UnLikeStory] 取消点赞故事失败", zap.Error(err), zap.Int64("userId", userId), zap.Int64("storyId", storyId))
		return false, err
	}
	if removed {
		log2.Log().Info("[UnLikeStory] 取消点赞故事成功", zap.Int64("userId", userId), zap.Int64("storyId", storyId))
	}
	return removed, nil
}

func CreateLikeStoryTimelineItem(ctx context.Context, item *LikeItem) error {
	var num int64
	// 日志：检查是否已存在
	log2.Log().Info("[CreateLikeStoryTimelineItem] 检查是否已存在", zap.Int64("userID", item.UserID), zap.Int64("storyID", item.StoryID), zap.Int64("timelineID", item.TimelineID))
	err := DataBase().WithContext(ctx).Model(&LikeItem{}).
		Where("user_id = ? and story_id = ? and timeline_id = ?",
			item.UserID, item.StoryID, item.TimelineID).
		Count(&num).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		// 日志：数据库查询失败
		log2.Log().Error("[CreateLikeStoryTimelineItem] 查询失败", zap.Error(err), zap.Any("item", item))
		return err
	}
	if err == gorm.ErrRecordNotFound {
		// 日志：未找到记录
		log2.Log().Warn("[CreateLikeStoryTimelineItem] 未找到记录", zap.Int64("userID", item.UserID), zap.Int64("storyID", item.StoryID), zap.Int64("timelineID", item.TimelineID))
		return nil
	}
	if num > 0 {
		// 日志：已存在
		log2.Log().Info("[CreateLikeStoryTimelineItem] 已存在", zap.Int64("userID", item.UserID), zap.Int64("storyID", item.StoryID), zap.Int64("timelineID", item.TimelineID))
		return nil
	}
	// 日志：准备创建点赞记录
	log2.Log().Info("[CreateLikeStoryTimelineItem] 创建点赞记录", zap.Any("item", item))
	err = DataBase().WithContext(ctx).Model(&LikeItem{}).Create(item).Error
	if err != nil {
		// 日志：创建失败
		log2.Log().Error("[CreateLikeStoryTimelineItem] 创建失败", zap.Error(err), zap.Any("item", item))
		return err
	}
	// 日志：创建成功
	log2.Log().Info("[CreateLikeStoryTimelineItem] 创建成功", zap.Int64("id", int64(item.ID)))
	return nil
}

func CreateLikeStoryBoardItem(ctx context.Context, item *LikeItem) error {
	_, err := LikeStoryboard(ctx, item.UserID, item.StoryID, item.StoryboardId, item.GroupID)
	return err
}

func LikeStoryboard(ctx context.Context, userId int64, storyId int64, storyboardId int64, groupId int64) (bool, error) {
	log2.Log().Info("[LikeStoryboard] 尝试点赞故事板", zap.Int64("userId", userId), zap.Int64("storyId", storyId), zap.Int64("storyboardId", storyboardId))
	var created bool
	err := DataBase().WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var item LikeItem
		err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("user_id = ? AND storyboard_id = ? AND like_item_type = ?", userId, storyboardId, LikeItemTypeStoryboard).
			First(&item).Error
		switch {
		case err == nil:
			if !item.Deleted && item.LikeType == LikeEventTypeLike {
				log2.Log().Info("[LikeStoryboard] 已点赞，无需重复", zap.Int64("userId", userId), zap.Int64("storyboardId", storyboardId))
				return nil
			}
			updates := map[string]interface{}{
				"deleted":       false,
				"like_type":     LikeEventTypeLike,
				"group_id":      groupId,
				"story_id":      storyId,
				"storyboard_id": storyboardId,
			}
			if err := tx.Model(&LikeItem{}).
				Where("id = ?", item.ID).
				Updates(updates).Error; err != nil {
				return err
			}
			created = true
		case errors.Is(err, gorm.ErrRecordNotFound):
			newItem := &LikeItem{
				UserID:       userId,
				GroupID:      groupId,
				StoryID:      storyId,
				StoryboardId: storyboardId,
				LikeType:     LikeEventTypeLike,
				LikeItemType: LikeItemTypeStoryboard,
			}
			if err := tx.Create(newItem).Error; err != nil {
				return err
			}
			created = true
		default:
			return err
		}

		if !created {
			return nil
		}

		return tx.Model(&StoryBoard{}).
			Where("id = ?", storyboardId).
			Update("like_num", gorm.Expr("CASE WHEN like_num IS NULL THEN 1 ELSE like_num + 1 END")).Error
	})
	if err != nil {
		log2.Log().Error("[LikeStoryboard] 点赞故事板失败", zap.Error(err), zap.Int64("userId", userId), zap.Int64("storyId", storyId), zap.Int64("storyboardId", storyboardId))
		return false, err
	}
	if created {
		log2.Log().Info("[LikeStoryboard] 点赞故事板成功", zap.Int64("userId", userId), zap.Int64("storyId", storyId), zap.Int64("storyboardId", storyboardId))
	}
	return created, nil
}

func UnLikeStoryboard(ctx context.Context, userId int64, storyboardId int64) (bool, error) {
	log2.Log().Info("[UnLikeStoryboard] 尝试取消点赞故事板", zap.Int64("userId", userId), zap.Int64("storyboardId", storyboardId))
	var removed bool
	err := DataBase().WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var item LikeItem
		err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("user_id = ? AND storyboard_id = ? AND like_item_type = ?", userId, storyboardId, LikeItemTypeStoryboard).
			First(&item).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			log2.Log().Info("[UnLikeStoryboard] 未找到点赞记录", zap.Int64("userId", userId), zap.Int64("storyboardId", storyboardId))
			return nil
		}
		if err != nil {
			return err
		}
		if item.Deleted {
			log2.Log().Info("[UnLikeStoryboard] 点赞记录已删除", zap.Int64("userId", userId), zap.Int64("storyboardId", storyboardId))
			return nil
		}

		if err := tx.Model(&LikeItem{}).
			Where("id = ?", item.ID).
			Updates(map[string]interface{}{"deleted": true}).Error; err != nil {
			return err
		}

		if err := tx.Model(&StoryBoard{}).
			Where("id = ?", storyboardId).
			Update("like_num", gorm.Expr("CASE WHEN like_num > 0 THEN like_num - 1 ELSE 0 END")).Error; err != nil {
			return err
		}

		removed = true
		return nil
	})
	if err != nil {
		log2.Log().Error("[UnLikeStoryboard] 取消点赞故事板失败", zap.Error(err), zap.Int64("userId", userId), zap.Int64("storyboardId", storyboardId))
		return false, err
	}
	if removed {
		log2.Log().Info("[UnLikeStoryboard] 取消点赞故事板成功", zap.Int64("userId", userId), zap.Int64("storyboardId", storyboardId))
	}
	return removed, nil
}

func DeleteLikeItem(ctx context.Context, itemID int64) error {
	// 日志：准备删除LikeItem
	log2.Log().Info("[DeleteLikeItem] 尝试删除LikeItem", zap.Int64("itemID", itemID))
	err := DataBase().WithContext(ctx).Model(&LikeItem{}).
		Update("deleted", 1).
		Where("id = ?", itemID).Error
	if err != nil {
		// 日志：删除失败
		log2.Log().Error("[DeleteLikeItem] 删除失败", zap.Error(err), zap.Int64("itemID", itemID))
		return err
	}
	// 日志：删除成功
	log2.Log().Info("[DeleteLikeItem] 删除成功", zap.Int64("itemID", itemID))
	return nil
}

func GetLikeItem(ctx context.Context, itemID int64) (*LikeItem, error) {
	item := new(LikeItem)
	// 日志：准备查询LikeItem
	log2.Log().Info("[GetLikeItem] 查询LikeItem", zap.Int64("itemID", itemID))
	err := DataBase().WithContext(ctx).
		Model(&LikeItem{}).
		First(item).
		Where("id = ?", itemID).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		// 日志：查询失败
		log2.Log().Error("[GetLikeItem] 查询失败", zap.Error(err), zap.Int64("itemID", itemID))
		return nil, err
	}
	if err == gorm.ErrRecordNotFound {
		// 日志：未找到记录
		log2.Log().Warn("[GetLikeItem] 未找到记录", zap.Int64("itemID", itemID))
		return nil, nil
	}
	// 日志：查询成功
	log2.Log().Info("[GetLikeItem] 查询成功", zap.Any("item", item))
	return item, nil
}

func GetLikeItemByUser(ctx context.Context, userID int) (list []*LikeItem, err error) {
	list = make([]*LikeItem, 0)
	// 日志：准备查询用户点赞记录
	log2.Log().Info("[GetLikeItemByUser] 查询用户点赞记录", zap.Int("userID", userID))
	err = DataBase().WithContext(ctx).Model(&LikeItem{}).
		Where("user_id = ?", userID).
		Scan(&list).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		// 日志：查询失败
		log2.Log().Error("[GetLikeItemByUser] 查询失败", zap.Error(err), zap.Int("userID", userID))
		return nil, err
	}
	if err == gorm.ErrRecordNotFound {
		// 日志：未找到记录
		log2.Log().Warn("[GetLikeItemByUser] 未找到记录", zap.Int("userID", userID))
		return nil, nil
	}
	// 日志：查询成功
	log2.Log().Info("[GetLikeItemByUser] 查询成功", zap.Int("count", len(list)))
	return list, nil
}

func GetLikeItemByStory(ctx context.Context, storyId int64) (list []*LikeItem, err error) {
	list = make([]*LikeItem, 0)
	// 日志：准备查询故事点赞记录
	log2.Log().Info("[GetLikeItemByStory] 查询故事点赞记录", zap.Int64("storyId", storyId))
	err = DataBase().WithContext(ctx).Model(&LikeItem{}).
		Where("story_id = ?", storyId).
		Scan(&list).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		// 日志：查询失败
		log2.Log().Error("[GetLikeItemByStory] 查询失败", zap.Error(err), zap.Int64("storyId", storyId))
		return nil, err
	}
	if err == gorm.ErrRecordNotFound {
		// 日志：未找到记录
		log2.Log().Warn("[GetLikeItemByStory] 未找到记录", zap.Int64("storyId", storyId))
		return nil, nil
	}
	// 日志：查询成功
	log2.Log().Info("[GetLikeItemByStory] 查询成功", zap.Int("count", len(list)))
	return list, nil
}

func GetLikeItemByStoryAndUser(ctx context.Context, storyId int64, userId int) (*LikeItem, error) {
	item := new(LikeItem)
	// 日志：准备查询用户对故事的点赞记录
	log2.Log().Info("[GetLikeItemByStoryAndUser] 查询用户对故事的点赞记录", zap.Int64("storyId", storyId), zap.Int("userId", userId))
	err := DataBase().WithContext(ctx).Model(&LikeItem{}).
		Where("story_id = ?", storyId).
		Where("user_id = ?", userId).
		Where("deleted = ?", 0).
		Where("like_item_type = ?", LikeItemTypeStory).
		First(item).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		// 日志：查询失败
		log2.Log().Error("[GetLikeItemByStoryAndUser] 查询失败", zap.Error(err), zap.Int64("storyId", storyId), zap.Int("userId", userId))
		return nil, err
	}
	if err == gorm.ErrRecordNotFound {
		// 日志：未找到记录
		log2.Log().Warn("[GetLikeItemByStoryAndUser] 未找到记录", zap.Int64("storyId", storyId), zap.Int("userId", userId))
		return nil, nil
	}
	// 日志：查询成功
	log2.Log().Info("[GetLikeItemByStoryAndUser] 查询成功", zap.Any("item", item))
	return item, nil
}

// 根据一组故事id，以及一个用户id来获取喜欢的列表
func GetLikeItemByStoriesAndUser(ctx context.Context, storyIds []int64, userId int) (list []*LikeItem, err error) {
	list = make([]*LikeItem, 0)
	// 日志：准备批量查询用户对多故事的点赞记录
	log2.Log().Info("[GetLikeItemByStoriesAndUser] 批量查询用户对故事的点赞记录", zap.Any("storyIds", storyIds), zap.Int("userId", userId))
	err = DataBase().WithContext(ctx).Model(&LikeItem{}).
		Where("story_id in (?) and user_id = ?", storyIds, userId).
		Where("deleted = ?", 0).
		Where("like_item_type = ?", LikeItemTypeStory).
		Scan(&list).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		// 日志：查询失败
		log2.Log().Error("[GetLikeItemByStoriesAndUser] 查询失败", zap.Error(err), zap.Any("storyIds", storyIds), zap.Int("userId", userId))
		return nil, err
	}
	if err == gorm.ErrRecordNotFound {
		// 日志：未找到记录
		log2.Log().Warn("[GetLikeItemByStoriesAndUser] 未找到记录", zap.Any("storyIds", storyIds), zap.Int("userId", userId))
		return nil, nil
	}
	// 日志：查询成功
	log2.Log().Info("[GetLikeItemByStoriesAndUser] 查询成功", zap.Int("count", len(list)))
	return list, nil
}

func GetLikeItemByGroup(ctx context.Context, groupId []int64, userId int) (list []*LikeItem, err error) {
	list = make([]*LikeItem, 0)
	// 日志：准备批量查询用户对小组的点赞记录
	log2.Log().Info("[GetLikeItemByGroup] 批量查询用户对小组的点赞记录", zap.Any("groupId", groupId), zap.Int("userId", userId))
	err = DataBase().WithContext(ctx).Model(&LikeItem{}).
		Where("group_id in (?) and user_id = ?", groupId, userId).
		Where("deleted = ?", 0).
		Where("like_item_type = ?", LikeItemTypeGroup).
		Scan(&list).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		// 日志：查询失败
		log2.Log().Error("[GetLikeItemByGroup] 查询失败", zap.Error(err), zap.Any("groupId", groupId), zap.Int("userId", userId))
		return nil, err
	}
	if err == gorm.ErrRecordNotFound {
		// 日志：未找到记录
		log2.Log().Warn("[GetLikeItemByGroup] 未找到记录", zap.Any("groupId", groupId), zap.Int("userId", userId))
		return nil, nil
	}
	// 日志：查询成功
	log2.Log().Info("[GetLikeItemByGroup] 查询成功", zap.Int("count", len(list)))
	return list, nil
}

// 根据一组角色id，以及一个用户id来获取喜欢的列表
func GetLikeItemByRolesAndUser(ctx context.Context, roleIds []int64, userId int) (list []*LikeItem, err error) {
	list = make([]*LikeItem, 0)
	// 日志：准备批量查询用户对角色的点赞记录
	log2.Log().Info("[GetLikeItemByRolesAndUser] 批量查询用户对角色的点赞记录", zap.Any("roleIds", roleIds), zap.Int("userId", userId))
	err = DataBase().WithContext(ctx).Model(&LikeItem{}).
		Where("role_id in (?) and user_id = ?", roleIds, userId).
		Where("deleted = ?", 0).
		Where("like_item_type = ?", LikeItemTypeRole).
		Scan(&list).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		// 日志：查询失败
		log2.Log().Error("[GetLikeItemByRolesAndUser] 查询失败", zap.Error(err), zap.Any("roleIds", roleIds), zap.Int("userId", userId))
		return nil, err
	}
	if err == gorm.ErrRecordNotFound {
		// 日志：未找到记录
		log2.Log().Warn("[GetLikeItemByRolesAndUser] 未找到记录", zap.Any("roleIds", roleIds), zap.Int("userId", userId))
		return nil, nil
	}
	// 日志：查询成功
	log2.Log().Info("[GetLikeItemByRolesAndUser] 查询成功", zap.Int("count", len(list)))
	return list, nil
}

// // 根据一组故事板id，以及一个用户id来获取喜欢的列表
func GetLikeItemByStoryBoardsAndUser(ctx context.Context, storyboardIds []int64, userId int) (list []*LikeItem, err error) {
	list = make([]*LikeItem, 0)
	// 日志：准备批量查询用户对故事板的点赞记录
	log2.Log().Info("[GetLikeItemByStoryBoardsAndUser] 批量查询用户对故事板的点赞记录", zap.Any("storyboardIds", storyboardIds), zap.Int("userId", userId))
	err = DataBase().WithContext(ctx).Model(&LikeItem{}).
		Where("storyboard_id in (?) and user_id = ?", storyboardIds, userId).
		Where("deleted = ?", 0).
		Where("like_item_type = ?", LikeItemTypeStoryboard).
		Scan(&list).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		// 日志：查询失败
		log2.Log().Error("[GetLikeItemByStoryBoardsAndUser] 查询失败", zap.Error(err), zap.Any("storyboardIds", storyboardIds), zap.Int("userId", userId))
		return nil, err
	}
	if err == gorm.ErrRecordNotFound {
		// 日志：未找到记录
		log2.Log().Warn("[GetLikeItemByStoryBoardsAndUser] 未找到记录", zap.Any("storyboardIds", storyboardIds), zap.Int("userId", userId))
		return nil, nil
	}
	// 日志：查询成功
	log2.Log().Info("[GetLikeItemByStoryBoardsAndUser] 查询成功", zap.Int("count", len(list)))
	return list, nil
}

func GetLikeItemByStoryBoard(ctx context.Context, storyId int64, storyboardId int64) (list []*LikeItem, err error) {
	list = make([]*LikeItem, 0)
	// 日志：准备查询故事板点赞记录
	log2.Log().Info("[GetLikeItemByStoryBoard] 查询故事板点赞记录", zap.Int64("storyId", storyId), zap.Int64("storyboardId", storyboardId))
	err = DataBase().WithContext(ctx).Model(&LikeItem{}).
		Where("story_id = ? and storyboard_id = ? and like_item_type = ? and deleted = ?", storyId, storyboardId, LikeItemTypeStoryboard, false).
		Scan(&list).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		// 日志：查询失败
		log2.Log().Error("[GetLikeItemByStoryBoard] 查询失败", zap.Error(err), zap.Int64("storyId", storyId), zap.Int64("storyboardId", storyboardId))
		return nil, err
	}
	if err == gorm.ErrRecordNotFound {
		// 日志：未找到记录
		log2.Log().Warn("[GetLikeItemByStoryBoard] 未找到记录", zap.Int64("storyId", storyId), zap.Int64("storyboardId", storyboardId))
		return nil, nil
	}
	// 日志：查询成功
	log2.Log().Info("[GetLikeItemByStoryBoard] 查询成功", zap.Int("count", len(list)))
	return list, nil
}

func GetLikeItemByStoryBoardAndUser(ctx context.Context, storyboardId int64, userId int) (*LikeItem, error) {
	item := new(LikeItem)
	// 日志：准备查询用户对故事板的点赞记录
	log2.Log().Info("[GetLikeItemByStoryBoardAndUser] 查询用户对故事板的点赞记录", zap.Int64("storyboardId", storyboardId), zap.Int("userId", userId))
	err := DataBase().WithContext(ctx).Model(&LikeItem{}).
		Where("storyboard_id = ? and user_id = ? and like_item_type = ? and deleted = ?", storyboardId, userId, LikeItemTypeStoryboard, false).
		First(item).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		// 日志：查询失败
		log2.Log().Error("[GetLikeItemByStoryBoardAndUser] 查询失败", zap.Error(err), zap.Int64("storyboardId", storyboardId), zap.Int("userId", userId))
		return nil, err
	}
	if err == gorm.ErrRecordNotFound {
		// 日志：未找到记录
		log2.Log().Warn("[GetLikeItemByStoryBoardAndUser] 未找到记录", zap.Int64("storyboardId", storyboardId), zap.Int("userId", userId))
		return nil, nil
	}
	// 日志：查询成功
	log2.Log().Info("[GetLikeItemByStoryBoardAndUser] 查询成功", zap.Any("item", item))
	return item, nil
}

func GetLikeItemByStoryRoleAndUser(ctx context.Context, roleId int64, userId int) (*LikeItem, error) {
	item := new(LikeItem)
	// 日志：准备查询用户对角色的点赞记录
	log2.Log().Info("[GetLikeItemByStoryRoleAndUser] 查询用户对角色的点赞记录", zap.Int64("roleId", roleId), zap.Int("userId", userId))
	err := DataBase().WithContext(ctx).Model(&LikeItem{}).
		Where("role_id = ? and user_id = ?", roleId, userId).
		First(item).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		// 日志：查询失败
		log2.Log().Error("[GetLikeItemByStoryRoleAndUser] 查询失败", zap.Error(err), zap.Int64("roleId", roleId), zap.Int("userId", userId))
		return nil, err
	}
	if err == gorm.ErrRecordNotFound {
		// 日志：未找到记录
		log2.Log().Warn("[GetLikeItemByStoryRoleAndUser] 未找到记录", zap.Int64("roleId", roleId), zap.Int("userId", userId))
		return nil, nil
	}
	// 日志：查询成功
	log2.Log().Info("[GetLikeItemByStoryRoleAndUser] 查询成功", zap.Any("item", item))
	return item, nil
}

func LikeStoryRole(ctx context.Context, userId int64, storyId int64, roleId int64) (bool, error) {
	log2.Log().Info("[LikeStoryRole] 尝试点赞角色", zap.Int64("userId", userId), zap.Int64("storyId", storyId), zap.Int64("roleId", roleId))
	var created bool
	err := DataBase().WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var item LikeItem
		err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("user_id = ? AND role_id = ? AND like_item_type = ?", userId, roleId, LikeItemTypeRole).
			First(&item).Error
		switch {
		case err == nil:
			if !item.Deleted && item.LikeType == LikeEventTypeLike {
				log2.Log().Info("[LikeStoryRole] 已点赞，无需重复", zap.Int64("userId", userId), zap.Int64("roleId", roleId))
				return nil
			}
			updates := map[string]interface{}{
				"deleted":   false,
				"like_type": LikeEventTypeLike,
				"story_id":  storyId,
				"role_id":   roleId,
			}
			if err := tx.Model(&LikeItem{}).Where("id = ?", item.ID).Updates(updates).Error; err != nil {
				return err
			}
			created = true
			return nil
		case errors.Is(err, gorm.ErrRecordNotFound):
			newItem := &LikeItem{
				UserID:       userId,
				StoryID:      storyId,
				RoleID:       roleId,
				LikeType:     LikeEventTypeLike,
				LikeItemType: LikeItemTypeRole,
			}
			if err := tx.Create(newItem).Error; err != nil {
				return err
			}
			created = true
			return nil
		default:
			return err
		}
	})
	if err != nil {
		log2.Log().Error("[LikeStoryRole] 点赞失败", zap.Error(err), zap.Int64("userId", userId), zap.Int64("storyId", storyId), zap.Int64("roleId", roleId))
		return false, err
	}
	if created {
		log2.Log().Info("[LikeStoryRole] 点赞成功", zap.Int64("userId", userId), zap.Int64("storyId", storyId), zap.Int64("roleId", roleId))
	}
	return created, nil
}

func UnLikeStoryRole(ctx context.Context, userId int64, storyId int64, roleId int64) (bool, error) {
	log2.Log().Info("[UnLikeStoryRole] 尝试取消点赞", zap.Int64("userId", userId), zap.Int64("storyId", storyId), zap.Int64("roleId", roleId))
	var removed bool
	err := DataBase().WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var item LikeItem
		err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("user_id = ? AND role_id = ? AND like_item_type = ?", userId, roleId, LikeItemTypeRole).
			First(&item).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			log2.Log().Info("[UnLikeStoryRole] 未找到点赞记录", zap.Int64("userId", userId), zap.Int64("roleId", roleId))
			return nil
		}
		if err != nil {
			return err
		}
		if item.Deleted {
			log2.Log().Info("[UnLikeStoryRole] 已取消点赞", zap.Int64("userId", userId), zap.Int64("roleId", roleId))
			return nil
		}
		updates := map[string]interface{}{
			"deleted": true,
		}
		if err := tx.Model(&LikeItem{}).Where("id = ?", item.ID).Updates(updates).Error; err != nil {
			return err
		}
		removed = true
		return nil
	})
	if err != nil {
		log2.Log().Error("[UnLikeStoryRole] 取消点赞失败", zap.Error(err), zap.Int64("userId", userId), zap.Int64("storyId", storyId), zap.Int64("roleId", roleId))
		return false, err
	}
	if removed {
		log2.Log().Info("[UnLikeStoryRole] 取消点赞成功", zap.Int64("userId", userId), zap.Int64("storyId", storyId), zap.Int64("roleId", roleId))
	}
	return removed, nil
}

// 新增：分页获取LikeItem列表
func GetLikeItemList(ctx context.Context, offset, limit int) ([]*LikeItem, error) {
	var items []*LikeItem
	// 日志：准备分页获取LikeItem列表
	log2.Log().Info("[GetLikeItemList] 分页获取LikeItem列表", zap.Int("offset", offset), zap.Int("limit", limit))
	err := DataBase().Model(&LikeItem{}).
		WithContext(ctx).
		Offset(offset).
		Limit(limit).
		Order("create_at desc").
		Find(&items).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		// 日志：查询失败
		log2.Log().Error("[GetLikeItemList] 查询失败", zap.Error(err), zap.Int("offset", offset), zap.Int("limit", limit))
		return nil, err
	}
	// 日志：查询成功
	log2.Log().Info("[GetLikeItemList] 查询成功", zap.Int("count", len(items)))
	return items, nil
}

// 新增：通过主键唯一查询
func GetLikeItemByID(ctx context.Context, id int64) (*LikeItem, error) {
	item := &LikeItem{}
	// 日志：准备通过主键唯一查询LikeItem
	log2.Log().Info("[GetLikeItemByID] 通过主键唯一查询LikeItem", zap.Int64("id", id))
	err := DataBase().Model(item).
		WithContext(ctx).
		Where("id = ?", id).
		First(item).Error
	if err != nil {
		// 日志：查询失败
		log2.Log().Error("[GetLikeItemByID] 查询失败", zap.Error(err), zap.Int64("id", id))
		return nil, err
	}
	// 日志：查询成功
	log2.Log().Info("[GetLikeItemByID] 查询成功", zap.Any("item", item))
	return item, nil
}
