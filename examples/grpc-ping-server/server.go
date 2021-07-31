package main

import (
	"fmt"
	grpc "grpc-connection-library/grpc"
)

func main() {
	fmt.Println("GRPC Ping Server Example!")

	port := "50051"

	server, err := grpc.NewGRPCConnection(grpc.WithPort(port), grpc.WithConnectionType(grpc.Server))
	if err != nil {
		fmt.Println("Failed to create GRPC Connection - ", err.Error())
		return
	}
	server.ListenAndServe()
}