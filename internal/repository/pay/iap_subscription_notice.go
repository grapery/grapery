package pay

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// IAPSubscriptionNotice 订阅变更提示（客户端 ACK 后不再返回）。
type IAPSubscriptionNotice struct {
	ID        string         `gorm:"column:id;primaryKey;size:36" json:"id"`
	UserID    string         `gorm:"column:user_id;index;size:64;not null" json:"user_id"`
	Kind      string         `gorm:"column:kind;size:64;not null;index" json:"kind"`
	TitleKey  string         `gorm:"column:title_key;size:128;not null" json:"title_key"`
	BodyKey   string         `gorm:"column:body_key;size:128;not null" json:"body_key"`
	BodyArgs  string         `gorm:"column:body_args;type:json" json:"body_args"`
	AckedAt   *time.Time     `gorm:"column:acked_at;index" json:"acked_at"`
	CreatedAt time.Time      `gorm:"column:created_at" json:"created_at"`
	UpdatedAt time.Time      `gorm:"column:updated_at" json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deleted_at"`
}

func (IAPSubscriptionNotice) TableName() string {
	return "iap_subscription_notices"
}

// SubscriptionNoticePayload API 返回结构。
type SubscriptionNoticePayload struct {
	ID        string            `json:"id"`
	Kind      string            `json:"kind"`
	TitleKey  string            `json:"title_key"`
	BodyKey   string            `json:"body_key"`
	BodyArgs  map[string]string `json:"body_args,omitempty"`
	CreatedAt time.Time         `json:"created_at"`
}

func noticeKeysForKind(kind string) (titleKey, bodyKey string) {
	switch kind {
	case ChangeKindUpgradeSuccess:
		return "subscription_notice_upgrade_success_title", "subscription_notice_upgrade_success_body"
	case ChangeKindCancelScheduled:
		return "subscription_notice_cancel_scheduled_title", "subscription_notice_cancel_scheduled_body"
	case ChangeKindExpired:
		return "subscription_notice_expired_title", "subscription_notice_expired_body"
	case ChangeKindDowngradeScheduled:
		return "subscription_notice_downgrade_scheduled_title", "subscription_notice_downgrade_scheduled_body"
	case ChangeKindRenewed:
		return "subscription_notice_renewed_title", "subscription_notice_renewed_body"
	default:
		return "subscription_notice_generic_title", "subscription_notice_generic_body"
	}
}

// Notice kind constants for DB storage (match iOS keys).
const (
	ChangeKindUpgradeSuccess     = "upgrade_success"
	ChangeKindCancelScheduled    = "cancel_scheduled"
	ChangeKindExpired            = "expired"
	ChangeKindDowngradeScheduled = "downgrade_scheduled"
	ChangeKindRenewed            = "renewed"
)

// MapChangeKindToNoticeKind maps internal change kind to client notice kind.
func MapChangeKindToNoticeKind(changeKind string) string {
	switch changeKind {
	case "upgrade":
		return ChangeKindUpgradeSuccess
	case "cancel_renewal":
		return ChangeKindCancelScheduled
	case "expired", "revoked":
		return ChangeKindExpired
	case "downgrade_scheduled":
		return ChangeKindDowngradeScheduled
	case "renewal", "initial":
		return ChangeKindRenewed
	default:
		return ""
	}
}

// CreateSubscriptionNotice inserts a pending notice for the user.
func CreateSubscriptionNotice(ctx context.Context, userID, noticeKind string, bodyArgs map[string]string) (*SubscriptionNoticePayload, error) {
	if userID == "" || noticeKind == "" {
		return nil, errors.New("user id and notice kind required")
	}
	titleKey, bodyKey := noticeKeysForKind(noticeKind)
	argsJSON, _ := json.Marshal(bodyArgs)
	now := time.Now()
	row := IAPSubscriptionNotice{
		ID:        uuid.New().String(),
		UserID:    userID,
		Kind:      noticeKind,
		TitleKey:  titleKey,
		BodyKey:   bodyKey,
		BodyArgs:  string(argsJSON),
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := DataBase().WithContext(ctx).Create(&row).Error; err != nil {
		return nil, err
	}
	return row.toPayload(), nil
}

func (n *IAPSubscriptionNotice) toPayload() *SubscriptionNoticePayload {
	args := map[string]string{}
	if n.BodyArgs != "" {
		_ = json.Unmarshal([]byte(n.BodyArgs), &args)
	}
	return &SubscriptionNoticePayload{
		ID:        n.ID,
		Kind:      n.Kind,
		TitleKey:  n.TitleKey,
		BodyKey:   n.BodyKey,
		BodyArgs:  args,
		CreatedAt: n.CreatedAt,
	}
}

// GetPendingSubscriptionNotice returns the latest un-ACK notice for a user.
func GetPendingSubscriptionNotice(ctx context.Context, userID string) (*SubscriptionNoticePayload, error) {
	if userID == "" {
		return nil, nil
	}
	var row IAPSubscriptionNotice
	err := DataBase().WithContext(ctx).
		Where("user_id = ? AND acked_at IS NULL", userID).
		Order("created_at DESC").
		Limit(1).
		First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return row.toPayload(), nil
}

// AckSubscriptionNotice marks a notice as read for the given user.
func AckSubscriptionNotice(ctx context.Context, noticeID, userID string) error {
	if noticeID == "" || userID == "" {
		return errors.New("notice id and user id required")
	}
	now := time.Now()
	res := DataBase().WithContext(ctx).Model(&IAPSubscriptionNotice{}).
		Where("id = ? AND user_id = ? AND acked_at IS NULL", noticeID, userID).
		Updates(map[string]interface{}{"acked_at": now, "updated_at": now})
	if res.Error != nil {
		return res.Error
	}
	return nil
}
