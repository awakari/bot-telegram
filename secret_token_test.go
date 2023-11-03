package main

import (
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"math/rand"
	"testing"
)

func TestSecretToken(t *testing.T) {
	for i := 0; i < 100; i++ {
		secret := make([]byte, binary.MaxVarintLen64)
		binary.PutUvarint(secret, rand.Uint64())
		fmt.Println(base64.URLEncoding.EncodeToString(secret[:6]))
	}
}
