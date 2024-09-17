package source_telegram

import (
	"context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type Service interface {
	Login(ctx context.Context, code int64, replicaIdx uint) (success bool, err error)
}

type service struct {
	uri string
}

func NewService(uri string) Service {
	return service{
		uri: uri,
	}
}

func (svc service) Login(ctx context.Context, code int64, replicaIdx uint) (success bool, err error) {
	req := &LoginRequest{
		Code:         uint32(code),
		ReplicaIndex: uint32(replicaIdx),
	}
	creds := grpc.WithTransportCredentials(insecure.NewCredentials())
	var replicaMatch bool
	for {
		replicaMatch, success, err = svc.loginOnce(ctx, req, creds)
		if err != nil {
			break
		}
		if replicaMatch {
			break
		}
	}
	return
}

func (svc service) loginOnce(ctx context.Context, req *LoginRequest, creds grpc.DialOption) (replicaMatch, success bool, err error) {
	var conn *grpc.ClientConn
	conn, err = grpc.NewClient(svc.uri, creds)
	var resp *LoginResponse
	if err == nil {
		defer conn.Close()
		client := NewServiceClient(conn)
		resp, err = client.Login(ctx, req)
	}
	if err == nil {
		replicaMatch = resp.ReplicaMatch
		success = resp.Success
	}
	return
}
