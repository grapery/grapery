package models

import (
	"context"
	"encoding/json"

	"gorm.io/gorm"
)

// 场景，剧情，故事板
type StoryBoard struct {
	IDBase
	Title       string
	Description string
	CreatorID   int64
	StoryID     int64
	PrevId      int64
	Avatar      string
	Status      int
	// 0: 初始化，1：生成中，2：生成完成，3：生成失败
	Stage    int
	Params   string
	ForkAble bool
	ForkNum  int
	Level    int
	IsAiGen  bool
}

func (board StoryBoard) TableName() string {
	return "story_board"
}

func CreateStoryBoard(ctx context.Context, board *StoryBoard) (int64, error) {
	if err := DataBase().Model(board).
		WithContext(ctx).
		Create(board).Error; err != nil {
		return 0, err
	}
	return int64(board.IDBase.ID), nil
}

func GetStoryboard(ctx context.Context, id int64) (*StoryBoard, error) {
	board := &StoryBoard{}
	err := DataBase().Model(board).
		WithContext(ctx).
		Where("id = ? and status >= 0", id).
		First(board).Error
	if err != nil {
		return nil, err
	}
	return board, nil
}

func GetStoryboardsByPrevId(ctx context.Context, prevId int64) ([]*StoryBoard, error) {
	var boards []*StoryBoard
	err := DataBase().Model(&StoryBoard{}).
		WithContext(ctx).
		Where("prev_id = ? and status >= 0", prevId).Find(&boards).Error
	if err != nil {
		return nil, err
	}
	return boards, nil
}

func GetStoryboardsByStory(ctx context.Context, storyID int64) ([]*StoryBoard, error) {
	var boards []*StoryBoard
	err := DataBase().Model(&StoryBoard{}).
		WithContext(ctx).
		Where("story_id = ? and status >= 0", storyID).Find(&boards).Error
	if err != nil {
		return nil, err
	}
	return boards, nil
}

func GetStoryboardsByCreator(ctx context.Context, creatorID int64) ([]*StoryBoard, error) {
	var boards []*StoryBoard
	err := DataBase().Model(&StoryBoard{}).
		WithContext(ctx).
		Where("creator_id = ? and status >= 0", creatorID).Find(&boards).Error
	if err != nil {
		return nil, err
	}
	return boards, nil
}

func DelStoryboard(ctx context.Context, id int64) error {
	err := DataBase().Model(&StoryBoard{}).WithContext(ctx).
		Where("id = ?", id).Update("status = ?", -1).Error
	return err
}

func UpdateStoryboard(ctx context.Context, board *StoryBoard) error {
	return DataBase().Model(board).WithContext(ctx).
		Where("id = ?", board.IDBase.ID).Updates(board).Error
}

func UpdateStoryboardMultiColumn(ctx context.Context, id int64, columns map[string]interface{}) error {
	return DataBase().Model(&StoryBoard{}).WithContext(ctx).
		Where("id = ?", id).Updates(columns).Error
}

func GetStoryContributors(ctx context.Context, storyID int64) ([]*User, error) {
	contributors := make([]*User, 0)
	err := DataBase().Model(&User{}).WithContext(ctx).
		Where("id in (select distinct(creator_id) from story_board where story_id = ?)", storyID).
		Find(&contributors).
		Error
	if err != nil {
		return nil, err
	}
	return contributors, nil
}

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
	Uuid           string
	Step           int
	LLmPlatform    string
	NegativePrompt string
	PositivePrompt string
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
	// 1:故事，2：故事板
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

type StoryBoardScene struct {
	IDBase
	Content      string
	CharacterIds string
	CreatorId    int64
	StoryId      int64
	BoardId      int64
	ImagePrompts string
	AudioPrompts string
	VideoPrompts string
	IsGenerating int
	GenResult    string
	Status       int
}

func (board StoryBoardScene) TableName() string {
	return "story_board_sence"
}

func CreateStoryBoardScene(ctx context.Context, scene *StoryBoardScene) (int64, error) {
	scene.Status = 1
	if err := DataBase().Model(scene).
		WithContext(ctx).
		Create(scene).Error; err != nil {
		return 0, err
	}
	return int64(scene.IDBase.ID), nil
}

func GetStoryBoardScene(ctx context.Context, id int64) (*StoryBoardScene, error) {
	scene := &StoryBoardScene{}
	err := DataBase().Model(scene).
		WithContext(ctx).
		Where("id = ?", id).
		Where("status >= 0").
		First(scene).Error
	if err != nil {
		return nil, err
	}
	return scene, nil
}

func GetStoryBoardSceneByBoard(ctx context.Context, boardId int64) ([]*StoryBoardScene, error) {
	scenes := make([]*StoryBoardScene, 0)
	err := DataBase().Model(&StoryBoardScene{}).
		WithContext(ctx).
		Where("board_id = ?", boardId).
		Where("status >= 0").
		Find(&scenes).Error
	if err != nil {
		return nil, err
	}
	return scenes, nil
}

func GetStoryBoardScenesByBoard(ctx context.Context, boardId int64) ([]*StoryBoardScene, error) {
	scenes := make([]*StoryBoardScene, 0)
	err := DataBase().Model(&StoryBoardScene{}).
		WithContext(ctx).
		Where("board_id = ?", boardId).
		Where("status >= 0").
		Find(&scenes).Error
	if err != nil {
		return nil, err
	}
	return scenes, nil
}

func DelStoryBoardScene(ctx context.Context, id int64) error {
	err := DataBase().Model(&StoryBoardScene{}).WithContext(ctx).
		Where("id = ?", id).
		Update("status = ?", -1).Error
	return err
}

func UpdateStoryBoardScene(ctx context.Context, scene *StoryBoardScene) error {
	return DataBase().Model(scene).WithContext(ctx).
		Where("id = ?", scene.IDBase.ID).
		Where("status >= 0").
		Updates(scene).Error
}

func UpdateStoryBoardSceneMultiColumn(ctx context.Context, id int64, columns map[string]interface{}) error {
	return DataBase().Model(&StoryBoardScene{}).
		WithContext(ctx).
		Where("id = ?", id).
		Where("status >= 0").
		Updates(columns).Error
}

func SetGenResult(ctx context.Context, id int64, result string) error {
	return DataBase().Model(&StoryBoardScene{}).
		WithContext(ctx).
		Where("id = ?", id).
		Where("status >= 0").
		Update("gen_result = ?", result).Error
}

func SetGenerating(ctx context.Context, id int64, isGenerating int) error {
	return DataBase().Model(&StoryBoardScene{}).
		WithContext(ctx).
		Where("id = ?", id).
		Where("status >= 0").
		Update("is_generating = ?", isGenerating).Error
}

func UpdateStoryBoardSceneStatus(ctx context.Context, id int64, status int) error {
	return DataBase().Model(&StoryBoardScene{}).
		WithContext(ctx).
		Where("id = ?", id).
		Where("status >= 0").
		Update("status = ?", status).Error
}

func BatchUpdateStoryBoardSceneStatus(ctx context.Context, ids []int64, status int) error {
	return DataBase().Model(&StoryBoardScene{}).
		WithContext(ctx).
		Where("id in (?)", ids).
		Where("status >= 0").
		Update("status = ?", status).Error
}

func UpdateStoryBoardSceneContent(ctx context.Context, id int64, content string) error {
	return DataBase().Model(&StoryBoardScene{}).
		WithContext(ctx).
		Where("id = ?", id).
		Where("status >= 0").
		Update("content = ?", content).Error
}

func UpdateStoryBoardSceneCharacterIds(ctx context.Context, id int64, characterIds string) error {
	return DataBase().Model(&StoryBoardScene{}).
		WithContext(ctx).
		Where("id = ?", id).
		Where("status >= 0").
		Update("character_ids = ?", characterIds).Error
}

func UpdateStoryBoardSceneImagePrompts(ctx context.Context, id int64, imagePrompts string) error {
	return DataBase().Model(&StoryBoardScene{}).
		WithContext(ctx).
		Where("id = ?", id).
		Where("status >= 0").
		Update("image_prompts = ?", imagePrompts).Error
}

func UpdateStoryBoardSceneAudioPrompts(ctx context.Context, id int64, audioPrompts string) error {
	return DataBase().Model(&StoryBoardScene{}).
		WithContext(ctx).
		Where("id = ?", id).
		Where("status >= 0").
		Update("audio_prompts = ?", audioPrompts).Error
}

func UpdateStoryBoardSceneVideoPrompts(ctx context.Context, id int64, videoPrompts string) error {
	return DataBase().Model(&StoryBoardScene{}).
		WithContext(ctx).
		Where("id = ?", id).
		Where("status >= 0").
		Update("video_prompts = ?", videoPrompts).Error
}

func UpdateStoryBoardSceneGenResult(ctx context.Context, id int64, genResult string) error {
	return DataBase().Model(&StoryBoardScene{}).
		WithContext(ctx).
		Where("id = ?", id).
		Where("status >= 0").
		Update("gen_result = ?", genResult).Error
}

type StoryBoardRole struct {
	IDBase
	CreatorId int64
	StoryId   int64
	BoardId   int64
	RoleId    int64
	Name      string
	Avatar    string
	Status    int
}

func (board StoryBoardRole) TableName() string {
	return "story_board_role"
}

func CreateStoryBoardRole(ctx context.Context, role *StoryBoardRole) (int64, error) {
	role.Status = 1
	if err := DataBase().Model(role).
		WithContext(ctx).
		Create(role).Error; err != nil {
		return 0, err
	}
	return int64(role.IDBase.ID), nil
}

func GetStoryBoardRoles(ctx context.Context, boardId int64) ([]*StoryBoardRole, error) {
	role := make([]*StoryBoardRole, 0)
	err := DataBase().Model(role).
		WithContext(ctx).
		Where("board_id = ?", boardId).
		Where("status >= 0").
		Scan(&role).Error
	if err != nil {
		return nil, err
	}
	return role, nil
}

func GetStoryBoardRolesByBoard(ctx context.Context, boardId int64) ([]*StoryBoardRole, error) {
	roles := make([]*StoryBoardRole, 0)
	err := DataBase().Model(&StoryBoardRole{}).
		WithContext(ctx).
		Where("board_id = ?", boardId).
		Where("status >= 0").
		Find(&roles).Error
	if err != nil {
		return nil, err
	}
	return roles, nil
}

func DelStoryBoardRole(ctx context.Context, id int64) error {
	err := DataBase().Model(&StoryBoardRole{}).WithContext(ctx).
		Where("id = ?", id).
		Update("status = ?", -1).Error
	return err
}

func UpdateStoryBoardRole(ctx context.Context, role *StoryBoardRole) error {
	return DataBase().Model(role).WithContext(ctx).
		Where("id = ?", role.IDBase.ID).
		Where("status >= 0").
		Updates(role).Error
}

func UpdateStoryBoardRoleMultiColumn(ctx context.Context, id int64, columns map[string]interface{}) error {
	return DataBase().Model(&StoryBoardRole{}).
		WithContext(ctx).
		Where("id = ?", id).
		Where("status >= 0").
		Updates(columns).Error
}

// 获取角色参与的某一个故事的所有故事板
func GetStoryBoardsByRoleID(ctx context.Context, roleID int64) ([]*StoryBoard, error) {
	var boards []*StoryBoard
	if err := DataBase().Model(&StoryBoard{}).
		Where("role_id = ?", roleID).
		Where("status >= 0").
		Find(&boards).Error; err != nil {
		return nil, err
	}
	return boards, nil
}
