package service

import (
	"encoding/json"
	"fmt"
	"math"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/grapestree/fgrapery/grapery/internal/domain"
)

const defaultWorkflowRouteConfidence = 0.95

// BuildWorkflowContentProfile turns runtime input into a small stable routing
// contract. Explicit profile values win; conservative derivation fills only
// commonly available fields so callers can adopt the router incrementally.
func BuildWorkflowContentProfile(surface, action string, input map[string]any) domain.WorkflowContentProfile {
	profile := domain.WorkflowContentProfile{
		"surface": strings.TrimSpace(surface),
		"action":  strings.TrimSpace(action),
	}
	if explicit, ok := input["contentProfile"].(map[string]any); ok {
		mergeWorkflowProfile(profile, explicit)
	}
	artifact := lastWorkflowSegment(surface)
	if artifact == "" {
		artifact = strings.TrimSpace(action)
	}
	setWorkflowProfileDefault(profile, "artifact", artifact)
	setWorkflowProfileDefault(profile, "operation", workflowString(input, "operation", "editOperation"))
	setWorkflowProfileDefault(profile, "language", workflowString(input, "language"))
	setWorkflowProfileDefault(profile, "style", workflowString(input, "style", "comicStyle"))
	setWorkflowProfileDefault(profile, "qualityTier", workflowString(input, "qualityTier"))

	text := workflowString(input, "chapterContent", "storyContent", "content", "userInput", "prompt", "seedPrompt")
	mode, actionIntensity, dialogueDensity := inferWorkflowNarrativeProfile(text)
	setWorkflowProfileDefault(profile, "narrativeMode", mode)
	setWorkflowProfileDefault(profile, "actionIntensity", actionIntensity)
	setWorkflowProfileDefault(profile, "dialogueDensity", dialogueDensity)
	setWorkflowProfileDefault(profile, "characterCount", workflowCollectionCount(input, "characters", "characterIds"))
	setWorkflowProfileDefault(profile, "sceneCount", workflowNumeric(input, "sceneCount", "imageCount"))

	hasParent := workflowHasValue(input, "parentStoryboardId", "parentFragmentId", "parentId")
	hasTarget := workflowHasValue(input, "targetDraftFragmentId", "storyboardId", "fragmentId")
	if hasParent {
		setWorkflowProfileDefault(profile, "operation", "fork")
		setWorkflowProfileDefault(profile, "branchMode", "continuation")
	}
	continuity := "standard"
	if hasParent || hasTarget {
		continuity = "strong"
	}
	setWorkflowProfileDefault(profile, "continuityLevel", continuity)

	references := workflowCollectionCount(input, "referenceImages", "imageUrls", "referenceSlots")
	coverage := "none"
	if references > 0 {
		coverage = "partial"
	}
	setWorkflowProfileDefault(profile, "referenceCoverage", coverage)
	return profile
}

func mergeWorkflowProfile(target, source map[string]any) {
	for key, value := range source {
		key = strings.TrimSpace(key)
		if key != "" && value != nil {
			target[key] = value
		}
	}
}

func setWorkflowProfileDefault(profile map[string]any, key string, value any) {
	if _, exists := profile[key]; exists || workflowEmptyValue(value) {
		return
	}
	profile[key] = value
}

func workflowEmptyValue(value any) bool {
	switch typed := value.(type) {
	case nil:
		return true
	case string:
		return strings.TrimSpace(typed) == ""
	default:
		return false
	}
}

// ValidateWorkflowBindingConditions rejects malformed routing rules when a
// binding is saved, instead of letting a bad operator break creation traffic.
func ValidateWorkflowBindingConditions(conditions map[string]any) error {
	if len(conditions) == 0 {
		return nil
	}
	for _, group := range []string{"all", "any"} {
		raw, ok := conditions[group]
		if !ok {
			continue
		}
		if len(conditions) != 1 {
			return errorsForWorkflowRule(group + " cannot be combined with direct profile fields")
		}
		rules, err := workflowRules(raw)
		if err != nil {
			return err
		}
		for _, rule := range rules {
			if err := validateWorkflowRule(rule); err != nil {
				return err
			}
		}
		return nil
	}
	for field := range conditions {
		if strings.TrimSpace(field) == "" {
			return errorsForWorkflowRule("routing profile field cannot be empty")
		}
	}
	return nil
}

func validateWorkflowRule(rule map[string]any) error {
	field, _ := rule["field"].(string)
	if strings.TrimSpace(field) == "" {
		return errorsForWorkflowRule("routing rule field is required")
	}
	op, _ := rule["op"].(string)
	op = strings.ToLower(strings.TrimSpace(op))
	if op == "" {
		op = "eq"
	}
	switch op {
	case "exists", "eq", "neq", "gte", "gt", "lte", "lt", "contains":
		return nil
	case "in", "not_in":
		if len(workflowSlice(rule["value"])) == 0 {
			return errorsForWorkflowRule(op + " requires a non-empty value array")
		}
		return nil
	default:
		return fmt.Errorf("unsupported routing operator %q", op)
	}
}

func lastWorkflowSegment(surface string) string {
	parts := strings.FieldsFunc(strings.ToLower(strings.TrimSpace(surface)), func(r rune) bool { return r == '.' || r == '/' || r == ':' })
	if len(parts) == 0 {
		return ""
	}
	return parts[len(parts)-1]
}

func workflowString(input map[string]any, keys ...string) string {
	for _, key := range keys {
		if value, ok := input[key]; ok {
			if text, ok := value.(string); ok && strings.TrimSpace(text) != "" {
				return strings.TrimSpace(text)
			}
		}
	}
	return ""
}

func workflowHasValue(input map[string]any, keys ...string) bool {
	return workflowString(input, keys...) != ""
}

func workflowNumeric(input map[string]any, keys ...string) int {
	for _, key := range keys {
		if value, ok := input[key]; ok {
			if number, ok := workflowFloat(value); ok {
				return int(number)
			}
		}
	}
	return 0
}

func workflowCollectionCount(input map[string]any, keys ...string) int {
	for _, key := range keys {
		value, ok := input[key]
		if !ok || value == nil {
			continue
		}
		ref := reflect.ValueOf(value)
		if ref.Kind() == reflect.Array || ref.Kind() == reflect.Slice || ref.Kind() == reflect.Map {
			return ref.Len()
		}
	}
	return 0
}

func inferWorkflowNarrativeProfile(text string) (string, float64, float64) {
	text = strings.TrimSpace(text)
	if text == "" {
		return "general", 0, 0
	}
	lower := strings.ToLower(text)
	actionTerms := []string{"追", "跑", "冲", "打", "战斗", "爆炸", "跳", "逃", "射击", "追逐", "fight", "chase", "run", "jump", "explode", "shoot"}
	dialogueTerms := []string{"说道", "问道", "回答", "对话", "告诉", "低声", "喊道", "said", "asked", "replied", "told"}
	actionHits, dialogueHits := 0, 0
	for _, term := range actionTerms {
		actionHits += strings.Count(lower, term)
	}
	for _, term := range dialogueTerms {
		dialogueHits += strings.Count(lower, term)
	}
	quoteHits := strings.Count(text, "\"") + strings.Count(text, "“") + strings.Count(text, "”") + strings.Count(text, "「") + strings.Count(text, "」")
	runes := math.Max(float64(utf8.RuneCountInString(text)), 1)
	actionIntensity := math.Min(float64(actionHits)/math.Max(runes/80, 1), 1)
	dialogueDensity := math.Min((float64(dialogueHits)*8+float64(quoteHits)*3)/runes, 1)
	mode := "general"
	if actionIntensity >= 0.35 && actionIntensity >= dialogueDensity {
		mode = "action"
	} else if dialogueDensity >= 0.12 {
		mode = "dialogue"
	}
	return mode, roundWorkflowScore(actionIntensity), roundWorkflowScore(dialogueDensity)
}

func roundWorkflowScore(value float64) float64 { return math.Round(value*1000) / 1000 }

// ResolveWorkflowEntries selects the first priority-ordered eligible binding.
// A binding without conditions is the explicit fallback and is chosen only
// when no conditional binding matches.
func ResolveWorkflowEntries(entries []*domain.WorkflowCatalogEntry, profile domain.WorkflowContentProfile) (*domain.WorkflowResolution, error) {
	if len(entries) == 0 {
		return nil, fmt.Errorf("no active workflow binding")
	}
	candidates := make([]string, 0, len(entries))
	var fallback *domain.WorkflowCatalogEntry
	for _, entry := range entries {
		if entry == nil {
			continue
		}
		candidates = append(candidates, entry.Release.ID)
		if len(entry.Binding.Conditions) == 0 {
			if fallback == nil {
				fallback = entry
			}
			continue
		}
		matched, reason, err := workflowConditionsMatch(entry.Binding.Conditions, profile)
		if err != nil {
			return nil, fmt.Errorf("binding %s conditions: %w", entry.Binding.ID, err)
		}
		if matched {
			return workflowResolution(entry, profile, reason, false, defaultWorkflowRouteConfidence, candidates), nil
		}
	}
	if fallback != nil {
		return workflowResolution(fallback, profile, "default binding fallback", true, 0.5, candidates), nil
	}
	return nil, fmt.Errorf("no workflow binding matched content profile")
}

func workflowResolution(entry *domain.WorkflowCatalogEntry, profile domain.WorkflowContentProfile, reason string, fallback bool, confidence float64, candidates []string) *domain.WorkflowResolution {
	return &domain.WorkflowResolution{
		Entry: *entry, RouterVersion: domain.WorkflowRouterVersion, Profile: profile,
		RouteReason: reason, Confidence: confidence, Fallback: fallback,
		CandidateIDs: append([]string(nil), candidates...),
	}
}

func workflowConditionsMatch(conditions map[string]any, profile map[string]any) (bool, string, error) {
	if err := ValidateWorkflowBindingConditions(conditions); err != nil {
		return false, "", err
	}
	if raw, ok := conditions["all"]; ok {
		rules, err := workflowRules(raw)
		if err != nil {
			return false, "", err
		}
		for _, rule := range rules {
			matched, err := workflowRuleMatches(rule, profile)
			if err != nil || !matched {
				return false, "", err
			}
		}
		return true, fmt.Sprintf("matched all %d routing rules", len(rules)), nil
	}
	if raw, ok := conditions["any"]; ok {
		rules, err := workflowRules(raw)
		if err != nil {
			return false, "", err
		}
		for _, rule := range rules {
			matched, err := workflowRuleMatches(rule, profile)
			if err != nil {
				return false, "", err
			}
			if matched {
				return true, "matched one routing rule", nil
			}
		}
		return false, "", nil
	}
	keys := make([]string, 0, len(conditions))
	for key := range conditions {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		actual, exists := workflowProfileValue(profile, key)
		if !exists || !workflowEqual(actual, conditions[key]) {
			return false, "", nil
		}
	}
	return true, "matched binding profile fields: " + strings.Join(keys, ", "), nil
}

func workflowRules(raw any) ([]map[string]any, error) {
	encoded, err := json.Marshal(raw)
	if err != nil {
		return nil, err
	}
	var rules []map[string]any
	if err := json.Unmarshal(encoded, &rules); err != nil {
		return nil, errorsForWorkflowRule("all/any must be an array of routing rules")
	}
	if len(rules) == 0 {
		return nil, errorsForWorkflowRule("routing rule list cannot be empty")
	}
	return rules, nil
}

func errorsForWorkflowRule(message string) error { return fmt.Errorf("%s", message) }

func workflowRuleMatches(rule map[string]any, profile map[string]any) (bool, error) {
	field, _ := rule["field"].(string)
	field = strings.TrimSpace(field)
	if field == "" {
		return false, errorsForWorkflowRule("routing rule field is required")
	}
	operator, _ := rule["op"].(string)
	operator = strings.ToLower(strings.TrimSpace(operator))
	if operator == "" {
		operator = "eq"
	}
	actual, exists := workflowProfileValue(profile, field)
	expected := rule["value"]
	switch operator {
	case "exists":
		want, ok := expected.(bool)
		if !ok {
			want = true
		}
		return exists == want, nil
	case "eq":
		return exists && workflowEqual(actual, expected), nil
	case "neq":
		return !exists || !workflowEqual(actual, expected), nil
	case "in", "not_in":
		items := workflowSlice(expected)
		matched := false
		for _, item := range items {
			if workflowEqual(actual, item) {
				matched = true
				break
			}
		}
		if operator == "not_in" {
			matched = !matched
		}
		return exists && matched, nil
	case "gte", "gt", "lte", "lt":
		left, lok := workflowFloat(actual)
		right, rok := workflowFloat(expected)
		if !exists || !lok || !rok {
			return false, nil
		}
		switch operator {
		case "gte":
			return left >= right, nil
		case "gt":
			return left > right, nil
		case "lte":
			return left <= right, nil
		default:
			return left < right, nil
		}
	case "contains":
		return exists && strings.Contains(strings.ToLower(fmt.Sprint(actual)), strings.ToLower(fmt.Sprint(expected))), nil
	default:
		return false, fmt.Errorf("unsupported routing operator %q", operator)
	}
}

func workflowProfileValue(profile map[string]any, path string) (any, bool) {
	var current any = profile
	for _, part := range strings.Split(path, ".") {
		object, ok := current.(map[string]any)
		if !ok {
			return nil, false
		}
		current, ok = object[part]
		if !ok {
			return nil, false
		}
	}
	return current, true
}

func workflowEqual(left, right any) bool {
	if lnum, ok := workflowFloat(left); ok {
		if rnum, ok := workflowFloat(right); ok {
			return math.Abs(lnum-rnum) < 0.000001
		}
	}
	return strings.EqualFold(strings.TrimSpace(fmt.Sprint(left)), strings.TrimSpace(fmt.Sprint(right)))
}

func workflowFloat(value any) (float64, bool) {
	switch number := value.(type) {
	case int:
		return float64(number), true
	case int32:
		return float64(number), true
	case int64:
		return float64(number), true
	case float32:
		return float64(number), true
	case float64:
		return number, true
	case json.Number:
		parsed, err := number.Float64()
		return parsed, err == nil
	case string:
		parsed, err := strconv.ParseFloat(strings.TrimSpace(number), 64)
		return parsed, err == nil
	default:
		return 0, false
	}
}

func workflowSlice(value any) []any {
	ref := reflect.ValueOf(value)
	if !ref.IsValid() || (ref.Kind() != reflect.Array && ref.Kind() != reflect.Slice) {
		return nil
	}
	out := make([]any, ref.Len())
	for i := range out {
		out[i] = ref.Index(i).Interface()
	}
	return out
}
