package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"unicode/utf8"

	"go.uber.org/zap"

	"github.com/grapestree/fgrapery/grapery/internal/domain"
)

// 与 voyager StoryEditorView.allGenres 一致
var allowedStoryPrefillGenres = []string{
	"奇幻", "科幻", "悬疑", "爱情", "武侠", "历史",
	"都市", "恐怖", "冒险", "喜剧", "青春", "其他",
}

func truncateRunes(s string, max int) string {
	if max <= 0 {
		return ""
	}
	runes := []rune(strings.TrimSpace(s))
	if len(runes) <= max {
		return string(runes)
	}
	return string(runes[:max])
}

func normalizePrefillGenre(s string) string {
	s = strings.TrimSpace(s)
	for _, a := range allowedStoryPrefillGenres {
		if s == a {
			return a
		}
	}
	return "其他"
}

func clampPrefillTags(tags []string, maxN int) []string {
	if maxN <= 0 {
		return nil
	}
	var out []string
	for _, t := range tags {
		t = strings.TrimSpace(t)
		if t == "" {
			continue
		}
		if utf8.RuneCountInString(t) > 20 {
			t = truncateRunes(t, 20)
		}
		out = append(out, t)
		if len(out) >= maxN {
			break
		}
	}
	return out
}

var jsonFenceRE = regexp.MustCompile(`(?s)` + "```(?:json)?\\s*([\\s\\S]*?)```")

func extractJSONFromModelText(text string) string {
	text = strings.TrimSpace(text)
	if m := jsonFenceRE.FindStringSubmatch(text); len(m) > 1 {
		return strings.TrimSpace(m[1])
	}
	return text
}

type fragmentPrefillAIJSON struct {
	Title               string                   `json:"title"`
	Description         string                   `json:"description"`
	Summary             string                   `json:"summary"`
	Style               string                   `json:"style"`
	Genre               string                   `json:"genre"`
	Tags                []string                 `json:"tags"`
	SuggestedCharacters []fragmentPrefillCharRaw `json:"suggestedCharacters"`
}

type fragmentPrefillCharRaw struct {
	Name       string `json:"name"`
	Role       string `json:"role"`
	Background string `json:"background"`
}

// resolveFragmentStyleForPrefillPrompt 按碎片 DB 的 style（与创作页提交的 value slug 一致）查 fragment_comic_styles，与 styles/next 同源；返回提示词段落与 API 应返回的 style 字符串（可能为空表示仅用模型输出）。
func (s *Service) resolveFragmentStyleForPrefillPrompt(ctx context.Context, fragment *domain.Fragment) (promptParagraph string, fixedResponseStyle string, useAIFallback bool) {
	var raw string
	if fragment != nil && fragment.Style != nil {
		raw = strings.TrimSpace(*fragment.Style)
	}
	if raw == "" {
		return "【碎片创作风格】未记录：用户未在创作页选择风格或旧数据无该字段。JSON 的 style 请用简短中文概括画面/叙事气质（不超过 16 字）。",
			"", true
	}
	if s.comicStyleSvc == nil {
		return fmt.Sprintf("【碎片创作风格】数据库 style 原始值为 %q。服务端未加载全局风格目录时，JSON 的 style 请基于该值用中文概括（不超过 16 字）。", raw),
			truncateRunes(raw, 64), false
	}
	item, err := s.comicStyleSvc.LookupByValue(ctx, raw)
	if err != nil {
		s.logger.Warn("fragment story prefill: comic style lookup failed",
			zap.String("styleValue", raw),
			zap.Error(err))
		return fmt.Sprintf("【碎片创作风格】数据库 style=%q；目录查询失败，JSON 的 style 请基于该值用中文概括（不超过 16 字）。", raw),
			truncateRunes(raw, 64), false
	}
	if item != nil {
		return fmt.Sprintf(
			"【碎片创作风格】与碎片创作页相同数据源（表 fragment_comic_styles，接口与拉取风格列表一致）：展示名「%s」，value=%q。JSON 的 style 字段必须与展示名「%s」完全一致（逐字相同，勿改字）。",
			item.Name, item.Value, item.Name,
		), item.Name, false
	}
	return fmt.Sprintf(
		"【碎片创作风格】数据库 style=%q，在 fragment_comic_styles 中未找到对应 value。JSON 的 style 请基于该字符串理解视觉方向并用中文概括（不超过 16 字）。",
		raw,
	), truncateRunes(raw, 64), false
}

// ExpandFragmentStoryPrefillAI 基于碎片内容调用大模型生成新建故事预填（不写库、不标记已转换）。
func (s *Service) ExpandFragmentStoryPrefillAI(ctx context.Context, userID, fragmentID string, req domain.FragmentStoryPrefillAIRequest) (*domain.FragmentStoryPrefillAIResponse, error) {
	hasHuoshan := s.genAPI != nil && s.genAPI.HuoshanInternalClient() != nil
	if !hasHuoshan && s.geminiClient == nil {
		return nil, errors.New("AI text generation is not configured")
	}
	sc := req.SceneCount
	if sc < 2 || sc > 8 {
		sc = 3
	}

	fragment, err := s.getFragmentByID(ctx, fragmentID)
	if err != nil {
		return nil, fmt.Errorf("fragment not found: %w", err)
	}

	owner := strings.TrimSpace(fragment.UserID)
	if owner == "" {
		owner = strings.TrimSpace(fragment.CreatorID)
	}
	if owner != userID {
		return nil, errors.New("permission denied: not fragment owner")
	}
	if fragment.IsConverted || (fragment.ConvertedToStoryID != nil && strings.TrimSpace(*fragment.ConvertedToStoryID) != "") {
		return nil, errors.New("fragment already converted to story")
	}

	content := strings.TrimSpace(fragment.Content)
	caption := strings.TrimSpace(fragment.Caption)
	topic := strings.TrimSpace(fragment.Topic)
	media := fragment.MediaURLs
	if len(media) == 0 && fragment.ImageUrls != "" {
		_ = json.Unmarshal([]byte(fragment.ImageUrls), &media)
	}
	var imgNote string
	if len(media) > 0 {
		imgNote = media[0]
	}
	refImageURLs := media

	styleParagraph, fixedStyle, useStyleAI := s.resolveFragmentStyleForPrefillPrompt(ctx, fragment)
	genreList := strings.Join(allowedStoryPrefillGenres, "、")

	prompt := fmt.Sprintf(`你是中文故事创作助手。用户有一段「故事碎片」，要扩展为可做多格分镜（约 %d 格）的长故事设定。

%s

请只输出一段合法 JSON（不要 markdown、不要注释），字段如下：
{
  "title": "故事标题，必须不超过 7 个汉字（含标点则占用字数），精炼有吸引力",
  "description": "扩写后的故事简介与设定汇总，必须不超过 200 个汉字（含标点），与创作页「故事描述」上限一致",
  "summary": "一句话梗概，30 字以内",
  "style": "字符串：遵守上文对「碎片创作风格」的说明",
  "genre": "必须从以下择一：%s",
  "tags": ["标签1","标签2"],
  "suggestedCharacters": [{"name":"角色名","background":"可选，一句人设"}]
}

约束：
- tags 最多 3 个，每个不超过 10 字
- suggestedCharacters 最多 5 个，name 必填（人物、动物、拟人器物均可；静物主角用简短识别名如「红伞小满」「窗台橘猫」）
- description 不超过 200 字，承接碎片内容，可补充世界观与冲突，不要与碎片明显矛盾
- 扩写须做剧情合理拓展：description 在 200 字内包含「主体想要什么 / 主要阻碍 / 行动或误会 / 代价或悬念」中的至少三项，让读者感到故事可继续分镜；不要只复述碎片画面
- 若碎片主角非人类，扩写保持物种/器物身份一致，用可观察行为表达执念，避免硬套人类社会剧情；例如动物守护、器物逃离遗忘、静物争夺光线或位置
- suggestedCharacters 的 background 不只是外观，也应一句话写出可推动剧情的目标、恐惧或执念
- 扩写与 summary 须与上文碎片创作风格一致

【碎片正文】
%s

【碎片标题/旁白】
%s

【话题】
%s

【参考图 URL（可能为空）】
%s
`, sc, styleParagraph, genreList, content, caption, topic, imgNote)

	ai := s.AIService()
	raw, err := ai.GenerateTextForFragmentStoryPrefill(ctx, prompt, refImageURLs)
	if err != nil {
		s.logger.Error("fragment story prefill: AI text generation failed",
			zap.String("fragmentID", fragmentID),
			zap.Error(err))
		return nil, fmt.Errorf("AI generation failed: %w", err)
	}

	jsonStr := extractJSONFromModelText(raw)
	var parsed fragmentPrefillAIJSON
	if err := json.Unmarshal([]byte(jsonStr), &parsed); err != nil {
		s.logger.Warn("fragment story prefill: json parse failed",
			zap.String("fragmentID", fragmentID),
			zap.String("snippet", truncateRunes(jsonStr, 200)),
			zap.Error(err))
		return nil, errors.New("AI returned invalid JSON")
	}

	title := truncateRunes(parsed.Title, 7)
	if title == "" {
		title = truncateRunes(content, 7)
		if title == "" {
			title = "新故事"
		}
	}

	desc := strings.TrimSpace(parsed.Description)
	if len([]rune(desc)) > 200 {
		desc = truncateRunes(desc, 200)
	}

	chars := make([]domain.FragmentStoryPrefillCharacter, 0, 5)
	for i, c := range parsed.SuggestedCharacters {
		if i >= 5 {
			break
		}
		n := strings.TrimSpace(c.Name)
		if n == "" {
			continue
		}
		chars = append(chars, domain.FragmentStoryPrefillCharacter{
			Name:       truncateRunes(n, 32),
			Background: strings.TrimSpace(c.Background),
		})
	}

	styleOut := strings.TrimSpace(fixedStyle)
	if useStyleAI || styleOut == "" {
		styleOut = truncateRunes(strings.TrimSpace(parsed.Style), 64)
		if styleOut == "" {
			styleOut = "未指定"
		}
	}

	resp := &domain.FragmentStoryPrefillAIResponse{
		Title:               title,
		Description:         desc,
		Summary:             strings.TrimSpace(parsed.Summary),
		Style:               styleOut,
		Genre:               normalizePrefillGenre(parsed.Genre),
		Tags:                clampPrefillTags(parsed.Tags, 3),
		SuggestedCharacters: chars,
	}

	s.logger.Info("fragment story prefill ok",
		zap.String("fragmentID", fragmentID),
		zap.String("title", resp.Title),
		zap.String("style", resp.Style),
		zap.Bool("styleFromCatalog", !useStyleAI && strings.TrimSpace(fixedStyle) != ""),
		zap.Int("sceneCount", sc))

	return resp, nil
}
