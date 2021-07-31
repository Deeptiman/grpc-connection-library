package grpc

import (
	"google.golang.org/grpc"
)

type Options func(*GRPC)

func WithConnectionType(connectionType ConnectionType) Options {
	return func(g *GRPC) {
		g.connectionType = connectionType
	}
}

func WithAddress(address string) Options {
	return func(g *GRPC) {
		g.serverAddress = address
	}
}

func WithPort(port string) Options {
	return func(g *GRPC) {
		g.serverPort = port
	}
}

func WithServerOptions(serverOption ...grpc.ServerOption) Options {
	return func(g *GRPC) {
		g.serverOptions = serverOption
	}
}
