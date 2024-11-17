package models

import (
	"context"

	log "github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

const (
	LikeTypeUnknown = iota
	LikeTypeLike
	LikeTypeDislike
)

const (
	LikeItemTypeUnknown = iota
	LikeItemTypeGroup
	LikeItemTypeTimeline
	LikeItemTypeStory
	LikeItemTypeStoryboard
	LikeItemTypeRole
	LikeItemTypeComment
)

type LikeItem struct {
	IDBase
	UserID       int64 `json:"user_id,omitempty"`
	GroupID      int64 `json:"group_id,omitempty"`
	TimelineID   int64 `json:"timeline_id,omitempty"`
	StoryID      int64 `json:"story_id,omitempty"`
	StoryboardId int64 `json:"storyboard_id,omitempty"`
	LikeType     int64 `json:"like_type,omitempty"`
	LikeItemType int64 `json:"like_item_type,omitempty"`
}

func (l LikeItem) TableName() string {
	return "like_item"
}

func CreateLikeStoryItem(ctx context.Context, item *LikeItem) error {
	var num int64
	err := DataBase().WithContext(ctx).Model(&LikeItem{}).
		Where("user_id = ? and story_id = ?",
			item.UserID, item.StoryID).
		Count(&num).Error
	if err != nil {
		return err
	}
	if num > 0 {
		return nil
	}
	err = DataBase().WithContext(ctx).Model(&LikeItem{}).Create(item).Error
	if err != nil {
		log.Errorf("create likeitem failed: %s", err.Error())
		return err
	}
	return nil
}

func CreateLikeStoryTimelineItem(ctx context.Context, item *LikeItem) error {
	var num int64
	err := DataBase().WithContext(ctx).Model(&LikeItem{}).
		Where("user_id = ? and story_id = ? and timeline_id = ?",
			item.UserID, item.StoryID, item.TimelineID).
		Count(&num).Error
	if err != nil {
		return err
	}
	if num > 0 {
		return nil
	}
	err = DataBase().WithContext(ctx).Model(&LikeItem{}).Create(item).Error
	if err != nil {
		log.Errorf("create likeitem failed: %s", err.Error())
		return err
	}
	return nil
}

func CreateLikeStoryBoardItem(ctx context.Context, item *LikeItem) error {
	var num int64
	err := DataBase().WithContext(ctx).Model(&LikeItem{}).
		Where("user_id = ? and story_id = ? and storyboard_id = ?",
			item.UserID, item.StoryID, item.StoryboardId).
		Count(&num).Error
	if err != nil {
		log.Error("create likeitem failed: ", err)
		return err
	}
	if num > 0 {
		log.Info("likeitem already exist")
		return nil
	}
	err = DataBase().WithContext(ctx).Model(&LikeItem{}).Create(item).Error
	if err != nil {
		log.Errorf("create likeitem failed: %s", err.Error())
		return err
	}
	return nil
}

func DeleteLikeItem(ctx context.Context, itemID int64) error {
	err := DataBase().WithContext(ctx).Model(&LikeItem{}).
		Update("deleted= ? ", 1).
		Where("id = ?", itemID).Error
	if err != nil {
		log.Error("update like item failed: ", err)
		return err
	}
	return nil
}

func GetLikeItem(ctx context.Context, itemID int64) (*LikeItem, error) {
	item := new(LikeItem)
	err := DataBase().WithContext(ctx).Model(&LikeItem{}).First(item).
		Where("id = ?", itemID).Error
	if err != nil {
		return nil, err
	}
	return item, nil
}

func GetLikeItemByUser(ctx context.Context, userID int) (list []*LikeItem, err error) {
	list = make([]*LikeItem, 0)
	err = DataBase().WithContext(ctx).Model(&LikeItem{}).
		Where("user_id = ?", userID).
		Scan(&list).Error
	if err != nil {
		return nil, err
	}
	return list, nil
}

func GetLikeItemByStory(ctx context.Context, storyId int64) (list []*LikeItem, err error) {
	list = make([]*LikeItem, 0)
	err = DataBase().WithContext(ctx).Model(&LikeItem{}).
		Where("story_id = ?", storyId).
		Scan(&list).Error
	if err != nil {
		return nil, err
	}
	return list, nil
}

func GetLikeItemByStoryAndUser(ctx context.Context, storyId int64, userId int) (*LikeItem, error) {
	item := new(LikeItem)
	err := DataBase().WithContext(ctx).Model(&LikeItem{}).
		Where("story_id = ?", storyId).
		Where("user_id = ?", userId).
		Where("deleted = ?", 0).
		Where("like_item_type = ?", LikeItemTypeStory).
		First(item).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return nil, err
	}
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	return item, nil
}

func GetLikeItemByStoryBoard(ctx context.Context, storyId int64, storyboardId int64) (list []*LikeItem, err error) {
	list = make([]*LikeItem, 0)
	err = DataBase().WithContext(ctx).Model(&LikeItem{}).
		Where("story_id = ? and storyboard_id = ?", storyId, storyboardId).
		Scan(&list).Error
	if err != nil {
		return nil, err
	}
	return list, nil
}

type WatchType int64

const (
	WatchTypeUnknown WatchType = iota
	WatchTypeIsWatch
	WatchTypeIsUnWatch
)

type WatchItemType int64

const (
	WatchItemTypeUnknown = iota
	WatchItemTypeUser
	WatchItemTypeGroup
	WatchItemTypeTimeline
	WatchItemTypeStory
	WatchItemTypeStoryboard
)

type WatchItem struct {
	IDBase
	UserID         int64         `json:"user_id,omitempty"`
	GroupID        int64         `json:"group_id,omitempty"`
	TimelineID     int64         `json:"timeline_id,omitempty"`
	StoryID        int64         `json:"story_id,omitempty"`
	RoleID         int64         `json:"role_id,omitempty"`
	WatchType      WatchType     `json:"watch_type,omitempty"`
	WatchItemTypes WatchItemType `json:"watch_item_type,omitempty"`
}

func (w WatchItem) TableName() string {
	return "watch_item"
}

func GetWatchItemByGroup(ctx context.Context, groupId int64) (list []*WatchItem, err error) {
	list = make([]*WatchItem, 0)
	err = DataBase().WithContext(ctx).Model(&WatchItem{}).
		Where("group_id = ?", groupId).
		Scan(&list).Error
	if err != nil {
		return nil, err
	}
	return list, nil
}

func GetWatchItemByStory(ctx context.Context, storyId int64) (list []*WatchItem, err error) {
	list = make([]*WatchItem, 0)
	err = DataBase().WithContext(ctx).Model(&WatchItem{}).
		Where("story_id = ?", storyId).
		Scan(&list).Error
	if err != nil {
		return nil, err
	}
	return list, nil
}

func GetWatchItemByUser(ctx context.Context, userID int) (list []*WatchItem, err error) {
	list = make([]*WatchItem, 0)
	err = DataBase().WithContext(ctx).Model(&WatchItem{}).
		Where("user_id = ?", userID).
		Scan(&list).Error
	if err != nil {
		return nil, err
	}
	return list, nil
}

func CreateWatchStoryItem(ctx context.Context, item *WatchItem) error {
	var num int64
	err := DataBase().WithContext(ctx).Model(&WatchItem{}).
		Where("user_id = ? and story_id = ?",
			item.UserID, item.StoryID).
		Count(&num).Error
	if err != nil {
		log.Error("create WatchItem failed: ", err)
		return err
	}
	if num > 0 {
		log.Info("WatchItem already exist")
		return nil
	}
	err = DataBase().WithContext(ctx).Model(&WatchItem{}).Create(item).Error
	if err != nil {
		log.Errorf("create WatchItem failed: %s", err.Error())
		return err
	}
	return nil
}

func CreateWatchGroupItem(ctx context.Context, item *WatchItem) error {
	var num int64
	err := DataBase().WithContext(ctx).Model(&WatchItem{}).
		Where("user_id = ? and group_id = ?",
			item.UserID, item.GroupID).
		Count(&num).Error
	if err != nil {
		log.Error("create WatchItem failed: ", err)
		return err
	}
	if num > 0 {
		log.Info("WatchItem already exist")
		return nil
	}
	err = DataBase().WithContext(ctx).Model(&WatchItem{}).Create(item).Error
	if err != nil {
		log.Errorf("create WatchItem failed: %s", err.Error())
		return err
	}
	return nil
}

func CreateWatchRoleItem(ctx context.Context, item *WatchItem) error {
	var num int64
	err := DataBase().WithContext(ctx).Model(&WatchItem{}).
		Where("user_id = ? and role_id = ?",
			item.UserID, item.RoleID).
		Count(&num).Error
	if err != nil {
		log.Error("create WatchItem failed: ", err)
		return err
	}
	if num > 0 {
		log.Info("WatchItem already exist")
		return nil
	}
	err = DataBase().WithContext(ctx).Model(&WatchItem{}).Create(item).Error
	if err != nil {
		log.Errorf("create WatchItem failed: %s", err.Error())
		return err
	}
	return nil
}

func UnWatchItem(ctx context.Context, itemID int64) error {
	err := DataBase().WithContext(ctx).Model(&WatchItem{}).
		Update("deleted= ? ", 1).
		Where("id = ?", itemID).Error
	if err != nil {
		log.Error("update watch item failed: ", err)
		return err
	}
	return nil
}