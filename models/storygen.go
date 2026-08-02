package models

import (
	"context"
	"encoding/json"

	"github.com/grapery/common-protoc/gen"
	"github.com/grapery/grapery/utils/log"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type StoryGen struct {
	ID       uint  `gorm:"primary_key,column:id" json:"id,omitempty"`
	IDBase         // 主键ID、创建/更新时间、软删除
	OriginID int64 `gorm:"column:origin_id" json:"origin_id,omitempty"` // 源故事ID
	BoardID  int64 `gorm:"column:board_id" json:"board_id,omitempty"`   // 故事板ID
	RoleID   int64 `gorm:"column:role_id" json:"role_id,omitempty"`     // 角色ID
	UserId   int64 `gorm:"column:user_id" json:"user_id,omitempty"`     // 用户ID

	TaskId string `gorm:"column:task_id" json:"task_id,omitempty"` // 任务ID,关联故事板的uuid
	UUID   string `gorm:"column:uuid" json:"uuid,omitempty"`       // 唯一标识，对接三方平台的uuid
	Seed   int    `gorm:"column:seed" json:"seed,omitempty"`       // 故事版在生成过程中，需要保持种子

	Params    string `gorm:"column:params" json:"params,omitempty"`         // 生成参数
	Status    int    `gorm:"column:status" json:"status,omitempty"`         // 记录状态（0:无效,1:有效）
	Content   string `gorm:"column:content" json:"content,omitempty"`       // 生成内容
	TokenNum  int    `gorm:"column:token_num" json:"token_num,omitempty"`   // token数
	ImageUrls string `gorm:"column:image_urls" json:"image_urls,omitempty"` // 生成图片URL，逗号分隔

	StartTime  int64 `gorm:"column:start_time" json:"start_time,omitempty"`   // 生成开始时间
	FinishTime int64 `gorm:"column:finish_time" json:"finish_time,omitempty"` // 生成完成时间

	LLmPlatform string             `gorm:"column:llm_platform" json:"llm_platform,omitempty"` // 使用的大模型平台
	Priority    int                `gorm:"column:priority" json:"priority,omitempty"`         // 优先级
	TaskType    gen.RenderType     `gorm:"column:gen_type" json:"gen_type,omitempty"`         // 生成类型
	GenStatus   gen.StoryGenStatus `gorm:"column:gen_status" json:"gen_status,omitempty"`     // 生成状态（0:未生成,1:生成中,2:完成,3:失败）
}

func (s StoryGen) TableName() string {
	return "story_gen"
}

func (s *StoryGen) String() string {
	data, _ := json.Marshal(s)
	return string(data)
}

func CreateStoryGen(ctx context.Context, gen *StoryGen) (int64, error) {
	log.Log().Info("[CreateStoryGen] 创建StoryGen参数", zap.Any("gen", gen))
	if err := DataBase().Model(gen).
		WithContext(ctx).
		Create(gen).Error; err != nil {
		log.Log().Error("[CreateStoryGen] 创建失败", zap.Error(err), zap.Any("gen", gen))
		return 0, err
	}
	log.Log().Info("[CreateStoryGen] 创建成功", zap.Int64("id", int64(gen.ID)))
	return int64(gen.ID), nil
}

func GetStoryGen(ctx context.Context, id int64) (*StoryGen, error) {
	gen := &StoryGen{}
	log.Log().Info("[GetStoryGen] 查询StoryGen", zap.Int64("id", id))
	err := DataBase().Model(gen).
		WithContext(ctx).
		Where("id = ?", id).
		First(gen).Error
	if err != nil {
		log.Log().Error("[GetStoryGen] 查询失败", zap.Error(err), zap.Int64("id", id))
		return nil, err
	}
	log.Log().Info("[GetStoryGen] 查询成功", zap.Int64("id", id))
	return gen, nil
}

func GetStoryGensByStory(ctx context.Context, storyID int64, status int) ([]*StoryGen, error) {
	var gens []*StoryGen
	log.Log().Info("[GetStoryGensByStory] 查询StoryGen列表", zap.Int64("storyID", storyID), zap.Int("status", status))
	err := DataBase().Model(&StoryGen{}).
		WithContext(ctx).
		Where("origin_id = ? and status = ?", storyID, status).
		Where("board_id = ?", 0).
		Order("create_at desc").
		Find(&gens).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		log.Log().Error("[GetStoryGensByStory] 查询失败", zap.Error(err), zap.Int64("storyID", storyID), zap.Int("status", status))
		return nil, err
	}
	log.Log().Info("[GetStoryGensByStory] 查询成功", zap.Int("count", len(gens)))
	return gens, nil
}

func GetStoryGensByStoryBoard(ctx context.Context, boardId int64, status int) ([]*StoryGen, error) {
	var gens []*StoryGen
	log.Log().Info("[GetStoryGensByStoryBoard] 查询StoryGen列表", zap.Int64("boardId", boardId), zap.Int("status", status))
	err := DataBase().Model(&StoryGen{}).
		WithContext(ctx).
		Where("board_id = ? and status = ?", boardId, status).
		Where("origin_id = ?", 0).
		Find(&gens).Order("create_at desc").Error
	if err != nil {
		log.Log().Error("[GetStoryGensByStoryBoard] 查询失败", zap.Error(err), zap.Int64("boardId", boardId), zap.Int("status", status))
		return nil, err
	}
	log.Log().Info("[GetStoryGensByStoryBoard] 查询成功", zap.Int("count", len(gens)))
	return gens, nil
}

func GetStoryGensByStoryAndBoard(ctx context.Context, storyID int64, boardID int64, status int) ([]*StoryGen, error) {
	var gens []*StoryGen
	log.Log().Info("[GetStoryGensByStoryAndBoard] 查询StoryGen列表", zap.Int64("storyID", storyID), zap.Int64("boardID", boardID), zap.Int("status", status))
	err := DataBase().Model(&StoryGen{}).
		WithContext(ctx).
		Where("origin_id = ? and status = ?", storyID, status).
		Where("board_id = ?", boardID).
		Order("create_at desc").
		Find(&gens).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		log.Log().Error("[GetStoryGensByStoryAndBoard] 查询失败", zap.Error(err), zap.Int64("storyID", storyID), zap.Int64("boardID", boardID), zap.Int("status", status))
		return nil, err
	}
	log.Log().Info("[GetStoryGensByStoryAndBoard] 查询成功", zap.Int("count", len(gens)))
	return gens, nil
}

func GetStoryGensByStoryAndRole(ctx context.Context, storyID int64, roleId int64) (*StoryGen, error) {
	gen := &StoryGen{}
	log.Log().Info("[GetStoryGensByStoryAndRole] 查询StoryGen", zap.Int64("storyID", storyID), zap.Int64("roleId", roleId))
	err := DataBase().Model(gen).
		WithContext(ctx).
		Where("origin_id = ?", storyID).
		Where("role_id = ?", roleId).
		Limit(1).
		Order("create_at desc").
		Find(&gen).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		log.Log().Error("[GetStoryGensByStoryAndRole] 查询失败", zap.Error(err), zap.Int64("storyID", storyID), zap.Int64("roleId", roleId))
		return nil, err
	}
	log.Log().Info("[GetStoryGensByStoryAndRole] 查询成功")
	return gen, nil
}

func DelStoryGen(ctx context.Context, id int64) error {
	log.Log().Info("[DelStoryGen] 删除StoryGen", zap.Int64("id", id))
	err := DataBase().Model(&StoryGen{}).WithContext(ctx).
		Where("id = ?", id).
		Update("status = ?", 0).Error
	if err != nil {
		log.Log().Error("[DelStoryGen] 删除失败", zap.Error(err), zap.Int64("id", id))
	}
	return err
}

func UpdateStoryGen(ctx context.Context, gen *StoryGen) error {
	log.Log().Info("[UpdateStoryGen] 更新StoryGen", zap.Int64("id", int64(gen.ID)))
	err := DataBase().Model(gen).WithContext(ctx).
		Where("id = ?", gen.ID).
		Updates(gen).Error
	if err != nil {
		log.Log().Error("[UpdateStoryGen] 更新失败", zap.Error(err), zap.Int64("id", int64(gen.ID)))
	}
	return err
}

func UpdateStoryGenMultiColumn(ctx context.Context, id int64, columns map[string]interface{}) error {
	log.Log().Info("[UpdateStoryGenMultiColumn] 更新StoryGen多列", zap.Int64("id", id), zap.Any("columns", columns))
	err := DataBase().Model(&StoryGen{}).
		WithContext(ctx).
		Where("id = ?", id).
		Updates(columns).Error
	if err != nil {
		log.Log().Error("[UpdateStoryGenMultiColumn] 更新失败", zap.Error(err), zap.Int64("id", id), zap.Any("columns", columns))
	}
	return err
}

// 分页获取StoryGen列表
func GetStoryGenList(ctx context.Context, offset, limit int) ([]*StoryGen, error) {
	var gens []*StoryGen
	log.Log().Info("[GetStoryGenList] 分页获取StoryGen列表", zap.Int("offset", offset), zap.Int("limit", limit))
	err := DataBase().Model(&StoryGen{}).
		WithContext(ctx).
		Offset(offset).
		Limit(limit).
		Order("create_at desc").
		Find(&gens).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		log.Log().Error("[GetStoryGenList] 查询失败", zap.Error(err), zap.Int("offset", offset), zap.Int("limit", limit))
		return nil, err
	}
	log.Log().Info("[GetStoryGenList] 查询成功", zap.Int("count", len(gens)))
	return gens, nil
}

// 通过TaskId获取StoryGen
func GetStoryGenByTaskId(ctx context.Context, taskId string) (*StoryGen, error) {
	gen := &StoryGen{}
	log.Log().Info("[GetStoryGenByTaskId] 查询StoryGen", zap.String("taskId", taskId))
	err := DataBase().Model(gen).
		WithContext(ctx).
		Where("task_id = ?", taskId).
		First(gen).Error
	if err != nil {
		log.Log().Error("[GetStoryGenByTaskId] 查询失败", zap.Error(err), zap.String("taskId", taskId))
		return nil, err
	}
	log.Log().Info("[GetStoryGenByTaskId] 查询成功", zap.String("taskId", taskId))
	return gen, nil
}
