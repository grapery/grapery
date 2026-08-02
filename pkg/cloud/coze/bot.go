package coze

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	cozego "github.com/coze-dev/coze-go"
	"github.com/grapery/grapery/utils/log" // 引入 zap 日志封装
	"go.uber.org/zap"
)

// AgentBot 定义，参考 Coze Bot 对象文档 https://www.coze.cn/open/docs/developer_guides/bot_object
// 代表一个 Coze 智能体（Bot）对象，包含所有核心属性。每个字段均严格对齐官方文档，便于序列化和 API 对接。
type AgentBot struct {
	BotID          string          `json:"bot_id"`           // 智能体唯一标识，系统自动生成，区分不同 Bot
	Name           string          `json:"name"`             // 智能体名称，必填，长度 1-40 字符
	Description    string          `json:"description"`      // 智能体描述信息，选填，长度 0-200 字符
	IconURL        string          `json:"icon_url"`         // 智能体头像地址，选填，图片 URL
	CreateTime     int64           `json:"create_time"`      // 创建时间，10 位 Unix 时间戳（秒），系统自动生成
	UpdateTime     int64           `json:"update_time"`      // 更新时间，10 位 Unix 时间戳（秒），系统自动生成
	Version        string          `json:"version"`          // 智能体最新版本号，系统自动生成
	PromptInfo     *PromptInfo     `json:"prompt_info"`      // 智能体提示词配置，包含系统提示词等
	OnboardingInfo *OnboardingInfo `json:"onboarding_info"`  // 智能体开场白配置，包含欢迎语和推荐问题
	BotMode        int             `json:"bot_mode"`         // 智能体模式，0: 单 Agent，1: 多 Agent，默认为 0
	PluginInfoList []*PluginInfo   `json:"plugin_info_list"` // 插件信息列表，Bot 可用的插件集合
	ModelInfo      *ModelInfo      `json:"model_info"`       // 智能体使用的模型信息
	Knowledge      *Knowledge      `json:"knowledge"`        // 智能体绑定的知识库信息
}

func (bot *AgentBot) GetBotID() string {
	return bot.BotID
}

func (bot *AgentBot) String() string {
	botJson, err := json.Marshal(bot)
	if err != nil {
		return ""
	}
	return string(botJson)
}

// PromptInfo 定义智能体的提示词配置。
// prompt: 智能体的系统提示词，影响 Bot 的行为风格。
type PromptInfo struct {
	Prompt string `json:"prompt"` // 智能体提示词，支持多行文本，建议 0-2000 字符
}

// OnboardingInfo 定义智能体的开场白配置。
// prologue: 开场白内容，用户首次对话时展示。
// suggested_questions: 推荐问题列表，辅助用户快速上手。
type OnboardingInfo struct {
	Prologue           string   `json:"prologue"`            // 开场白内容，支持多行文本，建议 0-200 字符
	SuggestedQuestions []string `json:"suggested_questions"` // 推荐问题列表，最多 10 条
}

// PluginInfo 定义插件信息。
// 包含插件基本信息及其可用工具列表。
type PluginInfo struct {
	PluginID    string       `json:"plugin_id"`     // 插件唯一标识，系统分配
	Name        string       `json:"name"`          // 插件名称
	Description string       `json:"description"`   // 插件描述
	IconURL     string       `json:"icon_url"`      // 插件头像 URL
	APIInfoList []*PluginAPI `json:"api_info_list"` // 插件工具列表，每个插件可包含多个 API 工具
}

// PluginAPI 定义插件工具信息。
// 包含工具的唯一标识、名称和描述。
type PluginAPI struct {
	APIID       string `json:"api_id"`      // 工具唯一标识，系统分配
	Name        string `json:"name"`        // 工具名称
	Description string `json:"description"` // 工具描述
}

// ModelInfo 定义智能体使用的模型信息。
// 包含模型唯一标识和名称。
type ModelInfo struct {
	ModelID   string `json:"model_id"`   // 模型唯一标识，系统分配
	ModelName string `json:"model_name"` // 模型名称，如 gpt-3.5、qwen-turbo 等
}

// Knowledge 定义智能体绑定的知识库信息。
// knowledge_infos: 绑定的知识库列表。
type Knowledge struct {
	KnowledgeInfos []*KnowledgeInfo `json:"knowledge_infos"` // 绑定的知识库列表，支持多个知识库
}

// KnowledgeInfo 定义单个知识库的信息。
// id: 知识库唯一标识。
// name: 知识库名称。
type KnowledgeInfo struct {
	ID   string `json:"id"`   // 知识库唯一标识，系统分配
	Name string `json:"name"` // 知识库名称
}

// 创建智能体
// CreateAgentBot 创建一个新的 Coze 智能体（Bot）。
//
// 参数：
//
//	token           - API 鉴权 Token，必填
//	spaceID         - 工作空间 ID，必填
//	name            - 智能体名称，必填
//	description     - 智能体描述，选填
//	iconFileID      - 智能体头像文件 ID，选填
//	promptInfo      - 智能体提示词配置，选填
//	onboardingInfo  - 智能体开场白配置，选填
//	pluginIdList    - 插件 ID 列表，支持数组/map，选填
//	workflowIdList  - 工作流 ID 列表，支持数组/map，选填
//	modelInfoConfig - 模型配置信息，支持 map/结构体，选填
//
// 返回值：
//
//	string - 创建成功返回 BotID
//	error  - 失败时返回详细错误信息
//
// 该方法会自动组装请求体，调用 Coze 官方 API 创建智能体，并处理所有异常和分支，日志全量覆盖。
func (c *HuoShanCozeClient) CreateAgentBot(
	ctx context.Context,
	name string,
	description string,
	iconFileID string,
	promptInfo *PromptInfo,
	onboardingInfo *OnboardingInfo,
	pluginIdList interface{},
	workflowIdList interface{},
	modelInfoConfig interface{},
) (string, error) {
	const method = "CreateAgentBot"
	url := "https://api.coze.cn/v1/bot/create"
	// 构造请求体
	body := map[string]interface{}{
		"space_id": SPACEID,
		"name":     name,
	}
	if description != "" {
		body["description"] = description
	}
	if iconFileID != "" {
		body["icon_file_id"] = iconFileID
	}
	if promptInfo != nil {
		body["prompt_info"] = promptInfo
	}
	if onboardingInfo != nil {
		body["onboarding_info"] = onboardingInfo
	}
	if pluginIdList != nil {
		body["plugin_id_list"] = pluginIdList
	}
	if workflowIdList != nil {
		body["workflow_id_list"] = workflowIdList
	}
	if modelInfoConfig != nil {
		body["model_info_config"] = modelInfoConfig
	}
	log.Log().Info(method+": 构造请求体", zap.Any("body", body))
	jsonBody, err := json.Marshal(body)
	if err != nil {
		log.Log().Error(method+": 请求体序列化失败", zap.Any("body", body), zap.Error(err))
		return "", fmt.Errorf("marshal body failed: %w", err)
	}
	// 构造请求
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		log.Log().Error(method+": 创建 HTTP 请求失败", zap.String("url", url), zap.Error(err))
		return "", fmt.Errorf("new request failed: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+APIKey)
	req.Header.Set("Content-Type", "application/json")
	log.Log().Info(method+": 发送 HTTP 请求", zap.String("url", url))
	// 发送请求
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Log().Error(method+": HTTP 请求失败", zap.String("url", url), zap.Error(err))
		return "", fmt.Errorf("http request failed: %w", err)
	}
	defer resp.Body.Close()
	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Log().Error(method+": 读取响应体失败", zap.Error(err))
		return "", fmt.Errorf("read response failed: %w", err)
	}
	log.Log().Info(method+": 收到响应", zap.String("respBody", string(respBody)))
	// 解析返回
	var result struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
		Data struct {
			BotID string `json:"bot_id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		log.Log().Error(method+": 响应体反序列化失败", zap.String("respBody", string(respBody)), zap.Error(err))
		return "", fmt.Errorf("unmarshal response failed: %w, body: %s", err, string(respBody))
	}
	if result.Code != 0 {
		log.Log().Warn(method+": 创建 Bot 失败", zap.Int("code", result.Code), zap.String("msg", result.Msg))
		return "", fmt.Errorf("create bot failed: %s", result.Msg)
	}
	log.Log().Info(method+": 创建 Bot 成功", zap.String("bot_id", result.Data.BotID))
	return result.Data.BotID, nil
}

func (c *HuoShanCozeClient) CreateAgentBotV2(
	ctx context.Context,
	name string,
	description string,
	iconFileID string,
	promptInfo *PromptInfo,
	onboardingInfo *OnboardingInfo,
	pluginIdList interface{},
	workflowIdList interface{},
	modelInfoConfig interface{},
) (string, error) {
	authCli := cozego.NewTokenAuth(SERVICE_TOKEN)
	cozeCli := cozego.NewCozeAPI(authCli, cozego.WithBaseURL("https://api.coze.cn"))
	req := cozego.CreateBotsReq{
		SpaceID:         SPACEID,
		Name:            name,
		Description:     description,
		IconFileID:      iconFileID,
		PromptInfo:      &cozego.BotPromptInfo{},
		OnboardingInfo:  &cozego.BotOnboardingInfo{},
		WorkflowIDList:  &cozego.WorkflowIDList{},
		ModelInfoConfig: &cozego.BotModelInfoConfig{},
	}
	if promptInfo != nil && promptInfo.Prompt != "" {
		log.Log().Info("创建 Bot 设置提示词", zap.String("prompt", promptInfo.Prompt))
		req.PromptInfo.Prompt = promptInfo.Prompt
	}
	if onboardingInfo != nil && onboardingInfo.Prologue != "" {
		log.Log().Info("创建 Bot 设置开场白", zap.String("prologue", onboardingInfo.Prologue))
		req.OnboardingInfo.Prologue = onboardingInfo.Prologue
		req.OnboardingInfo.SuggestedQuestions = onboardingInfo.SuggestedQuestions
	}
	reqData, _ := json.Marshal(req)
	log.Log().Info("创建 Bot 请求体", zap.String("reqData", string(reqData)))
	resp, err := cozeCli.Bots.Create(ctx, &req)
	if err != nil {
		log.Log().Error("创建 Bot 失败", zap.Error(err))
		return "", fmt.Errorf("create bot failed: %w", err)
	}
	log.Log().Info("创建 Bot 成功", zap.String("bot_id", resp.BotID))
	return resp.BotID, nil
}

// UpdateAgentBot 更新已存在的 Coze 智能体（Bot）配置。
//
// 该方法支持通过 API 更新通过扣子平台或 API 创建的所有智能体，支持修改名称、描述、头像、人设、开场白、知识库、插件、工作流、模型等。
//
// 参数：
//
//	token           - 个人访问令牌（PAT），需 edit 权限
//	botID           - 待更新的智能体 ID，必填
//	name            - 智能体名称，选填
//	description     - 智能体描述，选填
//	iconFileID      - 智能体头像文件 ID，选填
//	promptInfo      - 智能体提示词配置，选填
//	onboardingInfo  - 智能体开场白配置，选填
//	knowledge       - 智能体知识库配置，选填
//	pluginIdList    - 插件 ID 列表配置，选填
//	workflowIdList  - 工作流 ID 列表配置，选填
//	modelInfoConfig - 模型配置信息，选填
//
// 返回值：
//
//	error - 更新成功返回 nil，失败返回详细错误信息
//
// 日志全量覆盖，便于排查。
func (c *HuoShanCozeClient) UpdateAgentBot(
	ctx context.Context,
	botID string,
	description string,
	iconFileID string,
	promptInfo *PromptInfo,
	onboardingInfo *OnboardingInfo,
	knowledge interface{},
	pluginIdList interface{},
	workflowIdList interface{},
	modelInfoConfig interface{},
) error {
	const method = "UpdateAgentBot"
	url := "https://api.coze.cn/v1/bot/update"
	// 构造请求体
	body := map[string]interface{}{
		"space_id": SPACEID,
		"bot_id":   botID,
	}
	if description != "" {
		body["description"] = description
	}
	if iconFileID != "" {
		body["icon_file_id"] = iconFileID
	}
	if promptInfo != nil {
		body["prompt_info"] = promptInfo
	}
	if onboardingInfo != nil {
		body["onboarding_info"] = onboardingInfo
	}
	if knowledge != nil {
		body["knowledge"] = knowledge
	}
	if pluginIdList != nil {
		body["plugin_id_list"] = pluginIdList
	}
	if workflowIdList != nil {
		body["workflow_id_list"] = workflowIdList
	}
	if modelInfoConfig != nil {
		body["model_info_config"] = modelInfoConfig
	}
	log.Log().Info(method+": 构造请求体", zap.Any("body", body))
	jsonBody, err := json.Marshal(body)
	if err != nil {
		log.Log().Error(method+": 请求体序列化失败", zap.Any("body", body), zap.Error(err))
		return fmt.Errorf("marshal body failed: %w", err)
	}
	// 构造请求
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		log.Log().Error(method+": 创建 HTTP 请求失败", zap.String("url", url), zap.Error(err))
		return fmt.Errorf("new request failed: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+APIKey)
	req.Header.Set("Content-Type", "application/json")
	log.Log().Info(method+": 发送 HTTP 请求", zap.String("url", url))
	// 发送请求
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Log().Error(method+": HTTP 请求失败", zap.String("url", url), zap.Error(err))
		return fmt.Errorf("http request failed: %w", err)
	}
	defer resp.Body.Close()
	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Log().Error(method+": 读取响应体失败", zap.Error(err))
		return fmt.Errorf("read response failed: %w", err)
	}
	log.Log().Info(method+": 收到响应", zap.String("respBody", string(respBody)))
	// 解析返回
	var result struct {
		Code   int    `json:"code"`
		Msg    string `json:"msg"`
		Detail struct {
			LogID string `json:"logid"`
		} `json:"detail"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		log.Log().Error(method+": 响应体反序列化失败", zap.String("respBody", string(respBody)), zap.Error(err))
		return fmt.Errorf("unmarshal response failed: %w, body: %s", err, string(respBody))
	}
	if result.Code != 0 {
		log.Log().Warn(method+": 更新 Bot 失败", zap.Int("code", result.Code), zap.String("msg", result.Msg), zap.String("logid", result.Detail.LogID))
		return fmt.Errorf("update bot failed: %s, logid: %s", result.Msg, result.Detail.LogID)
	}
	log.Log().Info(method+": 更新 Bot 成功", zap.String("bot_id", botID), zap.String("logid", result.Detail.LogID))
	return nil
}

func (c *HuoShanCozeClient) UpdateAgentBotV2(
	ctx context.Context,
	botID string,
	description string,
	iconFileID string,
	promptInfo *PromptInfo,
	onboardingInfo *OnboardingInfo,
	knowledge interface{},
	pluginIdList interface{},
	workflowIdList interface{},
	modelInfoConfig interface{},
) error {
	authCli := cozego.NewTokenAuth(c.GetAPIKey())
	cozeCli := cozego.NewCozeAPI(authCli, cozego.WithBaseURL("https://api.coze.cn"))
	req := cozego.UpdateBotsReq{
		BotID:       botID,
		Description: description,
		IconFileID:  iconFileID,
		PromptInfo: &cozego.BotPromptInfo{
			Prompt: promptInfo.Prompt,
		},
		OnboardingInfo:  &cozego.BotOnboardingInfo{},
		WorkflowIDList:  &cozego.WorkflowIDList{},
		ModelInfoConfig: &cozego.BotModelInfoConfig{},
	}
	_, err := cozeCli.Bots.Update(ctx, &req)
	if err != nil {
		log.Log().Error("更新 Bot 失败", zap.Error(err))
		return fmt.Errorf("update bot failed: %w", err)
	}
	log.Log().Info("更新 Bot 成功", zap.String("bot_id", botID))
	return nil
}

// PublishAgentBot 发布指定智能体到 API、Chat SDK 或自定义渠道。
//
// 参数：
//
//	token         - 访问令牌（需 publish 权限）
//	botID         - 要发布的智能体 ID，必填
//	connectorIDs  - 发布渠道 ID 列表，必填（API: 1024，ChatSDK: 999，自定义渠道为自定义 ID）
//
// 返回值：
//
//	botID   - 发布后智能体 ID
//	version - 发布后智能体版本号
//	error   - 失败时返回详细错误信息
//
// 日志全量覆盖，便于排查。
func (c *HuoShanCozeClient) PublishAgentBot(ctx context.Context, botID string) (string, string, error) {
	const method = "PublishAgentBot"
	url := "https://api.coze.cn/v1/bot/publish"
	// 构造请求体
	body := map[string]interface{}{
		"space_id":      SPACEID,
		"bot_id":        botID,
		"connector_ids": 1024,
	}
	log.Log().Info(method+": 构造请求体", zap.Any("body", body))
	jsonBody, err := json.Marshal(body)
	if err != nil {
		log.Log().Error(method+": 请求体序列化失败", zap.Any("body", body), zap.Error(err))
		return "", "", fmt.Errorf("marshal body failed: %w", err)
	}
	// 构造请求
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		log.Log().Error(method+": 创建 HTTP 请求失败", zap.String("url", url), zap.Error(err))
		return "", "", fmt.Errorf("new request failed: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+APIKey)
	req.Header.Set("Content-Type", "application/json")
	log.Log().Info(method+": 发送 HTTP 请求", zap.String("url", url))
	// 发送请求
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Log().Error(method+": HTTP 请求失败", zap.String("url", url), zap.Error(err))
		return "", "", fmt.Errorf("http request failed: %w", err)
	}
	defer resp.Body.Close()
	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Log().Error(method+": 读取响应体失败", zap.Error(err))
		return "", "", fmt.Errorf("read response failed: %w", err)
	}
	log.Log().Info(method+": 收到响应", zap.String("respBody", string(respBody)))
	// 解析返回
	var result struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
		Data struct {
			BotID   string `json:"bot_id"`
			Version string `json:"version"`
		} `json:"data"`
		Detail struct {
			LogID string `json:"logid"`
		} `json:"detail"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		log.Log().Error(method+": 响应体反序列化失败", zap.String("respBody", string(respBody)), zap.Error(err))
		return "", "", fmt.Errorf("unmarshal response failed: %w, body: %s", err, string(respBody))
	}
	if result.Code != 0 {
		log.Log().Warn(method+": 发布 Bot 失败", zap.Int("code", result.Code), zap.String("msg", result.Msg), zap.String("logid", result.Detail.LogID))
		return "", "", fmt.Errorf("publish bot failed: %s, logid: %s", result.Msg, result.Detail.LogID)
	}
	log.Log().Info(method+": 发布 Bot 成功", zap.String("bot_id", result.Data.BotID), zap.String("version", result.Data.Version), zap.String("logid", result.Detail.LogID))
	return result.Data.BotID, result.Data.Version, nil
}

func (c *HuoShanCozeClient) PublishAgentBotV2(ctx context.Context, botID string) (string, string, error) {
	authCli := cozego.NewTokenAuth(c.GetAPIKey())
	cozeCli := cozego.NewCozeAPI(authCli, cozego.WithBaseURL("https://api.coze.cn"))
	req := cozego.PublishBotsReq{
		BotID:        botID,
		ConnectorIDs: []string{"1024"},
	}
	resp, err := cozeCli.Bots.Publish(ctx, &req)
	if err != nil {
		log.Log().Error("publish Bot 失败", zap.Error(err))
		return "", "", fmt.Errorf("publish bot failed: %w", err)
	}
	log.Log().Info("publish Bot 成功", zap.String("bot_id", botID))
	return resp.BotID, resp.BotVersion, nil
}

// GetAgentBotConfig 获取指定智能体的配置信息（支持草稿/已发布版本）。
//
// 参数：
//
//	token        - 访问令牌（需 getMetadata 权限）
//	botID        - 智能体 ID，必填
//	isPublished  - 是否获取已发布版本（true:已发布，false:草稿），可选，默认 true
//
// 返回值：
//
//	data  - 智能体配置信息（原始 JSON map）
//	error - 失败时返回详细错误信息
//
// 日志全量覆盖，便于排查。
func (c *HuoShanCozeClient) GetAgentBotConfig(token string, botID string, isPublished *bool) (map[string]interface{}, error) {
	const method = "GetAgentBotConfig"
	baseURL := "https://api.coze.cn/v1/bots/" + botID
	// 构造 URL
	url := baseURL
	if isPublished != nil {
		url += "?is_published="
		if *isPublished {
			url += "true"
			log.Log().Info(method+": 查询已发布版本", zap.String("bot_id", botID))
		} else {
			url += "false"
			log.Log().Info(method+": 查询草稿版本", zap.String("bot_id", botID))
		}
	} else {
		log.Log().Info(method+": 未指定 isPublished，默认查询已发布版本", zap.String("bot_id", botID))
	}
	log.Log().Info(method+": 构造请求 URL", zap.String("url", url))
	// 构造请求
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Log().Error(method+": 创建 HTTP 请求失败", zap.String("url", url), zap.Error(err), zap.String("bot_id", botID))
		return nil, fmt.Errorf("new request failed: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	log.Log().Info(method+": 发送 HTTP 请求", zap.String("url", url), zap.String("bot_id", botID))
	// 发送请求
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Log().Error(method+": HTTP 请求失败", zap.String("url", url), zap.Error(err), zap.String("bot_id", botID))
		return nil, fmt.Errorf("http request failed: %w", err)
	}
	defer func() {
		log.Log().Info(method+": 关闭响应体", zap.String("bot_id", botID))
		resp.Body.Close()
	}()
	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Log().Error(method+": 读取响应体失败", zap.Error(err), zap.String("bot_id", botID))
		return nil, fmt.Errorf("read response failed: %w", err)
	}
	log.Log().Info(method+": 收到响应", zap.String("respBody", string(respBody)), zap.String("bot_id", botID))
	// 解析返回
	var result struct {
		Code   int                    `json:"code"`
		Msg    string                 `json:"msg"`
		Data   map[string]interface{} `json:"data"`
		Detail struct {
			LogID string `json:"logid"`
		} `json:"detail"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		log.Log().Error(method+": 响应体反序列化失败", zap.String("respBody", string(respBody)), zap.Error(err), zap.String("bot_id", botID))
		return nil, fmt.Errorf("unmarshal response failed: %w, body: %s", err, string(respBody))
	}
	if result.Code != 0 {
		log.Log().Warn(method+": 获取 Bot 配置失败", zap.Int("code", result.Code), zap.String("msg", result.Msg), zap.String("logid", result.Detail.LogID), zap.String("bot_id", botID))
		return nil, fmt.Errorf("get bot config failed: %s, logid: %s", result.Msg, result.Detail.LogID)
	}
	if result.Data == nil {
		log.Log().Warn(method+": 返回数据为空", zap.String("bot_id", botID), zap.String("logid", result.Detail.LogID))
		return nil, fmt.Errorf("get bot config success but data is nil, logid: %s", result.Detail.LogID)
	}
	log.Log().Info(method+": 获取 Bot 配置成功", zap.String("bot_id", botID), zap.String("logid", result.Detail.LogID))
	return result.Data, nil
}

// 删除智能体
func (c HuoShanCozeClient) DeleteAgentBot(ctx context.Context, botID string) error {
	const method = "DeleteAgentBot"
	url := "https://api.coze.cn/v1/bots/" + botID + "/unpublish"
	body := map[string]interface{}{
		"connector_id":     1024,
		"unpublish_reason": "normal_update",
	}
	log.Log().Info(method+": 构造请求体", zap.Any("body", body))
	jsonBody, err := json.Marshal(body)
	if err != nil {
		log.Log().Error(method+": 请求体序列化失败", zap.Any("body", body), zap.Error(err))
		return fmt.Errorf("marshal body failed: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		log.Log().Error(method+": 创建 HTTP 请求失败", zap.String("url", url), zap.Error(err))
		return fmt.Errorf("new request failed: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+APIKey)
	req.Header.Set("Content-Type", "application/json")
	log.Log().Info(method+": 发送 HTTP 请求", zap.String("url", url))
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Log().Error(method+": HTTP 请求失败", zap.String("url", url), zap.Error(err))
		return fmt.Errorf("http request failed: %w", err)
	}
	defer resp.Body.Close()
	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Log().Error(method+": 读取响应体失败", zap.Error(err))
		return fmt.Errorf("read response failed: %w", err)
	}
	log.Log().Info(method+": 收到响应", zap.String("respBody", string(respBody)))
	// 解析返回
	var result struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		log.Log().Error(method+": 响应体反序列化失败", zap.String("respBody", string(respBody)), zap.Error(err))
		return fmt.Errorf("unmarshal response failed: %w, body: %s", err, string(respBody))
	}
	if result.Code != 0 {
		log.Log().Warn(method+": 删除 Bot 失败", zap.Int("code", result.Code), zap.String("msg", result.Msg))
		return fmt.Errorf("delete bot failed: %s", result.Msg)
	}
	log.Log().Info(method+": 删除 Bot 成功", zap.String("bot_id", botID))
	return nil
}
