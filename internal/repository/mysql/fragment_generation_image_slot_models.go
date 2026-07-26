package mysql

// FragmentGenerationImageSlotDB stores the durable per-image state for a fragment generation task.
type FragmentGenerationImageSlotDB struct {
	ID           string `gorm:"primaryKey;size:36"`
	TaskID       string `gorm:"size:36;not null;index:idx_fgis_task;index:idx_fgis_task_index,unique"`
	FragmentID   string `gorm:"size:36;index:idx_fgis_fragment"`
	Index        int    `gorm:"column:slot_index;type:int;not null;index:idx_fgis_task_index,unique"`
	Title        string `gorm:"size:80"`
	Caption      string `gorm:"type:text"`
	Status       string `gorm:"size:24;not null;default:'planned';index"`
	ImageURL     string `gorm:"type:text"`
	AssetID      string `gorm:"size:36;index"`
	ErrorMessage string `gorm:"type:text"`
	MetadataJSON string `gorm:"column:metadata;type:json"`
	CreatedAt    int64  `gorm:"type:bigint;autoCreateTime;index"`
	UpdatedAt    int64  `gorm:"type:bigint;autoUpdateTime"`
}

func (FragmentGenerationImageSlotDB) TableName() string {
	return "fragment_generation_image_slots"
}
