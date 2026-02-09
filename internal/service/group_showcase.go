package service

import (
	"context"
	"errors"
	"time"

	"github.com/grapestree/fgrapery/grapery/internal/domain"
	"github.com/grapestree/fgrapery/grapery/internal/utils"
	"go.uber.org/zap"
)

// GroupShowcaseService 小组展示服务层
type GroupShowcaseService struct {
	repo   domain.Repository
	logger *zap.Logger
}

// NewGroupShowcaseService 创建小组展示服务
func NewGroupShowcaseService(repo domain.Repository, logger *zap.Logger) *GroupShowcaseService {
	return &GroupShowcaseService{
		repo:   repo,
		logger: logger,
	}
}

// AddShowcase 添加展示内容到小组
func (s *GroupShowcaseService) AddShowcase(ctx context.Context, groupID, userID string, req *domain.AddGroupShowcaseRequest) (*domain.GroupShowcase, error) {
	s.logger.Info("adding showcase to group",
		zap.String("groupID", groupID),
		zap.String("userID", userID),
		zap.String("contentID", req.ContentID),
		zap.String("contentType", string(req.ContentType)))

	// 1. 检查小组是否存在
	group, err := s.repo.GroupByID(ctx, groupID)
	if err != nil {
		s.logger.Error("group not found", zap.String("groupID", groupID), zap.Error(err))
		return nil, errors.New("group not found")
	}

	// 2. 检查用户是否有权限添加（组长、管理员、模组，或内容作者本人）
	role, err := s.repo.GetMemberRole(ctx, groupID, userID)
	if err != nil {
		s.logger.Error("failed to get member role", zap.Error(err))
		return nil, errors.New("not a group member")
	}

	hasPermission := role == domain.RoleOwner || role == domain.RoleAdmin ||
		role == domain.RoleModerator

	// 3. 验证内容是否存在并检查权限
	switch req.ContentType {
	case domain.GroupShowcaseTypeFragment:
		fragment, err := s.repo.FragmentByID(ctx, req.ContentID)
		if err != nil {
			return nil, errors.New("fragment not found")
		}
		// 如果不是管理员，则需要是碎片作者
		if !hasPermission && fragment.AuthorID != userID {
			return nil, errors.New("permission denied: not fragment owner")
		}

	case domain.GroupShowcaseTypeStory:
		story, err := s.repo.StoryByID(ctx, req.ContentID)
		if err != nil {
			return nil, errors.New("story not found")
		}
		// 如果不是管理员，则需要是故事作者
		if !hasPermission && story.AuthorID != userID {
			return nil, errors.New("permission denied: not story owner")
		}

	default:
		return nil, errors.New("invalid content type")
	}

	// 4. 创建展示记录
	now := time.Now().Unix()
	showcase := &domain.GroupShowcase{
		ID:          utils.GenerateID(),
		GroupID:     groupID,
		ContentID:   req.ContentID,
		ContentType: req.ContentType,
		AddedBy:     userID,
		Status:      domain.GroupShowcaseStatusActive,
		SortOrder:   req.SortOrder,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := s.repo.AddGroupShowcase(ctx, showcase); err != nil {
		s.logger.Error("failed to add showcase", zap.Error(err))
		return nil, errors.New("failed to add showcase")
	}

	// 5. 加载详细信息
	showcase.Group = group
	switch showcase.ContentType {
	case domain.GroupShowcaseTypeFragment:
		fragment, _ := s.repo.FragmentByID(ctx, showcase.ContentID)
		showcase.Fragment = fragment
	case domain.GroupShowcaseTypeStory:
		story, _ := s.repo.StoryByID(ctx, showcase.ContentID)
		showcase.Story = story
	}

	s.logger.Info("showcase added successfully", zap.String("showcaseID", showcase.ID))
	return showcase, nil
}

// RemoveShowcase 移除小组展示内容
func (s *GroupShowcaseService) RemoveShowcase(ctx context.Context, showcaseID, userID string) error {
	s.logger.Info("removing showcase", zap.String("showcaseID", showcaseID), zap.String("userID", userID))

	// 1. 获取展示信息
	showcase, err := s.repo.GetGroupShowcaseByID(ctx, showcaseID)
	if err != nil {
		return errors.New("showcase not found")
	}

	// 2. 检查权限（组长、管理员、模组，或添加者本人）
	role, err := s.repo.GetMemberRole(ctx, showcase.GroupID, userID)
	if err != nil {
		return errors.New("not a group member")
	}

	hasPermission := role == domain.RoleOwner || role == domain.RoleAdmin ||
		role == domain.RoleModerator || showcase.AddedBy == userID

	if !hasPermission {
		return errors.New("permission denied")
	}

	if err := s.repo.RemoveGroupShowcase(ctx, showcaseID); err != nil {
		s.logger.Error("failed to remove showcase", zap.Error(err))
		return errors.New("failed to remove showcase")
	}

	s.logger.Info("showcase removed successfully", zap.String("showcaseID", showcaseID))
	return nil
}

// GetGroupShowcases 获取小组展示列表
func (s *GroupShowcaseService) GetGroupShowcases(ctx context.Context, groupID string, contentType domain.GroupShowcaseRelationType, limit, offset int) (*domain.ListGroupShowcasesResponse, error) {
	s.logger.Debug("getting group showcases", zap.String("groupID", groupID), zap.String("type", string(contentType)))

	// 检查小组是否存在
	if _, err := s.repo.GroupByID(ctx, groupID); err != nil {
		return nil, errors.New("group not found")
	}

	showcases, total, err := s.repo.GetGroupShowcases(ctx, groupID, contentType, limit, offset)
	if err != nil {
		s.logger.Error("failed to get showcases", zap.Error(err))
		return nil, errors.New("failed to get showcases")
	}

	// 加载详细信息
	for _, showcase := range showcases {
		switch showcase.ContentType {
		case domain.GroupShowcaseTypeFragment:
			fragment, err := s.repo.FragmentByID(ctx, showcase.ContentID)
			if err == nil {
				showcase.Fragment = fragment
			}
		case domain.GroupShowcaseTypeStory:
			story, err := s.repo.StoryByID(ctx, showcase.ContentID)
			if err == nil {
				showcase.Story = story
			}
		}
	}

	return &domain.ListGroupShowcasesResponse{
		Showcases: showcases,
		Total:     int(total),
	}, nil
}

// UpdateShowcaseOrder 更新展示排序
func (s *GroupShowcaseService) UpdateShowcaseOrder(ctx context.Context, showcaseID, userID string, sortOrder int) error {
	s.logger.Info("updating showcase order", zap.String("showcaseID", showcaseID), zap.Int("sortOrder", sortOrder))

	// 1. 获取展示信息
	showcase, err := s.repo.GetGroupShowcaseByID(ctx, showcaseID)
	if err != nil {
		return errors.New("showcase not found")
	}

	// 2. 检查权限（组长、管理员、模组）
	role, err := s.repo.GetMemberRole(ctx, showcase.GroupID, userID)
	if err != nil {
		return errors.New("not a group member")
	}

	hasPermission := role == domain.RoleOwner || role == domain.RoleAdmin ||
		role == domain.RoleModerator

	if !hasPermission {
		return errors.New("permission denied")
	}

	if err := s.repo.UpdateGroupShowcaseOrder(ctx, showcaseID, sortOrder); err != nil {
		s.logger.Error("failed to update showcase order", zap.Error(err))
		return errors.New("failed to update showcase order")
	}

	return nil
}
