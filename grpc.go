package main

import (
	"fmt"
	pb "grpc-connection-library/ping"
	pool "grpc-connection-library/pool"
	"io/ioutil"
	"net"
	"os"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/grpclog"
)

type ConnectionType int

type ConnState connectivity.State

var (
	DefaultConnectionType        = Client
	DefaultInsecureState         = true
	DefaultScheme                = "dns"
	DefaultPort                  = "9000"
	DefaultPoolSize       uint64 = 60

	RetriableCodes = []codes.Code{codes.ResourceExhausted, codes.Unavailable}
)

const (
	Server ConnectionType = iota
	Client

	Idle ConnState = iota
	Connecting
	Ready
	TransientFailure
	ShutDown
)

type GRPC struct {
	connectionType ConnectionType
	server         *grpc.Server
	client         *grpc.ClientConn
	pool           *pool.ConnPool
	serverPort     string
	serverOptions  []grpc.ServerOption
	log            grpclog.LoggerV2
}

func NewGRPCConnection(opts ...Options) (*GRPC, error) {

	grpcConn := &GRPC{
		connectionType: DefaultConnectionType,
		pool:           pool.NewConnPool(pool.WithAddress("localhost:9000")),
		log:            grpclog.NewLoggerV2(os.Stdout, ioutil.Discard, ioutil.Discard),
	}

	grpclog.SetLoggerV2(grpcConn.log)

	for _, opt := range opts {
		opt(grpcConn)
	}

	fmt.Println("NewGRPCConnection ! ConnectionType : ", grpcConn.connectionType)

	switch grpcConn.connectionType {
	case Server:
		fmt.Println("Connecting to GRPC Server....")
		return grpcConn, grpcConn.ListenAndServe()
	case Client:
		fmt.Println("Connecting to GRPC Client....")

		grpcConn.GetConn()

		return grpcConn, nil
	}

	return grpcConn, nil
}

func (g *GRPC) ListenAndServe() error {

	port := g.serverPort
	listener, err := net.Listen("tcp", ":"+port)
	if err != nil {
		g.log.Fatalf("GRPC Server listen failed - %v", err)
	}
	fmt.Println("Server Port - ", port)
	serverOptions := g.serverOptions

	grpcServer := grpc.NewServer(serverOptions...)
	pb.RegisterPingServiceServer(grpcServer, &pb.PingService{})
	if err = grpcServer.Serve(listener); err != nil {
		fmt.Println("failed start server - %v", err)
		g.log.Fatal(err)
		return err
	}
	g.server = grpcServer
	g.log.Infoln("GRPC Server listening on - port : ", port)
	return nil
}

func (g *GRPC) GetConn() (*grpc.ClientConn, error) {

	// Testing conn
	connTry := g.pool.MaxPoolSize
	for connTry > 0 {
		batchItem := g.pool.GetConnBatch()
		if batchItem.Item == nil {
			return nil, fmt.Errorf("No Grpc connection instance found")
		}
		fmt.Println("ConnTry=", connTry, " : GRPC Conn -- ", batchItem.Item.(*grpc.ClientConn).GetState().String())
		connTry--
		time.Sleep(1 * time.Second)
	}
	return nil, nil
}

func (g *GRPC) GetGrpcConnectivityState(state ConnState) string {

	switch state {
	case Idle:
		return "IDLE"
	case Connecting:
		return "CONNECTING"
	case Ready:
		return "READY"
	case TransientFailure:
		return "TRANSIENT_FAILURE"
	case ShutDown:
		return "SHUTDOWN"
	default:
		return "Invalid-State"
	}
}