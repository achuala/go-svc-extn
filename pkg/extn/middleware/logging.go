package middleware

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/achuala/go-svc-extn/gen/go/options"
	"github.com/go-kratos/kratos/v2/errors"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware"
	"github.com/go-kratos/kratos/v2/transport"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
)

type Redacter interface {
	Redact() string
}

// Server is a server logging middleware.
func Server(logger log.Logger) middleware.Middleware {
	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req any) (reply any, err error) {
			return logMiddleware(ctx, req, handler, logger, "server")
		}
	}
}

// Client is a client logging middleware.
func Client(logger log.Logger) middleware.Middleware {
	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req any) (reply any, err error) {
			return logMiddleware(ctx, req, handler, logger, "client")
		}
	}
}

func logMiddleware(ctx context.Context, req any, handler middleware.Handler, logger log.Logger, kind string) (reply any, err error) {
	var (
		code      int32
		reason    string
		operation string
		component string
	)
	startTime := time.Now()
	if kind == "server" {
		if info, ok := transport.FromServerContext(ctx); ok {
			component = info.Kind().String()
			operation = info.Operation()
		}
	} else if kind == "client" {
		if info, ok := transport.FromClientContext(ctx); ok {
			component = info.Kind().String()
			operation = info.Operation()
		}
	}
	reply, err = handler(ctx, req)
	if se := errors.FromError(err); se != nil {
		code = se.Code
		reason = se.Reason
	}

	rid := getCorrelationIdFromCtx(ctx)
	level, stack := extractError(err)

	ctxFields := make([]any, 0)
	if rid != "" {
		ctxFields = append(ctxFields, "rid", rid)
	}
	if stack != "" {
		ctxFields = append(ctxFields, "stack", stack)
	}
	if reason != "" {
		ctxFields = append(ctxFields, "reason", reason)
	}
	logFields := append(ctxFields,
		"kind", kind,
		"component", component,
		"op", operation,
		"req", extractArgs(req),
		"resp", extractArgs(reply),
		"code", code,
		"latency", time.Since(startTime).Seconds(),
	)
	_ = log.WithContext(ctx, logger).Log(level, logFields...)
	return
}

// extractArgs returns the string representation of the req
func extractArgs(req any) string {
	switch v := req.(type) {
	case proto.Message:
		clone := proto.Clone(v)
		handleSensitiveData(clone.ProtoReflect())
		return fmt.Sprintf("%+v", clone)
	case Redacter:
		return v.Redact()
	case fmt.Stringer:
		return v.String()
	default:
		return fmt.Sprintf("%+v", req)
	}
}

// extractError returns the log level and string representation of the error
func extractError(err error) (log.Level, string) {
	if err != nil {
		return log.LevelError, fmt.Sprintf("%+v", err)
	}
	return log.LevelInfo, ""
}

func handleSensitiveData(m protoreflect.Message) {
	m.Range(func(fd protoreflect.FieldDescriptor, v protoreflect.Value) bool {
		opts := fd.Options().(*descriptorpb.FieldOptions)

		switch typed := v.Interface().(type) {
		case protoreflect.Message:
			handleSensitiveData(typed)
		case protoreflect.Map:
			typed.Range(func(key protoreflect.MapKey, value protoreflect.Value) bool {
				if msg, ok := value.Interface().(protoreflect.Message); ok {
					handleSensitiveData(msg)
				}
				if msg, ok := key.Interface().(protoreflect.Message); ok {
					handleSensitiveData(msg)
				}
				return true
			})
		case protoreflect.List:
			for i := range typed.Len() {
				if msg, ok := typed.Get(i).Interface().(protoreflect.Message); ok {
					handleSensitiveData(msg)
				}
			}
		}

		ext := proto.GetExtension(opts, options.E_Sensitive)
		extVal, ok := ext.(*options.Sensitive)
		if !ok || extVal == nil {
			return true
		}

		if extVal.GetRedact() || extVal.Pii {
			m.Clear(fd)
		} else if extVal.GetMask() {
			m.Set(fd, protoreflect.ValueOfString(maskString(v.String())))
		}

		return true
	})
}

func maskString(value string) string {
	if len(value) <= 4 {
		return "****"
	}
	return strings.Repeat("*", len(value)-4) + value[len(value)-4:]
}
