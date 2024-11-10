package http

import (
	"context"
	"time"

	extnmw "github.com/achuala/go-svc-extn/pkg/extn/middleware"
	kjson "github.com/go-kratos/kratos/v2/encoding/json"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware/recovery"
	"github.com/go-kratos/kratos/v2/middleware/tracing"
	khttp "github.com/go-kratos/kratos/v2/transport/http"
	"go.opentelemetry.io/contrib/propagators/b3"
	"google.golang.org/protobuf/encoding/protojson"
)

type HttpClient struct {
	Conn *khttp.Client
}

type HttpClientConfig struct {
	Endpoint string
	Timeout  time.Duration
}

func NewHttpClient(ctx context.Context, httpClientCfg HttpClientConfig, logger log.Logger) (*HttpClient, error) {
	kjson.MarshalOptions = protojson.MarshalOptions{
		UseProtoNames: true,
	}
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
