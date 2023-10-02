package messages

import (
	"context"
	"fmt"
	"github.com/awakari/bot-telegram/service"
	"github.com/awakari/client-sdk-go/api"
	"github.com/awakari/client-sdk-go/model/usage"
	"google.golang.org/grpc/metadata"
	"gopkg.in/telebot.v3"
	"strconv"
	"time"
)

func DetailsHandlerFunc(awakariClient api.Client, groupId string) telebot.HandlerFunc {
	return func(tgCtx telebot.Context) (err error) {
		//
		groupIdCtx := metadata.AppendToOutgoingContext(context.TODO(), "x-awakari-group-id", groupId)
		userId := strconv.FormatInt(tgCtx.Sender().ID, 10)
		//
		var respTxt string
		var u usage.Usage
		u, err = awakariClient.ReadUsage(groupIdCtx, userId, usage.SubjectPublishEvents)
		var l usage.Limit
		if err == nil {
			l, err = awakariClient.ReadUsageLimit(groupIdCtx, userId, usage.SubjectPublishEvents)
		}
		if err == nil {
			var t string
			switch l.UserId {
			case "":
				t = "default"
			default:
				t = "custom"
			}
			var expires string
			switch l.Expires.IsZero() {
			case true:
				expires = "&lt;not set&gt;"
			default:
				expires = l.Expires.Format(time.RFC3339)
			}
			respTxt += fmt.Sprintf(service.UsageLimitsFmt, u.Count, l.Count, t, expires)
			err = tgCtx.Send(respTxt, telebot.ModeHTML)
		}
		//
		return
	}
}
