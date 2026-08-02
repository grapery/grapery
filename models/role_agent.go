package models

import (
	"context"

	"gorm.io/gorm"
)

type RoleAgent struct {
	ID uint `gorm:"primary_key,column:id" json:"id,omitempty"`
	IDBase
	RoleID      int64  `gorm:"column:role_id" json:"role_id,omitempty"`                // 角色ID
	RoleName    string `gorm:"column:role_name" json:"role_name,omitempty"`            // 角色名称
	RoleAvatar  string `gorm:"column:role_avatar" json:"role_avatar,omitempty"`        // 角色头像
	AgentID     int64  `gorm:"column:agent_id" json:"agent_id,omitempty"`              // AgentID
	AgentType   string `gorm:"column:agent_type" json:"agent_type,omitempty"`          // Agent类型
	AgentName   string `gorm:"column:agent_name" json:"agent_name,omitempty"`          // Agent名称
	AgentDetail string `gorm:"column:agent_detail,text" json:"agent_detail,omitempty"` // Agent详情
	AgentDesc   string `gorm:"column:agent_desc" json:"agent_desc,omitempty"`          // Agent描述
	Version     string `gorm:"column:version" json:"version,omitempty"`                // 版本号
	BotID       string `gorm:"column:bot_id" json:"bot_id,omitempty"`                  // BotID
	IsPublished bool   `gorm:"column:is_published" json:"is_published,omitempty"`      // 是否发布
	IsDeleted   bool   `gorm:"column:is_deleted" json:"is_deleted,omitempty"`          // 是否删除
	IsForbidden bool   `gorm:"column:is_forbidden" json:"is_forbidden,omitempty"`      // 是否禁用
	BotConfig   string `gorm:"column:bot_config,text" json:"bot_config,omitempty"`     // Bot配置
	BotDetail   string `gorm:"column:bot_detail,text" json:"bot_detail,omitempty"`     // Bot详情
	BotDesc     string `gorm:"column:bot_desc,text" json:"bot_desc,omitempty"`         // Bot描述
}

func (r RoleAgent) TableName() string {
	return "role_agents"
}

// CreateRoleAgent 创建RoleAgent映射
func CreateRoleAgent(ctx context.Context, agent *RoleAgent) error {
	return DataBase().WithContext(ctx).Model(&RoleAgent{}).Create(agent).Error
}

func GetRoleAgentByRoleID(ctx context.Context, roleID int64) (*RoleAgent, error) {
	agent := &RoleAgent{}
	err := DataBase().WithContext(ctx).
		Model(agent).
		Where("role_id = ?", roleID).
		Where("is_deleted = ?", false).
		Where("is_forbidden = ?", false).
		First(agent).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return agent, nil
}

func GetRoleAgentByAgentID(ctx context.Context, agentID int64) (*RoleAgent, error) {
	agent := &RoleAgent{}
	err := DataBase().WithContext(ctx).Model(agent).Where("agent_id = ?", agentID).First(agent).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return agent, nil
}

// GetRoleAgent 根据RoleID、AgentID、Version获取RoleAgent
func GetRoleAgent(ctx context.Context, roleID, agentID, version int64) (*RoleAgent, error) {
	agent := &RoleAgent{}
	err := DataBase().WithContext(ctx).Model(agent).Where("role_id = ? AND agent_id = ? AND version = ?", roleID, agentID, version).First(agent).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return agent, nil
}

// UpdateRoleAgent 根据RoleID、AgentID、Version更新RoleAgent（只更新非零字段）
func UpdateRoleAgent(ctx context.Context, roleID int64, updates map[string]interface{}) error {
	return DataBase().WithContext(ctx).Model(&RoleAgent{}).
		Where("role_id = ?", roleID).
		Where("is_deleted = ?", false).
		Where("is_forbidden = ?", false).
		Updates(updates).Error
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
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return agents, nil
}
