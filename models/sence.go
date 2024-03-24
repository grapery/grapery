package models

// 场景，剧情，故事板
type Sence struct {
	IDBase
	Name        string `json:"name"`
	Description string `json:"description"`
	CreatorID   int64  `json:"creator_id"`
	Prompt      string `json:"prompt"`
	Ref         string `json:"ref"`
	Avatar      string `json:"avatar"`
	Negtive     string `json:"negtive"`
	Positive    string `json:"positive"`
	Style       string `json:"style"`
	Status      int    `json:"status"`
}

func (sence Sence) TableName() string {
	return "sence"
}

type Timeline struct {
	IDBase
	Name        string `json:"name"`
	RootItemId  int64  `json:"root_item_id"`
	ForkItemId  int64  `json:"fork_item_id"`
	Creator     int64  `json:"creator"`
	Description string `json:"description"`
	ProjectId   int64  `json:"project_id"`
	Avatar      string `json:"avatar"`
	Status      int    `json:"status"`
}

func (timeline Timeline) TableName() string {
	return "timeline"
}
