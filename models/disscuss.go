package models

import (
	"context"
	"fmt"
	"time"

	log "github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type DiscussStatus int

const (
	DiscussStatusClosed DiscussStatus = iota + 1
	DiscussStatusOpen
	DiscussStatusPending
	DiscussStatusArchived
)

// DiscussCategory 讨论分类
type DiscussCategory int

const (
	DiscussCategoryGeneral       DiscussCategory = iota + 1 // 一般讨论
	DiscussCategoryQnA                                      // 问答
	DiscussCategoryIdeas                                    // 想法
	DiscussCategoryAnnouncements                            // 公告
	DiscussCategoryShowAndTell                              // 展示
)

// Disscuss 讨论组/评论区
type Disscuss struct {
	ID uint `gorm:"primary_key,column:id" json:"id,omitempty"`
	IDBase
	Creator     int64           `gorm:"column:creator" json:"creator,omitempty"`             // 创建者ID
	StoryID     int64           `gorm:"column:story_id" json:"story_id,omitempty"`           // 故事ID
	GroupID     int64           `gorm:"column:group_id" json:"group_id,omitempty"`           // 群组ID
	Title       string          `gorm:"column:title" json:"title,omitempty"`                 // 标题
	Content     string          `gorm:"column:content;type:text" json:"content,omitempty"`   // 讨论内容
	Category    DiscussCategory `gorm:"column:category" json:"category,omitempty"`           // 讨论分类
	Status      DiscussStatus   `gorm:"column:status" json:"status,omitempty"`               // 状态
	IsPinned    bool            `gorm:"column:is_pinned" json:"is_pinned,omitempty"`         // 是否置顶
	IsLocked    bool            `gorm:"column:is_locked" json:"is_locked,omitempty"`         // 是否锁定
	ViewCount   int64           `gorm:"column:view_count" json:"view_count,omitempty"`       // 查看次数
	LikeCount   int64           `gorm:"column:like_count" json:"like_count,omitempty"`       // 点赞数
	ReplyCount  int64           `gorm:"column:reply_count" json:"reply_count,omitempty"`     // 回复数
	LastReplyAt *time.Time      `gorm:"column:last_reply_at" json:"last_reply_at,omitempty"` // 最后回复时间
	LastReplyBy int64           `gorm:"column:last_reply_by" json:"last_reply_by,omitempty"` // 最后回复者
	Tags        string          `gorm:"column:tags" json:"tags,omitempty"`                   // 标签，逗号分隔
	Attachments string          `gorm:"column:attachments" json:"attachments,omitempty"`     // 附件
}

func (d Disscuss) TableName() string {
	return "disscuss"
}

// Create 创建讨论
func (d *Disscuss) Create(ctx context.Context) error {
	if err := DataBase().Model(d).Create(d).Error; err != nil {
		log.Errorf("create new discuss [%d] failed : [%s]", d.ID, err.Error())
		return fmt.Errorf("create new discuss [%d] failed", d.ID)
	}
	return nil
}

// Update 更新讨论
func (d *Disscuss) Update(ctx context.Context) error {
	if err := DataBase().Model(d).Updates(d).Error; err != nil {
		log.Errorf("update discuss [%d] failed : [%s]", d.ID, err.Error())
		return fmt.Errorf("update discuss [%d] failed", d.ID)
	}
	return nil
}

// Delete 软删除讨论
func (d *Disscuss) Delete(ctx context.Context) error {
	if err := DataBase().Model(d).
		Update("deleted", 1).
		Where("id = ?", d.ID).Error; err != nil {
		log.Errorf("delete discuss [%d] failed", d.ID)
		return fmt.Errorf("delete discuss [%d] failed", d.ID)
	}
	return nil
}

// GetDiscussByID 通过ID获取讨论
func GetDiscussByID(ctx context.Context, id int64) (*Disscuss, error) {
	discuss := &Disscuss{}
	err := DataBase().Model(discuss).
		WithContext(ctx).
		Where("id = ? AND deleted = 0", id).
		First(discuss).Error
	if err != nil {
		return nil, err
	}
	return discuss, nil
}

// GetDiscussList 获取讨论列表
func GetDiscussList(ctx context.Context, storyID int64, category DiscussCategory, status DiscussStatus, offset, limit int) ([]*Disscuss, error) {
	var discusses []*Disscuss
	query := DataBase().Model(&Disscuss{}).WithContext(ctx).Where("deleted = 0")

	if storyID > 0 {
		query = query.Where("story_id = ?", storyID)
	}
	if category > 0 {
		query = query.Where("category = ?", category)
	}
	if status > 0 {
		query = query.Where("status = ?", status)
	}

	err := query.Order("is_pinned DESC, last_reply_at DESC, create_at DESC").
		Offset(offset).
		Limit(limit).
		Find(&discusses).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return nil, err
	}
	return discusses, nil
}

// GetDiscussByCreator 获取用户创建的讨论
func GetDiscussByCreator(ctx context.Context, creatorID int64, offset, limit int) ([]*Disscuss, error) {
	var discusses []*Disscuss
	err := DataBase().Model(&Disscuss{}).
		WithContext(ctx).
		Where("creator = ? AND deleted = 0", creatorID).
		Order("create_at DESC").
		Offset(offset).
		Limit(limit).
		Find(&discusses).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return nil, err
	}
	return discusses, nil
}

// IncrementViewCount 增加查看次数
func IncrementViewCount(ctx context.Context, discussID int64) error {
	return DataBase().Model(&Disscuss{}).
		WithContext(ctx).
		Where("id = ?", discussID).
		Update("view_count", gorm.Expr("view_count + 1")).Error
}

// IncrementLikeCount 增加点赞数
func IncrementLikeCount(ctx context.Context, discussID int64) error {
	return DataBase().Model(&Disscuss{}).
		WithContext(ctx).
		Where("id = ?", discussID).
		Update("like_count", gorm.Expr("like_count + 1")).Error
}

// DecrementLikeCount 减少点赞数
func DecrementLikeCount(ctx context.Context, discussID int64) error {
	return DataBase().Model(&Disscuss{}).
		WithContext(ctx).
		Where("id = ?", discussID).
		Update("like_count", gorm.Expr("like_count - 1")).Error
}

// UpdateLastReply 更新最后回复信息
func UpdateLastReply(ctx context.Context, discussID int64, userID int64) error {
	now := time.Now()
	return DataBase().Model(&Disscuss{}).
		WithContext(ctx).
		Where("id = ?", discussID).
		Updates(map[string]interface{}{
			"last_reply_at": &now,
			"last_reply_by": userID,
			"reply_count":   gorm.Expr("reply_count + 1"),
		}).Error
}

// PinDiscuss 置顶讨论
func PinDiscuss(ctx context.Context, discussID int64, pinned bool) error {
	return DataBase().Model(&Disscuss{}).
		WithContext(ctx).
		Where("id = ?", discussID).
		Update("is_pinned", pinned).Error
}

// LockDiscuss 锁定讨论
func LockDiscuss(ctx context.Context, discussID int64, locked bool) error {
	return DataBase().Model(&Disscuss{}).
		WithContext(ctx).
		Where("id = ?", discussID).
		Update("is_locked", locked).Error
}

// UpdateDiscussStatus 更新讨论状态
func UpdateDiscussStatus(ctx context.Context, discussID int64, status DiscussStatus) error {
	return DataBase().Model(&Disscuss{}).
		WithContext(ctx).
		Where("id = ?", discussID).
		Update("status", status).Error
}

// DiscussReply 讨论回复
type DiscussReply struct {
	ID uint `gorm:"primary_key,column:id" json:"id,omitempty"`
	IDBase
	DiscussID   int64  `gorm:"column:discuss_id" json:"discuss_id,omitempty"`     // 讨论ID
	UserID      int64  `gorm:"column:user_id" json:"user_id,omitempty"`           // 用户ID
	Content     string `gorm:"column:content;type:text" json:"content,omitempty"` // 回复内容
	ParentID    int64  `gorm:"column:parent_id" json:"parent_id,omitempty"`       // 父回复ID（用于嵌套回复）
	LikeCount   int64  `gorm:"column:like_count" json:"like_count,omitempty"`     // 点赞数
	IsSolution  bool   `gorm:"column:is_solution" json:"is_solution,omitempty"`   // 是否为解决方案
	Attachments string `gorm:"column:attachments" json:"attachments,omitempty"`   // 附件
}

func (dr DiscussReply) TableName() string {
	return "discuss_reply"
}

// Create 创建回复
func (dr *DiscussReply) Create(ctx context.Context) error {
	if err := DataBase().Model(dr).Create(dr).Error; err != nil {
		log.Errorf("create new discuss reply [%d] failed : [%s]", dr.ID, err.Error())
		return fmt.Errorf("create new discuss reply [%d] failed", dr.ID)
	}
	return nil
}

// Update 更新回复
func (dr *DiscussReply) Update(ctx context.Context) error {
	if err := DataBase().Model(dr).Updates(dr).Error; err != nil {
		log.Errorf("update discuss reply [%d] failed : [%s]", dr.ID, err.Error())
		return fmt.Errorf("update discuss reply [%d] failed", dr.ID)
	}
	return nil
}

// Delete 软删除回复
func (dr *DiscussReply) Delete(ctx context.Context) error {
	if err := DataBase().Model(dr).
		Update("deleted", 1).
		Where("id = ?", dr.ID).Error; err != nil {
		log.Errorf("delete discuss reply [%d] failed", dr.ID)
		return fmt.Errorf("delete discuss reply [%d] failed", dr.ID)
	}
	return nil
}

// GetDiscussReplies 获取讨论回复列表
func GetDiscussReplies(ctx context.Context, discussID int64, offset, limit int) ([]*DiscussReply, error) {
	var replies []*DiscussReply
	err := DataBase().Model(&DiscussReply{}).
		WithContext(ctx).
		Where("discuss_id = ? AND deleted = 0", discussID).
		Order("create_at ASC").
		Offset(offset).
		Limit(limit).
		Find(&replies).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return nil, err
	}
	return replies, nil
}

// IncrementReplyLikeCount 增加回复点赞数
func IncrementReplyLikeCount(ctx context.Context, replyID int64) error {
	return DataBase().Model(&DiscussReply{}).
		WithContext(ctx).
		Where("id = ?", replyID).
		Update("like_count", gorm.Expr("like_count + 1")).Error
}

// DecrementReplyLikeCount 减少回复点赞数
func DecrementReplyLikeCount(ctx context.Context, replyID int64) error {
	return DataBase().Model(&DiscussReply{}).
		WithContext(ctx).
		Where("id = ?", replyID).
		Update("like_count", gorm.Expr("like_count - 1")).Error
}

// SetAsSolution 设置为解决方案
func SetAsSolution(ctx context.Context, replyID int64, isSolution bool) error {
	return DataBase().Model(&DiscussReply{}).
		WithContext(ctx).
		Where("id = ?", replyID).
		Update("is_solution", isSolution).Error
}

// SearchDiscuss 搜索讨论
func SearchDiscuss(ctx context.Context, keyword string, offset, limit int) ([]*Disscuss, error) {
	var discusses []*Disscuss
	err := DataBase().Model(&Disscuss{}).
		WithContext(ctx).
		Where("deleted = 0 AND (title LIKE ? OR content LIKE ?)", "%"+keyword+"%", "%"+keyword+"%").
		Order("create_at DESC").
		Offset(offset).
		Limit(limit).
		Find(&discusses).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return nil, err
	}
	return discusses, nil
}
