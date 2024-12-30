package pay

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

var (
	ErrUserNotFound    = errors.New("user not found")
	ErrOrderNotFound   = errors.New("order not found")
	ErrQuotaExceeded   = errors.New("quota exceeded")
	ErrInvalidStatus   = errors.New("invalid status")
	ErrInsufficientVIP = errors.New("insufficient vip level")
)

// vipServiceImpl 实现 VIPService 接口
type vipServiceImpl struct {
	config VIPConfig
	store  VIPStore // 数据存储接口
}

// NewVIPService 创建VIP服务实例
func NewVIPService(store VIPStore, config VIPConfig) VIPService {
	return &vipServiceImpl{
		config: config,
		store:  store,
	}
}

// GetVIPInfo 获取会员信息
func (s *vipServiceImpl) GetVIPInfo(userID int64) (*VIPInfo, error) {
	return s.store.GetVIPInfo(userID)
}

// UpdateVIPStatus 更新会员状态
func (s *vipServiceImpl) UpdateVIPStatus(userID int64, status VIPStatus) error {
	info, err := s.store.GetVIPInfo(userID)
	if err != nil {
		return err
	}

	info.Status = status
	info.UpdatedAt = time.Now()
	return s.store.UpdateVIPInfo(info)
}

// UpdateAutoRenew 更新自动续费状态
func (s *vipServiceImpl) UpdateAutoRenew(userID int64, autoRenew bool) error {
	info, err := s.store.GetVIPInfo(userID)
	if err != nil {
		return err
	}

	info.AutoRenew = autoRenew
	info.UpdatedAt = time.Now()
	return s.store.UpdateVIPInfo(info)
}

// GetQuota 获取用户额度信息
func (s *vipServiceImpl) GetQuota(userID int64) (used int, limit int, err error) {
	info, err := s.store.GetVIPInfo(userID)
	if err != nil {
		return 0, 0, err
	}
	return info.QuotaUsed, info.QuotaLimit, nil
}

// ConsumeQuota 消费用户额度
func (s *vipServiceImpl) ConsumeQuota(userID int64, amount int) error {
	info, err := s.store.GetVIPInfo(userID)
	if err != nil {
		return err
	}

	if info.QuotaUsed+amount > info.QuotaLimit {
		return ErrQuotaExceeded
	}

	info.QuotaUsed += amount
	info.UpdatedAt = time.Now()
	return s.store.UpdateVIPInfo(info)
}

// CreateOrder 创建会员订单
func (s *vipServiceImpl) CreateOrder(userID int64, level VIPLevel) (*VIPOrder, error) {
	order := &VIPOrder{
		OrderID:   uuid.New().String(), // 需要实现此函数
		UserID:    userID,
		Level:     level,
		Status:    OrderStatusPending,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	return s.store.CreateOrder(order)
}

// GetOrder 获取订单信息
func (s *vipServiceImpl) GetOrder(orderID string) (*VIPOrder, error) {
	return s.store.GetOrder(orderID)
}

// UpdateOrderStatus 更新订单状态
func (s *vipServiceImpl) UpdateOrderStatus(orderID string, status VIPOrderStatus) error {
	order, err := s.store.GetOrder(orderID)
	if err != nil {
		return err
	}

	order.Status = status
	if status == OrderStatusPaid {
		now := time.Now()
		order.PaymentTime = &now
	}
	order.UpdatedAt = time.Now()

	return s.store.UpdateOrder(order)
}

// CanUseAI 检查是否可以使用AI
func (s *vipServiceImpl) CanUseAI(userID int64) bool {
	info, err := s.store.GetVIPInfo(userID)
	if err != nil {
		return false
	}
	return info.Status == VIPStatusActive && info.Level > VIPLevelNone
}

// GetMaxRoles 获取最大可创建角色数
func (s *vipServiceImpl) GetMaxRoles(userID int64) int {
	info, err := s.store.GetVIPInfo(userID)
	if err != nil {
		return s.config.BasicVIP.MaxRoles
	}

	switch info.Level {
	case VIPLevelPro:
		return s.config.ProVIP.MaxRoles
	case VIPLevelBasic:
		return s.config.BasicVIP.MaxRoles
	default:
		return s.config.BasicVIP.MaxRoles
	}
}

// GetMaxContexts 获取最大可创建上下文数
func (s *vipServiceImpl) GetMaxContexts(userID int64) int {
	info, err := s.store.GetVIPInfo(userID)
	if err != nil {
		return s.config.BasicVIP.MaxContexts
	}

	switch info.Level {
	case VIPLevelPro:
		return s.config.ProVIP.MaxContexts
	case VIPLevelBasic:
		return s.config.BasicVIP.MaxContexts
	default:
		return s.config.BasicVIP.MaxContexts
	}
}

// GetAvailableModels 获取可用的模型列表
func (s *vipServiceImpl) GetAvailableModels(userID int64) []string {
	info, err := s.store.GetVIPInfo(userID)
	if err != nil {
		return s.config.BasicVIP.Models
	}

	switch info.Level {
	case VIPLevelPro:
		return s.config.ProVIP.Models
	case VIPLevelBasic:
		return s.config.BasicVIP.Models
	default:
		return s.config.BasicVIP.Models
	}
}
