package mysql

import (
	"time"

	"gorm.io/gorm"
)

// GenerationStepAuditRecordDB agent 生成步骤审计（与 grapery-agent domain.GenerationStepAudit 对齐）。
type GenerationStepAuditRecordDB struct {
	ID               string         `gorm:"primaryKey;size:64"`
	Sequence         int            `gorm:"not null;default:0"`
	RunID            string         `gorm:"size:64;not null;index"`
	TaskID           string         `gorm:"size:64;index"`
	BusinessType     string         `gorm:"size:40;index"`
	BusinessID       string         `gorm:"size:64;index"`
	AgentVersion     string         `gorm:"size:80;index"`
	StepName         string         `gorm:"size:120;not null;index"`
	Attempt          int            `gorm:"not null;default:1"`
	Status           string         `gorm:"size:20;not null;index"`
	Provider         string         `gorm:"size:80"`
	Model            string         `gorm:"size:120"`
	Prompt           string         `gorm:"type:longtext"`
	InputRefsJSON    string         `gorm:"type:json"`
	RawOutput        string         `gorm:"type:longtext"`
	ParsedOutputJSON string         `gorm:"type:json"`
	ErrorCode        string         `gorm:"size:80"`
	ErrorMessage     string         `gorm:"type:text"`
	InputTokens      int            `gorm:"default:0"`
	OutputTokens     int            `gorm:"default:0"`
	TotalTokens      int            `gorm:"default:0"`
	DurationMs       int64          `gorm:"default:0"`
	StartedAt        *time.Time     `gorm:"index"`
	EndedAt          *time.Time
	MetadataJSON     string         `gorm:"type:json"`
	CreatedAt        time.Time      `gorm:"autoCreateTime;index"`
	UserID           string         `gorm:"size:36;index"`
	DeletedAt        gorm.DeletedAt `gorm:"index"`
}

func (GenerationStepAuditRecordDB) TableName() string {
	return "generation_step_audit_records"
}
