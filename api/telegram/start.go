package telegram

import (
	"errors"
	"fmt"
	"gopkg.in/telebot.v3"
)

const msgStartPrivate = `
• Send a text to publish a simple message.
• Send the <pre>/list</pre> command to list own subscriptions.
• Send a command like: 
  <pre>/sub &lt;sub_name&gt; &lt;word1&gt; &lt;word2&gt; ...</pre>
  to create a simple text matching subscription.
• For advanced usage, use the keyboard buttons.
`

const LabelMsgSend = "New Message"
const LabelSubCreate = "New Subscription"
const LabelUsageQuota = "My Usage Quota"

var ErrChatType = errors.New("unsupported chat type (supported options: \"private\")")

var btnMsgNewCustom = telebot.Btn{
	Text: LabelMsgSend,
	WebApp: &telebot.WebApp{
		URL: "https://awakari.app/msg-new.html",
	},
}

var btnSubNewCustom = telebot.Btn{
	Text: LabelSubCreate,
	WebApp: &telebot.WebApp{
		URL: "https://awakari.app/sub-new.html",
	},
}

var btnUsageQuota = telebot.Btn{
	Text: LabelUsageQuota,
	WebApp: &telebot.WebApp{
		URL: "https://awakari.app/usage-quota.html",
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
	err = ctx.Bot().SetMenuButton(ctx.Sender(), &telebot.MenuButton{
		Type: telebot.MenuButtonCommands,
		Text: "☰",
	})
	if err == nil {
		m := &telebot.ReplyMarkup{ResizeKeyboard: true}
		m.Reply(
			m.Row(btnMsgNewCustom),
			m.Row(btnSubNewCustom),
			m.Row(btnUsageQuota),
		)
		err = ctx.Send(msgStartPrivate, m, telebot.ModeHTML)
	}
	return
}
