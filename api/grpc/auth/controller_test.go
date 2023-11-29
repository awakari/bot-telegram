package auth

import (
	"context"
	"fmt"
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
		c := NewController("6668123457:ZAJALGCBOGw8q9k2yBidb6kepmrBVGOrBLb")
		RegisterServiceServer(srv, c)
		reflection.Register(srv)
		grpc_health_v1.RegisterHealthServer(srv, health.NewServer())
		conn, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
		if err == nil {
			err = srv.Serve(conn)
		}
		if err != nil {
			log.Error("", err)
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
		dataCheckString string
		hash            string
		err             error
	}{
		"ok": {
			dataCheckString: "auth_date=1701244713\nfirst_name=John\nid=123456789\nlast_name=Doe\nphoto_url=https://t.me/i/userpic/210/qBxcMsnvkqf2zmUdzvQp4Zo9VLy1_4CWfgM0CzTtDSo.jpg\nusername=john_doe",
			hash:            "a63aba9f5f6b1abce3aacb5f9e6e3acc2a5da984602ac3609ec8b41be9134e09",
		},
		"fail": {
			dataCheckString: "auth_date=1701244713\nfirst_name=John\nid=123456789\nlast_name=Doe\nphoto_url=https://t.me/i/userpic/210/qBxcMsnvkqf2zmUdzvQp4Zo9VLy1_4CWfgM0CzTtDSo.jpg\nusername=john_doe",
			hash:            "b63aba9f5f6b1abce3aacb5f9e6e3acc2a5da984602ac3609ec8b41be9134e09",
			err:             status.Error(codes.Unauthenticated, "hash mismatch: expected b63aba9f5f6b1abce3aacb5f9e6e3acc2a5da984602ac3609ec8b41be9134e09, calculated a63aba9f5f6b1abce3aacb5f9e6e3acc2a5da984602ac3609ec8b41be9134e09"),
		},
		"empty": {
			err: status.Error(codes.Unauthenticated, "hash mismatch: expected , calculated d63ae722ab5283f3764edfa1ee005f4e2e33ced57466b08a93738ae45994fc89"),
		},
	}
	//
	for k, c := range cases {
		t.Run(k, func(t *testing.T) {
			req := ValidateRequest{
				DataCheckString: c.dataCheckString,
				Hash:            c.hash,
			}
			_, err = client.Validate(context.TODO(), &req)
			assert.ErrorIs(t, err, c.err)
		})
	}
}
