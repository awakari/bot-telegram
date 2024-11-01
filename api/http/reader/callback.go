package reader

import (
	"strconv"
)

type Callback struct {
	Url    string `json:"url"`
	Format string `json:"fmt"`
}

func MakeCallbackUrl(urlBase string, chatId int64) string {
	return urlBase + "/" + strconv.FormatInt(chatId, 10)
}
