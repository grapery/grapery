package service

import (
	"time"

	"github.com/google/uuid"
	"github.com/grapestree/fgrapery/grapery/internal/common"
	"github.com/grapestree/fgrapery/grapery/internal/domain"
)

// ApplyNewUserWelcomePoints sets users.points for first-time account creation.
func ApplyNewUserWelcomePoints(user *domain.User) {
	if user == nil {
		return
	}
	user.Points = common.NewUserWelcomeCredits
}

// NewUserWelcomeMembership returns free-tier membership with welcome AI generation quota.
func NewUserWelcomeMembership(userID string, now int64) *domain.Membership {
	if now == 0 {
		now = time.Now().Unix()
	}
	return &domain.Membership{
		ID:           uuid.New().String(),
		UserID:       userID,
		Tier:         "free",
		Status:       string(common.MembershipStatusActive),
		StartDate:    now,
		AutoRenew:    false,
		TokenQuota:   common.NewUserWelcomeTokenQuota,
		TokenUsed:    0,
		StorageQuota: common.DefaultFreeTierStorageBytes,
		StorageUsed:  0,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
}
