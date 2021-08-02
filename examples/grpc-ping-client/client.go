package main

import (
	"fmt"
	grpc "github.com/Deeptiman/grpc-connection-library/grpc"
)

func main() {
	address := "localhost:50051"
	client, err := grpc.NewGRPCConnection(grpc.WithAddress(address), grpc.WithConnectionType(grpc.Client))
	if err != nil {
		fmt.Println("Failed to create GRPC Connection - ", err.Error())
		return
	}
	client.TestGetConn()

	fmt.Println("GRPC Client Connected .... ")
}
