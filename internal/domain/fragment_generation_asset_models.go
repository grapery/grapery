package domain

const (
	FragmentGenerationAssetSourceAIGeneration    = "ai_generation"
	FragmentGenerationAssetSourcePanelGeneration = "panel_generation"

	FragmentGenerationAssetKindSceneFinal          = "scene_final"
	FragmentGenerationAssetKindReferenceAsset      = "reference_asset"
	FragmentGenerationAssetKindAnchorImage         = "anchor_image"
	FragmentGenerationAssetKindUserReference       = "user_reference"
	FragmentGenerationAssetKindCharacterTurnaround = "character_turnaround"

	FragmentGenerationAssetEntityCharacter = "character"
	FragmentGenerationAssetEntityProp      = "prop"
	FragmentGenerationAssetEntityLocation  = "location"
	FragmentGenerationAssetEntityScene     = "scene"
)

// FragmentGenerationAsset is the queryable record for every image produced or reused
// during fragment generation, including consistency helper assets.
type FragmentGenerationAsset struct {
	ID           string `json:"id"`
	FragmentID   string `json:"fragmentId"`
	Source       string `json:"source"`
	TaskID       string `json:"taskId,omitempty"`
	Kind         string `json:"kind"`
	EntityKind   string `json:"entityKind,omitempty"`
	EntityKey    string `json:"entityKey,omitempty"`
	SceneIndex   *int   `json:"sceneIndex,omitempty"`
	URL          string `json:"url"`
	StorageKey   string `json:"storageKey,omitempty"`
	AspectRatio  string `json:"aspectRatio,omitempty"`
	TokensUsed   int    `json:"tokensUsed,omitempty"`
	Provider     string `json:"provider,omitempty"`
	Model        string `json:"model,omitempty"`
	SeriesSeed   int    `json:"seriesSeed,omitempty"`
	SceneSeed    int    `json:"sceneSeed,omitempty"`
	DisplayOrder int    `json:"displayOrder,omitempty"`
	MetadataJSON string `json:"-"`
	CreatedAt    int64  `json:"createdAt,omitempty"`
	UpdatedAt    int64  `json:"updatedAt,omitempty"`
}
