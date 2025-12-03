package cmd

import (
	"fmt"
	"os"

	"github.com/basel-ax/perp-dex-funding-rate-arb-bot/cmd/trade"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "funding-rate-arb-bot",
	Short: "A bot for funding rate arbitrage on perpetual DEXs.",
	Long:  `A command-line tool to execute funding rate arbitrage strategies on various perpetual derivative exchanges.`,
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.AddCommand(trade.TradeCmd)
}
