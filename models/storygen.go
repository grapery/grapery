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
type StoryGen struct {
	IDBase
	OriginID       int64
	BoardID        int64
	RoleID         int64
	Uuid           string
	Step           int
	LLmPlatform    string
	NegativePrompt string
	PositivePrompt string
	//
	UserId   int64
	Priority int
	// 0: 未生成，1：生成中，2：生成完成，3：生成失败
	GenStatus StoryGenStatus
	// 0: 不重绘，1：重绘，
	Regen      int32
	Params     string
	Status     int
	Content    string
	TokenNum   int
	ImageUrls  string
	StartTime  int64
	FinishTime int64
	GenType    int
	// 1:故事，2:故事板,3:角色
	TaskType int
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
