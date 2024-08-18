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

// Server is an server logging middleware.
func Server(logger log.Logger) middleware.Middleware {
	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req interface{}) (reply interface{}, err error) {
			var (
				code      int32
				reason    string
				kind      string
				operation string
			)
			startTime := time.Now()
			if info, ok := transport.FromServerContext(ctx); ok {
				kind = info.Kind().String()
				operation = info.Operation()
			}
			reply, err = handler(ctx, req)
			if se := errors.FromError(err); se != nil {
				code = se.Code
				reason = se.Reason
			}
			level, stack := extractError(err)
			_ = log.WithContext(ctx, logger).Log(level,
				"kind", "server",
				"component", kind,
				"op", operation,
				"req", extractArgs(req),
				"resp", extractArgs(reply),
				"code", code,
				"reason", reason,
				"stack", stack,
				"latency", time.Since(startTime).Seconds(),
			)
			return
		}
	}
}

// Client is a client logging middleware.
func Client(logger log.Logger) middleware.Middleware {
	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req interface{}) (reply interface{}, err error) {
			var (
				code      int32
				reason    string
				kind      string
				operation string
			)
			startTime := time.Now()
			if info, ok := transport.FromClientContext(ctx); ok {
				kind = info.Kind().String()
				operation = info.Operation()
			}
			reply, err = handler(ctx, req)
			if se := errors.FromError(err); se != nil {
				code = se.Code
				reason = se.Reason
			}
			level, stack := extractError(err)
			_ = log.WithContext(ctx, logger).Log(level,
				"kind", "client",
				"component", kind,
				"op", operation,
				"req", extractArgs(req),
				"resp", extractArgs(reply),
				"code", code,
				"reason", reason,
				"stack", stack,
				"latency", time.Since(startTime).Seconds(),
			)
			return
		}
	}
}

// extractArgs returns the string of the req
func extractArgs(req interface{}) string {
	if protoMsg, ok := req.(proto.Message); ok {
		clone := proto.Clone(protoMsg)
		handleSenstiveData(clone.ProtoReflect())
		return fmt.Sprintf("%+v", clone)
	} else if redacter, ok := req.(Redacter); ok {
		return redacter.Redact()
	} else if stringer, ok := req.(fmt.Stringer); ok {
		return stringer.String()
	}
	return fmt.Sprintf("%+v", req)
}

// extractError returns the string of the error
func extractError(err error) (log.Level, string) {
	if err != nil {
		return log.LevelError, fmt.Sprintf("%+v", err)
	}
	return log.LevelInfo, ""
}

func handleSenstiveData(m protoreflect.Message) {
	m.Range(func(fd protoreflect.FieldDescriptor, v protoreflect.Value) bool {
		opts := fd.Options().(*descriptorpb.FieldOptions)

		switch typed := v.Interface().(type) {
		case protoreflect.Message:
			handleSenstiveData(typed)
		case protoreflect.Map:
			typed.Range(func(key protoreflect.MapKey, value protoreflect.Value) bool {
				if _, ok := value.Interface().(protoreflect.Message); ok {
					handleSenstiveData(value.Message())
				}
				if _, ok := key.Interface().(protoreflect.Message); ok {
					handleSenstiveData(key.Value().Message())
				}
				return true
			})
		case protoreflect.List:
			for i := 0; i < typed.Len(); i++ {
				if _, ok := typed.Get(i).Interface().(protoreflect.Message); ok {
					handleSenstiveData(typed.Get(i).Message())
				}
			}
		}

		// Get extension from field
		ext := proto.GetExtension(opts, options.E_Sensitive)
		// Check if equal to bool as expected
		extVal, ok := ext.(*options.Sensitive)
		if !ok {
			return true
		}

		// If true clear field and move on
		if extVal != nil {
			if extVal.GetRedact() || extVal.Pii {
				m.Clear(fd)
			} else if extVal.GetMask() {
				maskedValue := maskString(v.String())
				m.Set(fd, protoreflect.ValueOfString(maskedValue))
			}
		}

		return true
	})

}

func maskString(value string) string {
	if len(value) <= 4 {
		// If the string length is less than or equal to 4, just return "****" for masking.
		return "****"
	}

	// Mask all characters except the last 4 with "*".
	maskedValue := strings.Repeat("*", len(value)-4) + value[len(value)-4:]
	return maskedValue
}
