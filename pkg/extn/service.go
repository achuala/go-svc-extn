package extn

import (
	"strconv"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware"
	"github.com/go-kratos/kratos/v2/middleware/metadata"
	"github.com/go-kratos/kratos/v2/middleware/recovery"
	"github.com/go-kratos/kratos/v2/middleware/tracing"
	"github.com/go-kratos/kratos/v2/transport/grpc"
	"github.com/go-kratos/kratos/v2/transport/http"
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

func NewGrpcService(port int, logger log.Logger, mw []middleware.Middleware) (*grpc.Server, func(), error) {
	// Set up B3 Propagator
	b3Propagator := b3.New(b3.WithInjectEncoding(b3.B3MultipleHeader | b3.B3SingleHeader))

	// Default middlewares
	defaultMiddlewares := []middleware.Middleware{
		recovery.Recovery(),
		metadata.Server(),
		tracing.Server(tracing.WithPropagator(b3Propagator)),
	}
	// Combine default middlewares with custom middlewares
	allMiddlewares := append(defaultMiddlewares, mw...)

	// gRPC server options
	var opts = []grpc.ServerOption{
		grpc.Middleware(allMiddlewares...),
		grpc.Address(":" + strconv.Itoa(port)),
	}
	// Create gRPC server
	srv := grpc.NewServer(opts...)

	// Register all provided services
	for _, registerService := range cfg.Services {
		registerService(srv)
	}

	// Return server and shutdown function
	return srv, func() {
		srv.GracefulStop()
	}, nil
}
