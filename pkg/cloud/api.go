package cloud

import "context"

type AgentType string

const (
	AgentTypeAzure   AgentType = "azure"
	AgentTypeLocal   AgentType = "local"
	AgentTypeTencent AgentType = "tencent"
	AgentTypeOpenAI  AgentType = "openai"
	AgentTypeGroq    AgentType = "groq"
	AgentTypeZhipu   AgentType = "zhipu"
)

// Agent is a cloud agent

type Agent interface {
	GetName() string
	GetType() AgentType
}

type AgentManage struct {
	Agents map[string]Agent
	Ctx    context.Context
}

func (am *AgentManage) AddAgent(agent Agent) {

}

func (am *AgentManage) GetAgent(name string) Agent {
	return am.Agents[name]
}

func NewAgentManage() *AgentManage {
	return &AgentManage{
		Agents: make(map[string]Agent),
	}
}
