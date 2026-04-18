package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/grapestree/fgrapery/grapery/internal/repository/mysql"
)

const fragmentComicStyleBatchSize = 8

// fragmentComicStyleOpTimeout 漫画风格批次含 Gemini 调用，耗时常超过网关/客户端默认超时。
// 使用 WithoutCancel + 本超时，避免 c.Request.Context 在客户端断开后取消 DB/AI 导致 context canceled。
const fragmentComicStyleOpTimeout = 4 * time.Minute

var fragmentComicStyleSlugRe = regexp.MustCompile(`^[a-z][a-z0-9_]{1,62}$`)

// FragmentComicStyleItem is returned to HTTP layer (maps to FragmentStyle JSON).
type FragmentComicStyleItem struct {
	ID       string
	Value    string
	Name     string
	Icon     string
	Category *string
}

// FragmentComicStyleService reads the global fragment_comic_styles pool (no per-user cursor).
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

// NextBatch returns the first fragmentComicStyleBatchSize rows by id (global catalog head).
// If the table is empty, inserts via AI (when configured) and retries until rows exist or attempts exhaust.
func (s *FragmentComicStyleService) NextBatch(ctx context.Context, userID string) ([]FragmentComicStyleItem, error) {
	if userID == "" {
		return nil, errors.New("userID required")
	}
	if s.db == nil {
		return nil, errors.New("database not configured")
	}

	// 不随 HTTP 请求取消：否则客户端超时/断连会取消 Gemini 与 gorm，日志见 context canceled。
	opCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), fragmentComicStyleOpTimeout)
	defer cancel()

	var out []FragmentComicStyleItem

	err := s.db.WithContext(opCtx).Transaction(func(tx *gorm.DB) error {
		const maxRounds = 6
		var lastGenErr error

		for round := 0; round < maxRounds; round++ {
			var batch []mysql.FragmentComicStyle
			if err := tx.Order("id ASC").Limit(fragmentComicStyleBatchSize).Find(&batch).Error; err != nil {
				return err
			}

			if len(batch) >= fragmentComicStyleBatchSize {
				out = toStyleItems(batch)
				return nil
			}

			if len(batch) > 0 {
				out = toStyleItems(batch)
				return nil
			}

			if s.aiGen == nil {
				return errors.New("Gemini client not configured and no styles in database")
			}

			inserted, genErr := s.generateAndInsertStyles(opCtx, tx, userID, fragmentComicStyleBatchSize)
			if genErr != nil {
				lastGenErr = genErr
				s.logger.Warn("AI comic style generation failed", zap.Int("round", round+1), zap.Error(genErr))
			}
			if inserted == 0 {
				if genErr != nil {
					return fmt.Errorf("generate comic styles: %w", genErr)
				}
				return errors.New("AI returned no valid comic styles")
			}
		}

		if lastGenErr != nil {
			return fmt.Errorf("could not fill comic style batch: %w", lastGenErr)
		}
		return errors.New("could not fill comic style batch")
	})

	return out, err
}

// NextBatchDBOnly returns the first fragmentComicStyleBatchSize rows by id without calling AI.
func (s *FragmentComicStyleService) NextBatchDBOnly(ctx context.Context, userID string) ([]FragmentComicStyleItem, error) {
	if userID == "" {
		return nil, errors.New("userID required")
	}
	if s.db == nil {
		return nil, errors.New("database not configured")
	}
	opCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 30*time.Second)
	defer cancel()

	var batch []mysql.FragmentComicStyle
	if err := s.db.WithContext(opCtx).Order("id ASC").Limit(fragmentComicStyleBatchSize).Find(&batch).Error; err != nil {
		return nil, err
	}
	return toStyleItems(batch), nil
}

// LookupByValue 按碎片表 `style` 字段存储的 value slug 查询全局目录 fragment_comic_styles（与创作页 `PostFragmentStylesNext` 同源）。
// 未命中或 value 为空时返回 (nil, nil)；仅数据库错误时返回 err。
func (s *FragmentComicStyleService) LookupByValue(ctx context.Context, value string) (*FragmentComicStyleItem, error) {
	v := strings.TrimSpace(value)
	if v == "" || s.db == nil {
		return nil, nil
	}
	var row mysql.FragmentComicStyle
	err := s.db.WithContext(ctx).Where("LOWER(value) = ?", strings.ToLower(v)).First(&row).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	items := toStyleItems([]mysql.FragmentComicStyle{row})
	if len(items) == 0 {
		return nil, nil
	}
	out := items[0]
	return &out, nil
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
