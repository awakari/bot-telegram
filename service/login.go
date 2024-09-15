package service

import (
	"errors"
	"fmt"
	"gopkg.in/telebot.v3"
	"regexp"
	"strconv"
)

type LoginCodeHandler struct {
	FromUserIds  map[int64]bool
	SourceUserId int64
}

var rLoginCode = regexp.MustCompile(`Login code: (\d+).*`)
var ErrInvalidLoginCodeMsg = errors.New("invalid login code message")

func (h LoginCodeHandler) Handle(tgCtx telebot.Context) (err error) {
	msg := tgCtx.Message()
	fromId := msg.OriginalSender.ID
	if fromId != h.SourceUserId {
		err = fmt.Errorf("message is forwared from %d", fromId)
		return
	}
	userId := msg.Sender.ID
	if !h.FromUserIds[userId] {
		err = fmt.Errorf("message is forwarded by user %d", userId)
		return
	}
	var code uint64
	code, err = GetLoginCode(msg.Text)
	if err != nil {
		return
	}
	fmt.Printf("LOGIN CODE: %d\n", code)
	return
}

func GetLoginCode(txt string) (code uint64, err error) {
	matches := rLoginCode.FindStringSubmatch(txt)
	switch len(matches) {
	case 0, 1:
		err = fmt.Errorf("%w: %s", ErrInvalidLoginCodeMsg, txt)
	default:
		code, err = strconv.ParseUint(matches[1], 10, 16)
	}
	return
}
