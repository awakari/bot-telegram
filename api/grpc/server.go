package grpc

import (
	"fmt"
	grpcAuth "github.com/awakari/bot-telegram/api/grpc/auth"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"
	"net"
)

func Serve(port uint16, controllerAuth grpcAuth.Controller) (err error) {
	srv := grpc.NewServer()
	grpcAuth.RegisterServiceServer(srv, controllerAuth)
	reflection.Register(srv)
	grpc_health_v1.RegisterHealthServer(srv, health.NewServer())
	conn, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err == nil {
		err = srv.Serve(conn)
	}
	return
}