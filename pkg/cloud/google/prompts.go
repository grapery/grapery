package google

import (
	"fmt"
	"strings"
	"text/template"
)

// PromptTemplates 提示词模板集合
var PromptTemplates = map[string]*PromptTemplate{
	// 图片生成模板
	"portrait_photo": {
		Name:        "portrait_photo",
		Description: "专业人像摄影模板",
		Template:    "A professional portrait photo of {{.subject}}, {{.description}}, {{.lighting}}, {{.background}}, {{.style}}, {{.camera_angle}}, high quality, detailed, realistic",
		Variables:   []string{"subject", "description", "lighting", "background", "style", "camera_angle"},
		Category:    "portrait",
		Tags:        []string{"portrait", "photo", "professional"},
		Examples: []PromptExample{
			{
				Title:       "商务人像",
				Description: "专业的商务人像照片",
				Prompt:      "A professional portrait photo of a businesswoman, wearing a navy blue blazer, soft natural lighting, clean white background, corporate style, eye-level angle, high quality, detailed, realistic",
				Result:      "生成专业的商务女性人像照片",
			},
		},
	},

	"product_photography": {
		Name:        "product_photography",
		Description: "产品摄影模板",
		Template:    "Product photography of {{.product}}, {{.description}}, {{.lighting}}, {{.background}}, {{.angle}}, {{.style}}, commercial quality, studio lighting, professional",
		Variables:   []string{"product", "description", "lighting", "background", "angle", "style"},
		Category:    "product",
		Tags:        []string{"product", "commercial", "photography"},
		Examples: []PromptExample{
			{
				Title:       "电子产品",
				Description: "现代电子产品的商业摄影",
				Prompt:      "Product photography of a smartphone, sleek modern design, soft studio lighting, minimalist white background, 45-degree angle, clean commercial style, commercial quality, studio lighting, professional",
				Result:      "生成现代智能手机的商业产品照片",
			},
		},
	},

	"landscape_art": {
		Name:        "landscape_art",
		Description: "风景艺术模板",
		Template:    "{{.art_style}} landscape of {{.location}}, {{.time_of_day}}, {{.weather}}, {{.composition}}, {{.mood}}, {{.details}}, masterpiece quality",
		Variables:   []string{"art_style", "location", "time_of_day", "weather", "composition", "mood", "details"},
		Category:    "landscape",
		Tags:        []string{"landscape", "art", "nature"},
		Examples: []PromptExample{
			{
				Title:       "日出山景",
				Description: "壮观的日出山景画作",
				Prompt:      "Oil painting landscape of mountain range, golden hour sunrise, clear sky, wide panoramic composition, majestic and peaceful mood, snow-capped peaks and misty valleys, masterpiece quality",
				Result:      "生成壮观的日出山景油画",
			},
		},
	},

	"logo_design": {
		Name:        "logo_design",
		Description: "标志设计模板",
		Template:    "{{.style}} logo design for {{.company}}, {{.industry}}, {{.elements}}, {{.colors}}, {{.typography}}, {{.concept}}, clean, professional, scalable",
		Variables:   []string{"style", "company", "industry", "elements", "colors", "typography", "concept"},
		Category:    "design",
		Tags:        []string{"logo", "design", "branding"},
		Examples: []PromptExample{
			{
				Title:       "科技公司标志",
				Description: "现代科技公司的标志设计",
				Prompt:      "Modern minimalist logo design for TechCorp, technology industry, geometric shapes and circuit patterns, blue and white colors, clean sans-serif typography, innovation and connectivity concept, clean, professional, scalable",
				Result:      "生成现代科技公司的标志设计",
			},
		},
	},

	// 视频生成模板
	"cinematic_scene": {
		Name:        "cinematic_scene",
		Description: "电影场景模板",
		Template:    "Cinematic {{.scene_type}} scene, {{.setting}}, {{.mood}}, {{.lighting}}, {{.camera_movement}}, {{.duration}}, {{.style}}, film quality, professional",
		Variables:   []string{"scene_type", "setting", "mood", "lighting", "camera_movement", "duration", "style"},
		Category:    "cinematic",
		Tags:        []string{"cinematic", "video", "film"},
		Examples: []PromptExample{
			{
				Title:       "城市夜景",
				Description: "电影级的城市夜景场景",
				Prompt:      "Cinematic establishing shot scene, bustling city at night, mysterious and vibrant mood, neon lighting, slow push-in camera movement, 10 seconds, noir style, film quality, professional",
				Result:      "生成电影级的城市夜景视频",
			},
		},
	},

	"product_demo": {
		Name:        "product_demo",
		Description: "产品演示模板",
		Template:    "Product demonstration video of {{.product}}, {{.features}}, {{.setting}}, {{.lighting}}, {{.camera_angles}}, {{.duration}}, {{.style}}, commercial quality",
		Variables:   []string{"product", "features", "setting", "lighting", "camera_angles", "duration", "style"},
		Category:    "commercial",
		Tags:        []string{"product", "demo", "commercial"},
		Examples: []PromptExample{
			{
				Title:       "智能手表演示",
				Description: "智能手表的功能演示视频",
				Prompt:      "Product demonstration video of smartwatch, fitness tracking and notifications, modern lifestyle setting, bright natural lighting, multiple camera angles, 15 seconds, clean modern style, commercial quality",
				Result:      "生成智能手表的功能演示视频",
			},
		},
	},

	"nature_documentary": {
		Name:        "nature_documentary",
		Description: "自然纪录片模板",
		Template:    "Nature documentary style video, {{.subject}}, {{.environment}}, {{.behavior}}, {{.lighting}}, {{.camera_work}}, {{.duration}}, {{.mood}}, BBC quality",
		Variables:   []string{"subject", "environment", "behavior", "lighting", "camera_work", "duration", "mood"},
		Category:    "documentary",
		Tags:        []string{"nature", "documentary", "wildlife"},
		Examples: []PromptExample{
			{
				Title:       "鸟类觅食",
				Description: "鸟类觅食的自然纪录片片段",
				Prompt:      "Nature documentary style video, eagle hunting, mountain forest environment, soaring and diving behavior, golden hour lighting, aerial and ground camera work, 20 seconds, majestic and natural mood, BBC quality",
				Result:      "生成老鹰觅食的自然纪录片视频",
			},
		},
	},
}

// PromptTips 提示词技巧库
var PromptTips = map[string][]string{
	"lighting": {
		"soft natural lighting", "dramatic lighting", "golden hour lighting",
		"studio lighting", "neon lighting", "candlelight", "moonlight",
		"backlighting", "side lighting", "rim lighting",
	},

	"camera_angles": {
		"eye-level angle", "low angle", "high angle", "bird's eye view",
		"worm's eye view", "dutch angle", "over-the-shoulder",
		"close-up", "wide shot", "medium shot",
	},

	"art_styles": {
		"photorealistic", "oil painting", "watercolor", "digital art",
		"sketch", "anime style", "cartoon style", "vintage style",
		"modern minimalist", "abstract", "impressionist", "surreal",
	},

	"moods": {
		"peaceful", "dramatic", "mysterious", "vibrant", "melancholic",
		"energetic", "serene", "intense", "playful", "elegant",
		"nostalgic", "futuristic", "romantic", "heroic",
	},

	"compositions": {
		"rule of thirds", "centered composition", "leading lines",
		"symmetrical", "asymmetrical", "diagonal", "triangular",
		"golden ratio", "framing", "layering",
	},

	"camera_movements": {
		"static shot", "pan left/right", "tilt up/down", "dolly in/out",
		"truck left/right", "boom up/down", "arc movement", "handheld",
		"steady cam", "drone shot", "crane shot",
	},

	"video_styles": {
		"cinematic", "documentary", "commercial", "music video",
		"corporate", "lifestyle", "tutorial", "news", "social media",
		"vlog", "cinematic", "artistic",
	},
}

// DefaultPromptManager 默认提示词管理器
type DefaultPromptManager struct {
	templates map[string]*PromptTemplate
}

// NewPromptManager 创建提示词管理器
func NewPromptManager() *DefaultPromptManager {
	return &DefaultPromptManager{
		templates: PromptTemplates,
	}
}

// GetTemplate 获取模板
func (pm *DefaultPromptManager) GetTemplate(name string) (*PromptTemplate, error) {
	template, exists := pm.templates[name]
	if !exists {
		return nil, ErrTemplateNotFound
	}
	return template, nil
}

// ListTemplates 列出所有模板
func (pm *DefaultPromptManager) ListTemplates(category string) ([]*PromptTemplate, error) {
	var templates []*PromptTemplate
	for _, template := range pm.templates {
		if category == "" || template.Category == category {
			templates = append(templates, template)
		}
	}
	return templates, nil
}

// CreateTemplate 创建模板
func (pm *DefaultPromptManager) CreateTemplate(template *PromptTemplate) error {
	if template.Name == "" {
		return ErrInvalidTemplate
	}
	pm.templates[template.Name] = template
	return nil
}

// UpdateTemplate 更新模板
func (pm *DefaultPromptManager) UpdateTemplate(template *PromptTemplate) error {
	if template.Name == "" {
		return ErrInvalidTemplate
	}
	pm.templates[template.Name] = template
	return nil
}

// DeleteTemplate 删除模板
func (pm *DefaultPromptManager) DeleteTemplate(name string) error {
	if _, exists := pm.templates[name]; !exists {
		return ErrTemplateNotFound
	}
	delete(pm.templates, name)
	return nil
}

// RenderTemplate 渲染模板
func (pm *DefaultPromptManager) RenderTemplate(templateName string, variables map[string]string) (string, error) {
	tmpl, err := pm.GetTemplate(templateName)
	if err != nil {
		return "", err
	}

	t, err := template.New("prompt").Parse(tmpl.Template)
	if err != nil {
		return "", ErrRenderingFailed
	}

	var buf strings.Builder
	err = t.Execute(&buf, variables)
	if err != nil {
		return "", ErrRenderingFailed
	}

	return buf.String(), nil
}

// GetPromptTips 获取提示词技巧
func GetPromptTips(category string) []string {
	if tips, exists := PromptTips[category]; exists {
		return tips
	}
	return []string{}
}

// BuildAdvancedPrompt 构建高级提示词
func BuildAdvancedPrompt(basePrompt string, options map[string]string) string {
	var parts []string

	// 添加基础提示词
	parts = append(parts, basePrompt)

	// 添加风格
	if style, exists := options["style"]; exists && style != "" {
		parts = append(parts, style+" style")
	}

	// 添加质量描述
	if quality, exists := options["quality"]; exists && quality != "" {
		parts = append(parts, quality+" quality")
	} else {
		parts = append(parts, "high quality")
	}

	// 添加技术细节
	if details, exists := options["details"]; exists && details != "" {
		parts = append(parts, details)
	}

	// 添加负面提示词
	if negative, exists := options["negative"]; exists && negative != "" {
		parts = append(parts, "avoiding "+negative)
	}

	return strings.Join(parts, ", ")
}

// ValidatePrompt 验证提示词
func ValidatePrompt(prompt string) error {
	if strings.TrimSpace(prompt) == "" {
		return ErrEmptyPrompt
	}

	// 检查提示词长度
	if len(prompt) > 4000 {
		return fmt.Errorf("提示词过长，最大长度为4000字符")
	}

	// 检查是否包含敏感内容（简单检查）
	sensitiveWords := []string{"explicit", "adult", "nsfw", "violence", "hate"}
	lowerPrompt := strings.ToLower(prompt)
	for _, word := range sensitiveWords {
		if strings.Contains(lowerPrompt, word) {
			return fmt.Errorf("提示词包含敏感内容")
		}
	}

	return nil
}
