package middleware

import (
	"context"

	"github.com/achuala/go-svc-extn/pkg/util/idgen"
	"github.com/go-kratos/kratos/v2/errors"
	"github.com/go-kratos/kratos/v2/middleware"
	"github.com/go-kratos/kratos/v2/transport"
)

type CtxKey string

const CtxCorrelationIdKey = CtxKey("x-correlation-id")
const CtxSystemPeerKey = CtxKey("x-system-peer")
const CtxSignedHeadersKey = CtxKey("x-signed-headers")
const CtxAuthorizationKey = CtxKey("Authorization")

func getCorrelationIdFromCtx(ctx context.Context) string {
	correlationId := ctx.Value(CtxCorrelationIdKey)
	if correlationId == nil {
		return idgen.NewId()
	}
	return correlationId.(string)
}

func ServerSecurityHeaderValidator() middleware.Middleware {
	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req interface{}) (reply interface{}, err error) {
			if tr, ok := transport.FromServerContext(ctx); ok {
				authHeader := tr.RequestHeader().Get(string(CtxAuthorizationKey))
				signatureHeader := tr.RequestHeader().Get(string(CtxSignedHeadersKey))
				if len(authHeader) == 0 || len(signatureHeader) == 0 {
					return nil, errors.Unauthorized("UNAUTHORIZED", "missing authorization/signature headers")
				}
			}
			return handler(ctx, req)
		}
	}
}

func ServerCorrelationIdInjector() middleware.Middleware {
	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req interface{}) (reply interface{}, err error) {
			if tr, ok := transport.FromServerContext(ctx); ok {
				correlationId := tr.RequestHeader().Get(string(CtxCorrelationIdKey))
				ctx = transport.NewServerContext(context.WithValue(ctx, CtxCorrelationIdKey, correlationId), tr)
			}
			return handler(ctx, req)
		}
	}
}

func ClientCorrelationIdInjector() middleware.Middleware {

	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req interface{}) (reply interface{}, err error) {
			if tr, ok := transport.FromClientContext(ctx); ok {
				tr.RequestHeader().Set(string(CtxCorrelationIdKey), getCorrelationIdFromCtx(ctx))
			}
			return handler(ctx, req)
		}
	}
}
