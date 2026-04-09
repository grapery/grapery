package mysql

import (
	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// FragmentComicStyle 全局漫画/插画风格池（碎片创作 UI）
type FragmentComicStyle struct {
	ID        int64  `gorm:"primaryKey;autoIncrement"`
	PublicID  string `gorm:"size:36;uniqueIndex;not null"`
	Value     string `gorm:"size:64;uniqueIndex;not null"`
	Name      string `gorm:"size:64;not null"`
	Icon      string `gorm:"size:128"`
	Category  string `gorm:"size:64"`
	Source    string `gorm:"size:12;not null;default:seed"` // seed | ai
	CreatedAt int64  `gorm:"autoCreateTime"`
}

func (FragmentComicStyle) TableName() string { return "fragment_comic_styles" }

// UserFragmentComicStyleCursor 每用户已消费到的最大 fragment_comic_styles.id
type UserFragmentComicStyleCursor struct {
	UserID      string `gorm:"primaryKey;size:36"`
	LastStyleID int64  `gorm:"not null;default:0"`
	UpdatedAt   int64  `gorm:"autoUpdateTime"`
}

func (UserFragmentComicStyleCursor) TableName() string { return "user_fragment_comic_style_cursors" }

type fragmentComicStyleSeed struct {
	Value    string
	Name     string
	Icon     string
	Category string
}

// SeedFragmentComicStylesIfEmpty inserts initial styles when table is empty.
func SeedFragmentComicStylesIfEmpty(db *gorm.DB, log *zap.Logger) error {
	var n int64
	if err := db.Model(&FragmentComicStyle{}).Count(&n).Error; err != nil {
		return err
	}
	if n > 0 {
		return nil
	}
	seeds := []fragmentComicStyleSeed{
		{"anime", "日系动漫", "sparkles", "illustration"},
		{"cel_shading", "赛璐璐", "paintbrush.pointed", "illustration"},
		{"chibi", "Q版", "face.smiling", "illustration"},
		{"manga", "漫画", "book.closed", "illustration"},
		{"digital_painting", "数字绘画", "paintbrush", "digital"},
		{"concept_art", "概念艺术", "lightbulb", "digital"},
		{"cyberpunk", "赛博朋克", "brain.head.profile", "scifi"},
		{"ink_punk", "水墨朋克", "drop", "artistic"},
		{"3d_render", "3D渲染", "cube", "3d"},
		{"synthetic_impressionism", "合成印象派", "scribble", "artistic"},
		{"pop_surrealism", "流行超现实主义", "theatermasks", "surreal"},
		{"vaporwave", "蒸汽波美学", "waveform.path", "aesthetic"},
	}
	for _, s := range seeds {
		row := FragmentComicStyle{
			PublicID: uuid.New().String(),
			Value:    s.Value,
			Name:     s.Name,
			Icon:     s.Icon,
			Category: s.Category,
			Source:   "seed",
		}
		if err := db.Create(&row).Error; err != nil {
			if log != nil {
				log.Warn("seed fragment_comic_styles row failed", zap.String("value", s.Value), zap.Error(err))
			}
			continue
		}
	}
	if log != nil {
		log.Info("seeded fragment_comic_styles", zap.Int("count", len(seeds)))
	}
	return nil
}
