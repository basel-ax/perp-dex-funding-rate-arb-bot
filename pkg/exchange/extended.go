package exchange

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"time"
)

const (
	ExtendedMainnetBaseURL = "https://api.starknet.extended.exchange"
	ExtendedTestnetBaseURL = "https://api.starknet.sepolia.extended.exchange"
)

// Extended is the implementation for the Extended exchange
type Extended struct {
	client  *http.Client
	apiKey  string
	baseURL string
	testnet bool
}

// NewExtended creates a new Extended exchange client
func NewExtended(apiKey string, testnet bool) *Extended {
	baseURL := ExtendedMainnetBaseURL
	if testnet {
		baseURL = ExtendedTestnetBaseURL
	}
	return &Extended{
		client:  &http.Client{Timeout: 10 * time.Second},
		apiKey:  apiKey,
		baseURL: baseURL,
		testnet: testnet,
	}
}

// Name returns the name of the exchange
func (e *Extended) Name() string {
	return "Extended"
}

// SetTestnet switches between testnet and mainnet
func (e *Extended) SetTestnet(testnet bool) {
	e.testnet = testnet
	if testnet {
		e.baseURL = ExtendedTestnetBaseURL
	} else {
		e.baseURL = ExtendedMainnetBaseURL
	}
}

// ExtendedMarketStats represents the statistics for a market on Extended
type ExtendedMarketStats struct {
	FundingRate string `json:"fundingRate"`
}

// ExtendedMarket represents a single market on Extended
type ExtendedMarket struct {
	Name        string              `json:"name"`
	MarketStats ExtendedMarketStats `json:"marketStats"`
}

// ExtendedMarketsResponse is the response structure for the markets endpoint
type ExtendedMarketsResponse struct {
	Status string           `json:"status"`
	Data   []ExtendedMarket `json:"data"`
}

// GetFundingRates fetches funding rates for all markets
func (e *Extended) GetFundingRates() ([]*FundingRate, error) {
	endpoint := "/api/v1/info/markets"
	body, err := e.sendRequest("GET", endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get markets from Extended: %w", err)
	}

	var response ExtendedMarketsResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal markets response from Extended: %w", err)
	}

	if response.Status != "ok" {
		return nil, fmt.Errorf("Extended API returned non-ok status: %s", string(body))
	}

	var fundingRates []*FundingRate
	for _, market := range response.Data {
		rate, err := strconv.ParseFloat(market.MarketStats.FundingRate, 64)
		if err != nil {
			// Skip markets where funding rate is not a valid float
			continue
		}
		fundingRates = append(fundingRates, &FundingRate{
			Market: market.Name,
			Rate:   rate,
			// NextTime is not directly in this response, would require another call or parsing from marketStats
		})
	}

	return fundingRates, nil
}

// GetOrderbook is a placeholder
func (e *Extended) GetOrderbook(market string) (map[string]interface{}, error) {
	return nil, fmt.Errorf("GetOrderbook not implemented for Extended")
}

// PlaceOrder simulates placing an order on Extended
func (e *Extended) PlaceOrder(market string, side OrderSide, orderType OrderType, amount, price float64) (*Order, error) {
	// Placing an order on Extended requires a complex signed message (Starknet signature).
	// This is a placeholder to simulate the action.
	// A real implementation would require a Starknet signing library.
	fmt.Printf("Simulating placing order on Extended: %s %s %f @ %f\n", side, market, amount, price)

	// Return a simulated order confirmation
	return &Order{
		ID:        fmt.Sprintf("extended-simulated-%d", time.Now().UnixNano()),
		Market:    market,
		Side:      side,
		Type:      orderType,
		Price:     price,
		Amount:    amount,
		Status:    "NEW",
		Timestamp: time.Now().Unix(),
	}, nil
}

// GetOrderStatus is a placeholder
func (e *Extended) GetOrderStatus(orderID string, market string) (*Order, error) {
	return nil, fmt.Errorf("GetOrderStatus not implemented for Extended")
}

// CancelOrder is a placeholder
func (e *Extended) CancelOrder(orderID string, market string) error {
	fmt.Printf("Simulating cancelling order on Extended: %s\n", orderID)
	return nil
}

// ExtendedBalanceData represents the balance data from Extended
type ExtendedBalanceData struct {
	Balance string `json:"balance"`
}

// ExtendedBalanceResponse is the response structure for the balance endpoint
type ExtendedBalanceResponse struct {
	Status string              `json:"status"`
	Data   ExtendedBalanceData `json:"data"`
}

// GetBalance fetches the balance for a specific asset
func (e *Extended) GetBalance(asset string) (float64, error) {
	endpoint := "/api/v1/user/balance"
	body, err := e.sendRequest("GET", endpoint, nil)
	if err != nil {
		return 0, fmt.Errorf("failed to get balance from Extended: %w", err)
	}

	var response ExtendedBalanceResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return 0, fmt.Errorf("failed to unmarshal balance response from Extended: %w", err)
	}
	if response.Status != "OK" {
		return 0, fmt.Errorf("Extended API returned non-OK status for balance: %s", string(body))
	}

	balance, err := strconv.ParseFloat(response.Data.Balance, 64)
	if err != nil {
		return 0, fmt.Errorf("failed to parse balance float from Extended: %w", err)
	}

	return balance, nil
}

// sendRequest is a helper function to make HTTP requests to the Extended API
func (e *Extended) sendRequest(method, endpoint string, data []byte) ([]byte, error) {
	url := e.baseURL + endpoint
	req, err := http.NewRequest(method, url, bytes.NewBuffer(data))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Api-Key", e.apiKey)
	req.Header.Set("User-Agent", "FundingRateArbBot/1.0")

	resp, err := e.client.Do(req)
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

func (e *Extended) ClosePosition(market string, side OrderSide, amount float64) (*Order, error) {
	// To close a position, we place an order on the opposite side.
	closeSide := Sell
	if side == Sell {
		closeSide = Buy
	}

	fmt.Printf("Simulating closing %s position on Extended for %s\n", side, market)
	// Using a market order to close, so price is irrelevant (can be 0).
	return e.PlaceOrder(market, closeSide, Market, amount, 0)
}
