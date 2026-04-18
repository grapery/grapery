package common

// BaseModel provides common fields for all entities.
// This struct should be embedded in all domain models that require
// ID tracking and timestamp management. It includes:
// - ID: Primary key for the entity
// - CreatedAt: Unix timestamp of when the entity was created
// - UpdatedAt: Unix timestamp of when the entity was last updated
type BaseModel struct {
	ID        string `gorm:"primaryKey" json:"id"`
	CreatedAt int64  `json:"createdAt"`
	UpdatedAt int64  `json:"updatedAt"`
}

// Timestamps provides creation and update times for entities.
// This struct is useful for models that don't need an ID field
// but still require timestamp tracking.
// - CreatedAt: Unix timestamp of when the entity was created
// - UpdatedAt: Unix timestamp of when the entity was last updated
type Timestamps struct {
	CreatedAt int64 `json:"createdAt"`
	UpdatedAt int64 `json:"updatedAt"`
}

// EngagementStats provides common engagement metrics for content entities.
// This struct should be embedded in models that track user engagement
// such as posts, articles, videos, or any interactable content.
// - Likes: Number of likes the content has received
// - Comments: Number of comments on the content
// - Shares: Number of times the content has been shared
// - Views: Number of views the content has received
type EngagementStats struct {
	Likes    int `json:"likes"`
	Comments int `json:"comments"`
	Shares   int `json:"shares"`
	Views    int `json:"views"`
}

// SocialStats provides follower/following metrics for user profiles.
// This struct should be embedded in user-related models that track
// social connections and network size.
// - Followers: Number of users following this entity
// - Following: Number of users this entity is following
type SocialStats struct {
	Followers int `json:"followers"`
	Following int `json:"following"`
}

// EntityRef provides a lightweight reference to any entity.
// This struct is useful for reducing payload size when full entity
// details are not needed, such as in list views or nested references.
// - ID: Unique identifier of the entity
// - Name: Display name of the entity (optional)
// - Avatar: URL or path to the entity's avatar image (optional)
type EntityRef struct {
	ID     string `json:"id"`
	Name   string `json:"name,omitempty"`
	Avatar string `json:"avatar,omitempty"`
}

// UserRef provides a lightweight reference to a user.
// This struct should be used when user information needs to be included
// but full user profile details are not required. It's particularly
// useful in post authors, comment creators, and activity feeds.
// - ID: Unique identifier of the user
// - Username: The user's unique username/handle
// - Name: The user's display name
// - Avatar: URL or path to the user's profile picture (optional)
type UserRef struct {
	ID       string `json:"id"`
	Username string `json:"username"`
	Name     string `json:"name"`
	Avatar   string `json:"avatar,omitempty"`
}
