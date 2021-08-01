package main

import (
	"fmt"
	grpc "grpc-connection-library/grpc"
)

func main() {
	fmt.Println("GRPC Ping Server Example!")

	server, err := grpc.NewGRPCConnection(grpc.WithConnectionType(grpc.Server))
	if err != nil {
		fmt.Println("Failed to create GRPC Connection - ", err.Error())
		return
	}
	server.ListenAndServe()
}
