package story

import "encoding/json"

// StoryChapter 表示故事章节的完整结构
type StoryChapter struct {
	ChapterSummary    ChapterSummary           `json:"章节情节简述,omitempty"`
	ChapterDetailInfo ChapterDetailInformation `json:"章节详细情节,omitempty"`
}

type StoryChapterV2 struct {
	ChapterSummary ChapterSummary `json:"章节情节简述,omitempty"`
	Characters     []Character    `json:"参与人物,omitempty"`
}

// ChapterSummary 表示章节的基本信息
type ChapterSummary struct {
	Title      string      `json:"章节题目,omitempty"`
	Content    string      `json:"章节内容,omitempty"`
	Characters []Character `json:"参与人物,omitempty"`
}

// ChapterDetailInformation 包含多个详细情节
type ChapterDetailInformation []*DetailScene

// DetailScene 表示具体的场景信息
type DetailScene struct {
	ID          string      `json:"情节id,omitempty"`
	Content     string      `json:"情节内容,omitempty"`
	Characters  []Character `json:"参与人物,omitempty"`
	ImagePrompt string      `json:"图片提示词,omitempty"`
}

// Character 表示角色信息
type Character struct {
	ID          string `json:"角色id,omitempty"`
	Name        string `json:"角色姓名,omitempty"`
	Description string `json:"角色描述,omitempty"`
}

type CharacterDetailConverter struct {
	Description     string `json:"description,omitempty"`
	ShortTermGoal   string `json:"short_term_goal,omitempty"`
	LongTermGoal    string `json:"long_term_goal,omitempty"`
	Personality     string `json:"personality,omitempty"`
	Background      string `json:"background,omitempty"`
	HandlingStyle   string `json:"handling_style,omitempty"`
	CognitionRange  string `json:"cognition_range,omitempty"`
	AbilityFeatures string `json:"ability_features,omitempty"`
	Appearance      string `json:"appearance,omitempty"`
	DressPreference string `json:"dress_preference,omitempty"`
}

func (c *CharacterDetailConverter) ToPrompt() string {
	// 将 CharacterDetailConverter 转换为适合提示的字符串格式
	return "角色描述: " + c.Description + "\n" +
		"短期目标: " + c.ShortTermGoal + "\n" +
		"长期目标: " + c.LongTermGoal + "\n" +
		"性格特征: " + c.Personality + "\n" +
		"角色背景: " + c.Background + "\n" +
		"处事风格: " + c.HandlingStyle + "\n" +
		"认知范围: " + c.CognitionRange + "\n" +
		"能力特点: " + c.AbilityFeatures + "\n" +
		"外貌特征: " + c.Appearance + "\n" +
		"穿着喜好: " + c.DressPreference
}

// CharacterDetail 表示角色的详细信息
type CharacterDetail struct {
	Description   string `json:"角色描述,omitempty"`
	ShortTermGoal string `json:"角色短期目标,omitempty"`
	LongTermGoal  string `json:"角色长期目标,omitempty"`
	Personality   string `json:"性格特征,omitempty"`
	Background    string `json:"角色背景,omitempty"`
	// 处事风格
	HandlingStyle string `json:"处事风格,omitempty"`
	// 认知范围
	CognitionRange string `json:"认知范围,omitempty"`
	// 能力特点
	AbilityFeatures string `json:"能力特点,omitempty"`
	// 外貌特征
	Appearance string `json:"外貌特征,omitempty"`
	// 穿着喜好
	DressPreference string `json:"穿着喜好,omitempty"`
}

func (c *CharacterDetail) String() string {
	json, err := json.Marshal(c)
	if err != nil {
		return ""
	}
	return string(json)
}

// StoryInfo 表示故事的完整信息
type StoryInfo struct {
	StoryNameAndTheme StoryNameAndTheme `json:"故事名称和主题,omitempty"`
	StoryChapters     []ChapterInfo     `json:"故事章节,omitempty"`
}

func (s *StoryInfo) String() string {
	json, err := json.Marshal(s)
	if err != nil {
		return ""
	}
	return string(json)
}

// StoryNameAndTheme 表示故事的名称和主题信息
type StoryNameAndTheme struct {
	Name        string `json:"故事名称,omitempty"`
	Theme       string `json:"故事主题,omitempty"`
	Description string `json:"故事简介,omitempty"`
}

// ChapterInfo 表示单个章节的信息
type ChapterInfo struct {
	ID      string `json:"章节ID,omitempty"`
	Title   string `json:"章节题目,omitempty"`
	Content string `json:"章节内容,omitempty"`
}
