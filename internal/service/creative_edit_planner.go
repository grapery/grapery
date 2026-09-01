package service

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/grapestree/fgrapery/grapery/internal/domain"
)

var creativeTargetPattern = regexp.MustCompile(`第\s*([0-9一二两三四五六七八九十]+)\s*(?:张|页|格|幕|个画面|个镜头)`)
var creativeCountPattern = regexp.MustCompile(`(?:改成|调整为|做成|生成|要)?\s*([1-8一二两三四五六七八])\s*(?:张|页|格|幕|个画面|个镜头)`)

func planCreativeEdit(input string, hasExisting bool, unit string) domain.CreativeEditPlan {
	text := strings.TrimSpace(input)
	plan := domain.CreativeEditPlan{Operation: "create", Preserve: []string{"角色身份与核心外观", "既有世界设定", "未被点名的画面"}}
	if !hasExisting {
		return plan
	}
	plan.Operation = "append"
	plan.Preserve = []string{"既有画面", "角色连续性", "世界设定", "前序因果"}
	targets := extractCreativeTargetIndexes(text)
	if strings.Contains(text, "最后一") || strings.Contains(text, "最后张") || strings.Contains(text, "最后一格") || strings.Contains(text, "结尾画面") {
		// -1 is resolved by the client against the latest local result.
		targets = append(targets, -1)
	}
	plan.TargetIndexes = uniqueInts(targets)

	if len(plan.TargetIndexes) > 0 {
		plan.Operation = "replace"
		plan.EstimatedRegenerationCount = len(plan.TargetIndexes)
		plan.RequestedChanges = []string{text}
		return plan
	}
	if isOptionAdjustment(text) {
		plan.Operation = "adjust_options"
		plan.RequestedChanges = []string{text}
		return plan
	}
	if containsAny(text, "改一下", "修改", "重画", "重绘", "换掉", "替换", "重新生成") {
		plan.Operation = "revise"
		plan.NeedsClarification = true
		plan.ClarificationQuestion = fmt.Sprintf("你想修改哪一%s？请直接说“第 3%s”或“最后一%s”，我会只重做目标画面。", unit, unit, unit)
		return plan
	}
	plan.RequestedChanges = []string{text}
	return plan
}

func extractCreativeTargetIndexes(input string) []int {
	matches := creativeTargetPattern.FindAllStringSubmatch(input, -1)
	result := make([]int, 0, len(matches))
	for _, match := range matches {
		if len(match) < 2 {
			continue
		}
		if n := parseCreativeNumber(match[1]); n > 0 {
			result = append(result, n)
		}
	}
	return result
}

func explicitCreativeCount(input string) int {
	match := creativeCountPattern.FindStringSubmatch(input)
	if len(match) < 2 {
		return 0
	}
	return parseCreativeNumber(match[1])
}

func explicitCreativeAspectRatio(input string) string {
	normalized := strings.NewReplacer("：", ":", " ", "").Replace(input)
	for _, ratio := range []string{"9:16", "16:9", "4:3", "3:4", "1:1"} {
		if strings.Contains(normalized, ratio) {
			return ratio
		}
	}
	return ""
}

func explicitCreativeStyle(input string) string {
	lower := strings.ToLower(input)
	switch {
	case strings.Contains(input, "水墨"):
		return "ink_wash"
	case strings.Contains(input, "日系") || strings.Contains(input, "动漫") || strings.Contains(lower, "anime"):
		return "anime"
	case strings.Contains(input, "电影写实") || strings.Contains(input, "写实") || strings.Contains(lower, "realistic"):
		return "realistic"
	case strings.Contains(input, "漫画") || strings.Contains(lower, "comic"):
		return "comic"
	case strings.Contains(input, "积木") || strings.Contains(lower, "lego"):
		return "vintage_clay"
	case strings.Contains(input, "80年代") || strings.Contains(lower, "80s"):
		return "eighties_tv"
	default:
		return ""
	}
}

func isOptionAdjustment(input string) bool {
	return containsAny(input, "画风", "风格", "比例", "横屏", "竖屏", "张", "页", "格", "公开", "私密")
}

func containsAny(input string, candidates ...string) bool {
	for _, candidate := range candidates {
		if strings.Contains(input, candidate) {
			return true
		}
	}
	return false
}

func parseCreativeNumber(raw string) int {
	if n, err := strconv.Atoi(raw); err == nil {
		return n
	}
	values := map[string]int{"一": 1, "二": 2, "两": 2, "三": 3, "四": 4, "五": 5, "六": 6, "七": 7, "八": 8, "九": 9, "十": 10}
	return values[raw]
}

func uniqueInts(values []int) []int {
	seen := map[int]bool{}
	result := make([]int, 0, len(values))
	for _, value := range values {
		if seen[value] {
			continue
		}
		seen[value] = true
		result = append(result, value)
	}
	return result
}
