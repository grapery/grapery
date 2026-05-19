package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/grapestree/fgrapery/grapery/internal/domain"
)

// decodePanelPlansFlexible unmarshals panels[] from arbitrary LLM-shaped JSON without
// aborting when non-panel fields disagree with our structs.
func decodePanelPlansFlexible(raw json.RawMessage) ([]domain.FragmentPanelPlanItem, error) {
	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 || string(raw) == "null" {
		return nil, fmt.Errorf("empty panels")
	}

	// Occasionally models double-encode the panels payload as a JSON string.
	var asString string
	if err := json.Unmarshal(raw, &asString); err == nil {
		t := strings.TrimSpace(asString)
		if t != "" && (strings.HasPrefix(t, "[") || strings.HasPrefix(t, "{")) {
			return decodePanelPlansFlexible(json.RawMessage(t))
		}
	}

	var strict []domain.FragmentPanelPlanItem
	if err := json.Unmarshal(raw, &strict); err == nil && panelPlanItemsLooksComplete(strict) {
		return strict, nil
	}

	var obj map[string]interface{}
	if err := json.Unmarshal(raw, &obj); err == nil && len(obj) > 0 && allNumericLikeKeys(obj) {
		got, err := panelsFromNumericObjectMap(obj)
		if err == nil && len(got) > 0 {
			return got, nil
		}
	}

	var arr []map[string]interface{}
	if err := json.Unmarshal(raw, &arr); err != nil {
		return nil, fmt.Errorf("panels must be JSON array or object map of panels: %w", err)
	}
	out := make([]domain.FragmentPanelPlanItem, 0, len(arr))
	for _, m := range arr {
		if m == nil {
			continue
		}
		out = append(out, fragmentPanelPlanItemFromMap(m))
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("no panel entries in panels array")
	}
	return out, nil
}

func decodeVisualBibleStrict(raw json.RawMessage) *domain.FragmentVisualBible {
	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 || string(raw) == "null" {
		return nil
	}
	var vb domain.FragmentVisualBible
	if err := json.Unmarshal(raw, &vb); err != nil {
		return nil
	}
	return &vb
}

func parseFragmentPanelPlanJSONOnce(blob string, panelCount int) ([]domain.FragmentPanelPlanItem, *domain.FragmentVisualBible, error) {
	blob = strings.TrimSpace(blob)
	if blob == "" {
		return nil, nil, fmt.Errorf("empty response")
	}

	switch {
	case strings.HasPrefix(blob, "["):
		panels, err := decodePanelPlansFlexible(json.RawMessage(blob))
		if err != nil {
			return nil, nil, err
		}
		var vb *domain.FragmentVisualBible
		normalizeFragmentVisualBible(&vb)
		plan, err := normalizeFragmentPanelPlan(panels, panelCount)
		return plan, vb, err
	case strings.HasPrefix(blob, "{"):
		var env struct {
			VisualBible json.RawMessage `json:"visualBible,omitempty"`
			Panels      json.RawMessage `json:"panels,omitempty"`
		}
		if err := json.Unmarshal([]byte(blob), &env); err != nil {
			return nil, nil, fmt.Errorf("%w", err)
		}
		if panelMsgLen(env.Panels) == 0 {
			return nil, nil, fmt.Errorf("missing panels")
		}
		panels, err := decodePanelPlansFlexible(env.Panels)
		if err != nil {
			return nil, nil, fmt.Errorf("panels: %w", err)
		}
		vb := decodeVisualBibleStrict(env.VisualBible)
		normalizeFragmentVisualBible(&vb)
		plan, err := normalizeFragmentPanelPlan(panels, panelCount)
		return plan, vb, err
	default:
		return nil, nil, fmt.Errorf("unexpected JSON outer kind")
	}
}

func panelMsgLen(raw json.RawMessage) int {
	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 || string(raw) == "null" {
		return 0
	}
	return len(raw)
}

func allNumericLikeKeys(obj map[string]interface{}) bool {
	if len(obj) == 0 {
		return false
	}
	for k := range obj {
		k = strings.TrimSpace(k)
		if _, err := strconv.Atoi(k); err != nil {
			return false
		}
	}
	return true
}

func panelsFromNumericObjectMap(obj map[string]interface{}) ([]domain.FragmentPanelPlanItem, error) {
	type keyed struct {
		idx int
		m   map[string]interface{}
	}
	pairs := make([]keyed, 0, len(obj))
	for k, v := range obj {
		idx, err := strconv.Atoi(strings.TrimSpace(k))
		if err != nil {
			continue
		}
		m, ok := v.(map[string]interface{})
		if !ok {
			continue
		}
		pairs = append(pairs, keyed{idx: idx, m: m})
	}
	if len(pairs) == 0 {
		return nil, fmt.Errorf("no panel objects")
	}
	sort.Slice(pairs, func(i, j int) bool { return pairs[i].idx < pairs[j].idx })
	out := make([]domain.FragmentPanelPlanItem, 0, len(pairs))
	for _, p := range pairs {
		item := fragmentPanelPlanItemFromMap(p.m)
		if p.idx >= 0 {
			item.Index = p.idx
		}
		out = append(out, item)
	}
	return out, nil
}

func fragmentPanelPlanItemFromMap(m map[string]interface{}) domain.FragmentPanelPlanItem {
	var p domain.FragmentPanelPlanItem
	p.Index = intFromAny(pickAny(m, "index", "idx", "panel_index", "panelIndex"))
	p.ImagePrompt = stringFromAny(
		pickAny(m, "image_prompt", "imagePrompt", "prompt", "imagePromptText", "scene_prompt"),
	)
	p.Caption = stringFromAny(pickAny(m, "caption", "description", "summary", "text"))
	p.ReferenceKeys = stringSliceFromAny(pickAny(m, "reference_keys", "referenceKeys", "references", "ref_keys"))
	p.LayoutIntent = stringFromAny(pickAny(m, "layout_intent", "layoutIntent"))
	p.CompositionPlan = stringFromAny(pickAny(m, "composition_plan", "compositionPlan"))
	p.ShotType = stringFromAny(pickAny(m, "shot_type", "shotType"))
	p.VisualHierarchy = stringFromAny(pickAny(m, "visual_hierarchy", "visualHierarchy"))
	p.ComicTexts = comicTextsFromAny(pickAny(m, "comic_texts", "comicTexts", "speech", "dialogue"))
	return p
}

func pickAny(m map[string]interface{}, keys ...string) interface{} {
	for _, k := range keys {
		if v, ok := m[k]; ok && v != nil {
			return v
		}
	}
	return nil
}

func intFromAny(v interface{}) int {
	switch x := v.(type) {
	case nil:
		return 0
	case float64:
		return int(x)
	case json.Number:
		i, err := x.Int64()
		if err != nil {
			return 0
		}
		return int(i)
	case int:
		return x
	case int64:
		return int(x)
	case string:
		i, err := strconv.Atoi(strings.TrimSpace(x))
		if err != nil {
			return 0
		}
		return i
	default:
		return 0
	}
}

func stringFromAny(v interface{}) string {
	if v == nil {
		return ""
	}
	switch x := v.(type) {
	case string:
		return strings.TrimSpace(x)
	case float64:
		return strconv.FormatInt(int64(x), 10)
	case json.Number:
		return x.String()
	case bool:
		if x {
			return "true"
		}
		return "false"
	default:
		b, err := json.Marshal(x)
		if err != nil {
			return ""
		}
		s := strings.TrimSpace(string(b))
		if len(s) >= 2 && s[0] == '"' && s[len(s)-1] == '"' {
			var inner string
			if err := json.Unmarshal(b, &inner); err == nil {
				return strings.TrimSpace(inner)
			}
		}
		return s
	}
}

func stringSliceFromAny(v interface{}) []string {
	if v == nil {
		return nil
	}
	switch x := v.(type) {
	case string:
		s := strings.TrimSpace(x)
		if s == "" {
			return nil
		}
		return []string{s}
	case []interface{}:
		var out []string
		for _, e := range x {
			if s := stringFromAny(e); s != "" {
				out = append(out, s)
			}
		}
		return out
	case []string:
		var out []string
		for _, s := range x {
			if t := strings.TrimSpace(s); t != "" {
				out = append(out, t)
			}
		}
		return out
	default:
		return nil
	}
}

func comicTextsFromAny(v interface{}) []domain.FragmentComicText {
	if v == nil {
		return nil
	}
	switch x := v.(type) {
	case map[string]interface{}:
		return []domain.FragmentComicText{fragmentComicTextFromMap(x)}
	case []interface{}:
		out := make([]domain.FragmentComicText, 0, len(x))
		for _, e := range x {
			m, ok := e.(map[string]interface{})
			if !ok {
				continue
			}
			out = append(out, fragmentComicTextFromMap(m))
		}
		return out
	case string:
		s := strings.TrimSpace(x)
		if s == "" {
			return nil
		}
		return []domain.FragmentComicText{{Type: "narration", Text: s}}
	default:
		return nil
	}
}

func fragmentComicTextFromMap(m map[string]interface{}) domain.FragmentComicText {
	c := domain.FragmentComicText{}
	c.Type = stringFromAny(pickAny(m, "type", "kind"))
	if strings.TrimSpace(c.Type) == "" {
		c.Type = "narration"
	}
	c.Text = stringFromAny(pickAny(m, "text", "content", "body", "value", "line"))
	c.Speaker = stringFromAny(pickAny(m, "speaker", "who", "role"))
	c.Position = stringFromAny(pickAny(m, "position", "place", "box"))
	return c
}

func panelPlanItemsLooksComplete(items []domain.FragmentPanelPlanItem) bool {
	if len(items) == 0 {
		return false
	}
	for _, p := range items {
		if strings.TrimSpace(p.ImagePrompt) == "" {
			return false
		}
	}
	return true
}
