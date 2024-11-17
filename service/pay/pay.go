package pay

type Order struct {
	Id        string  `json:"id"`
	Amount    float64 `json:"amount"`
	Status    string  `json:"status"`
	CreatedAt int64   `json:"created_at"`
	UpdatedAt int64   `json:"updated_at"`
}

type PayService interface {
	// 支付服务接口
	Pay(userId, orderId string, amount float64) error
	// 退款服务接口
	Refund(userId, orderId string, amount float64) error
	// 获取用户余额
	GetBalance(userId string) (float64, error)
	// 获取用户订单
	GetOrder(userId, orderId string) (Order, error)
	// 获取用户所有订单
	GetOrders(userId string) ([]Order, error)
	// 判断用户是否为vip
	IsVip(userId string) (bool, error)
	// 设置用户保证金
	SetBond(userId string, bond float64, storyId int64) error
}

var _ PayService = &PayServiceImpl{}

type PayServiceImpl struct{}

func (s *PayServiceImpl) Pay(userId, orderId string, amount float64) error {
	return nil
}
func (s *PayServiceImpl) Refund(userId, orderId string, amount float64) error {
	return nil
}
func (s *PayServiceImpl) GetBalance(userId string) (float64, error) {
	return 0, nil
}
func (s *PayServiceImpl) GetOrder(userId, orderId string) (Order, error) {
	return Order{}, nil
}
func (s *PayServiceImpl) GetOrders(userId string) ([]Order, error) {
	return []Order{}, nil
}
func (s *PayServiceImpl) IsVip(userId string) (bool, error) {
	return false, nil
}
func (s *PayServiceImpl) SetBond(userId string, bond float64, storyId int64) error {
	return nil
}
