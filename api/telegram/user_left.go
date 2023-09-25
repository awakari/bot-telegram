package telegram

import (
	"context"
	"errors"
	"github.com/awakari/bot-telegram/api/telegram/events"
	"github.com/awakari/bot-telegram/chats"
	"gopkg.in/telebot.v3"
)

func UserLeftHandlerFunc(chatStor chats.Storage) telebot.HandlerFunc {
	return func(ctx telebot.Context) (err error) {
		chat := ctx.Chat()
		if chat == nil {
			err = errors.New("user left a missing chat")
		}
		if err == nil {
			chatId := chat.ID
			events.StopChatReader(chatId)
			_ = chatStor.Delete(context.Background(), chatId)
		}
		return
	}
}
