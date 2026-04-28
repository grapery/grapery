package mysql

import (
	"time"

	"gorm.io/gorm"
)

type StoryboardGenerationRun struct {
	ID                    string         `gorm:"primaryKey;size:36"`
	StoryboardID          string         `gorm:"size:36;not null;index"`
	Storyboard            Storyboard     `gorm:"foreignKey:StoryboardID"`
	StoryID               string         `gorm:"size:36;not null;index"`
	Story                 Story          `gorm:"foreignKey:StoryID"`
	UserID                string         `gorm:"size:36;not null;index"`
	User                  User           `gorm:"foreignKey:UserID"`
	Status                string         `gorm:"size:20;not null;default:'pending';index"`
	Progress              int            `gorm:"default:0"`
	CurrentStep           string         `gorm:"size:80;index"`
	RequestJSON           string         `gorm:"type:json"`
	ContextSnapshotJSON   string         `gorm:"type:json"`
	AlignmentSnapshotJSON string         `gorm:"type:json"`
	StoryboardBibleJSON   string         `gorm:"type:json"`
	BeatsJSON             string         `gorm:"type:json"`
	ScenePlanJSON         string         `gorm:"type:json"`
	ConsistencyIssuesJSON string         `gorm:"type:json"`
	MetricsJSON           string         `gorm:"type:json"`
	ErrorCode             string         `gorm:"size:80;index"`
	ErrorMessage          string         `gorm:"type:text"`
	CreatedAt             time.Time      `gorm:"autoCreateTime;index"`
	UpdatedAt             time.Time      `gorm:"autoUpdateTime"`
	CompletedAt           *time.Time     `gorm:"index"`
	DeletedAt             gorm.DeletedAt `gorm:"index"`
}

type StoryboardGenerationAsset struct {
	ID           string                  `gorm:"primaryKey;size:36"`
	RunID        string                  `gorm:"size:36;not null;index:idx_storyboard_generation_asset_run_key"`
	Run          StoryboardGenerationRun `gorm:"foreignKey:RunID"`
	Kind         string                  `gorm:"size:80;not null;index"`
	AssetKey     string                  `gorm:"size:160;not null;index:idx_storyboard_generation_asset_run_key"`
	EntityID     string                  `gorm:"size:36;index"`
	ImageURL     string                  `gorm:"size:2048;not null"`
	Source       string                  `gorm:"size:80;index"`
	MetadataJSON string                  `gorm:"type:json"`
	CreatedAt    time.Time               `gorm:"autoCreateTime;index"`
	DeletedAt    gorm.DeletedAt          `gorm:"index"`
}

type AIPromptAuditRecord struct {
	ID                     string         `gorm:"primaryKey;size:36"`
	RunID                  string         `gorm:"size:36;index"`
	RelatedEntityType      string         `gorm:"size:80;index"`
	RelatedEntityID        string         `gorm:"size:36;index"`
	Step                   string         `gorm:"size:80;not null;index"`
	PromptKind             string         `gorm:"size:80;not null;index"`
	PromptTemplateVersion  string         `gorm:"size:80;index"`
	AlignmentSnapshotHash  string         `gorm:"size:64;index"`
	FullPromptHash         string         `gorm:"size:64;index"`
	Provider               string         `gorm:"size:80;index"`
	Model                  string         `gorm:"size:120;index"`
	Temperature            float64        `gorm:"default:0"`
	MaxTokens              int            `gorm:"default:0"`
	SystemPrompt           string         `gorm:"type:longtext"`
	UserPrompt             string         `gorm:"type:longtext"`
	AlignmentPrompt        string         `gorm:"type:longtext"`
	ReferencePreamble      string         `gorm:"type:longtext"`
	FinalPrompt            string         `gorm:"type:longtext"`
	ReferenceImageURLsJSON string         `gorm:"type:json"`
	Output                 string         `gorm:"type:longtext"`
	TokenUsageJSON         string         `gorm:"type:json"`
	MetadataJSON           string         `gorm:"type:json"`
	CreatedAt              time.Time      `gorm:"autoCreateTime;index"`
	DeletedAt              gorm.DeletedAt `gorm:"index"`
}
