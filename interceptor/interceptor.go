package interceptor

import (
	"context"
	"github.com/Deeptiman/grpc-connection-library/retry"
	"google.golang.org/grpc"
)

func UnaryClientInterceptor(retryOpts *retry.RetryOption) grpc.UnaryClientInterceptor {

	return func(parentCtx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		var err error
		for attempt := 1; attempt <= retryOpts.Retry; attempt++ {

			if err := retryOpts.RetryBackoff(attempt); err != nil {
				return err
			}

			err = invoker(parentCtx, method, req, reply, cc, opts...)
			if err == nil {
				return nil
			}

			if retry.IsContextError(err) {
				if parentCtx.Err() != nil {
					return err
				}
			}

			if !retry.IsRetriable(err, retryOpts.Codes) {
				return err
			}
			retryOpts.Backoff.ApplyBackoffDuration(attempt)
		}

		return err
	}
}
