package mysql

import (
	"errors"
	"strings"

	"github.com/grapestree/fgrapery/grapery/internal/auth"
	"github.com/grapestree/fgrapery/grapery/internal/common"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// SeedSystemAnonymousUser inserts the non-login placeholder user used to hold orphaned public content after account deletion completes.
func SeedSystemAnonymousUser(db *gorm.DB, log *zap.Logger, systemUserID string) error {
	if systemUserID == "" {
		return errors.New("system anonymous user id is empty")
	}
	var n int64
	if err := db.Model(&User{}).Where("id = ?", systemUserID).Count(&n).Error; err != nil {
		return err
	}
	if n > 0 {
		return nil
	}

	// bcrypt rejects passwords longer than 72 bytes; prefix + UUID exceeded that limit.
	// This account is non-login; a fixed sentinel password is sufficient.
	h, err := auth.HashPassword("system-anonymous-no-login-v1")
	if err != nil {
		return err
	}
	ref := strings.ReplaceAll(systemUserID, "-", "")
	if len(ref) > 18 {
		ref = ref[:18]
	}
	ref = "s" + ref
	if len(ref) > 20 {
		ref = ref[:20]
	}
	short := strings.ReplaceAll(systemUserID, "-", "")
	if len(short) > 24 {
		short = short[:24]
	}
	u := User{
		ID:             systemUserID,
		Username:       "deleted_user",
		Email:          "noreply_deleted_sys_" + short + "@invalid.local",
		PasswordHash:   h,
		DisplayName:    "已注销用户",
		Status:         string(common.StatusSystem),
		StoryboardCount: 0,
		FragmentsCount:  0,
		ReferralCode:    ref,
	}

	if err := db.Create(&u).Error; err != nil {
		if log != nil {
			log.Warn("system anonymous user seed skipped (possibly concurrent create)", zap.Error(err))
		}
		return nil // idempotent duplicate username/email in dev
	}
	if log != nil {
		log.Info("created system anonymous user for account deletion reassignment",
			zap.String("userId", systemUserID))
	}
	return nil
}
