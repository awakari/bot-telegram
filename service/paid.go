package service

import (
	"context"
	"fmt"
	"github.com/awakari/bot-telegram/model/usage"
	"github.com/awakari/bot-telegram/service/limits"
	"github.com/awakari/bot-telegram/util"
	"gopkg.in/telebot.v3"
	"log/slog"
	"time"
)

type PaidChatMemberHandler struct {
	GroupId           string
	Log               *slog.Logger
	LimitByChatIdSubs map[int64]int64
	SvcLimits         limits.Service
}

var noExpiration = time.Time{}

func (h PaidChatMemberHandler) Handle(tgCtx telebot.Context) (err error) {
	u := tgCtx.ChatMember()
	if u != nil {
		err = h.handleChatMember(tgCtx, u)
	}
	return
}

func (h PaidChatMemberHandler) handleChatMember(tgCtx telebot.Context, u *telebot.ChatMemberUpdate) (err error) {
	var chatId int64
	if u.Chat != nil {
		chatId = u.Chat.ID
	}
	var userId string
	var status telebot.MemberStatus
	if u.NewChatMember != nil {
		if u.NewChatMember.User != nil {
			userId = util.TelegramToAwakariUserId(u.NewChatMember.User.ID)
			tgCtx = tgCtx.Bot().NewContext(telebot.Update{
				Message: &telebot.Message{
					Chat: &telebot.Chat{
						ID: u.NewChatMember.User.ID,
					},
				},
			})
		}
		status = u.NewChatMember.Role
	}
	limitSubs, limitSubsOk := h.LimitByChatIdSubs[chatId]
	if limitSubsOk {
		err = h.handle(tgCtx, userId, status, limitSubs)
	}
	return
}

func (h PaidChatMemberHandler) handle(tgCtx telebot.Context, userId string, status telebot.MemberStatus, limitSubs int64) (err error) {
	switch status {
	case telebot.Member:
		err = h.joined(tgCtx, userId, limitSubs)
		h.Log.Log(context.TODO(), util.LogLevel(err), fmt.Sprintf("%s joined %d subscriptions level, error: %s", userId, limitSubs, err))
	case telebot.Kicked, telebot.Left:
		err = h.left(tgCtx, userId)
		h.Log.Log(context.TODO(), util.LogLevel(err), fmt.Sprintf("%s left %d subscriptions level, error: %s", userId, limitSubs, err))
	default:
		h.Log.Debug(fmt.Sprintf("the membership status of %s changed to %s", userId, status))
	}
	return
}

func (h PaidChatMemberHandler) joined(tgCtx telebot.Context, userId string, limitSubs int64) (err error) {
	ctx := context.TODO()
	var l usage.Limit
	l, err = h.SvcLimits.Get(context.TODO(), h.GroupId, userId, usage.SubjectSubscriptions)
	if err == nil && limitSubs > l.Count {
		err = h.SvcLimits.Set(ctx, h.GroupId, userId, usage.SubjectSubscriptions, limitSubs, noExpiration)
		if err == nil {
			_ = tgCtx.Send(fmt.Sprintf("Subscriptions limit has been set to %d", limitSubs), telebot.ModeHTML)
		}
	}
	return
}

func (h PaidChatMemberHandler) left(tgCtx telebot.Context, userId string) (err error) {
	err = h.SvcLimits.Delete(context.TODO(), h.GroupId, userId, usage.SubjectSubscriptions)
	if err == nil {
		_ = tgCtx.Send("Subscriptions limit has been reset to default", telebot.ModeHTML)
	}
	return
}
