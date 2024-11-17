package models

import "context"

type StoryRole struct {
	IDBase
	StoryID              int64    `json:"story_id"`
	CharacterName        string   `json:"character_name"`
	CharacterAvatar      string   `json:"character_avatar"`
	CharacterID          string   `json:"character_id"`
	CharacterType        string   `json:"character_type"`
	CharacterPrompt      string   `json:"character_prompt"`
	CharacterRefImages   []string `json:"character_ref_images"`
	CharacterDescription string   `json:"character_description"`
	CreatorID            int64    `json:"creator_id"`
	Status               int      `json:"status"`
}

func (s *StoryRole) TableName() string {
	return "story_role"
}

func GetStoryRoleByCreatorId(ctx context.Context, creatorId int64) ([]*StoryRole, error) {
	var roles []*StoryRole
	if err := DataBase().Model(&StoryRole{}).
		Where("creator_id = ?", creatorId).
		Find(&roles).Error; err != nil {
		return nil, err
	}
	return roles, nil
}

func CreateStoryRole(ctx context.Context, role *StoryRole) (int64, error) {
	if err := DataBase().Model(role).
		WithContext(ctx).
		Create(role).Error; err != nil {
		return 0, err
	}
	return int64(role.ID), nil
}

func GetStoryRole(ctx context.Context, storyID int64) ([]*StoryRole, error) {
	var roles []*StoryRole
	if err := DataBase().Model(&StoryRole{}).
		Where("story_id = ?", storyID).
		Find(&roles).Error; err != nil {
		return nil, err
	}
	return roles, nil
}

func GetStoryRoleByID(ctx context.Context, roleID int64) (*StoryRole, error) {
	var role StoryRole
	if err := DataBase().Model(&StoryRole{}).
		Where("id = ?", roleID).
		First(&role).Error; err != nil {
		return nil, err
	}
	return &role, nil
}

func UpdateStoryRole(ctx context.Context, roleID int64, needUpdateFields map[string]interface{}) error {
	if err := DataBase().Model(&StoryRole{}).
		Where("id = ?", roleID).
		WithContext(ctx).
		Updates(needUpdateFields).Error; err != nil {
		return err
	}
	return nil
}

func GetStoryRoleByName(ctx context.Context, name string, storyId int64) (*StoryRole, error) {
	var role StoryRole
	if err := DataBase().Model(&StoryRole{}).
		Where("character_name = ?", name).
		Where("story_id = ?", storyId).
		First(&role).Error; err != nil {
		return nil, err
	}
	return &role, nil
}
