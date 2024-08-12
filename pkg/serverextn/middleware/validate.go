package middleware

import (
	"context"

	"github.com/bufbuild/protovalidate-go"
	"google.golang.org/protobuf/reflect/protoreflect"

	"github.com/go-kratos/kratos/v2/errors"
	"github.com/go-kratos/kratos/v2/middleware"
)

// Validator is a validator middleware.
func Validator() middleware.Middleware {
	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req interface{}) (reply interface{}, err error) {
			if msg, ok := req.(protoreflect.ProtoMessage); ok {
				if v, err := protovalidate.New(); err == nil {
					if err = v.Validate(msg); err != nil {
						errMeta := make(map[string]string)
						if pvErr, ok := err.(*protovalidate.ValidationError); ok {
							for _, violation := range pvErr.Violations {
								errMeta[violation.FieldPath] = violation.Message
							}
						} else {
							errMeta["message"] = err.Error()
						}
						return nil, errors.BadRequest("FAILED_VALIDATION", "request validation failed").WithMetadata(errMeta)
					}
				} else {
					return nil, errors.BadRequest("VALIDATOR_INIT_FAILED", err.Error()).WithCause(err)
				}
			}
			return handler(ctx, req)
		}
	}
}
