package mysql

import (
	"context"
	"errors"
	"time"

	"github.com/grapestree/fgrapery/grapery/internal/domain"
	"gorm.io/gorm"
)

// CreateInvitationCode 创建邀请码
func (r *Repository) CreateInvitationCode(ctx context.Context, code *domain.InvitationCode) error {
	model := &InvitationCode{
		ID:          code.ID,
		Code:        code.Code,
		CreatedBy:   code.CreatedBy,
		IsActive:    code.IsActive,
		MaxUses:     code.MaxUses,
		CurrentUses: code.CurrentUses,
		Description: code.Description,
		CreatedAt:   time.Unix(code.CreatedAt, 0),
		UpdatedAt:   time.Unix(code.UpdatedAt, 0),
	}

	if code.ExpiresAt > 0 {
		model.ExpiresAt = time.Unix(code.ExpiresAt, 0)
	}

	return r.db.WithContext(ctx).Create(model).Error
}

// GetInvitationCodeByCode 根据邀请码获取邀请码信息
func (r *Repository) GetInvitationCodeByCode(ctx context.Context, code string) (*domain.InvitationCode, error) {
	var model InvitationCode
	if err := r.db.WithContext(ctx).
		Preload("Creator").
		Preload("User").
		Where("code = ?", code).
		First(&model).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return r.invitationCodeToDomain(&model), nil
}

// GetInvitationCodeByID 根据ID获取邀请码信息
func (r *Repository) GetInvitationCodeByID(ctx context.Context, id string) (*domain.InvitationCode, error) {
	var model InvitationCode
	if err := r.db.WithContext(ctx).
		Preload("Creator").
		Preload("User").
		Where("id = ?", id).
		First(&model).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return r.invitationCodeToDomain(&model), nil
}

// ListInvitationCodes 列出邀请码
func (r *Repository) ListInvitationCodes(ctx context.Context, createdBy string, limit, offset int) ([]*domain.InvitationCode, error) {
	var models []InvitationCode
	query := r.db.WithContext(ctx).Preload("Creator").Preload("User")

	if createdBy != "" {
		query = query.Where("created_by = ?", createdBy)
	}

	if limit > 0 {
		query = query.Limit(limit).Offset(offset)
	}

	if err := query.Order("created_at DESC").Find(&models).Error; err != nil {
		return nil, err
	}

	codes := make([]*domain.InvitationCode, len(models))
	for i, m := range models {
		codes[i] = r.invitationCodeToDomain(&m)
	}
	return codes, nil
}

// UpdateInvitationCode 更新邀请码
func (r *Repository) UpdateInvitationCode(ctx context.Context, code *domain.InvitationCode) error {
	model := &InvitationCode{
		ID:          code.ID,
		Code:        code.Code,
		CreatedBy:   code.CreatedBy,
		UsedBy:      code.UsedBy,
		IsActive:    code.IsActive,
		MaxUses:     code.MaxUses,
		CurrentUses: code.CurrentUses,
		Description: code.Description,
		UpdatedAt:   time.Unix(code.UpdatedAt, 0),
	}

	if code.UsedAt > 0 {
		model.UsedAt = time.Unix(code.UsedAt, 0)
	}

	if code.ExpiresAt > 0 {
		model.ExpiresAt = time.Unix(code.ExpiresAt, 0)
	}

	return r.db.WithContext(ctx).Model(&InvitationCode{}).
		Where("id = ?", code.ID).
		Updates(model).Error
}

// DeleteInvitationCode 删除邀请码（软删除）
func (r *Repository) DeleteInvitationCode(ctx context.Context, id string) error {
	result := r.db.WithContext(ctx).Delete(&InvitationCode{}, "id = ?", id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return domain.ErrNotFound
	}
	return nil
}

// UseInvitationCode 使用邀请码
func (r *Repository) UseInvitationCode(ctx context.Context, code string, userID string) error {
	// 获取邀请码
	invCode, err := r.GetInvitationCodeByCode(ctx, code)
	if err != nil {
		return err
	}

	// 验证邀请码
	if err := r.ValidateInvitationCode(ctx, code); err != nil {
		return err
	}

	// 更新使用信息
	now := time.Now().Unix()
	invCode.UsedBy = userID
	invCode.UsedAt = now
	invCode.CurrentUses++
	invCode.UpdatedAt = now

	return r.UpdateInvitationCode(ctx, invCode)
}

// ValidateInvitationCode 验证邀请码是否有效
func (r *Repository) ValidateInvitationCode(ctx context.Context, code string) error {
	invCode, err := r.GetInvitationCodeByCode(ctx, code)
	if err != nil {
		if err == domain.ErrNotFound {
			return errors.New("invitation code not found")
		}
		return err
	}

	// 检查是否启用
	if !invCode.IsActive {
		return errors.New("invitation code is disabled")
	}

	// 检查是否过期
	if invCode.ExpiresAt > 0 {
		now := time.Now().Unix()
		if now > invCode.ExpiresAt {
			return errors.New("invitation code has expired")
		}
	}

	// 检查使用次数
	if invCode.MaxUses > 0 && invCode.CurrentUses >= invCode.MaxUses {
		return errors.New("invitation code has reached maximum uses")
	}

	return nil
}

// invitationCodeToDomain 将数据库模型转换为domain模型
func (r *Repository) invitationCodeToDomain(m *InvitationCode) *domain.InvitationCode {
	code := &domain.InvitationCode{
		ID:          m.ID,
		Code:        m.Code,
		CreatedBy:   m.CreatedBy,
		UsedBy:      m.UsedBy,
		IsActive:    m.IsActive,
		MaxUses:     m.MaxUses,
		CurrentUses: m.CurrentUses,
		Description: m.Description,
		CreatedAt:   m.CreatedAt.Unix(),
		UpdatedAt:   m.UpdatedAt.Unix(),
	}

	if !m.UsedAt.IsZero() {
		code.UsedAt = m.UsedAt.Unix()
	}

	if !m.ExpiresAt.IsZero() {
		code.ExpiresAt = m.ExpiresAt.Unix()
	}

	if m.Creator.ID != "" {
		creator := r.userToDomain(m.Creator)
		code.Creator = &creator
	}

	if m.User.ID != "" {
		user := r.userToDomain(m.User)
		code.User = &user
	}

	return code
}

