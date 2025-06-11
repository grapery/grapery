package models

import (
	"context"
	"encoding/json"

	"gorm.io/gorm"
)

type StoryGenResult struct {
}

func (s StoryGenResult) String() string {
	data, _ := json.Marshal(s)
	return string(data)
}

type StoryGenStatus int

const (
	StoryGenStatusInit StoryGenStatus = iota
	StoryGenStatusRunning
	StoryGenStatusFinish
	StoryGenStatusError
)

// just for gen record
// StoryGen 记录一次生成任务的详细信息
// status: 0-无效 1-有效
type StoryGen struct {
	IDBase                        // 主键ID、创建/更新时间、软删除
	OriginID       int64          `gorm:"column:origin_id" json:"origin_id,omitempty"`             // 源故事ID
	BoardID        int64          `gorm:"column:board_id" json:"board_id,omitempty"`               // 故事板ID
	RoleID         int64          `gorm:"column:role_id" json:"role_id,omitempty"`                 // 角色ID
	Uuid           string         `gorm:"column:uuid" json:"uuid,omitempty"`                       // 唯一标识
	Step           int            `gorm:"column:step" json:"step,omitempty"`                       // 当前生成步数
	LLmPlatform    string         `gorm:"column:llm_platform" json:"llm_platform,omitempty"`       // 使用的大模型平台
	NegativePrompt string         `gorm:"column:negative_prompt" json:"negative_prompt,omitempty"` // 否定提示词
	PositivePrompt string         `gorm:"column:positive_prompt" json:"positive_prompt,omitempty"` // 正向提示词
	UserId         int64          `gorm:"column:user_id" json:"user_id,omitempty"`                 // 用户ID
	Priority       int            `gorm:"column:priority" json:"priority,omitempty"`               // 优先级
	GenStatus      StoryGenStatus `gorm:"column:gen_status" json:"gen_status,omitempty"`           // 生成状态（0:未生成,1:生成中,2:完成,3:失败）
	Regen          int32          `gorm:"column:regen" json:"regen,omitempty"`                     // 是否重绘（0:否,1:是）
	Params         string         `gorm:"column:params" json:"params,omitempty"`                   // 生成参数
	Status         int            `gorm:"column:status" json:"status,omitempty"`                   // 记录状态（0:无效,1:有效）
	Content        string         `gorm:"column:content" json:"content,omitempty"`                 // 生成内容
	TokenNum       int            `gorm:"column:token_num" json:"token_num,omitempty"`             // token数
	ImageUrls      string         `gorm:"column:image_urls" json:"image_urls,omitempty"`           // 生成图片URL，逗号分隔
	StartTime      int64          `gorm:"column:start_time" json:"start_time,omitempty"`           // 生成开始时间
	FinishTime     int64          `gorm:"column:finish_time" json:"finish_time,omitempty"`         // 生成完成时间
	GenType        int            `gorm:"column:gen_type" json:"gen_type,omitempty"`               // 生成类型
	TaskType       int            `gorm:"column:task_type" json:"task_type,omitempty"`             // 任务类型（1:故事,2:故事板,3:角色）
	TaskId         string         `gorm:"column:task_id" json:"task_id,omitempty"`                 // 任务ID
}

func (s StoryGen) TableName() string {
	return "story_gen"
}

func (s *StoryGen) String() string {
	data, _ := json.Marshal(s)
	return string(data)
}

func CreateStoryGen(ctx context.Context, gen *StoryGen) (int64, error) {
	if err := DataBase().Model(gen).
		WithContext(ctx).
		Create(gen).Error; err != nil {
		return 0, err
	}
	return int64(gen.IDBase.ID), nil
}

func GetStoryGen(ctx context.Context, id int64) (*StoryGen, error) {
	gen := &StoryGen{}
	err := DataBase().Model(gen).
		WithContext(ctx).
		Where("id = ?", id).
		First(gen).Error
	if err != nil {
		return nil, err
	}
	return gen, nil
}

func GetStoryGensByStory(ctx context.Context, storyID int64, status int) ([]*StoryGen, error) {
	var gens []*StoryGen
	err := DataBase().Model(&StoryGen{}).
		WithContext(ctx).
		Where("origin_id = ? and status = ?", storyID, status).
		Where("board_id = ?", 0).
		Order("create_at desc").
		Find(&gens).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return nil, err
	}
	return gens, nil
}

func GetStoryGensByStoryBoard(ctx context.Context, boardId int64, status int) ([]*StoryGen, error) {
	var gens []*StoryGen
	err := DataBase().Model(&StoryGen{}).
		WithContext(ctx).
		Where("board_id = ? and status = ?", boardId, status).
		Where("origin_id = ?", 0).
		Find(&gens).Order("create_at desc").Error
	if err != nil {
		return nil, err
	}
	return gens, nil
}

func GetStoryGensByStoryAndBoard(ctx context.Context, storyID int64, boardID int64, status int) ([]*StoryGen, error) {
	var gens []*StoryGen
	err := DataBase().Model(&StoryGen{}).
		WithContext(ctx).
		Where("origin_id = ? and status = ?", storyID, status).
		Where("board_id = ?", boardID).
		Order("create_at desc").
		Find(&gens).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return nil, err
	}
	return gens, nil
}

func GetStoryGensByStoryAndRole(ctx context.Context, storyID int64, roleId int64) (*StoryGen, error) {
	gen := &StoryGen{}
	err := DataBase().Model(gen).
		WithContext(ctx).
		Where("origin_id = ?", storyID).
		Where("role_id = ?", roleId).
		Limit(1).
		Order("create_at desc").
		Find(&gen).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return nil, err
	}
	return gen, nil
}

func DelStoryGen(ctx context.Context, id int64) error {
	err := DataBase().Model(&StoryGen{}).WithContext(ctx).
		Where("id = ?", id).
		Update("status = ?", 0).Error
	return err
}

func UpdateStoryGen(ctx context.Context, gen *StoryGen) error {
	return DataBase().Model(gen).WithContext(ctx).
		Where("id = ?", gen.IDBase.ID).
		Updates(gen).Error
}

func UpdateStoryGenMultiColumn(ctx context.Context, id int64, columns map[string]interface{}) error {
	return DataBase().Model(&StoryGen{}).
		WithContext(ctx).
		Where("id = ?", id).
		Updates(columns).Error
}

// 分页获取StoryGen列表
func GetStoryGenList(ctx context.Context, offset, limit int) ([]*StoryGen, error) {
	var gens []*StoryGen
	err := DataBase().Model(&StoryGen{}).
		WithContext(ctx).
		Offset(offset).
		Limit(limit).
		Order("create_at desc").
		Find(&gens).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return nil, err
	}
	return gens, nil
}

// 通过TaskId获取StoryGen
func GetStoryGenByTaskId(ctx context.Context, taskId string) (*StoryGen, error) {
	gen := &StoryGen{}
	err := DataBase().Model(gen).
		WithContext(ctx).
		Where("task_id = ?", taskId).
		First(gen).Error
	if err != nil {
		return nil, err
	}
	return gen, nil
}
