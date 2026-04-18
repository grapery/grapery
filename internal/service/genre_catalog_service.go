package service

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/grapestree/fgrapery/grapery/internal/domain"
	"github.com/grapestree/fgrapery/grapery/internal/utils"
	"go.uber.org/zap"
)

const (
	genreCatalogGeminiModel = "gemini-2.5-flash"
	genreCatalogPageSize    = 12
	genreCatalogSourceAI    = "ai"
)

var genreSlugPattern = regexp.MustCompile(`^[a-z][a-z0-9_]{0,39}$`)

// GenreCatalogPageResult GET /settings/preferences/genres/catalog 响应体。
type GenreCatalogPageResult struct {
	Page      int                      `json:"page"`
	Items     []GenreCatalogItemPublic `json:"items"`
	Generated bool                     `json:"generated"`
	HasMore   bool                     `json:"hasMore"`
}

// GenreCatalogItemPublic API 展示的体裁项（中/日/英标题；客户端按用户语言择一展示）。
type GenreCatalogItemPublic struct {
	Slug    string `json:"slug"`
	TitleZh string `json:"titleZh"`
	TitleEn string `json:"titleEn"`
	TitleJa string `json:"titleJa"`
	Emoji   string `json:"emoji"`
}

// GenreCatalogService 分页加载目录；库中无该页时用 AI 生成并落库。
type GenreCatalogService struct {
	repo   domain.GenreCatalogRepository
	ai     *AIGenerationService
	logger *zap.Logger
}

// NewGenreCatalogService constructs GenreCatalogService.
func NewGenreCatalogService(repo domain.GenreCatalogRepository, ai *AIGenerationService, logger *zap.Logger) *GenreCatalogService {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &GenreCatalogService{repo: repo, ai: ai, logger: logger}
}

// FetchPage loads page `page` from DB, or generates via AI when empty.
func (s *GenreCatalogService) FetchPage(ctx context.Context, userID string, page int) (*GenreCatalogPageResult, error) {
	if page < 0 {
		page = 0
	}
	rows, err := s.repo.ListByPage(page)
	if err != nil {
		return nil, err
	}
	if len(rows) > 0 {
		return &GenreCatalogPageResult{
			Page:      page,
			Items:     toPublicItems(rows),
			Generated: false,
			HasMore:   true,
		}, nil
	}

	if s.ai == nil || !s.ai.GeminiAvailable() {
		return nil, fmt.Errorf("genre catalog AI generation is not available")
	}

	genErr := s.repo.WithGenerationLock(ctx, page, func() error {
		again, e := s.repo.ListByPage(page)
		if e != nil {
			return e
		}
		if len(again) > 0 {
			return nil
		}
		return s.generateAndInsertPage(ctx, userID, page)
	})
	if genErr != nil {
		return nil, genErr
	}

	rows, err = s.repo.ListByPage(page)
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, fmt.Errorf("genre catalog page %d is still empty after generation", page)
	}
	return &GenreCatalogPageResult{
		Page:      page,
		Items:     toPublicItems(rows),
		Generated: true,
		HasMore:   true,
	}, nil
}

func toPublicItems(rows []*domain.GenreCatalogEntry) []GenreCatalogItemPublic {
	out := make([]GenreCatalogItemPublic, 0, len(rows))
	for _, r := range rows {
		if r == nil {
			continue
		}
		out = append(out, GenreCatalogItemPublic{
			Slug:    r.Slug,
			TitleZh: r.TitleZh,
			TitleEn: r.TitleEn,
			TitleJa: r.TitleJa,
			Emoji:   r.Emoji,
		})
	}
	return out
}

type aiGenreJSON struct {
	Slug    string `json:"slug"`
	TitleZh string `json:"titleZh"`
	TitleEn string `json:"titleEn"`
	TitleJa string `json:"titleJa"`
	Emoji   string `json:"emoji"`
}

func (s *GenreCatalogService) generateAndInsertPage(ctx context.Context, userID string, page int) error {
	existing, err := s.repo.AllSlugs()
	if err != nil {
		return err
	}
	exclude := strings.Join(existing, ", ")
	prompt := fmt.Sprintf(`You are helping a multilingual mobile reading app (Simplified Chinese, Japanese, English) expand "story genre" tags for user onboarding and discovery feed preferences.

Return ONLY a JSON array (no markdown, no code fences) of exactly %d objects. Each object MUST include all four text fields:
{"slug":"english_snake_case","titleZh":"2-8 chars Simplified Chinese","titleEn":"concise English genre name","titleJa":"concise Japanese genre name (kanji/katakana/hiragana as natural for JP readers)","emoji":"single unicode emoji"}

Rules:
- slug: lowercase a-z, digits, underscore only; must start with a letter; max 40 chars; must be unique and NOT any of: [%s]
- titleZh: natural Simplified Chinese label (not Pinyin)
- titleEn: natural English label
- titleJa: natural Japanese label (NOT romaji; appropriate script for the genre)
- emoji: one character, relevant to the genre
- Genres should be creative and distinct from the excluded list; suitable for public UGC stories/fragments.`, genreCatalogPageSize, exclude)

	res, err := s.ai.GenerateText(ctx, &GenerateTextRequest{
		UserID:            userID,
		OriginalPrompt:    prompt,
		SystemPrompt:      "You output valid JSON only. Every object must include titleZh, titleEn, and titleJa. No explanation.",
		Model:             genreCatalogGeminiModel,
		Temperature:       0.9,
		MaxTokens:         3072,
		RelatedEntityID:   fmt.Sprintf("genre_catalog_page_%d", page),
		RelatedEntityType: "genre_catalog",
		Metadata: map[string]interface{}{
			"page": page,
		},
	})
	if err != nil {
		return fmt.Errorf("ai generate genre page: %w", err)
	}

	raw := extractJSONArray(res.Text)
	if raw == nil {
		return fmt.Errorf("ai genre response has no json array")
	}

	var parsed []aiGenreJSON
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return fmt.Errorf("parse ai genre json: %w", err)
	}

	seenSlug := make(map[string]struct{})
	for _, ex := range existing {
		seenSlug[strings.ToLower(strings.TrimSpace(ex))] = struct{}{}
	}

	now := time.Now().Unix()
	var batch []*domain.GenreCatalogEntry
	for i, g := range parsed {
		if len(batch) >= genreCatalogPageSize {
			break
		}
		slug := strings.ToLower(strings.TrimSpace(g.Slug))
		if !genreSlugPattern.MatchString(slug) {
			continue
		}
		if _, dup := seenSlug[slug]; dup {
			continue
		}
		seenSlug[slug] = struct{}{}
		titleZh := strings.TrimSpace(g.TitleZh)
		if titleZh == "" {
			continue
		}
		titleEn := strings.TrimSpace(g.TitleEn)
		titleJa := strings.TrimSpace(g.TitleJa)
		if titleJa == "" {
			titleJa = titleZh
		}
		emoji := strings.TrimSpace(g.Emoji)
		if emoji == "" {
			emoji = "📖"
		}
		// first grapheme only for emoji field
		if r := []rune(emoji); len(r) > 0 {
			emoji = string(r[0])
		}
		batch = append(batch, &domain.GenreCatalogEntry{
			ID:        utils.GenerateID(),
			Slug:      slug,
			PageIndex: page,
			SortOrder: i,
			TitleZh:   titleZh,
			TitleEn:   titleEn,
			TitleJa:   titleJa,
			Emoji:     emoji,
			Source:    genreCatalogSourceAI,
			CreatedAt: now,
		})
	}

	if len(batch) == 0 {
		return fmt.Errorf("ai produced no valid genre entries")
	}

	if err := s.repo.InsertBatch(batch); err != nil {
		return err
	}

	s.logger.Info("inserted AI genre catalog page",
		zap.Int("page", page),
		zap.Int("count", len(batch)),
		zap.String("userID", userID))
	return nil
}

func extractJSONArray(text string) []byte {
	s := strings.TrimSpace(text)
	i := strings.Index(s, "[")
	j := strings.LastIndex(s, "]")
	if i < 0 || j <= i {
		return nil
	}
	return []byte(s[i : j+1])
}
