package main

import (
	"fmt"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/grpclog"
	"google.golang.org/grpc/status"
	pb "grpc-connection-library/ping"
	"io/ioutil"
	"net"
	"os"
)

type ConnectionInterceptor int

type ConnectionType int

var (
	DefaultConnectionType = Client
	DefaultInsecureState  = true
	DefaultScheme         = "dns"
	DefaultPort           = "9000"
	DefaultInterceptor    = UnaryClient

	RetriableCodes = []codes.Code{codes.ResourceExhausted, codes.Unavailable}
)

const (
	UnaryServer ConnectionInterceptor = iota
	UnaryClient
	StreamServer
	StreamClient

	Server ConnectionType = iota
	Client
)

type GRPC struct {
	options *GRPCOption
	server  *grpc.Server
	client  *grpc.ClientConn
	log     grpclog.LoggerV2
}

func NewGRPCConnection(opts ...Options) (*GRPC, error) {

	grpcConn := &GRPC{
		options: &GRPCOption{
			connectionType: DefaultConnectionType,
			insecure:       DefaultInsecureState,
			interceptor:    DefaultInterceptor,
			port:           DefaultPort,
			scheme:         DefaultScheme,
		},
		log: grpclog.NewLoggerV2(os.Stdout, ioutil.Discard, ioutil.Discard),
	}

	grpclog.SetLoggerV2(grpcConn.log)

	for _, opt := range opts {
		opt(grpcConn)
	}

	fmt.Println("NewGRPCConnection ! ConnectionType : ", grpcConn.options.connectionType)

	switch grpcConn.options.connectionType {
	case Server:
		fmt.Println("Connecting to GRPC Server....")
		return grpcConn, grpcConn.ListenAndServe()
	case Client:
		fmt.Println("Connecting to GRPC Client....")
		return grpcConn, grpcConn.ClientConn()
	}

	return grpcConn, nil
}

func (g *GRPC) ListenAndServe() error {

	port := g.options.port
	listener, err := net.Listen("tcp", ":"+port)
	if err != nil {
		g.log.Fatalf("GRPC Server listen failed - %v", err)
	}
	fmt.Println("Server Port - ", port)
	serverOptions := g.options.serverOptions

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

func (g *GRPC) ClientConn() error {

	var opts []grpc.DialOption

	if g.options.authority != "" {
		opts = append(opts, grpc.WithAuthority(g.options.authority))
	}

	if g.options.insecure {
		opts = append(opts, grpc.WithInsecure())
	} else {
		opts = append(opts, grpc.WithTransportCredentials(g.options.credentials))
	}

	// if g.options.interceptor == UnaryClient {
	// 	fmt.Println("UnaryClient ....")
	// 	opts = append(opts, grpc.WithUnaryInterceptor(UnaryClientInterceptor(g.options.retryOption)))
	// }

	g.options.address = fmt.Sprintf("localhost:%s", g.options.port)

	g.log.Infoln("Dial GRPC Server ....", g.options.address)

	conn, err := grpc.Dial(g.options.address, opts...)
	if err != nil {
		g.log.Fatal(err)
		return err
	}
	g.client = conn
	defer conn.Close()
	client := pb.NewPingServiceClient(g.client)

	g.log.Infoln("GRPC Client connected at - address : ", g.options.address, " : ConnState = ", g.client.GetState())

	respMsg, err := pb.SendPingMsg(client)
	if err != nil {
		return fmt.Errorf(" failed connect with address - %s err - %v", g.options.address, err)
	}
	g.log.Infoln("GRPC Pong msg - ", respMsg)

	return nil
}

func isRetriable(err error, callOpts RetryOption) bool {
	errCode := status.Code(err)
	for _, code := range callOpts.codes {
		if code == errCode {
			return true
		}
	}
	return false
}

func isContextError(err error) bool {
	code := status.Code(err)
	return code == codes.DeadlineExceeded || code == codes.Canceled
}
