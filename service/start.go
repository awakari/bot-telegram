package service

import (
	"errors"
	"fmt"
	"github.com/awakari/client-sdk-go/model/usage"
	"gopkg.in/telebot.v3"
	"time"
)

const fmtUsageLimit = `<pre>Usage:
  Count:     %d
  Limit:     %d
    Type:    %s
    Expires: %s
</pre>`

const msgStartPrivate = "Use the keyboard buttons."

const LabelSubList = "Subscriptions"
const LabelSubCreateBasic = "+ Basic"
const LabelSubCreateCustom = "+ Custom"
const LabelMsgDetails = "Messages Publishing"
const LabelMsgSendBasic = "▸ Basic"
const LabelMsgSendCustom = "▸ Custom"
const LabelLimitIncrease = "▲ Limit"

var ErrChatType = errors.New("unsupported chat type (supported options: \"private\")")

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

var btnUsageLimitsExtend = telebot.Btn{
	Text: LabelLimitIncrease,
	WebApp: &telebot.WebApp{
		URL: "https://awakari.app/price-calc.html",
	},
}

func GetReplyKeyboard() (kbd *telebot.ReplyMarkup) {
	kbd = &telebot.ReplyMarkup{}
	kbd.Reply(
		kbd.Row(btnSubList),
		kbd.Row(btnSubNewBasic, btnSubNewCustom, btnUsageLimitsExtend),
		kbd.Row(btnMsgs),
		kbd.Row(btnMsgNewBasic, btnMsgNewCustom, btnUsageLimitsExtend),
	)
	return
}

func StartHandlerFunc() telebot.HandlerFunc {
	return func(ctx telebot.Context) (err error) {
		chat := ctx.Chat()
		switch chat.Type {
		case telebot.ChatPrivate:
			err = startPrivate(ctx)
		default:
			err = fmt.Errorf("%w: %s", ErrChatType, chat.Type)
		}
		return
	}
}

func startPrivate(ctx telebot.Context) (err error) {
	if err == nil {
		m := GetReplyKeyboard()
		err = ctx.Send(msgStartPrivate, m, telebot.ModeHTML)
	}
	return
}

func FormatUsageLimit(u usage.Usage, l usage.Limit) (txt string) {
	var t string
	switch l.UserId {
	case "":
		t = "group"
	default:
		t = "user"
	}
	var expires string
	switch l.Expires.IsZero() {
	case true:
		expires = "never"
	default:
		expires = l.Expires.Format(time.RFC3339)
	}
	txt = fmt.Sprintf(fmtUsageLimit, u.Count, l.Count, t, expires)
	return
}
