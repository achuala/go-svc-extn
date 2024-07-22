package serverextn

import (
	"github.com/go-kratos/kratos/v2/transport/grpc"
	"github.com/go-kratos/kratos/v2/transport/http"
)

type ApiService interface {
	RegisterGrpc(server *grpc.Server)
	RegisterHttp(server *http.Server)
}

func RegisterServices(grpcServer *grpc.Server, httpServer *http.Server, services ...ApiService) {
	for _, service := range services {
		service.RegisterGrpc(grpcServer)
		service.RegisterHttp(httpServer)
	}
}

func NewGrpcService()
