package telegram

import "gopkg.in/telebot.v3"

type Handler interface {
	Handler() telebot.HandlerFunc
}
