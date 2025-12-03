package exchange

type OrderSide string

const (
	Buy  OrderSide = "BUY"
	Sell OrderSide = "SELL"
)

type OrderType string

const (
	Limit  OrderType = "LIMIT"
	Market OrderType = "MARKET"
)

type Order struct {
	ID        string
	Market    string
	Side      OrderSide
	Type      OrderType
	Price     float64
	Amount    float64
	Filled    float64
	Status    string
	Timestamp int64
}

type FundingRate struct {
	Market   string
	Rate     float64
	NextTime int64
}

type Exchange interface {
	Name() string
	SetTestnet(testnet bool)
	GetFundingRates() ([]*FundingRate, error)
	GetOrderbook(market string) (map[string]interface{}, error)
	PlaceOrder(market string, side OrderSide, orderType OrderType, amount, price float64) (*Order, error)
	GetOrderStatus(orderID string, market string) (*Order, error)
	CancelOrder(orderID string, market string) error
	GetBalance(asset string) (float64, error)
	ClosePosition(market string, side OrderSide, amount float64) (*Order, error)
}
