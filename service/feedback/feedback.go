package feedback

import (
	"context"
	"errors"
	"time"

	"github.com/grapery/grapery/models"
	log "github.com/sirupsen/logrus"
)

// 自定义错误类型
var (
	ErrBadRequest = errors.New("bad request")
	ErrNotFound   = errors.New("not found")
	ErrForbidden  = errors.New("forbidden")
	ErrInternal   = errors.New("internal server error")
)

// 错误类型检查函数
func IsBadRequestError(err error) bool {
	return errors.Is(err, ErrBadRequest)
}

func IsNotFoundError(err error) bool {
	return errors.Is(err, ErrNotFound)
}

func IsForbiddenError(err error) bool {
	return errors.Is(err, ErrForbidden)
}

// FeedbackService 用户反馈服务
type FeedbackService struct{}

// NewFeedbackService 创建反馈服务实例
func NewFeedbackService() *FeedbackService {
	return &FeedbackService{}
}

// CreateFeedbackRequest 创建反馈请求结构体
type CreateFeedbackRequest struct {
	FeedbackType models.FeedbackType `json:"feedback_type" binding:"required"` // 反馈类型
	Title        string              `json:"title" binding:"required"`         // 反馈标题
	Description  string              `json:"description" binding:"required"`   // 反馈描述
	Screenshots  string              `json:"screenshots"`                      // 截图URL列表，JSON格式
	ContactInfo  string              `json:"contact_info"`                     // 联系方式
	RelatedType  string              `json:"related_type"`                     // 关联内容类型
	RelatedID    int64               `json:"related_id"`                       // 关联内容ID
	Priority     int                 `json:"priority"`                         // 优先级 1-5
	Tags         string              `json:"tags"`                             // 标签，逗号分隔
}

// CreateFeedbackResponse 创建反馈响应结构体
type CreateFeedbackResponse struct {
	ID           uint                  `json:"id"`            // 反馈ID
	FeedbackType models.FeedbackType   `json:"feedback_type"` // 反馈类型
	Title        string                `json:"title"`         // 反馈标题
	Description  string                `json:"description"`   // 反馈描述
	Screenshots  string                `json:"screenshots"`   // 截图URL列表
	ContactInfo  string                `json:"contact_info"`  // 联系方式
	Status       models.FeedbackStatus `json:"status"`        // 处理状态
	RelatedType  string                `json:"related_type"`  // 关联内容类型
	RelatedID    int64                 `json:"related_id"`    // 关联内容ID
	Priority     int                   `json:"priority"`      // 优先级
	Tags         string                `json:"tags"`          // 标签
	CreateAt     time.Time             `json:"create_at"`     // 创建时间
}

// FeedbackListResponse 反馈列表响应结构体
type FeedbackListResponse struct {
	Feedbacks []*FeedbackItem `json:"feedbacks"` // 反馈列表
	Total     int64           `json:"total"`     // 总数
	Offset    int             `json:"offset"`    // 偏移量
	Limit     int             `json:"limit"`     // 限制数量
}

// FeedbackItem 反馈项结构体
type FeedbackItem struct {
	ID           uint                  `json:"id"`            // 反馈ID
	FeedbackType models.FeedbackType   `json:"feedback_type"` // 反馈类型
	Title        string                `json:"title"`         // 反馈标题
	Description  string                `json:"description"`   // 反馈描述
	Screenshots  string                `json:"screenshots"`   // 截图URL列表
	ContactInfo  string                `json:"contact_info"`  // 联系方式
	Status       models.FeedbackStatus `json:"status"`        // 处理状态
	AdminReply   string                `json:"admin_reply"`   // 管理员回复
	ProcessedAt  *time.Time            `json:"processed_at"`  // 处理时间
	RelatedType  string                `json:"related_type"`  // 关联内容类型
	RelatedID    int64                 `json:"related_id"`    // 关联内容ID
	Priority     int                   `json:"priority"`      // 优先级
	Tags         string                `json:"tags"`          // 标签
	CreateAt     time.Time             `json:"create_at"`     // 创建时间
	UpdateAt     time.Time             `json:"update_at"`     // 更新时间
}

// CreateFeedback 创建用户反馈
func (s *FeedbackService) CreateFeedback(ctx context.Context, userID int64, req *CreateFeedbackRequest) (*CreateFeedbackResponse, error) {
	// 验证反馈类型
	if req.FeedbackType < 1 || req.FeedbackType > 6 {
		return nil, ErrBadRequest
	}

	// 验证优先级
	if req.Priority < 1 || req.Priority > 5 {
		req.Priority = 1 // 默认优先级为1
	}

	// 创建反馈记录
	feedback := &models.UserFeedback{
		UserID:       userID,
		FeedbackType: req.FeedbackType,
		Title:        req.Title,
		Description:  req.Description,
		Screenshots:  req.Screenshots,
		ContactInfo:  req.ContactInfo,
		Status:       models.FeedbackStatusPending, // 默认状态为待处理
		RelatedType:  req.RelatedType,
		RelatedID:    req.RelatedID,
		Priority:     req.Priority,
		Tags:         req.Tags,
	}

	// 保存到数据库
	err := feedback.Create(ctx)
	if err != nil {
		log.Errorf("创建用户反馈失败: userID=%d, error=%v", userID, err)
		return nil, ErrInternal
	}

	// 构建响应
	response := &CreateFeedbackResponse{
		ID:           feedback.ID,
		FeedbackType: feedback.FeedbackType,
		Title:        feedback.Title,
		Description:  feedback.Description,
		Screenshots:  feedback.Screenshots,
		ContactInfo:  feedback.ContactInfo,
		Status:       feedback.Status,
		RelatedType:  feedback.RelatedType,
		RelatedID:    feedback.RelatedID,
		Priority:     feedback.Priority,
		Tags:         feedback.Tags,
		CreateAt:     feedback.CreateAt,
	}

	return response, nil
}

// GetUserFeedbackList 获取用户反馈列表
func (s *FeedbackService) GetUserFeedbackList(ctx context.Context, userID int64, offset, limit int) (*FeedbackListResponse, error) {
	// 验证分页参数
	if offset < 0 {
		offset = 0
	}
	if limit <= 0 || limit > 100 {
		limit = 20 // 默认每页20条
	}

	// 获取反馈列表
	feedbacks, total, err := models.GetUserFeedbackList(ctx, userID, offset, limit)
	if err != nil {
		log.Errorf("获取用户反馈列表失败: userID=%d, error=%v", userID, err)
		return nil, ErrInternal
	}

	// 构建响应
	items := make([]*FeedbackItem, 0, len(feedbacks))
	for _, feedback := range feedbacks {
		item := &FeedbackItem{
			ID:           feedback.ID,
			FeedbackType: feedback.FeedbackType,
			Title:        feedback.Title,
			Description:  feedback.Description,
			Screenshots:  feedback.Screenshots,
			ContactInfo:  feedback.ContactInfo,
			Status:       feedback.Status,
			AdminReply:   feedback.AdminReply,
			ProcessedAt:  feedback.ProcessedAt,
			RelatedType:  feedback.RelatedType,
			RelatedID:    feedback.RelatedID,
			Priority:     feedback.Priority,
			Tags:         feedback.Tags,
			CreateAt:     feedback.CreateAt,
			UpdateAt:     feedback.UpdateAt,
		}
		items = append(items, item)
	}

	response := &FeedbackListResponse{
		Feedbacks: items,
		Total:     total,
		Offset:    offset,
		Limit:     limit,
	}

	return response, nil
}

// GetFeedbackDetail 获取反馈详情
func (s *FeedbackService) GetFeedbackDetail(ctx context.Context, feedbackID uint, userID int64) (*FeedbackItem, error) {
	// 获取反馈详情
	feedback, err := models.GetFeedbackByID(ctx, feedbackID)
	if err != nil {
		log.Errorf("获取反馈详情失败: feedbackID=%d, error=%v", feedbackID, err)
		return nil, ErrInternal
	}

	if feedback == nil {
		return nil, ErrNotFound
	}

	// 检查权限：只有反馈创建者可以查看
	if feedback.UserID != userID {
		return nil, ErrForbidden
	}

	// 构建响应
	item := &FeedbackItem{
		ID:           feedback.ID,
		FeedbackType: feedback.FeedbackType,
		Title:        feedback.Title,
		Description:  feedback.Description,
		Screenshots:  feedback.Screenshots,
		ContactInfo:  feedback.ContactInfo,
		Status:       feedback.Status,
		AdminReply:   feedback.AdminReply,
		ProcessedAt:  feedback.ProcessedAt,
		RelatedType:  feedback.RelatedType,
		RelatedID:    feedback.RelatedID,
		Priority:     feedback.Priority,
		Tags:         feedback.Tags,
		CreateAt:     feedback.CreateAt,
		UpdateAt:     feedback.UpdateAt,
	}

	return item, nil
}

// GetFeedbackTypes 获取反馈类型列表
func (s *FeedbackService) GetFeedbackTypes() map[int]string {
	return map[int]string{
		1: "不合理的内容",
		2: "有冒犯的内容",
		3: "有违公德的内容",
		4: "有损害少年的内容",
		5: "有违反法律的内容",
		6: "其他",
	}
}

// GetFeedbackStatuses 获取反馈状态列表
func (s *FeedbackService) GetFeedbackStatuses() map[int]string {
	return map[int]string{
		1: "待处理",
		2: "处理中",
		3: "已解决",
		4: "已拒绝",
	}
}
