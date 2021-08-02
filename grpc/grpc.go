package grpc

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
	DefaultServerPort            = "50051"
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
	serverAddress  string
	serverPort     string
	serverOptions  []grpc.ServerOption
	pool           *pool.ConnPool
	log            grpclog.LoggerV2
}

func NewGRPCConnection(opts ...Options) (*GRPC, error) {

	grpcConn := &GRPC{
		connectionType: DefaultConnectionType,
		serverPort:     DefaultServerPort,
		log:            grpclog.NewLoggerV2(os.Stdout, ioutil.Discard, ioutil.Discard),
	}

	grpclog.SetLoggerV2(grpcConn.log)

	for _, opt := range opts {
		opt(grpcConn)
	}

	grpcConn.pool = pool.NewConnPool(pool.WithAddress(grpcConn.serverAddress))

	grpcConn.log.Infoln("NewGRPCConnection ! ConnectionType : ", grpcConn.connectionType)

	return grpcConn, nil
}

func (g *GRPC) ListenAndServe() error {

	port := g.serverPort
	listener, err := net.Listen("tcp", ":"+port)
	if err != nil {
		g.log.Fatalf("GRPC Server listen failed - %v", err)
	}
	g.log.Infoln("Server Port - ", port)
	serverOptions := g.serverOptions

	grpcServer := grpc.NewServer(serverOptions...)
	pb.RegisterPingServiceServer(grpcServer, &pb.PingService{})
	if err = grpcServer.Serve(listener); err != nil {
		g.log.Errorln("failed start server - %v", err)
		g.log.Fatal(err)
		return err
	}
	g.server = grpcServer
	g.log.Infoln("GRPC Server listening on - port : ", port)
	return nil
}

func (g *GRPC) GetConn() (*grpc.ClientConn, error) {

	batchItem := g.pool.GetConnBatch()
	if batchItem.Item == nil {
		return nil, fmt.Errorf("No Grpc connection instance found")
	}

	conn := batchItem.Item.(*grpc.ClientConn)

	g.log.Infoln("GRPC Conn State= ", conn.GetState().String())

	return conn, nil
}

func (g *GRPC) TestGetConn() (*grpc.ClientConn, error) {

	// Testing conn
	connTry := g.pool.MaxPoolSize
	for connTry > 0 {
		batchItem := g.pool.GetConnBatch()
		if batchItem.Item == nil {
			return nil, fmt.Errorf("No Grpc connection instance found")
		}
		g.log.Infoln("ConnTry=", connTry, " : GRPC Conn -- ", batchItem.Item.(*grpc.ClientConn).GetState().String())
		connTry--
		time.Sleep(1 * time.Second)
	}
	return nil, nil
}
