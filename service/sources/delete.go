package sources

import (
	"context"
	"errors"
	"fmt"
	"github.com/awakari/bot-telegram/api/grpc/source/feeds"
	"github.com/awakari/bot-telegram/api/grpc/source/telegram"
	"github.com/awakari/bot-telegram/service/support"
	"gopkg.in/telebot.v3"
	"strconv"
	"strings"
)

const CmdDelete = "src_del_req"
const CmdDeleteConfirm = "src_del"

type DeleteHandler struct {
	SvcSrcFeeds    feeds.Service
	SvcSrcTg       telegram.Service
	RestoreKbd     *telebot.ReplyMarkup
	GroupId        string
	SupportHandler support.Handler
}

func (dh DeleteHandler) RequestConfirmation(tgCtx telebot.Context, args ...string) (err error) {
	url := args[0]
	_ = tgCtx.Send("Are you sure? Reply \"yes\" or \"no\" to the next message:")
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
		err = dh.delete(tgCtx, url)
	default:
		err = tgCtx.Send("Subscription deletion cancelled", dh.RestoreKbd)
	}
	return
}

func (dh DeleteHandler) delete(tgCtx telebot.Context, url string) (err error) {
	userId := strconv.FormatInt(tgCtx.Sender().ID, 10)
	switch {
	case strings.HasPrefix(url, tgChPubLinkPrefix):
		err = dh.SvcSrcTg.Delete(context.TODO(), url)
		_ = dh.SupportHandler.Request(
			tgCtx,
			fmt.Sprintf("User %s deleted the source telegram channel: %s", userId, url),
		)
	default:
		err = dh.SvcSrcFeeds.Delete(context.TODO(), url, dh.GroupId, userId)
	}
	if err == nil {
		err = tgCtx.Send("Source feed deleted", dh.RestoreKbd)
	}
	return
}
