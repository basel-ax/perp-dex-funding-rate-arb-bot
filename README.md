# Perpetual DEX Funding Rate Arbitrage Bot

This is a Go application designed to perform funding rate arbitrage strategies on perpetual derivative exchanges (DEXs). The bot connects to multiple exchanges, monitors funding rates for specified markets, and executes trades to capitalize on significant differences.

The initial implementation includes connectors for **Lighter** and **Extended** exchanges.

**Disclaimer:** This is a trading bot that executes real trades. Use it at your own risk. The authors are not responsible for any financial losses. It is highly recommended to run this bot in `testnet` mode and thoroughly test your strategy before deploying it with real funds.

## Features

-   Modular design with a generic `Exchange` interface for easy expansion.
-   Connectors for Lighter and Extended perpetual DEXs.
-   Configuration driven by a `.env` file for easy management of parameters.
-   Command-line interface (CLI) for starting and stopping the bot.
-   Core logic for identifying and executing funding rate arbitrage opportunities.
-   Testnet and Mainnet modes.

## Getting Started

### Prerequisites

-   Go (version 1.18 or higher)
-   Git

### Installation

1.  **Clone the repository:**
    ```sh
    git clone <repository_url>
    cd perp-dex-funding-rate-arb-bot
    ```

2.  **Install dependencies:**
    The project uses Go modules. Dependencies will be automatically downloaded when you build or run the project. You can also download them manually:
    ```sh
    go mod tidy
    ```

### Configuration

Before running the bot, you need to set up your configuration.

1.  **Create a configuration file:**
    Copy the example configuration file `example.env` to a new file, for example, `.env`.
    ```sh
    cp example.env .env
    ```
    **Note:** It's recommended to use a different name like `my_config.env` and use the `--config` flag if you want to avoid committing your keys by accident. The application loads `example.env` by default.

2.  **Edit the configuration file:**
    Open your new `.env` file and fill in the required values:

    -   `LIGHTER_API_KEY`: Your API key for the Lighter exchange.
    -   `LIGHTER_PRIVATE_KEY`: Your API private key for the Lighter exchange.
    -   `EXTENDED_API_KEY`: Your API key for the Extended exchange.
    -   `TESTNET`: Set to `true` for testnet or `false` for mainnet. **Default is `true`**.
    -   `MARKETS`: A comma-separated list of markets to monitor (e.g., `BTC-USD,ETH-USD`).
    -   `MIN_FUNDING_RATE_DIFF`: The minimum percentage difference in funding rates to trigger a trade (e.g., `0.0001` for 0.01%).
    -   `POSITION_SIZE_USD`: The value of each arbitrage position in USD.
    -   `MAX_POSITION_USD`: The maximum total value of all open positions in USD.

## Usage

### Building the Application

You can build the executable with the following command:
```sh
go build .
```

### Running the Bot

To start the trading bot, run the `trade` command:

```sh
go run main.go trade
```

Or, if you have built the executable:
```sh
./perp-dex-funding-rate-arb-bot trade
```

The bot will start, load the configuration, and begin monitoring the funding rates on the specified markets. It will print log messages to the console.

To stop the bot, press `Ctrl+C`. The bot will perform a graceful shutdown.

### Running Tests

The project includes an integration test that simulates the full arbitrage cycle: opening and closing positions on both exchanges in testnet mode.

**Prerequisites for testing:**
- Ensure you have a `example.env` file (or a copy) correctly configured with your **testnet** API keys.
- `TESTNET` must be set to `true` in your `.env` file.

To run the tests, use the following command:
```sh
go test -v ./...
```
The `-v` flag provides verbose output, which is helpful for seeing the test's progress logs.

## Available Commands

-   `trade`: Starts the funding rate arbitrage trading bot.

## Project Structure

```
/
├── cmd/                # Cobra CLI commands
│   ├── root.go         # Root command setup
│   └── trade/
│       └── trade.go    # The 'trade' command
├── config/             # Configuration loading
│   └── config.go
├── pkg/                # Main application packages
│   ├── exchange/       # Exchange interfaces and implementations
│   │   ├── exchange.go
│   │   ├── lighter.go
│   │   └── extended.go
│   └── strategy/       # Trading strategy logic
│       └── funding_rate_arb.go
├── .gitignore
├── go.mod
├── go.sum
├── main.go             # Application entry point
├── example.env         # Example configuration file
└── README.md
```

## How It Works

1.  **Initialization**: The bot loads the configuration from the `.env` file and initializes the specified exchange clients.
2.  **Monitoring**: It enters a loop, periodically fetching the funding rates for the configured markets from both exchanges.
3.  **Analysis**: For each market, it compares the funding rates.
4.  **Execution**: If the absolute difference between the funding rates exceeds `MIN_FUNDING_RATE_DIFF`, the bot identifies an arbitrage opportunity.
    -   It will open a **long** position on the exchange with the lower funding rate.
    -   It will simultaneously open a **short** position on the exchange with the higher funding rate.
    -   The goal is to pay the lower funding rate and receive the higher one, profiting from the difference.
5.  **Position Management**: The bot keeps track of open positions to avoid opening duplicate trades for the same market. (Note: Closing positions automatically when the funding rate differential inverts is a feature for future implementation).

## Extending the Bot

To add support for a new exchange, you need to:

1.  Create a new file in the `pkg/exchange/` directory (e.g., `pkg/exchange/new_exchange.go`).
2.  Implement the `Exchange` interface defined in `pkg/exchange/exchange.go` for the new exchange.
3.  Update the `cmd/trade/trade.go` file to instantiate your new exchange client.