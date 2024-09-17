package service

import (
	"context"
	"errors"
	"fmt"
	apiGrpcSrcTg "github.com/awakari/bot-telegram/api/grpc/source-telegram"
	"gopkg.in/telebot.v3"
	"regexp"
	"strconv"
)

type LoginCodeHandler struct {
	FromUserIds  map[int64]uint
	SourceUserId int64
	SvcSrcTg     apiGrpcSrcTg.Service
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
	replicaIdx, replicaIdxPresent := h.FromUserIds[userId]
	if !replicaIdxPresent {
		err = fmt.Errorf("message is forwarded by user %d", userId)
		return
	}
	var code uint64
	code, err = GetLoginCode(msg.Text)
	if err != nil {
		return
	}
	err = h.SvcSrcTg.Login(context.TODO(), int64(code), replicaIdx)
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
