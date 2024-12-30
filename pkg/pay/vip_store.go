package pay

// VIPStore 定义VIP会员数据存储接口
type VIPStore interface {
	// VIP信息存储
	GetVIPInfo(userID int64) (*VIPInfo, error)
	CreateVIPInfo(*VIPInfo) error
	UpdateVIPInfo(*VIPInfo) error

	// 订单存储
	CreateOrder(*VIPOrder) (*VIPOrder, error)
	GetOrder(orderID string) (*VIPOrder, error)
	UpdateOrder(*VIPOrder) error
}
