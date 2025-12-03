package trade

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/basel-ax/perp-dex-funding-rate-arb-bot/config"
	"github.com/basel-ax/perp-dex-funding-rate-arb-bot/pkg/exchange"
	"github.com/basel-ax/perp-dex-funding-rate-arb-bot/pkg/notifications"
	"github.com/basel-ax/perp-dex-funding-rate-arb-bot/pkg/strategy"
)

var configPath string

// TradeCmd represents the trade command
var TradeCmd = &cobra.Command{
	Use:   "trade",
	Short: "Starts the funding rate arbitrage trading bot.",
	Long: `Initializes and runs the funding rate arbitrage strategy.
It connects to the configured exchanges, fetches funding rates,
and executes trades when an arbitrage opportunity is identified based on the provided configuration.`,
	Run: func(cmd *cobra.Command, args []string) {
		// Load configuration
		cfg, err := config.LoadConfig(configPath)
		if err != nil {
			log.Fatalf("cannot load config: %v", err)
		}

		// Setup logger
		logger := log.New(os.Stdout, "[ARB-BOT] ", log.LstdFlags)

		// Initialize exchanges
		logger.Printf("Initializing exchanges in %s mode...", map[bool]string{true: "Testnet", false: "Mainnet"}[cfg.Testnet])

		lighterEx := exchange.NewLighter(cfg.LighterAPIKey, cfg.LighterPrivateKey, cfg.Testnet)
		extendedEx := exchange.NewExtended(cfg.ExtendedAPIKey, cfg.ExtendedPrivateKey, cfg.ExtendedPublicKey, cfg.ExtendedVaultID, cfg.Testnet)

		// Initialize Telegram notifier
		notifier := notifications.NewTelegramNotifier(cfg.TelegramBotToken, cfg.TelegramChatID, logger)

		// Create the strategy
		arbStrategy := strategy.NewFundingRateArb(cfg, lighterEx, extendedEx, logger, notifier)

		// Handle graceful shutdown
		stop := make(chan struct{})
		osSignal := make(chan os.Signal, 1)
		signal.Notify(osSignal, syscall.SIGINT, syscall.SIGTERM)

		go func() {
			<-osSignal
			logger.Println("Interrupt signal received. Shutting down gracefully...")
			notifier.Stop()
			close(stop)
		}()

		// Start the notifier's poller
		notifier.Start()

		// Run the strategy
		arbStrategy.Run(stop)

		logger.Println("Bot has been shut down.")
	},
}

func init() {
	TradeCmd.Flags().StringVar(&configPath, "path", ".", "Path to the directory containing the .env file")
}
