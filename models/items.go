package models

import (
	"context"

	log "github.com/sirupsen/logrus"

	api "github.com/grapery/common-protoc/gen"
)

/*
内容承载的item:
图片，文字,视频，音乐
*/
type StoryItem struct {
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
	IsAIGenerate  bool          `json:"is_ai_generate,omitempty"`
}

func (i StoryItem) TableName() string {
	return "story_item"
}

func CreateStoryItem(ctx context.Context, item *StoryItem) error {
	err := DataBase().WithContext(ctx).Model(item).Create(item).Error
	if err != nil {
		log.Errorf("create item failed: %s", err.Error())
		return err
	}
	log.Info("create item : ", item.Title)
	return nil
}

func DeleteStoryItem(ctx context.Context, itemID int64) error {
	err := DataBase().WithContext(ctx).Model(&StoryItem{}).Update("delete = ? ", true).
		Where("id = ?", itemID).Error
	if err != nil {
		log.Error("update item failed: ", err)
		return err
	}
	return nil
}

func GetStoryItem(ctx context.Context, itemID int64) (*StoryItem, error) {
	item := new(StoryItem)
	err := DataBase().WithContext(ctx).Model(item).
		Where("id = ?", itemID).
		First(item).Error
	if err != nil {
		return nil, err
	}
	return item, nil
}

func GetStoryItemByTitle(ctx context.Context, title string) (*StoryItem, error) {
	item := new(StoryItem)
	err := DataBase().WithContext(ctx).
		Model(item).
		Where("title = ?", title).
		First(item).Error
	if err != nil {
		return nil, err
	}
	return item, nil
}

func GetStoryItemsByType(ctx context.Context, itemType api.ItemType) ([]*StoryItem, error) {
	items := new([]*StoryItem)
	err := DataBase().WithContext(ctx).
		Model(&StoryItem{}).
		Where("item_type = ?", itemType).
		First(items).Error
	if err != nil {
		return nil, err
	}
	return *items, nil
}

func GetStoryItemByProject(ctx context.Context, projectID int64, offset, number int) ([]*StoryItem, error) {
	items := new([]*StoryItem)
	err := DataBase().WithContext(ctx).Model(&StoryItem{}).
		Where("project_id = ?", projectID).
		Offset(offset).
		Limit(number).
		Scan(items).Error
	if err != nil {
		return nil, err
	}
	return *items, nil
}

func GetStoryItemByGroup(ctx context.Context, grouId int64, offset, number int) ([]*StoryItem, error) {
	items := new([]*StoryItem)
	err := DataBase().Model(StoryItem{}).
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

func GetStoryItemByUser(ctx context.Context, userId int64, offset, number int) ([]*StoryItem, error) {
	items := new([]*StoryItem)
	err := DataBase().WithContext(ctx).Model(&StoryItem{}).
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

func GetStoryItemByProjectAndCreator(ctx context.Context, projectID int64, userID int64, offset, number int) ([]*StoryItem, error) {
	items := new([]*StoryItem)
	err := DataBase().WithContext(ctx).Model(&StoryItem{}).
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

func UpdateStoryItemVisable(ctx context.Context, itemID int64, vtype api.ScopeType) error {
	err := DataBase().WithContext(ctx).Model(&StoryItem{}).Update("visable", vtype).
		Where("id = ? and deleted = ?", itemID, 0).Error
	if err != nil {
		return err
	}
	return nil
}

func UpdateStoryItemTags(ctx context.Context, itemID int64, tags string) error {
	err := DataBase().WithContext(ctx).Model(&StoryItem{}).Update("tags", tags).
		Where("id = ? and deleted = ?", itemID, 0).Error
	if err != nil {
		return err
	}
	return nil
}

func UpdateStoryItemTitle(ctx context.Context, itemID int64, title string) error {
	err := DataBase().WithContext(ctx).Model(&StoryItem{}).Update("title", title).
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

func CreateItemLiker(ctx context.Context, projectId, itemId, userId int64) error {
	item := &ItemLiker{
		ItemID: itemId,
		UserID: userId,
	}
	err := DataBase().WithContext(ctx).Model(item).Create(item).Error
	if err != nil {
		log.Errorf("create item liker failed: %s", err.Error())
		return err
	}
	return nil
}

func DeleteItemLiker(ctx context.Context, projectId, itemId, userId int64) error {
	item := &ItemLiker{
		ItemID: itemId,
		UserID: userId,
	}
	err := DataBase().WithContext(ctx).Model(item).Update("delete = ? ", true).
		Where("item_id = ? and user_id = ?", itemId, userId).Error
	if err != nil {
		log.Error("delete item liker failed: ", err)
		return err
	}
	return nil
}

type Timeline struct {
	IDBase
	Name        string `json:"name"`
	RootItemId  int64  `json:"root_item_id"`
	ForkItemId  int64  `json:"fork_item_id"`
	Creator     int64  `json:"creator"`
	Description string `json:"description"`
	ProjectId   int64  `json:"project_id"`
	Avatar      string `json:"avatar"`
	Status      int    `json:"status"`
}

func (timeline Timeline) TableName() string {
	return "timeline"
}
