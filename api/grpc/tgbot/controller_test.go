package tgbot

import (
	"context"
	"fmt"
	"github.com/awakari/bot-telegram/service/messages"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"
	"log/slog"
	"net"
	"os"
	"testing"
)

const port = 56789

var log = slog.Default()

func TestMain(m *testing.M) {
	go func() {
		srv := grpc.NewServer()
		c := NewController(
			[]byte("6668123457:ZAJALGCBOGw8q9k2yBidb6kepmrBVGOrBLb"),
			messages.ChanPostHandler{},
			nil,
			"",
			slog.Default(),
			nil,
			messages.Format{},
		)
		RegisterServiceServer(srv, c)
		reflection.Register(srv)
		grpc_health_v1.RegisterHealthServer(srv, health.NewServer())
		conn, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
		if err == nil {
			err = srv.Serve(conn)
		}
		if err != nil {
			log.Error(err.Error())
		}
	}()
	code := m.Run()
	os.Exit(code)
}

func TestController_Validate(t *testing.T) {
	//
	addr := fmt.Sprintf("localhost:%d", port)
	conn, err := grpc.Dial(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.Nil(t, err)
	client := NewServiceClient(conn)
	//
	cases := map[string]struct {
		token string
		err   error
	}{
		"fail": {
			token: `{
				"id": 123456789,
				"first_name": "ел",
				"last_name": "",
				"username": "el",
				"auth_date": 1703271842,
				"photo_url": "https://t.me/i/userpic/321/eZwlVwBo7HPBjQVUYv91UGeeKSFoXBbnt28fwa1Htsg.png",
				"hash": "d88c2dd8f3147bb82559fd554fc88c4c4ae49febda9f8d6c97227401aaeff7ef"
			}`,
			err: status.Error(codes.Unauthenticated, "invalid telegram creds"),
		},
		"invalid json": {
			token: "https://t.me/i/userpic/321/eZwlVwBo7HPBjQVUYv91UGeeKSFoXBbnt28fwa1Htsg.png",
			err:   status.Error(codes.Unauthenticated, "\"Syntax error at index 1: invalid char\\n\\n\\thttps://t.me/i/userpic/321/eZwlV\\n\\t.^..............................\\n\""),
		},
		"ok": {
			token: `{
				"id": 123456789,
				"first_name": "ел",
				"last_name": "",
				"username": "el",
				"auth_date": 1703271842,
				"photo_url": "https://t.me/i/userpic/321/eZwlVwBo7HPBjQVUYv91UGeeKSFoXBbnt28fwa1Htsg.png",
				"hash": "ef86665c59767a5dcecbcf4d427a9708577ea3d65d4cc5c4422abef876849170"
			}`,
		},
		"empty": {
			err: status.Error(codes.Unauthenticated, "\"Syntax error no sources available, the input json is empty: errors.SyntaxError{Pos:0, Src:\\\"\\\", Code:0x1, Msg:\\\"\\\"}\""),
		},
	}
	//
	for k, c := range cases {
		t.Run(k, func(t *testing.T) {
			req := AuthenticateRequest{
				Data: []byte(c.token),
			}
			_, err = client.Authenticate(context.TODO(), &req)
			assert.ErrorIs(t, err, c.err)
		})
	}
}
