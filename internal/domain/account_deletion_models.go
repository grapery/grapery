package domain

// AccountDeletionRequestRow is the persisted deletion request domain shape.
type AccountDeletionRequestRow struct {
	ID                  string
	UserID              string
	Status              string // pending | processing | completed | cancelled
	ScheduledDeletionAt int64
	Reason              string
	Feedback            string
	RequestedAt         int64
	ProcessedAt         *int64
	CancelledAt         *int64
	CancelledReason     string
}
