package mysql

import "time"

type WorkflowReleaseDB struct {
	ID               string    `gorm:"primaryKey;size:80"`
	WorkflowKey      string    `gorm:"size:120;not null;uniqueIndex:idx_workflow_release_version,priority:1"`
	Version          int       `gorm:"not null;uniqueIndex:idx_workflow_release_version,priority:2"`
	Name             string    `gorm:"size:200;not null"`
	Description      string    `gorm:"type:text"`
	Status           string    `gorm:"size:24;not null;index"`
	ManifestJSON     string    `gorm:"type:longtext"`
	DefinitionJSON   string    `gorm:"type:longtext;not null"`
	PromptBundleJSON string    `gorm:"type:longtext"`
	PoliciesJSON     string    `gorm:"type:text"`
	Checksum         string    `gorm:"size:64;not null;index"`
	CreatedBy        string    `gorm:"size:64"`
	ApprovedByJSON   string    `gorm:"type:text"`
	PublishedAt      time.Time `gorm:"index"`
	CreatedAt        time.Time `gorm:"autoCreateTime"`
}

func (WorkflowReleaseDB) TableName() string { return "workflow_releases" }

type PromptTemplateVersionDB struct {
	ID                  string    `gorm:"primaryKey;size:100"`
	PromptKey           string    `gorm:"size:140;not null;uniqueIndex:idx_prompt_template_version,priority:1"`
	Version             int       `gorm:"not null;uniqueIndex:idx_prompt_template_version,priority:2"`
	Type                string    `gorm:"size:24;not null"`
	SystemTemplate      string    `gorm:"type:longtext"`
	UserTemplate        string    `gorm:"type:longtext"`
	VariablesSchemaJSON string    `gorm:"type:longtext"`
	OutputSchemaJSON    string    `gorm:"type:longtext"`
	ModelConfigJSON     string    `gorm:"type:text"`
	Checksum            string    `gorm:"size:64;not null;index"`
	CreatedBy           string    `gorm:"size:64"`
	CreatedAt           time.Time `gorm:"autoCreateTime"`
}

func (PromptTemplateVersionDB) TableName() string { return "prompt_template_versions" }

type WorkflowBindingDB struct {
	ID             string    `gorm:"primaryKey;size:80"`
	Surface        string    `gorm:"size:140;not null;index:idx_workflow_binding_lookup,priority:1"`
	Action         string    `gorm:"size:140;not null;index:idx_workflow_binding_lookup,priority:2"`
	TenantID       string    `gorm:"size:64;index:idx_workflow_binding_lookup,priority:3"`
	WorkflowKey    string    `gorm:"size:120;not null;index"`
	ReleaseID      string    `gorm:"size:80;not null;index"`
	Priority       int       `gorm:"not null;default:0;index:idx_workflow_binding_lookup,priority:4"`
	Enabled        bool      `gorm:"not null;default:true;index"`
	ConditionsJSON string    `gorm:"type:text"`
	CreatedBy      string    `gorm:"size:64"`
	CreatedAt      time.Time `gorm:"autoCreateTime"`
	UpdatedAt      time.Time `gorm:"autoUpdateTime"`
}

func (WorkflowBindingDB) TableName() string { return "workflow_bindings" }
