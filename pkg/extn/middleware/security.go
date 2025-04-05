package middleware

import (
	"context"

	"github.com/achuala/go-svc-extn/pkg/util/idgen"
	"github.com/go-kratos/kratos/v2/errors"
	"github.com/go-kratos/kratos/v2/metadata"
	"github.com/go-kratos/kratos/v2/middleware"
	"github.com/go-kratos/kratos/v2/transport"
)

// CtxKey is a custom type for context keys
type CtxKey string

// Context keys for various headers
const (
	CtxCorrelationIdKey   CtxKey = "x-correlation-id"
	CtxSystemPeerKey      CtxKey = "x-system-peer"
	CtxSignedHeadersKey   CtxKey = "x-signed-headers"
	CtxAuthorizationKey   CtxKey = "Authorization"
	CtxRequestIDKey       CtxKey = "rid"
	CtxMdCorrelationIdKey CtxKey = "x-md-global-correlation-id"
	CtxMdRequestIdKey     CtxKey = "x-md-global-request-id"
)

// getCorrelationIdFromCtx retrieves the correlation ID from the context or generates a new one
func getCorrelationIdFromCtx(ctx context.Context) string {
	if correlationId, ok := ctx.Value(CtxCorrelationIdKey).(string); ok {
		return correlationId
	} else if rid, ok := ctx.Value(CtxRequestIDKey).(string); ok {
		return rid
	}
	return idgen.NewId()
}

func getCorrelationIdFromMetadata(ctx context.Context) string {
	if md, ok := metadata.FromServerContext(ctx); ok {
		if values := md.Get(string(CtxMdCorrelationIdKey)); len(values) > 0 {
			return string(values[0])
		} else if values := md.Get(string(CtxMdRequestIdKey)); len(values) > 0 {
			return string(values[0])
		}
	}
	return idgen.NewId()
}

// ServerSecurityHeaderValidator middleware validates the presence of required security headers
func ServerSecurityHeaderValidator() middleware.Middleware {
	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req any) (reply any, err error) {
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
		return func(ctx context.Context, req any) (reply any, err error) {
			if tr, ok := transport.FromServerContext(ctx); ok {
				correlationId := tr.RequestHeader().Get(string(CtxCorrelationIdKey))
				if correlationId == "" {
					correlationId = tr.RequestHeader().Get(string(CtxMdCorrelationIdKey))
					if correlationId == "" {
						correlationId = idgen.NewId()
					}
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
		return func(ctx context.Context, req any) (reply any, err error) {
			if tr, ok := transport.FromClientContext(ctx); ok {
				correlationId := getCorrelationIdFromCtx(ctx)
				tr.RequestHeader().Set(string(CtxCorrelationIdKey), correlationId)
				tr.RequestHeader().Set(string(CtxMdCorrelationIdKey), correlationId)
				ctx = metadata.AppendToClientContext(ctx, string(CtxMdCorrelationIdKey), correlationId)
			}
			return handler(ctx, req)
		}
	}
}
