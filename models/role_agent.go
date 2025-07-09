package models

import "context"

type RoleAgent struct {
	IDBase
	RoleID    int64  `gorm:"column:role_id" json:"role_id,omitempty"`       // 角色ID
	AgentID   int64  `gorm:"column:agent_id" json:"agent_id,omitempty"`     // AgentID
	AgentType string `gorm:"column:agent_type" json:"agent_type,omitempty"` // Agent类型
	AgentName string `gorm:"column:agent_name" json:"agent_name,omitempty"` // Agent名称
	AgentDesc string `gorm:"column:agent_desc" json:"agent_desc,omitempty"` // Agent描述
	Version   int64  `gorm:"column:version" json:"version,omitempty"`       // 版本号（秒级时间戳）
}

// CreateRoleAgent 创建RoleAgent映射
func CreateRoleAgent(ctx context.Context, agent *RoleAgent) error {
	return DataBase().WithContext(ctx).Model(&RoleAgent{}).Create(agent).Error
}

// GetRoleAgent 根据RoleID、AgentID、Version获取RoleAgent
func GetRoleAgent(ctx context.Context, roleID, agentID, version int64) (*RoleAgent, error) {
	agent := &RoleAgent{}
	err := DataBase().WithContext(ctx).Model(agent).Where("role_id = ? AND agent_id = ? AND version = ?", roleID, agentID, version).First(agent).Error
	if err != nil {
		return nil, err
	}
	return agent, nil
}

// UpdateRoleAgent 根据RoleID、AgentID、Version更新RoleAgent（只更新非零字段）
func UpdateRoleAgent(ctx context.Context, roleID, agentID, version int64, updates map[string]interface{}) error {
	return DataBase().WithContext(ctx).Model(&RoleAgent{}).Where("role_id = ? AND agent_id = ? AND version = ?", roleID, agentID, version).Updates(updates).Error
}

// DeleteRoleAgent 根据RoleID、AgentID、Version删除RoleAgent
func DeleteRoleAgent(ctx context.Context, roleID, agentID, version int64) error {
	return DataBase().WithContext(ctx).Model(&RoleAgent{}).Where("role_id = ? AND agent_id = ? AND version = ?", roleID, agentID, version).Delete(&RoleAgent{}).Error
}

// ListRoleAgentsByRoleID 根据RoleID和Version获取所有RoleAgent
func ListRoleAgentsByRoleID(ctx context.Context, roleID int64, version int64) ([]*RoleAgent, error) {
	var agents []*RoleAgent
	err := DataBase().WithContext(ctx).Model(&RoleAgent{}).Where("role_id = ? AND version = ?", roleID, version).Find(&agents).Error
	if err != nil {
		return nil, err
	}
	return agents, nil
}
