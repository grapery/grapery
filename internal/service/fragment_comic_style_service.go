package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/grapestree/fgrapery/grapery/internal/repository/mysql"
)

const fragmentComicStyleBatchSize = 8

var fragmentComicStyleSlugRe = regexp.MustCompile(`^[a-z][a-z0-9_]{1,62}$`)

// FragmentComicStyleItem is returned to HTTP layer (maps to FragmentStyle JSON).
type FragmentComicStyleItem struct {
	ID       string
	Value    string
	Name     string
	Icon     string
	Category *string
}

// FragmentComicStyleService serves paginated comic style batches per user cursor.
type FragmentComicStyleService struct {
	db     *gorm.DB
	aiGen  *AIGenerationService
	logger *zap.Logger
}

func NewFragmentComicStyleService(db *gorm.DB, aiGen *AIGenerationService, logger *zap.Logger) *FragmentComicStyleService {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &FragmentComicStyleService{
		db:     db,
		aiGen:  aiGen,
		logger: logger,
	}
}

// NextBatch returns the next fragmentComicStyleBatchSize styles for the user, advancing the cursor.
func (s *FragmentComicStyleService) NextBatch(ctx context.Context, userID string) ([]FragmentComicStyleItem, error) {
	if userID == "" {
		return nil, errors.New("userID required")
	}
	if s.db == nil {
		return nil, errors.New("database not configured")
	}

	var out []FragmentComicStyleItem

	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var cur mysql.UserFragmentComicStyleCursor
		err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("user_id = ?", userID).
			First(&cur).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			cur = mysql.UserFragmentComicStyleCursor{UserID: userID, LastStyleID: 0}
			if err := tx.Create(&cur).Error; err != nil {
				return fmt.Errorf("create style cursor: %w", err)
			}
		} else if err != nil {
			return fmt.Errorf("load style cursor: %w", err)
		}

		lastID := cur.LastStyleID
		const maxRounds = 6
		var lastGenErr error

		for round := 0; round < maxRounds; round++ {
			var batch []mysql.FragmentComicStyle
			if err := tx.Where("id > ?", lastID).Order("id ASC").Limit(fragmentComicStyleBatchSize).Find(&batch).Error; err != nil {
				return err
			}

			if len(batch) >= fragmentComicStyleBatchSize {
				out = toStyleItems(batch)
				maxID := batch[fragmentComicStyleBatchSize-1].ID
				return tx.Model(&mysql.UserFragmentComicStyleCursor{}).
					Where("user_id = ?", userID).
					Update("last_style_id", maxID).Error
			}

			need := fragmentComicStyleBatchSize - len(batch)
			if s.aiGen == nil {
				if len(batch) == 0 {
					return errors.New("Gemini client not configured and no styles in database")
				}
				s.logger.Warn("AI generation unavailable; returning partial comic style batch",
					zap.Int("count", len(batch)))
				out = toStyleItems(batch)
				maxID := batch[len(batch)-1].ID
				return tx.Model(&mysql.UserFragmentComicStyleCursor{}).
					Where("user_id = ?", userID).
					Update("last_style_id", maxID).Error
			}

			inserted, genErr := s.generateAndInsertStyles(ctx, tx, userID, need)
			if genErr != nil {
				lastGenErr = genErr
				s.logger.Warn("AI comic style generation failed", zap.Int("round", round+1), zap.Error(genErr))
			}
			if inserted == 0 {
				if len(batch) > 0 {
					out = toStyleItems(batch)
					maxID := batch[len(batch)-1].ID
					return tx.Model(&mysql.UserFragmentComicStyleCursor{}).
						Where("user_id = ?", userID).
						Update("last_style_id", maxID).Error
				}
				if genErr != nil {
					return fmt.Errorf("generate comic styles: %w", genErr)
				}
				return errors.New("AI returned no valid comic styles")
			}
			// More rows may now exist with id > lastID; loop and re-query.
		}

		if lastGenErr != nil {
			return fmt.Errorf("could not fill comic style batch: %w", lastGenErr)
		}
		return errors.New("could not fill comic style batch")
	})

	return out, err
}

func toStyleItems(rows []mysql.FragmentComicStyle) []FragmentComicStyleItem {
	out := make([]FragmentComicStyleItem, 0, len(rows))
	for _, r := range rows {
		item := FragmentComicStyleItem{
			ID:    r.PublicID,
			Value: r.Value,
			Name:  r.Name,
			Icon:  r.Icon,
		}
		if r.Category != "" {
			c := r.Category
			item.Category = &c
		}
		out = append(out, item)
	}
	return out
}

type aiComicStyleRow struct {
	Value    string  `json:"value"`
	Name     string  `json:"name"`
	Icon     *string `json:"icon"`
	Category *string `json:"category"`
}

func (s *FragmentComicStyleService) generateAndInsertStyles(ctx context.Context, tx *gorm.DB, userID string, count int) (inserted int, err error) {
	if count <= 0 {
		return 0, nil
	}
	var existing []string
	if err := tx.Model(&mysql.FragmentComicStyle{}).Order("id DESC").Limit(300).Pluck("value", &existing).Error; err != nil {
		return 0, err
	}
	existingSet := make(map[string]struct{}, len(existing))
	for _, v := range existing {
		existingSet[strings.ToLower(v)] = struct{}{}
	}

	prompt := buildComicStyleAIPrompt(count, existing)
	var lastErr error
	for attempt := 0; attempt < 2; attempt++ {
		result, err := s.aiGen.GenerateText(ctx, &GenerateTextRequest{
			UserID:         userID,
			OriginalPrompt: prompt,
			SystemPrompt: `你是漫画与插画视觉风格命名助手。只输出 JSON，不要 markdown，不要解释。
输出为 JSON 数组，每项字段：value（必填，英文小写 slug，仅 a-z 数字下划线，2-40 字符）、name（必填，中文短名 2-12 字）、icon（可选，SF Symbol 名如 sparkles）、category（可选，英文小词如 illustration）。
风格要有辨识度且适合「故事碎片」配图，避免与已有 value 重复。`,
			Model:             "",
			Temperature:       0.55,
			MaxTokens:         2048,
			RelatedEntityType: "fragment_comic_style",
			Metadata: map[string]interface{}{
				"operation": "fragment_comic_style_batch",
				"count":     count,
			},
		})
		if err != nil {
			lastErr = err
			continue
		}
		rows, perr := parseAIComicStyleJSON(result.Text, count)
		if perr != nil {
			lastErr = perr
			continue
		}
		for _, row := range rows {
			v := strings.ToLower(strings.TrimSpace(row.Value))
			n := strings.TrimSpace(row.Name)
			if !fragmentComicStyleSlugRe.MatchString(v) || n == "" {
				continue
			}
			if _, dup := existingSet[v]; dup {
				continue
			}
			icon := ""
			if row.Icon != nil {
				icon = strings.TrimSpace(*row.Icon)
			}
			cat := ""
			if row.Category != nil {
				cat = strings.TrimSpace(*row.Category)
			}
			rec := mysql.FragmentComicStyle{
				PublicID: uuid.New().String(),
				Value:    v,
				Name:     n,
				Icon:     icon,
				Category: cat,
				Source:   "ai",
			}
			if err := tx.Create(&rec).Error; err != nil {
				s.logger.Debug("skip duplicate comic style insert", zap.String("value", v), zap.Error(err))
				continue
			}
			existingSet[v] = struct{}{}
			inserted++
			if inserted >= count {
				break
			}
		}
		if inserted > 0 {
			return inserted, nil
		}
	}
	if lastErr != nil {
		return 0, lastErr
	}
	return 0, errors.New("no rows inserted from AI response")
}

func buildComicStyleAIPrompt(count int, existing []string) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("请生成恰好 %d 个**新的**漫画/插画视觉风格选项。\n", count))
	if len(existing) > 0 {
		b.WriteString("数据库中已存在的 value（请勿重复）：\n")
		max := len(existing)
		if max > 120 {
			max = 120
		}
		b.WriteString(strings.Join(existing[:max], ", "))
		b.WriteString("\n")
	}
	b.WriteString("只输出 JSON 数组，无其它文本。")
	return b.String()
}

func parseAIComicStyleJSON(text string, maxWant int) ([]aiComicStyleRow, error) {
	s := strings.TrimSpace(text)
	s = strings.TrimPrefix(s, "```json")
	s = strings.TrimPrefix(s, "```JSON")
	s = strings.TrimPrefix(s, "```")
	s = strings.TrimSpace(s)
	if i := strings.LastIndex(s, "```"); i >= 0 {
		s = strings.TrimSpace(s[:i])
	}
	start := strings.Index(s, "[")
	end := strings.LastIndex(s, "]")
	if start < 0 || end <= start {
		return nil, errors.New("no JSON array in AI output")
	}
	s = s[start : end+1]
	var rows []aiComicStyleRow
	if err := json.Unmarshal([]byte(s), &rows); err != nil {
		return nil, fmt.Errorf("json decode: %w", err)
	}
	if len(rows) > maxWant*2 {
		rows = rows[:maxWant*2]
	}
	return rows, nil
}
