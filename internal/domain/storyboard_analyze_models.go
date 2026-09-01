package domain

// StoryboardGenerationIntent is the structured intent returned by analyze for storyboard creation.
type StoryboardGenerationIntent struct {
	UserInput             string   `json:"userInput"`
	SceneCount            int      `json:"sceneCount"`
	Style                 string   `json:"style,omitempty"`
	AspectRatio           string   `json:"aspectRatio,omitempty"`
	Language              string   `json:"language,omitempty"`
	StoryID               string   `json:"storyId,omitempty"`
	ParentStoryboardID    string   `json:"parentStoryboardId,omitempty"`
	CharacterIDs          []string `json:"characterIds,omitempty"`
	SceneIDs              []string `json:"sceneIds,omitempty"`
	UseComicPagePipeline  bool     `json:"useComicPagePipeline,omitempty"`
	TargetDraftStoryboard string   `json:"targetDraftStoryboardId,omitempty"`
}

// StoryboardFrameworkAlignment explains which story constraints will be inherited
// before generation starts. It intentionally stays compact so clients can surface
// it as a low-friction confirmation card instead of another creation form.
type StoryboardFrameworkAlignment struct {
	Status                string   `json:"status"` // aligned | needs_confirmation | blocked
	StoryTitle            string   `json:"storyTitle,omitempty"`
	StoryGenre            string   `json:"storyGenre,omitempty"`
	StoryPremise          string   `json:"storyPremise,omitempty"`
	ParentStoryboardTitle string   `json:"parentStoryboardTitle,omitempty"`
	ParentEnding          string   `json:"parentEnding,omitempty"`
	InheritedFacts        []string `json:"inheritedFacts,omitempty"`
	Warnings              []string `json:"warnings,omitempty"`
	RequiresConfirmation  bool     `json:"requiresConfirmation,omitempty"`
	Blocking              bool     `json:"blocking,omitempty"`
}

// StoryboardAnalyzeRequest is the first-phase planning request for conversational storyboard creation.
type StoryboardAnalyzeRequest struct {
	UserInput               string `json:"userInput"`
	StoryID                 string `json:"storyId,omitempty"`
	ParentStoryboardID      string `json:"parentStoryboardId,omitempty"`
	SceneCount              int    `json:"sceneCount,omitempty"`
	Style                   string `json:"style,omitempty"`
	AspectRatio             string `json:"aspectRatio,omitempty"`
	Language                string `json:"language,omitempty"`
	TargetDraftStoryboardID string `json:"targetDraftStoryboardId,omitempty"`
	UseComicPagePipeline    *bool  `json:"useComicPagePipeline,omitempty"`
}

// StoryboardCharacterCandidate is a character suggestion surfaced in planning.
type StoryboardCharacterCandidate struct {
	ID   string `json:"id,omitempty"`
	Name string `json:"name"`
	Hint string `json:"hint,omitempty"`
}

// StoryboardRecommendedOptions is client-facing configuration advice after analyze.
type StoryboardRecommendedOptions struct {
	StyleCandidates []string `json:"styleCandidates,omitempty"`
	CanStart        bool     `json:"canStart"`
}

// StoryboardAnalyzeResponse is the first-phase planning response.
// IntentType: new_storyboard | revise_current | adjust_options | chat_only | ask_clarification
type StoryboardAnalyzeResponse struct {
	AssistantMessage    string                         `json:"assistantMessage"`
	IntentType          string                         `json:"intentType,omitempty"`
	EditPlan            CreativeEditPlan               `json:"editPlan"`
	GenerationIntent    StoryboardGenerationIntent     `json:"generationIntent"`
	CharacterCandidates []StoryboardCharacterCandidate `json:"characterCandidates,omitempty"`
	RecommendedOptions  StoryboardRecommendedOptions   `json:"recommendedOptions"`
	FrameworkAlignment  *StoryboardFrameworkAlignment  `json:"frameworkAlignment,omitempty"`
}
