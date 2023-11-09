package chats

import (
	"context"
	"errors"
	"fmt"
	"gopkg.in/telebot.v3"
)

func UserLeftHandlerFunc(chatStor Storage) telebot.HandlerFunc {
	return func(ctx telebot.Context) (err error) {
		chat := ctx.Chat()
		if chat == nil {
			err = errors.New("user left a missing chat")
		}
		if err == nil {
			fmt.Printf("user left chat %d\n", chat.ID)
			chatId := chat.ID
			StopChatReaders(chatId)
			_, _ = chatStor.Delete(context.Background(), chatId)
		}
		return
	}
}
