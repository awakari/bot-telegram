package messages

import (
	"context"
	"fmt"
	"github.com/awakari/bot-telegram/service"
	"github.com/awakari/bot-telegram/service/sources"
	"github.com/awakari/bot-telegram/service/usage"
	"github.com/awakari/client-sdk-go/api"
	awkUsage "github.com/awakari/client-sdk-go/model/usage"
	"google.golang.org/grpc/metadata"
	"gopkg.in/telebot.v3"
	"time"
)

type UsageHandler struct {
	ClientAwk api.Client
	GroupId   string
}

const LabelPubAddSource = "+ Own Source"

var btnPubAddSource = telebot.Btn{
	Text: LabelPubAddSource,
	WebApp: &telebot.WebApp{
		URL: "https://awakari.app/source-add.html",
	},
}

func (uh UsageHandler) Show(tgCtx telebot.Context) (err error) {
	groupIdCtx := metadata.AppendToOutgoingContext(context.TODO(), service.KeyGroupId, uh.GroupId)
	userId := fmt.Sprintf(service.FmtUserId, tgCtx.Sender().ID)
	var u awkUsage.Usage
	if err == nil {
		u, err = uh.ClientAwk.ReadUsage(groupIdCtx, userId, awkUsage.SubjectPublishEvents)
	}
	var l awkUsage.Limit
	if err == nil {
		l, err = uh.ClientAwk.ReadUsageLimit(groupIdCtx, userId, awkUsage.SubjectPublishEvents)
	}
	if err == nil {
		m := &telebot.ReplyMarkup{}
		m.Inline(m.Row(telebot.Btn{
			Text: usage.LabelIncrease,
			Data: fmt.Sprintf("%s %d", usage.CmdIncrease, awkUsage.SubjectPublishEvents),
		}))
		err = tgCtx.Send(fmt.Sprintf("Published Today: %d\n(including events from own sources)\nDaily Limit: %d", u.Count, l.Count), m)
	}
	if err == nil {
		m := &telebot.ReplyMarkup{}
		var expires string
		switch {
		case l.Expires.IsZero():
			expires = "never"
		case l.Expires.After(time.Now()):
			m.Inline(m.Row(telebot.Btn{
				Text: usage.LabelExtend,
				Data: fmt.Sprintf("%s %d", usage.CmdExtend, awkUsage.SubjectPublishEvents),
			}))
			expires = l.Expires.Format(time.RFC3339)
		default:
			expires = l.Expires.Format(time.RFC3339)
		}
		err = tgCtx.Send(fmt.Sprintf("Limit Expires: %s", expires), m)
	}
	if err == nil {
		m := &telebot.ReplyMarkup{}
		m.Inline(
			m.Row(
				telebot.Btn{
					Text: "Telegram - All",
					Data: sources.CmdTgChListAll,
				},
				telebot.Btn{
					Text: "Telegram - Own",
					Data: sources.CmdTgChListOwn,
				},
			),
			m.Row(
				telebot.Btn{
					Text: "Feeds - All",
					Data: sources.CmdFeedListAll,
				},
				telebot.Btn{
					Text: "Feeds - Own",
					Data: sources.CmdFeedListOwn,
				},
			),
			m.Row(
				telebot.Btn{
					Text: "Sites - All",
					Data: sources.CmdSitesListAll,
				},
				telebot.Btn{
					Text: "Sites - Own",
					Data: sources.CmdSitesListOwn,
				},
			),
		)
		err = tgCtx.Send("Sources:", m)
	}
	if err == nil {
		m := &telebot.ReplyMarkup{
			ResizeKeyboard: true,
		}
		m.Reply(m.Row(service.BtnMainMenu, btnPubAddSource))
		err = tgCtx.Send("To add own source, use the corresponding reply keyboard button.", m)
	}
	return
}
