package telegram

import (
	"errors"
	"fmt"
	"gopkg.in/telebot.v3"
)

const msgStartPrivate = `
• Send a message to publish a simple text to Awakari.
• Send the following command to create a simple text matching subscription:
	<pre>/sub &lt;sub_name&gt; &lt;word1&gt; &lt;word2&gt; ...</pre>
• For advanced usage, use the keyboard buttons.
`

const LabelWebAppSubCreate = "New Subscription"
const LabelWebAppMsgSend = "New Message"

var ErrChatType = errors.New("unsupported chat type (supported options: \"private\")")

var btnSubNewCustom = telebot.Btn{
	Text: LabelWebAppSubCreate,
	WebApp: &telebot.WebApp{
		URL: "https://awakari.app/sub-new.html",
	},
}

var btnMsgNewCustom = telebot.Btn{
	Text: LabelWebAppMsgSend,
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
		m.Row(btnMsgNewCustom, btnSubNewCustom),
	)
	err = ctx.Send(msgStartPrivate, m, telebot.ModeHTML)
	return
}
