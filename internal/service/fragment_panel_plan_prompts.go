package service

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/grapestree/fgrapery/grapery/internal/domain"
)

const fragmentPanelGeminiReferenceImagePreamble = "上方为用户参考图。请先在心里完成锚点判断（不要单独输出成段说明）：" +
	"识别主体（人物/物体/建筑/自然物）、室内外空间、时间感与光线、主色与情绪、构图重心；" +
	"判断用户可能希望保留的身份特征与世界氛围。" +
	"将该图视为故事世界与视觉风格的锚点，而不是要求每一格都像素级复刻同一 photograph。"

func panelPlanNarrativeRhythm(panelCount int) string {
	switch {
	case panelCount == 1:
		return "1 格：选择最有叙事张力、让读者想追问前因后果的一个瞬间。"
	case panelCount == 2:
		return "2 格：第一格建立预期，第二格以视角、信息、情绪或后果改变预期。"
	case panelCount == 3:
		return "3 格：引子、变化、回响；第三格回应前文但不必解释所有信息。"
	default:
		return fmt.Sprintf("%d 格：建立局面后，交替安排尝试、变化、代价与回响；允许一格氛围停顿，但不能连续停顿。", panelCount)
	}
}

func fragmentPanelPlanLayoutAddon(req domain.FragmentPanelGenerationRequest) string {
	var parts []string
	switch strings.TrimSpace(req.DialogueMode) {
	case "none":
		parts = append(parts, "本任务：各格画面不要出现对白气泡或旁白框。")
	case "auto":
		parts = append(parts, "只在叙事确有需要时使用短对白、思想泡或旁白框。")
	case "from_user_input":
		parts = append(parts, "图中文字只能来自用户文字，不得补写新对白。")
	}
	return strings.Join(parts, "\n")
}

func panelPlanLayoutWithVisualEvidence(layoutAddon string, evidence []domain.FragmentVisualEvidence) string {
	if len(evidence) == 0 {
		return layoutAddon
	}
	b, err := json.Marshal(evidence)
	if err != nil {
		return layoutAddon
	}
	parts := []string{strings.TrimSpace(layoutAddon)}
	parts = append(parts, "参考图多模态视觉事实 JSON：\n"+string(b)+"\n分镜规划必须优先沿用这些可见事实；若创作补充未在图片中出现，不得写入 visualBible 的 immutableTraits。")
	return strings.TrimSpace(strings.Join(parts, "\n"))
}
