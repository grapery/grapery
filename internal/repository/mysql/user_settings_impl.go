package mysql

import (
	"errors"
	"fmt"

	"github.com/grapestree/fgrapery/grapery/internal/domain"
	"gorm.io/gorm"
)

// UserSettingsRepositoryImpl implements domain.UserSettingsRepository
type UserSettingsRepositoryImpl struct {
	db *gorm.DB
}

// NewUserSettingsRepository creates a new UserSettingsRepository instance
func NewUserSettingsRepository(db *gorm.DB) domain.UserSettingsRepository {
	return &UserSettingsRepositoryImpl{db: db}
}

// GetUserSettings gets user settings by user ID
func (r *UserSettingsRepositoryImpl) GetUserSettings(userID string) (*domain.UserSettings, error) {
	var model UserSettings
	if err := r.db.First(&model, "user_id = ?", userID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("user settings not found")
		}
		return nil, fmt.Errorf("failed to get user settings: %w", err)
	}
	return ModelToUserSettings(&model), nil
}

// CreateUserSettings creates user settings
func (r *UserSettingsRepositoryImpl) CreateUserSettings(settings *domain.UserSettings) error {
	model := UserSettingsToModel(settings)
	if err := r.db.Create(model).Error; err != nil {
		return fmt.Errorf("failed to create user settings: %w", err)
	}
	return nil
}

// UpdateUserSettings updates user settings
func (r *UserSettingsRepositoryImpl) UpdateUserSettings(settings *domain.UserSettings) error {
	model := UserSettingsToModel(settings)
	result := r.db.Where("user_id = ?", settings.UserID).Updates(model)
	if result.Error != nil {
		return fmt.Errorf("failed to update user settings: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return errors.New("user settings not found")
	}
	return nil
}
