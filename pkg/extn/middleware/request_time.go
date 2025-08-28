package middleware

import (
	"context"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware"
)

// Define a private type for our context key to avoid collisions.
type requestTimeKey struct{}

// InjectRequestTime is a Kratos middleware that injects the current time into the context.
func InjectRequestTime() middleware.Middleware {
	return func(handler middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req any) (reply any, err error) {
			// Get the time as soon as the request comes in.
			startTime := time.Now()

			// Store it in the context with our private key.
			ctx = context.WithValue(ctx, requestTimeKey{}, startTime)

			// Call the next handler in the chain.
			return handler(ctx, req)
		}
	}
}

// GetRequestTime retrieves the injected time from the context.
// It returns the zero value for time.Time if the key is not found.
func GetRequestTime(ctx context.Context) time.Time {
	t, ok := ctx.Value(requestTimeKey{}).(time.Time)
	if !ok {
		log.Warn("request time not found in context, use InjectRequestTime middleware as part of the middleware chain")
		return time.Now()
	}
	return t
}
