package mysql

// FragmentGenerationAssetDB stores every generated/reused fragment image asset in a queryable table.
type FragmentGenerationAssetDB struct {
	ID           string `gorm:"primaryKey;size:36"`
	FragmentID   string `gorm:"size:36;not null;index:idx_fga_fragment;index:idx_fga_fragment_kind;index:idx_fga_fragment_entity"`
	Source       string `gorm:"size:40;not null;index:idx_fga_source_task"`
	TaskID       string `gorm:"size:36;index:idx_fga_source_task"`
	Kind         string `gorm:"size:40;not null;index:idx_fga_fragment_kind"`
	EntityKind   string `gorm:"size:40;index"`
	EntityKey    string `gorm:"size:80;index:idx_fga_fragment_entity"`
	SceneIndex   *int   `gorm:"type:int;index"`
	URL          string `gorm:"type:text;not null"`
	StorageKey   string `gorm:"size:500"`
	AspectRatio  string `gorm:"size:20"`
	TokensUsed   int    `gorm:"type:int;default:0"`
	Provider     string `gorm:"size:50"`
	Model        string `gorm:"size:100"`
	SeriesSeed   int    `gorm:"type:int;default:0"`
	SceneSeed    int    `gorm:"type:int;default:0"`
	DisplayOrder int    `gorm:"type:int;default:0"`
	MetadataJSON string `gorm:"column:metadata;type:json"`
	CreatedAt    int64  `gorm:"type:bigint;autoCreateTime;index"`
	UpdatedAt    int64  `gorm:"type:bigint;autoUpdateTime"`
}

func (FragmentGenerationAssetDB) TableName() string {
	return "fragment_generation_assets"
}
