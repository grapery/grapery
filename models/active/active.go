package active

import (
	"time"
)

type Active struct {
	AID       int64
	CreatedAt time.Time
	DeletedAt time.Time
	Deleted   bool
}
