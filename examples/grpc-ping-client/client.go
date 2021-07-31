package main

import (
	"flag"
	"log"
	"fmt"
	"os"
	"google.golang.org/grpc"
	pb "grpc-ping-client/ping"
)

var (
	logger                             = log.New(os.Stdout, "", 0)
	serverAddr, serverHost, message    string
	insecure, skipVerify, sendUpstream bool
)

func main() {

	//flag.StringVar(&serverAddr, "server", "", "Server address (host:port)")
	flag.StringVar(&serverHost, "server-host", "", "Host name to which server IP should resolve")
	flag.BoolVar(&insecure, "insecure", false, "Skip SSL validation? [false]")
	flag.BoolVar(&skipVerify, "skip-verify", false, "Skip server hostname verification in SSL validation [false]")
	flag.StringVar(&message, "message", "Hi there", "The body of the content sent to server")
	flag.BoolVar(&sendUpstream, "relay", false, "Direct ping to relay the request to a ping-upstream service [false]")

	flag.Parse()

	log.Println("Insecure --- ", insecure)
	log.Println("Relay ---- ", sendUpstream)
	log.Println("ServerHost ---- ", serverHost)

	var opts []grpc.DialOption
	if serverHost != "" {
		opts = append(opts, grpc.WithAuthority(serverHost))
	}

	// if insecure {
	// 	opts = append(opts, grpc.WithInsecure())
	// } else {
	// 	cred := credentials.NewTLS(&tls.Config{
	// 		InsecureSkipVerify: skipVerify,
	// 	})
	// 	opts = append(opts, grpc.WithTransportCredentials(cred))
	// }

	opts = append(opts, grpc.WithInsecure())

	serverAddr = "localhost:9000"

	log.Println("Dial GRPC Server  -- ", serverAddr)

	conn, err := grpc.Dial(serverAddr, opts...)
	if err != nil {
		logger.Printf("Failed to dial: %v", err)
	}
	log.Println("GRPC Connected  at - address : ", serverAddr," : ConnState = ", conn.GetState())
	defer conn.Close()
	client := pb.NewPingServiceClient(conn)

	err = pb.SendPingMsg(client)
	if err != nil {
		fmt.Printf(" failed connect with address - %s err - %v", serverAddr, err)
		return 
	}
	fmt.Println("GRPC Client Connected .... ")
}