package domain

// Tag 标签
type Tag struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Category   string `json:"category"` // genre, theme, style, mood
	UsageCount int    `json:"usageCount"`
	CreatedAt  int64  `json:"createdAt"`
}

// StoryTag 故事标签关联
type StoryTag struct {
	ID        string `json:"id"`
	StoryID   string `json:"storyId"`
	TagID     string `json:"tagId"`
	CreatedAt int64  `json:"createdAt"`

	// Relations
	Story *Story `json:"story,omitempty"`
	Tag   *Tag   `json:"tag,omitempty"`
}

// CharacterTag 角色标签关联
type CharacterTag struct {
	ID          string `json:"id"`
	CharacterID string `json:"characterId"`
	TagID       string `json:"tagId"`
	CreatedAt   int64  `json:"createdAt"`

	// Relations
	Character *Character `json:"character,omitempty"`
	Tag       *Tag       `json:"tag,omitempty"`
}
