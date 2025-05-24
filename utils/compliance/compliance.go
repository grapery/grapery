package compliance

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/grapery/grapery/utils/log"
	"go.uber.org/zap"
)

var (
	GlobalComplianceTool *ComplianceTool
)

func GetComplianceTool() *ComplianceTool {
	return GlobalComplianceTool
}

type ComplianceTool struct {
	Address string
	Secret  string
}

func init() {
	GlobalComplianceTool = &ComplianceTool{}
}

func Init(address string, secret string) *ComplianceTool {
	return &ComplianceTool{
		Address: address,
		Secret:  secret,
	}
}

func (c *ComplianceTool) TextCompliance(content string) error {
	return nil
	if c.Address == "" || c.Secret == "" {
		return fmt.Errorf("compliance tool not initialized")
	}
	// 构造请求体
	type ServiceParameters struct {
		Content string `json:"content"`
	}
	reqBody := map[string]interface{}{
		"Service":           "comment_detection",
		"ServiceParameters": ServiceParameters{Content: content},
	}
	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		log.Log().Error("marshal compliance request", zap.Error(err))
		return err
	}
	// 发送 POST 请求
	req, err := http.NewRequest("POST", c.Address, bytes.NewBuffer(bodyBytes))
	if err != nil {
		log.Log().Error("create compliance request", zap.Error(err))
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Acs-Api-Key", c.Secret)
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Log().Error("send compliance request", zap.Error(err))
		return err
	}
	defer resp.Body.Close()
	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Log().Error("read compliance response", zap.Error(err))
		return err
	}
	if resp.StatusCode != 200 {
		log.Log().Error("compliance response status", zap.Int("status", resp.StatusCode), zap.ByteString("body", respBody))
		return fmt.Errorf("compliance api error: %d", resp.StatusCode)
	}
	// 解析响应体
	var respData struct {
		Code int `json:"Code"`
		Data struct {
			Labels string `json:"Labels"`
			Reason string `json:"Reason"`
		} `json:"Data"`
		Message string `json:"Message"`
	}
	if err := json.Unmarshal(respBody, &respData); err != nil {
		log.Log().Error("unmarshal compliance response", zap.Error(err))
		return err
	}
	if respData.Code != 200 {
		return fmt.Errorf("compliance api business error: %d, %s", respData.Code, respData.Message)
	}
	if respData.Data.Labels != "" {
		// 命中风险内容
		return fmt.Errorf("content not compliant: %s, reason: %s", respData.Data.Labels, respData.Data.Reason)
	}
	return nil
}

func (*ComplianceTool) ImageCompliance(image string) error {
	return nil
}
