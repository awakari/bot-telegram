package sources

import (
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/awakari/bot-telegram/config"
	"github.com/awakari/bot-telegram/service"
	"github.com/google/uuid"
	"github.com/mmcdole/gofeed"
	"gopkg.in/telebot.v3"
	"net/http"
	"time"
)

const srcTypeTgCh = "tgch"
const srcTypeFeed = "feed"
const limitCountMax = 1_440
const daysMax = 3_652
const priceMax = 10_000
const feedFetchTimeout = 1 * time.Minute
const PurposeSrcAdd = "src_add"
const srcAddrLenMax = 64

type addPayload struct {
	Limit srcLimit `json:"limit"`
	Price srcPrice `json:"price"`
	Src   src      `json:"src"`
}

type srcLimit struct {
	TimeDays uint16 `json:"timeDays"`
	Count    uint16 `json:"count"`
}

type srcPrice struct {
	Total float64 `json:"total"`
	Unit  string  `json:"unit"`
}

type src struct {
	Addr string `json:"addr"`
	Type string `json:"type"`
}

type addOrder struct {
	Limit srcLimit `json:"limit"`
	Src   src      `json:"src"`
}

var errInvalidAddPayload = errors.New("invalid add source payload")

func (ap addPayload) validate(cfgPayment config.PaymentConfig, bot *telebot.Bot) (err error) {
	if err == nil && (ap.Limit.Count < 1 || ap.Limit.Count > limitCountMax) {
		err = fmt.Errorf("%w: count limit is %d, should in the range of 1..%d", errInvalidAddPayload, ap.Limit.Count, limitCountMax)
	}
	if err == nil && (ap.Limit.TimeDays < 1 || ap.Limit.Count > daysMax) {
		err = fmt.Errorf("%w: time in days is %d, should in the range of 1..%d", errInvalidAddPayload, ap.Limit.TimeDays, daysMax)
	}
	if err == nil && (ap.Price.Total < 1 || ap.Price.Total > 10_000) {
		err = fmt.Errorf("%w: total price is %f, should in the range of 1..%d", errInvalidAddPayload, ap.Price.Total, priceMax)
	}
	if err == nil && ap.Price.Unit != cfgPayment.Currency.Code {
		err = fmt.Errorf("%w: currency is %s, should be %s", errInvalidAddPayload, ap.Price.Unit, cfgPayment.Currency.Code)
	}
	if err == nil && len(ap.Src.Addr) > srcAddrLenMax {
		err = fmt.Errorf("%w: source address too long: %s, should not be more than %d", errInvalidAddPayload, ap.Src.Addr, srcAddrLenMax)
	}
	switch ap.Src.Type {
	case srcTypeTgCh:
		var chat *telebot.Chat
		chat, err = bot.ChatByUsername(ap.Src.Addr)
		if err == nil && chat.Type != telebot.ChatChannel {
			err = fmt.Errorf("%w: telegram chat type is %s, should be %s", errInvalidAddPayload, chat.Type, telebot.ChatChannel)
		}
	case srcTypeFeed:
		clientHttp := http.Client{
			Timeout: feedFetchTimeout,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: true,
				},
			},
		}
		var resp *http.Response
		resp, err = clientHttp.Get(ap.Src.Addr)
		if err == nil {
			defer resp.Body.Close()
			_, err = gofeed.NewParser().Parse(resp.Body)
		}
	default:
		err = fmt.Errorf("%w: unrecognized source type %s", errInvalidAddPayload, ap.Src.Type)
	}
	return
}

func AddInvoiceHandlerFunc(cfgPayment config.PaymentConfig, kbd *telebot.ReplyMarkup) service.ArgHandlerFunc {
	return func(tgCtx telebot.Context, args ...string) (err error) {
		var ap addPayload
		err = json.Unmarshal([]byte(args[0]), &ap)
		if err == nil {
			err = ap.validate(cfgPayment, tgCtx.Bot())
		}
		var orderPayloadData []byte
		if err == nil {
			orderPayloadData, err = json.Marshal(addOrder{
				Limit: ap.Limit,
				Src:   ap.Src,
			})
		}
		var orderData []byte
		if err == nil {
			o := service.Order{
				Purpose: PurposeSrcAdd,
				Payload: base64.URLEncoding.EncodeToString(orderPayloadData),
			}
			orderData, err = json.Marshal(o)
		}
		fmt.Printf("Order data: %s\n", orderData)
		if err == nil {
			label := fmt.Sprintf("Source: %s", ap.Src.Addr)
			price := int(ap.Price.Total * cfgPayment.Currency.SubFactor)
			invoice := telebot.Invoice{
				Start:       uuid.NewString(),
				Title:       fmt.Sprintf("Add custom source for %d days", ap.Limit.TimeDays),
				Description: label,
				Payload:     string(orderData),
				Currency:    cfgPayment.Currency.Code,
				Prices: []telebot.Price{
					{
						Label:  label,
						Amount: price,
					},
				},
				Token: cfgPayment.Provider.Token,
				Total: price,
			}
			err = tgCtx.Send("To proceed, please pay the below invoice", kbd)
			_, err = tgCtx.Bot().Send(tgCtx.Sender(), &invoice)
		}
		return
	}
}
