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
	GroupID     uint64
	ProjectID   uint64
	UserID      uint64
	Visable     bool
	Title       string
	Description string
	ItemType    api.ItemType
}

func (it Item) TableName() string {
	return "item"
}

func CreateItem(repo *Repository, item *Item) error {
	log.Info("create item : ", item.Title)
	return nil
}

func DeleteItem(repo *Repository, itemID uint64) error {
	err := repo.DB().Update("delete = ? ", true).Where("id = ?", itemID).Error
	if err != nil {
		log.Error("update item failed: ", err)
		return err
	}
	return nil
}

func GetItem(repo *Repository, itemID uint64) (*Item, error) {
	item := new(Item)
	err := repo.DB().First(item).Where("id = ?", itemID).Error
	if err != nil {
		return nil, err
	}
	return item, nil
}

func GetItemByTitle(repo *Repository, title string) (*Item, error) {
	item := new(Item)
	err := repo.DB().First(item).Where("title = ?", title).Error
	if err != nil {
		return nil, err
	}
	return item, nil
}

func GetItemsByType(repo *Repository, itemType api.ItemType) ([]*Item, error) {
	items := new([]*Item)
	err := repo.DB().First(items).Where("item_type = ?", itemType).Error
	if err != nil {
		return nil, err
	}
	return *items, nil
}

func GetItemByProject(repo *Repository, projectID uint64) ([]*Item, error) {
	items := new([]*Item)
	err := repo.DB().Find(items).Where("project_id = ?", projectID).Error
	if err != nil {
		return nil, err
	}
	return *items, nil
}

func GetItemByProjectAndCreator(repo *Repository, projectID uint64, userID uint64) ([]*Item, error) {
	items := new([]*Item)
	err := repo.DB().Find(items).Where("project_id = ? and user_id = ?", projectID, userID).Error
	if err != nil {
		return nil, err
	}
	return *items, nil
}
