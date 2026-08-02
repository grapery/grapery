package models

import (
	"context"
	"time"
)

// EmailVerificationStatus 邮箱验证状态
// 0-待验证，1-已验证，2-已过期
type EmailVerificationStatus int

const (
	EmailVerificationStatusPending  EmailVerificationStatus = iota // 0: 待验证
	EmailVerificationStatusVerified                                // 1: 已验证
	EmailVerificationStatusExpired                                 // 2: 已过期
)

// EmailVerification 对应表 `email_verifications`
type EmailVerification struct {
	ID uint `gorm:"primary_key,column:id" json:"id,omitempty"`
	IDBase

	UserID            int64                   `gorm:"column:user_id;index:idx_user_id" json:"user_id"`                                                // 用户ID
	Email             string                  `gorm:"column:email;size:255;index:idx_email" json:"email"`                                             // 邮箱
	VerificationToken string                  `gorm:"column:verification_token;size:255;uniqueIndex:uk_verification_token" json:"verification_token"` // 验证令牌
	Status            EmailVerificationStatus `gorm:"column:status;index:idx_status" json:"status"`                                                   // 状态
	ExpireTime        time.Time               `gorm:"column:expire_time;index:idx_expire_time" json:"expire_time"`                                    // 过期时间
	VerifiedAt        *time.Time              `gorm:"column:verified_at" json:"verified_at"`                                                          // 验证时间
}

func (e EmailVerification) TableName() string { return "email_verifications" }

// CreateEmailVerification 创建
func CreateEmailVerification(ctx context.Context, ev *EmailVerification) error {
	return DataBase().WithContext(ctx).Create(ev).Error
}

// GetEmailVerification 获取
func GetEmailVerification(ctx context.Context, id uint) (*EmailVerification, error) {
	var ev EmailVerification
	if err := DataBase().WithContext(ctx).Where("id = ? and deleted = ?", id, 0).First(&ev).Error; err != nil {
		return nil, err
	}
	return &ev, nil
}

// GetEmailVerificationByToken 根据令牌获取（唯一）
func GetEmailVerificationByToken(ctx context.Context, token string) (*EmailVerification, error) {
	var ev EmailVerification
	if err := DataBase().WithContext(ctx).Where("verification_token = ? and deleted = ?", token, 0).First(&ev).Error; err != nil {
		return nil, err
	}
	return &ev, nil
}

// GetEmailVerificationsByUserID 根据用户ID获取（分页）
func GetEmailVerificationsByUserID(ctx context.Context, userID int64, offset, limit int) ([]*EmailVerification, error) {
	var list []*EmailVerification
	if err := DataBase().WithContext(ctx).
		Where("user_id = ? and deleted = ?", userID, 0).
		Order("create_at desc").
		Offset(offset).Limit(limit).
		Find(&list).Error; err != nil {
		return nil, err
	}
	return list, nil
}

// ListEmailVerifications 列表（分页）
func ListEmailVerifications(ctx context.Context, offset, limit int) ([]*EmailVerification, error) {
	var list []*EmailVerification
	if err := DataBase().WithContext(ctx).
		Where("deleted = ?", 0).
		Order("create_at desc").
		Offset(offset).Limit(limit).
		Find(&list).Error; err != nil {
		return nil, err
	}
	return list, nil
}

// GetEmailVerificationsByEmail 根据邮箱获取（分页）
func GetEmailVerificationsByEmail(ctx context.Context, email string, offset, limit int) ([]*EmailVerification, error) {
	var list []*EmailVerification
	if err := DataBase().WithContext(ctx).
		Where("email = ? and deleted = ?", email, 0).
		Order("create_at desc").
		Offset(offset).Limit(limit).
		Find(&list).Error; err != nil {
		return nil, err
	}
	return list, nil
}

// UpdateEmailVerification 更新
func UpdateEmailVerification(ctx context.Context, id uint, updates map[string]interface{}) error {
	if len(updates) == 0 {
		return nil
	}
	return DataBase().WithContext(ctx).
		Model(&EmailVerification{}).
		Where("id = ? and deleted = ?", id, 0).
		Updates(updates).Error
}

// VerifyEmail 将状态置为已验证并记录时间
func VerifyEmail(ctx context.Context, id uint) error {
	now := time.Now()
	return DataBase().WithContext(ctx).
		Model(&EmailVerification{}).
		Where("id = ? and deleted = ?", id, 0).
		Updates(map[string]interface{}{
			"status":      EmailVerificationStatusVerified,
			"verified_at": &now,
		}).Error
}

// ExpireEmailVerification 置为过期
func ExpireEmailVerification(ctx context.Context, id uint) error {
	return DataBase().WithContext(ctx).
		Model(&EmailVerification{}).
		Where("id = ? and deleted = ?", id, 0).
		Update("status", EmailVerificationStatusExpired).Error
}

// CleanupExpiredEmailVerifications 清理过期（仅置状态）
func CleanupExpiredEmailVerifications(ctx context.Context) error {
	now := time.Now()
	return DataBase().WithContext(ctx).
		Model(&EmailVerification{}).
		Where("expire_time < ? and status = ? and deleted = ?", now, EmailVerificationStatusPending, 0).
		Update("status", EmailVerificationStatusExpired).Error
}

// SoftDeleteEmailVerification 软删除
func SoftDeleteEmailVerification(ctx context.Context, id uint) error {
	return DataBase().WithContext(ctx).
		Model(&EmailVerification{}).
		Where("id = ?", id).
		Update("deleted", 1).Error
}

// IsExpired 是否过期
func (e *EmailVerification) IsExpired() bool { return time.Now().After(e.ExpireTime) }
