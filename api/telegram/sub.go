package telegram

import (
	"errors"
	"fmt"
	"gopkg.in/telebot.v3"
	"regexp"
	"strings"
)

const CmdPrefixSubCreateSimplePrefix = "/sub"
const argSep = " "

var ErrCreateSubNotEnoughArgs = errors.New("not enough arguments to create a text subscription")

var whiteSpaceRegex = regexp.MustCompile(`\p{Zs}+`)

func CreateTextSubscription(ctx telebot.Context) (err error) {
	txt := ctx.Text()
	argStr, ok := strings.CutPrefix(txt, CmdPrefixSubCreateSimplePrefix+" ")
	if !ok {
		err = ErrCreateSubNotEnoughArgs
	}
	var args []string
	if err == nil {
		argStr = whiteSpaceRegex.ReplaceAllString(argStr, argSep)
		args = strings.Split(argStr, argSep)
	}
	if len(args) < 2 {
		err = ErrCreateSubNotEnoughArgs
	}
	var name string
	var keywords []string
	if err == nil {
		name = args[0]
		keywords = args[1:]
		fmt.Printf("Create a simpe subscription with name \"%s\" and keywords \"%+v\"\n", name, keywords)
	}
	return
}
