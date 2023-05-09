package cfip

import (
	"github.com/go-telegram-bot-api/telegram-bot-api"
)

func PushTgBot(c string) error {
	bot, err := tgbotapi.NewBotAPI("1")
	if err != nil {
		return err
	}
	bot.Debug = false
	msg := tgbotapi.NewMessage(1, c)
	_, err = bot.Send(msg)
	if err != nil {
		return err
	}
	return nil
}
