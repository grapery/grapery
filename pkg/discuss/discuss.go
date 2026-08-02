package discuss

import (
	"context"

	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/grapery/grapery/models"
	"github.com/grapery/grapery/utils/cache"
)

var logger, _ = zap.NewDevelopment()

var server DiscussServer

func init() {
	server = NewDiscussService()
}

func GetDiscussService() DiscussServer {
	return server
}

func NewDiscussService() DiscussServer {
	return &DiscussService{
		cache: cache.GetDiscussCache(),
	}
}

// 基础请求和响应结构
type CreateDiscussRequest struct {
	UserId      int64  `json:"user_id"`
	StoryId     int64  `json:"story_id"`
	GroupId     int64  `json:"group_id"`
	Title       string `json:"title"`
	Content     string `json:"content"`
	Category    int32  `json:"category"`
	Tags        string `json:"tags"`
	Attachments string `json:"attachments"`
}

type CreateDiscussResponse struct {
	Code      int32  `json:"code"`
	Message   string `json:"message"`
	DiscussId int64  `json:"discuss_id"`
}

type GetDiscussRequest struct {
	DiscussId int64 `json:"discuss_id"`
}

type GetDiscussResponse struct {
	Code    int32    `json:"code"`
	Message string   `json:"message"`
	Discuss *Discuss `json:"discuss"`
}

type ListDiscussesRequest struct {
	StoryId  int64 `json:"story_id"`
	Category int32 `json:"category"`
	Status   int32 `json:"status"`
	Offset   int32 `json:"offset"`
	PageSize int32 `json:"page_size"`
}

type ListDiscussesResponse struct {
	Code      int32      `json:"code"`
	Message   string     `json:"message"`
	Total     int64      `json:"total"`
	Discusses []*Discuss `json:"discusses"`
}

type UpdateDiscussRequest struct {
	UserId      int64  `json:"user_id"`
	DiscussId   int64  `json:"discuss_id"`
	Title       string `json:"title"`
	Content     string `json:"content"`
	Category    int32  `json:"category"`
	Tags        string `json:"tags"`
	Attachments string `json:"attachments"`
}

type UpdateDiscussResponse struct {
	Code    int32  `json:"code"`
	Message string `json:"message"`
}

type DeleteDiscussRequest struct {
	UserId    int64 `json:"user_id"`
	DiscussId int64 `json:"discuss_id"`
}

type DeleteDiscussResponse struct {
	Code    int32  `json:"code"`
	Message string `json:"message"`
}

type Discuss struct {
	Id          int64  `json:"id"`
	Creator     int64  `json:"creator"`
	StoryId     int64  `json:"story_id"`
	GroupId     int64  `json:"group_id"`
	Title       string `json:"title"`
	Content     string `json:"content"`
	Category    int32  `json:"category"`
	Status      int32  `json:"status"`
	IsPinned    bool   `json:"is_pinned"`
	IsLocked    bool   `json:"is_locked"`
	ViewCount   int64  `json:"view_count"`
	LikeCount   int64  `json:"like_count"`
	ReplyCount  int64  `json:"reply_count"`
	LastReplyAt int64  `json:"last_reply_at"`
	LastReplyBy int64  `json:"last_reply_by"`
	Tags        string `json:"tags"`
	Attachments string `json:"attachments"`
	CreatedAt   int64  `json:"created_at"`
	UpdatedAt   int64  `json:"updated_at"`
}

type DiscussServer interface {
	// 讨论管理
	CreateDiscuss(ctx context.Context, req *CreateDiscussRequest) (*CreateDiscussResponse, error)
	GetDiscuss(ctx context.Context, req *GetDiscussRequest) (*GetDiscussResponse, error)
	UpdateDiscuss(ctx context.Context, req *UpdateDiscussRequest) (*UpdateDiscussResponse, error)
	DeleteDiscuss(ctx context.Context, req *DeleteDiscussRequest) (*DeleteDiscussResponse, error)
	ListDiscusses(ctx context.Context, req *ListDiscussesRequest) (*ListDiscussesResponse, error)
}

// DiscussService 讨论服务
type DiscussService struct {
	cache *cache.DiscussCache
}

// CreateDiscuss 创建讨论
func (s *DiscussService) CreateDiscuss(ctx context.Context, req *CreateDiscussRequest) (*CreateDiscussResponse, error) {
	logger.Info("CreateDiscuss called", zap.Any("req", req))

	// 参数校验
	if req.Title == "" {
		logger.Error("CreateDiscuss failed: title is empty")
		return &CreateDiscussResponse{
			Code:    400,
			Message: "title is empty",
		}, nil
	}
	if req.Content == "" {
		logger.Error("CreateDiscuss failed: content is empty")
		return &CreateDiscussResponse{
			Code:    400,
			Message: "content is empty",
		}, nil
	}

	// 使用事务创建讨论
	var discussID uint
	err := models.DataBase().Transaction(func(tx *gorm.DB) error {
		discuss := &models.Disscuss{
			Creator:     req.UserId,
			StoryID:     req.StoryId,
			GroupID:     req.GroupId,
			Title:       req.Title,
			Content:     req.Content,
			Category:    models.DiscussCategory(req.Category),
			Status:      models.DiscussStatusOpen,
			IsPinned:    false,
			IsLocked:    false,
			ViewCount:   0,
			LikeCount:   0,
			ReplyCount:  0,
			Tags:        req.Tags,
			Attachments: req.Attachments,
		}

		if err := tx.WithContext(ctx).Create(discuss).Error; err != nil {
			logger.Error("CreateDiscuss failed: create discuss error", zap.Error(err))
			return err
		}
		discussID = discuss.ID
		return nil
	})

	if err != nil {
		logger.Error("CreateDiscuss failed: transaction error", zap.Error(err))
		return &CreateDiscussResponse{
			Code:    500,
			Message: err.Error(),
		}, nil
	}

	logger.Info("CreateDiscuss success", zap.Uint("discuss_id", discussID), zap.Int64("user_id", req.UserId))

	// 失效相关缓存
	go func() {
		if cacheErr := s.cache.InvalidateDiscussListCache(context.Background(), req.StoryId); cacheErr != nil {
			logger.Error("Failed to invalidate discuss list cache", zap.Error(cacheErr))
		}
	}()

	return &CreateDiscussResponse{
		Code:      200,
		Message:   "success",
		DiscussId: int64(discussID),
	}, nil
}

// GetDiscuss 获取讨论详情
func (s *DiscussService) GetDiscuss(ctx context.Context, req *GetDiscussRequest) (*GetDiscussResponse, error) {
	logger.Info("GetDiscuss called", zap.Int64("discuss_id", req.DiscussId))

	// 从数据库获取
	discuss, err := models.GetDiscussByID(ctx, req.DiscussId)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return &GetDiscussResponse{
				Code:    404,
				Message: "discuss not found",
			}, nil
		}
		logger.Error("GetDiscuss failed: get discuss error", zap.Error(err))
		return &GetDiscussResponse{
			Code:    500,
			Message: err.Error(),
		}, nil
	}

	// 增加查看次数
	go func() {
		if err := models.IncrementViewCount(context.Background(), req.DiscussId); err != nil {
			logger.Error("Failed to increment view count", zap.Error(err))
		}
	}()

	// 转换为API格式
	apiDiscuss := &Discuss{
		Id:          int64(discuss.ID),
		Creator:     discuss.Creator,
		StoryId:     discuss.StoryID,
		GroupId:     discuss.GroupID,
		Title:       discuss.Title,
		Content:     discuss.Content,
		Category:    int32(discuss.Category),
		Status:      int32(discuss.Status),
		IsPinned:    discuss.IsPinned,
		IsLocked:    discuss.IsLocked,
		ViewCount:   discuss.ViewCount,
		LikeCount:   discuss.LikeCount,
		ReplyCount:  discuss.ReplyCount,
		LastReplyAt: discuss.LastReplyAt.Unix(),
		LastReplyBy: discuss.LastReplyBy,
		Tags:        discuss.Tags,
		Attachments: discuss.Attachments,
		CreatedAt:   discuss.CreateAt.Unix(),
		UpdatedAt:   discuss.UpdateAt.Unix(),
	}

	return &GetDiscussResponse{
		Code:    200,
		Message: "success",
		Discuss: apiDiscuss,
	}, nil
}

// ListDiscusses 获取讨论列表
func (s *DiscussService) ListDiscusses(ctx context.Context, req *ListDiscussesRequest) (*ListDiscussesResponse, error) {
	logger.Info("ListDiscusses called", zap.Any("req", req))

	// 从数据库获取
	discusses, err := models.GetDiscussList(ctx, req.StoryId,
		models.DiscussCategory(req.Category),
		models.DiscussStatus(req.Status),
		int(req.Offset), int(req.PageSize))
	if err != nil {
		logger.Error("ListDiscusses failed: get discuss list error", zap.Error(err))
		return &ListDiscussesResponse{
			Code:      500,
			Message:   err.Error(),
			Total:     0,
			Discusses: []*Discuss{},
		}, nil
	}

	// 转换为API格式
	apiDiscusses := make([]*Discuss, 0, len(discusses))
	for _, discuss := range discusses {
		apiDiscusses = append(apiDiscusses, &Discuss{
			Id:          int64(discuss.ID),
			Creator:     discuss.Creator,
			StoryId:     discuss.StoryID,
			GroupId:     discuss.GroupID,
			Title:       discuss.Title,
			Content:     discuss.Content,
			Category:    int32(discuss.Category),
			Status:      int32(discuss.Status),
			IsPinned:    discuss.IsPinned,
			IsLocked:    discuss.IsLocked,
			ViewCount:   discuss.ViewCount,
			LikeCount:   discuss.LikeCount,
			ReplyCount:  discuss.ReplyCount,
			LastReplyAt: discuss.LastReplyAt.Unix(),
			LastReplyBy: discuss.LastReplyBy,
			Tags:        discuss.Tags,
			Attachments: discuss.Attachments,
			CreatedAt:   discuss.CreateAt.Unix(),
			UpdatedAt:   discuss.UpdateAt.Unix(),
		})
	}

	return &ListDiscussesResponse{
		Code:      200,
		Message:   "success",
		Total:     int64(len(discusses)),
		Discusses: apiDiscusses,
	}, nil
}

// UpdateDiscuss 更新讨论
func (s *DiscussService) UpdateDiscuss(ctx context.Context, req *UpdateDiscussRequest) (*UpdateDiscussResponse, error) {
	logger.Info("UpdateDiscuss called", zap.Int64("discuss_id", req.DiscussId))

	// 权限验证：只有创建者可以更新讨论
	if !s.hasPermissionToUpdate(ctx, req.UserId, req.DiscussId) {
		logger.Warn("UpdateDiscuss failed: permission denied",
			zap.Int64("user_id", req.UserId),
			zap.Int64("discuss_id", req.DiscussId))
		return &UpdateDiscussResponse{
			Code:    403,
			Message: "no permission to update discuss",
		}, nil
	}

	// 使用事务更新讨论
	err := models.DataBase().Transaction(func(tx *gorm.DB) error {
		updates := map[string]interface{}{}

		if req.Title != "" {
			updates["title"] = req.Title
		}
		if req.Content != "" {
			updates["content"] = req.Content
		}
		if req.Category > 0 {
			updates["category"] = req.Category
		}
		if req.Tags != "" {
			updates["tags"] = req.Tags
		}
		if req.Attachments != "" {
			updates["attachments"] = req.Attachments
		}

		if len(updates) == 0 {
			return nil // 没有需要更新的字段
		}

		if err := tx.Model(&models.Disscuss{}).
			WithContext(ctx).
			Where("id = ?", req.DiscussId).
			Updates(updates).Error; err != nil {
			logger.Error("UpdateDiscuss failed: update discuss error", zap.Error(err))
			return err
		}

		return nil
	})

	if err != nil {
		logger.Error("UpdateDiscuss failed: transaction error", zap.Error(err))
		return &UpdateDiscussResponse{
			Code:    500,
			Message: err.Error(),
		}, nil
	}

	// 失效相关缓存
	go func() {
		if cacheErr := s.cache.InvalidateDiscussCache(context.Background(), req.DiscussId); cacheErr != nil {
			logger.Error("Failed to invalidate discuss cache", zap.Error(cacheErr))
		}
	}()

	return &UpdateDiscussResponse{
		Code:    200,
		Message: "success",
	}, nil
}

// DeleteDiscuss 删除讨论
func (s *DiscussService) DeleteDiscuss(ctx context.Context, req *DeleteDiscussRequest) (*DeleteDiscussResponse, error) {
	logger.Info("DeleteDiscuss called", zap.Int64("discuss_id", req.DiscussId))

	// 权限验证：只有创建者可以删除讨论
	if !s.hasPermissionToDelete(ctx, req.UserId, req.DiscussId) {
		logger.Warn("DeleteDiscuss failed: permission denied",
			zap.Int64("user_id", req.UserId),
			zap.Int64("discuss_id", req.DiscussId))
		return &DeleteDiscussResponse{
			Code:    403,
			Message: "no permission to delete discuss",
		}, nil
	}

	// 使用事务删除讨论
	var storyID int64
	err := models.DataBase().Transaction(func(tx *gorm.DB) error {
		// 先获取讨论信息
		var discuss models.Disscuss
		if err := tx.WithContext(ctx).Where("id = ?", req.DiscussId).First(&discuss).Error; err != nil {
			logger.Error("DeleteDiscuss failed: get discuss error", zap.Error(err))
			return err
		}
		storyID = discuss.StoryID

		// 软删除讨论
		if err := tx.WithContext(ctx).Model(&models.Disscuss{}).
			Where("id = ?", req.DiscussId).
			Update("deleted", 1).Error; err != nil {
			logger.Error("DeleteDiscuss failed: delete discuss error", zap.Error(err))
			return err
		}

		return nil
	})

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return &DeleteDiscussResponse{
				Code:    404,
				Message: "discuss not found",
			}, nil
		}
		return &DeleteDiscussResponse{
			Code:    500,
			Message: err.Error(),
		}, nil
	}

	// 失效相关缓存
	go func() {
		if cacheErr := s.cache.InvalidateDiscussCache(context.Background(), req.DiscussId); cacheErr != nil {
			logger.Error("Failed to invalidate discuss cache", zap.Error(cacheErr))
		}
		if cacheErr := s.cache.InvalidateDiscussListCache(context.Background(), storyID); cacheErr != nil {
			logger.Error("Failed to invalidate discuss list cache", zap.Error(cacheErr))
		}
	}()

	return &DeleteDiscussResponse{
		Code:    200,
		Message: "success",
	}, nil
}

// hasPermissionToUpdate 检查用户是否有权限更新讨论
func (s *DiscussService) hasPermissionToUpdate(ctx context.Context, userID int64, discussID int64) bool {
	var discuss models.Disscuss
	if err := models.DataBase().WithContext(ctx).Where("id = ?", discussID).First(&discuss).Error; err != nil {
		logger.Error("failed to get discuss for permission check", zap.Error(err))
		return false
	}

	// 只有讨论创建者可以更新
	return discuss.Creator == userID
}

// hasPermissionToDelete 检查用户是否有权限删除讨论
func (s *DiscussService) hasPermissionToDelete(ctx context.Context, userID int64, discussID int64) bool {
	var discuss models.Disscuss
	if err := models.DataBase().WithContext(ctx).Where("id = ?", discussID).First(&discuss).Error; err != nil {
		logger.Error("failed to get discuss for permission check", zap.Error(err))
		return false
	}

	// 只有讨论创建者可以删除
	return discuss.Creator == userID
}
