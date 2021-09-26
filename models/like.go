package models

import (
	log "github.com/sirupsen/logrus"

	"github.com/grapery/grapery/utils/errors"
)

type LikeItem struct {
	IDBase
	UserID    uint64 `json:"user_id,omitempty"`
	GroupID   uint64 `json:"group_id,omitempty"`
	ProjectID uint64 `json:"project_id,omitempty"`
	ItemID    uint64 `json:"item_id,omitempty"`
	LikeType  uint64 `json:"like_type,omitempty"`
}

func (l LikeItem) TableName() string {
	return "like_item"
}

func CreateLikeItem(repo *Repository, item *LikeItem) error {
	var num int64
	err := repo.DB().Model(&LikeItem{}).Where("user_id = ? and group_id = ? and project_id = ? and item_id = ?",
		item.UserID, item.GroupID, item.ProjectID, item.ItemID).Count(&num).Error
	if err != nil {
		return err
	}
	if num > 0 {
		return errors.ErrLikeItemIsExist
	}
	err = repo.DB().Model(&LikeItem{}).Create(item).Error
	if err != nil {
		log.Error("create likeitem failed: %s", err.Error())
		return err
	}
	return nil
}

func DeleteLikeItem(repo *Repository, itemID uint64) error {
	err := repo.DB().Model(&LikeItem{}).
		Update("deleted= ? ", 1).
		Where("id = ?", itemID).Error
	if err != nil {
		log.Error("update like item failed: ", err)
		return err
	}
	return nil
}

func GetLikeItem(repo *Repository, itemID uint64) (*LikeItem, error) {
	item := new(LikeItem)
	err := repo.DB().Model(&LikeItem{}).First(item).
		Where("id = ?", itemID).Error
	if err != nil {
		return nil, err
	}
	return item, nil
}

func GetLiteItemByProjectAndUser(repo *Repository, projectID int, userID int) (list []*LikeItem, err error) {
	list = make([]*LikeItem, 0)
	err = repo.DB().Model(&LikeItem{}).
		Where("project_id = ? and user_id = ?", projectID, userID).
		Scan(&list).Error
	if err != nil {
		return nil, err
	}
	return list, nil
}

func GetLiteItemByUser(repo *Repository, userID int) (list []*LikeItem, err error) {
	list = make([]*LikeItem, 0)
	err = repo.DB().Model(&LikeItem{}).
		Where("user_id = ?", userID).
		Scan(&list).Error
	if err != nil {
		return nil, err
	}
	return list, nil
}
