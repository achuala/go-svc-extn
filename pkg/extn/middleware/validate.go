package middleware

import (
	"context"
	"strconv"
	"strings"

	"buf.build/gen/go/bufbuild/protovalidate/protocolbuffers/go/buf/validate"
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
			errMeta[fieldPathString(violation.Proto.Field.Elements)] = *violation.Proto.Message
		}
	} else {
		errMeta["message"] = err.Error()
	}
	return errors.BadRequest("VALIDATION_FAILED", "request validation failed").WithMetadata(errMeta)
}

func fieldPathString(path []*validate.FieldPathElement) string {
	var result strings.Builder
	for i, element := range path {
		if i > 0 {
			result.WriteByte('.')
		}
		result.WriteString(element.GetFieldName())
		subscript := element.GetSubscript()
		if subscript == nil {
			continue
		}
		result.WriteByte('[')
		switch value := subscript.(type) {
		case *validate.FieldPathElement_Index:
			result.WriteString(strconv.FormatUint(value.Index, 10))
		case *validate.FieldPathElement_BoolKey:
			result.WriteString(strconv.FormatBool(value.BoolKey))
		case *validate.FieldPathElement_IntKey:
			result.WriteString(strconv.FormatInt(value.IntKey, 10))
		case *validate.FieldPathElement_UintKey:
			result.WriteString(strconv.FormatUint(value.UintKey, 10))
		case *validate.FieldPathElement_StringKey:
			result.WriteString(strconv.Quote(value.StringKey))
		}
		result.WriteByte(']')
	}
	return result.String()
}
