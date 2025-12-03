package notifications

import (
	"fmt"
	"log"
	"time"

	"gopkg.in/telebot.v3"
)

// TelegramNotifier handles sending messages to a Telegram chat.
type TelegramNotifier struct {
	bot    *telebot.Bot
	chatID int64
	logger *log.Logger
}

// NewTelegramNotifier creates and initializes a new Telegram notifier.
// It returns nil if the bot token or chat ID is not provided.
func NewTelegramNotifier(token string, chatID int64, logger *log.Logger) *TelegramNotifier {
	if token == "" || chatID == 0 {
		logger.Println("Telegram token or chat ID not provided, Telegram notifier disabled.")
		return nil
	}

	pref := telebot.Settings{
		Token:  token,
		Poller: &telebot.LongPoller{Timeout: 10 * time.Second},
	}

	bot, err := telebot.NewBot(pref)
	if err != nil {
		logger.Printf("Could not create Telegram bot: %v", err)
		return nil
	}

	logger.Println("Telegram notifier initialized successfully.")
	return &TelegramNotifier{
		bot:    bot,
		chatID: chatID,
		logger: logger,
	}
}

// Start begins polling for updates. This is required by the telebot library to send messages.
func (tn *TelegramNotifier) Start() {
	if tn == nil {
		return
	}
	go tn.bot.Start()
}

// Stop stops the bot from polling.
func (tn *TelegramNotifier) Stop() {
	if tn == nil {
		return
	}
	tn.bot.Stop()
}

// SendMessage sends a plain text message to the configured chat.
func (tn *TelegramNotifier) SendMessage(message string) {
	if tn == nil {
		return // Do nothing if the notifier is not initialized
	}

	recipient := &telebot.Chat{ID: tn.chatID}

	_, err := tn.bot.Send(recipient, message, &telebot.SendOptions{ParseMode: telebot.ModeMarkdown})
	if err != nil {
		tn.logger.Printf("Failed to send Telegram message: %v", err)
	}
}

// SendPositionNotification sends a formatted message about a trading event.
func (tn *TelegramNotifier) SendPositionNotification(action, exchangeName, market string, positionSizeUSD float64, err error) {
	if tn == nil {
		return
	}

	status := "✅ SUCCESS"
	if err != nil {
		status = "❌ FAILED"
	}

	message := fmt.Sprintf(
		"**%s Position Event**\n\n"+
			"**Status:** %s\n"+
			"**Exchange:** `%s`\n"+
			"**Market:** `%s`\n"+
			"**Position Size:** `%.2f USD`",
		action, status, exchangeName, market, positionSizeUSD,
	)

	if err != nil {
		message += fmt.Sprintf("\n**Error:** `%v`", err)
	}

	tn.SendMessage(message)
}
