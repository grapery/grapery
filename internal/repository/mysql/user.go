package mysql

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/grapestree/fgrapery/grapery/internal/cache"
	"github.com/grapestree/fgrapery/grapery/internal/common"
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

// UserByPhone 根据手机号获取用户（规范化后的国内 11 位，不含 +86）
func (r *Repository) UserByPhone(ctx context.Context, phone string) (*domain.User, error) {
	if phone == "" {
		return nil, nil
	}
	var user User
	if err := r.db.WithContext(ctx).Where("phone = ?", phone).First(&user).Error; err != nil {
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
	if err := r.db.WithContext(ctx).Create(dbMembership).Error; err != nil {
		return err
	}
	cache.InvalidateMembership(ctx, r.cache, membership.UserID)
	return nil
}

// UpdateMembership 更新会员信息
func (r *Repository) UpdateMembership(ctx context.Context, membership *domain.Membership) error {
	dbMembership := r.membershipFromDomain(membership)
	if err := r.db.WithContext(ctx).Save(dbMembership).Error; err != nil {
		return err
	}
	cache.InvalidateMembership(ctx, r.cache, membership.UserID)
	return nil
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

	var phoneVerifiedAt *int64
	if user.PhoneVerifiedAt > 0 {
		pv := user.PhoneVerifiedAt
		phoneVerifiedAt = &pv
	}

	return &domain.User{
		BaseModel: common.BaseModel{
			ID:        user.ID,
			CreatedAt: user.CreatedAt,
			UpdatedAt: user.UpdatedAt,
		},
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
		SocialStats: common.SocialStats{
			Followers: user.Followers,
			Following: user.Following,
		},
		StoryboardCount:      user.StoryboardCount,
		Status:               user.Status,
		EmailVerified:        user.EmailVerified,
		Phone:                user.Phone,
		PhoneVerifiedAt:      phoneVerifiedAt,
		PendingOAuthPhoneSMS: user.PendingOAuthPhoneSMS,
		LastLoginAt:          lastLoginAt,
		Points:               user.Points,
		ReferralCode:         user.ReferralCode,
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

	var phoneVerifiedAt int64
	if user.PhoneVerifiedAt != nil {
		phoneVerifiedAt = *user.PhoneVerifiedAt
	}

	return &User{
		ID:                   user.ID,
		Username:             user.Username,
		Email:                user.Email,
		PasswordHash:         user.PasswordHash,
		DisplayName:          user.DisplayName,
		Avatar:               user.Avatar,
		Background:           user.Background,
		Bio:                  user.Bio,
		Location:             user.Location,
		Website:              user.Website,
		AIPromptPreferences:  user.AIPromptPreferences,
		DateOfBirth:          dateOfBirth,
		Followers:            user.Followers,
		Following:            user.Following,
		StoryboardCount:      user.StoryboardCount,
		Status:               user.Status,
		EmailVerified:        user.EmailVerified,
		Phone:                user.Phone,
		PhoneVerifiedAt:      phoneVerifiedAt,
		PendingOAuthPhoneSMS: user.PendingOAuthPhoneSMS,
		LastLoginAt:          lastLoginAt,
		Points:               user.Points,
		ReferralCode:         user.ReferralCode,
		CreatedAt:            user.CreatedAt,
		UpdatedAt:            user.UpdatedAt,
	}
}

// userSettingsToDomain 转换 UserSettings 到 domain
func (r *Repository) userSettingsToDomain(settings *UserSettings) *domain.UserSettings {
	var preferredGenres []string
	if strings.TrimSpace(settings.PreferredGenresJSON) != "" {
		_ = json.Unmarshal([]byte(settings.PreferredGenresJSON), &preferredGenres)
	}
	return &domain.UserSettings{
		BaseModel: common.BaseModel{
			ID:        settings.ID,
			CreatedAt: 0, // UserSettings doesn't have CreatedAt in MySQL model
			UpdatedAt: settings.UpdatedAt,
		},
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
		ShowPublicStories:         settings.ShowPublicStories,
		ShowPublicFragments:       settings.ShowPublicFragments,
		ShowPublicBookmarks:       settings.ShowPublicBookmarks,
		AIEnabled:                 settings.AIEnabled,
		AIDataSharing:             settings.AIDataSharing,
		NotificationSettings:      settings.NotificationSettings,
		PreferredGenres:           preferredGenres,
	}
}

// userSettingsFromDomain 从 domain 转换到 UserSettings
func (r *Repository) userSettingsFromDomain(settings *domain.UserSettings) *UserSettings {
	preferredGenresJSON := "[]"
	if b, err := json.Marshal(settings.PreferredGenres); err == nil {
		preferredGenresJSON = string(b)
	}
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
		ShowPublicStories:         settings.ShowPublicStories,
		ShowPublicFragments:       settings.ShowPublicFragments,
		ShowPublicBookmarks:       settings.ShowPublicBookmarks,
		AIEnabled:                 settings.AIEnabled,
		AIDataSharing:             settings.AIDataSharing,
		NotificationSettings:      settings.NotificationSettings,
		PreferredGenresJSON:       preferredGenresJSON,
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

// ========== Referral System (StoryCreationAppUI Design) ==========

// GetUserByReferralCode 根据邀请码获取用户
func (r *Repository) GetUserByReferralCode(ctx context.Context, referralCode string) (*domain.User, error) {
	var user User
	if err := r.db.WithContext(ctx).Where("referral_code = ?", referralCode).First(&user).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return r.userToDomainPtr(&user), nil
}

// CreateUserReferral 创建邀请记录
func (r *Repository) CreateUserReferral(ctx context.Context, referral *domain.UserReferral) error {
	dbReferral := &UserReferral{
		ID:           referral.ID,
		ReferrerID:   referral.ReferrerID,
		RefereeID:    referral.RefereeID,
		ReferralCode: referral.ReferralCode,
		PointsEarned: referral.PointsEarned,
		Status:       referral.Status,
		CreatedAt:    time.Unix(referral.CreatedAt, 0),
	}
	if referral.RewardedAt > 0 {
		rewardedAt := time.Unix(referral.RewardedAt, 0)
		dbReferral.RewardedAt = &rewardedAt
	}
	return r.db.WithContext(ctx).Create(dbReferral).Error
}

// GetUserReferralByReferee 根据被邀请人ID获取邀请记录
func (r *Repository) GetUserReferralByReferee(ctx context.Context, refereeID string) (*domain.UserReferral, error) {
	var referral UserReferral
	if err := r.db.WithContext(ctx).Where("referee_id = ?", refereeID).First(&referral).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return r.userReferralToDomain(&referral), nil
}

// GetReferralsByUser 获取用户的邀请列表
func (r *Repository) GetReferralsByUser(ctx context.Context, referrerID string, limit, offset int) ([]*domain.UserReferral, error) {
	var referrals []UserReferral
	if err := r.db.WithContext(ctx).
		Where("referrer_id = ?", referrerID).
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&referrals).Error; err != nil {
		return nil, err
	}

	result := make([]*domain.UserReferral, len(referrals))
	for i, ref := range referrals {
		result[i] = r.userReferralToDomain(&ref)
	}
	return result, nil
}

// GetReferralStats 获取用户邀请统计
func (r *Repository) GetReferralStats(ctx context.Context, userID string) (*domain.ReferralStats, error) {
	stats := &domain.ReferralStats{}

	// 总邀请数
	var totalCount int64
	if err := r.db.WithContext(ctx).Model(&UserReferral{}).
		Where("referrer_id = ?", userID).
		Count(&totalCount).Error; err != nil {
		return nil, err
	}
	stats.TotalReferrals = int(totalCount)

	// 通过邀请获得的积分
	var totalPoints int64
	if err := r.db.WithContext(ctx).Model(&UserReferral{}).
		Where("referrer_id = ? AND status = ?", userID, domain.ReferralStatusRewarded).
		Select("COALESCE(SUM(points_earned), 0)").
		Scan(&totalPoints).Error; err != nil {
		return nil, err
	}
	stats.PointsEarned = int(totalPoints)

	// 待完成邀请数
	var pendingCount int64
	if err := r.db.WithContext(ctx).Model(&UserReferral{}).
		Where("referrer_id = ? AND status = ?", userID, domain.ReferralStatusPending).
		Count(&pendingCount).Error; err != nil {
		return nil, err
	}
	stats.PendingReferrals = int(pendingCount)

	return stats, nil
}

// AddUserPoints 增加用户积分
func (r *Repository) AddUserPoints(ctx context.Context, userID string, points int) error {
	return r.db.WithContext(ctx).Model(&User{}).
		Where("id = ?", userID).
		UpdateColumn("points", gorm.Expr("points + ?", points)).Error
}

// userReferralToDomain 转换 UserReferral 到 domain
func (r *Repository) userReferralToDomain(referral *UserReferral) *domain.UserReferral {
	result := &domain.UserReferral{
		ID:           referral.ID,
		ReferrerID:   referral.ReferrerID,
		RefereeID:    referral.RefereeID,
		ReferralCode: referral.ReferralCode,
		PointsEarned: referral.PointsEarned,
		Status:       referral.Status,
		CreatedAt:    referral.CreatedAt.Unix(),
	}
	if referral.RewardedAt != nil {
		result.RewardedAt = referral.RewardedAt.Unix()
	}
	return result
}
