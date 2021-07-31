package main

import (
	"fmt"
	"log"
	"net"
	"google.golang.org/grpc"
	pb "grpc-ping-server/ping"
)

type PingMsgService struct {
	pb.UnimplementedPingServiceServer
}

var conn *grpc.ClientConn
func main() {
	fmt.Println("GRPC Ping Server Example!")

	port := "50051"

	listener, err := net.Listen("tcp", ":"+port)
	if err != nil {
		log.Fatalf("net.Listen: %v", err)
	}
	
	fmt.Println("GRPC Server listening at - ", port)

	grpcServer := grpc.NewServer()
	pb.RegisterPingServiceServer(grpcServer, &pb.PingService{})
	if err = grpcServer.Serve(listener); err != nil {
		log.Fatal(err)
	}
	
	fmt.Println("GRPC RegisterPingServiceServer .... ")
}