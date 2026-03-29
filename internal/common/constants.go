package common

// BaseStatus defines common status values across all entities
// Use this for: User status, general entity state, simple on/off states
type BaseStatus string

const (
	// Common statuses
	StatusActive    BaseStatus = "active"    // Entity is active and in use
	StatusDraft     BaseStatus = "draft"     // Entity is in draft/not yet published
	StatusPending   BaseStatus = "pending"   // Awaiting action or verification
	StatusDeleted   BaseStatus = "deleted"   // Soft-deleted, can be restored
	StatusFailed    BaseStatus = "failed"    // Operation failed
	StatusCancelled BaseStatus = "cancelled" // Operation cancelled by user
	StatusSuspended BaseStatus = "suspended" // Temporarily disabled (e.g., user accounts)
	StatusExpired   BaseStatus = "expired"   // Time-based expiration (e.g., memberships)
)

// IsValid returns true if the status is a valid BaseStatus
func (s BaseStatus) IsValid() bool {
	switch s {
	case StatusActive, StatusDraft, StatusPending, StatusDeleted, StatusFailed,
		StatusCancelled, StatusSuspended, StatusExpired:
		return true
	}
	return false
}

// String returns the string representation of the status
func (s BaseStatus) String() string {
	return string(s)
}

// IsTerminal returns true if the status is a terminal state (no further transitions expected)
func (s BaseStatus) IsTerminal() bool {
	return s == StatusDeleted || s == StatusFailed || s == StatusCancelled || s == StatusExpired
}

// IsActive returns true if the entity is in an active/usable state
func (s BaseStatus) IsActive() bool {
	return s == StatusActive
}

// TaskStatus defines statuses for async tasks (AI generation, rendering, etc.)
// Use this for: AITask, RenderTask, and any long-running background operations
type TaskStatus string

const (
	TaskStatusPending    TaskStatus = "pending"    // Queued, not yet started
	TaskStatusProcessing TaskStatus = "processing" // Currently being processed
	TaskStatusCompleted  TaskStatus = "completed"  // Successfully finished
	TaskStatusFailed     TaskStatus = "failed"     // Failed with error
	TaskStatusCancelled  TaskStatus = "cancelled"  // Cancelled by user
)

// IsValid returns true if the status is a valid TaskStatus
func (s TaskStatus) IsValid() bool {
	switch s {
	case TaskStatusPending, TaskStatusProcessing, TaskStatusCompleted,
		TaskStatusFailed, TaskStatusCancelled:
		return true
	}
	return false
}

// String returns the string representation of the task status
func (s TaskStatus) String() string {
	return string(s)
}

// IsTerminal returns true if the task is in a terminal state
func (s TaskStatus) IsTerminal() bool {
	return s == TaskStatusCompleted || s == TaskStatusFailed || s == TaskStatusCancelled
}

// IsInProgress returns true if the task is currently being processed
func (s TaskStatus) IsInProgress() bool {
	return s == TaskStatusProcessing
}

// CanTransitionTo returns true if a transition from current to new status is valid
func (s TaskStatus) CanTransitionTo(new TaskStatus) bool {
	// Valid transitions: pending -> processing -> (completed/failed/cancelled)
	// Can also cancel from pending or processing
	switch s {
	case TaskStatusPending:
		return new == TaskStatusProcessing || new == TaskStatusCancelled
	case TaskStatusProcessing:
		return new == TaskStatusCompleted || new == TaskStatusFailed || new == TaskStatusCancelled
	default:
		return false // Terminal states cannot transition
	}
}

// ContentStatus defines statuses for user-generated content
// Use this for: Story, Fragment, and any creative content
type ContentStatus string

const (
	ContentStatusDraft     ContentStatus = "draft"     // Work in progress, not visible to others
	ContentStatusPublished ContentStatus = "published" // Published and visible to intended audience
	ContentStatusArchived  ContentStatus = "archived"  // Archived, not actively shown but preserved
	ContentStatusRendering ContentStatus = "rendering" // Currently being rendered (video, images, etc.)
)

// IsValid returns true if the status is a valid ContentStatus
func (s ContentStatus) IsValid() bool {
	switch s {
	case ContentStatusDraft, ContentStatusPublished, ContentStatusArchived, ContentStatusRendering:
		return true
	}
	return false
}

// String returns the string representation of the content status
func (s ContentStatus) String() string {
	return string(s)
}

// IsPublic returns true if the content is in a publicly visible state
func (s ContentStatus) IsPublic() bool {
	return s == ContentStatusPublished
}

// IsEditable returns true if the content can still be edited
func (s ContentStatus) IsEditable() bool {
	return s == ContentStatusDraft || s == ContentStatusArchived
}

// MembershipStatus defines statuses for membership and subscription entities
// Use this for: Membership, SubscriptionOrder
type MembershipStatus string

const (
	MembershipStatusActive    MembershipStatus = "active"    // Membership is active
	MembershipStatusExpired   MembershipStatus = "expired"   // Membership has expired
	MembershipStatusCancelled MembershipStatus = "cancelled" // Cancelled by user
)

// IsValid returns true if the status is a valid MembershipStatus
func (s MembershipStatus) IsValid() bool {
	switch s {
	case MembershipStatusActive, MembershipStatusExpired, MembershipStatusCancelled:
		return true
	}
	return false
}

// String returns the string representation of the membership status
func (s MembershipStatus) String() string {
	return string(s)
}

// IsActive returns true if the membership is currently active
func (s MembershipStatus) IsActive() bool {
	return s == MembershipStatusActive
}

// Default free-tier limits (keep in sync with registration in auth service).
const (
	// 需覆盖「角色三视图」等连续多张图（预检/扣费约 AIImageBillingUnitTokens/张）。
	DefaultFreeTierTokenQuota   = 25000
	DefaultFreeTierStorageBytes = 100 * 1024 * 1024 // 100 MiB

	// AIImageBillingUnitTokens 图片生成预检与 usage 缺失时的单张参考（与常见 TotalTokens 量级一致）。
	AIImageBillingUnitTokens = 4096
)

// OrderStatus defines statuses for payment and order entities
// Use this for: SubscriptionOrder, PaymentTransaction
type OrderStatus string

const (
	OrderStatusPending   OrderStatus = "pending"   // Order created, awaiting payment
	OrderStatusPaid      OrderStatus = "paid"      // Payment successfully received
	OrderStatusFailed    OrderStatus = "failed"    // Payment failed
	OrderStatusRefunded  OrderStatus = "refunded"  // Payment refunded
	OrderStatusCancelled OrderStatus = "cancelled" // Order cancelled before payment
)

// IsValid returns true if the status is a valid OrderStatus
func (s OrderStatus) IsValid() bool {
	switch s {
	case OrderStatusPending, OrderStatusPaid, OrderStatusFailed,
		OrderStatusRefunded, OrderStatusCancelled:
		return true
	}
	return false
}

// String returns the string representation of the order status
func (s OrderStatus) String() string {
	return string(s)
}

// IsSuccessful returns true if the order was successfully paid
func (s OrderStatus) IsSuccessful() bool {
	return s == OrderStatusPaid
}

// CanBeRefunded returns true if the order can be refunded
func (s OrderStatus) CanBeRefunded() bool {
	return s == OrderStatusPaid
}

// DeletionStatus defines statuses for account deletion requests
// Use this for: AccountDeletionRequest
type DeletionStatus string

const (
	DeletionStatusPending    DeletionStatus = "pending"    // Deletion requested, awaiting processing
	DeletionStatusProcessing DeletionStatus = "processing" // Currently being processed
	DeletionStatusCompleted  DeletionStatus = "completed"  // Deletion completed
	DeletionStatusCancelled  DeletionStatus = "cancelled"  // Deletion cancelled by user
)

// IsValid returns true if the status is a valid DeletionStatus
func (s DeletionStatus) IsValid() bool {
	switch s {
	case DeletionStatusPending, DeletionStatusProcessing,
		DeletionStatusCompleted, DeletionStatusCancelled:
		return true
	}
	return false
}

// String returns the string representation of the deletion status
func (s DeletionStatus) String() string {
	return string(s)
}

// IsTerminal returns true if the deletion request is in a terminal state
func (s DeletionStatus) IsTerminal() bool {
	return s == DeletionStatusCompleted || s == DeletionStatusCancelled
}

// CanBeCancelled returns true if the deletion request can still be cancelled
func (s DeletionStatus) CanBeCancelled() bool {
	return s == DeletionStatusPending || s == DeletionStatusProcessing
}

// PublicationStatus defines statuses for publication records
// Use this for: StoryPublication, content publication tracking
type PublicationStatus string

const (
	PublicationStatusPublished   PublicationStatus = "published"   // Content is published
	PublicationStatusUnpublished PublicationStatus = "unpublished" // Content is unpublished
)

// IsValid returns true if the status is a valid PublicationStatus
func (s PublicationStatus) IsValid() bool {
	switch s {
	case PublicationStatusPublished, PublicationStatusUnpublished:
		return true
	}
	return false
}

// String returns the string representation of the publication status
func (s PublicationStatus) String() string {
	return string(s)
}

// IsPublished returns true if the content is currently published
func (s PublicationStatus) IsPublished() bool {
	return s == PublicationStatusPublished
}
