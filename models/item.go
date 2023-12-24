package models

import (
	log "github.com/sirupsen/logrus"

	api "github.com/grapery/common-protoc/gen"
)

/*
内容承载的item:
图片，文字,视频，音乐
*/
type Item struct {
	IDBase
	ProjectID     int64         `json:"project_id,omitempty"`
	UserID        int64         `json:"user_id,omitempty"`
	Visable       api.ScopeType `json:"visable,omitempty"`
	Title         string        `json:"title,omitempty"`
	Description   string        `json:"description,omitempty"`
	ItemType      api.ItemType  `json:"item_type,omitempty"`
	Content       string        `json:"content,omitempty"`
	Url           string        `json:"url,omitempty"`
	Size          string        `json:"size,omitempty"`
	PrevId        int64         `json:"prev_id,omitempty"`
	NextId        int64         `json:"next_id,omitempty"`
	Token         string        `json:"token,omitempty"`
	IsHiddenToken bool          `json:"is_hidden_token,omitempty"`
	Tags          string        `json:"tags,omitempty"`
	LikeCount     int64         `json:"like_count,omitempty"`
}

func (i Item) TableName() string {
	return "items"
}

func CreateItem(repo *Repository, item *Item) error {
	err := repo.DB().Model(item).Create(item).Error
	if err != nil {
		log.Errorf("create item failed: %s", err.Error())
		return err
	}
	log.Info("create item : ", item.Title)
	return nil
}

func DeleteItem(repo *Repository, itemID int64) error {
	err := repo.DB().Model(&Item{}).Update("delete = ? ", true).
		Where("id = ?", itemID).Error
	if err != nil {
		log.Error("update item failed: ", err)
		return err
	}
	return nil
}

func GetItem(repo *Repository, itemID int64) (*Item, error) {
	item := new(Item)
	err := repo.DB().Model(item).
		Where("id = ?", itemID).
		First(item).Error
	if err != nil {
		return nil, err
	}
	return item, nil
}

func GetItemByTitle(repo *Repository, title string) (*Item, error) {
	item := new(Item)
	err := repo.DB().
		Model(item).
		Where("title = ?", title).
		First(item).Error
	if err != nil {
		return nil, err
	}
	return item, nil
}

func GetItemsByType(repo *Repository, itemType api.ItemType) ([]*Item, error) {
	items := new([]*Item)
	err := repo.DB().
		Model(&Item{}).
		Where("item_type = ?", itemType).
		First(items).Error
	if err != nil {
		return nil, err
	}
	return *items, nil
}

func GetItemByProject(repo *Repository, projectID int64, offset, number int) ([]*Item, error) {
	items := new([]*Item)
	err := repo.DB().Model(&Item{}).
		Where("project_id = ?", projectID).
		Offset(offset).
		Limit(number).
		Scan(items).Error
	if err != nil {
		return nil, err
	}
	return *items, nil
}

func GetItemByGroup(repo *Repository, grouId int64, offset, number int) ([]*Item, error) {
	items := new([]*Item)
	err := DataBase().Model(Item{}).
		Where("project_id in (?)",
			DataBase().
				Model(Project{}).
				Select("project_id").
				Where("group_id = ?", grouId)).
		Order("create_at").
		Offset(offset).Limit(number).Scan(items).Error
	if err != nil {
		return nil, err
	}
	return *items, nil
}

func GetItemByUser(repo *Repository, userId int64, offset, number int) ([]*Item, error) {
	items := new([]*Item)
	err := repo.DB().Model(&Item{}).
		Where("user_id = ?", userId).
		Order("create_at").
		Offset(offset).
		Limit(number).
		Scan(items).Error
	if err != nil {
		return nil, err
	}
	return *items, nil
}

func GetItemByProjectAndCreator(repo *Repository, projectID int64, userID int64, offset, number int) ([]*Item, error) {
	items := new([]*Item)
	err := repo.DB().Model(&Item{}).
		Where("project_id = ? and user_id = ?", projectID, userID).
		Order("create_at").
		Offset(offset).
		Limit(number).
		Scan(items).Error
	if err != nil {
		return nil, err
	}
	return *items, nil
}

func UpdateItemVisable(repo *Repository, itemID int64, vtype api.ScopeType) error {
	err := repo.DB().Model(&Item{}).Update("visable", vtype).
		Where("id = ? and deleted = ?", itemID, 0).Error
	if err != nil {
		return err
	}
	return nil
}

func UpdateItemTags(repo *Repository, itemID int64, tags string) error {
	err := repo.DB().Model(&Item{}).Update("tags", tags).
		Where("id = ? and deleted = ?", itemID, 0).Error
	if err != nil {
		return err
	}
	return nil
}

func UpdateItemTitle(repo *Repository, itemID int64, title string) error {
	err := repo.DB().Model(&Item{}).Update("title", title).
		Where("id = ? and deleted = ?", itemID, 0).Error
	if err != nil {
		return err
	}
	return nil
}

type ItemLiker struct {
	IDBase
	ItemID int64 `json:"item_id,omitempty"`
	UserID int64 `json:"user_id,omitempty"`
	Ltype  int64 `json:"ltype,omitempty"`
}

func CreateItemLiker(repo *Repository, projectId, itemId, userId int64) error {
	item := &ItemLiker{
		ItemID: itemId,
		UserID: userId,
	}
	err := repo.DB().Model(item).Create(item).Error
	if err != nil {
		log.Errorf("create item liker failed: %s", err.Error())
		return err
	}
	return nil
}

func DeleteItemLiker(repo *Repository, projectId, itemId, userId int64) error {
	item := &ItemLiker{
		ItemID: itemId,
		UserID: userId,
	}
	err := repo.DB().Model(item).Update("delete = ? ", true).
		Where("item_id = ? and user_id = ?", itemId, userId).Error
	if err != nil {
		log.Error("delete item liker failed: ", err)
		return err
	}
	return nil
}
