package main

import (
	"google.golang.org/grpc"
)

type Options func(*GRPC)

func WithConnectionType(connectionType ConnectionType) Options {
	return func(g *GRPC) {
		g.connectionType = connectionType
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
