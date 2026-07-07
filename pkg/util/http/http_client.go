package http

import (
	"context"
	"log/slog"
	"time"

	extnmw "github.com/achuala/go-svc-extn/pkg/extn/middleware"
	"github.com/go-kratos/kratos/contrib/otel/v3/tracing"
	"github.com/go-kratos/kratos/v3/middleware"
	"github.com/go-kratos/kratos/v3/middleware/recovery"
	khttp "github.com/go-kratos/kratos/v3/transport/http"
	"go.opentelemetry.io/contrib/propagators/b3"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

type HttpClient struct {
	Conn *khttp.Client
}

type HttpClientConfig struct {
	Endpoint string
	Timeout  time.Duration
}

func NewHttpClient(ctx context.Context, httpClientCfg HttpClientConfig, logger *slog.Logger) (*HttpClient, error) {
	b3Propagator := b3.New(b3.WithInjectEncoding(b3.B3MultipleHeader | b3.B3SingleHeader))
	httpClient, err := khttp.NewClient(ctx, khttp.WithEndpoint(httpClientCfg.Endpoint), khttp.WithMiddleware(
		recovery.Recovery(),
		tracing.Client(tracing.WithPropagator(b3Propagator)),
		extnmw.ClientCorrelationIdInjector(),
		extnmw.Client(logger),
	), khttp.WithTimeout(httpClientCfg.Timeout))

	if err != nil {
		return nil, err
	}
	return &HttpClient{Conn: httpClient}, nil
}

func NewHttpClientWithOptions(ctx context.Context, httpClientCfg HttpClientConfig, logger *slog.Logger, options ...khttp.ClientOption) (*HttpClient, error) {
	b3Propagator := b3.New(b3.WithInjectEncoding(b3.B3MultipleHeader | b3.B3SingleHeader))
	allOptions := []khttp.ClientOption{
		khttp.WithEndpoint(httpClientCfg.Endpoint),
		khttp.WithMiddleware(
			recovery.Recovery(),
			tracing.Client(tracing.WithPropagator(b3Propagator)),
			extnmw.ClientCorrelationIdInjector(),
			extnmw.Client(logger),
		),
		khttp.WithTimeout(httpClientCfg.Timeout),
	}
	allOptions = append(allOptions, options...)
	httpClient, err := khttp.NewClient(ctx, allOptions...)

	if err != nil {
		return nil, err
	}
	return &HttpClient{Conn: httpClient}, nil
}

func NewHttpClientWithMiddleware(ctx context.Context, httpClientCfg HttpClientConfig, logger *slog.Logger, customMiddlewares ...middleware.Middleware) (*HttpClient, error) {
	b3Propagator := b3.New(b3.WithInjectEncoding(b3.B3MultipleHeader | b3.B3SingleHeader))
	middlewares := []middleware.Middleware{
		recovery.Recovery(),
		tracing.Client(tracing.WithPropagator(b3Propagator)),
		extnmw.ClientCorrelationIdInjector(),
	}
	middlewares = append(middlewares, customMiddlewares...)
	middlewares = append(middlewares, extnmw.Client(logger))
	httpClient, err := khttp.NewClient(ctx, khttp.WithEndpoint(httpClientCfg.Endpoint), khttp.WithMiddleware(
		middlewares...,
	), khttp.WithTimeout(httpClientCfg.Timeout))
	if err != nil {
		return nil, err
	}
	return &HttpClient{Conn: httpClient}, nil
}

func CustomProtoJSONEncoder(opts protojson.MarshalOptions) khttp.EncodeRequestFunc {
	return func(ctx context.Context, contentType string, in any) ([]byte, error) {
		if pb, ok := in.(proto.Message); ok {
			return opts.Marshal(pb)
		}
		return nil, nil
	}
}
