package story

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
type ChapterDetailInformation struct {
	Details []*DetailScene `json:"详细情节,omitempty"`
}

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

// StoryInfo 表示故事的完整信息
type StoryInfo struct {
	StoryNameAndTheme StoryNameAndTheme `json:"故事名称和主题,omitempty"`
	StoryChapters     []ChapterInfo     `json:"故事章节,omitempty"`
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

// Example usage:
/*
chapter := &StoryChapter{
	ChapterSummary: ChapterSummary{
		Title:   "地球生存环境恶化",
		Content: "地球资源日益枯竭，人类将目光投向了火星...",
	},
	ChapterDetailInfo: ChapterDetailInformation{
		Details: map[string]DetailScene{
			"详细情节-1": {
				Content: "气候变化，温室效应加剧...",
				Characters: []Character{
					{
						ID:          "1",
						Name:        "马克",
						Description: "马克是一名经验丰富的宇航员...",
					},
				},
				ImagePrompt: "一个城市被严重的雾霾笼罩...",
			},
		},
	},
}

story := &StoryInfo{
	StoryNameAndTheme: StoryNameAndTheme{
		Name:        "火星绿洲",
		Theme:       "人类在火星上的生存",
		Description: "在2023年，国际火星探索任务成功地将首批人类送至火星...",
	},
	StoryChapters: []ChapterInfo{
		{
			ID:      "1",
			Title:   "火星上的孤岛",
			Content: "马克在火星表面执行任务时，遭遇了一场突如其来的沙尘暴...",
		},
		{
			ID:      "2",
			Title:   "生存挑战",
			Content: "马克意识到自己必须生存下去...",
		},
	},
}
*/
