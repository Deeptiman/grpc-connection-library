package main

import (
	"fmt"
)

func main() {

	fmt.Println("grpc-connection-library")

	if _, err := NewGRPCConnection(WithAddress("dns:///localhost:9000")); err != nil {
		fmt.Println("Failed to create GRPC Connection")
	}

	fmt.Println("GRPC Connection Established !")
}