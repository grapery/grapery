package mysql

import (
	"context"
	"time"

	"github.com/grapestree/fgrapery/grapery/internal/domain"
	"gorm.io/gorm"
)

// UserDevice GORM model
type UserDevice struct {
	ID           string         `gorm:"primaryKey;size:36"`
	UserID       string         `gorm:"size:36;not null;index"`
	User         User           `gorm:"foreignKey:UserID"`
	DeviceToken  string         `gorm:"size:512;not null;uniqueIndex"`
	Platform     string         `gorm:"size:20;not null;index"` // ios, android, macos, etc.
	PushProvider string         `gorm:"size:20;not null"`       // apns, fcm
	DeviceModel  string         `gorm:"size:100"`
	OSVersion    string         `gorm:"size:50"`
	AppVersion   string         `gorm:"size:20"`
	AppBuild     string         `gorm:"size:20"`
	Locale       string         `gorm:"size:10"`
	Timezone     string         `gorm:"size:50"`
	IsActive     bool           `gorm:"default:true;index"`
	LastActiveAt int64          `gorm:"type:bigint"`
	CreatedAt    int64          `gorm:"type:bigint;autoCreateTime"`
	UpdatedAt    int64          `gorm:"type:bigint;autoUpdateTime"`
	DeletedAt    gorm.DeletedAt `gorm:"index"`
}

// TableName returns the table name
func (UserDevice) TableName() string {
	return "user_devices"
}

// CreateUserDevice creates a new user device
func (r *Repository) CreateUserDevice(ctx context.Context, device *domain.UserDevice) error {
	model := domainToUserDeviceModel(device)
	return r.db.WithContext(ctx).Create(model).Error
}

// GetUserDevice gets a user device by ID
func (r *Repository) GetUserDevice(ctx context.Context, id string) (*domain.UserDevice, error) {
	var model UserDevice
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&model).Error; err != nil {
		return nil, err
	}
	return userDeviceModelToDomain(&model), nil
}

// GetUserDeviceByToken gets a user device by device token
func (r *Repository) GetUserDeviceByToken(ctx context.Context, deviceToken string) (*domain.UserDevice, error) {
	var model UserDevice
	if err := r.db.WithContext(ctx).Where("device_token = ?", deviceToken).First(&model).Error; err != nil {
		return nil, err
	}
	return userDeviceModelToDomain(&model), nil
}

// GetUserDevicesByUserID gets all devices for a user
func (r *Repository) GetUserDevicesByUserID(ctx context.Context, userID string) ([]*domain.UserDevice, error) {
	var models []UserDevice
	if err := r.db.WithContext(ctx).Where("user_id = ?", userID).Order("last_active_at DESC").Find(&models).Error; err != nil {
		return nil, err
	}
	devices := make([]*domain.UserDevice, len(models))
	for i, m := range models {
		devices[i] = userDeviceModelToDomain(&m)
	}
	return devices, nil
}

// GetActiveUserDevicesByUserID gets all active devices for a user
func (r *Repository) GetActiveUserDevicesByUserID(ctx context.Context, userID string) ([]*domain.UserDevice, error) {
	var models []UserDevice
	if err := r.db.WithContext(ctx).Where("user_id = ? AND is_active = ?", userID, true).Order("last_active_at DESC").Find(&models).Error; err != nil {
		return nil, err
	}
	devices := make([]*domain.UserDevice, len(models))
	for i, m := range models {
		devices[i] = userDeviceModelToDomain(&m)
	}
	return devices, nil
}

// GetUserDevicesByPlatform gets all devices for a user by platform
func (r *Repository) GetUserDevicesByPlatform(ctx context.Context, userID string, platform domain.DevicePlatform) ([]*domain.UserDevice, error) {
	var models []UserDevice
	if err := r.db.WithContext(ctx).Where("user_id = ? AND platform = ? AND is_active = ?", userID, string(platform), true).Find(&models).Error; err != nil {
		return nil, err
	}
	devices := make([]*domain.UserDevice, len(models))
	for i, m := range models {
		devices[i] = userDeviceModelToDomain(&m)
	}
	return devices, nil
}

// UpdateUserDevice updates a user device
func (r *Repository) UpdateUserDevice(ctx context.Context, device *domain.UserDevice) error {
	model := domainToUserDeviceModel(device)
	return r.db.WithContext(ctx).Save(model).Error
}

// DeleteUserDevice deletes a user device by ID
func (r *Repository) DeleteUserDevice(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Delete(&UserDevice{}, "id = ?", id).Error
}

// DeleteUserDeviceByToken deletes a user device by token
func (r *Repository) DeleteUserDeviceByToken(ctx context.Context, deviceToken string) error {
	return r.db.WithContext(ctx).Delete(&UserDevice{}, "device_token = ?", deviceToken).Error
}

// DeactivateUserDevice deactivates a user device
func (r *Repository) DeactivateUserDevice(ctx context.Context, deviceToken string) error {
	return r.db.WithContext(ctx).Model(&UserDevice{}).
		Where("device_token = ?", deviceToken).
		Update("is_active", false).Error
}

// UpdateUserDeviceLastActive updates the last active time for a device
func (r *Repository) UpdateUserDeviceLastActive(ctx context.Context, deviceToken string, lastActiveAt int64) error {
	return r.db.WithContext(ctx).Model(&UserDevice{}).
		Where("device_token = ?", deviceToken).
		Updates(map[string]interface{}{
			"last_active_at": lastActiveAt,
			"is_active":      true,
		}).Error
}

// domainToUserDeviceModel converts domain model to GORM model
func domainToUserDeviceModel(d *domain.UserDevice) *UserDevice {
	return &UserDevice{
		ID:           d.ID,
		UserID:       d.UserID,
		DeviceToken:  d.DeviceToken,
		Platform:     string(d.Platform),
		PushProvider: d.PushProvider,
		DeviceModel:  d.DeviceModel,
		OSVersion:    d.OSVersion,
		AppVersion:   d.AppVersion,
		AppBuild:     d.AppBuild,
		Locale:       d.Locale,
		Timezone:     d.Timezone,
		IsActive:     d.IsActive,
		LastActiveAt: d.LastActiveAt,
		CreatedAt:    d.CreatedAt,
		UpdatedAt:    d.UpdatedAt,
	}
}

// userDeviceModelToDomain converts GORM model to domain model
func userDeviceModelToDomain(m *UserDevice) *domain.UserDevice {
	return &domain.UserDevice{
		ID:           m.ID,
		UserID:       m.UserID,
		DeviceToken:  m.DeviceToken,
		Platform:     domain.DevicePlatform(m.Platform),
		PushProvider: m.PushProvider,
		DeviceModel:  m.DeviceModel,
		OSVersion:    m.OSVersion,
		AppVersion:   m.AppVersion,
		AppBuild:     m.AppBuild,
		Locale:       m.Locale,
		Timezone:     m.Timezone,
		IsActive:     m.IsActive,
		LastActiveAt: m.LastActiveAt,
		CreatedAt:    m.CreatedAt,
		UpdatedAt:    m.UpdatedAt,
	}
}

// RegisterOrUpdateDevice registers a new device or updates an existing one
func (r *Repository) RegisterOrUpdateDevice(ctx context.Context, device *domain.UserDevice) error {
	now := time.Now().Unix()

	// Check if device already exists
	existing, err := r.GetUserDeviceByToken(ctx, device.DeviceToken)
	if err == nil && existing != nil {
		// Update existing device
		existing.UserID = device.UserID
		existing.Platform = device.Platform
		existing.PushProvider = device.PushProvider
		existing.DeviceModel = device.DeviceModel
		existing.OSVersion = device.OSVersion
		existing.AppVersion = device.AppVersion
		existing.AppBuild = device.AppBuild
		existing.Locale = device.Locale
		existing.Timezone = device.Timezone
		existing.IsActive = true
		existing.LastActiveAt = now
		existing.UpdatedAt = now
		return r.UpdateUserDevice(ctx, existing)
	}

	// Create new device
	device.IsActive = true
	device.LastActiveAt = now
	device.CreatedAt = now
	device.UpdatedAt = now
	return r.CreateUserDevice(ctx, device)
}
