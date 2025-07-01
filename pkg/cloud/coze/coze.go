package coze

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
)

var (
	APIKey     = os.Getenv("COZE_API_KEY")
	Endpoint   = "https://api.coze.cn"
	AppName    = "grapery"
	APPID      = "7521236942802206759"
	CozeClient *HuoShanCozeClient
)

func init() {
	CozeClient, _ = NewCozeClient()
}

func GetCozeClient() *HuoShanCozeClient {
	return CozeClient
}

type HuoShanCozeClient struct {
}

func NewCozeClient() (*HuoShanCozeClient, error) {
	if APIKey == "" {
		return nil, errors.New("COZE_API_KEY environment variable is not set")
	}
	client := &HuoShanCozeClient{}
	return client, nil
}

func (c *HuoShanCozeClient) GetAPIKey() string {
	return os.Getenv("COZE_API_KEY")
}

func (c *HuoShanCozeClient) RefreshToken() string {
	return ""
}

/*
curl -X POST 'https://api.coze.cn/v1/workflow/stream_run' \
-H "Authorization: Bearer " \
-H "Content-Type: application/json" \
-d '{
  "workflow_id": "7521281122689925147",
  "app_id": "7521236942802206759",
  "parameters": {}
}'
*/

type CozeInitStoryboardParams struct {
	Title       string
	Description string
	Background  string
	Roles       []CozeRoleInfo
	IsAsync     bool
}

func (c CozeInitStoryboardParams) String() string {
	data, _ := json.Marshal(c)
	return string(data)
}

type CozeWorkflowRunParams struct {
	WorkflowID string                 `json:"workflow_id"` // 工作流ID，必填
	Parameters map[string]interface{} `json:"parameters"`  // 工作流参数，可选
}

type CozeWorkflowRunResponse struct {
	Code      int    `json:"code"`       // 状态码，0为成功
	Msg       string `json:"msg"`        // 状态信息
	Data      string `json:"data"`       // 工作流执行结果
	DebugURL  string `json:"debug_url"`  // 调试页面URL
	ExecuteID string `json:"execute_id"` // 执行ID
}

func (c *HuoShanCozeClient) WorkflowRun(ctx context.Context, workflowID string, params CozeWorkflowRunParams) (string, error) {
	// 序列化为JSON
	jsonData, err := json.Marshal(params)
	if err != nil {
		return "", err
	}
	log.Printf("WorkflowRun params: %s\n", string(jsonData))
	// 创建HTTP请求
	req, err := http.NewRequestWithContext(ctx, "POST", Endpoint+"/v1/workflow/run", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", err
	}
	// 设置请求头
	req.Header.Set("Authorization", "Bearer "+APIKey)
	req.Header.Set("Content-Type", "application/json")

	// 发送请求
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	// 读取响应体
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	// 解析响应
	var apiResp CozeWorkflowRunResponse
	err = json.Unmarshal(body, &apiResp)
	if err != nil {
		return "", err
	}

	// 判断API返回状态
	if apiResp.Code != 0 {
		return "", errors.New(apiResp.Msg)
	}
	ret, err := ParseCozeOutput(apiResp.Data)
	if err != nil {
		return "", err
	}
	fmt.Println("coze return: ", ret)
	return ret["output"], nil
}

func (c *HuoShanCozeClient) InitStoryboard(ctx context.Context, params CozeInitStoryboardParams) (string, error) {
	if APPID == "" {
		return "", errors.New("workflowID and appID cannot be empty")
	}
	workflowID := "7521281122689925147"
	// 构造API参数
	apiParams := CozeWorkflowRunParams{
		WorkflowID: workflowID,
		Parameters: map[string]interface{}{
			"app_id":      APPID,
			"title":       params.Title,
			"description": params.Description,
			"background":  params.Background,
			"roles":       params.Roles,
		},
	}

	ret, err := c.WorkflowRun(ctx, workflowID, apiParams)
	if err != nil {
		return "", err
	}
	println(ret)
	return ret, nil
}

type CozeRoleInfo struct {
	RoleID          string
	RoleName        string
	RoleImage       string
	RoleDescription string
}

type CozeStoryBackgroundImageParams struct {
	OriginalPrompt string
	StoryDesc      string
	Roles          []CozeRoleInfo
}

func (c *HuoShanCozeClient) StoryBackgroundImage(ctx context.Context, params CozeStoryBackgroundImageParams) (string, error) {
	workflowID := "7521281065416146980"
	apiParams := CozeWorkflowRunParams{
		WorkflowID: workflowID,
		Parameters: map[string]interface{}{
			"app_id":        APPID,
			"origin_prompt": params.OriginalPrompt,
			"story":         params.StoryDesc,
			"roles":         params.Roles,
		},
	}
	ret, err := c.WorkflowRun(ctx, workflowID, apiParams)
	if err != nil {
		return "", err
	}
	return ret, nil
}

type CozeStoryRoleBackgroundImageParams struct {
	StoryTitle string
	StoryDesc  string
	RoleName   string
	RoleDesc   string
	RoleImage  string
	Style      string
}

func (c *HuoShanCozeClient) StoryRoleBackgroundImage(ctx context.Context, params CozeStoryRoleBackgroundImageParams) (string, error) {
	workflowID := "7521281006461419535"
	apiParams := CozeWorkflowRunParams{
		WorkflowID: workflowID,
		Parameters: map[string]interface{}{
			"app_id":      APPID,
			"story_title": params.StoryTitle,
			"story_desc":  params.StoryDesc,
			"role_name":   params.RoleName,
			"role_desc":   params.RoleDesc,
			"role_image":  params.RoleImage,
			"style":       params.Style,
		},
	}
	ret, err := c.WorkflowRun(ctx, workflowID, apiParams)
	if err != nil {
		return "", err
	}
	return ret, nil
}

type CozeStoryRoleImageParams struct {
	Description     string
	ShortTermGoal   string
	LongTermGoal    string
	Personality     string
	Background      string
	HandlingStyle   string
	CognitionRange  string
	AbilityFeatures string
	Appearance      string
	DressPreference string
	RefImage        string
	Style           string
}

func (c *HuoShanCozeClient) StoryRoleImage(ctx context.Context, params CozeStoryRoleImageParams) (string, error) {
	workflowID := "7521280952544641065"
	apiParams := CozeWorkflowRunParams{
		WorkflowID: workflowID,
		Parameters: map[string]interface{}{
			"app_id":           APPID,
			"description":      params.Description,
			"short_term_goal":  params.ShortTermGoal,
			"long_term_goal":   params.LongTermGoal,
			"personality":      params.Personality,
			"background":       params.Background,
			"handling_style":   params.HandlingStyle,
			"cognition_range":  params.CognitionRange,
			"ability_features": params.AbilityFeatures,
			"appearance":       params.Appearance,
			"dress_preference": params.DressPreference,
			"ref_image":        params.RefImage,
			"style":            params.Style,
		},
	}
	ret, err := c.WorkflowRun(ctx, workflowID, apiParams)
	if err != nil {
		return "", err
	}
	return ret, nil
}

type CozeStoryWriteParams struct {
	StoryTitle string
	StoryDesc  string
}

func (c *HuoShanCozeClient) StoryWrite(ctx context.Context, params CozeStoryWriteParams) (string, error) {
	workflowID := "7521280915441516587"
	apiParams := CozeWorkflowRunParams{
		WorkflowID: workflowID,
		Parameters: map[string]interface{}{
			"app_id":      APPID,
			"story_title": params.StoryTitle,
			"story_desc":  params.StoryDesc,
		},
	}
	ret, err := c.WorkflowRun(ctx, workflowID, apiParams)
	if err != nil {
		return "", err
	}
	return ret, nil
}

type CozeStoryboardImageParams struct {
	OriginPrompt  string
	SenceRefImage string
	Storyboard    string
	Roles         []CozeRoleInfo
}

func (c *HuoShanCozeClient) StoryboardImage(ctx context.Context, params CozeStoryboardImageParams) (string, error) {
	workflowID := "7521280840124153910"
	apiParams := CozeWorkflowRunParams{
		WorkflowID: workflowID,
		Parameters: map[string]interface{}{
			"app_id":          APPID,
			"origin_prompt":   params.OriginPrompt,
			"sence_ref_image": params.SenceRefImage,
			"storyboard":      params.Storyboard,
			"roles":           params.Roles,
		},
	}
	ret, err := c.WorkflowRun(ctx, workflowID, apiParams)
	if err != nil {
		return "", err
	}
	return ret, nil
}

type CozeStoryboardImageListParams struct {
	Storyboard string
	Roles      []CozeRoleInfo
}

func (c *HuoShanCozeClient) StoryboardImageList(ctx context.Context, params CozeStoryboardImageListParams) (string, error) {
	workflowID := "7521280840124153910"
	apiParams := CozeWorkflowRunParams{
		WorkflowID: workflowID,
		Parameters: map[string]interface{}{
			"app_id":     APPID,
			"storyboard": params.Storyboard,
			"roles":      params.Roles,
		},
	}
	ret, err := c.WorkflowRun(ctx, workflowID, apiParams)
	if err != nil {
		return "", err
	}
	return ret, nil
}

type CozeStoryboardVideoParams struct {
	Prompt   string
	RefImage string
	Style    string
}

func (c *HuoShanCozeClient) StoryboardVideo(ctx context.Context, params CozeStoryboardVideoParams) (string, error) {
	workflowID := "7521280710168150059"
	apiParams := CozeWorkflowRunParams{
		WorkflowID: workflowID,
		Parameters: map[string]interface{}{
			"app_id":    APPID,
			"prompt":    params.Prompt,
			"ref_image": params.RefImage,
			"style":     params.Style,
		},
	}
	ret, err := c.WorkflowRun(ctx, workflowID, apiParams)
	if err != nil {
		return "", err
	}
	return ret, nil
}

type CozeStoryboardWriterParams struct {
	StoryChapter    string
	StoryContent    string
	StoryCharacters string
	StoryBackground string
	ImageStyle      string
	PrevContent     string // 可选，上一章节内容
}

func (c CozeStoryboardWriterParams) String() string {
	data, _ := json.Marshal(c)
	return string(data)
}

func (c *HuoShanCozeClient) StoryboardWriter(ctx context.Context, params CozeStoryboardWriterParams) (string, error) {
	workflowID := "7521280682498015251"
	apiParams := CozeWorkflowRunParams{
		WorkflowID: workflowID,
		Parameters: map[string]interface{}{
			"app_id":           APPID,
			"story_chapter":    params.StoryChapter,
			"story_content":    params.StoryContent,
			"story_characters": params.StoryCharacters,
			"story_background": params.StoryBackground,
			"image_style":      "吉卜力风格",            // 这里可以根据需要调整
			"prev_content":     params.PrevContent, // 可选参数
		},
	}
	ret, err := c.WorkflowRun(ctx, workflowID, apiParams)
	if err != nil {
		return "", err
	}
	return ret, nil
}

type CozeStoryboardContinueParams struct {
	Title            string
	Description      string
	Background       string
	StoryName        string
	StoryPrevContent string
	Roles            []CozeRoleInfo
}

func (c CozeStoryboardContinueParams) String() string {
	data, _ := json.Marshal(c)
	return string(data)
}

func (c *HuoShanCozeClient) StoryboardContinue(ctx context.Context, params CozeStoryboardContinueParams) (string, error) {
	workflowID := "7521279737222922276"
	apiParams := CozeWorkflowRunParams{
		WorkflowID: workflowID,
		Parameters: map[string]interface{}{
			"app_id":             APPID,
			"title":              params.Title,
			"description":        params.Description,
			"background":         params.Background,
			"story_name":         params.StoryName,
			"story_prev_content": params.StoryPrevContent,
			"roles":              params.Roles,
		},
	}
	ret, err := c.WorkflowRun(ctx, workflowID, apiParams)
	if err != nil {
		return "", err
	}
	return ret, nil
}

type CozeStoryRoleDetailParams struct {
	StoryName   string
	StoryDesc   string
	RoleName    string
	Description string
	OtherRoles  string
}

func (c CozeStoryRoleDetailParams) String() string {
	data, _ := json.Marshal(c)
	return string(data)
}

func (c *HuoShanCozeClient) StoryRoleDetail(ctx context.Context, params CozeStoryRoleDetailParams) (string, error) {
	workflowID := "7521279583335219236"
	apiParams := CozeWorkflowRunParams{
		WorkflowID: workflowID,
		Parameters: map[string]interface{}{
			"app_id":      APPID,
			"story_name":  params.StoryName,
			"story_desc":  params.StoryDesc,
			"name":        params.RoleName,
			"description": params.Description,
			"other_roles": params.OtherRoles, // 可选，其他角色信息
		},
	}
	ret, err := c.WorkflowRun(ctx, workflowID, apiParams)
	if err != nil {
		return "", err
	}
	return ret, nil
}

type CozeStoryRoleDetailContinueParams struct {
	StoryName   string
	StoryDesc   string
	RoleName    string
	Description string
	OtherRoles  string
	History     string
}

func (c CozeStoryRoleDetailContinueParams) String() string {
	data, _ := json.Marshal(c)
	return string(data)
}

func (c *HuoShanCozeClient) StoryRoleDetailContinue(ctx context.Context, params CozeStoryRoleDetailContinueParams) (string, error) {
	workflowID := "7521894522772996111"
	apiParams := CozeWorkflowRunParams{
		WorkflowID: workflowID,
		Parameters: map[string]interface{}{
			"app_id":      APPID,
			"story_name":  params.StoryName,
			"story_desc":  params.StoryDesc,
			"name":        params.RoleName,
			"description": params.Description,
			"other_roles": params.OtherRoles, // 可选，其他角色信息
			"history":     params.History,    // 可选，历史信息
		},
	}
	ret, err := c.WorkflowRun(ctx, workflowID, apiParams)
	if err != nil {
		return "", err
	}
	return ret, nil
}

///////////////////////////////////////////////////

// CozeChatRequest 定义chat_v3 API的请求参数结构体
// 参考官方文档：https://www.coze.cn/open/docs/developer_guides/chat_v3#417f17b5
// POST https://api.coze.cn/open_api/v2/chat
type CozeChatRequest struct {
	ConversationID string                 `json:"conversation_id,omitempty"` // 会话ID，可选
	BotID          string                 `json:"bot_id"`                    // 智能体ID，必填
	User           string                 `json:"user"`                      // 用户ID，必填
	Query          string                 `json:"query"`                     // 用户输入，必填
	Stream         bool                   `json:"stream"`                    // 是否流式
	Extra          map[string]interface{} `json:"extra,omitempty"`           // 额外参数，可选
}

// CozeChatResponse 定义chat_v3 API的响应结构体（非流式）
type CozeChatResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    struct {
		ConversationID string `json:"conversation_id"`
		Messages       []struct {
			Role        string `json:"role"`
			Type        string `json:"type"`
			Content     string `json:"content"`
			ContentType string `json:"content_type"`
			ExtraInfo   any    `json:"extra_info"`
		} `json:"messages"`
	} `json:"data"`
}

// 聊天参数结构体，描述与角色/助手对话所需的参数
// RoleName 建议为BotID，StoryName/StoryDesc等可根据业务扩展
type CozeChatWithRoleParams struct {
	StoryName string // 场景/故事名
	StoryDesc string // 场景/故事描述或用户输入
	RoleName  string // 角色名或BotID
	RoleDesc  string // 角色描述
	RoleImage string // 角色头像
}

// ChatWithRole 非流式：与指定角色智能体对话
func (c *HuoShanCozeClient) ChatWithRole(ctx context.Context, params CozeChatWithRoleParams) (string, error) {
	if params.RoleName == "" || params.StoryName == "" || params.StoryDesc == "" {
		return "", errors.New("RoleName, StoryName, StoryDesc 不能为空")
	}
	// 构造请求体
	reqBody := CozeChatRequest{
		BotID:  params.RoleName, // 这里假设RoleName即BotID，实际应传BotID
		User:   "user1",         // 可根据实际业务传递用户ID
		Query:  params.StoryDesc,
		Stream: false,
	}
	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}
	request, err := http.NewRequestWithContext(ctx, "POST", Endpoint+"/open_api/v2/chat", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", err
	}
	request.Header.Set("Authorization", "Bearer "+APIKey)
	request.Header.Set("Content-Type", "application/json")
	client := &http.Client{}
	resp, err := client.Do(request)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	var chatResp CozeChatResponse
	err = json.Unmarshal(body, &chatResp)
	if err != nil {
		return "", err
	}
	if chatResp.Code != 0 {
		return "", errors.New(chatResp.Message)
	}
	if len(chatResp.Data.Messages) > 0 {
		return chatResp.Data.Messages[0].Content, nil
	}
	return "", nil
}

// ContinueChatWithRole 非流式：继续与角色对话
func (c *HuoShanCozeClient) ContinueChatWithRole(ctx context.Context, params CozeChatWithRoleParams) (string, error) {
	// 这里假设ConversationID由业务维护
	return c.ChatWithRole(ctx, params)
}

// ContinueChatWithAssistant 非流式：继续与助手对话
func (c *HuoShanCozeClient) ContinueChatWithAssistant(ctx context.Context, params CozeChatWithRoleParams) (string, error) {
	// 这里假设RoleName为助手BotID
	return c.ChatWithRole(ctx, params)
}

// ChatWithRoleStream 流式：与指定角色智能体对话（流式）
func (c *HuoShanCozeClient) ChatWithRoleStream(ctx context.Context, params CozeChatWithRoleParams) (string, error) {
	if params.RoleName == "" || params.StoryName == "" || params.StoryDesc == "" {
		return "", errors.New("RoleName, StoryName, StoryDesc 不能为空")
	}
	reqBody := CozeChatRequest{
		BotID:  params.RoleName, // 这里假设RoleName即BotID，实际应传BotID
		User:   "user1",
		Query:  params.StoryDesc,
		Stream: true,
	}
	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}
	request, err := http.NewRequestWithContext(ctx, "POST", Endpoint+"/open_api/v2/chat", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", err
	}
	request.Header.Set("Authorization", "Bearer "+APIKey)
	request.Header.Set("Content-Type", "application/json")
	client := &http.Client{}
	resp, err := client.Do(request)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	// 简单拼接流式内容
	var result string
	buf := make([]byte, 1024)
	for {
		n, err := resp.Body.Read(buf)
		if n > 0 {
			result += string(buf[:n])
		}
		if err != nil {
			break
		}
	}
	return result, nil
}

// ContinueChatWithRoleStream 流式：继续与角色对话
func (c *HuoShanCozeClient) ContinueChatWithRoleStream(ctx context.Context, params CozeChatWithRoleParams) (string, error) {
	return c.ChatWithRoleStream(ctx, params)
}

// ContinueChatWithAssistantStream 流式：继续与助手对话
func (c *HuoShanCozeClient) ContinueChatWithAssistantStream(ctx context.Context, params CozeChatWithRoleParams) (string, error) {
	return c.ChatWithRoleStream(ctx, params)
}

// ParseCozeOutput 解析Coze返回的output字符串为map
// 输入示例：output 字符串
// 返回：map[string]string 或 error
func ParseCozeOutput(output string) (map[string]string, error) {
	// 1. 去掉前后的 --- 分隔符和多余空白
	start := strings.Index(output, "{")
	end := strings.LastIndex(output, "}")
	if start == -1 || end == -1 || end <= start {
		return nil, errors.New("output格式不正确，未找到json对象")
	}
	jsonStr := output[start : end+1]

	// 2. 去掉多余的换行符
	jsonStr = strings.ReplaceAll(jsonStr, "\n", "")

	// 3. 反序列化
	var result map[string]string
	err := json.Unmarshal([]byte(jsonStr), &result)
	if err != nil {
		return nil, err
	}
	return result, nil
}
