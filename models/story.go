package models

import (
	"context"
	"encoding/json"
	"time"

	"github.com/grapery/common-protoc/gen"
	api "github.com/grapery/common-protoc/gen"
	"github.com/grapery/grapery/utils/log"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

const (
	WebpFormat = 1
	PngFormat  = 2
	JpgFormat  = 3
)

type ChapterStruct struct {
	HashID     string                `json:"hash_id,omitempty"`
	Title      string                `json:"title,omitempty"`
	OriginDesc string                `json:"origin_desc,omitempty"`
	Content    string                `json:"content,omitempty"`
	IsEnd      bool                  `json:"is_end,omitempty"`
	Avatar     string                `json:"avatar,omitempty"`
	Prev       string                `json:"prev,omitempty"`
	Roles      map[string]RoleStruct `json:"roles,omitempty"`
}

type RoleStruct struct {
	Name        string `json:"name,omitempty"`
	Description string `json:"description,omitempty"`
	Avatar      string `json:"avatar,omitempty"`
}

type StoryStruct struct {
	Title      string                `json:"title,omitempty"`
	OriginDesc string                `json:"origin_desc,omitempty"`
	Background string                `json:"background,omitempty"`
	AllRoles   map[string]RoleStruct `json:"all_roles,omitempty"`
	Chapters   []ChapterStruct       `json:"chapters,omitempty"`
}

type StoryParams struct {
	UserId       string `json:"user_id,omitempty"`
	StoryContent string `json:"story_content,omitempty"`
	Theme        string `json:"theme,omitempty"`
}

type StoryBoardParams struct {
	StoryContent      string `json:"story_content,omitempty"`
	Theme             string `json:"theme,omitempty"`
	UserId            string `json:"user_id,omitempty"`
	StoryBoardContent string `json:"story_board_content,omitempty"`
}

// 只是生图参数
type StoryImagesParams struct {
	ID uint `gorm:"primary_key,column:id" json:"id,omitempty"`
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

// Story 故事主表
type Story struct {
	ID uint `gorm:"primary_key,column:id" json:"id,omitempty"`
	IDBase
	Title       string `gorm:"column:title;index" json:"title,omitempty"`           // 故事标题
	Name        string `gorm:"column:name" json:"name,omitempty"`                   // 故事名
	ShortDesc   string `gorm:"column:short_desc" json:"short_desc,omitempty"`       // 简短描述
	CreatorID   int64  `gorm:"column:creator_id;index" json:"creator_id,omitempty"` // 创建者ID
	OwnerID     int64  `gorm:"column:owner_id" json:"owner_id,omitempty"`           // 拥有者ID
	GroupID     int64  `gorm:"column:group_id;index" json:"group_id,omitempty"`     // 所属群组ID
	Origin      string `gorm:"column:origin" json:"origin,omitempty"`               // 原始描述
	RootBoardID int    `gorm:"column:root_board_id" json:"root_board_id,omitempty"` // 根故事板ID

	Avatar string `gorm:"column:avatar" json:"avatar,omitempty"` // 封面

	AIGen     bool          `gorm:"column:ai_gen" json:"ai_gen,omitempty"`         // 是否AI生成
	ScopeType api.ScopeType `gorm:"column:scope_type" json:"scope_type,omitempty"` // 可见性
	Status    StoryStatus   `gorm:"column:status" json:"status,omitempty"`         // 状态
	IsClose   bool          `gorm:"column:is_close" json:"is_close,omitempty"`     // 是否关闭

	Params     string `gorm:"column:params" json:"params,omitempty"`           // 生成参数
	Style      string `gorm:"column:style" json:"style,omitempty"`             // 风格
	StyleDesc  string `gorm:"column:style_desc" json:"style_desc,omitempty"`   // 风格描述
	StyleImage string `gorm:"column:style_image" json:"style_image,omitempty"` // 风格图片
	Subject    string `gorm:"column:subject" json:"subject,omitempty"`         // 主题

	LikeCount      int64  `gorm:"column:like_count" json:"like_count,omitempty"`             // 点赞数
	CommentCount   int64  `gorm:"column:comment_count" json:"comment_count,omitempty"`       // 评论数
	ShareCount     int64  `gorm:"column:share_count" json:"share_count,omitempty"`           // 分享数
	FollowCount    int64  `gorm:"column:follow_count" json:"follow_count,omitempty"`         // 关注数
	TotalBoards    int64  `gorm:"column:total_boards" json:"total_boards,omitempty"`         // 故事板总数
	TotalRoles     int64  `gorm:"column:total_roles" json:"total_roles,omitempty"`           // 角色总数
	TotalMembers   int64  `gorm:"column:total_members" json:"total_members,omitempty"`       // 参与人数
	SenceMaxNumber int64  `gorm:"column:sence_max_number" json:"sence_max_number,omitempty"` // 场景最大数量
	Cover          string `gorm:"column:cover" json:"cover,omitempty"`                       // 封面
	TokenNum       int64  `gorm:"column:token_num" json:"token_num,omitempty"`               // 消耗token数
}

func (s *Story) TableName() string {
	return "story"
}

func CreateStory(ctx context.Context, s *Story) (int64, error) {
	if s.Avatar == "" {
		s.Avatar = "https://grapery-dev.oss-cn-shanghai.aliyuncs.com/default.jpg"
	}
	// 日志：创建故事前的参数
	log.Log().Info("[CreateStory] 创建故事参数", zap.Any("story", s))
	if err := DataBase().
		Model(s).
		WithContext(ctx).
		Create(s).Error; err != nil {
		// 日志：创建失败
		log.Log().Error("[CreateStory] 创建故事失败", zap.Error(err), zap.Any("story", s))
		return 0, err
	}
	// 日志：创建成功
	log.Log().Info("[CreateStory] 创建故事成功", zap.Int64("story_id", int64(s.ID)))
	return int64(s.ID), nil
}

func UpdateStory(ctx context.Context, s *Story) error {
	// 日志：更新前参数
	log.Log().Info("[UpdateStory] 更新故事参数", zap.Any("story", s))
	err := DataBase().
		Model(s).
		WithContext(ctx).
		Updates(s).Error
	if err != nil {
		// 日志：更新失败
		log.Log().Error("[UpdateStory] 更新故事失败", zap.Error(err), zap.Any("story", s))
	}
	return err
}

func IncreStoryCommentCount(ctx context.Context, storyId int64) error {
	err := DataBase().Model(&Story{}).
		WithContext(ctx).
		Where("id = ?", storyId).
		Update("comment_count", gorm.Expr("comment_count + 1")).Error
	return err
}

func DecreaseStoryCommentCount(ctx context.Context, storyId int64) error {
	err := DataBase().Model(&Story{}).
		WithContext(ctx).
		Where("id = ?", storyId).
		Update("comment_count", gorm.Expr("comment_count - 1")).Error
	return err
}

func UpdateStorySpecColumns(ctx context.Context, storyId int64, columns map[string]interface{}) error {
	// 日志：指定列更新前参数
	log.Log().Info("[UpdateStorySpecColumns] storyId/columns", zap.Int64("storyId", storyId), zap.Any("columns", columns))
	err := DataBase().
		Model(&Story{}).
		WithContext(ctx).
		Where("id = ?", storyId).
		Updates(columns).Error
	if err != nil {
		log.Log().Error("[UpdateStorySpecColumns] 更新指定列失败", zap.Error(err), zap.Int64("storyId", storyId), zap.Any("columns", columns))
		return err
	}
	return nil
}

func GetStory(ctx context.Context, id int64) (*Story, error) {
	s := &Story{}
	log.Log().Info("[GetStory] 查询故事", zap.Int64("id", id))
	err := DataBase().
		Model(s).
		WithContext(ctx).
		Where("id = ?", id).First(s).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			log.Log().Warn("[GetStory] 未找到故事", zap.Int64("id", id))
			return nil, nil
		}
		log.Log().Error("[GetStory] 查询故事失败", zap.Error(err), zap.Int64("id", id))
		return nil, err
	}
	// 日志：查询成功
	log.Log().Info("[GetStory] 查询故事成功", zap.Int64("id", id))
	if s.Avatar == "" {
		s.Avatar = "https://grapery-dev.oss-cn-shanghai.aliyuncs.com/defatul_avatar.jpg"
	}
	if s.Style == "" {
		s.Style = "吉卜力风格"
	}
	if s.StyleDesc == "" {
		s.StyleDesc = "吉卜力风格，以宫崎骏的动画风格为特点，适合怀旧主题创作。"
	}
	if s.StyleImage == "" {
		s.StyleImage = "https://grapery-dev.oss-cn-shanghai.aliyuncs.com/defatul_avatar.jpg"
	}
	if s.Cover == "" {
		s.Cover = "https://grapery-dev.oss-cn-shanghai.aliyuncs.com/defatul_avatar.jpg"
	}
	if s.SenceMaxNumber == 0 {
		s.SenceMaxNumber = 5
	}
	return s, nil
}

func GetStoryByCreatorID(ctx context.Context, creatorID int64) (*Story, error) {
	s := &Story{}
	log.Log().Info("[GetStoryByCreatorID] 查询creatorId故事", zap.Int64("creatorID", creatorID))
	err := DataBase().
		Model(s).
		WithContext(ctx).
		Where("creator_id = ?", creatorID).
		First(s).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			log.Log().Warn("[GetStoryByCreatorID] 未找到故事", zap.Int64("creatorID", creatorID))
			return nil, nil
		}
		log.Log().Error("[GetStoryByCreatorID] 查询失败", zap.Error(err), zap.Int64("creatorID", creatorID))
		return nil, err
	}
	log.Log().Info("[GetStoryByCreatorID] 查询成功", zap.Int64("creatorID", creatorID))
	return s, nil
}

func GetStoryByOwnerID(ctx context.Context, ownerID int64) ([]*Story, error) {
	s := make([]*Story, 0)
	log.Log().Info("[GetStoryByOwnerID] 查询ownerId故事", zap.Int64("ownerID", ownerID))
	err := DataBase().Model(s).WithContext(ctx).Where("owner_id = ?", ownerID).Find(s).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			log.Log().Warn("[GetStoryByOwnerID] 未找到故事", zap.Int64("ownerID", ownerID))
			return nil, nil
		}
		log.Log().Error("[GetStoryByOwnerID] 查询失败", zap.Error(err), zap.Int64("ownerID", ownerID))
		return nil, err
	}
	log.Log().Info("[GetStoryByOwnerID] 查询成功", zap.Int64("ownerID", ownerID), zap.Int("count", len(s)))
	return s, nil
}

func GetStoryByGroupID(ctx context.Context, groupID int64, page int, pageSize int) ([]*Story, int64, bool, error) {
	var total int64
	var hasMore bool
	s := make([]*Story, 0)
	log.Log().Info("[GetStoryByGroupID] 查询groupId故事", zap.Int64("groupID", groupID), zap.Int("page", page), zap.Int("pageSize", pageSize))
	err := DataBase().Model(s).
		WithContext(ctx).
		Where("group_id = ?", groupID).
		Count(&total).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			log.Log().Warn("[GetStoryByGroupID] 未找到故事", zap.Int64("groupID", groupID))
			return nil, 0, false, nil
		}
		log.Log().Error("[GetStoryByGroupID] 查询失败", zap.Error(err), zap.Int64("groupID", groupID))
		return nil, 0, false, err
	}
	err = DataBase().Model(s).
		WithContext(ctx).
		Where("group_id = ?", groupID).
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&s).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			log.Log().Warn("[GetStoryByGroupID] 未找到故事", zap.Int64("groupID", groupID))
			return nil, 0, false, nil
		}
		log.Log().Error("[GetStoryByGroupID] 查询失败", zap.Error(err), zap.Int64("groupID", groupID))
		return nil, 0, false, err
	}
	log.Log().Info("[GetStoryByGroupID] 查询成功", zap.Int64("groupID", groupID), zap.Int("count", len(s)))
	hasMore = total > int64(page+pageSize)
	return s, total, hasMore, nil
}

func GetStoriesByName(ctx context.Context, name string, offset, number int) ([]*Story, int64, error) {
	var stories []*Story
	var total int64
	log.Log().Info("[GetStoriesByName] 查询故事列表", zap.String("name", name), zap.Int("offset", offset), zap.Int("number", number))
	if err := DataBase().Model(&Story{}).
		WithContext(ctx).
		Where("title like ?", "%"+name+"%").
		Where("deleted = ?", 0).
		Count(&total).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			log.Log().Warn("[GetStoriesByName] 未找到故事", zap.String("name", name))
			return nil, 0, nil
		}
		log.Log().Error("[GetStoriesByName] 统计总数失败", zap.Error(err), zap.String("name", name))
		return nil, 0, err
	}
	if err := DataBase().Model(&Story{}).
		WithContext(ctx).
		Where("title like ?", "%"+name+"%").
		Where("deleted = ?", 0).
		Offset(offset).
		Limit(number).
		Find(&stories).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			log.Log().Warn("[GetStoriesByName] 未找到故事", zap.String("name", name))
			return nil, 0, nil
		}
		log.Log().Error("[GetStoriesByName] 查询失败", zap.Error(err), zap.String("name", name))
		return nil, 0, err
	}
	log.Log().Info("[GetStoriesByName] 查询成功", zap.String("name", name), zap.Int64("total", total), zap.Int("count", len(stories)))
	return stories, total, nil
}

func GetUserCreatedStoryboardsWithStoryId(ctx context.Context, userId int, storyId int, offset, number int) ([]*StoryBoard, int64, error) {
	var boards []*StoryBoard
	var total int64
	log.Log().Info("[GetUserCreatedStoryboardsWithStoryId] 查询用户创建的故事板", zap.Int("userId", userId), zap.Int("storyId", storyId), zap.Int("offset", offset), zap.Int("number", number))
	query := DataBase().
		WithContext(ctx).
		Model(&StoryBoard{}).
		Where("creator_id = ?", userId).
		Where("stage = ?", gen.StoryboardStage_STORYBOARD_STAGE_PUBLISHED)
	if storyId > 0 {
		query = query.Where("id != ?", storyId)
	}
	// Count total records
	if err := query.Count(&total).Error; err != nil {
		log.Log().Error("[GetUserCreatedStoryboardsWithStoryId] 统计总数失败", zap.Error(err), zap.Int("userId", userId), zap.Int("storyId", storyId))
		return nil, 0, err
	}
	// Get paginated records
	if err := query.
		Order("create_at desc").
		Offset((offset - 1) * number).
		Limit(number).Scan(&boards).Error; err != nil {
		log.Log().Error("[GetUserCreatedStoryboardsWithStoryId] 查询失败", zap.Error(err), zap.Int("userId", userId), zap.Int("storyId", storyId))
		return nil, 0, err
	}
	log.Log().Info("[GetUserCreatedStoryboardsWithStoryId] 查询成功", zap.Int("userId", userId), zap.Int64("total", total), zap.Int("count", len(boards)))
	return boards, total, nil
}

func GetUserCreatedRolesWithStoryId(ctx context.Context, userId int, storyId int, offset, number int) ([]*StoryRole, int64, error) {
	var roles = make([]*StoryRole, 0)
	var total int64
	log.Log().Info("[GetUserCreatedRolesWithStoryId] 查询用户创建的角色", zap.Int("userId", userId), zap.Int("storyId", storyId), zap.Int("offset", offset), zap.Int("number", number))
	query := DataBase().Table("story_role").
		WithContext(ctx).
		Where("creator_id = ?", userId).
		Where("deleted = 0")
	if storyId > 0 {
		query = query.Where("story_id != ?", storyId)
	}
	if err := query.Count(&total).Error; err != nil {
		log.Log().Error("[GetUserCreatedRolesWithStoryId] 统计总数失败", zap.Error(err), zap.Int("userId", userId), zap.Int("storyId", storyId))
		return nil, 0, err
	}
	if total == 0 {
		log.Log().Warn("[GetUserCreatedRolesWithStoryId] 未找到角色", zap.Int("userId", userId), zap.Int("storyId", storyId))
		return nil, 0, nil
	}
	if err := query.Order("create_at desc").
		Offset((offset - 1) * number).
		Limit(number).
		Scan(&roles).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			log.Log().Warn("[GetUserCreatedRolesWithStoryId] 未找到角色", zap.Int("userId", userId), zap.Int("storyId", storyId))
			return nil, 0, nil
		}
		log.Log().Error("[GetUserCreatedRolesWithStoryId] 查询失败", zap.Error(err), zap.Int("userId", userId), zap.Int("storyId", storyId))
		return nil, 0, err
	}
	log.Log().Info("[GetUserCreatedRolesWithStoryId] 查询成功", zap.Int("userId", userId), zap.Int64("total", total), zap.Int("count", len(roles)))
	return roles, total, nil
}

func GetUserFollowedStoryIds(ctx context.Context, userId int) ([]int64, error) {
	var storyIds []int64
	log.Log().Info("[GetUserFollowedStoryIds] 查询用户关注的故事ID", zap.Int("userId", userId))
	err := DataBase().Model(&WatchItem{}).WithContext(ctx).
		Select("distinct story_id").
		Where("user_id = ?", userId).
		Where("watch_item_type = ?", WatchItemTypeStory).
		Where("watch_type = ?", WatchTypeIsWatching).
		Where("deleted = 0").
		Pluck("story_id", &storyIds).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			log.Log().Warn("[GetUserFollowedStoryIds] 未找到关注故事", zap.Int("userId", userId))
			return nil, nil
		}
		log.Log().Error("[GetUserFollowedStoryIds] 查询失败", zap.Error(err), zap.Int("userId", userId))
		return nil, err
	}
	log.Log().Info("[GetUserFollowedStoryIds] 查询成功", zap.Int("userId", userId), zap.Int("count", len(storyIds)))
	return storyIds, nil
}

// 根据故事id列表获取故事列表
func GetStoriesByIDs(ctx context.Context, ids []int64) ([]*Story, error) {
	// 去重ID，避免重复查询
	ids = UniqueInt64s(ids)
	if len(ids) == 0 {
		log.Log().Warn("[GetStoriesByIDs] ID列表为空")
		return nil, nil
	}

	var stories []*Story
	log.Log().Info("[GetStoriesByIDs] 批量查询故事", zap.Int64s("ids", ids))
	err := DataBase().Model(&Story{}).
		WithContext(ctx).
		Where("id in (?)", ids).
		Order("create_at desc").
		Find(&stories).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			log.Log().Warn("[GetStoriesByIDs] 未找到故事", zap.Int64s("ids", ids))
			return nil, nil
		}
		log.Log().Error("[GetStoriesByIDs] 查询失败", zap.Error(err), zap.Int64s("ids", ids))
		return nil, err
	}
	log.Log().Info("[GetStoriesByIDs] 查询成功", zap.Int("count", len(stories)))
	return stories, nil
}

func GetTrendingStories(ctx context.Context, offset, pageSize int, starttime, endtime int64) ([]*Story, int64, error) {
	var stories []*Story
	var total int64
	start := time.Unix(starttime, 0)
	end := time.Unix(endtime, 0)

	query := DataBase().Model(&Story{}).
		WithContext(ctx).
		Where("create_at >= ? and create_at <= ?", start, end).
		// 根据点赞数,关注数排序
		Order("like_count desc, follow_count desc")
	if err := query.Count(&total).Error; err != nil {
		log.Log().Error("[GetTrendingStories] 统计总数失败", zap.Error(err), zap.Int("offset", offset), zap.Int("pageSize", pageSize))
		return nil, 0, err
	}
	if total == 0 {
		log.Log().Warn("[GetTrendingStories] 未找到热门故事", zap.Time("start", start), zap.Time("end", end))
		return nil, 0, nil
	}
	log.Log().Info("[GetTrendingStories] 查询热门故事", zap.Time("start", start), zap.Time("end", end), zap.Int("offset", offset), zap.Int("pageSize", pageSize))
	err := query.
		Offset((offset - 1) * pageSize).
		Limit(pageSize).
		Find(&stories).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			log.Log().Warn("[GetTrendingStories] 未找到热门故事", zap.Time("start", start), zap.Time("end", end))
			return nil, 0, nil
		}
		log.Log().Error("[GetTrendingStories] 查询失败", zap.Error(err), zap.Time("start", start), zap.Time("end", end))
		return nil, 0, err
	}
	log.Log().Info("[GetTrendingStories] 查询成功", zap.Int("count", len(stories)))
	return stories, total, nil
}

func GetTrendingStoryRoles(ctx context.Context, offset, pageSize int, starttime, endtime int64) ([]*StoryRole, int64, error) {
	var roles []*StoryRole
	start := time.Unix(starttime, 0)
	end := time.Unix(endtime, 0)
	var total int64
	query := DataBase().Model(&StoryRole{}).
		WithContext(ctx).
		Where("create_at >= ? and create_at <= ?", start, end).
		// 根据参与故事、点赞数,关注数排序
		Order("like_count desc, follow_count desc")
	if err := query.Count(&total).Error; err != nil {
		log.Log().Error("[GetTrendingStoryRoles] 统计总数失败", zap.Error(err), zap.Time("start", start), zap.Time("end", end))
		return nil, 0, err
	}
	log.Log().Info("[GetTrendingStoryRoles] 查询热门角色", zap.Time("start", start), zap.Time("end", end), zap.Int("offset", offset), zap.Int("pageSize", pageSize))
	err := DataBase().Model(&StoryRole{}).
		WithContext(ctx).
		Where("create_at >= ? and create_at <= ?", start, end).
		// 根据参与故事、点赞数,关注数排序
		Order("like_count desc, follow_count desc").
		Offset(offset).
		Limit(pageSize).
		Find(&roles).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			log.Log().Warn("[GetTrendingStoryRoles] 未找到热门角色", zap.Time("start", start), zap.Time("end", end))
			return nil, 0, nil
		}
		log.Log().Error("[GetTrendingStoryRoles] 查询失败", zap.Error(err), zap.Time("start", start), zap.Time("end", end))
		return nil, 0, err
	}
	log.Log().Info("[GetTrendingStoryRoles] 查询成功", zap.Int("count", len(roles)))
	return roles, total, nil
}

// 新增：分页获取Story列表
func GetStoryList(ctx context.Context, offset, limit int) ([]*Story, int64, error) {
	var stories []*Story
	var total int64
	query := DataBase().Model(&Story{}).
		WithContext(ctx).
		Order("create_at desc")
	if err := query.Count(&total).Error; err != nil {
		log.Log().Error("[GetStoryList] 统计总数失败", zap.Error(err), zap.Int("offset", offset), zap.Int("limit", limit))
		return nil, 0, err
	}
	log.Log().Info("[GetStoryList] 分页获取故事列表", zap.Int("offset", offset), zap.Int("limit", limit))
	err := query.
		WithContext(ctx).
		Offset((offset - 1) * limit).
		Limit(limit).
		Order("create_at desc").
		Find(&stories).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		log.Log().Error("[GetStoryList] 查询失败", zap.Error(err), zap.Int("offset", offset), zap.Int("limit", limit))
		return nil, 0, err
	}
	log.Log().Info("[GetStoryList] 查询成功", zap.Int("count", len(stories)))
	return stories, total, nil
}

// 新增：通过Title唯一查询
func GetStoryByTitleUnique(ctx context.Context, title string) (*Story, error) {
	story := &Story{}
	err := DataBase().Model(story).
		WithContext(ctx).
		Where("title = ?", title).
		First(story).Error
	if err != nil {
		return nil, err
	}
	return story, nil
}

func UpdateStoryStyle(ctx context.Context, storyId int64, style string) error {
	err := DataBase().Model(&Story{}).
		WithContext(ctx).
		Where("id = ?", storyId).
		Update("style", style).Error
	return err
}

func UpdateStorySenceMaxNumber(ctx context.Context, storyId int64, senceMaxNumber int64) error {
	err := DataBase().Model(&Story{}).
		WithContext(ctx).
		Where("id = ?", storyId).
		Update("sence_max_number", senceMaxNumber).Error
	return err
}

func UpdateStoryAvatar(ctx context.Context, storyId int64, avatar string) error {
	err := DataBase().Model(&Story{}).
		WithContext(ctx).
		Where("id = ?", storyId).
		Update("avatar", avatar).Error
	return err
}

func UpdateStoryCover(ctx context.Context, storyId int64, cover string) error {
	err := DataBase().Model(&Story{}).
		WithContext(ctx).
		Where("id = ?", storyId).
		Update("cover", cover).Error
	return err
}

/*
	查询一个故事的所有参与者

需要连表查询，查询出所有参与者的ID，然后根据ID查询出所有参与者的信息
根据故事id，在故事板表中（包含storyId），查出有效的发布状态的故事板的creatorId，然后根据creatorId查询出所有参与者的信息
最后将所有参与者的信息合并，返回。注意分页处理，根据参与者创建的故事板的点赞数排序，每页10条,返回当页的数据，以及是否还有更多，以及总数
*/
func GetStoryParticipants(ctx context.Context, storyId int64, page int, pageSize int) ([]*User, bool, int64, error) {
	var users []*User
	var userIds []int64
	var total int64
	var hasMore bool
	query := DataBase().Model(&StoryBoard{}).
		WithContext(ctx).
		Where("story_id = ?", storyId).
		Where("stage = ?", api.StoryboardStage_STORYBOARD_STAGE_FINISHED)
	query = query.Select("creator_id").Group("creator_id")
	query = query.Order("like_count desc")
	query = query.Offset((page - 1) * pageSize).Limit(pageSize)
	query = query.Find(&userIds)
	query.Count(&total)
	if total > int64(page*pageSize) {
		hasMore = true
	}
	users, err := GetUsersByIds(ctx, userIds)
	if err != nil {
		return nil, false, 0, err
	}
	return users, hasMore, total, nil

}

func IncreaseStoryRoleNum(ctx context.Context, storyId int64, roleId int64, num int64) error {
	err := DataBase().Model(&Story{}).
		WithContext(ctx).
		Where("id = ?", storyId).
		Update("total_roles", gorm.Expr("total_roles + ?", num)).Error
	return err
}

func DecreaseStoryRolesNum(ctx context.Context, storyId int64, roleId int64, num int64) error {
	err := DataBase().Model(&Story{}).
		WithContext(ctx).
		Where("id = ?", storyId).
		Update("total_roles", gorm.Expr("total_roles - ?", num)).Error
	return err
}

func IncreaseStoryBoardsNum(ctx context.Context, storyId int64, num int64) error {
	err := DataBase().Model(&Story{}).
		WithContext(ctx).
		Where("id = ?", storyId).
		Update("total_boards", gorm.Expr("total_boards + ?", num)).Error
	return err
}

func DecreaseStoryBoardsNum(ctx context.Context, storyId int64, num int64) error {
	err := DataBase().Model(&Story{}).
		WithContext(ctx).
		Where("id = ?", storyId).
		Update("total_boards", gorm.Expr("total_boards - ?", num)).Error
	return err
}

func IncreaseStoryFollowersNum(ctx context.Context, storyId int64, num int64) error {
	err := DataBase().Model(&Story{}).
		WithContext(ctx).
		Where("id = ?", storyId).
		Update("follow_count", gorm.Expr("follow_count + ?", num)).Error
	return err
}

func DecreaseStoryFollowersNum(ctx context.Context, storyId int64, num int64) error {
	err := DataBase().Model(&Story{}).
		WithContext(ctx).
		Where("id = ?", storyId).
		Update("follow_count", gorm.Expr("follow_count - ?", num)).Error
	return err
}

func IncreaseStoryLikeNum(ctx context.Context, storyId int64, num int64) error {
	err := DataBase().Model(&Story{}).
		WithContext(ctx).
		Where("id = ?", storyId).
		Update("like_count", gorm.Expr("like_count + ?", num)).Error
	return err
}

func DecreaseStoryLikeNum(ctx context.Context, storyId int64, num int64) error {
	err := DataBase().Model(&Story{}).
		WithContext(ctx).
		Where("id = ?", storyId).
		Update("like_count", gorm.Expr("like_count - ?", num)).Error
	return err
}

func IncreaseStoryShareNum(ctx context.Context, storyId int64, num int64) error {
	err := DataBase().Model(&Story{}).
		WithContext(ctx).
		Where("id = ?", storyId).
		Update("share_count", gorm.Expr("share_count + ?", num)).Error
	return err
}

func DecreaseStoryShareNum(ctx context.Context, storyId int64, num int64) error {
	err := DataBase().Model(&Story{}).
		WithContext(ctx).
		Where("id = ?", storyId).
		Update("share_count", gorm.Expr("share_count - ?", num)).Error
	return err
}

func IncreaseStoryCommenNum(ctx context.Context, storyId int64, num int64) error {
	err := DataBase().Model(&Story{}).
		WithContext(ctx).
		Where("id = ?", storyId).
		Update("comment_count", gorm.Expr("comment_count + ?", num)).Error
	return err
}

func DecreaseStoryCommenNum(ctx context.Context, storyId int64, num int64) error {
	err := DataBase().Model(&Story{}).
		WithContext(ctx).
		Where("id = ?", storyId).
		Update("comment_count", gorm.Expr("comment_count - ?", num)).Error
	return err
}
