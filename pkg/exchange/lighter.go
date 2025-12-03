package exchange

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
)

const (
	LighterMainnetBaseURL = "https://mainnet.zklighter.elliot.ai"
	LighterTestnetBaseURL = "https://testnet.zklighter.elliot.ai"
)

type Lighter struct {
	client     *http.Client
	apiKey     string
	privateKey string
	baseURL    string
	testnet    bool
}

func NewLighter(apiKey, privateKey string, testnet bool) *Lighter {
	baseURL := LighterMainnetBaseURL
	if testnet {
		baseURL = LighterTestnetBaseURL
	}
	return &Lighter{
		client:     &http.Client{},
		apiKey:     apiKey,
		privateKey: privateKey,
		baseURL:    baseURL,
		testnet:    testnet,
	}
}

func (l *Lighter) Name() string {
	return "Lighter"
}

func (l *Lighter) SetTestnet(testnet bool) {
	l.testnet = testnet
	if testnet {
		l.baseURL = LighterTestnetBaseURL
	} else {
		l.baseURL = LighterMainnetBaseURL
	}
}

func (l *Lighter) GetFundingRates() ([]*FundingRate, error) {
	// The provided Lighter documentation does not have a specific endpoint for funding rates.
	// This is a placeholder. You would need to find the correct endpoint or method
	// to get this data. For now, it will return an error.
	return nil, errors.New("funding rate endpoint not available in Lighter documentation")
}

func (l *Lighter) GetOrderbook(market string) (map[string]interface{}, error) {
	// The documentation mentions OrderApi's order_book_details but doesn't provide a clear REST endpoint.
	// This is a placeholder.
	url := fmt.Sprintf("%s/order_book_details?market=%s", l.baseURL, market)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := l.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get orderbook: %s", resp.Status)
	}

	var orderbook map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&orderbook); err != nil {
		return nil, err
	}
	return orderbook, nil
}

func (l *Lighter) PlaceOrder(market string, side OrderSide, orderType OrderType, amount, price float64) (*Order, error) {
	// This requires a signed transaction using the SignerClient which is not fully documented for a Go implementation.
	// This is a placeholder for where the order placement logic would go.
	// We will simulate a successful order placement.
	fmt.Printf("Simulating placing order on Lighter: %s %s %f @ %f\n", side, market, amount, price)
	return &Order{
		ID:        "lighter-simulated-order-id",
		Market:    market,
		Side:      side,
		Type:      orderType,
		Price:     price,
		Amount:    amount,
		Status:    "NEW",
		Timestamp: 0, // Should be actual timestamp
	}, nil
}

func (l *Lighter) GetOrderStatus(orderID string, market string) (*Order, error) {
	// Placeholder. The documentation doesn't provide a clear REST endpoint to get order status by ID.
	return nil, errors.New("get order status endpoint not available in Lighter documentation")
}

func (l *Lighter) CancelOrder(orderID string, market string) error {
	// Placeholder. This would also require a signed transaction.
	fmt.Printf("Simulating cancelling order on Lighter: %s\n", orderID)
	return nil
}

func (l *Lighter) GetBalance(asset string) (float64, error) {
	// Placeholder. The documentation mentions AccountApi but no clear REST endpoint.
	return 0, errors.New("get balance endpoint not available in Lighter documentation")
}

func (l *Lighter) sendRequest(method, endpoint string, data []byte) ([]byte, error) {
	url := l.baseURL + endpoint
	req, err := http.NewRequest(method, url, bytes.NewBuffer(data))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	// Authentication headers would go here if specified in the API docs.

	resp, err := l.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("API error: %s - %s", resp.Status, string(body))
	}

	return body, nil
}

func (l *Lighter) ClosePosition(market string, side OrderSide, amount float64) (*Order, error) {
	// To close a position, we place an order on the opposite side.
	closeSide := Sell
	if side == Sell {
		closeSide = Buy
	}

	fmt.Printf("Simulating closing %s position on Lighter for %s\n", side, market)
	// Using a market order to close, so price is irrelevant (can be 0).
	return l.PlaceOrder(market, closeSide, Market, amount, 0)
}
