package chats

import "time"

type Chat struct {
	Key     Key
	GroupId string
	UserId  string
	State   State
	Expires time.Time
}
