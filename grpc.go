package main

import (
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/grpclog"
	"google.golang.org/grpc/status"
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
			scheme:         DefaultScheme,
		},
		log: grpclog.NewLoggerV2(os.Stdout, ioutil.Discard, ioutil.Discard),
	}

	grpclog.SetLoggerV2(grpcConn.log)

	for _, opt := range opts {
		opt(grpcConn)
	}

	switch grpcConn.options.connectionType {
	case Server:
		return grpcConn, grpcConn.ListenAndServe()
	case Client:
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

	serverOptions := g.options.serverOptions
	grpcServer := grpc.NewServer(serverOptions...)
	if err = grpcServer.Serve(listener); err != nil {
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

	if g.options.interceptor == UnaryClient {
		opts = append(opts, grpc.WithUnaryInterceptor(UnaryClientInterceptor(g.options.retryOption)))
	}

	conn, err := grpc.Dial(g.options.address, opts...)
	if err != nil {
		g.log.Fatal(err)
		return err
	}
	g.client = conn
	g.log.Infoln("GRPC Client connected at - address : ", g.options.address)
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
