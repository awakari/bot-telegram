package service

import (
	"errors"
	"fmt"
	"gopkg.in/telebot.v3"
	"strings"
)

type RootHandler struct {
	ReplyHandlers  map[string]ArgHandlerFunc
	ForwardHandler telebot.HandlerFunc
	TxtHandlers    map[string]telebot.HandlerFunc
}

func (h RootHandler) Handle(tgCtx telebot.Context) (err error) {
	switch {
	case tgCtx.Message().IsReply():
		err = h.handleReply(tgCtx)
	case tgCtx.Message().IsForwarded() && h.ForwardHandler != nil:
		err = h.ForwardHandler(tgCtx)
	default:
		txt := tgCtx.Text()
		hTxt, hTxtOk := h.TxtHandlers[txt]
		switch hTxtOk {
		case true:
			err = hTxt(tgCtx)
		default:
			err = errors.New(fmt.Sprintf("unrecognized command, use the reply keyboard menu"))
		}
	}
	return
}

func (h RootHandler) handleReply(tgCtx telebot.Context) (err error) {
	msgResp := tgCtx.Message()
	txtResp := msgResp.Text
	msgReq := msgResp.ReplyTo
	txtReq := msgReq.Text
	args := strings.Split(txtReq, " ")
	handlerKey := args[0]
	rh, rhOk := h.ReplyHandlers[handlerKey]
	switch rhOk {
	case true:
		args = append(args, txtResp)
		err = rh(tgCtx, args...)
	default:
		err = errors.New(fmt.Sprintf("unknown reply handler key: %s", handlerKey))
	}
	return
}
