package pool

import (
	"container/list"
	interceptor "grpc-connection-library/interceptor"
	pb "grpc-connection-library/ping"
	retry "grpc-connection-library/retry"
	"io/ioutil"
	"os"
	"reflect"
	"sync"
	"sync/atomic"

	batch "github.com/Deeptiman/go-batch"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/grpclog"
)

type ConnPool struct {
	Conn                 *grpc.ClientConn
	MaxPoolSize          uint64
	ConnInstanceReplicas *list.List
	ConnInstanceBatch    *batch.Batch
	ConnBatchQueue       *Queue
	Options              *PoolConnOptions
	PipelineDoneChan     chan interface{}
	Lock                 sync.Mutex
	Log                  grpclog.LoggerV2
}

type ConnectionInterceptor int

const (
	UnaryServer ConnectionInterceptor = iota
	UnaryClient
)

var (
	DefaultConnBatch      uint64                = 20
	DefaultMaxPoolSize    uint64                = 60
	DefaultScheme         string                = "dns"
	DefaultGrpcInsecure   bool                  = true
	DefaultInterceptor    ConnectionInterceptor = UnaryClient
	DefaultRetriableCodes                       = []codes.Code{codes.Aborted, codes.Unknown, codes.ResourceExhausted, codes.Unavailable}

	ConnIndex         uint64 = 0
	ConnPoolPipeline  uint64 = 0
	ConnRecreateCount uint64 = 0
	IsConnRecreate    bool   = false
)

type BatchPipelineFn func(batchItems interface{}) batch.BatchItems

func NewConnPool(opts ...PoolOptions) *ConnPool {

	connPool := &ConnPool{
		MaxPoolSize:          DefaultMaxPoolSize,
		ConnInstanceReplicas: list.New(),
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
		Lock:             sync.Mutex{},
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
		client := pb.NewPingServiceClient(c.Conn)

		c.Log.Infoln("GRPC Client connected at - address : ", address, " : ConnState = ", c.Conn.GetState())

		respMsg, err := pb.SendPingMsg(client)
		if err != nil {
			return nil, err
		}
		c.Log.Infoln("GRPC Pong msg - ", respMsg)

		return c.Conn, nil
	}

	return retry.RetryClientConnection(connectionFactory, c.Options.retryOption)
}

func (c *ConnPool) ConnectionPoolPipeline(conn *grpc.ClientConn, pipelineDoneChan chan interface{}) {

	// 1
	connInstance := func(done chan interface{}) <-chan *grpc.ClientConn {

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
	connReplicas := func(connInstanceCh <-chan *grpc.ClientConn) <-chan *grpc.ClientConn {
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
	connBatch := func(connInstanceCh <-chan *grpc.ClientConn) chan []batch.BatchItems {

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
	connEnqueue := func(connSupplyCh <-chan []batch.BatchItems) <-chan batch.BatchItems {
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

	// Pipeline
	for s := range connEnqueue(connBatch(connReplicas(connInstance(done)))) {
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

func (c *ConnPool) GetConnIndex() uint64 {
	return atomic.LoadUint64(&ConnIndex)
}

func (c *ConnPool) IncreaseConnIndex() {
	atomic.AddUint64(&ConnIndex, 1)
}

func (c *ConnPool) EnqueConnBatch(connItems batch.BatchItems) {
	c.ConnBatchQueue.Enqueue(connItems)
}

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

func (c *ConnPool) GetConnPoolSize() uint64 {
	return atomic.LoadUint64(&ConnPoolPipeline)
}
