package service

import (
	"context"
	"errors"
	"fmt"
	"github.com/awakari/bot-telegram/model/usage"
	"github.com/awakari/bot-telegram/service/limits"
	"github.com/awakari/bot-telegram/util"
	"gopkg.in/telebot.v3"
	"time"
)

type PaidChatMemberHandler struct {
	GroupId                      string
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

func (h PaidChatMemberHandler) handleChatMember(tgCtx telebot.Context, upd *telebot.ChatMemberUpdate) (err error) {
	var user *telebot.User
	bot := tgCtx.Bot()
	if upd.NewChatMember != nil {
		if upd.NewChatMember.User != nil {
			user = upd.NewChatMember.User
			tgCtx = bot.NewContext(telebot.Update{
				Message: &telebot.Message{
					Chat: &telebot.Chat{
						ID: user.ID,
					},
				},
			})
		}
	}
	err = errors.Join(err, h.ensureLimit(tgCtx, h.LimitByChatIdSubscriptions, user, usage.SubjectSubscriptions))
	err = errors.Join(err, h.ensureLimit(tgCtx, h.LimitByChatIdInterests, user, usage.SubjectInterests))
	err = errors.Join(err, h.ensureLimit(tgCtx, h.LimitByChatIdInterestsPublic, user, usage.SubjectInterestsPublic))
	return
}

func (h PaidChatMemberHandler) ensureLimit(
	tgCtx telebot.Context,
	limitByChatId map[int64]int64,
	user *telebot.User,
	subj usage.Subject,
) (
	err error,
) {

	ctx := context.TODO()
	bot := tgCtx.Bot()
	chanByLimit, limitMax, errMember := h.memberChannels(bot, user, limitByChatId)
	if errMember != nil {
		err = errors.Join(err, errMember)
	}

	userId := util.TelegramToAwakariUserId(user.ID)
	switch len(chanByLimit) {
	case 0:
		errLimDel := h.SvcLimits.Delete(ctx, h.GroupId, userId, subj)
		switch errLimDel {
		case nil:
			_ = tgCtx.Send(fmt.Sprintf("Limit has been reset to default: %s", subj.Description()), telebot.ModeHTML)
		default:
			err = errors.Join(err, fmt.Errorf("failed to delete the %s limit for %s: %w", subj.Description(), userId, errLimDel))
		}
	default:
		for limit, chat := range chanByLimit {
			switch limit {
			case limitMax:
				limitCurr, errLimGet := h.SvcLimits.Get(ctx, h.GroupId, userId, subj)
				switch errLimGet {
				case nil:
					if limitMax > limitCurr.Count {
						errLimSet := h.SvcLimits.Set(ctx, h.GroupId, userId, subj, limitMax, noExpiration)
						switch errLimSet {
						case nil:
							_ = tgCtx.Send(fmt.Sprintf("Limit has been set to %d: %s", limitMax, subj.Description()), telebot.ModeHTML)
						default:
							err = errors.Join(err, fmt.Errorf("failed to set the %s limit for %s to %d: %w", subj.Description(), userId, limitMax, errLimGet))
						}
					}
				default:
					err = errors.Join(err, fmt.Errorf("failed to get the %s limit for %s: %w", subj.Description(), userId, errLimGet))
				}
			default:
				member := &telebot.ChatMember{
					User: user,
				}
				errRemove := bot.Ban(chat, member)
				if errRemove == nil {
					errRemove = bot.Unban(chat, user)
				}
				switch errRemove {
				case nil:
					_ = tgCtx.Send(
						fmt.Sprintf(
							"You have been removed from the channel with limit of %s: %d, because you are a member of another channel with limit of %s: %d",
							subj.Description(),
							limit,
							subj.Description(),
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

func (h PaidChatMemberHandler) memberChannels(
	bot *telebot.Bot,
	user *telebot.User,
	limitByChatId map[int64]int64,
) (
	chanByLimit map[int64]*telebot.Chat,
	limitMax int64,
	err error,
) {
	chanByLimit = make(map[int64]*telebot.Chat)
	for chatId, limit := range limitByChatId {
		chat := &telebot.Chat{
			ID: chatId,
		}
		member, errMember := bot.ChatMemberOf(chat, user)
		if errMember != nil {
			err = errors.Join(err, fmt.Errorf("failed to check if a user %d is member of chat %d: %w", user.ID, chatId, errMember))
		}
		if member.Role == telebot.Member {
			chanByLimit[limit] = chat
			if limit > limitMax {
				limitMax = limit
			}
		}
	}
	return
}
