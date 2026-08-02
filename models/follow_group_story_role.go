package models

import (
	"context"
	"errors"

	log2 "github.com/grapery/grapery/utils/log"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type WatchType int64

const (
	WatchTypeUnknown WatchType = iota
	WatchTypeIsWatching
	WatchTypeIsUnWatch
)

type WatchItemType int64

const (
	WatchItemTypeUnknown = iota
	WatchItemTypeGroup
	WatchItemTypeTimeline
	WatchItemTypeStory
	WatchItemTypeStoryboard
	WatchItemTypeStoryRole
)

type WatchItem struct {
	ID uint `gorm:"primary_key,column:id" json:"id,omitempty"`
	IDBase
	UserID        int64         `gorm:"column:user_id;index;uniqueIndex:uniq_watch_item,priority:1" json:"user_id,omitempty"`
	GroupID       int64         `gorm:"column:group_id;uniqueIndex:uniq_watch_item,priority:4" json:"group_id,omitempty"`
	TimelineID    int64         `gorm:"column:timeline_id;uniqueIndex:uniq_watch_item,priority:5" json:"timeline_id,omitempty"`
	StoryID       int64         `gorm:"column:story_id;uniqueIndex:uniq_watch_item,priority:6" json:"story_id,omitempty"`
	RoleID        int64         `gorm:"column:role_id;uniqueIndex:uniq_watch_item,priority:7" json:"role_id,omitempty"`
	WatchType     WatchType     `gorm:"column:watch_type" json:"watch_type,omitempty"`
	WatchItemType WatchItemType `gorm:"column:watch_item_type;uniqueIndex:uniq_watch_item,priority:2" json:"watch_item_type,omitempty"`
}

func (w WatchItem) TableName() string {
	return "watch_item"
}

func GetWatchItemByGroup(ctx context.Context, groupId int64) (list []*WatchItem, err error) {
	list = make([]*WatchItem, 0)
	// 日志：准备查询小组关注记录
	log2.Log().Info("[GetWatchItemByGroup] 查询小组关注记录", zap.Int64("groupId", groupId))
	err = DataBase().WithContext(ctx).Model(&WatchItem{}).
		Where("group_id = ?", groupId).
		Scan(&list).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		// 日志：查询失败
		log2.Log().Error("[GetWatchItemByGroup] 查询失败", zap.Error(err), zap.Int64("groupId", groupId))
		return nil, err
	}
	if err == gorm.ErrRecordNotFound {
		// 日志：未找到记录
		log2.Log().Warn("[GetWatchItemByGroup] 未找到记录", zap.Int64("groupId", groupId))
		return nil, nil
	}
	// 日志：查询成功
	log2.Log().Info("[GetWatchItemByGroup] 查询成功", zap.Int("count", len(list)))
	return list, nil
}

func GetWatchItemByStory(ctx context.Context, storyId int64) (list []*WatchItem, err error) {
	list = make([]*WatchItem, 0)
	// 日志：准备查询故事关注记录
	log2.Log().Info("[GetWatchItemByStory] 查询故事关注记录", zap.Int64("storyId", storyId))
	err = DataBase().WithContext(ctx).Model(&WatchItem{}).
		Where("story_id = ?", storyId).
		Scan(&list).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		// 日志：查询失败
		log2.Log().Error("[GetWatchItemByStory] 查询失败", zap.Error(err), zap.Int64("storyId", storyId))
		return nil, err
	}
	if err == gorm.ErrRecordNotFound {
		// 日志：未找到记录
		log2.Log().Warn("[GetWatchItemByStory] 未找到记录", zap.Int64("storyId", storyId))
		return nil, nil
	}
	// 日志：查询成功
	log2.Log().Info("[GetWatchItemByStory] 查询成功", zap.Int("count", len(list)))
	return list, nil
}

func GetWatchItemByUser(ctx context.Context, userID int) (list []*WatchItem, err error) {
	list = make([]*WatchItem, 0)
	// 日志：准备查询用户关注记录
	log2.Log().Info("[GetWatchItemByUser] 查询用户关注记录", zap.Int("userID", userID))
	err = DataBase().WithContext(ctx).Model(&WatchItem{}).
		Where("user_id = ?", userID).
		Scan(&list).Error
	if err != nil {
		// 日志：查询失败
		log2.Log().Error("[GetWatchItemByUser] 查询失败", zap.Error(err), zap.Int("userID", userID))
		return nil, err
	}
	// 日志：查询成功
	log2.Log().Info("[GetWatchItemByUser] 查询成功", zap.Int("count", len(list)))
	return list, nil
}

func CreateWatchStoryItem(ctx context.Context, userId int, storyId int64, groupId int64) (bool, error) {
	return WatchStory(ctx, int64(userId), storyId, groupId)
}

func UnWatchStoryItem(ctx context.Context, userId int, storyId int64) (bool, error) {
	return UnWatchStory(ctx, int64(userId), storyId)
}

func WatchStory(ctx context.Context, userId int64, storyId int64, groupId int64) (bool, error) {
	log2.Log().Info("[WatchStory] 尝试关注故事", zap.Int64("userId", userId), zap.Int64("storyId", storyId))
	var created bool
	err := DataBase().WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var item WatchItem
		err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("user_id = ? AND story_id = ? AND watch_item_type = ?", userId, storyId, WatchItemTypeStory).
			First(&item).Error
		switch {
		case err == nil:
			if !item.Deleted && item.WatchType == WatchTypeIsWatching {
				log2.Log().Info("[WatchStory] 已关注，无需重复", zap.Int64("userId", userId), zap.Int64("storyId", storyId))
				return nil
			}
			updates := map[string]interface{}{
				"deleted":    false,
				"watch_type": WatchTypeIsWatching,
				"story_id":   storyId,
				"group_id":   groupId,
			}
			if err := tx.Model(&WatchItem{}).
				Where("id = ?", item.ID).
				Updates(updates).Error; err != nil {
				return err
			}
			created = true
		case errors.Is(err, gorm.ErrRecordNotFound):
			newItem := &WatchItem{
				UserID:        userId,
				StoryID:       storyId,
				GroupID:       groupId,
				WatchType:     WatchTypeIsWatching,
				WatchItemType: WatchItemTypeStory,
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

		if err := tx.Model(&Story{}).
			Where("id = ?", storyId).
			Update("follow_count", gorm.Expr("CASE WHEN follow_count IS NULL THEN 1 ELSE follow_count + 1 END")).Error; err != nil {
			return err
		}
		if groupId > 0 {
			if err := tx.Model(&GroupProfile{}).
				Where("group_id = ? AND deleted = 0", groupId).
				Update("followers", gorm.Expr("CASE WHEN followers IS NULL THEN 1 ELSE followers + 1 END")).Error; err != nil {
				log2.Log().Warn("[WatchStory] 更新群组关注人数失败", zap.Error(err), zap.Int64("groupId", groupId))
			}
		}
		profile := &UserProfile{UserId: userId}
		if err := profile.IncrementWatchingStoryNum(ctx); err != nil {
			log2.Log().Warn("[WatchStory] 增加用户关注故事数失败", zap.Error(err), zap.Int64("userId", userId), zap.Int64("storyId", storyId))
		}
		return nil
	})
	if err != nil {
		log2.Log().Error("[WatchStory] 关注故事失败", zap.Error(err), zap.Int64("userId", userId), zap.Int64("storyId", storyId))
		return false, err
	}
	if created {
		log2.Log().Info("[WatchStory] 关注故事成功", zap.Int64("userId", userId), zap.Int64("storyId", storyId))
	}
	return created, nil
}

func UnWatchStory(ctx context.Context, userId int64, storyId int64) (bool, error) {
	log2.Log().Info("[UnWatchStory] 尝试取消关注故事", zap.Int64("userId", userId), zap.Int64("storyId", storyId))
	var removed bool
	err := DataBase().WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var item WatchItem
		err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("user_id = ? AND story_id = ? AND watch_item_type = ?", userId, storyId, WatchItemTypeStory).
			First(&item).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			log2.Log().Info("[UnWatchStory] 未找到关注记录", zap.Int64("userId", userId), zap.Int64("storyId", storyId))
			return nil
		}
		if err != nil {
			return err
		}
		if item.Deleted || item.WatchType == WatchTypeIsUnWatch {
			log2.Log().Info("[UnWatchStory] 关注记录已取消", zap.Int64("userId", userId), zap.Int64("storyId", storyId))
			return nil
		}

		if err := tx.Model(&WatchItem{}).
			Where("id = ?", item.ID).
			Updates(map[string]interface{}{
				"deleted":    true,
				"watch_type": WatchTypeIsUnWatch,
			}).Error; err != nil {
			return err
		}

		if err := tx.Model(&Story{}).
			Where("id = ?", storyId).
			Update("follow_count", gorm.Expr("CASE WHEN follow_count > 0 THEN follow_count - 1 ELSE 0 END")).Error; err != nil {
			return err
		}
		profile := &UserProfile{UserId: userId}
		if err := profile.DecrementWatchingStoryNum(ctx); err != nil {
			log2.Log().Warn("[UnWatchStory] 减少用户关注故事数失败", zap.Error(err), zap.Int64("userId", userId), zap.Int64("storyId", storyId))
		}
		var story Story
		if err := tx.Where("id = ?", storyId).Select("group_id").First(&story).Error; err == nil && story.GroupID > 0 {
			if err := tx.Model(&GroupProfile{}).
				Where("group_id = ? AND deleted = 0", story.GroupID).
				Update("followers", gorm.Expr("CASE WHEN followers > 0 THEN followers - 1 ELSE 0 END")).Error; err != nil {
				log2.Log().Warn("[UnWatchStory] 更新群组关注人数失败", zap.Error(err), zap.Int64("groupId", story.GroupID))
			}
		}

		removed = true
		return nil
	})
	if err != nil {
		log2.Log().Error("[UnWatchStory] 取消关注故事失败", zap.Error(err), zap.Int64("userId", userId), zap.Int64("storyId", storyId))
		return false, err
	}
	if removed {
		log2.Log().Info("[UnWatchStory] 取消关注故事成功", zap.Int64("userId", userId), zap.Int64("storyId", storyId))
	}
	return removed, nil
}

func CreateWatchGroupItem(ctx context.Context, userId int, groupId int64) error {
	log2.Log().Info("[CreateWatchGroupItem] 尝试关注群组", zap.Int("userId", userId), zap.Int64("groupId", groupId))
	return DataBase().WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var item WatchItem
		err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("user_id = ? AND group_id = ? AND watch_item_type = ?", userId, groupId, WatchItemTypeGroup).
			First(&item).Error
		created := false
		switch {
		case err == nil:
			if !item.Deleted && item.WatchType == WatchTypeIsWatching {
				log2.Log().Info("[CreateWatchGroupItem] 已关注，无需重复", zap.Int("userId", userId), zap.Int64("groupId", groupId))
				return nil
			}
			updates := map[string]interface{}{
				"deleted":    false,
				"watch_type": WatchTypeIsWatching,
			}
			if err := tx.Model(&WatchItem{}).Where("id = ?", item.ID).Updates(updates).Error; err != nil {
				return err
			}
			created = true
		case errors.Is(err, gorm.ErrRecordNotFound):
			if err := tx.Create(&WatchItem{
				UserID:        int64(userId),
				GroupID:       groupId,
				WatchType:     WatchTypeIsWatching,
				WatchItemType: WatchItemTypeGroup,
			}).Error; err != nil {
				return err
			}
			created = true
		default:
			return err
		}
		if created {
			profile := &UserProfile{UserId: int64(userId)}
			if err := profile.IncrementWatchingGroupNum(ctx); err != nil {
				log2.Log().Warn("[CreateWatchGroupItem] 增加用户关注群组数失败", zap.Error(err), zap.Int("userId", userId), zap.Int64("groupId", groupId))
			}
			if err := tx.Model(&GroupProfile{}).
				Where("group_id = ? AND deleted = 0", groupId).
				Update("followers", gorm.Expr("CASE WHEN followers IS NULL THEN 1 ELSE followers + 1 END")).Error; err != nil {
				log2.Log().Warn("[CreateWatchGroupItem] 更新群组关注人数失败", zap.Error(err), zap.Int64("groupId", groupId))
			}
			if err := IncGroupProfileFollowers(ctx, groupId); err != nil {
				log2.Log().Warn("[CreateWatchGroupItem] IncGroupProfileFollowers failed", zap.Error(err), zap.Int64("groupId", groupId))
			}
		}
		log2.Log().Info("[CreateWatchGroupItem] 处理完成", zap.Int("userId", userId), zap.Int64("groupId", groupId), zap.Bool("created", created))
		return nil
	})
}

func UnWatchGroupItem(ctx context.Context, userId int, groupId int64) error {
	log2.Log().Info("[UnWatchGroupItem] 尝试取消关注群组", zap.Int("userId", userId), zap.Int64("groupId", groupId))
	return DataBase().WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var item WatchItem
		err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("user_id = ? AND group_id = ? AND watch_item_type = ?", userId, groupId, WatchItemTypeGroup).
			First(&item).Error
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				log2.Log().Info("[UnWatchGroupItem] 未找到关注记录", zap.Int("userId", userId), zap.Int64("groupId", groupId))
				return nil
			}
			return err
		}
		if item.Deleted || item.WatchType != WatchTypeIsWatching {
			log2.Log().Info("[UnWatchGroupItem] 已处于未关注状态", zap.Int("userId", userId), zap.Int64("groupId", groupId))
			return nil
		}
		if err := tx.Model(&WatchItem{}).
			Where("id = ?", item.ID).
			Updates(map[string]interface{}{
				"deleted":    true,
				"watch_type": WatchTypeIsUnWatch,
			}).Error; err != nil {
			return err
		}
		profile := &UserProfile{UserId: int64(userId)}
		if err := profile.DecrementWatchingGroupNum(ctx); err != nil {
			log2.Log().Warn("[UnWatchGroupItem] 减少用户关注群组数失败", zap.Error(err), zap.Int("userId", userId), zap.Int64("groupId", groupId))
		}
		if err := tx.Model(&GroupProfile{}).
			Where("group_id = ? AND deleted = 0", groupId).
			Update("followers", gorm.Expr("CASE WHEN followers > 0 THEN followers - 1 ELSE 0 END")).Error; err != nil {
			log2.Log().Warn("[UnWatchGroupItem] 更新群组关注人数失败", zap.Error(err), zap.Int64("groupId", groupId))
		}
		if err := DecGroupProfileFollowers(ctx, groupId); err != nil {
			log2.Log().Warn("[UnWatchGroupItem] DecGroupProfileFollowers failed", zap.Error(err), zap.Int64("groupId", groupId))
		}
		log2.Log().Info("[UnWatchGroupItem] 取消关注成功", zap.Int("userId", userId), zap.Int64("groupId", groupId))
		return nil
	})
}

func UnWatchGroupItemByGroup(ctx context.Context, groupId int64) error {
	// 日志：准备取消关注
	log2.Log().Info("[UnWatchGroupItemByGroup] 取消关注", zap.Int64("groupId", groupId))
	err := DataBase().WithContext(ctx).Model(&WatchItem{}).
		Where("group_id = ?",
			groupId).
		Update("deleted", 1).Error
	if err != nil {
		// 日志：取消失败
		log2.Log().Error("[UnWatchGroupItemByGroup] 取消失败", zap.Error(err), zap.Int64("groupId", groupId))
		return err
	}
	// 日志：取消成功
	log2.Log().Info("[UnWatchGroupItemByGroup] 取消成功", zap.Int64("groupId", groupId))
	return nil
}

func CreateWatchRoleItem(ctx context.Context, userId int, storyId int64, roleId int64) error {
	var num int64
	// 日志：检查是否已存在关注记录
	log2.Log().Info("[CreateWatchRoleItem] 检查是否已存在", zap.Int("userId", userId), zap.Int64("storyId", storyId), zap.Int64("roleId", roleId))
	err := DataBase().WithContext(ctx).Model(&WatchItem{}).
		Where("user_id = ? and role_id = ?",
			userId, roleId).
		Count(&num).Error
	if err != nil {
		// 日志：数据库查询失败
		log2.Log().Error("[CreateWatchRoleItem] 查询失败", zap.Error(err), zap.Int("userId", userId), zap.Int64("roleId", roleId))
		return err
	}
	if num > 0 {
		// 日志：已存在
		log2.Log().Info("[CreateWatchRoleItem] 已存在", zap.Int("userId", userId), zap.Int64("roleId", roleId))
		return nil
	}
	// 日志：准备创建关注记录
	log2.Log().Info("[CreateWatchRoleItem] 创建关注记录", zap.Int("userId", userId), zap.Int64("storyId", storyId), zap.Int64("roleId", roleId))
	err = DataBase().WithContext(ctx).Model(&WatchItem{}).Create(&WatchItem{
		UserID:        int64(userId),
		StoryID:       storyId,
		RoleID:        roleId,
		WatchType:     WatchTypeIsWatching,
		WatchItemType: WatchItemTypeStoryRole,
	}).Error
	if err != nil {
		// 日志：创建失败
		log2.Log().Error("[CreateWatchRoleItem] 创建失败", zap.Error(err), zap.Int("userId", userId), zap.Int64("storyId", storyId), zap.Int64("roleId", roleId))
		return err
	}
	// 日志：创建成功
	log2.Log().Info("[CreateWatchRoleItem] 创建成功", zap.Int("userId", userId), zap.Int64("storyId", storyId), zap.Int64("roleId", roleId))
	return nil
}

func WatchStoryRole(ctx context.Context, userId int64, storyId int64, roleId int64) (bool, error) {
	log2.Log().Info("[WatchStoryRole] 尝试关注角色", zap.Int64("userId", userId), zap.Int64("storyId", storyId), zap.Int64("roleId", roleId))
	var created bool
	err := DataBase().WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var item WatchItem
		err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("user_id = ? AND role_id = ? AND watch_item_type = ?", userId, roleId, WatchItemTypeStoryRole).
			First(&item).Error
		switch {
		case err == nil:
			if !item.Deleted && item.WatchType == WatchTypeIsWatching {
				log2.Log().Info("[WatchStoryRole] 已关注，无需重复", zap.Int64("userId", userId), zap.Int64("roleId", roleId))
				return nil
			}
			updates := map[string]interface{}{
				"deleted":    false,
				"watch_type": WatchTypeIsWatching,
				"story_id":   storyId,
				"role_id":    roleId,
			}
			if err := tx.Model(&WatchItem{}).Where("id = ?", item.ID).Updates(updates).Error; err != nil {
				return err
			}
			created = true
			return nil
		case errors.Is(err, gorm.ErrRecordNotFound):
			newItem := &WatchItem{
				UserID:        userId,
				StoryID:       storyId,
				RoleID:        roleId,
				WatchType:     WatchTypeIsWatching,
				WatchItemType: WatchItemTypeStoryRole,
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
		log2.Log().Error("[WatchStoryRole] 关注失败", zap.Error(err), zap.Int64("userId", userId), zap.Int64("storyId", storyId), zap.Int64("roleId", roleId))
		return false, err
	}
	if created {
		log2.Log().Info("[WatchStoryRole] 关注成功", zap.Int64("userId", userId), zap.Int64("storyId", storyId), zap.Int64("roleId", roleId))
	}
	return created, nil
}

func UnWatchStoryRole(ctx context.Context, userId int64, storyId int64, roleId int64) (bool, error) {
	log2.Log().Info("[UnWatchStoryRole] 尝试取消关注", zap.Int64("userId", userId), zap.Int64("storyId", storyId), zap.Int64("roleId", roleId))
	var removed bool
	err := DataBase().WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var item WatchItem
		err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("user_id = ? AND role_id = ? AND watch_item_type = ?", userId, roleId, WatchItemTypeStoryRole).
			First(&item).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			log2.Log().Info("[UnWatchStoryRole] 未找到关注记录", zap.Int64("userId", userId), zap.Int64("roleId", roleId))
			return nil
		}
		if err != nil {
			return err
		}
		if item.Deleted || item.WatchType == WatchTypeIsUnWatch {
			log2.Log().Info("[UnWatchStoryRole] 已取消关注", zap.Int64("userId", userId), zap.Int64("roleId", roleId))
			return nil
		}
		updates := map[string]interface{}{
			"deleted":    true,
			"watch_type": WatchTypeIsUnWatch,
		}
		if err := tx.Model(&WatchItem{}).Where("id = ?", item.ID).Updates(updates).Error; err != nil {
			return err
		}
		removed = true
		return nil
	})
	if err != nil {
		log2.Log().Error("[UnWatchStoryRole] 取消关注失败", zap.Error(err), zap.Int64("userId", userId), zap.Int64("storyId", storyId), zap.Int64("roleId", roleId))
		return false, err
	}
	if removed {
		log2.Log().Info("[UnWatchStoryRole] 取消关注成功", zap.Int64("userId", userId), zap.Int64("storyId", storyId), zap.Int64("roleId", roleId))
	}
	return removed, nil
}

// 根据一组故事id，以及一个用户id来获取关注的列表
func GetWatchItemByStoriesAndUser(ctx context.Context, storyIds []int64, userId int) (list []*WatchItem, err error) {
	list = make([]*WatchItem, 0)
	// 日志：准备批量查询用户对多故事的点赞记录
	log2.Log().Info("[GetWatchItemByStoriesAndUser] 批量查询用户对故事的点赞记录", zap.Any("storyIds", storyIds), zap.Int("userId", userId))
	err = DataBase().WithContext(ctx).Model(&WatchItem{}).
		Where("story_id in (?) and user_id = ?", storyIds, userId).
		Where("deleted = ?", 0).
		Where("watch_item_type = ?", WatchItemTypeStory).
		Scan(&list).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		// 日志：查询失败
		log2.Log().Error("[GetWatchItemByStoriesAndUser] 查询失败", zap.Error(err), zap.Any("storyIds", storyIds), zap.Int("userId", userId))
		return nil, err
	}
	if err == gorm.ErrRecordNotFound {
		// 日志：未找到记录
		log2.Log().Warn("[GetWatchItemByStoriesAndUser] 未找到记录", zap.Any("storyIds", storyIds), zap.Int("userId", userId))
		return nil, nil
	}
	// 日志：查询成功
	log2.Log().Info("[GetWatchItemByStoriesAndUser] 查询成功", zap.Int("count", len(list)))
	return list, nil
}

// 根据一组故事id，以及一个用户id来获取关注的列表
func GetWatchItemByStoryAndUser(ctx context.Context, storyId int64, userId int) (*WatchItem, error) {
	item := new(WatchItem)
	// 日志：准备查询用户对故事的点赞记录
	log2.Log().Info("[GetWatchItemByStoryAndUser] 查询用户对故事的点赞记录", zap.Int64("storyId", storyId), zap.Int("userId", userId))
	err := DataBase().WithContext(ctx).Model(&WatchItem{}).
		Where("story_id = ? and user_id = ?", storyId, userId).
		Where("deleted = ?", 0).
		Where("watch_item_type = ?", WatchItemTypeStory).
		First(item).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		// 日志：查询失败
		log2.Log().Error("[GetWatchItemByStoryAndUser] 查询失败", zap.Error(err), zap.Int64("storyId", storyId), zap.Int("userId", userId))
		return nil, err
	}
	if err == gorm.ErrRecordNotFound {
		// 日志：未找到记录
		log2.Log().Warn("[GetWatchItemByStoryAndUser] 未找到记录", zap.Int64("storyId", storyId), zap.Int("userId", userId))
		return nil, nil
	}
	// 日志：查询成功
	log2.Log().Info("[GetWatchItemByStoryAndUser] 查询成功", zap.Any("item", item))
	return item, nil
}

func GetWatchItemByStoryRoleAndUser(ctx context.Context, roleId int64, userId int64) (*WatchItem, error) {
	item := new(WatchItem)
	// 日志：准备查询用户对角色的点赞记录
	log2.Log().Info("[GetWatchItemByStoryRoleAndUser] 查询用户对角色的点赞记录", zap.Int64("roleId", roleId), zap.Int64("userId", userId))
	err := DataBase().WithContext(ctx).Model(&WatchItem{}).
		Where("role_id = ? and user_id = ?", roleId, userId).
		Where("deleted = ?", 0).
		Where("watch_item_type = ?", WatchItemTypeStoryRole).
		First(item).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		// 日志：查询失败
		log2.Log().Error("[GetWatchItemByStoryRoleAndUser] 查询失败", zap.Error(err), zap.Int64("roleId", roleId), zap.Int64("userId", userId))
		return nil, err
	}
	if err == gorm.ErrRecordNotFound {
		// 日志：未找到记录
		log2.Log().Warn("[GetWatchItemByStoryRoleAndUser] 未找到记录", zap.Int64("roleId", roleId), zap.Int64("userId", userId))
		return nil, nil
	}
	// 日志：查询成功
	log2.Log().Info("[GetWatchItemByStoryRoleAndUser] 查询成功", zap.Any("item", item))
	return item, nil
}

func GetWatchItemByGroupAndUser(ctx context.Context, groupId int64, userId int64) (*WatchItem, error) {
	item := new(WatchItem)
	// 日志：准备查询用户对小组的点赞记录
	log2.Log().Info("[GetWatchItemByGroupAndUser] 查询用户对小组的点赞记录", zap.Int64("groupId", groupId), zap.Int64("userId", userId))
	err := DataBase().WithContext(ctx).Model(&WatchItem{}).
		Where("group_id = ? and user_id = ?", groupId, userId).
		Where("deleted = ?", 0).
		Where("watch_item_type = ?", WatchItemTypeGroup).
		First(item).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		// 日志：查询失败
		log2.Log().Error("[GetWatchItemByGroupAndUser] 查询失败", zap.Error(err), zap.Int64("groupId", groupId), zap.Int64("userId", userId))
		return nil, err
	}
	if err == gorm.ErrRecordNotFound {
		// 日志：未找到记录
		log2.Log().Warn("[GetWatchItemByGroupAndUser] 未找到记录", zap.Int64("groupId", groupId), zap.Int64("userId", userId))
		return nil, nil
	}
	// 日志：查询成功
	log2.Log().Info("[GetWatchItemByGroupAndUser] 查询成功", zap.Any("item", item))
	return item, nil
}

// 根据一组角色id，以及一个用户id来获取关注的列表
func GetWatchItemByRolesAndUser(ctx context.Context, roleIds []int64, userId int) (list []*WatchItem, err error) {
	list = make([]*WatchItem, 0)
	// 日志：准备批量查询用户对角色的点赞记录
	log2.Log().Info("[GetWatchItemByRolesAndUser] 批量查询用户对角色的点赞记录", zap.Any("roleIds", roleIds), zap.Int("userId", userId))
	err = DataBase().WithContext(ctx).Model(&WatchItem{}).
		Where("role_id in (?) and user_id = ?", roleIds, userId).
		Where("deleted = ?", 0).
		Where("watch_item_type = ?", WatchItemTypeStoryRole).
		Scan(&list).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		// 日志：查询失败
		log2.Log().Error("[GetWatchItemByRolesAndUser] 查询失败", zap.Error(err), zap.Any("roleIds", roleIds), zap.Int("userId", userId))
		return nil, err
	}
	if err == gorm.ErrRecordNotFound {
		// 日志：未找到记录
		log2.Log().Warn("[GetWatchItemByRolesAndUser] 未找到记录", zap.Any("roleIds", roleIds), zap.Int("userId", userId))
		return nil, nil
	}
	// 日志：查询成功
	log2.Log().Info("[GetWatchItemByRolesAndUser] 查询成功", zap.Int("count", len(list)))
	return list, nil
}

// 根据一组小组id，以及一个用户id来获取喜欢的列表
func GetWatchItemByGroupsAndUser(ctx context.Context, groupIds []int64, userId int) (list []*WatchItem, err error) {
	list = make([]*WatchItem, 0)
	// 日志：准备批量查询用户对小组的点赞记录
	log2.Log().Info("[GetWatchItemByGroupsAndUser] 批量查询用户对小组的点赞记录", zap.Any("groupIds", groupIds), zap.Int("userId", userId))
	err = DataBase().WithContext(ctx).Model(&WatchItem{}).
		Where("group_id in (?) and user_id = ?", groupIds, userId).
		Where("deleted = ?", 0).
		Where("watch_item_type = ?", WatchItemTypeGroup).
		Scan(&list).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		// 日志：查询失败
		log2.Log().Error("[GetWatchItemByGroupsAndUser] 查询失败", zap.Error(err), zap.Any("groupIds", groupIds), zap.Int("userId", userId))
		return nil, err
	}
	if err == gorm.ErrRecordNotFound {
		// 日志：未找到记录
		log2.Log().Warn("[GetWatchItemByGroupsAndUser] 未找到记录", zap.Any("groupIds", groupIds), zap.Int("userId", userId))
		return nil, nil
	}
	// 日志：查询成功
	log2.Log().Info("[GetWatchItemByGroupsAndUser] 查询成功", zap.Int("count", len(list)))
	return list, nil
}

func GetStoriesIdByUserFollow(ctx context.Context, userId int64) ([]int64, error) {
	var storiesIds []int64
	// 日志：准备查询用户关注的所有故事ID
	log2.Log().Info("[GetStoriesIdByUserFollow] 查询用户关注的所有故事ID", zap.Int64("userId", userId))
	if err := DataBase().Model(&WatchItem{}).
		Where("user_id = ?", userId).
		Where("watch_item_type = ?", WatchItemTypeStory).
		Where("watch_type = ?", WatchTypeIsWatching).
		Where("deleted = ?", 0).
		Pluck("story_id", &storiesIds).Error; err != nil {
		// 日志：查询失败
		log2.Log().Error("[GetStoriesIdByUserFollow] 查询失败", zap.Error(err), zap.Int64("userId", userId))
		return nil, err
	} else {
		if err != nil && err != gorm.ErrRecordNotFound {
			// 日志：查询失败
			log2.Log().Error("[GetStoriesIdByUserFollow] 查询失败", zap.Error(err), zap.Int64("userId", userId))
			return nil, err
		}
		if err == gorm.ErrRecordNotFound {
			// 日志：未找到记录
			log2.Log().Warn("[GetStoriesIdByUserFollow] 未找到记录", zap.Int64("userId", userId))
			return nil, nil
		}
	}
	// 日志：查询成功
	log2.Log().Info("[GetStoriesIdByUserFollow] 查询成功", zap.Int64s("storyIds", storiesIds))
	return storiesIds, nil
}

func GetStoryRolesIDByUserFollow(ctx context.Context, userId int64) ([]int64, error) {
	var rolesIds []int64
	// 日志：准备查询用户关注的所有角色ID
	log2.Log().Info("[GetStoryRolesIDByUserFollow] 查询用户关注的所有角色ID", zap.Int64("userId", userId))
	if err := DataBase().Model(&WatchItem{}).
		Where("user_id = ?", userId).
		Where("watch_item_type = ?", WatchItemTypeStoryRole).
		Where("watch_type = ?", WatchTypeIsWatching).
		Where("deleted = ?", 0).
		Pluck("role_id", &rolesIds).Error; err != nil {
		// 日志：查询失败
		log2.Log().Error("[GetStoryRolesIDByUserFollow] 查询失败", zap.Error(err), zap.Int64("userId", userId))
		return nil, err
	} else {
		if err != nil && err != gorm.ErrRecordNotFound {
			// 日志：查询失败
			log2.Log().Error("[GetStoryRolesIDByUserFollow] 查询失败", zap.Error(err), zap.Int64("userId", userId))
			return nil, err
		}
		if err == gorm.ErrRecordNotFound {
			// 日志：未找到记录
			log2.Log().Warn("[GetStoryRolesIDByUserFollow] 未找到记录", zap.Int64("userId", userId))
			return nil, nil
		}
	}
	// 日志：查询成功
	log2.Log().Info("[GetStoryRolesIDByUserFollow] 查询成功", zap.Int64s("roleIds", rolesIds))
	return rolesIds, nil
}
