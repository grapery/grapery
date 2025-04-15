package models

import (
	"context"
	"encoding/json"

	api "github.com/grapery/common-protoc/gen"
)

const (
	WebpFormat = 1
	PngFormat  = 2
	JpgFormat  = 3
)

type ChapterStruct struct {
	HashID     string
	Title      string
	OriginDesc string
	Content    string
	IsEnd      bool
	Avatar     string
	Prev       string
	Roles      map[string]RoleStruct
}

type RoleStruct struct {
	Name        string
	Description string
	Avatar      string
}

type StoryStruct struct {
	Title      string
	OriginDesc string
	Background string
	AllRoles   map[string]RoleStruct
	Chapters   []ChapterStruct
}

type StoryParams struct {
	UserId       string
	StoryContent string
	Theme        string
}

type StoryBoardParams struct {
	StoryContent      string
	Theme             string
	UserId            string
	StoryBoardContent string
}

// 只是生图参数
type StoryImagesParams struct {
	IDBase
	// 角色描述
	Roles []StoryRole `json:"roles"`
	// 故事描述,根据Origin拆解生成的信息
	StoryDescription string `json:"story_description"`
	// 漫画ID总数
	NumIds int32 `json:"num_ids"`
	// 生成步数
	NumSteps int32 `json:"num_steps"`
	// 使用的生成模型
	SdModel string `json:"sd_model"`
	// 用户提供的参考图
	RefImage string `json:"ref_image"`
	// 漫画布局
	ComicLayoutStyle string `json:"comic_layout_style"`
	// 漫画风格
	ComicStyle string `json:"comic_style"`
	// 和参考图的相似度
	StyleStrengthRatio float64 `json:"style_strength_ratio"`
	// 故事默认的否定项
	NegativePrompt string `json:"negative_prompt"`
	// 输出质量
	OutputQuality int32 `json:"output_quality"`
	// 引导缩放
	GuidanceScale float32 `json:"guidance_scale"`
	// 输出格式
	OutputFormat int32 `json:"output_format"`
	// 输出宽高
	ImageWidth  int32 `json:"image_width"`
	ImageHeight int32 `json:"image_height"`
	// 自注意力模型层数
	Self32AttentionLayers int32 `json:"self_32_attention_layers"`
	// 自注意力模型层数
	Self64AttentionLayers int32 `json:"self_64_attention_layers"`
	// 自注意力模型层数
	Self128AttentionLayers int32 `json:"self_128_attention_layers"`

	Version int64 `json:"version"`
}

func (s StoryParams) String() string {
	data, _ := json.Marshal(s)
	return string(data)
}

type StoryStatus int

const (
	StoryStatusNotSpecified StoryStatus = 0
	StoryStatusDraft        StoryStatus = 1
	StoryStatusOpen         StoryStatus = 2
	StoryStatusClose        StoryStatus = 3
)

type Story struct {
	IDBase
	Title        string
	Name         string
	ShortDesc    string
	CreatorID    int64
	OwnerID      int64
	GroupID      int64
	Origin       string
	RootBoardID  int
	AIGen        bool
	Avatar       string
	OriginAvatar string
	Visable      api.ScopeType
	Status       StoryStatus
	IsAchieve    bool
	IsClose      bool
	IsPrivate    bool
	Params       string
	Style        string
	StyleDesc    string
	StyleImage   string
	Subject      string
	SubjectDesc  string

	LikeCount    int64
	CommentCount int64
	ShareCount   int64
	FollowCount  int64
	TotalBoards  int64
	TotalRoles   int64
	TotalMembers int64
}

func (s *Story) TableName() string {
	return "story"
}

func CreateStory(ctx context.Context, s *Story) (int64, error) {
	if s.Avatar == "" {
		s.Avatar = "https://grapery-1301865260.cos.ap-shanghai.myqcloud.com/avator/tmp3evp1xxl.png"
	}
	if err := DataBase().Model(s).WithContext(ctx).Create(s).Error; err != nil {
		return 0, err
	}
	return int64(s.ID), nil
}

func UpdateStory(ctx context.Context, s *Story) error {
	err := DataBase().Model(s).WithContext(ctx).Updates(s).Error
	return err
}

func UpdateStorySpecColumns(ctx context.Context, storyId int64, columns map[string]interface{}) error {
	err := DataBase().Model(&Story{}).
		WithContext(ctx).Where("id = ?", storyId).
		Updates(columns).Error
	if err != nil {
		return err
	}
	return nil
}

func GetStory(ctx context.Context, id int64) (*Story, error) {
	s := &Story{}
	err := DataBase().Model(s).WithContext(ctx).Where("id = ?", id).First(s).Error
	if err != nil {
		return nil, err
	}
	if s.Avatar == "" {
		s.Avatar = "https://grapery-1301865260.cos.ap-shanghai.myqcloud.com/avator/tmp3evp1xxl.png"
	}
	return s, nil
}

func GetStoryByCreatorID(ctx context.Context, creatorID int64) (*Story, error) {
	s := &Story{}
	err := DataBase().Model(s).WithContext(ctx).Where("creator_id = ?", creatorID).First(s).Error
	if err != nil {
		return nil, err
	}
	return s, nil
}

func GetStoryByOwnerID(ctx context.Context, ownerID int64) ([]*Story, error) {
	s := make([]*Story, 0)
	err := DataBase().Model(s).WithContext(ctx).Where("owner_id = ?", ownerID).Find(s).Error
	if err != nil {
		return nil, err
	}
	return s, nil
}

func GetStoryByGroupID(ctx context.Context, groupID int64, page int, pageSize int) ([]*Story, error) {
	s := make([]*Story, 0)
	err := DataBase().Model(s).
		WithContext(ctx).
		Where("group_id = ?", groupID).
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&s).Error
	if err != nil {
		return nil, err
	}
	return s, nil
}

func GetStoriesByName(ctx context.Context, name string, offset, number int) ([]*Story, int64, error) {
	var stories []*Story
	var total int64
	if err := DataBase().Model(&Story{}).
		Where("title like ?", "%"+name+"%").
		Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if err := DataBase().Model(&Story{}).
		Where("title like ?", "%"+name+"%").
		Offset(offset).
		Limit(number).
		Find(&stories).Error; err != nil {
		return nil, 0, err
	}
	return stories, total, nil
}

func GetUserCreatedStoryboardsWithStoryId(ctx context.Context, userId int, storyId int, offset, number int) ([]*StoryBoard, int64, error) {
	var boards []*StoryBoard
	var total int64

	query := DataBase().
		Model(&StoryBoard{}).
		Where("creator_id = ?", userId)
	if storyId > 0 {
		query = query.Where("id != ?", storyId)
	}

	// Count total records
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Get paginated records
	if err := query.
		Order("create_at desc").
		Offset(offset).
		Limit(number).Scan(&boards).Error; err != nil {
		return nil, 0, err
	}

	return boards, total, nil
}

func GetUserCreatedRolesWithStoryId(ctx context.Context, userId int, storyId int, offset, number int) ([]*StoryRole, int64, error) {
	var roles = make([]*StoryRole, 0)
	var total int64

	query := DataBase().Table("story_role").Where("creator_id = ?", userId)
	if storyId > 0 {
		query = query.Where("story_id != ?", storyId)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if err := query.Order("create_at desc").Offset(offset).Limit(number).Scan(&roles).Error; err != nil {
		return nil, 0, err
	}
	return roles, total, nil
}

func GetUserFollowedStoryIds(ctx context.Context, userId int) ([]int64, error) {
	var storyIds []int64
	err := DataBase().Model(&WatchItem{}).
		Select("distinct story_id").
		Where("user_id = ? and deleted = 0 and watch_item_type = ? and watch_type = ?",
			userId, WatchItemTypeStory, WatchTypeIsWatch).
		Pluck("story_id", &storyIds).Error
	if err != nil {
		return nil, err
	}
	return storyIds, nil
}

// 根据故事id列表获取故事列表
func GetStoriesByIDs(ctx context.Context, ids []int64) ([]*Story, error) {
	var stories []*Story
	err := DataBase().Model(&Story{}).
		WithContext(ctx).
		Where("id in (?)", ids).
		Order("create_at desc").
		Find(&stories).Error
	if err != nil {
		return nil, err
	}
	return stories, nil
}
