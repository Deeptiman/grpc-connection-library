package pool

import (
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"grpc-connection-library/retry"
)

type PoolConnOptions struct {
	address     string
	port        string
	interceptor ConnectionInterceptor
	retryOption *retry.RetryOption
	credentials credentials.TransportCredentials
	authority   string
	insecure    bool
	scheme      string
	connBatch   uint64
}

type PoolOptions func(*ConnPool)

func WithAddress(address string) PoolOptions {
	return func(c *ConnPool) {
		c.Options.address = address
	}
}

func WithAuthority(authority string) PoolOptions {
	return func(c *ConnPool) {
		c.Options.authority = authority
	}
}

func WithInsecure(insecure bool) PoolOptions {
	return func(c *ConnPool) {
		c.Options.insecure = insecure
	}
}

func WithScheme(scheme string) PoolOptions {
	return func(c *ConnPool) {
		c.Options.scheme = scheme
	}
}

func WithConnectionInterceptor(interceptor ConnectionInterceptor) PoolOptions {
	return func(c *ConnPool) {
		c.Options.interceptor = interceptor
	}
}

func WithRetry(retry int) PoolOptions {
	return func(c *ConnPool) {
		c.Options.retryOption.Retry = retry
	}
}

func WithCodes(codes []codes.Code) PoolOptions {
	return func(c *ConnPool) {
		c.Options.retryOption.Codes = codes
	}
}

func WithCredentials(credentials credentials.TransportCredentials) PoolOptions {
	return func(c *ConnPool) {
		c.Options.credentials = credentials
	}
}
