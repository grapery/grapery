package models

import (
	"context"
	"time"

	"gorm.io/gorm"
)

// PasswordResetStatus 密码重置状态
// 0-待处理，1-成功，2-失败，3-已过期
type PasswordResetStatus int

const (
	PasswordResetStatusPending PasswordResetStatus = iota // 0: 待处理
	PasswordResetStatusSuccess                            // 1: 成功
	PasswordResetStatusFailed                             // 2: 失败
	PasswordResetStatusExpired                            // 3: 已过期
)

// PasswordReset 对应表 `password_resets`
// 参考 sql/auth_tables.sql 的建表定义
type PasswordReset struct {
	ID uint `gorm:"primary_key,column:id" json:"id,omitempty"`
	IDBase

	UserID      int64               `gorm:"column:user_id;index:idx_user_id" json:"user_id"`                           // 用户ID
	Email       string              `gorm:"column:email;size:255;index:idx_email" json:"email"`                        // 邮箱
	Phone       string              `gorm:"column:phone;size:20;index:idx_phone" json:"phone"`                         // 手机号
	ResetToken  string              `gorm:"column:reset_token;size:255;uniqueIndex:uk_reset_token" json:"reset_token"` // 重置令牌
	OldPassword string              `gorm:"column:old_password;size:255" json:"old_password"`                          // 旧密码
	NewPassword string              `gorm:"column:new_password;size:255" json:"new_password"`                          // 新密码
	Status      PasswordResetStatus `gorm:"column:status;index:idx_status" json:"status"`                              // 状态
	ExpireTime  time.Time           `gorm:"column:expire_time;index:idx_expire_time" json:"expire_time"`               // 过期时间
	IPAddress   string              `gorm:"column:ip_address;size:45" json:"ip_address"`                               // IP地址
}

// TableName 指定表名
func (p PasswordReset) TableName() string { return "password_resets" }

// CreatePasswordReset 创建密码重置记录
func CreatePasswordReset(ctx context.Context, pr *PasswordReset) error {
	return DataBase().WithContext(ctx).Create(pr).Error
}

// GetPasswordReset 根据ID获取记录
func GetPasswordReset(ctx context.Context, id uint) (*PasswordReset, error) {
	var pr PasswordReset
	if err := DataBase().WithContext(ctx).Where("id = ? and deleted = ?", id, 0).First(&pr).Error; err != nil {
		return nil, err
	}
	return &pr, nil
}

// GetPasswordResetByToken 根据重置令牌获取记录（唯一）
func GetPasswordResetByToken(ctx context.Context, token string) (*PasswordReset, error) {
	var pr PasswordReset
	if err := DataBase().WithContext(ctx).Where("reset_token = ? and deleted = ?", token, 0).First(&pr).Error; err != nil {
		return nil, err
	}
	return &pr, nil
}

// GetPasswordResetsByUserID 根据用户ID获取其重置记录（分页）
func GetPasswordResetsByUserID(ctx context.Context, userID int64, offset, limit int) ([]*PasswordReset, error) {
	var list []*PasswordReset
	if err := DataBase().WithContext(ctx).
		Where("user_id = ? and deleted = ?", userID, 0).
		Order("create_at desc").
		Offset(offset).Limit(limit).
		Find(&list).Error; err != nil {
		return nil, err
	}
	return list, nil
}

// GetPasswordResetsByStatus 根据状态获取记录（分页）
func GetPasswordResetsByStatus(ctx context.Context, status PasswordResetStatus, offset, limit int) ([]*PasswordReset, error) {
	var list []*PasswordReset
	if err := DataBase().WithContext(ctx).
		Where("status = ? and deleted = ?", status, 0).
		Order("create_at desc").
		Offset(offset).Limit(limit).
		Find(&list).Error; err != nil {
		return nil, err
	}
	return list, nil
}

// GetPasswordResetsByEmail 根据邮箱获取记录（分页）
func GetPasswordResetsByEmail(ctx context.Context, email string, offset, limit int) ([]*PasswordReset, error) {
	var list []*PasswordReset
	if err := DataBase().WithContext(ctx).
		Where("email = ? and deleted = ?", email, 0).
		Order("create_at desc").
		Offset(offset).Limit(limit).
		Find(&list).Error; err != nil {
		return nil, err
	}
	return list, nil
}

// GetPasswordResetsByPhone 根据手机号获取记录（分页）
func GetPasswordResetsByPhone(ctx context.Context, phone string, offset, limit int) ([]*PasswordReset, error) {
	var list []*PasswordReset
	if err := DataBase().WithContext(ctx).
		Where("phone = ? and deleted = ?", phone, 0).
		Order("create_at desc").
		Offset(offset).Limit(limit).
		Find(&list).Error; err != nil {
		return nil, err
	}
	return list, nil
}

// ListPasswordResets 列表（分页）
func ListPasswordResets(ctx context.Context, offset, limit int) ([]*PasswordReset, error) {
	var list []*PasswordReset
	if err := DataBase().WithContext(ctx).
		Where("deleted = ?", 0).
		Order("create_at desc").
		Offset(offset).Limit(limit).
		Find(&list).Error; err != nil {
		return nil, err
	}
	return list, nil
}

// UpdatePasswordReset 更新（仅允许更新少量字段）
func UpdatePasswordReset(ctx context.Context, id uint, updates map[string]interface{}) error {
	if len(updates) == 0 {
		return nil
	}
	return DataBase().WithContext(ctx).
		Model(&PasswordReset{}).
		Where("id = ? and deleted = ?", id, 0).
		Updates(updates).Error
}

// UpdatePasswordResetStatus 更新状态
func UpdatePasswordResetStatus(ctx context.Context, id uint, status PasswordResetStatus) error {
	return DataBase().WithContext(ctx).
		Model(&PasswordReset{}).
		Where("id = ? and deleted = ?", id, 0).
		Update("status", status).Error
}

// MarkPasswordResetExpired 置为过期
func MarkPasswordResetExpired(ctx context.Context, id uint) error {
	return UpdatePasswordResetStatus(ctx, id, PasswordResetStatusExpired)
}

// SoftDeletePasswordReset 软删除（deleted=1）
func SoftDeletePasswordReset(ctx context.Context, id uint) error {
	return DataBase().WithContext(ctx).
		Model(&PasswordReset{}).
		Where("id = ?", id).
		Update("deleted", 1).Error
}

// CleanupExpiredPasswordResets 清理已过期记录（不删除，仅状态置为过期）
func CleanupExpiredPasswordResets(ctx context.Context) error {
	now := time.Now()
	return DataBase().WithContext(ctx).
		Model(&PasswordReset{}).
		Where("expire_time < ? and status = ? and deleted = ?", now, PasswordResetStatusPending, 0).
		Update("status", PasswordResetStatusExpired).Error
}

// IsExpired 是否已过期
func (p *PasswordReset) IsExpired() bool { return time.Now().After(p.ExpireTime) }

// IsPending 是否待处理
func (p *PasswordReset) IsPending() bool { return p.Status == PasswordResetStatusPending }

// BeforeUpdate 钩子以确保 update_at 自动更新（借助 gorm 的 autoUpdateTime 已处理，这里留空）
func (p *PasswordReset) BeforeUpdate(tx *gorm.DB) (err error) { return nil }
