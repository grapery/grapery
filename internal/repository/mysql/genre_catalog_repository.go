package mysql

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/grapestree/fgrapery/grapery/internal/domain"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// GenreCatalogRepositoryImpl implements domain.GenreCatalogRepository.
type GenreCatalogRepositoryImpl struct {
	db *gorm.DB
}

// NewGenreCatalogRepository creates a GenreCatalogRepository.
func NewGenreCatalogRepository(db *gorm.DB) domain.GenreCatalogRepository {
	return &GenreCatalogRepositoryImpl{db: db}
}

func genreLockName(pageIndex int) string {
	return fmt.Sprintf("genre_catalog_page_%d", pageIndex)
}

// ListByPage returns catalog rows for a page, ordered by sort_order.
func (r *GenreCatalogRepositoryImpl) ListByPage(pageIndex int) ([]*domain.GenreCatalogEntry, error) {
	var rows []GenreCatalogEntry
	if err := r.db.Where("page_index = ?", pageIndex).Order("sort_order ASC").Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("list genre catalog page: %w", err)
	}
	out := make([]*domain.GenreCatalogEntry, 0, len(rows))
	for i := range rows {
		out = append(out, genreModelToDomain(&rows[i]))
	}
	return out, nil
}

// InsertBatch inserts rows; ignores duplicate slugs (concurrent AI / retries).
func (r *GenreCatalogRepositoryImpl) InsertBatch(entries []*domain.GenreCatalogEntry) error {
	if len(entries) == 0 {
		return nil
	}
	for _, e := range entries {
		if e == nil {
			continue
		}
		m := genreDomainToModel(e)
		if err := r.db.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "slug"}},
			DoNothing: true,
		}).Create(m).Error; err != nil {
			return fmt.Errorf("insert genre catalog: %w", err)
		}
	}
	return nil
}

// AllSlugs returns every slug in stable order (page_index, sort_order).
func (r *GenreCatalogRepositoryImpl) AllSlugs() ([]string, error) {
	var slugs []string
	err := r.db.Model(&GenreCatalogEntry{}).
		Order("page_index ASC, sort_order ASC").
		Pluck("slug", &slugs).Error
	if err != nil {
		return nil, fmt.Errorf("list genre slugs: %w", err)
	}
	return slugs, nil
}

// CountByPage counts rows on a page.
func (r *GenreCatalogRepositoryImpl) CountByPage(pageIndex int) (int64, error) {
	var n int64
	if err := r.db.Model(&GenreCatalogEntry{}).Where("page_index = ?", pageIndex).Count(&n).Error; err != nil {
		return 0, err
	}
	return n, nil
}

// WithGenerationLock uses MySQL GET_LOCK to serialize AI page generation.
func (r *GenreCatalogRepositoryImpl) WithGenerationLock(ctx context.Context, pageIndex int, fn func() error) error {
	name := genreLockName(pageIndex)
	var lockRes int
	if err := r.db.WithContext(ctx).Raw("SELECT GET_LOCK(?, ?)", name, 25).Scan(&lockRes).Error; err != nil {
		return fmt.Errorf("get_lock: %w", err)
	}
	if lockRes != 1 {
		return errors.New("could not acquire genre catalog generation lock")
	}
	defer func() {
		var released int
		_ = r.db.WithContext(ctx).Raw("SELECT RELEASE_LOCK(?)", name).Scan(&released).Error
	}()

	return fn()
}

func genreModelToDomain(m *GenreCatalogEntry) *domain.GenreCatalogEntry {
	if m == nil {
		return nil
	}
	return &domain.GenreCatalogEntry{
		ID:        m.ID,
		Slug:      m.Slug,
		PageIndex: m.PageIndex,
		SortOrder: m.SortOrder,
		TitleZh:   m.TitleZh,
		TitleEn:   m.TitleEn,
		TitleJa:   m.TitleJa,
		Emoji:     m.Emoji,
		Source:    m.Source,
		CreatedAt: m.CreatedAt,
	}
}

func genreDomainToModel(d *domain.GenreCatalogEntry) *GenreCatalogEntry {
	if d == nil {
		return nil
	}
	return &GenreCatalogEntry{
		ID:        d.ID,
		Slug:      d.Slug,
		PageIndex: d.PageIndex,
		SortOrder: d.SortOrder,
		TitleZh:   d.TitleZh,
		TitleEn:   d.TitleEn,
		TitleJa:   d.TitleJa,
		Emoji:     d.Emoji,
		Source:    d.Source,
		CreatedAt: d.CreatedAt,
	}
}

// SeedGenreCatalogPage0IfEmpty inserts the canonical first page when the table is empty.
func SeedGenreCatalogPage0IfEmpty(db *gorm.DB, log func(string, ...interface{})) error {
	var n int64
	if err := db.Model(&GenreCatalogEntry{}).Count(&n).Error; err != nil {
		return err
	}
	if n > 0 {
		return nil
	}
	now := time.Now().Unix()
	rows := []GenreCatalogEntry{
		{ID: "genre_seed_scifi", Slug: "scifi", PageIndex: 0, SortOrder: 0, TitleZh: "科幻", TitleEn: "Sci-Fi", TitleJa: "SF", Emoji: "🚀", Source: "seed", CreatedAt: now},
		{ID: "genre_seed_romance", Slug: "romance", PageIndex: 0, SortOrder: 1, TitleZh: "爱情", TitleEn: "Romance", TitleJa: "恋愛", Emoji: "💕", Source: "seed", CreatedAt: now},
		{ID: "genre_seed_mystery", Slug: "mystery", PageIndex: 0, SortOrder: 2, TitleZh: "悬疑", TitleEn: "Mystery", TitleJa: "ミステリー", Emoji: "🔍", Source: "seed", CreatedAt: now},
		{ID: "genre_seed_fantasy", Slug: "fantasy", PageIndex: 0, SortOrder: 3, TitleZh: "奇幻", TitleEn: "Fantasy", TitleJa: "ファンタジー", Emoji: "🧙", Source: "seed", CreatedAt: now},
		{ID: "genre_seed_youth", Slug: "youth", PageIndex: 0, SortOrder: 4, TitleZh: "青春", TitleEn: "Youth", TitleJa: "青春", Emoji: "🌸", Source: "seed", CreatedAt: now},
		{ID: "genre_seed_history", Slug: "history", PageIndex: 0, SortOrder: 5, TitleZh: "历史", TitleEn: "History", TitleJa: "歴史", Emoji: "📜", Source: "seed", CreatedAt: now},
		{ID: "genre_seed_urban", Slug: "urban", PageIndex: 0, SortOrder: 6, TitleZh: "都市", TitleEn: "Urban", TitleJa: "都市", Emoji: "🏙️", Source: "seed", CreatedAt: now},
		{ID: "genre_seed_comedy", Slug: "comedy", PageIndex: 0, SortOrder: 7, TitleZh: "搞笑", TitleEn: "Comedy", TitleJa: "コメディ", Emoji: "😄", Source: "seed", CreatedAt: now},
		{ID: "genre_seed_healing", Slug: "healing", PageIndex: 0, SortOrder: 8, TitleZh: "治愈", TitleEn: "Healing", TitleJa: "ヒーリング", Emoji: "🌿", Source: "seed", CreatedAt: now},
		{ID: "genre_seed_horror", Slug: "horror", PageIndex: 0, SortOrder: 9, TitleZh: "惊悚", TitleEn: "Horror", TitleJa: "ホラー", Emoji: "👻", Source: "seed", CreatedAt: now},
		{ID: "genre_seed_adventure", Slug: "adventure", PageIndex: 0, SortOrder: 10, TitleZh: "冒险", TitleEn: "Adventure", TitleJa: "冒険", Emoji: "⚔️", Source: "seed", CreatedAt: now},
		{ID: "genre_seed_slice", Slug: "slice", PageIndex: 0, SortOrder: 11, TitleZh: "日常", TitleEn: "Slice of Life", TitleJa: "日常", Emoji: "☕", Source: "seed", CreatedAt: now},
	}
	if err := db.Create(&rows).Error; err != nil {
		return fmt.Errorf("seed genre catalog page 0: %w", err)
	}
	if log != nil {
		log("seeded genre_catalog_entries page 0 (%d rows)", len(rows))
	}
	return nil
}

// BackfillGenreCatalogTitleJa adds Japanese titles for known seed slugs and fills any remaining empty title_ja from title_zh (legacy / AI rows).
func BackfillGenreCatalogTitleJa(db *gorm.DB) error {
	if db == nil {
		return nil
	}
	migrator := db.Migrator()
	if !migrator.HasTable(&GenreCatalogEntry{}) || !migrator.HasColumn(&GenreCatalogEntry{}, "title_ja") {
		return nil
	}
	if err := db.Exec(`
		UPDATE genre_catalog_entries
		SET title_ja = title_zh
		WHERE (title_ja IS NULL OR title_ja = '') AND title_zh <> ''
	`).Error; err != nil {
		return fmt.Errorf("backfill title_ja from title_zh: %w", err)
	}
	seedJP := map[string]string{
		"scifi":     "SF",
		"romance":   "恋愛",
		"mystery":   "ミステリー",
		"fantasy":   "ファンタジー",
		"youth":     "青春",
		"history":   "歴史",
		"urban":     "都市",
		"comedy":    "コメディ",
		"healing":   "ヒーリング",
		"horror":    "ホラー",
		"adventure": "冒険",
		"slice":     "日常",
	}
	for slug, ja := range seedJP {
		if err := db.Model(&GenreCatalogEntry{}).Where("slug = ?", slug).Update("title_ja", ja).Error; err != nil {
			return fmt.Errorf("backfill title_ja for %s: %w", slug, err)
		}
	}
	return nil
}
