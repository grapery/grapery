package active

import (
	"context"

	api "github.com/grapery/common-protoc/gen"
	"github.com/grapery/grapery/models"
	"go.uber.org/zap"
)

var (
	server         ActiveServer
	logFieldModels = zap.Fields(
		zap.String("module", "pkg"))
)

func init() {
	server = NewActiveService()
}

func GetActiveServer() ActiveServer {
	return server
}

func NewActiveService() *ActiveService {
	return &ActiveService{}
}

// need do some log
type ActiveServer interface {
	WriteGroupActive(ctx context.Context, group *models.Group, story *models.Story, role *models.StoryRole, userId int64, activeType api.ActiveType) error
	WriteStoryActive(ctx context.Context, group *models.Group, story *models.Story, board *models.StoryBoard, role *models.StoryRole, userId int64, activeType api.ActiveType) error
	WriteRoleActive(ctx context.Context, group *models.Group, story *models.Story, role *models.StoryRole, userId int64, activeType api.ActiveType) error
}

type ActiveService struct {
}

// 写入小组活动
func (ts *ActiveService) WriteGroupActive(ctx context.Context, group *models.Group, story *models.Story, role *models.StoryRole, userId int64, activeType api.ActiveType) error {
	activeItem := &models.Active{
		UserId:     userId,
		ActiveType: activeType,
		GroupId:    int64(group.ID),
		Content:    group.Name,
		Status:     1,
	}
	switch activeType {
	case api.ActiveType_FollowGroup:
		activeItem.Content = group.Name
	case api.ActiveType_JoinGroup:
		activeItem.Content = group.Name
	case api.ActiveType_LikeGroup:
		activeItem.Content = group.Name
	}
	err := activeItem.Create()
	if err != nil {
		return err
	}
	return nil
}

// 写入故事活动
func (ts *ActiveService) WriteStoryActive(ctx context.Context, group *models.Group, story *models.Story, boards *models.StoryBoard, role *models.StoryRole, userId int64, activeType api.ActiveType) error {
	activeItem := &models.Active{
		UserId:     userId,
		ActiveType: activeType,
		GroupId:    int64(group.ID),
		StoryId:    int64(story.ID),
		Status:     1,
	}
	switch activeType {
	case api.ActiveType_NewStory:
		activeItem.Content = story.Title
	case api.ActiveType_FollowStory:
		activeItem.Content = story.Title
	case api.ActiveType_LikeStory:
		activeItem.Content = story.Title
	}
	err := activeItem.Create()
	if err != nil {
		return err
	}
	return nil
}

// 写入角色活动
func (ts *ActiveService) WriteRoleActive(ctx context.Context, group *models.Group, story *models.Story, info *models.StoryRole, userId int64, activeType api.ActiveType) error {
	activeItem := &models.Active{
		UserId:      userId,
		ActiveType:  activeType,
		StoryRoleId: int64(info.ID),
		GroupId:     int64(group.ID),
		StoryId:     int64(story.ID),
		Status:      1,
	}
	switch activeType {
	case api.ActiveType_NewRole:
		activeItem.Content = info.CharacterName
	case api.ActiveType_FollowRole:
		activeItem.Content = info.CharacterName
	case api.ActiveType_LikeRole:
		activeItem.Content = info.CharacterName
	}
	err := activeItem.Create()
	if err != nil {
		return err
	}
	return nil
}
