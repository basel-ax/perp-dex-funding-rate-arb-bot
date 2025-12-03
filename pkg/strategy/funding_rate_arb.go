package strategy

import (
	"log"
	"math"
	"sync"
	"time"

	"github.com/basel-ax/perp-dex-funding-rate-arb-bot/config"
	"github.com/basel-ax/perp-dex-funding-rate-arb-bot/pkg/exchange"
	"github.com/basel-ax/perp-dex-funding-rate-arb-bot/pkg/notifications"
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
	notifier  *notifications.TelegramNotifier
	positions map[string]*PositionInfo
	mu        sync.Mutex
}

// NewFundingRateArb creates a new arbitrage strategy instance.
func NewFundingRateArb(cfg config.Config, ex1, ex2 exchange.Exchange, logger *log.Logger, notifier *notifications.TelegramNotifier) *Strategy {
	return &Strategy{
		config:    cfg,
		exchange1: ex1,
		exchange2: ex2,
		logger:    logger,
		notifier:  notifier,
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

		s.mu.Lock()
		position, exists := s.positions[market]
		s.mu.Unlock()

		// Condition to OPEN a position
		if !exists && math.Abs(diff) > s.config.MinFundingRateDiff {
			if diff > 0 {
				// rate1 is higher, short on exchange1, long on exchange2
				s.executeArbitrage(market, s.exchange2, s.exchange1, diff)
			} else {
				// rate2 is higher, short on exchange2, long on exchange1
				s.executeArbitrage(market, s.exchange1, s.exchange2, -diff)
			}
		} else if exists { // Condition to CLOSE a position
			// Close if the rate difference has inverted or flattened.
			shouldClose := false
			// Case 1: We are short exchange1 because its rate was higher.
			if position.ShortExchange.Name() == s.exchange1.Name() && diff <= 0 {
				shouldClose = true
			}
			// Case 2: We are short exchange2 because its rate was higher.
			if position.ShortExchange.Name() == s.exchange2.Name() && diff >= 0 {
				shouldClose = true
			}

			if shouldClose {
				s.logger.Printf("Funding rate difference for %s is no longer favorable. Closing position.", market)
				s.closeArbitrage(position)
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
	s.notifier.SendPositionNotification("OPEN LONG", longEx.Name(), market, s.config.PositionSizeUSD, err)
	if err != nil {
		s.logger.Printf("Failed to place LONG order on %s: %v", longEx.Name(), err)
		return // Don't proceed to short if long fails
	}
	s.logger.Printf("Successfully placed LONG order: ID %s", longOrder.ID)

	s.logger.Printf("Placing SHORT order on %s for %f of %s at price %.2f", shortEx.Name(), amount, market, currentPrice)
	shortOrder, err := shortEx.PlaceOrder(market, exchange.Sell, exchange.Market, amount, currentPrice)
	s.notifier.SendPositionNotification("OPEN SHORT", shortEx.Name(), market, s.config.PositionSizeUSD, err)
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

// closeArbitrage closes an open arbitrage position and sends notifications.
func (s *Strategy) closeArbitrage(position *PositionInfo) {
	s.mu.Lock()
	// Check if it's still there, might have been closed by another thread.
	if _, exists := s.positions[position.Market]; !exists {
		s.mu.Unlock()
		return
	}
	// remove from map immediately to prevent re-entry
	delete(s.positions, position.Market)
	s.mu.Unlock()

	s.logger.Printf("Closing arbitrage position for %s...", position.Market)

	// Amount needs to be calculated based on SizeUSD and current price
	var currentPrice float64
	if position.Market == "BTC-USD" {
		currentPrice = 60000.0
	} else if position.Market == "ETH-USD" {
		currentPrice = 3000.0
	} else {
		s.logger.Printf("No placeholder price for market %s, cannot calculate close order amount.", position.Market)
		return
	}
	amount := position.SizeUSD / currentPrice

	// Close positions
	_, longCloseErr := position.LongExchange.ClosePosition(position.Market, exchange.Buy, amount)
	s.notifier.SendPositionNotification("CLOSE LONG", position.LongExchange.Name(), position.Market, position.SizeUSD, longCloseErr)
	if longCloseErr != nil {
		s.logger.Printf("Failed to close LONG position on %s: %v", position.LongExchange.Name(), longCloseErr)
	} else {
		s.logger.Printf("Successfully closed LONG position on %s.", position.LongExchange.Name())
	}

	_, shortCloseErr := position.ShortExchange.ClosePosition(position.Market, exchange.Sell, amount)
	s.notifier.SendPositionNotification("CLOSE SHORT", position.ShortExchange.Name(), position.Market, position.SizeUSD, shortCloseErr)
	if shortCloseErr != nil {
		s.logger.Printf("Failed to close SHORT position on %s: %v", position.ShortExchange.Name(), shortCloseErr)
	} else {
		s.logger.Printf("Successfully closed SHORT position on %s.", position.ShortExchange.Name())
	}
}
