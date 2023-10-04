package subscriptions

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/awakari/bot-telegram/service"
	"gopkg.in/telebot.v3"
	"strconv"
)

type ExtendOrder struct {
	SubId   string `json:"subId"`
	DaysAdd uint64 `json:"daysAdd"`
}

const PurposeExtend = "sub_extend"
const CmdExtend = "extend"
const ReqSubExtend = "sub_extend"
const daysMin = 10
const daysMax = 365
const pricePerDay = 0.1

func ExtendReqHandlerFunc() service.ArgHandlerFunc {
	return func(tgCtx telebot.Context, args ...string) (err error) {
		subId := args[0]
		_ = tgCtx.Send(fmt.Sprintf("Reply the number of days to extend (%d-%d):", daysMin, daysMax))
		err = tgCtx.Send(
			fmt.Sprintf("%s %s", ReqSubExtend, subId),
			&telebot.ReplyMarkup{
				ForceReply:  true,
				Placeholder: strconv.Itoa(expiresDefaultDays),
			},
		)
		return
	}
}

func ExtendReplyHandlerFunc(paymentProviderToken string, kbd *telebot.ReplyMarkup) service.ArgHandlerFunc {
	return func(tgCtx telebot.Context, args ...string) (err error) {
		if len(args) != 3 {
			err = errors.New("invalid argument count")
		}
		subId, daysReply := args[1], args[2]
		var countDays uint64
		countDays, err = strconv.ParseUint(daysReply, 10, 16)
		if err == nil {
			if countDays < daysMin || countDays > daysMax {
				err = errors.New(fmt.Sprintf("invalid days count, should be %d-%d", daysMin, daysMax))
			}
		}
		var orderPayloadData []byte
		if err == nil {
			orderPayloadData, err = json.Marshal(ExtendOrder{
				SubId:   subId,
				DaysAdd: countDays,
			})
		}
		var orderData []byte
		if err == nil {
			o := service.Order{
				Purpose: PurposeExtend,
				Payload: string(orderPayloadData),
			}
			orderData, err = json.Marshal(o)
		}
		if err == nil {
			invoice := telebot.Invoice{
				Title:       "Subscription Extension",
				Description: fmt.Sprintf("Add %d days to the subscription %s", countDays, subId),
				Payload:     string(orderData),
				Currency:    service.Currency,
				Prices: []telebot.Price{
					{
						Label:  fmt.Sprintf("Add %d days to the subscription %s", countDays, subId),
						Amount: int(float64(countDays) * pricePerDay * service.SubCurrencyFactor),
					},
				},
				Token: paymentProviderToken,
				Total: int(float64(countDays) * pricePerDay * service.SubCurrencyFactor),
			}
			_, err = tgCtx.Bot().Send(tgCtx.Sender(), &invoice)
		}
		return
	}
}
