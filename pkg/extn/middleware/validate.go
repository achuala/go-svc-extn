package middleware

import (
	"context"

	"github.com/bufbuild/protovalidate-go"
	"google.golang.org/protobuf/reflect/protoreflect"

	"fmt"
	"sync"

	"github.com/go-kratos/kratos/v2/errors"
	"github.com/go-kratos/kratos/v2/middleware"
)

var (
	validator *protovalidate.Validator
	once      sync.Once
)

// Validator is a validator middleware.
func Validator() middleware.Middleware {
	once.Do(func() {
		v, err := protovalidate.New()
		if err != nil {
			panic(fmt.Sprintf("failed to initialize validator: %v", err))
		}
		validator = v
	})

	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req interface{}) (reply interface{}, err error) {
			if msg, ok := req.(protoreflect.ProtoMessage); ok {
				if err = validator.Validate(msg); err != nil {
					return nil, handleValidationError(err)
				}
			}
			return handler(ctx, req)
		}
	}
}

// handleValidationError processes the validation error and returns a formatted error response
func handleValidationError(err error) error {
	errMeta := make(map[string]string)
	if pvErr, ok := err.(*protovalidate.ValidationError); ok {
		for _, violation := range pvErr.Violations {
			errMeta[violation.FieldPath] = violation.Message
		}
	} else {
		errMeta["message"] = err.Error()
	}
	return errors.BadRequest("VALIDATION_FAILED", "request validation failed").WithMetadata(errMeta)
}
