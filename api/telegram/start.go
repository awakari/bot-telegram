package telegram

import (
	"errors"
	"fmt"
	"gopkg.in/telebot.v3"
)

const msgStartPrivate = `
• Send a text to publish a simple message.
• Send a command like: 
  <pre>/sub &lt;sub_name&gt; &lt;word1&gt; &lt;word2&gt; ...</pre>
  to create a simple text matching subscription.
• Send the <pre>/list</pre> command to list own subscriptions.
• Send the <pre>/usage</pre> command to see own usage limits.
• For advanced usage, use the keyboard buttons.
`

const LabelSubCreate = "+ Advanced"
const LabelMsgSend = "🖂 Advanced"
const LabelUsageLimitsExtend = "⬆ Extend Limits"

var ErrChatType = errors.New("unsupported chat type (supported options: \"private\")")

var btnSubList = telebot.Btn{
	Text: "Subscriptions",
}

var btnSubNewBasic = telebot.Btn{
	Text: "+ Basic",
}

var btnSubNewCustom = telebot.Btn{
	Text: LabelSubCreate,
	WebApp: &telebot.WebApp{
		URL: "https://awakari.app/sub-new.html",
	},
}

var btnMsgs = telebot.Btn{
	Text: "Messages",
}

var btnMsgNewBasic = telebot.Btn{
	Text: "+ Basic",
}

var btnMsgNewCustom = telebot.Btn{
	Text: LabelMsgSend,
	WebApp: &telebot.WebApp{
		URL: "https://awakari.app/msg-new.html",
	},
}

var btnUsageLimitsExtend = telebot.Btn{
	Text: LabelUsageLimitsExtend,
	WebApp: &telebot.WebApp{
		URL: "https://awakari.app/price-calc.html",
	},
}

func GetReplyKeyboard() (kbd *telebot.ReplyMarkup) {
	kbd = &telebot.ReplyMarkup{
		ResizeKeyboard: true,
	}
	kbd.Reply(
		kbd.Row(btnSubList, btnSubNewBasic, btnSubNewCustom),
		kbd.Row(btnMsgs, btnMsgNewBasic, btnMsgNewCustom),
		kbd.Row(btnUsageLimitsExtend),
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
