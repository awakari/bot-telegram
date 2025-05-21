package reader

import (
	"net/url"
	"strconv"
)

type Subscription struct {
	Url    string `json:"url"`
	Format string `json:"fmt"`
}

func MakeCallbackUrl(urlBase string, chatId int64, userId string) (u string) {
	u = urlBase + "/" + strconv.FormatInt(chatId, 10)
	if userId != "" {
		u += "?userId=" + url.QueryEscape(userId)
	}
	return
}
