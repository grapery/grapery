package mysql

import (
	"context"
	"time"

	"github.com/grapestree/fgrapery/grapery/internal/domain"
	"gorm.io/gorm"
)

// ========== User Repository Methods ==========

// UserByID 根据 ID 获取用户
func (r *Repository) UserByID(ctx context.Context, id string) (*domain.User, error) {
	var user User
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&user).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return r.userToDomainPtr(&user), nil
}

// UserByEmail 根据邮箱获取用户
func (r *Repository) UserByEmail(ctx context.Context, email string) (*domain.User, error) {
	var user User
	if err := r.db.WithContext(ctx).Where("email = ?", email).First(&user).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return r.userToDomainPtr(&user), nil
}

// UserByUsername 根据用户名获取用户
func (r *Repository) UserByUsername(ctx context.Context, username string) (*domain.User, error) {
	var user User
	if err := r.db.WithContext(ctx).Where("username = ?", username).First(&user).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return r.userToDomainPtr(&user), nil
}

// CreateUser 创建用户
func (r *Repository) CreateUser(ctx context.Context, user *domain.User) error {
	dbUser := r.userFromDomain(user)
	return r.db.WithContext(ctx).Create(dbUser).Error
}

// UpdateUser 更新用户
func (r *Repository) UpdateUser(ctx context.Context, user *domain.User) error {
	dbUser := r.userFromDomain(user)
	return r.db.WithContext(ctx).Save(dbUser).Error
}

// DeleteUser 删除用户（软删除）
func (r *Repository) DeleteUser(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Delete(&User{}, "id = ?", id).Error
}

// ListUsers 获取用户列表
func (r *Repository) ListUsers(ctx context.Context, limit, offset int) ([]*domain.User, error) {
	var users []User
	if err := r.db.WithContext(ctx).Limit(limit).Offset(offset).Find(&users).Error; err != nil {
		return nil, err
	}

	result := make([]*domain.User, len(users))
	for i, u := range users {
		result[i] = r.userToDomainPtr(&u)
	}
	return result, nil
}

// ========== User Settings ==========

// UserSettings 获取用户设置
func (r *Repository) UserSettings(ctx context.Context, userID string) (*domain.UserSettings, error) {
	var settings UserSettings
	if err := r.db.WithContext(ctx).Where("user_id = ?", userID).First(&settings).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return r.userSettingsToDomain(&settings), nil
}

// CreateUserSettings 创建用户设置
func (r *Repository) CreateUserSettings(ctx context.Context, settings *domain.UserSettings) error {
	dbSettings := r.userSettingsFromDomain(settings)
	return r.db.WithContext(ctx).Create(dbSettings).Error
}

// UpdateUserSettings 更新用户设置
func (r *Repository) UpdateUserSettings(ctx context.Context, settings *domain.UserSettings) error {
	dbSettings := r.userSettingsFromDomain(settings)
	return r.db.WithContext(ctx).Save(dbSettings).Error
}

// ========== Membership ==========

// Membership 获取会员信息
func (r *Repository) Membership(ctx context.Context, userID string) (*domain.Membership, error) {
	var membership Membership
	if err := r.db.WithContext(ctx).Where("user_id = ?", userID).First(&membership).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return r.membershipToDomain(&membership), nil
}

// CreateMembership 创建会员信息
func (r *Repository) CreateMembership(ctx context.Context, membership *domain.Membership) error {
	dbMembership := r.membershipFromDomain(membership)
	return r.db.WithContext(ctx).Create(dbMembership).Error
}

// UpdateMembership 更新会员信息
func (r *Repository) UpdateMembership(ctx context.Context, membership *domain.Membership) error {
	dbMembership := r.membershipFromDomain(membership)
	return r.db.WithContext(ctx).Save(dbMembership).Error
}

// ========== Helper Functions ==========

// userToDomainPtr 转换 User 到 domain.User 指针
func (r *Repository) userToDomainPtr(user *User) *domain.User {
	var dateOfBirth *int64
	if user.DateOfBirth != 0 {
		dob := user.DateOfBirth
		dateOfBirth = &dob
	}

	var lastLoginAt *int64
	if user.LastLoginAt != 0 {
		lla := user.LastLoginAt
		lastLoginAt = &lla
	}

	return &domain.User{
		ID:                  user.ID,
		Username:            user.Username,
		Email:               user.Email,
		PasswordHash:        user.PasswordHash,
		DisplayName:         user.DisplayName,
		Avatar:              user.Avatar,
		Background:          user.Background,
		Bio:                 user.Bio,
		Location:            user.Location,
		Website:             user.Website,
		AIPromptPreferences: user.AIPromptPreferences,
		DateOfBirth:         dateOfBirth,
		Followers:           user.Followers,
		Following:           user.Following,
		StoryboardCount:     user.StoryboardCount,
		Status:              user.Status,
		EmailVerified:       user.EmailVerified,
		LastLoginAt:         lastLoginAt,
		CreatedAt:           user.CreatedAt,
		UpdatedAt:           user.UpdatedAt,
	}
}

// userFromDomain 从 domain.User 转换到 User
func (r *Repository) userFromDomain(user *domain.User) *User {
	var dateOfBirth int64
	if user.DateOfBirth != nil {
		dateOfBirth = *user.DateOfBirth
	}

	var lastLoginAt int64
	if user.LastLoginAt != nil {
		lastLoginAt = *user.LastLoginAt
	}

	return &User{
		ID:                  user.ID,
		Username:            user.Username,
		Email:               user.Email,
		PasswordHash:        user.PasswordHash,
		DisplayName:         user.DisplayName,
		Avatar:              user.Avatar,
		Background:          user.Background,
		Bio:                 user.Bio,
		Location:            user.Location,
		Website:             user.Website,
		AIPromptPreferences: user.AIPromptPreferences,
		DateOfBirth:         dateOfBirth,
		Followers:           user.Followers,
		Following:           user.Following,
		StoryboardCount:     user.StoryboardCount,
		Status:              user.Status,
		EmailVerified:       user.EmailVerified,
		LastLoginAt:         lastLoginAt,
		CreatedAt:           user.CreatedAt,
		UpdatedAt:           user.UpdatedAt,
	}
}

// userSettingsToDomain 转换 UserSettings 到 domain
func (r *Repository) userSettingsToDomain(settings *UserSettings) *domain.UserSettings {
	return &domain.UserSettings{
		ID:                        settings.ID,
		UserID:                    settings.UserID,
		Language:                  settings.Language,
		Theme:                     settings.Theme,
		FontSize:                  settings.FontSize,
		DataSaver:                 settings.DataSaver,
		ProfileVisibility:         settings.ProfileVisibility,
		DefaultStoryVisibility:    settings.DefaultStoryVisibility,
		DefaultFragmentVisibility: settings.DefaultFragmentVisibility,
		AllowFollowFrom:           settings.AllowFollowFrom,
		AllowCommentsFrom:         settings.AllowCommentsFrom,
		AllowMessagesFrom:         settings.AllowMessagesFrom,
		ShowOnlineStatus:          settings.ShowOnlineStatus,
		ShowReadReceipts:          settings.ShowReadReceipts,
		AIEnabled:                 settings.AIEnabled,
		AIDataSharing:             settings.AIDataSharing,
		NotificationSettings:      settings.NotificationSettings,
		UpdatedAt:                 settings.UpdatedAt,
	}
}

// userSettingsFromDomain 从 domain 转换到 UserSettings
func (r *Repository) userSettingsFromDomain(settings *domain.UserSettings) *UserSettings {
	return &UserSettings{
		ID:                        settings.ID,
		UserID:                    settings.UserID,
		Language:                  settings.Language,
		Theme:                     settings.Theme,
		FontSize:                  settings.FontSize,
		DataSaver:                 settings.DataSaver,
		ProfileVisibility:         settings.ProfileVisibility,
		DefaultStoryVisibility:    settings.DefaultStoryVisibility,
		DefaultFragmentVisibility: settings.DefaultFragmentVisibility,
		AllowFollowFrom:           settings.AllowFollowFrom,
		AllowCommentsFrom:         settings.AllowCommentsFrom,
		AllowMessagesFrom:         settings.AllowMessagesFrom,
		ShowOnlineStatus:          settings.ShowOnlineStatus,
		ShowReadReceipts:          settings.ShowReadReceipts,
		AIEnabled:                 settings.AIEnabled,
		AIDataSharing:             settings.AIDataSharing,
		NotificationSettings:      settings.NotificationSettings,
		UpdatedAt:                 settings.UpdatedAt,
	}
}

// membershipToDomain 转换 Membership 到 domain
func (r *Repository) membershipToDomain(membership *Membership) *domain.Membership {
	startDate := membership.StartDate.Unix()

	var endDate *int64
	if membership.EndDate != nil {
		ed := membership.EndDate.Unix()
		endDate = &ed
	}

	return &domain.Membership{
		ID:           membership.ID,
		UserID:       membership.UserID,
		Tier:         membership.Tier,
		Status:       membership.Status,
		StartDate:    startDate,
		EndDate:      endDate,
		AutoRenew:    membership.AutoRenew,
		TokenQuota:   membership.TokenQuota,
		TokenUsed:    membership.TokenUsed,
		StorageQuota: membership.StorageQuota,
		StorageUsed:  membership.StorageUsed,
		CreatedAt:    membership.CreatedAt.Unix(),
		UpdatedAt:    membership.UpdatedAt.Unix(),
	}
}

// membershipFromDomain 从 domain 转换到 Membership
func (r *Repository) membershipFromDomain(membership *domain.Membership) *Membership {
	startDate := time.Unix(membership.StartDate, 0)

	var endDate *time.Time
	if membership.EndDate != nil {
		ed := time.Unix(*membership.EndDate, 0)
		endDate = &ed
	}

	return &Membership{
		ID:           membership.ID,
		UserID:       membership.UserID,
		Tier:         membership.Tier,
		Status:       membership.Status,
		StartDate:    startDate,
		EndDate:      endDate,
		AutoRenew:    membership.AutoRenew,
		TokenQuota:   membership.TokenQuota,
		TokenUsed:    membership.TokenUsed,
		StorageQuota: membership.StorageQuota,
		StorageUsed:  membership.StorageUsed,
		CreatedAt:    time.Unix(membership.CreatedAt, 0),
		UpdatedAt:    time.Unix(membership.UpdatedAt, 0),
	}
}
