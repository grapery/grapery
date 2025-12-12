package mysql

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/grapestree/fgrapery/grapery/internal/domain"
	"gorm.io/gorm"
)

// ==================== Agent Operations ====================

// GetAgentByCharacterID retrieves an agent by character ID
func (r *Repository) GetAgentByCharacterID(ctx context.Context, characterID string) (*domain.Agent, error) {
	var agent Agent
	if err := r.db.WithContext(ctx).
		Preload("Character").
		Preload("Character.Author").
		Where("character_id = ?", characterID).
		First(&agent).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return ModelToAgent(&agent), nil
}

// GetAgentByID retrieves an agent by ID
func (r *Repository) GetAgentByID(ctx context.Context, id string) (*domain.Agent, error) {
	var agent Agent
	if err := r.db.WithContext(ctx).
		Preload("Character").
		Preload("Character.Author").
		First(&agent, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return ModelToAgent(&agent), nil
}

// ListAgents retrieves agents with pagination
func (r *Repository) ListAgents(ctx context.Context, limit, offset int) ([]*domain.Agent, error) {
	var agents []Agent
	query := r.db.WithContext(ctx).
		Preload("Character").
		Preload("Character.Author").
		Order("created_at DESC")

	if limit > 0 {
		query = query.Limit(limit).Offset(offset)
	}

	if err := query.Find(&agents).Error; err != nil {
		return nil, err
	}

	result := make([]*domain.Agent, len(agents))
	for i, a := range agents {
		result[i] = ModelToAgent(&a)
	}
	return result, nil
}

// CreateAgent creates a new agent for a character
func (r *Repository) CreateAgent(ctx context.Context, agent *domain.Agent) error {
	// Check if character already has an agent
	var count int64
	if err := r.db.WithContext(ctx).
		Model(&Agent{}).
		Where("character_id = ?", agent.CharacterID).
		Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return errors.New("character already has an agent")
	}

	// Set default values
	if agent.ID == "" {
		agent.ID = uuid.New().String()
	}
	if agent.Status == "" {
		agent.Status = domain.AgentStatusActive
	}
	if agent.Temperature == 0 {
		agent.Temperature = 0.7
	}
	if agent.MaxTokens == 0 {
		agent.MaxTokens = 2048
	}

	dbAgent := AgentToModel(agent)
	return r.db.WithContext(ctx).Create(dbAgent).Error
}

// UpdateAgent updates an existing agent
func (r *Repository) UpdateAgent(ctx context.Context, agent *domain.Agent) error {
	dbAgent := AgentToModel(agent)
	return r.db.WithContext(ctx).Model(&Agent{}).Where("id = ?", agent.ID).Updates(dbAgent).Error
}

// DeleteAgent deletes an agent
func (r *Repository) DeleteAgent(ctx context.Context, id string) error {
	result := r.db.WithContext(ctx).Delete(&Agent{}, "id = ?", id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return domain.ErrNotFound
	}
	return nil
}

// IncrementAgentInteractionCount increments the interaction count for an agent
func (r *Repository) IncrementAgentInteractionCount(ctx context.Context, agentID string) error {
	return r.db.WithContext(ctx).
		Model(&Agent{}).
		Where("id = ?", agentID).
		UpdateColumn("interaction_count", gorm.Expr("interaction_count + ?", 1)).
		Error
}

// ==================== AgentSkill Operations ====================

// ListAgentSkills retrieves all skills for an agent with filters
func (r *Repository) ListAgentSkills(ctx context.Context, agentID string, filter *domain.SkillFilter) ([]*domain.AgentSkill, error) {
	query := r.db.WithContext(ctx).Where("agent_id = ?", agentID)

	// Apply filters if provided
	if filter != nil {
		if filter.Type != "" {
			query = query.Where("type = ?", filter.Type)
		}
		if filter.Status != "" {
			query = query.Where("status = ?", filter.Status)
		}
		if filter.Enabled != nil {
			query = query.Where("enabled = ?", *filter.Enabled)
		}

		// Apply sorting
		switch filter.SortBy {
		case "usage_count":
			query = query.Order("usage_count " + filter.Order)
		case "priority":
			query = query.Order("priority " + filter.Order)
		case "success_rate":
			query = query.Order("(success_count * 100.0 / NULLIF(usage_count, 0)) " + filter.Order)
		case "created_at":
			query = query.Order("created_at " + filter.Order)
		default:
			query = query.Order("priority DESC, created_at DESC")
		}

		// Apply pagination
		if filter.Limit > 0 {
			query = query.Limit(filter.Limit).Offset(filter.Offset)
		}
	} else {
		// Default ordering if no filter provided
		query = query.Order("priority DESC, created_at DESC")
	}

	var skills []AgentSkill
	if err := query.Find(&skills).Error; err != nil {
		return nil, err
	}
	return ModelsToAgentSkills(skills), nil
}

// GetAgentSkillByID retrieves a skill by ID
func (r *Repository) GetAgentSkillByID(ctx context.Context, id string) (*domain.AgentSkill, error) {
	var skill AgentSkill
	if err := r.db.WithContext(ctx).First(&skill, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return ModelToAgentSkill(&skill), nil
}

// GetAgentSkillByName retrieves a skill by agent ID and name
func (r *Repository) GetAgentSkillByName(ctx context.Context, agentID, name string) (*domain.AgentSkill, error) {
	var skill AgentSkill
	if err := r.db.WithContext(ctx).
		Where("agent_id = ? AND name = ?", agentID, name).
		First(&skill).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return ModelToAgentSkill(&skill), nil
}

// CreateAgentSkill creates a new skill for an agent
func (r *Repository) CreateAgentSkill(ctx context.Context, skill *domain.AgentSkill) error {
	// Check if skill name already exists for this agent
	var count int64
	if err := r.db.WithContext(ctx).
		Model(&AgentSkill{}).
		Where("agent_id = ? AND name = ?", skill.AgentID, skill.Name).
		Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return errors.New("skill with this name already exists for this agent")
	}

	// Set defaults
	if skill.ID == "" {
		skill.ID = uuid.New().String()
	}
	if skill.Status == "" {
		skill.Status = domain.SkillStatusActive
	}

	// Create skill
	dbSkill := AgentSkillToModel(skill)
	if err := r.db.WithContext(ctx).Create(dbSkill).Error; err != nil {
		return err
	}

	// Update agent's skill count
	return r.db.WithContext(ctx).
		Model(&Agent{}).
		Where("id = ?", skill.AgentID).
		UpdateColumn("skill_count", gorm.Expr("skill_count + ?", 1)).
		Error
}

// UpdateAgentSkill updates an existing skill
func (r *Repository) UpdateAgentSkill(ctx context.Context, skill *domain.AgentSkill) error {
	dbSkill := AgentSkillToModel(skill)
	return r.db.WithContext(ctx).Model(&AgentSkill{}).Where("id = ?", skill.ID).Updates(dbSkill).Error
}

// DeleteAgentSkill deletes a skill
func (r *Repository) DeleteAgentSkill(ctx context.Context, id string) error {
	// Get skill to find agent ID
	var skill AgentSkill
	if err := r.db.WithContext(ctx).First(&skill, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return domain.ErrNotFound
		}
		return err
	}

	// Delete skill
	result := r.db.WithContext(ctx).Delete(&AgentSkill{}, "id = ?", id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return domain.ErrNotFound
	}

	// Update agent's skill count
	return r.db.WithContext(ctx).
		Model(&Agent{}).
		Where("id = ?", skill.AgentID).
		UpdateColumn("skill_count", gorm.Expr("GREATEST(skill_count - ?, 0)", 1)).
		Error
}

// IncrementSkillUsage updates skill statistics
func (r *Repository) IncrementSkillUsage(ctx context.Context, skillID string, success bool, executionTime int) error {
	updates := map[string]interface{}{
		"usage_count": gorm.Expr("usage_count + ?", 1),
	}

	if success {
		updates["success_count"] = gorm.Expr("success_count + ?", 1)
	} else {
		updates["failure_count"] = gorm.Expr("failure_count + ?", 1)
	}

	// Update average execution time
	// New average = (old_avg * usage_count + new_time) / (usage_count + 1)
	updates["avg_execution_time"] = gorm.Expr(
		"(avg_execution_time * usage_count + ?) / (usage_count + ?)",
		executionTime, 1,
	)

	return r.db.WithContext(ctx).
		Model(&AgentSkill{}).
		Where("id = ?", skillID).
		Updates(updates).
		Error
}

// ==================== AgentSkillUsage Operations ====================

// CreateAgentSkillUsage records a skill usage
func (r *Repository) CreateAgentSkillUsage(ctx context.Context, usage *domain.AgentSkillUsage) error {
	if usage.ID == "" {
		usage.ID = uuid.New().String()
	}
	dbUsage := AgentSkillUsageToModel(usage)
	return r.db.WithContext(ctx).Create(dbUsage).Error
}

// ListAgentSkillUsages retrieves skill usages with filters
func (r *Repository) ListAgentSkillUsages(ctx context.Context, filter domain.SkillUsageFilter) ([]*domain.AgentSkillUsage, error) {
	query := r.db.WithContext(ctx).Model(&AgentSkillUsage{})

	if filter.AgentID != "" {
		query = query.Where("agent_id = ?", filter.AgentID)
	}
	if filter.SkillID != "" {
		query = query.Where("skill_id = ?", filter.SkillID)
	}
	if filter.UserID != "" {
		query = query.Where("user_id = ?", filter.UserID)
	}
	if filter.Success != nil {
		query = query.Where("success = ?", *filter.Success)
	}
	if filter.FromTime > 0 {
		query = query.Where("created_at >= ?", filter.FromTime)
	}
	if filter.ToTime > 0 {
		query = query.Where("created_at <= ?", filter.ToTime)
	}

	query = query.Order("created_at DESC")

	if filter.Limit > 0 {
		query = query.Limit(filter.Limit).Offset(filter.Offset)
	}

	var usages []AgentSkillUsage
	if err := query.Find(&usages).Error; err != nil {
		return nil, err
	}
	return ModelsToAgentSkillUsages(usages), nil
}

// GetSkillUsageStats retrieves statistics for a skill
func (r *Repository) GetSkillUsageStats(ctx context.Context, skillID string, fromTime, toTime int64) (*domain.SkillUsageStats, error) {
	var stats struct {
		TotalUsages      int64
		SuccessCount     int64
		FailureCount     int64
		AvgExecutionTime float64
		TotalTokens      int64
	}

	query := r.db.WithContext(ctx).Model(&AgentSkillUsage{}).Where("skill_id = ?", skillID)
	if fromTime > 0 {
		query = query.Where("created_at >= ?", fromTime)
	}
	if toTime > 0 {
		query = query.Where("created_at <= ?", toTime)
	}

	if err := query.Select(
		"COUNT(*) as total_usages",
		"SUM(CASE WHEN success THEN 1 ELSE 0 END) as success_count",
		"SUM(CASE WHEN NOT success THEN 1 ELSE 0 END) as failure_count",
		"AVG(execution_time) as avg_execution_time",
		"SUM(tokens_used) as total_tokens",
	).Scan(&stats).Error; err != nil {
		return nil, err
	}

	successRate := 0.0
	if stats.TotalUsages > 0 {
		successRate = float64(stats.SuccessCount) / float64(stats.TotalUsages) * 100
	}

	return &domain.SkillUsageStats{
		TotalUsages:      stats.TotalUsages,
		SuccessCount:     stats.SuccessCount,
		FailureCount:     stats.FailureCount,
		SuccessRate:      successRate,
		AvgExecutionTime: int(stats.AvgExecutionTime),
		TotalTokens:      stats.TotalTokens,
	}, nil
}

// ==================== AgentInteraction Operations ====================

// CreateAgentInteraction records an agent interaction
func (r *Repository) CreateAgentInteraction(ctx context.Context, interaction *domain.AgentInteraction) error {
	if interaction.ID == "" {
		interaction.ID = uuid.New().String()
	}
	dbInteraction := AgentInteractionToModel(interaction)
	return r.db.WithContext(ctx).Create(dbInteraction).Error
}

// GetAgentInteraction retrieves an interaction by ID
func (r *Repository) GetAgentInteraction(ctx context.Context, id string) (*domain.AgentInteraction, error) {
	var interaction AgentInteraction
	if err := r.db.WithContext(ctx).
		Preload("Agent").
		Preload("User").
		Preload("Character").
		First(&interaction, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return ModelToAgentInteraction(&interaction), nil
}

// ListAgentInteractions retrieves interactions with filters
func (r *Repository) ListAgentInteractions(ctx context.Context, filter *domain.InteractionFilter) ([]*domain.AgentInteraction, error) {
	query := r.db.WithContext(ctx).Model(&AgentInteraction{})

	if filter.AgentID != "" {
		query = query.Where("agent_id = ?", filter.AgentID)
	}
	if filter.UserID != "" {
		query = query.Where("user_id = ?", filter.UserID)
	}
	if filter.CharacterID != "" {
		query = query.Where("character_id = ?", filter.CharacterID)
	}
	if filter.Type != "" {
		query = query.Where("type = ?", filter.Type)
	}

	query = query.Order("created_at DESC")

	if filter.Limit > 0 {
		query = query.Limit(filter.Limit).Offset(filter.Offset)
	}

	var interactions []AgentInteraction
	if err := query.Find(&interactions).Error; err != nil {
		return nil, err
	}
	return ModelsToAgentInteractions(interactions), nil
}

// GetInteractionStats retrieves statistics for an agent
func (r *Repository) GetInteractionStats(ctx context.Context, agentID string) (map[string]interface{}, error) {
	var stats struct {
		TotalInteractions int64
		SuccessCount      int64
		FailureCount      int64
		AvgDuration       float64
		TotalTokens       int64
	}

	if err := r.db.WithContext(ctx).Model(&AgentInteraction{}).
		Where("agent_id = ?", agentID).
		Select(
			"COUNT(*) as total_interactions",
			"SUM(CASE WHEN success THEN 1 ELSE 0 END) as success_count",
			"SUM(CASE WHEN NOT success THEN 1 ELSE 0 END) as failure_count",
			"AVG(duration) as avg_duration",
			"SUM(tokens_used) as total_tokens",
		).
		Scan(&stats).Error; err != nil {
		return nil, err
	}

	successRate := 0.0
	if stats.TotalInteractions > 0 {
		successRate = float64(stats.SuccessCount) / float64(stats.TotalInteractions) * 100
	}

	return map[string]interface{}{
		"totalInteractions": stats.TotalInteractions,
		"successCount":      stats.SuccessCount,
		"failureCount":      stats.FailureCount,
		"successRate":       successRate,
		"avgDuration":       int(stats.AvgDuration),
		"totalTokens":       stats.TotalTokens,
	}, nil
}

// ==================== AgentMemory Operations ====================

// CreateAgentMemory creates a new memory
func (r *Repository) CreateAgentMemory(ctx context.Context, memory *domain.AgentMemory) error {
	if memory.ID == "" {
		memory.ID = uuid.New().String()
	}
	dbMemory := AgentMemoryToModel(memory)
	return r.db.WithContext(ctx).Create(dbMemory).Error
}

// GetAgentMemory retrieves a memory by ID
func (r *Repository) GetAgentMemory(ctx context.Context, id string) (*domain.AgentMemory, error) {
	var memory AgentMemory
	if err := r.db.WithContext(ctx).First(&memory, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return ModelToAgentMemory(&memory), nil
}

// ListAgentMemories retrieves memories with filters
func (r *Repository) ListAgentMemories(ctx context.Context, filter *domain.MemoryFilter) ([]*domain.AgentMemory, error) {
	query := r.db.WithContext(ctx).Model(&AgentMemory{})

	if filter.AgentID != "" {
		query = query.Where("agent_id = ?", filter.AgentID)
	}
	if filter.UserID != "" {
		query = query.Where("user_id = ?", filter.UserID)
	}
	if filter.MemoryType != "" {
		query = query.Where("memory_type = ?", filter.MemoryType)
	}
	if filter.Key != "" {
		query = query.Where("key = ?", filter.Key)
	}

	// Order by importance and access frequency
	query = query.Order("importance DESC, access_count DESC, created_at DESC")

	if filter.Limit > 0 {
		query = query.Limit(filter.Limit).Offset(filter.Offset)
	}

	var memories []AgentMemory
	if err := query.Find(&memories).Error; err != nil {
		return nil, err
	}
	return ModelsToAgentMemories(memories), nil
}

// UpdateAgentMemory updates a memory
func (r *Repository) UpdateAgentMemory(ctx context.Context, memory *domain.AgentMemory) error {
	dbMemory := AgentMemoryToModel(memory)
	return r.db.WithContext(ctx).Model(&AgentMemory{}).Where("id = ?", memory.ID).Updates(dbMemory).Error
}

// DeleteAgentMemory deletes a memory
func (r *Repository) DeleteAgentMemory(ctx context.Context, id string) error {
	result := r.db.WithContext(ctx).Delete(&AgentMemory{}, "id = ?", id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return domain.ErrNotFound
	}
	return nil
}

// IncrementMemoryAccess increments the access count and updates last accessed time
func (r *Repository) IncrementMemoryAccess(ctx context.Context, memoryID string) error {
	now := r.db.WithContext(ctx).NowFunc()
	return r.db.WithContext(ctx).
		Model(&AgentMemory{}).
		Where("id = ?", memoryID).
		Updates(map[string]interface{}{
			"access_count":  gorm.Expr("access_count + ?", 1),
			"last_accessed": now,
		}).
		Error
}

// CleanExpiredMemories deletes expired memories
func (r *Repository) CleanExpiredMemories(ctx context.Context) error {
	now := r.db.WithContext(ctx).NowFunc()
	return r.db.WithContext(ctx).
		Where("expires_at IS NOT NULL AND expires_at < ?", now).
		Delete(&AgentMemory{}).
		Error
}
