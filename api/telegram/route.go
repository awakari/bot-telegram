package telegram

import (
	"errors"
	"fmt"
	"github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

var ErrUnrecognizedCommand = errors.New("unrecognized command")

type RouteHandler struct {
	handlerByCmd map[string]Handler
}

func NewRouteHandler(handlerByCmd map[string]Handler) Handler {
	return RouteHandler{
		handlerByCmd: handlerByCmd,
	}
}

func (r RouteHandler) Handle(req *tgbotapi.Message) (resp tgbotapi.MessageConfig, err error) {
	cmd := req.Command()
	h, ok := r.handlerByCmd[cmd]
	if ok {
		resp, err = h.Handle(req)
	} else {
		err = fmt.Errorf("%w: %s", ErrUnrecognizedCommand, cmd)
	}
	return
}
