package domain

// GenreCatalogEntry 发现页体裁目录（分页存储；可种子数据或 AI 生成）。
type GenreCatalogEntry struct {
	ID        string
	Slug      string
	PageIndex int
	SortOrder int
	TitleZh   string
	TitleEn   string
	TitleJa   string
	Emoji     string
	Source    string // seed | ai
	CreatedAt int64
}
