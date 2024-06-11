package chats

import "errors"

var ErrAlreadyExists = errors.New("chat already exists")
var ErrNotFound = errors.New("chat or interest not found")
var ErrInternal = errors.New("internal failure")
