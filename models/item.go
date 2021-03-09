package models

import (
	log "github.com/sirupsen/logrus"

	api "github.com/grapery/grapery/api"
)

/*
内容承载的item:
图片，文字,视频，音乐
*/
type Item struct {
	IDBase
	GroupID     uint64          `json:"group_id,omitempty"`
	ProjectID   uint64          `json:"project_id,omitempty"`
	UserID      uint64          `json:"user_id,omitempty"`
	Visable     api.VisibleType `json:"visable,omitempty"`
	Title       string          `json:"title,omitempty"`
	Description string          `json:"description,omitempty"`
	ItemType    api.ItemType    `json:"item_type,omitempty"`
	Tags        string          `json:"tags,omitempty"`
}

func (it Item) TableName() string {
	return "item"
}

func CreateItem(repo *Repository, item *Item) error {
	err := repo.DB().Model(item).Create(item).Error
	if err != nil {
		log.Error("create item failed: %s", err.Error())
		return err
	}
	log.Info("create item : ", item.Title)
	return nil
}

func DeleteItem(repo *Repository, itemID uint64) error {
	err := repo.DB().Model(&Item{}).Update("delete = ? ", true).
		Where("id = ?", itemID).Error
	if err != nil {
		log.Error("update item failed: ", err)
		return err
	}
	return nil
}

func GetItem(repo *Repository, itemID uint64) (*Item, error) {
	item := new(Item)
	err := repo.DB().Model(item).First(item).
		Where("id = ?", itemID).Error
	if err != nil {
		return nil, err
	}
	return item, nil
}

func GetItemByTitle(repo *Repository, title string) (*Item, error) {
	item := new(Item)
	err := repo.DB().Model(item).First(item).
		Where("title = ?", title).Error
	if err != nil {
		return nil, err
	}
	return item, nil
}

func GetItemsByType(repo *Repository, itemType api.ItemType) ([]*Item, error) {
	items := new([]*Item)
	err := repo.DB().Model(&Item{}).First(items).
		Where("item_type = ?", itemType).Error
	if err != nil {
		return nil, err
	}
	return *items, nil
}

func GetItemByProject(repo *Repository, projectID uint64, offset, number int) ([]*Item, error) {
	items := new([]*Item)
	err := repo.DB().Model(&Item{}).Find(items).
		Where("project_id = ?", projectID).
		Offset(offset).Limit(number).Error
	if err != nil {
		return nil, err
	}
	return *items, nil
}

func GetItemByProjectAndCreator(repo *Repository, projectID uint64, userID uint64, offset, number int) ([]*Item, error) {
	items := new([]*Item)
	err := repo.DB().Model(&Item{}).Find(items).
		Where("project_id = ? and user_id = ?", projectID, userID).
		Offset(offset).Limit(number).Error
	if err != nil {
		return nil, err
	}
	return *items, nil
}

func UpdateItemVisable(repo *Repository, itemID uint64, vtype api.VisibleType) error {
	err := repo.DB().Model(&Item{}).Update("visable", vtype).
		Where("id = ? and deleted = ?", itemID, 0).Error
	if err != nil {
		return err
	}
	return nil
}

func UpdateItemTags(repo *Repository, itemID uint64, tags string) {
	err := repo.DB().Model(&Item{}).Update("tags", tags).
		Where("id = ? and deleted = ?", itemID, 0).Error
	if err != nil {
		return err
	}
	return nil
}

func UpdateItemTitle(repo *Repository, itemID uint64, title string) {
	err := repo.DB().Model(&Item{}).Update("title", title).
		Where("id = ? and deleted = ?", itemID, 0).Error
	if err != nil {
		return err
	}
	return nil
}
