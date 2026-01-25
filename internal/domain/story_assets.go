package domain

// StoryScene describes a reusable scene asset owned by a story.
type StoryScene struct {
	ID           string   `json:"id"`
	StoryID      string   `json:"storyId"`
	Title        string   `json:"title"`
	Description  string   `json:"description,omitempty"`
	Image        string   `json:"image,omitempty"`
	Location     string   `json:"location,omitempty"`
	TimeOfDay    string   `json:"timeOfDay,omitempty"`
	SourceType   string   `json:"sourceType"`   // manual, upload, ai
	SourcePrompt string   `json:"sourcePrompt"` // AI prompt or user input
	SourceImage  string   `json:"sourceImage"`  // optional original image
	CreatedBy    string   `json:"createdBy"`    // user id
	LastEditedBy string   `json:"lastEditedBy"`
	CreatedAt    int64    `json:"createdAt"`
	UpdatedAt    int64    `json:"updatedAt"`
	IsPublic     bool     `json:"isPublic"`
	Tags         []string `json:"tags,omitempty"`
	// Relations
	Story *Story `json:"story,omitempty"`
}

// StoryboardCharacterRef links a storyboard node to a character.
type StoryboardCharacterRef struct {
	ID           string      `json:"id"`
	StoryboardID string      `json:"storyboardId"`
	CharacterID  string      `json:"characterId"`
	Role         string      `json:"role,omitempty"`
	Order        int         `json:"order"`
	Notes        string      `json:"notes,omitempty"`
	CreatedAt    int64       `json:"createdAt"`
	UpdatedAt    int64       `json:"updatedAt"`
	Storyboard   *Storyboard `json:"storyboard,omitempty"`
	Character    *Character  `json:"character,omitempty"`
}

// StoryboardSceneRef links a storyboard node to a story-level scene.
type StoryboardSceneRef struct {
	ID             string      `json:"id"`
	StoryboardID   string      `json:"storyboardId"`
	StorySceneID   string      `json:"storySceneId"`
	Sequence       int         `json:"sequence"`
	IsPrimaryScene bool        `json:"isPrimaryScene,omitempty"`
	CreatedAt      int64       `json:"createdAt"`
	UpdatedAt      int64       `json:"updatedAt"`
	Storyboard     *Storyboard `json:"storyboard,omitempty"`
	StoryScene     *StoryScene `json:"scene,omitempty"`
}
