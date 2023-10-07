package service

import (
	"errors"
	"fmt"
	"github.com/awakari/bot-telegram/service/chats"
	"github.com/awakari/bot-telegram/service/messages"
	"github.com/awakari/bot-telegram/service/subscriptions"
	"github.com/awakari/client-sdk-go/api"
	"gopkg.in/telebot.v3"
	"log/slog"
)

const msgStartPrivate = "Use the reply keyboard buttons."

const LabelSubList = "Subscriptions"
const LabelSubCreateBasic = "+ Basic"
const LabelSubCreateCustom = "+ Custom"
const LabelMsgDetails = "Messages Publishing"
const LabelMsgSendBasic = "▸ Basic"
const LabelMsgSendCustom = "▸ Custom"
const LabelLimitIncrease = "▲ Limit"

var ErrChatType = errors.New("unsupported chat type (supported options: \"private\", \"group\")")

var btnSubList = telebot.Btn{
	Text: LabelSubList,
}

var btnSubNewBasic = telebot.Btn{
	Text: LabelSubCreateBasic,
}

var btnSubNewCustom = telebot.Btn{
	Text: LabelSubCreateCustom,
	WebApp: &telebot.WebApp{
		URL: "https://awakari.app/sub-new.html",
	},
}

var btnMsgs = telebot.Btn{
	Text: LabelMsgDetails,
}

var btnMsgNewBasic = telebot.Btn{
	Text: LabelMsgSendBasic,
}

var btnMsgNewCustom = telebot.Btn{
	Text: LabelMsgSendCustom,
	WebApp: &telebot.WebApp{
		URL: "https://awakari.app/msg-new.html",
	},
}

var btnLimitIncrSubs = telebot.Btn{
	Text: LabelLimitIncrease,
	WebApp: &telebot.WebApp{
		URL: "https://awakari.app/price-calc-subs.html",
	},
}

var btnLimitIncrMsgs = telebot.Btn{
	Text: LabelLimitIncrease,
	WebApp: &telebot.WebApp{
		URL: "https://awakari.app/price-calc-msgs.html",
	},
}

func MakeReplyKeyboard() (kbd *telebot.ReplyMarkup) {
	kbd = &telebot.ReplyMarkup{}
	kbd.Reply(
		kbd.Row(btnSubList),
		kbd.Row(btnSubNewBasic, btnSubNewCustom, btnLimitIncrSubs),
		kbd.Row(btnMsgs),
		kbd.Row(btnMsgNewBasic, btnMsgNewCustom, btnLimitIncrMsgs),
	)
	return
}

func StartHandlerFunc(
	log *slog.Logger,
	clientAwk api.Client,
	chatStor chats.Storage,
	groupId string,
	msgFmt messages.Format,
	kbd *telebot.ReplyMarkup,
) telebot.HandlerFunc {
	return func(tgCtx telebot.Context) (err error) {
		chat := tgCtx.Chat()
		switch chat.Type {
		case telebot.ChatPrivate:
			err = tgCtx.Send(msgStartPrivate, kbd, telebot.ModeHTML)
		case telebot.ChatGroup:
			err = subscriptions.Start(tgCtx, log, clientAwk, chatStor, groupId, msgFmt)
		default:
			err = fmt.Errorf("%w: %s", ErrChatType, chat.Type)
		}
		return
	}
}
