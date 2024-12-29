package models

import (
	"context"
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

func GetStoryBoardByStoryAndPrevId(ctx context.Context, storyID int64, prevId int64, page int, pageSize int, orderBy string) ([]*StoryBoard, error) {
	var boards []*StoryBoard
	query := DataBase().Model(&StoryBoard{}).
		WithContext(ctx).
		Where("story_id = ? and prev_id = ? and status >= 0", storyID, prevId)

	if orderBy != "" {
		if orderBy == "create_at" {
			query = query.Order("create_at desc")
		} else if orderBy == "update_at" {
			query = query.Order("update_at desc")
		} else if orderBy == "fork_num" {
			query = query.Order("fork_num desc")
		} else if orderBy == "like" {
			query = query.Order("like desc")
		}
	}

	err := query.Offset(page * pageSize).Limit(pageSize).Find(&boards).Error
	if err != nil {
		return nil, err
	}
	return boards, nil
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
	var boardsIDs []int64
	if err := DataBase().Select("board_id").Model(&StoryBoardRole{}).
		Where("role_id = ?", roleID).
		Where("status >= 0").
		Find(&boardsIDs).Limit(10).Error; err != nil {
		return nil, err
	}
	if len(boardsIDs) == 0 {
		return nil, nil
	}
	err := DataBase().Model(&StoryBoard{}).
		WithContext(ctx).
		Where("id in (?)", boardsIDs).
		Where("status >= 0").
		Find(&boards).Error
	if err != nil {
		return nil, err
	}
	return boards, nil
}

func GetStoryBoardSencesByRoleID(ctx context.Context, roleID int64) ([]*StoryBoardScene, error) {
	var boards []*StoryBoard
	var boardsIDs []int64
	if err := DataBase().Select("board_id").Model(&StoryBoardRole{}).
		Where("role_id = ?", roleID).
		Where("status > 0").
		Find(&boardsIDs).Limit(10).Error; err != nil {
		return nil, err
	}
	if len(boardsIDs) == 0 {
		return nil, nil
	}
	err := DataBase().Model(&StoryBoard{}).
		WithContext(ctx).
		Where("id in (?)", boardsIDs).
		Where("status > 0").
		Find(&boards).Error
	if err != nil {
		return nil, err
	}
	var scenes []*StoryBoardScene
	err = DataBase().Model(&StoryBoardScene{}).
		WithContext(ctx).
		Where("board_id in (?)", boardsIDs).
		Where("status > 0").
		Find(&scenes).Error
	if err != nil {
		return nil, err
	}
	return scenes, nil
}
