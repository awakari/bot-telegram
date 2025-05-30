package service

import (
	"context"
	"errors"
	"fmt"
	"github.com/awakari/bot-telegram/model/usage"
	"github.com/awakari/bot-telegram/service/limits"
	"github.com/awakari/bot-telegram/util"
	"gopkg.in/telebot.v3"
	"log/slog"
	"time"
)

type PaidChatMemberHandler struct {
	GroupId                      string
	Log                          *slog.Logger
	LimitByChatIdSubscriptions   map[int64]int64
	LimitByChatIdInterests       map[int64]int64
	LimitByChatIdInterestsPublic map[int64]int64
	SvcLimits                    limits.Service
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
	var userTg *telebot.User
	bot := tgCtx.Bot()
	if u.NewChatMember != nil {
		if u.NewChatMember.User != nil {
			userTg = u.NewChatMember.User
			tgCtx = bot.NewContext(telebot.Update{
				Message: &telebot.Message{
					Chat: &telebot.Chat{
						ID: userTg.ID,
					},
				},
			})
		}
	}
	err = errors.Join(err, h.ensureLimit(tgCtx, h.LimitByChatIdSubscriptions, userTg, usage.SubjectSubscriptions))
	err = errors.Join(err, h.ensureLimit(tgCtx, h.LimitByChatIdInterests, userTg, usage.SubjectInterests))
	err = errors.Join(err, h.ensureLimit(tgCtx, h.LimitByChatIdInterestsPublic, userTg, usage.SubjectInterestsPublic))
	return
}

func (h PaidChatMemberHandler) ensureLimit(tgCtx telebot.Context, limitByChatId map[int64]int64, userTg *telebot.User, subj usage.Subject) (err error) {
	ctx := context.TODO()
	bot := tgCtx.Bot()
	entitled := make(map[int64]*telebot.Chat)
	var limitMax int64
	for chatId, limit := range limitByChatId {
		chat := &telebot.Chat{
			ID: chatId,
		}
		member, errMember := bot.ChatMemberOf(chat, userTg)
		if errMember != nil {
			err = errors.Join(err, fmt.Errorf("failed to check if a user %d is member of chat %d: %w", userTg.ID, chatId, errMember))
		}
		if member.Role == telebot.Member {
			entitled[limit] = chat
			if limit > limitMax {
				limitMax = limit
			}
		}
	}
	userId := util.TelegramToAwakariUserId(userTg.ID)
	switch len(entitled) {
	case 0:
		errLimDel := h.SvcLimits.Delete(ctx, h.GroupId, userId, subj)
		switch errLimDel {
		case nil:
			_ = tgCtx.Send(fmt.Sprintf("Limit has been reset to default: %s", subj), telebot.ModeHTML)
		default:
			err = errors.Join(err, fmt.Errorf("failed to delete the %s limit for %s: %w", subj, userId, errLimDel))
		}
	default:
		for limit, chat := range entitled {
			switch limit {
			case limitMax:
				limitCurr, errLimGet := h.SvcLimits.Get(ctx, h.GroupId, userId, usage.SubjectSubscriptions)
				switch errLimGet {
				case nil:
					if limitMax > limitCurr.Count {
						errLimSet := h.SvcLimits.Set(ctx, h.GroupId, userId, usage.SubjectSubscriptions, limitMax, noExpiration)
						switch errLimSet {
						case nil:
							_ = tgCtx.Send(fmt.Sprintf("Limit has been set to %d: %s", limitMax, subj), telebot.ModeHTML)
						default:
							err = errors.Join(err, fmt.Errorf("failed to set the %s limit for %s to %d: %w", subj, userId, limitMax, errLimGet))
						}
					}
				default:
					err = errors.Join(err, fmt.Errorf("failed to get the %s limit for %s: %w", subj, userId, errLimGet))
				}
			default:
				member := &telebot.ChatMember{
					User: userTg,
				}
				errRemove := bot.Ban(chat, member)
				if errRemove == nil {
					errRemove = bot.Unban(chat, userTg)
				}
				switch errRemove {
				case nil:
					_ = tgCtx.Send(
						fmt.Sprintf(
							"You have been removed from the channel with limit of %s: %d, because you are a member of another channel with limit of %s: %d",
							subj,
							limit,
							subj,
							limitMax,
						),
						telebot.ModeHTML,
					)
				default:
					err = errors.Join(err, fmt.Errorf("failed to remove the user %s from the channel %d: %w", userId, chat.ID, errRemove))
				}
			}
		}
	}
	return
}
