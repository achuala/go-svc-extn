package extn

import (
	"log/slog"
	"strconv"

	"github.com/go-kratos/kratos/contrib/otel/v3/tracing"
	"github.com/go-kratos/kratos/v3/middleware"
	"github.com/go-kratos/kratos/v3/middleware/metadata"
	"github.com/go-kratos/kratos/v3/middleware/recovery"
	"github.com/go-kratos/kratos/v3/transport/grpc"
	"github.com/go-kratos/kratos/v3/transport/http"
	"go.opentelemetry.io/contrib/propagators/b3"
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

func NewGrpcService(port int, logger *slog.Logger, mw []middleware.Middleware) (*grpc.Server, func(), error) {
	_ = logger

	b3Propagator := b3.New(b3.WithInjectEncoding(b3.B3MultipleHeader | b3.B3SingleHeader))

	defaultMiddlewares := []middleware.Middleware{
		recovery.Recovery(),
		metadata.Server(),
		tracing.Server(tracing.WithPropagator(b3Propagator)),
	}
	allMiddlewares := append(defaultMiddlewares, mw...)

	var opts = []grpc.ServerOption{
		grpc.Middleware(allMiddlewares...),
		grpc.Address(":" + strconv.Itoa(port)),
	}
	srv := grpc.NewServer(opts...)

	return srv, func() {
		srv.GracefulStop()
	}, nil
}
