package mcps

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// registerTools wires Grapery story/character tools onto the official MCP server.
func (s *McpService) registerTools(server *mcp.Server) {
	type toolDef struct {
		name        string
		description string
		schema      map[string]any
		handler     func(context.Context, map[string]interface{}) ([]byte, error)
	}

	defs := []toolDef{
		{
			name:        "create_story",
			description: "调用 Grapery 引擎创建一个新的 AI 故事。",
			schema: objectSchema(map[string]any{
				"group_id":        numberProp("目标小组或社区 ID"),
				"creator_id":      numberProp("发起操作的用户 ID"),
				"user_id":         numberProp("用户 ID（creator_id 的备选）"),
				"title":           stringProp("故事标题"),
				"short_desc":      stringProp("一句话概述"),
				"summary":         stringProp("概述（short_desc 备选）"),
				"description":     stringProp("详细剧情描述"),
				"origin":          stringProp("灵感来源或背景"),
				"background":      stringProp("作品设定背景"),
				"style":           stringProp("画面/叙事风格"),
				"subject":         stringProp("主题"),
				"scene_count":     numberProp("场景数量"),
				"prompt":          stringProp("正向提示词"),
				"negative_prompt": stringProp("需要避免的元素"),
				"ref_image":       stringProp("参考图 URL"),
				"layout_style":    stringProp("分镜布局风格"),
			}, []string{"group_id", "title"}),
			handler: s.handleCreateStory,
		},
		{
			name:        "get_story",
			description: "按 ID 获取故事详情。",
			schema: objectSchema(map[string]any{
				"id":       stringProp("故事 ID"),
				"story_id": stringProp("故事 ID（id 备选）"),
			}, nil),
			handler: s.handleGetStory,
		},
		{
			name:        "create_character",
			description: "基于故事设定创建人物角色并生成详细画像。",
			schema: objectSchema(map[string]any{
				"story_id":       numberProp("所属故事 ID"),
				"creator_id":     numberProp("发起操作的用户 ID"),
				"user_id":        numberProp("用户 ID（creator_id 备选）"),
				"name":           stringProp("角色姓名"),
				"description":    stringProp("角色核心设定"),
				"avatar":         stringProp("头像 URL 或外貌描述"),
				"character_type": stringProp("角色类型，如主角/反派/配角"),
				"prompt":         stringProp("角色生成提示词"),
				"ref_images":     arrayProp("参考图 URL 列表"),
				"detail":         objectProp("角色细节对象"),
			}, []string{"story_id", "name", "description"}),
			handler: s.handleCreateCharacter,
		},
		{
			name:        "get_character",
			description: "按 ID 获取角色详情。",
			schema: objectSchema(map[string]any{
				"id":           stringProp("角色 ID"),
				"character_id": stringProp("角色 ID（id 备选）"),
			}, nil),
			handler: s.handleGetCharacter,
		},
		{
			name:        "create_storyboard",
			description: "根据剧情描述生成新的故事板，支持复用或新增角色。",
			schema: objectSchema(map[string]any{
				"story_id":        numberProp("所属故事 ID"),
				"creator_id":      numberProp("发起操作的用户 ID"),
				"user_id":         numberProp("用户 ID（creator_id 备选）"),
				"title":           stringProp("分镜标题"),
				"content":         stringProp("情节描述"),
				"background":      stringProp("场景背景"),
				"style":           stringProp("视觉风格"),
				"negative_prompt": stringProp("需要避免的画面元素"),
				"ref_image":       stringProp("参考图 URL"),
				"layout_style":    stringProp("分镜排版"),
				"scene_count":     numberProp("场景数量"),
				"prev_board_id":   numberProp("上一段分镜 ID"),
				"role_ids":        arrayProp("已有角色 ID 列表"),
				"roles":           arrayProp("新建角色定义列表"),
			}, []string{"story_id", "title"}),
			handler: s.handleCreateStoryboard,
		},
		{
			name:        "follow_character",
			description: "关注一个角色。",
			schema: objectSchema(map[string]any{
				"character_id": stringProp("角色 ID"),
				"user_id":      numberProp("用户 ID"),
			}, []string{"character_id", "user_id"}),
			handler: s.handleFollowCharacter,
		},
		{
			name:        "unfollow_character",
			description: "取消关注一个角色。",
			schema: objectSchema(map[string]any{
				"character_id": stringProp("角色 ID"),
				"user_id":      numberProp("用户 ID"),
			}, []string{"character_id", "user_id"}),
			handler: s.handleUnfollowCharacter,
		},
		{
			name:        "like_story",
			description: "点赞故事。",
			schema: objectSchema(map[string]any{
				"story_id": stringProp("故事 ID"),
				"user_id":  numberProp("用户 ID"),
			}, []string{"story_id"}),
			handler: s.handleLikeStory,
		},
		{
			name:        "unlike_story",
			description: "取消点赞故事。",
			schema: objectSchema(map[string]any{
				"story_id": stringProp("故事 ID"),
				"user_id":  numberProp("用户 ID"),
			}, []string{"story_id"}),
			handler: s.handleUnlikeStory,
		},
	}

	for _, def := range defs {
		def := def
		server.AddTool(&mcp.Tool{
			Name:        def.name,
			Description: def.description,
			InputSchema: def.schema,
		}, s.wrapToolHandler(def.handler))
	}
}

func (s *McpService) wrapToolHandler(
	fn func(context.Context, map[string]interface{}) ([]byte, error),
) mcp.ToolHandler {
	return func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args, err := decodeToolArgs(req)
		if err != nil {
			return toolErrorResult(err), nil
		}
		ctx = s.injectContextFromRequest(ctx, req, args)

		response, err := fn(ctx, args)
		if err != nil {
			return toolErrorResult(err), nil
		}
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: string(response)}},
		}, nil
	}
}

func decodeToolArgs(req *mcp.CallToolRequest) (map[string]interface{}, error) {
	args := map[string]interface{}{}
	if req == nil || req.Params == nil || len(req.Params.Arguments) == 0 {
		return args, nil
	}
	if err := json.Unmarshal(req.Params.Arguments, &args); err != nil {
		return nil, fmt.Errorf("invalid tool arguments: %w", err)
	}
	return args, nil
}

func (s *McpService) injectContextFromRequest(
	ctx context.Context,
	req *mcp.CallToolRequest,
	args map[string]interface{},
) context.Context {
	if req != nil && req.Extra != nil && req.Extra.Header != nil {
		if trace := req.Extra.Header.Get("X-Trace-Id"); trace != "" {
			args["trace_id"] = trace
		}
		if userID := req.Extra.Header.Get("X-User-Id"); userID != "" {
			if _, ok := args["user_id"]; !ok {
				args["user_id"] = userID
			}
			if _, ok := args["creator_id"]; !ok {
				args["creator_id"] = userID
			}
		}
	}
	return s.injectContextValues(ctx, args)
}

func toolErrorResult(err error) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		IsError: true,
		Content: []mcp.Content{&mcp.TextContent{Text: err.Error()}},
	}
}

func objectSchema(properties map[string]any, required []string) map[string]any {
	schema := map[string]any{
		"type":       "object",
		"properties": properties,
	}
	if len(required) > 0 {
		schema["required"] = required
	}
	return schema
}

func stringProp(description string) map[string]any {
	return map[string]any{"type": "string", "description": description}
}

func numberProp(description string) map[string]any {
	return map[string]any{"type": "number", "description": description}
}

func arrayProp(description string) map[string]any {
	return map[string]any{"type": "array", "description": description, "items": map[string]any{}}
}

func objectProp(description string) map[string]any {
	return map[string]any{"type": "object", "description": description}
}
