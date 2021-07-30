package retry

import (
	"os"
	"io/ioutil"
	"errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/grpclog"
)

type RetryOption struct {
	Retry   int
	Address string
	Backoff Backoff
	Codes   []codes.Code
}

type ConnState connectivity.State

const (
	Idle ConnState = iota
	Connecting
	Ready
	TransientFailure
	ShutDown
)

var (
	ErrRetryMaxLimit = errors.New("Retry Limit Exceed!")
	DefaultRetryCount int = 5
	log = grpclog.NewLoggerV2(os.Stdout, ioutil.Discard, ioutil.Discard)
)

type ClientConnFactory func(address string) (*grpc.ClientConn, error)

type ServerConnFactory func(address string) (*grpc.Server, error)

func RetryClientConnection(factory ClientConnFactory, retryOption *RetryOption) (*grpc.ClientConn, error) {

	grpclog.SetLoggerV2(log)

	var conn *grpc.ClientConn
	var err error
	for i := 1; i <= retryOption.Retry; i++ {

		log.Infoln("grpc:connect:conn - retry=", i)

		conn, err = factory(retryOption.Address)
		if err != nil {

			if !isRetriable(err, retryOption.Codes) {
				return nil, err
			}

			if err := retryOption.retryBackoff(i); err != nil {
				return nil, err
			}	

			continue
		}

		if conn != nil && failedConnState(conn.GetState()){
			if err := retryOption.retryBackoff(i); err != nil {
				return nil, err
			}
			continue
		}

		if conn != nil && conn.GetState() == connectivity.Ready {
			break
		}
	}
	return conn, err
}

func isRetriable(err error, codes []codes.Code) bool {
	errCode := status.Code(err)
	for _, code := range codes {
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

func failedConnState(state connectivity.State) bool {

	return state == connectivity.TransientFailure || 
	state == connectivity.Shutdown
}

func(r *RetryOption) retryBackoff(attempt int) error {
	
	log.Infoln("grpc:connect:error - retry=", attempt)

	r.Backoff.ApplyBackoffDuration(attempt)
	if attempt == r.Retry {
		return ErrRetryMaxLimit
	}
	return nil
}