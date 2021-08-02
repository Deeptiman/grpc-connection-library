package pool

import (
	interceptor "grpc-connection-library/interceptor"
	pb "grpc-connection-library/ping"
	retry "grpc-connection-library/retry"
	"io/ioutil"
	"os"
	"reflect"
	"sync/atomic"

	batch "github.com/Deeptiman/go-batch"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/grpclog"
)

// ConnPool struct defines several fields to construct a connection pool. The gRPC connection instance replicated
// using batch processing and queried using reflect.SelectCase that gets the connection object
// from the list of cases as a pseudo-random choice.
//
//  Conn: The base gRPC connection that keeps the initial client connection instance.
//
//  MaxPoolSize: The maximum number of connections created concurrently in the connection pool.
//
//  ConnInstanceBatch: The batch processing will help creates multiple replicas of base gRPC connection instances concurrently.
//  https://github.com/Deeptiman/go-batch library used to perform the batch processing.
//
//
//  Options: The options will keep the connection pool configurations to create gRPC dialoptions for base connection instances.
//
//  PipelineDoneChan: The connection pool runs a concurrency pipeline, so the "PipelineDoneChan" channel will be called after
//  all the stages of the pipeline finishes.
//
//  Log: The gRPC log will show the internal connection lifecycle that will be useful to debug the connection flow.
type ConnPool struct {
	Conn              *grpc.ClientConn
	MaxPoolSize       uint64
	ConnInstanceBatch *batch.Batch
	ConnBatchQueue    *Queue
	Options           *PoolConnOptions
	PipelineDoneChan  chan interface{}
	Log               grpclog.LoggerV2
}

// ConnectionInterceptor defines the interceptors[UnaryServer:UnaryClient] type for a gRPC connection.
type ConnectionInterceptor int

const (
	// ConnectionInterceptor constants
	UnaryServer ConnectionInterceptor = iota
	UnaryClient
)

var (
	// maximum number of connection instances stored for a single batch
	DefaultConnBatch uint64 = 20

	// The total size of the connection pool to store the gRPC connection instances
	DefaultMaxPoolSize uint64 = 60

	// gRPC connection scheme to override the default scheme "passthrough" to "dns"
	DefaultScheme string = "dns"

	// the authentication [enable/disable] bool flag
	DefaultGrpcInsecure bool = true

	// This gRPC connection library currently only supports one type of interceptors to send msg to the server that doesn't expect a response
	DefaultInterceptor ConnectionInterceptor = UnaryClient

	// possible retriable gRPC connection failure codes
	DefaultRetriableCodes = []codes.Code{codes.Aborted, codes.Unknown, codes.ResourceExhausted, codes.Unavailable}

	ConnIndex         uint64 = 0
	ConnPoolPipeline  uint64 = 0
	ConnRecreateCount uint64 = 0
	IsConnRecreate    bool   = false
)

// NewConnPool will create the connection pool object that will instantiate the configurations for connection batch,
// retryOptions, interceptor.
func NewConnPool(opts ...PoolOptions) *ConnPool {

	connPool := &ConnPool{
		MaxPoolSize: DefaultMaxPoolSize,
		Options: &PoolConnOptions{
			insecure:    DefaultGrpcInsecure,
			scheme:      DefaultScheme,
			interceptor: DefaultInterceptor,
			connBatch:   DefaultConnBatch,
			retryOption: &retry.RetryOption{
				Retry: retry.DefaultRetryCount,
				Codes: DefaultRetriableCodes,
				Backoff: &retry.Backoff{
					Strategy: retry.Linear,
				},
			},
		},
		PipelineDoneChan: make(chan interface{}),
		Log:              grpclog.NewLoggerV2(os.Stdout, ioutil.Discard, ioutil.Discard),
	}

	for _, opt := range opts {
		opt(connPool)
	}
	connPool.Options.retryOption.Address = connPool.Options.address
	connPool.ConnInstanceBatch = batch.NewBatch(batch.WithMaxItems(connPool.Options.connBatch))
	connPool.ConnBatchQueue = NewQueue(connPool.MaxPoolSize)

	return connPool
}

// ClientConn will create the initial gRPC client connection instance. The connection factory works as a higher
// order function for gRPC retry policy in case of connection failure retries.
func (c *ConnPool) ClientConn() (*grpc.ClientConn, error) {

	connectionFactory := func(address string) (*grpc.ClientConn, error) {

		var opts []grpc.DialOption
		if c.Options.authority != "" {
			opts = append(opts, grpc.WithAuthority(c.Options.authority))
		}

		if c.Options.insecure {
			opts = append(opts, grpc.WithInsecure())
		} else {
			opts = append(opts, grpc.WithTransportCredentials(c.Options.credentials))
		}

		if c.Options.interceptor == UnaryClient {
			// WithUnaryInterceptor DialOption parameter will set the UnaryClient option type for the interceptor RPC calls
			opts = append(opts, grpc.WithUnaryInterceptor(interceptor.UnaryClientInterceptor(c.Options.retryOption)))
		}

		address = c.Options.scheme + ":///" + address

		c.Log.Infoln("Dial GRPC Server ....", address)

		conn, err := grpc.Dial(address, opts...)
		if err != nil {
			c.Log.Fatal(err)
			return nil, err
		}
		c.Conn = conn

		// gRPC connection instance creates Ping service to test connection health.
		client := pb.NewPingServiceClient(c.Conn)

		c.Log.Infoln("GRPC Client connected at - address : ", address, " : ConnState = ", c.Conn.GetState())

		// The SendPingMsg sends a test msg to the target server address to get the Pong response msg back to verify the connection flow.
		respMsg, err := pb.SendPingMsg(client)
		if err != nil {
			return nil, err
		}
		c.Log.Infoln("GRPC Pong msg - ", respMsg)

		return c.Conn, nil
	}

	return retry.RetryClientConnection(connectionFactory, c.Options.retryOption)
}

// ConnectionPoolPipeline follows the concurrency pipeline technique to create a connection pool in a higher
// concurrent scenarios. The pipeline has several stages that use the Fan-In, Fan-Out technique to process the
// data pipeline using channels.
//
// The entire process of creating the connection pool becomes a powerful function using the pipeline technique.
// There are four different stages in this pipeline that works as a generator pattern to create a connection pool.
//
//  1#connInstancefn: This stage will create the initial gRPC connection instance that gets passed to the next pipeline stage for replication.
//
//
//  2#connReplicasfn: The cloning process of the initial gRPC connection object will begin here. The connection instance gets passed to the next stage iteratively via channels.
//
//
//  3#connBatchfn: This stage will start the batch processing using the
//  github.com/Deeptiman/go-batch library. The MaxPoolSize is divided into multiple batches and released via a supply channel from go-batch library internal implementation.
//
//
//  4#connEnqueuefn:   The connection queue reads through the go-batch client supply channel and stores the connection instances as channel case in []reflect.SelectCase.
//  So, whenever the client requests a connection instance, reflect.SelectCase retrieves the conn instances from the case using the pseudo-random technique.
func (c *ConnPool) ConnectionPoolPipeline(conn *grpc.ClientConn, pipelineDoneChan chan interface{}) {

	// 1
	connInstancefn := func(done chan interface{}) <-chan *grpc.ClientConn {

		connCh := make(chan *grpc.ClientConn)

		conn, err := c.ClientConn()
		if err != nil {
			done <- err
		}

		go func() {
			c.Log.Infoln("1#connInstance ...")
			defer close(connCh)
			select {
			case connCh <- conn:
				c.Log.Infoln("GRPC Connection Status - ", conn.GetState().String())
			}
		}()
		return connCh
	}

	// 2
	connReplicasfn := func(connInstanceCh <-chan *grpc.ClientConn) <-chan *grpc.ClientConn {
		connInstanceReplicaCh := make(chan *grpc.ClientConn)
		go func() {
			c.Log.Infoln("2#connReplicas ...")
			defer close(connInstanceReplicaCh)
			for conn := range connInstanceCh {
				for i := 0; uint64(i) < c.MaxPoolSize; i++ {
					select {
					case connInstanceReplicaCh <- conn:
					}
				}
			}
		}()
		return connInstanceReplicaCh
	}

	// 3
	connBatchfn := func(connInstanceCh <-chan *grpc.ClientConn) chan []batch.BatchItems {

		go func() {
			c.Log.Infoln("3#connBatch ...")
			c.ConnInstanceBatch.StartBatchProcessing()
			for conn := range connInstanceCh {
				select {
				case c.ConnInstanceBatch.Item <- conn:
				}
			}
		}()
		return c.ConnInstanceBatch.Consumer.Supply.ClientSupplyCh
	}

	// 4
	connEnqueuefn := func(connSupplyCh <-chan []batch.BatchItems) <-chan batch.BatchItems {
		receiveBatchCh := make(chan batch.BatchItems)
		go func() {
			c.Log.Infoln("4#connEnqueue ...")
			defer close(receiveBatchCh)
			for supply := range connSupplyCh {

				for _, s := range supply {
					c.EnqueConnBatch(s)
					select {
					case receiveBatchCh <- s:
					}
				}
			}
		}()
		return receiveBatchCh
	}

	poolSize := c.GetConnPoolSize()
	if poolSize > 0 {
		select {
		case pipelineDoneChan <- "Done":
			c.Log.Infoln("Pool Exists - Size : ", poolSize)
			if (c.MaxPoolSize - poolSize) == 1 {
				atomic.AddUint64(&ConnRecreateCount, 1)
				IsConnRecreate = true
			}
			return
		}
	}

	// Recreate connection pool
	if IsConnRecreate {
		c.Log.Infoln("Connection Recreate !!!")
		c.ConnBatchQueue.itemSelect = make([]reflect.SelectCase, c.MaxPoolSize)
		c.ConnBatchQueue.enqueCh = make([]chan batch.BatchItems, 0, c.MaxPoolSize)
		c.PipelineDoneChan = make(chan interface{})
		c.ConnInstanceBatch.Unlock()
		IsConnRecreate = false
	}
	done := make(chan interface{})

	// Concurrency Pipeline
	for s := range connEnqueuefn(connBatchfn(connReplicasfn(connInstancefn(done)))) {
		go func(s batch.BatchItems) {

			atomic.AddUint64(&ConnPoolPipeline, 1)
			if c.GetConnPoolSize() == c.MaxPoolSize {
				select {
				case pipelineDoneChan <- "Done":
					return
				}
			}
		}(s)
	}

	select {
	case <-done:
		return
	}
}

// EnqueConnBatch will enqueue the batchItems received from the go-batch supply channel.
func (c *ConnPool) EnqueConnBatch(connItems batch.BatchItems) {
	c.ConnBatchQueue.Enqueue(connItems)
}

// GetConnBatch will retrieve the batch item from the connection queue that dequeues the items using the pseudo-random technique.
func (c *ConnPool) GetConnBatch() batch.BatchItems {

	batchItemCh := make(chan batch.BatchItems)
	defer close(batchItemCh)
	go c.ConnectionPoolPipeline(c.Conn, c.PipelineDoneChan)

	select {
	case <-c.PipelineDoneChan:
		c.Log.Infoln("Pipeline Done Channel !")
		poolSize := c.GetConnPoolSize() - 1
		atomic.StoreUint64(&ConnPoolPipeline, poolSize)
		return c.ConnBatchQueue.Dequeue()
	}
}

// The number of connections created by the connection pool
func (c *ConnPool) GetConnPoolSize() uint64 {
	return atomic.LoadUint64(&ConnPoolPipeline)
}
