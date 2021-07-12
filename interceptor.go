package main

import (
	"context"
	"fmt"

	"google.golang.org/grpc"
)

func UnaryClientInterceptor(retryOpts RetryOption) grpc.UnaryClientInterceptor {

	fmt.Println("Unary Client Interceptor --- ")

	return func(parentCtx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {

		fmt.Println("Unary Client Interceptor Function Call --- ")

		var err error
		for attempt := 1; attempt <= retryOpts.retry; attempt++ {

			err = invoker(parentCtx, method, req, reply, cc, opts...)
			if err == nil {
				return nil
			}

			if err != nil {

				if isContextError(err) {
					return err
				}

				if !isRetriable(err, retryOpts) {
					return err
				}
			}

			fmt.Println("grpc-retry : attempt = ", attempt)

			retryOpts.backoff.ApplyBackoffDuration(attempt)
		}

		return err
	}
}
