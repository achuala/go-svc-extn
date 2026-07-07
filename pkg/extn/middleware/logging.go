package middleware

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/achuala/go-svc-extn/gen/go/options"
	"github.com/go-kratos/kratos/v3/errors"
	"github.com/go-kratos/kratos/v3/middleware"
	"github.com/go-kratos/kratos/v3/transport"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
)

type Redacter interface {
	Redact() string
}

// Server is a server logging middleware.
func Server(logger *slog.Logger) middleware.Middleware {
	if logger == nil {
		logger = slog.Default()
	}
	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req any) (reply any, err error) {
			return logMiddleware(ctx, req, handler, logger, "server")
		}
	}
}

// Client is a client logging middleware.
func Client(logger *slog.Logger) middleware.Middleware {
	if logger == nil {
		logger = slog.Default()
	}
	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req any) (reply any, err error) {
			return logMiddleware(ctx, req, handler, logger, "client")
		}
	}
}

func logMiddleware(ctx context.Context, req any, handler middleware.Handler, logger *slog.Logger, kind string) (reply any, err error) {
	var (
		code      int32
		reason    string
		operation string
		component string
	)
	startTime := time.Now()

	switch kind {
	case "server":
		if info, ok := transport.FromServerContext(ctx); ok {
			component = info.Kind().String()
			operation = info.Operation()
		}
	case "client":
		if info, ok := transport.FromClientContext(ctx); ok {
			component = info.Kind().String()
			operation = info.Operation()
		}
	}

	reply, err = handler(ctx, req)

	rid := getCorrelationIdFromCtx(ctx)

	if se := errors.FromError(err); se != nil {
		code = se.Code
		reason = se.Reason
	}

	level, stack := extractError(err)

	attrs := []slog.Attr{
		slog.String("kind", kind),
		slog.String("component", component),
		slog.String("op", operation),
		slog.String("req", extractArgs(req)),
		slog.Int64("code", int64(code)),
		slog.Float64("latency", time.Since(startTime).Seconds()),
	}
	if rid != "" {
		attrs = append(attrs, slog.String("rid", rid))
	}
	if stack != "" {
		attrs = append(attrs, slog.String("stack", stack))
	}
	if reason != "" {
		attrs = append(attrs, slog.String("reason", reason))
	}
	if reply != nil {
		attrs = append(attrs, slog.String("resp", extractArgs(reply)))
	}
	if err != nil {
		attrs = append(attrs, slog.Any("error", err))
	}

	logger.LogAttrs(ctx, level, kind+" request", attrs...)
	return
}

var jsonOpts = &protojson.MarshalOptions{
	EmitUnpopulated: false,
	UseProtoNames:   true,
	UseEnumNumbers:  false,
}

// extractArgs returns the string representation of the req
func extractArgs(req any) string {
	switch v := req.(type) {
	case proto.Message:
		clone := proto.Clone(v)
		handleSensitiveData(clone.ProtoReflect())
		json, err := jsonOpts.Marshal(clone)
		if err != nil {
			return fmt.Sprintf("%v", clone)
		}
		return string(json)
	case Redacter:
		return v.Redact()
	case fmt.Stringer:
		return v.String()
	default:
		return fmt.Sprintf("%v", req)
	}
}

// extractError returns the log level and string representation of the error
func extractError(err error) (slog.Level, string) {
	if err != nil {
		return slog.LevelError, fmt.Sprintf("%+v", err)
	}
	return slog.LevelInfo, ""
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
	return "****" + value[len(value)-4:]
}
