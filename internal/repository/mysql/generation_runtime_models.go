package mysql

import "time"

type GenerationExecutionDB struct {
	ID                string     `gorm:"primaryKey;size:64"`
	UserID            string     `gorm:"size:36;index"`
	Kind              string     `gorm:"size:40;not null;index;uniqueIndex:idx_generation_execution_idempotency,priority:2"`
	Status            string     `gorm:"size:24;not null;index"`
	Phase             string     `gorm:"size:80;index"`
	Progress          int        `gorm:"default:0"`
	AgentVersion      string     `gorm:"size:120;index"`
	UserIntent        string     `gorm:"type:text"`
	InputJSON         string     `gorm:"type:longtext"`
	OutputJSON        string     `gorm:"type:longtext"`
	ParentRunID       string     `gorm:"size:64;index"`
	BranchIndex       int        `gorm:"default:0"`
	Strategy          string     `gorm:"size:120"`
	ContentIDsJSON    string     `gorm:"type:text"`
	ToolCallsJSON     string     `gorm:"type:longtext"`
	Error             string     `gorm:"type:text"`
	ErrorCode         string     `gorm:"size:80;index"`
	TokensUsed        int        `gorm:"default:0"`
	ModelProvider     string     `gorm:"size:80"`
	ModelName         string     `gorm:"size:160"`
	CheckpointID      string     `gorm:"size:160;index"`
	ClientRequestID   *string    `gorm:"size:160;uniqueIndex:idx_generation_execution_idempotency,priority:3"`
	IdempotencyUserID *string    `gorm:"size:36;uniqueIndex:idx_generation_execution_idempotency,priority:1"`
	SourceTaskID      string     `gorm:"size:64;index"`
	Sequence          int64      `gorm:"not null;default:0"`
	CreatedAt         time.Time  `gorm:"autoCreateTime;index"`
	UpdatedAt         time.Time  `gorm:"autoUpdateTime"`
	CompletedAt       *time.Time `gorm:"index"`
}

func (GenerationExecutionDB) TableName() string { return "generation_executions" }

type GenerationEventDB struct {
	ID          string     `gorm:"primaryKey;size:64"`
	RunID       string     `gorm:"size:64;not null;uniqueIndex:idx_generation_event_sequence,priority:1"`
	Sequence    int64      `gorm:"not null;uniqueIndex:idx_generation_event_sequence,priority:2"`
	Type        string     `gorm:"size:80;not null;index"`
	PayloadJSON string     `gorm:"type:longtext"`
	PublishedAt *time.Time `gorm:"index"`
	CreatedAt   time.Time  `gorm:"autoCreateTime;index"`
}

func (GenerationEventDB) TableName() string { return "generation_events" }

type GenerationCheckpointDB struct {
	ID        string    `gorm:"primaryKey;size:160"`
	State     []byte    `gorm:"type:longblob;not null"`
	ExpiresAt time.Time `gorm:"index"`
	CreatedAt time.Time `gorm:"autoCreateTime"`
	UpdatedAt time.Time `gorm:"autoUpdateTime"`
}

func (GenerationCheckpointDB) TableName() string { return "generation_checkpoints" }
