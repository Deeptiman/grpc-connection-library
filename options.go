package main

import (
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
)

type GRPCOption struct {
	connectionType ConnectionType
	address        string
	host           string
	port           string
	authority      string
	insecure       bool
	scheme         string
	poolSize       uint64
	retryOption    RetryOption
	serverOptions  []grpc.ServerOption
	credentials    credentials.TransportCredentials
	interceptor    ConnectionInterceptor
}

type RetryOption struct {
	retry   int
	backoff Backoff
	codes   []codes.Code
}

type Options func(*GRPC)

func WithMaxConnPoolSize(size uint64) Options {
	return func(g *GRPC) {
		g.options.poolSize = size
	}
}

func WithConnectionType(connectionType ConnectionType) Options {
	return func(g *GRPC) {
		g.options.connectionType = connectionType
	}
}

func WithAddress(address string) Options {
	return func(g *GRPC) {
		g.options.address = address
	}
}

func WithHost(host string) Options {
	return func(g *GRPC) {
		g.options.host = host
	}
}

func WithPort(port string) Options {
	return func(g *GRPC) {
		g.options.port = port
	}
}

func WithConnectionInterceptor(interceptor ConnectionInterceptor) Options {
	return func(g *GRPC) {
		g.options.interceptor = interceptor
	}
}

func WithRetryOption(retry int, backoff Backoff) Options {
	return func(g *GRPC) {
		g.options.retryOption = RetryOption{retry, backoff, RetriableCodes}
	}
}

func WithMaxRetry(retry int) Options {
	return func(g *GRPC) {
		g.options.retryOption.retry = retry
	}
}

func WithBackoff(backoff Backoff) Options {
	return func(g *GRPC) {
		g.options.retryOption.backoff = backoff
	}
}

func WithScheme(scheme string) Options {
	return func(g *GRPC) {
		g.options.scheme = scheme
	}
}

func WithTransportCredentials(credentials credentials.TransportCredentials) Options {
	return func(g *GRPC) {
		g.options.credentials = credentials
	}
}

func WithServerOptions(serverOption ...grpc.ServerOption) Options {
	return func(g *GRPC) {
		g.options.serverOptions = serverOption
	}
}
