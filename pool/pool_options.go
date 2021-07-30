package pool

import (
	"google.golang.org/grpc/codes"
	"grpc-connection-library/retry"
)

type PoolConnOptions struct {
	address     string
	host        string
	port        string
	interceptor ConnectionInterceptor
	retryOption *retry.RetryOption
	authority   string
	insecure    bool
	scheme      string
}

type PoolOptions func(*ConnPool)

func WithAddress(address string) PoolOptions {
	return func(c *ConnPool) {
		c.Options.address = address
	}
}

func WithHost(host string) PoolOptions {
	return func(c *ConnPool) {
		c.Options.host = host
	}
}

func WithPort(port string) PoolOptions {
	return func(c *ConnPool) {
		c.Options.port = port
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
