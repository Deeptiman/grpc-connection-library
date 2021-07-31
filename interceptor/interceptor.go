package interceptor

import (
	"context"
	"fmt"
	"google.golang.org/grpc"
	"grpc-connection-library/retry"
)

func UnaryClientInterceptor(retryOpts *retry.RetryOption) grpc.UnaryClientInterceptor {

	fmt.Println("Unary Client Interceptor --- ")

	return func(parentCtx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {

		fmt.Println("Unary Client Interceptor Function Call --- ")

		var err error
		for attempt := 1; attempt <= retryOpts.Retry; attempt++ {

			if err := retryOpts.RetryBackoff(attempt); err != nil {
				return err
			}

			err = invoker(parentCtx, method, req, reply, cc, opts...)
			fmt.Println("Unary Client Interceptor")
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

			fmt.Println("grpc-retry : attempt = ", attempt, " -- err = ", err.Error())

			retryOpts.Backoff.ApplyBackoffDuration(attempt)
		}

		return err
	}
}
