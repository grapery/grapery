package mysql

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/grapestree/fgrapery/grapery/internal/common"
	"github.com/grapestree/fgrapery/grapery/internal/domain"
	"gorm.io/gorm"
)

// GetOrCreateCharacterAnalytics 获取或创建角色分析数据
func (r *Repository) GetOrCreateCharacterAnalytics(ctx context.Context, characterID string) (*domain.CharacterAnalytics, error) {
	var analytics CharacterAnalytics
	err := r.db.WithContext(ctx).Where("character_id = ?", characterID).First(&analytics).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// 创建新的分析记录
			analytics = CharacterAnalytics{
				ID:                   uuid.New().String(),
				CharacterID:          characterID,
				UsersWhoChattedCount: 0,
				TotalMessagesSent:    0,
				TotalTokensConsumed:  0,
				UpdatedAt:            time.Now(),
			}
			if err := r.db.WithContext(ctx).Create(&analytics).Error; err != nil {
				return nil, err
			}
		} else {
			return nil, err
		}
	}

	return r.characterAnalyticsToDomain(&analytics), nil
}

// CharacterAnalyticsByCharacterID 根据角色 ID 获取分析数据
func (r *Repository) CharacterAnalyticsByCharacterID(ctx context.Context, characterID string) (*domain.CharacterAnalytics, error) {
	var analytics CharacterAnalytics
	if err := r.db.WithContext(ctx).
		Where("character_id = ?", characterID).
		First(&analytics).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	return r.characterAnalyticsToDomain(&analytics), nil
}

// UpdateCharacterAnalytics 更新角色分析数据
func (r *Repository) UpdateCharacterAnalytics(ctx context.Context, analytics *domain.CharacterAnalytics) error {
	updates := map[string]interface{}{
		"users_who_chatted_count": analytics.UsersWhoChattedCount,
		"total_messages_sent":     analytics.TotalMessagesSent,
		"total_tokens_consumed":   analytics.TotalTokensConsumed,
		"updated_at":              time.Now(),
	}

	return r.db.WithContext(ctx).
		Model(&CharacterAnalytics{}).
		Where("character_id = ?", analytics.CharacterID).
		Updates(updates).Error
}

// IncrementCharacterMessages 增加角色消息数
func (r *Repository) IncrementCharacterMessages(ctx context.Context, characterID string, count int) error {
	return r.db.WithContext(ctx).
		Model(&CharacterAnalytics{}).
		Where("character_id = ?", characterID).
		Updates(map[string]interface{}{
			"total_messages_sent": gorm.Expr("total_messages_sent + ?", count),
			"updated_at":          time.Now(),
		}).Error
}

// IncrementCharacterTokens 增加角色 token 消耗
func (r *Repository) IncrementCharacterTokens(ctx context.Context, characterID string, tokens int64) error {
	return r.db.WithContext(ctx).
		Model(&CharacterAnalytics{}).
		Where("character_id = ?", characterID).
		Updates(map[string]interface{}{
			"total_tokens_consumed": gorm.Expr("total_tokens_consumed + ?", tokens),
			"updated_at":            time.Now(),
		}).Error
}

// IncrementCharacterChatters 增加与角色聊天的用户数
func (r *Repository) IncrementCharacterChatters(ctx context.Context, characterID string) error {
	return r.db.WithContext(ctx).
		Model(&CharacterAnalytics{}).
		Where("character_id = ?", characterID).
		Updates(map[string]interface{}{
			"users_who_chatted_count": gorm.Expr("users_who_chatted_count + ?", 1),
			"updated_at":              time.Now(),
		}).Error
}

// characterAnalyticsToDomain 转换分析数据到 domain
func (r *Repository) characterAnalyticsToDomain(analytics *CharacterAnalytics) *domain.CharacterAnalytics {
	return &domain.CharacterAnalytics{
		BaseModel: common.BaseModel{
			ID:        analytics.ID,
			CreatedAt: analytics.UpdatedAt.Unix(), // Use UpdatedAt as CreatedAt since DB doesn't have CreatedAt
			UpdatedAt: analytics.UpdatedAt.Unix(),
		},
		CharacterID:          analytics.CharacterID,
		UsersWhoChattedCount: analytics.UsersWhoChattedCount,
		TotalMessagesSent:    analytics.TotalMessagesSent,
		TotalTokensConsumed:  analytics.TotalTokensConsumed,
	}
}
