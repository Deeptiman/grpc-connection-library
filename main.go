package main

import (
	"flag"
	"fmt"
)

func main() {

	fmt.Println("grpc-connection-library")

	var connectionType, port string
	flag.StringVar(&connectionType, "connectiontype", "client", "Connection Type Server/Client")
	flag.StringVar(&port, "port", "9000", "Server address")

	flag.Parse()

	if connectionType == "client" {
		if _, err := NewGRPCConnection(WithPort(port), WithConnectionType(Client)); err != nil {
			fmt.Println("Failed to create GRPC Connection - ", err.Error())
			return
		}
	} else {
		if _, err := NewGRPCConnection(WithPort(port), WithConnectionType(Server)); err != nil {
			fmt.Println("Failed to create GRPC Connection - ", err.Error())
			return
		}
	}

	fmt.Println("GRPC Connection Established !")
}
