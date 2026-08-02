package gemini

import (
	"fmt"
	"strings"
)

// VideoPromptSpec captures the recommended sections for a high quality video prompt.
type VideoPromptSpec struct {
	Theme           string   // 主体与设定，例如主角外观与总体概念
	Protagonist     string   // 主角描述
	Scene           string   // 场景环境或地点
	Action          string   // 主体动作或镜头中需要发生的行为
	Lighting        string   // 光照描述
	Camera          string   // 拍摄方式、镜头语言
	CameraMovements []string // 镜头运动描述
	Style           string   // 艺术或美学风格
	Stylization     []string // 造型或风格强化元素
	VisualStyle     string   // 视觉风格
	Mood            string   // 整体情绪
	AudioMood       string   // 音频氛围或配乐说明
	Additional      []string // 其他辅助信息，例如情绪、色彩、技术细节等
	UserPrompt      string   // 用户原始输入，方便拼接引用
	Aesthetic       *VideoAestheticSpec
}

type VideoAestheticSpec struct {
	Atmosphere   string
	ColorPalette string
	Lens         string
}

// BuildVideoPrompt assembles a verbose prompt string that covers the required sections.
func BuildVideoPrompt(spec VideoPromptSpec) string {
	var sentences []string

	combine := func(parts ...string) string {
		var cleaned []string
		for _, part := range parts {
			if trimmed := strings.TrimSpace(part); trimmed != "" {
				cleaned = append(cleaned, trimmed)
			}
		}
		return strings.Join(cleaned, ", ")
	}

	subject := combine(spec.Theme, spec.Protagonist)
	if subject != "" {
		sentences = append(sentences, fmt.Sprintf("Craft a cinematic sequence that highlights %s.", subject))
	}
	if scene := strings.TrimSpace(spec.Scene); scene != "" {
		sentences = append(sentences, fmt.Sprintf("The story unfolds %s.", scene))
	}
	if action := combine(spec.Action); action != "" {
		sentences = append(sentences, fmt.Sprintf("Show the subject engaging in %s.", action))
	}

	var visualDirectives []string
	if lighting := strings.TrimSpace(spec.Lighting); lighting != "" {
		visualDirectives = append(visualDirectives, "lighting that feels "+lighting)
	}
	if camera := strings.TrimSpace(spec.Camera); camera != "" {
		visualDirectives = append(visualDirectives, "camera language featuring "+camera)
	}
	if movements := cleanList(spec.CameraMovements); len(movements) > 0 {
		visualDirectives = append(visualDirectives, "camera movement such as "+strings.Join(movements, ", "))
	}
	styleBlend := cleanList(spec.Stylization)
	if style := strings.TrimSpace(spec.Style); style != "" {
		styleBlend = append(styleBlend, style)
	}
	if spec.VisualStyle != "" {
		styleBlend = append(styleBlend, strings.TrimSpace(spec.VisualStyle))
	}
	if len(styleBlend) > 0 {
		visualDirectives = append(visualDirectives, "stylistic cues like "+strings.Join(styleBlend, ", "))
	}
	if len(visualDirectives) > 0 {
		sentences = append(sentences, "Visual direction: "+strings.Join(visualDirectives, "; ")+".")
	}

	if mood := strings.TrimSpace(spec.Mood); mood != "" {
		sentences = append(sentences, fmt.Sprintf("Evoke a mood of %s throughout the sequence.", mood))
	}
	if audio := strings.TrimSpace(spec.AudioMood); audio != "" {
		sentences = append(sentences, fmt.Sprintf("Underscore the visuals with audio that conveys %s.", audio))
	}

	if spec.Aesthetic != nil {
		var aestheticDetails []string
		if v := strings.TrimSpace(spec.Aesthetic.Atmosphere); v != "" {
			aestheticDetails = append(aestheticDetails, "atmosphere of "+v)
		}
		if v := strings.TrimSpace(spec.Aesthetic.ColorPalette); v != "" {
			aestheticDetails = append(aestheticDetails, "color palette emphasizing "+v)
		}
		if v := strings.TrimSpace(spec.Aesthetic.Lens); v != "" {
			aestheticDetails = append(aestheticDetails, "lens characteristics of "+v)
		}
		if len(aestheticDetails) > 0 {
			sentences = append(sentences, "Cinematic texture: "+strings.Join(aestheticDetails, "; ")+".")
		}
	}

	if additions := cleanList(spec.Additional); len(additions) > 0 {
		sentences = append(sentences, "Incorporate additional cues: "+strings.Join(additions, "; ")+".")
	}

	if trimmed := strings.TrimSpace(spec.UserPrompt); trimmed != "" {
		sentences = append(sentences, fmt.Sprintf("Honor the user's directive: \"%s\".", trimmed))
	}

	return strings.TrimSpace(strings.Join(sentences, " "))
}

func cleanList(values []string) []string {
	var cleaned []string
	for _, v := range values {
		if trimmed := strings.TrimSpace(v); trimmed != "" {
			cleaned = append(cleaned, trimmed)
		}
	}
	return cleaned
}
