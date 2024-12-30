package pay

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// mockVIPStore 模拟存储实现
type mockVIPStore struct {
	vipInfos map[int64]*VIPInfo
	orders   map[string]*VIPOrder
}

func newMockStore() *mockVIPStore {
	return &mockVIPStore{
		vipInfos: make(map[int64]*VIPInfo),
		orders:   make(map[string]*VIPOrder),
	}
}

func (m *mockVIPStore) GetVIPInfo(userID int64) (*VIPInfo, error) {
	if info, ok := m.vipInfos[userID]; ok {
		return info, nil
	}
	return nil, ErrUserNotFound
}

func (m *mockVIPStore) CreateVIPInfo(info *VIPInfo) error {
	m.vipInfos[info.UserID] = info
	return nil
}

func (m *mockVIPStore) UpdateVIPInfo(info *VIPInfo) error {
	if _, ok := m.vipInfos[info.UserID]; !ok {
		return ErrUserNotFound
	}
	m.vipInfos[info.UserID] = info
	return nil
}

func (m *mockVIPStore) CreateOrder(order *VIPOrder) (*VIPOrder, error) {
	m.orders[order.OrderID] = order
	return order, nil
}

func (m *mockVIPStore) GetOrder(orderID string) (*VIPOrder, error) {
	if order, ok := m.orders[orderID]; ok {
		return order, nil
	}
	return nil, ErrOrderNotFound
}

func (m *mockVIPStore) UpdateOrder(order *VIPOrder) error {
	if _, ok := m.orders[order.OrderID]; !ok {
		return ErrOrderNotFound
	}
	m.orders[order.OrderID] = order
	return nil
}

// 测试用例
func TestVIPService(t *testing.T) {
	store := newMockStore()
	service := NewVIPService(store, DefaultVIPConfig)

	// 准备测试数据
	userID := int64(1)
	vipInfo := &VIPInfo{
		UserID:     userID,
		Level:      VIPLevelBasic,
		Status:     VIPStatusActive,
		ExpireTime: time.Now().Add(30 * 24 * time.Hour),
		QuotaLimit: 1000,
		QuotaUsed:  0,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
	store.CreateVIPInfo(vipInfo)

	t.Run("GetVIPInfo", func(t *testing.T) {
		info, err := service.GetVIPInfo(userID)
		assert.NoError(t, err)
		assert.Equal(t, VIPLevelBasic, info.Level)
	})

	t.Run("UpdateVIPStatus", func(t *testing.T) {
		err := service.UpdateVIPStatus(userID, VIPStatusExpired)
		assert.NoError(t, err)

		info, _ := service.GetVIPInfo(userID)
		assert.Equal(t, VIPStatusExpired, info.Status)
	})

	t.Run("UpdateAutoRenew", func(t *testing.T) {
		err := service.UpdateAutoRenew(userID, true)
		assert.NoError(t, err)

		info, _ := service.GetVIPInfo(userID)
		assert.True(t, info.AutoRenew)
	})

	t.Run("ConsumeQuota", func(t *testing.T) {
		// 测试正常消费
		err := service.ConsumeQuota(userID, 100)
		assert.NoError(t, err)

		used, limit, err := service.GetQuota(userID)
		assert.NoError(t, err)
		assert.Equal(t, 100, used)
		assert.Equal(t, 1000, limit)

		// 测试超额消费
		err = service.ConsumeQuota(userID, 1000)
		assert.Equal(t, ErrQuotaExceeded, err)
	})

	t.Run("CreateAndUpdateOrder", func(t *testing.T) {
		// 创建订单
		order, err := service.CreateOrder(userID, VIPLevelPro)
		assert.NoError(t, err)
		assert.Equal(t, OrderStatusPending, order.Status)

		// 更新订单状态
		err = service.UpdateOrderStatus(order.OrderID, OrderStatusPaid)
		assert.NoError(t, err)

		updatedOrder, _ := service.GetOrder(order.OrderID)
		assert.Equal(t, OrderStatusPaid, updatedOrder.Status)
		assert.NotNil(t, updatedOrder.PaymentTime)
	})

	t.Run("PermissionChecks", func(t *testing.T) {
		// 测试AI使用权限
		canUseAI := service.CanUseAI(userID)
		assert.True(t, canUseAI)

		// 测试最大角色数
		maxRoles := service.GetMaxRoles(userID)
		assert.Equal(t, DefaultVIPConfig.BasicVIP.MaxRoles, maxRoles)

		// 测试最大上下文数
		maxContexts := service.GetMaxContexts(userID)
		assert.Equal(t, DefaultVIPConfig.BasicVIP.MaxContexts, maxContexts)

		// 测试可用模型
		models := service.GetAvailableModels(userID)
		assert.Equal(t, DefaultVIPConfig.BasicVIP.Models, models)
	})

	t.Run("NonExistentUser", func(t *testing.T) {
		nonExistentUserID := int64(999)

		// 测试获取不存在用户的信息
		_, err := service.GetVIPInfo(nonExistentUserID)
		assert.Equal(t, ErrUserNotFound, err)

		// 测试更新不存在用户的状态
		err = service.UpdateVIPStatus(nonExistentUserID, VIPStatusActive)
		assert.Equal(t, ErrUserNotFound, err)

		// 测试不存在用户的AI使用权限
		canUseAI := service.CanUseAI(nonExistentUserID)
		assert.False(t, canUseAI)
	})
}
