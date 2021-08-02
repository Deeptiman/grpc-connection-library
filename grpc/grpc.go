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

// ConnectionType gRPC[client:server]
type ConnectionType int

// Connectivity State of gRPC connection [Idle,Connecting,Ready,TransientFailure,Shutdown]
type ConnState connectivity.State

var (
	DefaultConnectionType        = Client // DefaultConnectionType [Client], library initiate the gRPC client connection
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

// GRPC struct defines the Connection objects for both client and server.
//
//  connectionType: The gRPC connection library can be used as client and server connection. The connectiontype
//  can provide provide conn options to the user.
//
//  server:  The GRPC server connection instance gets stored in this field.
//
//  serverAddress: The gRPC server address that listens to certain ports.
//
//  serverOptions: There are several gRPC serverOptions can be set as a server configuration.
//
//  pool: The connection pool will create multiple gRPC client connections by replicating the base connection
//  instance into a reusable connection pool.
//
//  log: The gRPC log will show the internal connection lifecycle that will be useful to debug the connection flow.
type GRPC struct {
	connectionType ConnectionType
	server         *grpc.Server
	serverAddress  string
	serverPort     string
	serverOptions  []grpc.ServerOption
	pool           *pool.ConnPool
	log            grpclog.LoggerV2
}

// NewGRPCConnection will instantiate the GRPC object and will configure the field objects with the several Options
// parameters. The connection pool gets instantiated with the serverAddress set by the Options.WithAddress.
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

// ListenAndServe will start listening to the specific serverPort with the "tcp" network type. After the server listener socket
// is opened, the initial Ping request gets registered that can be used by any gRPC clients to sent Ping-Pong request to check the
// client-server gRPC connection health status.
func (g *GRPC) ListenAndServe() error {

	port := g.serverPort
	listener, err := net.Listen("tcp", ":"+port)
	if err != nil {
		g.log.Fatalf("GRPC Server listen failed - %v", err)
		return err
	}
	g.log.Infoln("Server Port - ", port)
	serverOptions := g.serverOptions

	grpcServer := grpc.NewServer(serverOptions...)
	// Register the Ping service request
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

// GetConn will provide the gRPC client connection instance by querying to the connection pool that retrieves the
// gRPC client connections via performing the batch processing.
func (g *GRPC) GetConn() (*grpc.ClientConn, error) {

	batchItem := g.pool.GetConnBatch()
	if batchItem.Item == nil {
		return nil, fmt.Errorf("No Grpc connection instance found")
	}

	conn := batchItem.Item.(*grpc.ClientConn)

	g.log.Infoln("GRPC Conn State= ", conn.GetState().String())

	return conn, nil
}

// TestGetConn to test the recurrent connection pool iteration to get the gRPC connection instances.
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
