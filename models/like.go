package models

import (
	"context"

	log "github.com/sirupsen/logrus"
)

const ()

type LikeItem struct {
	IDBase
	UserID    int64 `json:"user_id,omitempty"`
	ItemID    int64 `json:"item_id,omitempty"`
	GroupID   int64 `json:"group_id,omitempty"`
	ProjectID int64 `json:"project_id,omitempty"`
	ActiveID  int64 `json:"active_id,omitempty"`
	LikeType  int64 `json:"like_type,omitempty"`
}

func (l LikeItem) TableName() string {
	return "like_item"
}

func CreateLikeItem(ctx context.Context, item *LikeItem) error {
	var num int64
	err := DataBase().WithContext(ctx).Model(&LikeItem{}).
		Where("user_id = ? and item_id = ?",
			item.UserID, item.ItemID).
		Count(&num).Error
	if err != nil {
		return err
	}
	if num > 0 {
		return nil
	}
	err = DataBase().WithContext(ctx).Model(&LikeItem{}).Create(item).Error
	if err != nil {
		log.Error("create likeitem failed: %s", err.Error())
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

func GetLikeItemByProjectAndUser(ctx context.Context, projectID int, userID int) (list []*LikeItem, err error) {
	list = make([]*LikeItem, 0)
	err = DataBase().WithContext(ctx).Model(&LikeItem{}).
		Where("project_id = ? and user_id = ?", projectID, userID).
		Scan(&list).Error
	if err != nil {
		return nil, err
	}
	return list, nil
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
