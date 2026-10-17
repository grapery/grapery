package domain

// StoryboardComicPagePlan records the executable page options independently of
// SceneCount (the number of storyboard scenes/pages). PanelCount is per image.
// Stored in existing prompt_details JSON so retries use the accepted page plan.
type StoryboardComicPagePlan struct {
	PanelCount      int    `json:"panelCount"`
	LayoutPreset    string `json:"layoutPreset"`
	PageAspectRatio string `json:"pageAspectRatio"`
	DialogueMode    string `json:"dialogueMode"`
}
