package strategy

import (
	"log"
	"os"
	"testing"
	"time"

	"github.com/basel-ax/perp-dex-funding-rate-arb-bot/config"
	"github.com/basel-ax/perp-dex-funding-rate-arb-bot/pkg/exchange"
)

// TestArbitrageExecution is an integration test that opens and closes positions.
// It requires a valid `example.env` file with testnet API keys.
func TestArbitrageExecution(t *testing.T) {
	// Load config. The test expects to be run from the project root
	// or have the config file in a discoverable path.
	cfg, err := config.LoadConfig(".")
	if err != nil {
		t.Fatalf("cannot load config for test. Make sure example.env is present: %v", err)
	}

	// This is a safety check to ensure this test only runs in testnet mode.
	if !cfg.Testnet {
		t.Skip("Skipping execution test: TESTNET is not set to true in config")
	}

	logger := log.New(os.Stdout, "[ARB-TEST] ", log.LstdFlags)

	logger.Println("Initializing exchanges for integration test...")
	lighterEx := exchange.NewLighter(cfg.LighterAPIKey, cfg.LighterPrivateKey, true)
	extendedEx := exchange.NewExtended(cfg.ExtendedAPIKey, true)

	// --- Test Parameters ---
	market := "BTC-USD"         // Using a common market for the test
	positionSizeUSD := 10.0     // Use a very small, fixed size for testing
	placeholderPrice := 60000.0 // A recent approximate price to calculate order amount
	amount := positionSizeUSD / placeholderPrice

	// --- Scenario 1: Short Lighter, Long Extended ---
	logger.Printf("\n--- Starting Scenario 1: Short on Lighter, Long on Extended for %s ---\n", market)

	// Open positions
	logger.Println("Placing orders to open positions...")
	shortOrder, err := lighterEx.PlaceOrder(market, exchange.Sell, exchange.Market, amount, 0) // price 0 for market order
	if err != nil {
		t.Fatalf("Scenario 1: Failed to place SHORT order on Lighter: %v", err)
	}
	logger.Printf("Scenario 1: Placed SHORT order on Lighter: ID %s", shortOrder.ID)

	longOrder, err := extendedEx.PlaceOrder(market, exchange.Buy, exchange.Market, amount, 0)
	if err != nil {
		// If the second leg fails, we should try to close the first one to avoid an open position.
		logger.Printf("Scenario 1: Failed to place LONG order on Extended, attempting to reverse position on Lighter...")
		_, reverseErr := lighterEx.PlaceOrder(market, exchange.Buy, exchange.Market, amount, 0)
		if reverseErr != nil {
			logger.Printf("CRITICAL: Failed to reverse Lighter position: %v", reverseErr)
		}
		t.Fatalf("Scenario 1: Failed to place LONG order on Extended: %v", err)
	}
	logger.Printf("Scenario 1: Placed LONG order on Extended: ID %s", longOrder.ID)

	logger.Println("Scenario 1: Positions opened successfully. Waiting 30 seconds...")
	time.Sleep(30 * time.Second)

	// Close positions
	logger.Println("Scenario 1: Closing positions...")
	closeShort, err := lighterEx.ClosePosition(market, exchange.Sell, amount)
	if err != nil {
		t.Errorf("Scenario 1: Failed to close position on Lighter: %v", err)
	} else {
		logger.Printf("Scenario 1: Position closure order placed on Lighter: ID %s", closeShort.ID)
	}

	closeLong, err := extendedEx.ClosePosition(market, exchange.Buy, amount)
	if err != nil {
		t.Errorf("Scenario 1: Failed to close position on Extended: %v", err)
	} else {
		logger.Printf("Scenario 1: Position closure order placed on Extended: ID %s", closeLong.ID)
	}
	logger.Println("--- Finished Scenario 1 ---")

	// Pause between scenarios to let exchanges process
	logger.Println("Waiting 15 seconds before next scenario...")
	time.Sleep(15 * time.Second)

	// --- Scenario 2: Long Lighter, Short Extended ---
	logger.Printf("\n--- Starting Scenario 2: Long on Lighter, Short on Extended for %s ---\n", market)

	// Open positions
	logger.Println("Placing orders to open positions...")
	longOrder2, err := lighterEx.PlaceOrder(market, exchange.Buy, exchange.Market, amount, 0)
	if err != nil {
		t.Fatalf("Scenario 2: Failed to place LONG order on Lighter: %v", err)
	}
	logger.Printf("Scenario 2: Placed LONG order on Lighter: ID %s", longOrder2.ID)

	shortOrder2, err := extendedEx.PlaceOrder(market, exchange.Sell, exchange.Market, amount, 0)
	if err != nil {
		// Attempt to close the first leg if the second fails
		logger.Printf("Scenario 2: Failed to place SHORT order on Extended, attempting to reverse position on Lighter...")
		_, reverseErr := lighterEx.PlaceOrder(market, exchange.Sell, exchange.Market, amount, 0)
		if reverseErr != nil {
			logger.Printf("CRITICAL: Failed to reverse Lighter position: %v", reverseErr)
		}
		t.Fatalf("Scenario 2: Failed to place SHORT order on Extended: %v", err)
	}
	logger.Printf("Scenario 2: Placed SHORT order on Extended: ID %s", shortOrder2.ID)

	logger.Println("Scenario 2: Positions opened successfully. Waiting 30 seconds...")
	time.Sleep(30 * time.Second)

	// Close positions
	logger.Println("Scenario 2: Closing positions...")
	closeLong2, err := lighterEx.ClosePosition(market, exchange.Buy, amount)
	if err != nil {
		t.Errorf("Scenario 2: Failed to close position on Lighter: %v", err)
	} else {
		logger.Printf("Scenario 2: Position closure order placed on Lighter: ID %s", closeLong2.ID)
	}

	closeShort2, err := extendedEx.ClosePosition(market, exchange.Sell, amount)
	if err != nil {
		t.Errorf("Scenario 2: Failed to close position on Extended: %v", err)
	} else {
		logger.Printf("Scenario 2: Position closure order placed on Extended: ID %s", closeShort2.ID)
	}
	logger.Println("--- Finished Scenario 2 ---")
}
