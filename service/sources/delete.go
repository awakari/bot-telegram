package sources

import (
	"context"
	"errors"
	"fmt"
	"github.com/awakari/bot-telegram/api/grpc/source/feeds"
	"gopkg.in/telebot.v3"
	"strconv"
	"strings"
)

const CmdDelete = "src_del_req"
const CmdDeleteConfirm = "src_del"

type DeleteHandler struct {
	SvcSrcFeeds feeds.Service
	RestoreKbd  *telebot.ReplyMarkup
	GroupId     string
}

func (dh DeleteHandler) RequestConfirmation(tgCtx telebot.Context, args ...string) (err error) {
	url := args[0]
	_ = tgCtx.Send("Are you sure? Reply \"yes\" or \"no\" to the next message:", url)
	err = tgCtx.Send(
		fmt.Sprintf("%s %s", CmdDeleteConfirm, url),
		&telebot.ReplyMarkup{
			ForceReply:  true,
			Placeholder: "no",
		},
	)
	return
}

func (dh DeleteHandler) HandleConfirmation(tgCtx telebot.Context, args ...string) (err error) {
	if len(args) != 3 {
		err = errors.New("invalid argument count")
	}
	url, reply := args[1], strings.ToLower(args[2])
	switch reply {
	case "yes":
		userId := strconv.FormatInt(tgCtx.Sender().ID, 10)
		err = dh.SvcSrcFeeds.Delete(context.TODO(), url, dh.GroupId, userId)
		if err == nil {
			err = tgCtx.Send("Source feed deleted", dh.RestoreKbd)
		}
	default:
		err = tgCtx.Send("Subscription deletion cancelled", dh.RestoreKbd)
	}
	return
}
