package service

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestGetLoginCode(t *testing.T) {
	cases := map[string]struct {
		in  string
		out string
		err error
	}{
		"ok": {
			in:  "Login code: 12345. Do not give t...",
			out: "12345",
		},
		"invalid": {
			in:  "Login code:-123. Do not give t...",
			err: ErrInvalidLoginCodeMsg,
		},
		"multiple": {
			in:  "Login code: 12345. Login code: 23456.",
			out: "12345",
		},
		"empty": {
			err: ErrInvalidLoginCodeMsg,
		},
	}
	for k, c := range cases {
		t.Run(k, func(t *testing.T) {
			code, err := GetLoginCode(c.in)
			assert.Equal(t, c.out, code)
			assert.ErrorIs(t, err, c.err)
		})
	}
}
