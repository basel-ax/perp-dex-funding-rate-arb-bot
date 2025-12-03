package exchange

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"time"

	sdk "github.com/extended-protocol/extended-sdk-golang/src"
	"github.com/shopspring/decimal"
)

const (
	ExtendedMainnetBaseURL = "https://api.starknet.extended.exchange"
	ExtendedTestnetBaseURL = "https://api.starknet.sepolia.extended.exchange"
)

// Extended is the implementation for the Extended exchange
type Extended struct {
	client     *sdk.APIClient
	account    *sdk.StarkPerpetualAccount
	httpClient *http.Client // for requests not in the SDK
	apiKey     string
	baseURL    string
	testnet    bool
}

// NewExtended creates a new Extended exchange client
func NewExtended(apiKey, privateKey, publicKey string, vaultID int, testnet bool) *Extended {
	baseURL := ExtendedMainnetBaseURL
	if testnet {
		baseURL = ExtendedTestnetBaseURL
	}

	cfg := sdk.EndpointConfig{
		APIBaseURL: baseURL + "/api/v1",
	}

	account, err := sdk.NewStarkPerpetualAccount(
		uint64(vaultID),
		privateKey,
		publicKey,
		apiKey,
	)
	if err != nil {
		// Using log.Fatal here because returning an error would require changing the function signature
		// across multiple files, which is a larger refactor. For this bot, exiting is acceptable.
		log.Fatalf("Failed to create extended sdk account: %v", err)
	}

	client := sdk.NewAPIClient(cfg, account.APIKey(), account, 30*time.Second)

	return &Extended{
		client:     client,
		account:    account,
		httpClient: &http.Client{Timeout: 10 * time.Second},
		apiKey:     apiKey,
		baseURL:    baseURL,
		testnet:    testnet,
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

// GetFundingRates fetches funding rates for all markets
func (e *Extended) GetFundingRates() ([]*FundingRate, error) {
	markets, err := e.client.GetMarkets(context.Background(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get markets from Extended SDK: %w", err)
	}

	var fundingRates []*FundingRate
	for _, market := range markets {
		rate, _ := market.MarketStats.FundingRate.Float64()
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

// ExtendedMarketStatsResponse is the response structure for market stats
type ExtendedMarketStatsResponse struct {
	Status string `json:"status"`
	Data   struct {
		MarkPrice string `json:"markPrice"`
	} `json:"data"`
}

// GetMarkPrice fetches the current mark price for a given market.
func (e *Extended) GetMarkPrice(market string) (float64, error) {
	endpoint := fmt.Sprintf("/api/v1/info/markets/%s/stats", market)
	body, err := e.sendRequest("GET", endpoint, nil)
	if err != nil {
		return 0, fmt.Errorf("failed to get market stats from Extended: %w", err)
	}

	var response ExtendedMarketStatsResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return 0, fmt.Errorf("failed to unmarshal market stats response from Extended: %w", err)
	}
	if response.Status != "OK" {
		return 0, fmt.Errorf("Extended API returned non-OK status for market stats: %s", string(body))
	}

	markPrice, err := strconv.ParseFloat(response.Data.MarkPrice, 64)
	if err != nil {
		return 0, fmt.Errorf("failed to parse mark price float from Extended: %w", err)
	}

	return markPrice, nil
}

func (e *Extended) getStarknetDomain() sdk.StarknetDomain {
	if e.testnet {
		return sdk.StarknetDomain{
			Name:     "Perpetuals",
			Version:  "v0",
			ChainID:  "SN_SEPOLIA",
			Revision: "1",
		}
	}
	return sdk.StarknetDomain{
		Name:     "Perpetuals",
		Version:  "v0",
		ChainID:  "SN_MAIN",
		Revision: "1",
	}
}

// PlaceOrder sends a real, signed order to the Extended exchange using the SDK.
func (e *Extended) PlaceOrder(market string, side OrderSide, orderType OrderType, amount, price float64) (*Order, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// 1. Get market details from the exchange
	markets, err := e.client.GetMarkets(ctx, []string{market})
	if err != nil {
		return nil, fmt.Errorf("failed to get market details for %s: %w", market, err)
	}
	if len(markets) == 0 {
		return nil, fmt.Errorf("market %s not found on Extended", market)
	}
	marketInfo := markets[0]

	// 2. Prepare order parameters
	orderSide := sdk.OrderSideBuy
	if side == Sell {
		orderSide = sdk.OrderSideSell
	}

	nonce := int(time.Now().Unix())
	params := sdk.CreateOrderObjectParams{
		Market:                   marketInfo,
		Account:                  *e.account,
		SyntheticAmount:          decimal.NewFromFloat(amount),
		Side:                     orderSide,
		Signer:                   e.account.Sign,
		StarknetDomain:           e.getStarknetDomain(),
		SelfTradeProtectionLevel: sdk.SelfTradeProtectionAccount,
		Nonce:                    &nonce,
	}

	if orderType == Market {
		params.TimeInForce = sdk.TimeInForceIOC
		// For market orders, the price field is still required for slippage protection.
		// We'll calculate a price with a 5% buffer.
		markPrice, err := e.GetMarkPrice(market)
		if err != nil {
			return nil, fmt.Errorf("could not get mark price for market order: %w", err)
		}
		var orderPrice float64
		if side == Buy {
			orderPrice = markPrice * 1.05
		} else {
			orderPrice = markPrice * 0.95
		}
		params.Price = decimal.NewFromFloat(orderPrice)
	} else {
		params.TimeInForce = sdk.TimeInForceGTT
		params.Price = decimal.NewFromFloat(price)
	}

	// 3. Create and sign the order object
	fmt.Printf("\n==> Creating and signing Extended order for %s...\n", market)
	order, err := sdk.CreateOrderObject(params)
	if err != nil {
		return nil, fmt.Errorf("failed to create SDK order object: %w", err)
	}
	orderJSON, _ := json.Marshal(order)
	fmt.Printf("    Signed Order Payload: %s\n", string(orderJSON))

	// 4. Submit the order
	fmt.Println("    Submitting order to Extended API...")
	response, err := e.client.SubmitOrder(ctx, order)
	if err != nil {
		fmt.Printf("<== Extended Raw Error Response: %v\n", err)
		return nil, fmt.Errorf("failed to submit order via SDK: %w", err)
	}
	respJSON, _ := json.Marshal(response)
	fmt.Printf("<== Extended Raw Success Response: %s\n", string(respJSON))

	// 5. Return a standardized Order object
	return &Order{
		ID:        strconv.FormatInt(int64(response.Data.OrderID), 10),
		Market:    market,
		Side:      side,
		Type:      orderType,
		Price:     price,
		Amount:    amount,
		Status:    "NEW", // The SDK response doesn't include status, assuming NEW.
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

	resp, err := e.httpClient.Do(req)
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

	// Using a market order to close, so price is irrelevant (can be 0).
	return e.PlaceOrder(market, closeSide, Market, amount, 0)
}
