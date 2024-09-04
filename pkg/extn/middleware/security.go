package middleware

import (
	"context"

	"github.com/achuala/go-svc-extn/pkg/util/idgen"
	"github.com/go-kratos/kratos/v2/errors"
	"github.com/go-kratos/kratos/v2/middleware"
	"github.com/go-kratos/kratos/v2/transport"
)

// CtxKey is a custom type for context keys
type CtxKey string

// Context keys for various headers
const (
	CtxCorrelationIdKey CtxKey = "x-correlation-id"
	CtxSystemPeerKey    CtxKey = "x-system-peer"
	CtxSignedHeadersKey CtxKey = "x-signed-headers"
	CtxAuthorizationKey CtxKey = "Authorization"
)

// getCorrelationIdFromCtx retrieves the correlation ID from the context or generates a new one
func getCorrelationIdFromCtx(ctx context.Context) string {
	if correlationId, ok := ctx.Value(CtxCorrelationIdKey).(string); ok {
		return correlationId
	}
	return idgen.NewId()
}

// ServerSecurityHeaderValidator middleware validates the presence of required security headers
func ServerSecurityHeaderValidator() middleware.Middleware {
	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req interface{}) (reply interface{}, err error) {
			if tr, ok := transport.FromServerContext(ctx); ok {
				authHeader := tr.RequestHeader().Get(string(CtxAuthorizationKey))
				signatureHeader := tr.RequestHeader().Get(string(CtxSignedHeadersKey))
				if authHeader == "" || signatureHeader == "" {
					return nil, errors.Unauthorized("UNAUTHORIZED", "missing authorization or signature headers")
				}
			}
			return handler(ctx, req)
		}
	}
}

// ServerCorrelationIdInjector middleware injects the correlation ID into the server context
func ServerCorrelationIdInjector() middleware.Middleware {
	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req interface{}) (reply interface{}, err error) {
			if tr, ok := transport.FromServerContext(ctx); ok {
				correlationId := tr.RequestHeader().Get(string(CtxCorrelationIdKey))
				if correlationId == "" {
					correlationId = idgen.NewId()
				}
				ctx = context.WithValue(ctx, CtxCorrelationIdKey, correlationId)
				ctx = transport.NewServerContext(ctx, tr)
			}
			return handler(ctx, req)
		}
	}
}

// ClientCorrelationIdInjector middleware injects the correlation ID into the client request header
func ClientCorrelationIdInjector() middleware.Middleware {
	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req interface{}) (reply interface{}, err error) {
			if tr, ok := transport.FromClientContext(ctx); ok {
				correlationId := getCorrelationIdFromCtx(ctx)
				tr.RequestHeader().Set(string(CtxCorrelationIdKey), correlationId)
			}
			return handler(ctx, req)
		}
	}
}
