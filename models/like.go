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
	RoleID       int64 `json:"role_id,omitempty"`
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
	if err != nil && err != gorm.ErrRecordNotFound {
		return err
	}
	if err == gorm.ErrRecordNotFound {
		return nil
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
	if err != nil && err != gorm.ErrRecordNotFound {
		return err
	}
	if err == gorm.ErrRecordNotFound {
		return nil
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
	if err != nil && err != gorm.ErrRecordNotFound {
		return nil, err
	}
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	return item, nil
}

func GetLikeItemByUser(ctx context.Context, userID int) (list []*LikeItem, err error) {
	list = make([]*LikeItem, 0)
	err = DataBase().WithContext(ctx).Model(&LikeItem{}).
		Where("user_id = ?", userID).
		Scan(&list).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return nil, err
	}
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	return list, nil
}

func GetLikeItemByStory(ctx context.Context, storyId int64) (list []*LikeItem, err error) {
	list = make([]*LikeItem, 0)
	err = DataBase().WithContext(ctx).Model(&LikeItem{}).
		Where("story_id = ?", storyId).
		Scan(&list).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return nil, err
	}
	if err == gorm.ErrRecordNotFound {
		return nil, nil
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

// 根据一组故事id，以及一个用户id来获取喜欢的列表
func GetLikeItemByStoriesAndUser(ctx context.Context, storyIds []int64, userId int) (list []*LikeItem, err error) {
	list = make([]*LikeItem, 0)
	err = DataBase().WithContext(ctx).Model(&LikeItem{}).
		Where("story_id in (?) and user_id = ?", storyIds, userId).
		Where("deleted = ?", 0).
		Where("like_item_type = ?", LikeItemTypeStory).
		Scan(&list).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return nil, err
	}
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	return list, nil
}

func GetLikeItemByGroup(ctx context.Context, groupId []int64, userId int) (list []*LikeItem, err error) {
	list = make([]*LikeItem, 0)
	err = DataBase().WithContext(ctx).Model(&LikeItem{}).
		Where("group_id in (?) and user_id = ?", groupId, userId).
		Where("deleted = ?", 0).
		Where("like_item_type = ?", LikeItemTypeGroup).
		Scan(&list).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return nil, err
	}
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	return list, nil
}

// 根据一组角色id，以及一个用户id来获取喜欢的列表
func GetLikeItemByRolesAndUser(ctx context.Context, roleIds []int64, userId int) (list []*LikeItem, err error) {
	list = make([]*LikeItem, 0)
	err = DataBase().WithContext(ctx).Model(&LikeItem{}).
		Where("role_id in (?) and user_id = ?", roleIds, userId).
		Where("deleted = ?", 0).
		Where("like_item_type = ?", LikeItemTypeRole).
		Scan(&list).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return nil, err
	}
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	return list, nil
}

// // 根据一组故事板id，以及一个用户id来获取喜欢的列表
func GetLikeItemByStoryBoardsAndUser(ctx context.Context, storyboardIds []int64, userId int) (list []*LikeItem, err error) {
	list = make([]*LikeItem, 0)
	err = DataBase().WithContext(ctx).Model(&LikeItem{}).
		Where("storyboard_id in (?) and user_id = ?", storyboardIds, userId).
		Where("deleted = ?", 0).
		Where("like_item_type = ?", LikeItemTypeStoryboard).
		Scan(&list).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return nil, err
	}
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	return list, nil
}

func GetLikeItemByStoryBoard(ctx context.Context, storyId int64, storyboardId int64) (list []*LikeItem, err error) {
	list = make([]*LikeItem, 0)
	err = DataBase().WithContext(ctx).Model(&LikeItem{}).
		Where("story_id = ? and storyboard_id = ?", storyId, storyboardId).
		Scan(&list).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return nil, err
	}
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	return list, nil
}

func GetLikeItemByStoryBoardAndUser(ctx context.Context, storyboardId int64, userId int) (*LikeItem, error) {
	item := new(LikeItem)
	err := DataBase().WithContext(ctx).Model(&LikeItem{}).
		Where("storyboard_id = ? and user_id = ?", storyboardId, userId).
		First(item).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return nil, err
	}
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	return item, nil
}

func GetLikeItemByStoryRoleAndUser(ctx context.Context, roleId int64, userId int) (*LikeItem, error) {
	item := new(LikeItem)
	err := DataBase().WithContext(ctx).Model(&LikeItem{}).
		Where("role_id = ? and user_id = ?", roleId, userId).
		First(item).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return nil, err
	}
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	return item, nil
}

func LikeStoryRole(ctx context.Context, userId int, storyId int64, roleId int64) error {
	err := DataBase().WithContext(ctx).Model(&LikeItem{}).Create(&LikeItem{
		UserID:       int64(userId),
		StoryID:      storyId,
		RoleID:       roleId,
		LikeType:     LikeTypeLike,
		LikeItemType: LikeItemTypeRole,
	}).Error
	return err
}

func UnLikeStoryRole(ctx context.Context, userId int, storyId int64, roleId int64) error {
	err := DataBase().WithContext(ctx).Model(&LikeItem{}).
		Where("user_id = ? and story_id = ? and role_id = ?",
			userId, storyId, roleId).
		Update("deleted = ?", 1).Error
	return err
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
	WatchItemTypeStoryRole
)

type WatchItem struct {
	IDBase
	UserID        int64         `json:"user_id,omitempty"`
	GroupID       int64         `json:"group_id,omitempty"`
	TimelineID    int64         `json:"timeline_id,omitempty"`
	StoryID       int64         `json:"story_id,omitempty"`
	RoleID        int64         `json:"role_id,omitempty"`
	WatchType     WatchType     `json:"watch_type,omitempty"`
	WatchItemType WatchItemType `json:"watch_item_type,omitempty"`
}

func (w WatchItem) TableName() string {
	return "watch_item"
}

func GetWatchItemByGroup(ctx context.Context, groupId int64) (list []*WatchItem, err error) {
	list = make([]*WatchItem, 0)
	err = DataBase().WithContext(ctx).Model(&WatchItem{}).
		Where("group_id = ?", groupId).
		Scan(&list).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return nil, err
	}
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	return list, nil
}

func GetWatchItemByStory(ctx context.Context, storyId int64) (list []*WatchItem, err error) {
	list = make([]*WatchItem, 0)
	err = DataBase().WithContext(ctx).Model(&WatchItem{}).
		Where("story_id = ?", storyId).
		Scan(&list).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return nil, err
	}
	if err == gorm.ErrRecordNotFound {
		return nil, nil
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

func CreateWatchStoryItem(ctx context.Context, userId int, storyId int64, groupId int64) error {
	var num int64
	err := DataBase().WithContext(ctx).Model(&WatchItem{}).
		Where("user_id = ? and story_id = ?",
			userId, storyId).
		Count(&num).Error
	if err != nil {
		log.Error("create WatchItem failed: ", err)
		return err
	}
	if num > 0 {
		log.Info("WatchItem already exist")
		return nil
	}
	err = DataBase().WithContext(ctx).Model(&WatchItem{}).Create(&WatchItem{
		UserID:        int64(userId),
		StoryID:       storyId,
		GroupID:       groupId,
		WatchType:     WatchTypeIsWatch,
		WatchItemType: WatchItemTypeStory,
	}).Error
	if err != nil {
		log.Errorf("create WatchItem failed: %s", err.Error())
		return err
	}
	return nil
}

func UnWatchStoryItem(ctx context.Context, userId int, storyId int64) error {
	err := DataBase().WithContext(ctx).Model(&WatchItem{}).
		Where("user_id = ? and story_id = ?",
			userId, storyId).
		Update("deleted = ?", 1).Error
	return err
}

func CreateWatchGroupItem(ctx context.Context, userId int, groupId int64) error {
	var num int64
	err := DataBase().WithContext(ctx).Model(&WatchItem{}).
		Where("user_id = ? and group_id = ?",
			userId, groupId).
		Count(&num).Error
	if err != nil {
		log.Error("create WatchItem failed: ", err)
		return err
	}
	if num > 0 {
		log.Info("WatchItem already exist")
		return nil
	}
	err = DataBase().WithContext(ctx).Model(&WatchItem{}).Create(&WatchItem{
		UserID:        int64(userId),
		GroupID:       groupId,
		WatchType:     WatchTypeIsWatch,
		WatchItemType: WatchItemTypeGroup,
	}).Error
	if err != nil {
		log.Errorf("create WatchItem failed: %s", err.Error())
		return err
	}
	return nil
}

func UnWatchGroupItem(ctx context.Context, userId int, groupId int64) error {
	err := DataBase().WithContext(ctx).Model(&WatchItem{}).
		Where("user_id = ? and group_id = ?",
			userId, groupId).
		Update("deleted = ?", 1).Error
	return err
}

func CreateWatchRoleItem(ctx context.Context, userId int, storyId int64, roleId int64, groupId int64) error {
	var num int64
	err := DataBase().WithContext(ctx).Model(&WatchItem{}).
		Where("user_id = ? and role_id = ?",
			userId, roleId).
		Count(&num).Error
	if err != nil {
		log.Error("create WatchItem failed: ", err)
		return err
	}
	if num > 0 {
		log.Info("WatchItem already exist")
		return nil
	}
	err = DataBase().WithContext(ctx).Model(&WatchItem{}).Create(&WatchItem{
		UserID:        int64(userId),
		GroupID:       groupId,
		StoryID:       storyId,
		RoleID:        roleId,
		WatchType:     WatchTypeIsWatch,
		WatchItemType: WatchItemTypeStoryRole,
	}).Error
	if err != nil {
		log.Errorf("create WatchItem failed: %s", err.Error())
		return err
	}
	return nil
}

func WatchStoryRole(ctx context.Context, userId int, storyId int64, roleId int64, groupId int64) error {
	err := DataBase().WithContext(ctx).Model(&WatchItem{}).Create(&WatchItem{
		UserID:        int64(userId),
		GroupID:       groupId,
		StoryID:       storyId,
		RoleID:        roleId,
		WatchType:     WatchTypeIsWatch,
		WatchItemType: WatchItemTypeStoryRole,
	}).Error
	return err
}

func UnWatchStoryRole(ctx context.Context, userId int, storyId int64, roleId int64, groupId int64) error {
	err := DataBase().WithContext(ctx).Model(&WatchItem{}).
		Where("user_id = ? and story_id = ? and role_id = ? and group_id = ?",
			userId, storyId, roleId, groupId).
		Update("deleted = ?", 1).Error
	return err
}

// 根据一组故事id，以及一个用户id来获取关注的列表
func GetWatchItemByStoriesAndUser(ctx context.Context, storyIds []int64, userId int) (list []*WatchItem, err error) {
	list = make([]*WatchItem, 0)
	err = DataBase().WithContext(ctx).Model(&WatchItem{}).
		Where("story_id in (?) and user_id = ?", storyIds, userId).
		Where("deleted = ?", 0).
		Where("watch_item_type = ?", WatchItemTypeStory).
		Scan(&list).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return nil, err
	}
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	return list, nil
}

// 根据一组故事id，以及一个用户id来获取关注的列表
func GetWatchItemByStoryAndUser(ctx context.Context, storyId int64, userId int) (*WatchItem, error) {
	item := new(WatchItem)
	err := DataBase().WithContext(ctx).Model(&WatchItem{}).
		Where("story_id = ? and user_id = ?", storyId, userId).
		Where("deleted = ?", 0).
		Where("watch_item_type = ?", WatchItemTypeStory).
		First(item).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return nil, err
	}
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	return item, nil
}

func GetWatchItemByStoryboardAndUser(ctx context.Context, storyboardId int64, userId int) (*WatchItem, error) {
	item := new(WatchItem)
	err := DataBase().WithContext(ctx).Model(&WatchItem{}).
		Where("storyboard_id = ? and user_id = ?", storyboardId, userId).
		Where("deleted = ?", 0).
		Where("watch_item_type = ?", WatchItemTypeStoryboard).
		First(item).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return nil, err
	}
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	return item, nil
}

func GetWatchItemByStoryRoleAndUser(ctx context.Context, roleId int64, userId int64) (*WatchItem, error) {
	item := new(WatchItem)
	err := DataBase().WithContext(ctx).Model(&WatchItem{}).
		Where("role_id = ? and user_id = ?", roleId, userId).
		Where("deleted = ?", 0).
		Where("watch_item_type = ?", WatchItemTypeStoryRole).
		First(item).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return nil, err
	}
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	return item, nil
}

func GetWatchItemByGroupAndUser(ctx context.Context, groupId int64, userId int64) (*WatchItem, error) {
	item := new(WatchItem)
	err := DataBase().WithContext(ctx).Model(&WatchItem{}).
		Where("group_id = ? and user_id = ?", groupId, userId).
		Where("deleted = ?", 0).
		Where("watch_item_type = ?", WatchItemTypeGroup).
		First(item).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return nil, err
	}
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	return item, nil
}

func GetWatchItemByTargetUserAndUser(ctx context.Context, targetUserId int64, userId int64) (*WatchItem, error) {
	item := new(WatchItem)
	err := DataBase().WithContext(ctx).Model(&WatchItem{}).
		Where("user_id = ? and target_user_id = ?", userId, targetUserId).
		Where("deleted = ?", 0).
		Where("watch_item_type = ?", WatchItemTypeUser).
		First(item).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return nil, err
	}
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	return item, nil
}

// 根据一组角色id，以及一个用户id来获取关注的列表
func GetWatchItemByRolesAndUser(ctx context.Context, roleIds []int64, userId int) (list []*WatchItem, err error) {
	list = make([]*WatchItem, 0)
	err = DataBase().WithContext(ctx).Model(&WatchItem{}).
		Where("role_id in (?) and user_id = ?", roleIds, userId).
		Where("deleted = ?", 0).
		Where("watch_item_type = ?", WatchItemTypeStoryRole).
		Scan(&list).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return nil, err
	}
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	return list, nil
}

// 根据一组小组id，以及一个用户id来获取喜欢的列表
func GetWatchItemByGroupsAndUser(ctx context.Context, groupIds []int64, userId int) (list []*WatchItem, err error) {
	list = make([]*WatchItem, 0)
	err = DataBase().WithContext(ctx).Model(&WatchItem{}).
		Where("group_id in (?) and user_id = ?", groupIds, userId).
		Where("deleted = ?", 0).
		Where("watch_item_type = ?", WatchItemTypeGroup).
		Scan(&list).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return nil, err
	}
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	return list, nil
}

func GetStoriesIdByUserFollow(ctx context.Context, userId int64) ([]int64, error) {
	var storiesIds []int64
	if err := DataBase().Model(&WatchItem{}).
		Where("user_id = ?", userId).
		Where("watch_item_type = ?", WatchItemTypeStory).
		Where("watch_type = ?", WatchTypeIsWatch).
		Where("deleted = ?", 0).
		Pluck("story_id", &storiesIds).Error; err != nil {
		return nil, err
	} else {
		if err != nil && err != gorm.ErrRecordNotFound {
			return nil, err
		}
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
	}

	return storiesIds, nil
}

func GetStoryBoardRolesIDByUserFollow(ctx context.Context, userId int64) ([]int64, error) {
	var rolesIds []int64
	if err := DataBase().Model(&WatchItem{}).
		Where("user_id = ?", userId).
		Where("watch_item_type = ?", WatchItemTypeStoryRole).
		Where("watch_type = ?", WatchTypeIsWatch).
		Where("deleted = ?", 0).
		Pluck("role_id", &rolesIds).Error; err != nil {
		return nil, err
	} else {
		if err != nil && err != gorm.ErrRecordNotFound {
			return nil, err
		}
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
	}
	return rolesIds, nil
}
