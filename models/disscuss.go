package models

import (
	"context"

	"gorm.io/gorm"
)

type DiscussStatus int

const (
	DiscussStatusClosed DiscussStatus = iota + 1
	DiscussStatusOpen
	DiscussStatusPending
	DiscussStatusArchived
)

// Disscuss 讨论组/评论区
type Disscuss struct {
	IDBase
	Creator      int64  `gorm:"column:creator" json:"creator,omitempty"`             // 创建者ID
	StoryID      int64  `gorm:"column:story_id" json:"story_id,omitempty"`           // 故事ID
	GroupID      int64  `gorm:"column:group_id" json:"group_id,omitempty"`           // 群组ID
	Title        string `gorm:"column:title" json:"title,omitempty"`                 // 标题
	Status       int    `gorm:"column:status" json:"status,omitempty"`               // 状态
	Desc         string `gorm:"column:desc" json:"desc,omitempty"`                   // 描述
	TotalUser    int64  `gorm:"column:total_user" json:"total_user,omitempty"`       // 用户数
	TotalMessage int64  `gorm:"column:total_message" json:"total_message,omitempty"` // 消息数
}

func (d Disscuss) TableName() string {
	return "disscuss"
}

// 新增：分页获取Disscuss列表
func GetDisscussList(ctx context.Context, offset, limit int) ([]*Disscuss, error) {
	var dis []*Disscuss
	err := DataBase().Model(&Disscuss{}).
		WithContext(ctx).
		Offset(offset).
		Limit(limit).
		Order("create_at desc").
		Find(&dis).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return nil, err
	}
	return dis, nil
}

// 新增：通过主键唯一查询
func GetDisscussByID(ctx context.Context, id int64) (*Disscuss, error) {
	dis := &Disscuss{}
	err := DataBase().Model(dis).
		WithContext(ctx).
		Where("id = ?", id).
		First(dis).Error
	if err != nil {
		return nil, err
	}
	return dis, nil
}

func GetDisscussByCreator(creator string, pageSize, pageNum int) ([]*Disscuss, error) {
	result := make([]*Disscuss, 0)
	err := DataBase().Model(Disscuss{}).
		Where("creator = ?", creator).
		Offset(int(pageNum-1) * pageSize).
		Limit(pageSize).
		Scan(&result).
		Error
	if err != nil {
		return nil, err
	}
	return result, nil
}

func GetDisscussByStoryID(storyID int64, pageSize, pageNum int) ([]*Disscuss, error) {
	result := make([]*Disscuss, 0)
	err := DataBase().Model(Disscuss{}).
		Where("story_id = ?", storyID).
		Offset(int(pageNum-1) * pageSize).
		Limit(pageSize).
		Scan(&result).
		Error
	if err != nil {
		return nil, err
	}
	return result, nil
}

func SearchDisscuss(keyword string, pageSize, pageNum int) ([]*Disscuss, error) {
	result := make([]*Disscuss, 0)
	err := DataBase().Model(Disscuss{}).
		Where("title like ?", "%"+keyword+"%").
		Offset(int(pageNum-1) * pageSize).
		Limit(pageSize).
		Scan(&result).
		Error
	if err != nil {
		return nil, err
	}
	return result, nil
}
