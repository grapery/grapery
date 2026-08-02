package models

import (
	"context"
	"fmt"
	"time"

	log "github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// FeedbackType 反馈类型枚举
type FeedbackType int

const (
	FeedbackTypeInappropriate  FeedbackType = 1 // 不合理的内容
	FeedbackTypeOffensive      FeedbackType = 2 // 有冒犯的内容
	FeedbackTypeImmoral        FeedbackType = 3 // 有违公德的内容
	FeedbackTypeHarmfulToYouth FeedbackType = 4 // 有损害少年的内容
	FeedbackTypeIllegal        FeedbackType = 5 // 有违反法律的内容
	FeedbackTypeOther          FeedbackType = 6 // 其他
)

// FeedbackStatus 反馈状态枚举
type FeedbackStatus int

const (
	FeedbackStatusPending    FeedbackStatus = 1 // 待处理
	FeedbackStatusProcessing FeedbackStatus = 2 // 处理中
	FeedbackStatusResolved   FeedbackStatus = 3 // 已解决
	FeedbackStatusRejected   FeedbackStatus = 4 // 已拒绝
)

// UserFeedback 用户反馈模型
type UserFeedback struct {
	ID uint `gorm:"primary_key,column:id;autoIncrement" json:"id,omitempty"`
	IDBase
	UserID       int64          `gorm:"column:user_id;index" json:"user_id,omitempty"`              // 反馈用户ID
	FeedbackType FeedbackType   `gorm:"column:feedback_type" json:"feedback_type,omitempty"`        // 反馈类型
	Title        string         `gorm:"column:title;size:255" json:"title,omitempty"`               // 反馈标题
	Description  string         `gorm:"column:description;type:text" json:"description,omitempty"`  // 反馈描述
	Screenshots  string         `gorm:"column:screenshots;type:text" json:"screenshots,omitempty"`  // 截图URL列表，JSON格式存储
	ContactInfo  string         `gorm:"column:contact_info;size:500" json:"contact_info,omitempty"` // 联系方式
	Status       FeedbackStatus `gorm:"column:status" json:"status,omitempty"`                      // 处理状态
	AdminID      int64          `gorm:"column:admin_id" json:"admin_id,omitempty"`                  // 处理管理员ID
	AdminReply   string         `gorm:"column:admin_reply;type:text" json:"admin_reply,omitempty"`  // 管理员回复
	ProcessedAt  *time.Time     `gorm:"column:processed_at" json:"processed_at,omitempty"`          // 处理时间
	RelatedType  string         `gorm:"column:related_type;size:50" json:"related_type,omitempty"`  // 关联内容类型(story, comment, user等)
	RelatedID    int64          `gorm:"column:related_id" json:"related_id,omitempty"`              // 关联内容ID
	Priority     int            `gorm:"column:priority;default:1" json:"priority,omitempty"`        // 优先级 1-5
	Tags         string         `gorm:"column:tags;size:500" json:"tags,omitempty"`                 // 标签，逗号分隔
}

func (f UserFeedback) TableName() string {
	return "user_feedback"
}

// Create 创建反馈
func (f *UserFeedback) Create(ctx context.Context) error {
	err := DataBase().WithContext(ctx).Model(f).Create(f).First(f).Error
	if err != nil {
		log.Errorf("create user feedback [%d] failed [%s]", f.UserID, err.Error())
		return fmt.Errorf("create user feedback failed")
	}
	return nil
}

// Update 更新反馈
func (f *UserFeedback) Update(ctx context.Context) error {
	err := DataBase().WithContext(ctx).Model(f).Updates(f).Where("id = ?", f.ID).Error
	if err != nil {
		log.Errorf("update user feedback [%d] failed [%s]", f.ID, err.Error())
		return fmt.Errorf("update user feedback [%d] failed", f.ID)
	}
	return nil
}

// GetByID 根据ID获取反馈
func (f *UserFeedback) GetByID(ctx context.Context) error {
	err := DataBase().WithContext(ctx).Model(f).Where("id = ? and deleted = ?", f.ID, 0).First(f).Error
	if err == gorm.ErrRecordNotFound {
		log.Errorf("get user feedback [%d] info failed : [%s]", f.ID, err.Error())
		return nil
	}
	if err != nil {
		log.Errorf("get user feedback [%d] info failed : [%s]", f.ID, err.Error())
		return fmt.Errorf("get user feedback [%d] info failed", f.ID)
	}
	return nil
}

// Delete 删除反馈（软删除）
func (f *UserFeedback) Delete(ctx context.Context) error {
	err := DataBase().WithContext(ctx).Model(f).Update("deleted", 1).Where("id = ?", f.ID).Error
	if err != nil {
		log.Errorf("delete user feedback [%d] failed [%s]", f.ID, err.Error())
		return fmt.Errorf("delete user feedback [%d] failed", f.ID)
	}
	return nil
}

// UpdateStatus 更新反馈状态
func (f *UserFeedback) UpdateStatus(ctx context.Context, status FeedbackStatus, adminID int64, reply string) error {
	now := time.Now()
	f.Status = status
	f.AdminID = adminID
	f.AdminReply = reply
	f.ProcessedAt = &now

	err := DataBase().WithContext(ctx).Model(f).
		Updates(map[string]interface{}{
			"status":       status,
			"admin_id":     adminID,
			"admin_reply":  reply,
			"processed_at": now,
		}).
		Where("id = ?", f.ID).Error
	if err != nil {
		log.Errorf("update user feedback [%d] status failed [%s]", f.ID, err.Error())
		return fmt.Errorf("update user feedback [%d] status failed", f.ID)
	}
	return nil
}

// GetUserFeedbackList 获取用户的反馈列表
func GetUserFeedbackList(ctx context.Context, userID int64, offset, limit int) ([]*UserFeedback, int64, error) {
	var feedbacks []*UserFeedback
	var total int64

	// 获取总数
	err := DataBase().WithContext(ctx).Model(&UserFeedback{}).
		Where("user_id = ? and deleted = ?", userID, 0).
		Count(&total).Error
	if err != nil {
		log.Errorf("count user feedback failed [%s]", err.Error())
		return nil, 0, fmt.Errorf("count user feedback failed")
	}

	// 获取列表
	err = DataBase().WithContext(ctx).Model(&UserFeedback{}).
		Where("user_id = ? and deleted = ?", userID, 0).
		Order("create_at desc").
		Offset(offset).
		Limit(limit).
		Find(&feedbacks).Error
	if err != nil {
		log.Errorf("get user feedback list failed [%s]", err.Error())
		return nil, 0, fmt.Errorf("get user feedback list failed")
	}

	return feedbacks, total, nil
}

// GetAdminFeedbackList 获取管理员反馈列表（所有用户的反馈）
func GetAdminFeedbackList(ctx context.Context, status *FeedbackStatus, feedbackType *FeedbackType, offset, limit int) ([]*UserFeedback, int64, error) {
	var feedbacks []*UserFeedback
	var total int64

	query := DataBase().WithContext(ctx).Model(&UserFeedback{}).Where("deleted = ?", 0)

	// 状态过滤
	if status != nil {
		query = query.Where("status = ?", *status)
	}

	// 类型过滤
	if feedbackType != nil {
		query = query.Where("feedback_type = ?", *feedbackType)
	}

	// 获取总数
	err := query.Count(&total).Error
	if err != nil {
		log.Errorf("count admin feedback failed [%s]", err.Error())
		return nil, 0, fmt.Errorf("count admin feedback failed")
	}

	// 获取列表，按优先级和创建时间排序
	err = query.Order("priority desc, create_at desc").
		Offset(offset).
		Limit(limit).
		Find(&feedbacks).Error
	if err != nil {
		log.Errorf("get admin feedback list failed [%s]", err.Error())
		return nil, 0, fmt.Errorf("get admin feedback list failed")
	}

	return feedbacks, total, nil
}

// GetFeedbackStats 获取反馈统计信息
func GetFeedbackStats(ctx context.Context) (map[string]int64, error) {
	stats := make(map[string]int64)

	// 按状态统计
	var statusStats []struct {
		Status FeedbackStatus `json:"status"`
		Count  int64          `json:"count"`
	}

	err := DataBase().WithContext(ctx).Model(&UserFeedback{}).
		Select("status, count(*) as count").
		Where("deleted = ?", 0).
		Group("status").
		Scan(&statusStats).Error
	if err != nil {
		log.Errorf("get feedback status stats failed [%s]", err.Error())
		return nil, fmt.Errorf("get feedback status stats failed")
	}

	for _, stat := range statusStats {
		stats[fmt.Sprintf("status_%d", stat.Status)] = stat.Count
	}

	// 按类型统计
	var typeStats []struct {
		FeedbackType FeedbackType `json:"feedback_type"`
		Count        int64        `json:"count"`
	}

	err = DataBase().WithContext(ctx).Model(&UserFeedback{}).
		Select("feedback_type, count(*) as count").
		Where("deleted = ?", 0).
		Group("feedback_type").
		Scan(&typeStats).Error
	if err != nil {
		log.Errorf("get feedback type stats failed [%s]", err.Error())
		return nil, fmt.Errorf("get feedback type stats failed")
	}

	for _, stat := range typeStats {
		stats[fmt.Sprintf("type_%d", stat.FeedbackType)] = stat.Count
	}

	// 总数
	var total int64
	err = DataBase().WithContext(ctx).Model(&UserFeedback{}).
		Where("deleted = ?", 0).
		Count(&total).Error
	if err != nil {
		log.Errorf("get total feedback count failed [%s]", err.Error())
		return nil, fmt.Errorf("get total feedback count failed")
	}

	stats["total"] = total

	return stats, nil
}

// GetFeedbackByID 根据ID获取反馈详情
func GetFeedbackByID(ctx context.Context, id uint) (*UserFeedback, error) {
	var feedback UserFeedback
	err := DataBase().WithContext(ctx).Model(&UserFeedback{}).
		Where("id = ? and deleted = ?", id, 0).
		First(&feedback).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		log.Errorf("get feedback [%d] failed [%s]", id, err.Error())
		return nil, fmt.Errorf("get feedback [%d] failed", id)
	}
	return &feedback, nil
}
