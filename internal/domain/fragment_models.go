package domain

import "time"

// Fragment represents a fragment story - short, complete stories shared by users
type Fragment struct {
	ID        string    `json:"id" gorm:"primaryKey;type:varchar(36)"`
	CreatorID string    `json:"creatorId" gorm:"column:creator_id;type:varchar(36);not null;index"`
	Content   string    `json:"content" gorm:"column:content;type:text"`
	ImageUrls string    `json:"imageUrls" gorm:"column:image_urls;type:text"` // JSON array stored as text
	Visibility string   `json:"visibility" gorm:"column:visibility;type:varchar(20);not null;default:'public'"` // public, followers, private
	Likes     int       `json:"likes,omitempty" gorm:"column:likes;type:int;default:0"`
	Comments  int       `json:"comments,omitempty" gorm:"column:comments;type:int;default:0"`
	Shares    int       `json:"shares,omitempty" gorm:"column:shares;type:int;default:0"`
	Views     int       `json:"views,omitempty" gorm:"column:views;type:int;default:0"`
	CreatedAt time.Time `json:"createdAt" gorm:"column:created_at;type:datetime;autoCreateTime"`
	UpdatedAt time.Time `json:"updatedAt" gorm:"column:updated_at;type:datetime;autoUpdateTime"`
}

// TableName specifies the table name for Fragment
func (Fragment) TableName() string {
	return "fragments"
}

// FragmentLike represents a like relationship between a user and a fragment
type FragmentLike struct {
	ID         string    `json:"id" gorm:"primaryKey;type:varchar(36)"`
	FragmentID string    `json:"fragmentId" gorm:"column:fragment_id;type:varchar(36);not null;index:idx_fragment_user"`
	UserID     string    `json:"userId" gorm:"column:user_id;type:varchar(36);not null;index:idx_fragment_user;index:idx_user"`
	CreatedAt  time.Time `json:"createdAt" gorm:"column:created_at;type:datetime;autoCreateTime"`
}

// TableName specifies the table name for FragmentLike
func (FragmentLike) TableName() string {
	return "fragment_likes"
}

// FragmentVisibility constants
const (
	FragmentVisibilityPublic     = "public"
	FragmentVisibilityFollowers  = "followers"
	FragmentVisibilityPrivate    = "private"
)

// ValidFragmentVisibility returns true if the visibility string is valid
func ValidFragmentVisibility(visibility string) bool {
	switch visibility {
	case FragmentVisibilityPublic, FragmentVisibilityFollowers, FragmentVisibilityPrivate:
		return true
	default:
		return false
	}
}
