package usage

import (
	"fmt"
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

func HandleNewQuotaReply(paymentProviderToken string) func(tgCtx telebot.Context, awakariClient api.Client, groupId string, args ...string) (err error) {
	return func(tgCtx telebot.Context, awakariClient api.Client, groupId string, args ...string) (err error) {
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
		delta := limNew - limOld
		if delta > 0 {
			price := int(delta)
			invoice := telebot.Invoice{
				Title:       fmt.Sprintf("%s: quota increase", formatSubject(subj)),
				Description: fmt.Sprintf("From %d to %d", limOld, limNew),
				Payload:     "payload0",
				Currency:    "EUR",
				Prices: []telebot.Price{{
					Amount: price,
					Label:  fmt.Sprintf("%s: quota increase", formatSubject(subj)),
				}},
				Token:     paymentProviderToken,
				Total:     price,
				NeedName:  true,
				NeedEmail: true,
				SendEmail: true,
			}
			err = tgCtx.Send("Invoice", &invoice)
		}
		return
	}
}
