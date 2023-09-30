package usage

import (
	"fmt"
	"github.com/awakari/bot-telegram/api/telegram"
	"github.com/awakari/client-sdk-go/api"
	"github.com/awakari/client-sdk-go/model/usage"
	"gopkg.in/telebot.v3"
	"strconv"
)

const CmdQuotaReq = "quotareq"

const ReplyQuotaSet = "quotaset"

func RequestNewQuota(tgCtx telebot.Context, args ...string) (err error) {
	var subjCode int
	subjCode, err = strconv.Atoi(args[0])
	if err == nil {
		_ = tgCtx.Send("Please enter the new quota (non-negative integer):")
		err = tgCtx.Send(
			fmt.Sprintf("quotaset %d %s", subjCode, args[1]),
			&telebot.ReplyMarkup{
				ForceReply:  true,
				Placeholder: args[1],
			},
		)
	}
	return
}

func HandleNewQuotaReply(tgCtx telebot.Context, awakariClient api.Client, groupId string, args ...string) (err error) {
	var subjCode int
	subjCode, err = strconv.Atoi(args[0])
	var subj usage.Subject
	var limOld uint64
	if err == nil {
		subj = usage.Subject(subjCode)
		limOld, err = strconv.ParseUint(args[1], 10, 64)
	}
	var limNew uint64
	if err == nil {
		limNew, err = strconv.ParseUint(args[2], 10, 64)
	}
	_ = tgCtx.Send(fmt.Sprintf("Change %s limit from %d to %d", subj, limOld, limNew), telegram.GetReplyKeyboard())
	return
}
