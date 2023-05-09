package cfip

import (
	"github.com/go-telegram-bot-api/telegram-bot-api"
)

func PushTgBot(c string) error {
	bot, err := tgbotapi.NewBotAPI("5835666296:AAGed5EX2kRgGXjtMHx2RkOmtnjjETtZn1c")
	if err != nil {
		return err
	}
	bot.Debug = false
	msg := tgbotapi.NewMessage(5740566746, c)
	_, err = bot.Send(msg)
	if err != nil {
		return err
	}
	return nil
}
