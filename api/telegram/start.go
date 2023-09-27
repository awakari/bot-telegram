package telegram

import (
	"errors"
	"fmt"
	"gopkg.in/telebot.v3"
)

const msgStartPrivate = `
• Send a text to submit a simple message to Awakari.
• Send a command to create a simple text matching subscription:
	<pre>/sub &lt;sub_name&gt; &lt;keyword1&gt; &lt;keyword2&gt; ...</pre>
• To customize these options, choose a button below.
`

const LabelWebAppSubCreate = "New Custom Subscription"
const LabelWebAppMsgSend = "New Custom Message"

var ErrChatType = errors.New("unsupported chat type (supported options: \"private\")")

var btnSubNewCustom = telebot.Btn{
	Data:   "subCreateCustom",
	Unique: "subCreateCustomUnique",
	Text:   LabelWebAppSubCreate,
	WebApp: &telebot.WebApp{
		URL: "https://awakari.app/sub-new.html",
	},
}

var btnMsgNewCustom = telebot.Btn{
	Data:   "msgSendCustom",
	Unique: "msgSendCustomUnique",
	Text:   LabelWebAppMsgSend,
	WebApp: &telebot.WebApp{
		URL: "https://awakari.app/msg-new.html",
	},
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
	m := &telebot.ReplyMarkup{ResizeKeyboard: true}
	m.Reply(
		m.Row(btnSubNewCustom),
		m.Row(btnMsgNewCustom),
	)
	err = ctx.Send(msgStartPrivate, m, telebot.ModeHTML)
	return
}
