package mysql

// FragmentPanelGenerationTaskDB 多图故事碎片生成任务
type FragmentPanelGenerationTaskDB struct {
	ID               string `gorm:"primaryKey;size:36"`
	UserID           string `gorm:"size:36;not null;index:idx_fp_gen_user"`
	Status           string `gorm:"size:20;not null;default:'pending';index:idx_fp_gen_status"`
	RequestJSON      string `gorm:"type:text;not null"`
	PlanJSON         string `gorm:"type:text"`
	ResultJSON       string `gorm:"type:text"`
	MetricsJSON      string `gorm:"type:text"`
	Progress         int    `gorm:"type:int;default:0"`
	CurrentStep      string `gorm:"size:80"`
	ErrorMessage     string `gorm:"type:text"`
	DraftFragmentID  string `gorm:"size:36;index:idx_fp_gen_draft"`
	CreatedAt        int64  `gorm:"type:bigint;autoCreateTime;index:idx_fp_gen_created"`
	StartedAt        *int64 `gorm:"type:bigint"`
	CompletedAt      *int64 `gorm:"type:bigint"`
	UpdatedAt        int64  `gorm:"type:bigint;autoUpdateTime"`

	User User `gorm:"foreignKey:UserID"`
}

func (FragmentPanelGenerationTaskDB) TableName() string {
	return "fragment_panel_generation_tasks"
}
