package chats

import "errors"

var ErrAlreadyExists = errors.New("chat already exists")
var ErrNotFound = errors.New("chat or subscription not found")
var ErrInternal = errors.New("internal failure")
