package sources

import (
	"gopkg.in/telebot.v3"
)

type ExtendHandler struct {
}

const CmdExtend = "src_extend"

func (eh ExtendHandler) RequestInput(tgCtx telebot.Context, args ...string) (err error) {
	//url := args[0]
	//_ = tgCtx.Send(fmt.Sprintf("Reply the number of days to extend (%d-%d):", daysMin, daysMax))
	//err = tgCtx.Send(
	//    fmt.Sprintf("%s %s", ReqSrcExtend, subId),
	//    &telebot.ReplyMarkup{
	//        ForceReply:  true,
	//        Placeholder: strconv.Itoa(usage.ExpiresDefaultDays),
	//    },
	//)
	return
}
