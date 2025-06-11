package models

import (
	"context"

	"gorm.io/gorm"
)

const (
	ComicStyle  = "Japanese Anime"
	CinanaStyle = "Manga"
	AnimeStyle  = "Anime"
)

const (
	LayoutClassicStyle   = "Classic Comic Style"
	LayoutFourPanelStyle = "Four Pannel"
)

const (
	NegativePrompt = ``
	PositivePrompt = ``
)

type PromptType string

const (
	PromptTypeStory      PromptType = "story"
	PromptTypeStoryboard PromptType = "storyboard"
	PromptTypeImage      PromptType = "image"
	PromptTypeVideo      PromptType = "video"
	PromptTypeAudio      PromptType = "audio"
	PromptTypeText       PromptType = "text"
)

type PromptTemplate struct {
	Name           string     `json:"name"`
	Prompt         string     `json:"prompt"`
	NegativePrompt string     `json:"negative_prompt"`
	PromptType     PromptType `json:"prompt_type"`
}

var PreDefineTemplateEnVersion = []PromptTemplate{
	{
		Name:           "(No style)",
		Prompt:         "{prompt}",
		NegativePrompt: "",
		PromptType:     PromptTypeImage,
	},
	{
		Name:           "Japanese Anime",
		Prompt:         "anime artwork illustrating {prompt}. created by japanese anime studio. highly emotional. best quality, high resolution",
		NegativePrompt: "low quality, low resolution",
		PromptType:     PromptTypeImage,
	},
	{
		Name:           "Cinematic",
		Prompt:         "cinematic still {prompt} . emotional, harmonious, vignette, highly detailed, high budget, bokeh, cinemascope, moody, epic, gorgeous, film grain, grainy",
		NegativePrompt: "anime, cartoon, graphic, text, painting, crayon, graphite, abstract, glitch, deformed, mutated, ugly, disfigured",
		PromptType:     PromptTypeImage,
	},
	{
		Name:           "Disney Charactor",
		Prompt:         "A Pixar animation character of {prompt} . pixar-style, studio anime, Disney, high-quality",
		NegativePrompt: "lowres, bad anatomy, bad hands, text, bad eyes, bad arms, bad legs, error, missing fingers, extra digit, fewer digits, cropped, worst quality, low quality, normal quality, jpeg artifacts, signature, watermark, blurry, grayscale, noisy, sloppy, messy, grainy, highly detailed, ultra textured, photo",
		PromptType:     PromptTypeImage,
	},
	{
		Name:           "Photographic",
		Prompt:         "cinematic photo {prompt} . 35mm photograph, film, bokeh, professional, 4k, highly detailed",
		NegativePrompt: "drawing, painting, crayon, sketch, graphite, impressionist, noisy, blurry, soft, deformed, ugly",
		PromptType:     PromptTypeImage,
	},
	{
		Name:           "Comic book",
		Prompt:         "comic {prompt} . graphic illustration, comic art, graphic novel art, vibrant, highly detailed",
		NegativePrompt: "photograph, deformed, glitch, noisy, realistic, stock photo",
		PromptType:     PromptTypeImage,
	},
	{
		Name:           "Line art",
		Prompt:         "line art drawing {prompt} . professional, sleek, modern, minimalist, graphic, line art, vector graphics",
		NegativePrompt: "anime, photorealistic, 35mm film, deformed, glitch, blurry, noisy, off-center, deformed, cross-eyed, closed eyes, bad anatomy, ugly, disfigured, mutated, realism, realistic, impressionism, expressionism, oil, acrylic",
		PromptType:     PromptTypeImage,
	},
	{
		Name:           "Studio Ghibli",
		Prompt:         "Studio Ghibli anime style {prompt}. Miyazaki style, whimsical, colorful, fantasy, detailed background, hand-drawn animation, magical, dreamy atmosphere",
		NegativePrompt: "3D, photorealistic, dark, gritty, low quality, deformed, simple background",
		PromptType:     PromptTypeImage,
	},
	{
		Name:           "Pencil Sketch",
		Prompt:         "detailed pencil sketch of {prompt}. traditional art, graphite, shading, realistic drawing technique, fine details, textured paper, monochrome, artistic",
		NegativePrompt: "digital art, color, painting, cartoon, anime, 3D, flat, lack of texture, lack of detail",
		PromptType:     PromptTypeImage,
	},
	{
		Name:           "Stick Figure",
		Prompt:         "stick figure drawing of {prompt}. simple lines, minimalist, black and white, basic shapes, clean drawing, conceptual, expressive",
		NegativePrompt: "detailed, realistic, complex, textured, colorful, shaded, 3D, photorealistic",
		PromptType:     PromptTypeImage,
	},
	{
		Name:           "Clay Sculpture",
		Prompt:         "clay sculpture of {prompt}. 3D modeling, handmade, textured surface, stop motion style, claymation, tactile, physical medium, studio lighting",
		NegativePrompt: "2D, flat, drawing, digital art, smooth surface, cartoon, anime",
		PromptType:     PromptTypeImage,
	},
}

var PreDefineTemplateChVersion = []PromptTemplate{
	{
		Name:           "（无风格）",
		Prompt:         "{prompt}",
		NegativePrompt: "",
		PromptType:     PromptTypeImage,
	},
	{
		Name:           "日式动漫",
		Prompt:         "日本动漫风格插图展示{prompt}。由日本动漫工作室创作。情感强烈。最佳质量，高分辨率",
		NegativePrompt: "低质量，低分辨率",
		PromptType:     PromptTypeImage,
	},
	{
		Name:           "电影风格",
		Prompt:         "电影画面静帧{prompt}。富有情感，和谐，晕影效果，高度细节，高预算，景深效果，宽银幕，情绪化，史诗级，华丽，胶片颗粒感",
		NegativePrompt: "动漫，卡通，图形，文字，绘画，蜡笔，铅笔，抽象，故障，变形，突变，丑陋，畸形",
		PromptType:     PromptTypeImage,
	},
	{
		Name:           "迪士尼角色",
		Prompt:         "皮克斯动画风格的{prompt}角色。皮克斯风格，工作室动画，迪士尼，高质量",
		NegativePrompt: "低分辨率，错误的解剖结构，不良的手部，文本，错误的眼睛，不良的手臂，不良的腿部，错误，缺失的手指，多余的手指，较少的手指，裁剪，最差质量，低质量，普通质量，JPEG伪影，签名，水印，模糊，灰度，噪点，草率，凌乱，颗粒感，高度细节，超质感，照片",
		PromptType:     PromptTypeImage,
	},
	{
		Name:           "摄影风格",
		Prompt:         "电影摄影照片{prompt}。35毫米照片，胶片，散景，专业，4K，高度细节",
		NegativePrompt: "绘图，绘画，蜡笔，素描，铅笔，印象派，嘈杂，模糊，柔和，变形，丑陋",
		PromptType:     PromptTypeImage,
	},
	{
		Name:           "漫画书",
		Prompt:         "漫画{prompt}。图形插图，漫画艺术，图像小说艺术，色彩鲜明，高度细节",
		NegativePrompt: "照片，变形，故障，嘈杂，真实主义，库存照片",
		PromptType:     PromptTypeImage,
	},
	{
		Name:           "线条艺术",
		Prompt:         "线条艺术绘画{prompt}。专业，流畅，现代，极简主义，图形，线条艺术，矢量图形",
		NegativePrompt: "动漫，写实，35毫米胶片，变形，故障，模糊，嘈杂，偏离中心，变形，斗鸡眼，闭眼，解剖结构不良，丑陋，畸形，突变，现实主义，写实主义，印象派，表现主义，油彩，丙烯",
		PromptType:     PromptTypeImage,
	},
	{
		Name:           "中国水墨",
		Prompt:         "中国传统水墨画风格{prompt}。水墨山水，淡雅，意境深远，留白，书法元素，高质量",
		NegativePrompt: "西方风格，照片，彩色鲜艳，过度细节，现代，数字艺术",
		PromptType:     PromptTypeImage,
	},
	{
		Name:           "国风插画",
		Prompt:         "中国风插画{prompt}。传统与现代结合，色彩丰富，东方美学，精致细腻，国风元素",
		NegativePrompt: "西方风格，照片写实，黑白，草稿，简笔",
		PromptType:     PromptTypeImage,
	},
	{
		Name:           "吉卜力",
		Prompt:         "吉卜力工作室动画风格{prompt}。宫崎骏风格，奇幻梦幻，色彩鲜明，精细背景，手绘动画，魔法氛围，梦幻场景",
		NegativePrompt: "3D，照片写实，黑暗，粗糙，低质量，变形，简单背景",
		PromptType:     PromptTypeImage,
	},
	{
		Name:           "铅笔画",
		Prompt:         "精细铅笔素描{prompt}。传统艺术，石墨，阴影，写实绘画技巧，精细细节，纹理纸张，单色，艺术感",
		NegativePrompt: "数字艺术，彩色，绘画，卡通，动漫，3D，平面，缺乏纹理，缺乏细节",
		PromptType:     PromptTypeImage,
	},
	{
		Name:           "火柴人",
		Prompt:         "火柴人绘画{prompt}。简单线条，极简主义，黑白，基本形状，干净绘图，概念性，富有表现力",
		NegativePrompt: "详细，写实，复杂，纹理，彩色，阴影，3D，照片写实",
		PromptType:     PromptTypeImage,
	},
	{
		Name:           "泥塑风格",
		Prompt:         "泥塑作品{prompt}。3D建模，手工制作，纹理表面，定格动画风格，黏土动画，触感强，实体媒介，工作室灯光",
		NegativePrompt: "2D，平面，绘画，数字艺术，光滑表面，卡通，动漫",
		PromptType:     PromptTypeImage,
	},
}

// Prompt 提示词/生成参数
type Prompt struct {
	IDBase
	UserID      int64  `gorm:"column:user_id" json:"user_id,omitempty"`         // 用户ID
	Type        int    `gorm:"column:type" json:"type,omitempty"`               // 类型
	Content     string `gorm:"column:content" json:"content,omitempty"`         // 内容
	Status      int    `gorm:"column:status" json:"status,omitempty"`           // 状态
	Description string `gorm:"column:description" json:"description,omitempty"` // 描述
}

func NewPrompt() *Prompt {
	return &Prompt{}
}
func (p *Prompt) TableName() string {
	return "prompt"
}

func (p *Prompt) Create(ctx context.Context) error {
	return DataBase().Table(p.TableName()).WithContext(ctx).Create(p).Error
}

func (p *Prompt) Update(ctx context.Context) error {
	return DataBase().Table(p.TableName()).WithContext(ctx).Save(p).Error
}

func (p *Prompt) Delete(ctx context.Context) error {
	return DataBase().Table(p.TableName()).WithContext(ctx).Delete(p).Error
}

func GetPrompt(ctx context.Context, id int64) (*Prompt, error) {
	var prompt Prompt
	p, err := &prompt, DataBase().Table(prompt.TableName()).WithContext(ctx).Where("id = ?", id).First(&prompt).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return p, nil
}

func ListPrompt(ctx context.Context, userID int64, promptType PromptType) ([]*Prompt, error) {
	var prompts []*Prompt
	query := DataBase().Model(Prompt{}).WithContext(ctx)
	if userID != 0 {
		query = query.Where("user_id = ?", userID)
	}
	if promptType != "" {
		query = query.Where("prompt_type = ?", promptType)
	}
	err := query.Find(&prompts).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return prompts, nil
}

func ListPromptByGroupID(ctx context.Context, groupID int64, promptType PromptType) ([]*Prompt, error) {
	var prompts []*Prompt
	query := DataBase().Model(Prompt{}).WithContext(ctx)
	if groupID != 0 {
		query = query.Where("group_id = ?", groupID)
	}
	if promptType != "" {
		query = query.Where("prompt_type = ?", promptType)
	}
	err := query.Find(&prompts).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return prompts, nil
}

func ListPromptName(ctx context.Context, promptType PromptType) ([]string, error) {
	var prompts []string
	err := DataBase().Model(Prompt{}).WithContext(ctx).
		Where("prompt_type = ?", promptType).
		Pluck("name", &prompts).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return prompts, nil
}

// 新增：分页获取Prompt列表
func GetPromptList(ctx context.Context, offset, limit int) ([]*Prompt, error) {
	var prompts []*Prompt
	err := DataBase().Model(&Prompt{}).
		WithContext(ctx).
		Offset(offset).
		Limit(limit).
		Order("create_at desc").
		Find(&prompts).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return nil, err
	}
	return prompts, nil
}

// 新增：通过主键唯一查询
func GetPromptByID(ctx context.Context, id int64) (*Prompt, error) {
	prompt := &Prompt{}
	err := DataBase().Model(prompt).
		WithContext(ctx).
		Where("id = ?", id).
		First(prompt).Error
	if err != nil {
		return nil, err
	}
	return prompt, nil
}
