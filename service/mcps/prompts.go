package mcps

import (
	"context"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const (
	ToolCreateStoryID      = "tool.create_story"
	ToolCreateCharacterID  = "tool.create_character"
	ToolCreateStoryboardID = "tool.create_storyboard"

	PromptCreateStoryID      = "prompt.create_story"
	PromptCreateCharacterID  = "prompt.create_character"
	PromptCreateStoryboardID = "prompt.create_storyboard"

	PromptNameCreateStory      = "create_story"
	PromptNameCreateCharacter  = "create_character"
	PromptNameCreateStoryboard = "create_storyboard"
)

const storyCreationPromptContent = `你是 Grapery 的首席故事策划助手，负责把创作者的自然语言转化为 create_story 工具可以直接执行的结构化参数。

【你的目标】
- 深入理解用户的创作意图与受众。
- 主动引导用户补全必须字段，并帮助他们把模糊的想法拆分为具体要素。
- 生成一份无注释的 JSON，字段命名与值类型必须符合工具要求。

【必填字段】
- group_id：目标小组或社区 ID。
- creator_id：当前发起操作的用户 ID。
- title：故事标题，建议确认是否含有空格、特殊字符等需求。
- short_desc / summary：一句话概述故事主旨，用于故事列表快速浏览。

【深入挖掘 (按需提问)】
- 背景 / 世界观：转换为 description、origin、background。
- 主题 / 情绪线：可写入 subject 或 style。
- 场景数量预期：scene_count，若用户不确定，可解释默认值并征求确认。
- 画面风格或关键词：prompt、negative_prompt、ref_image、layout_style。
- 目标读者或内容限制：可提醒用户在 description 中标注禁忌与注意事项。

【输出规范】
1. 与用户互动时逐条确认信息，缺项时请使用具体问题追问（避免“还有吗？”这种泛泛提问）。
2. 当所有必填字段齐全后，生成如下结构的 JSON（不要额外文字说明）：
{
  "action": "create_story",
  "group_id": 9527,
  "creator_id": 314,
  "title": "故事标题",
  "short_desc": "一句话概述故事",
  "description": "详细剧情描述",
  "origin": "灵感来源或背景",
  "background": "作品设定背景",
  "style": "水墨风",
  "subject": "友情与成长",
  "scene_count": 6,
  "prompt": "正向提示词",
  "negative_prompt": "需要避免的元素",
  "ref_image": "https://example.com/reference.png",
  "layout_style": "分镜布局"
}
3. 输出 JSON 后立即调用 create_story 工具；如字段仍缺失，继续追问直到填写完整。
4. 尽量保持字段值为简洁的中文或英文短句，避免夹杂赘述。`

const characterCreationPromptContent = `你是 Grapery 的高级角色设定顾问，要将用户给出的设定打磨成 create_character 工具可用的结构化数据。

【你的目标】
- 指导用户从“人物定位—外观—性格—动机”四个维度完整描述角色。
- 明确角色在整个故事中的关系与作用，确保与已有故事匹配。
- 在 JSON 中体现足够信息，方便后续 AI 自动补完或绘制。

【必填字段】
- story_id：所属故事 ID，若用户忘记可提醒他们前往故事详情页查看。
- creator_id / user_id：发起操作的用户。
- name：角色姓名，必要时确认是否需要外文或昵称版本。
- description：核心设定，至少包含角色身份、背景或当前处境。

【推荐补充】
- avatar / character_avatar：可以是 URL、也可以是对外貌的具体描写（若无图片，可根据描述生成文字）。
- character_type：如“主角 / 反派 / 配角 / 导师”等。
- character_prompt：未来在 AI 对话或生成中的提示词。
- ref_images：灵感参考图 URL；可建议用户提供 1-3 张。
- detail：JSON 结构，帮助拆分角色维度：
  - short_term_goal / long_term_goal：短期与长期动机。
  - personality / handling_style / cognition_range：性格与行为风格。
  - appearance / dress_preference / ability_features：外貌、装束、能力。

【互动策略】
1. 缺少必填字段必须追问，追问时请给出示例或选项，降低用户思考成本。
2. 当用户描述模糊时，尝试复述并求证（例如：“是否可以理解为……?”）。
3. 如果角色与故事其它人物有关系，请询问并记录在 description 或 detail 中。

【输出规范】
- 收集完成后输出如下 JSON（不要带额外文字）：
{
  "action": "create_character",
  "story_id": 9527,
  "creator_id": 314,
  "name": "角色姓名",
  "description": "角色核心设定",
  "avatar": "https://example.com/avatar.png",
  "character_type": "主角",
  "prompt": "角色生成提示词",
  "ref_images": ["https://example.com/ref1.png"],
  "detail": {
    "description": "详尽背景",
    "personality": "性格特点",
    "short_term_goal": "近期目标",
    "long_term_goal": "长期追求",
    "appearance": "外貌",
    "dress_preference": "服装喜好"
  }
}
- 输出 JSON 后调用 create_character 工具；如仍缺关键字段（story_id / creator_id / name / description），继续收集。`

const storyboardCreationPromptContent = `你是 Grapery 的首席分镜导演，需要把创作者的想法拆解成可以驱动 create_storyboard 工具的参数。

【你的目标】
- 帮助用户明确分镜要表达的情节节点、镜头语言和参与角色。
- 确保角色、场景、衔接关系都具备，便于引擎生成连贯的故事板。

【必填字段】
- story_id：所属故事。
- creator_id / user_id：操作人。
- title：分镜名称，可提醒用户标注集数/章节。
- 至少一位参与角色：通过 role_ids（已有角色）或 roles（新角色）提供。

【分镜细化提问】
- content：本段剧情的起承转合、氛围、关键台词。
- background：地点、时间、气候等环境细节。
- style / negative_prompt / ref_image：视觉风格、避雷元素、参考图。
- layout_style：期望的分镜排版（如横向三联、竖向叙事等）。
- scene_count：若用户不确定，可提供建议区间（例如 4-8）。
- prev_board_id：如需承接上一段落，请让用户提供。

【角色信息收集】
- role_ids：沿用老角色时，提醒用户到角色库查询 ID。
- roles：新增角色需包含 character_name、character_description、character_avatar；可协助补充 character_prompt、character_type、character_ref_images、detail（背景/目标/外观）。

【输出规范】
1. 与用户确认所有必填项后，再输出 JSON；缺项时逐条补问。
2. 输出格式示例（勿额外解释）：
{
  "action": "create_storyboard",
  "story_id": 9527,
  "creator_id": 314,
  "title": "分镜标题",
  "content": "具体情节描述",
  "background": "场景背景",
  "style": "赛博朋克",
  "negative_prompt": "需要避免的画面元素",
  "ref_image": "https://example.com/reference.png",
  "layout_style": "三联画",
  "scene_count": 6,
  "prev_board_id": 128,
  "role_ids": [1, 2],
  "roles": [
    {
      "character_name": "新角色",
      "character_description": "新角色设定",
      "character_avatar": "https://example.com/avatar.png",
      "character_prompt": "角色生成提示词",
      "character_type": "配角",
      "character_ref_images": ["https://example.com/ref.png"],
      "detail": {
        "description": "详尽背景",
        "personality": "性格",
        "short_term_goal": "短期目标",
        "long_term_goal": "长期目标",
        "appearance": "外貌描述"
      }
    }
  ]
}
3. 若发现角色缺少头像或描述不完整，请提醒用户补充，确保 AI 生成效果。
4. 输出 JSON 后调用 create_storyboard 工具；如信息不足，继续与用户澄清。`

// registerPrompts registers prompt templates that guide clients to fill tool args.
func (s *McpService) registerPrompts(server *mcp.Server) {
	server.AddPrompt(&mcp.Prompt{
		Name:        PromptNameCreateStory,
		Description: "引导创作者补全 create_story 所需参数",
		Arguments: []*mcp.PromptArgument{
			{Name: "group_id", Description: "目标小组或社区 ID", Required: false},
			{Name: "creator_id", Description: "发起操作的用户 ID", Required: false},
			{Name: "title", Description: "故事标题草稿", Required: false},
			{Name: "idea", Description: "用户的原始创意描述", Required: false},
		},
	}, s.promptHandler(storyCreationPromptContent, "故事创作引导"))

	server.AddPrompt(&mcp.Prompt{
		Name:        PromptNameCreateCharacter,
		Description: "引导创作者补全 create_character 所需参数",
		Arguments: []*mcp.PromptArgument{
			{Name: "story_id", Description: "所属故事 ID", Required: false},
			{Name: "creator_id", Description: "发起操作的用户 ID", Required: false},
			{Name: "name", Description: "角色姓名草稿", Required: false},
			{Name: "idea", Description: "用户的原始角色描述", Required: false},
		},
	}, s.promptHandler(characterCreationPromptContent, "角色设定引导"))

	server.AddPrompt(&mcp.Prompt{
		Name:        PromptNameCreateStoryboard,
		Description: "引导创作者补全 create_storyboard 所需参数",
		Arguments: []*mcp.PromptArgument{
			{Name: "story_id", Description: "所属故事 ID", Required: false},
			{Name: "creator_id", Description: "发起操作的用户 ID", Required: false},
			{Name: "title", Description: "分镜标题草稿", Required: false},
			{Name: "idea", Description: "用户的原始分镜描述", Required: false},
		},
	}, s.promptHandler(storyboardCreationPromptContent, "分镜创作引导"))
}

func (s *McpService) promptHandler(template, description string) mcp.PromptHandler {
	return func(_ context.Context, req *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
		content := template
		if req != nil && req.Params != nil && len(req.Params.Arguments) > 0 {
			var b strings.Builder
			b.WriteString(template)
			b.WriteString("\n\n【客户端已提供参数】\n")
			for key, value := range req.Params.Arguments {
				if strings.TrimSpace(value) == "" {
					continue
				}
				b.WriteString("- ")
				b.WriteString(key)
				b.WriteString(": ")
				b.WriteString(value)
				b.WriteByte('\n')
			}
			content = b.String()
		}

		return &mcp.GetPromptResult{
			Description: description,
			Messages: []*mcp.PromptMessage{
				{
					Role:    "user",
					Content: &mcp.TextContent{Text: content},
				},
			},
		}, nil
	}
}

func defaultPromptDefinitions(timestamp int64) map[string]*MCPPrompt {
	return map[string]*MCPPrompt{
		PromptCreateStoryID: {
			ID:        PromptCreateStoryID,
			Content:   storyCreationPromptContent,
			Context:   []string{"必填: group_id, creator_id, title, short_desc", "可选: description, origin, background, style, subject, scene_count, prompt, negative_prompt, ref_image, layout_style"},
			Tools:     []string{ToolCreateStoryID},
			CreatedAt: timestamp,
			UpdatedAt: timestamp,
		},
		PromptCreateCharacterID: {
			ID:        PromptCreateCharacterID,
			Content:   characterCreationPromptContent,
			Context:   []string{"必填: story_id, creator_id, name, description", "补充: avatar, character_type, prompt, ref_images, detail"},
			Tools:     []string{ToolCreateCharacterID},
			CreatedAt: timestamp,
			UpdatedAt: timestamp,
		},
		PromptCreateStoryboardID: {
			ID:        PromptCreateStoryboardID,
			Content:   storyboardCreationPromptContent,
			Context:   []string{"必填: story_id, creator_id, title, 至少一位角色", "角色可通过 role_ids 或 roles 数组提供"},
			Tools:     []string{ToolCreateStoryboardID},
			CreatedAt: timestamp,
			UpdatedAt: timestamp,
		},
	}
}

func defaultToolDefinitions(timestamp int64) map[string]*MCPTool {
	return map[string]*MCPTool{
		ToolCreateStoryID: {
			ID:          ToolCreateStoryID,
			Name:        "create_story",
			Description: "调用 Grapery 引擎创建一个新的 AI 故事。",
			Parameters: []string{
				"group_id", "creator_id", "title", "short_desc", "summary", "description", "origin", "background", "style", "subject", "scene_count", "prompt", "negative_prompt", "ref_image", "layout_style",
			},
			CreatedAt: timestamp,
			UpdatedAt: timestamp,
		},
		ToolCreateCharacterID: {
			ID:          ToolCreateCharacterID,
			Name:        "create_character",
			Description: "基于故事设定创建人物角色并生成详细画像。",
			Parameters: []string{
				"story_id", "creator_id", "name", "description", "avatar", "character_type", "prompt", "ref_images", "detail",
			},
			CreatedAt: timestamp,
			UpdatedAt: timestamp,
		},
		ToolCreateStoryboardID: {
			ID:          ToolCreateStoryboardID,
			Name:        "create_storyboard",
			Description: "根据剧情描述生成新的故事板，支持复用或新增角色。",
			Parameters: []string{
				"story_id", "creator_id", "title", "content", "background", "style", "negative_prompt", "ref_image", "layout_style", "scene_count", "prev_board_id", "role_ids", "roles",
			},
			CreatedAt: timestamp,
			UpdatedAt: timestamp,
		},
	}
}
