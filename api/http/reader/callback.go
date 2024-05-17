package reader

import (
	"fmt"
	"strconv"
	"strings"
)

type Callback struct {
	Url    string `json:"url"`
	Format string `json:"fmt"`
}

func MakeCallbackUrl(urlBase string, chatId int64) string {
	return urlBase + "/" + strconv.FormatInt(chatId, 10)
}

func GetCallbackUrlChatId(cbUrl string) (chatId int64, err error) {
	cbUrlParts := strings.Split(cbUrl, "/")
	cbUrlPartsLen := len(cbUrlParts)
	if cbUrlPartsLen < 1 {
		err = fmt.Errorf("invalid callback url")
	}
	if err == nil {
		chatIdStr := cbUrlParts[cbUrlPartsLen-1]
		chatId, err = strconv.ParseInt(chatIdStr, 10, 64)
	}
	return
}
