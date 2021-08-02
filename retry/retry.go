package retry

import (
	"errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/grpclog"
	"google.golang.org/grpc/status"
	"io/ioutil"
	"os"
)

// RetryOption struct defines the configuration for retry behavior for the gRPC connection failure,
//
//  Retry: The number of retry attempts for gRPC connection retry.
//
//  Address: The target address of the server used for retry.
//
//  Backoff: The backoff strategy applies delays for each retry attempt.
//
//  Codes: A list of codes retriable codes that get matched with the error codes.
type RetryOption struct {
	Retry   int
	Address string
	Backoff *Backoff
	Codes   []codes.Code
}

// Connectivity State of gRPC connection [Idle,Connecting,Ready,TransientFailure,Shutdown]
type ConnState connectivity.State

const (
	Idle ConnState = iota // ConnState constants
	Connecting
	Ready
	TransientFailure
	ShutDown
)

var (
	ErrRetryMaxLimit      = errors.New("Retry Limit Exceed!") // ErrMsg after max retry failure
	DefaultRetryCount int = 5
	log                   = grpclog.NewLoggerV2(os.Stdout, ioutil.Discard, ioutil.Discard)
)

// ClientConnFactory is the abstract connection factory function to retry multiple times.
type ClientConnFactory func(address string) (*grpc.ClientConn, error)

// RetryClientConnection function will retry gRPC connection for max retry count using the ClientConnFactory.
func RetryClientConnection(factory ClientConnFactory, retryOption *RetryOption) (*grpc.ClientConn, error) {

	grpclog.SetLoggerV2(log)

	var conn *grpc.ClientConn
	var err error
	for i := 1; i <= retryOption.Retry; i++ {

		log.Infoln("grpc:connect:conn - retry=", i)

		conn, err = factory(retryOption.Address)
		if err != nil {

			log.Infoln("grpc:connect:err:", " - Msg = ", err.Error())

			if !IsRetriable(err, retryOption.Codes) {
				log.Infoln("grpc:connect:err:", " - Not Retriable = ", err.Error())
				return nil, err
			}

			if err := retryOption.RetryBackoff(i); err != nil {
				return nil, err
			}

			continue
		}

		if conn != nil && failedConnState(conn.GetState()) {
			if err := retryOption.RetryBackoff(i); err != nil {
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

// IsRetriable verifies the error codes with the defined retriable codes.
func IsRetriable(err error, codes []codes.Code) bool {
	errCode := status.Code(err)
	for _, code := range codes {
		if code == errCode {
			return true
		}
	}
	return false
}

// IsContextError verifies the error codes with context based error codes.
func IsContextError(err error) bool {
	code := status.Code(err)
	return code == codes.DeadlineExceeded || code == codes.Canceled
}

// failedConnState function matched the connectivity state with failure states.
func failedConnState(state connectivity.State) bool {
	return state == connectivity.Idle ||
		state == connectivity.TransientFailure ||
		state == connectivity.Shutdown
}

// RetryBackoff will apply the backoff strategy to provides a delay to each retry attempt.
func (r *RetryOption) RetryBackoff(attempt int) error {

	log.Infoln("grpc:connect:error - retry=", attempt)

	r.Backoff.ApplyBackoffDuration(attempt)
	if attempt == r.Retry {
		return ErrRetryMaxLimit
	}
	return nil
}
