package config

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

// Config stores all configuration for the application.
// The values are read by viper from a config file or environment variables.
type Config struct {
	LighterAPIKey      string   `mapstructure:"LIGHTER_API_KEY"`
	LighterPrivateKey  string   `mapstructure:"LIGHTER_PRIVATE_KEY"`
	ExtendedAPIKey     string   `mapstructure:"EXTENDED_API_KEY"`
	Testnet            bool     `mapstructure:"TESTNET"`
	Markets            []string `mapstructure:"MARKETS"`
	MinFundingRateDiff float64  `mapstructure:"MIN_FUNDING_RATE_DIFF"`
	PositionSizeUSD    float64  `mapstructure:"POSITION_SIZE_USD"`
	MaxPositionUSD     float64  `mapstructure:"MAX_POSITION_USD"`
}

// LoadConfig reads configuration from file or environment variables.
func LoadConfig(path string) (config Config, err error) {
	viper.AddConfigPath(path)
	viper.SetConfigName("example")
	viper.SetConfigType("env")

	viper.AutomaticEnv()

	err = viper.ReadInConfig()
	if err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			// Config file not found; ignore error if desired
			fmt.Println("config file not found, using environment variables")
		} else {
			// Config file was found but another error was produced
			return
		}
	}

	// Workaround for viper not splitting comma-separated strings from .env files
	if viper.IsSet("MARKETS") {
		markets := viper.GetString("MARKETS")
		viper.Set("MARKETS", strings.Split(markets, ","))
	}

	err = viper.Unmarshal(&config)
	return
}
