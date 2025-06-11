package models

import (
	"context"

	log "github.com/sirupsen/logrus"

	api "github.com/grapery/common-protoc/gen"
	"gorm.io/gorm"
)

/*
内容承载的item:
图片，文字,视频，音乐
*/
// StoryItem 内容承载的item（图片、文字、视频、音乐等）
type StoryItem struct {
	IDBase
	ProjectID     int64         `gorm:"column:project_id" json:"project_id,omitempty"`           // 项目ID
	UserID        int64         `gorm:"column:user_id" json:"user_id,omitempty"`                 // 用户ID
	Visable       api.ScopeType `gorm:"column:visable" json:"visable,omitempty"`                 // 可见性
	Title         string        `gorm:"column:title" json:"title,omitempty"`                     // 标题
	Description   string        `gorm:"column:description" json:"description,omitempty"`         // 描述
	ItemType      api.ItemType  `gorm:"column:item_type" json:"item_type,omitempty"`             // 类型
	Content       string        `gorm:"column:content" json:"content,omitempty"`                 // 内容
	Url           string        `gorm:"column:url" json:"url,omitempty"`                         // 资源URL
	Size          string        `gorm:"column:size" json:"size,omitempty"`                       // 大小
	PrevId        int64         `gorm:"column:prev_id" json:"prev_id,omitempty"`                 // 上一项ID
	NextId        int64         `gorm:"column:next_id" json:"next_id,omitempty"`                 // 下一项ID
	Token         string        `gorm:"column:token" json:"token,omitempty"`                     // token
	IsHiddenToken bool          `gorm:"column:is_hidden_token" json:"is_hidden_token,omitempty"` // 是否隐藏token
	Tags          string        `gorm:"column:tags" json:"tags,omitempty"`                       // 标签
	LikeCount     int64         `gorm:"column:like_count" json:"like_count,omitempty"`           // 点赞数
	IsAIGenerate  bool          `gorm:"column:is_ai_generate" json:"is_ai_generate,omitempty"`   // 是否AI生成
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

// 新增：分页获取StoryItem列表
func GetStoryItemList(ctx context.Context, offset, limit int) ([]*StoryItem, error) {
	var items []*StoryItem
	err := DataBase().Model(&StoryItem{}).
		WithContext(ctx).
		Offset(offset).
		Limit(limit).
		Order("create_at desc").
		Find(&items).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return nil, err
	}
	return items, nil
}

// 新增：通过主键唯一查询
func GetStoryItemByID(ctx context.Context, id int64) (*StoryItem, error) {
	item := &StoryItem{}
	err := DataBase().Model(item).
		WithContext(ctx).
		Where("id = ?", id).
		First(item).Error
	if err != nil {
		return nil, err
	}
	return item, nil
}
