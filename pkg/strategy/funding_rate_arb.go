package strategy

import (
	"log"
	"math"
	"sync"
	"time"

	"github.com/basel-ax/perp-dex-funding-rate-arb-bot/config"
	"github.com/basel-ax/perp-dex-funding-rate-arb-bot/pkg/exchange"
)

// PositionInfo tracks an open arbitrage position.
type PositionInfo struct {
	Market        string
	LongExchange  exchange.Exchange
	ShortExchange exchange.Exchange
	SizeUSD       float64
}

// Strategy holds the core logic for the funding rate arbitrage bot.
type Strategy struct {
	config    config.Config
	exchange1 exchange.Exchange
	exchange2 exchange.Exchange
	logger    *log.Logger
	positions map[string]*PositionInfo
	mu        sync.Mutex
}

// NewFundingRateArb creates a new arbitrage strategy instance.
func NewFundingRateArb(cfg config.Config, ex1, ex2 exchange.Exchange, logger *log.Logger) *Strategy {
	return &Strategy{
		config:    cfg,
		exchange1: ex1,
		exchange2: ex2,
		logger:    logger,
		positions: make(map[string]*PositionInfo),
	}
}

// Run starts the arbitrage strategy loop.
func (s *Strategy) Run(stop chan struct{}) {
	s.logger.Println("Starting funding rate arbitrage strategy...")
	s.logger.Printf("Exchanges: %s, %s", s.exchange1.Name(), s.exchange2.Name())
	s.logger.Printf("Markets: %v", s.config.Markets)
	s.logger.Printf("Minimum Rate Difference: %.4f%%", s.config.MinFundingRateDiff*100)
	s.logger.Printf("Position Size (USD): %.2f", s.config.PositionSizeUSD)

	// Run checks on a ticker
	ticker := time.NewTicker(1 * time.Minute) // Check every minute
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.checkFundingRates()
		case <-stop:
			s.logger.Println("Stopping strategy...")
			return
		}
	}
}

// checkFundingRates fetches and compares funding rates to find opportunities.
func (s *Strategy) checkFundingRates() {
	s.logger.Println("Checking for funding rate arbitrage opportunities...")

	rates1, err := s.exchange1.GetFundingRates()
	if err != nil {
		s.logger.Printf("Error getting funding rates from %s: %v", s.exchange1.Name(), err)
		return
	}

	rates2, err := s.exchange2.GetFundingRates()
	if err != nil {
		s.logger.Printf("Error getting funding rates from %s: %v", s.exchange2.Name(), err)
		return
	}

	rates1Map := make(map[string]float64)
	for _, r := range rates1 {
		rates1Map[r.Market] = r.Rate
	}

	rates2Map := make(map[string]float64)
	for _, r := range rates2 {
		rates2Map[r.Market] = r.Rate
	}

	for _, market := range s.config.Markets {
		rate1, ok1 := rates1Map[market]
		rate2, ok2 := rates2Map[market]

		if !ok1 || !ok2 {
			s.logger.Printf("Market %s not available on both exchanges, skipping.", market)
			continue
		}

		diff := rate1 - rate2
		s.logger.Printf("Market: %s | %s Rate: %.6f | %s Rate: %.6f | Diff: %.6f",
			market, s.exchange1.Name(), rate1, s.exchange2.Name(), rate2, diff)

		if math.Abs(diff) > s.config.MinFundingRateDiff {
			if diff > 0 {
				// rate1 is higher, short on exchange1, long on exchange2
				s.executeArbitrage(market, s.exchange2, s.exchange1, diff)
			} else {
				// rate2 is higher, short on exchange2, long on exchange1
				s.executeArbitrage(market, s.exchange1, s.exchange2, -diff)
			}
		}
	}
}

// executeArbitrage places the long and short orders to capitalize on a funding rate difference.
func (s *Strategy) executeArbitrage(market string, longEx, shortEx exchange.Exchange, rateDiff float64) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check if a position is already open for this market
	if _, exists := s.positions[market]; exists {
		s.logger.Printf("Position already open for market %s, skipping.", market)
		return
	}

	s.logger.Printf("Arbitrage opportunity found for %s!", market)
	s.logger.Printf("  - Long on: %s", longEx.Name())
	s.logger.Printf("  - Short on: %s", shortEx.Name())
	s.logger.Printf("  - Rate Difference: %.6f", rateDiff)

	// Check if opening a new position exceeds the max total position size
	if s.getTotalPositionValue()+s.config.PositionSizeUSD > s.config.MaxPositionUSD {
		s.logger.Printf("Cannot open new position, max total position size of %.2f USD would be exceeded.", s.config.MaxPositionUSD)
		return
	}

	// TODO: Fetch the current price to calculate the amount in the base currency.
	// This is a placeholder as the exchange interface does not yet support fetching price tickers.
	// Using a hardcoded price for BTC-USD for demonstration.
	var currentPrice float64
	if market == "BTC-USD" {
		currentPrice = 60000.0
	} else if market == "ETH-USD" {
		currentPrice = 3000.0
	} else {
		s.logger.Printf("No placeholder price for market %s, cannot calculate order amount.", market)
		return
	}

	amount := s.config.PositionSizeUSD / currentPrice

	// Place orders
	s.logger.Printf("Placing LONG order on %s for %f of %s at price %.2f", longEx.Name(), amount, market, currentPrice)
	longOrder, err := longEx.PlaceOrder(market, exchange.Buy, exchange.Market, amount, currentPrice)
	if err != nil {
		s.logger.Printf("Failed to place LONG order on %s: %v", longEx.Name(), err)
		return // Don't proceed to short if long fails
	}
	s.logger.Printf("Successfully placed LONG order: ID %s", longOrder.ID)

	s.logger.Printf("Placing SHORT order on %s for %f of %s at price %.2f", shortEx.Name(), amount, market, currentPrice)
	shortOrder, err := shortEx.PlaceOrder(market, exchange.Sell, exchange.Market, amount, currentPrice)
	if err != nil {
		s.logger.Printf("Failed to place SHORT order on %s: %v", shortEx.Name(), err)
		// TODO: Need to handle the case where the long order was placed but the short failed.
		// This would involve cancelling the long order immediately.
		s.logger.Println("CRITICAL: Long order was placed but short order failed. Manual intervention may be required.")
		return
	}
	s.logger.Printf("Successfully placed SHORT order: ID %s", shortOrder.ID)

	// Record the new position
	s.positions[market] = &PositionInfo{
		Market:        market,
		LongExchange:  longEx,
		ShortExchange: shortEx,
		SizeUSD:       s.config.PositionSizeUSD,
	}

	s.logger.Printf("Successfully opened arbitrage position for %s. Total position value: %.2f USD", market, s.getTotalPositionValue())
}

// getTotalPositionValue calculates the total value of all open positions.
func (s *Strategy) getTotalPositionValue() float64 {
	totalValue := 0.0
	for _, pos := range s.positions {
		totalValue += pos.SizeUSD
	}
	return totalValue
}
