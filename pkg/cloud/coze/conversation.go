package coze

/*
curl -X POST 'https://api.coze.cn/v1/conversation/create' \
-H "Authorization: Bearer " \
-H "Content-Type: application/json"
*/

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"

	graperylog "github.com/grapery/grapery/utils/log"
	"go.uber.org/zap"
)

// ConversationData结构体，兼容Coze官方返回
type ConversationData struct {
	ID            string `json:"id"`
	CreatedAt     int64  `json:"created_at"`
	LastSectionID string `json:"last_section_id"`
}

// ResponseDetail结构体，兼容Coze官方返回
type ResponseDetail struct {
	LogID string `json:"logid,omitempty"`
	// 可根据Coze文档补充更多字段
}

// CozeConversationCreateResponse结构体，兼容Coze官方返回
type CozeConversationCreateResponse struct {
	Code   int64            `json:"code"`
	Msg    string           `json:"msg"`
	Data   ConversationData `json:"data"`
	Detail ResponseDetail   `json:"detail"`
}

// 会话初始消息结构体（可根据官方EnterMessage对象进一步细化）
type EnterMessage struct {
	Role        string            `json:"role,omitempty"`         // 发送方角色
	Type        string            `json:"type,omitempty"`         // 消息类型
	Content     string            `json:"content,omitempty"`      // 消息内容
	ContentType string            `json:"content_type,omitempty"` // 内容类型，如"text"
	MetaData    map[string]string `json:"meta_data,omitempty"`    // 附加元数据，可选
	// 可根据官方文档补充更多字段，如 type, file_id, object_string 等
}

// 会话创建参数结构体，兼容Coze官方文档
type CozeConversationCreateParams struct {
	AppID       string            `json:"app_id,omitempty"`       // 应用ID，必填
	BotID       string            `json:"bot_id,omitempty"`       // 智能体ID，可选
	MetaData    map[string]string `json:"meta_data,omitempty"`    // 附加元数据，可选
	Messages    []EnterMessage    `json:"messages,omitempty"`     // 初始消息，可选
	ConnectorID string            `json:"connector_id,omitempty"` // 渠道ID，可选
}

func (c *HuoShanCozeClient) ConversationCreate(ctx context.Context, params CozeConversationCreateParams) (string, error) {
	// 自动补全AppID
	//params.AppID = APPID
	params.ConnectorID = "1024"
	graperylog.Log().Info("[ConversationCreate] 入口参数", zap.Any("params", params))

	// 参数序列化
	jsonData, err := json.Marshal(params)
	if err != nil {
		graperylog.Log().Error("[ConversationCreate] 参数序列化失败", zap.Error(err))
		return "", err
	}

	// 构造HTTP请求
	request, err := http.NewRequestWithContext(ctx, "POST", Endpoint+"/v1/conversation/create", bytes.NewBuffer(jsonData))
	if err != nil {
		graperylog.Log().Error("[ConversationCreate] 创建HTTP请求失败", zap.Error(err))
		return "", err
	}
	request.Header.Set("Authorization", "Bearer "+APIKey)
	request.Header.Set("Content-Type", "application/json")

	// 发送请求
	client := &http.Client{}
	resp, err := client.Do(request)
	if err != nil {
		graperylog.Log().Error("[ConversationCreate] HTTP请求发送失败", zap.Error(err))
		return "", err
	}
	defer resp.Body.Close()

	// 读取响应体
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		graperylog.Log().Error("[ConversationCreate] 读取响应体失败", zap.Error(err))
		return "", err
	}

	// 解析响应
	var respObj CozeConversationCreateResponse
	err = json.Unmarshal(body, &respObj)
	if err != nil {
		graperylog.Log().Error("[ConversationCreate] 响应体解析失败", zap.Error(err))
		return "", err
	}

	// 判断返回状态
	if respObj.Code != 0 {
		graperylog.Log().Error("[ConversationCreate] 创建会话失败", zap.String("msg", respObj.Msg))
		return "", errors.New(respObj.Msg)
	}

	graperylog.Log().Info("[ConversationCreate] 创建会话成功", zap.String("conversation_id", respObj.Data.ID))
	return respObj.Data.ID, nil
}

/*
curl -X POST 'https://api.coze.cn/v1/conversations/14234324/clear' \
-H "Authorization: Bearer pat_NaxFbsL7ZBDHWGKacShgXmowqhcrAeNZU9fp6YPmu4VrGpjyxPaJ6wiaLr9QhQ2i" \
-H "Content-Type: application/json"
*/

// 清空会话返回data结构体，兼容Coze官方文档
type ConversationClearData struct {
	ConversationID string `json:"conversation_id"` // 会话唯一标识
	ID             string `json:"id"`              // section唯一标识
}

// 清空会话返回结构体，兼容Coze官方文档
type CozeConversationClearResponse struct {
	Code   int64                 `json:"code"`
	Msg    string                `json:"msg"`
	Data   ConversationClearData `json:"data"`
	Detail ResponseDetail        `json:"detail"`
}

func (c *HuoShanCozeClient) ConversationClear(ctx context.Context, conversationId string) error {
	request, err := http.NewRequestWithContext(ctx, "POST", Endpoint+"/v1/conversations/"+conversationId+"/clear", nil)
	if err != nil {
		graperylog.Log().Error("[ConversationClear] 创建HTTP请求失败", zap.Error(err))
		return err
	}
	request.Header.Set("Authorization", "Bearer "+APIKey)
	request.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(request)
	if err != nil {
		graperylog.Log().Error("[ConversationClear] HTTP请求发送失败", zap.Error(err))
		return err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		graperylog.Log().Error("[ConversationClear] 读取响应体失败", zap.Error(err))
		return err
	}
	// 可选：解析响应体，判断code/msg等
	var respObj CozeConversationClearResponse
	err = json.Unmarshal(body, &respObj)
	if err != nil {
		graperylog.Log().Error("[ConversationClear] 响应体解析失败", zap.Error(err))
		return err
	}
	if respObj.Code != 0 {
		graperylog.Log().Error("[ConversationClear] 清空会话失败", zap.String("msg", respObj.Msg))
		return errors.New(respObj.Msg)
	}
	graperylog.Log().Info("[ConversationClear] 清空会话成功", zap.String("conversation_id", respObj.Data.ConversationID), zap.String("section_id", respObj.Data.ID))
	return nil
}
