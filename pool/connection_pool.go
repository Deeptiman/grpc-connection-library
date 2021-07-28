package pool

import (
	"container/list"
	batch "github.com/Deeptiman/go-batch"
	"google.golang.org/grpc"
	"google.golang.org/grpc/grpclog"
	"io/ioutil"
	"os"
	"reflect"
	"sync"
	"sync/atomic"
)

type ConnPool struct {
	Conn                 *grpc.ClientConn
	MaxPoolSize          uint64
	ConnInstanceReplicas *list.List
	ConnInstanceBatch    *batch.Batch
	ConnBatchQueue       *Queue
	ConnSelect           []reflect.SelectCase
	PipelineDoneChan     chan interface{}
	Lock                 sync.Mutex
	Log                  grpclog.LoggerV2
}

var (
	DefaultConnBatch  uint64 = 20
	ConnIndex         uint64 = 0
	ConnPoolPipeline  uint64 = 0
	ConnRecreateCount uint64 = 0
	IsConnRecreate    bool   = false
)

type BatchPipelineFn func(batchItems interface{}) batch.BatchItems

func NewConnPool(maxPoolSize uint64) *ConnPool {

	//queueSize := maxPoolSize / DefaultConnBatch
	return &ConnPool{
		MaxPoolSize:          maxPoolSize,
		ConnInstanceReplicas: list.New(),
		ConnInstanceBatch:    batch.NewBatch(batch.WithMaxItems(DefaultConnBatch)),
		ConnBatchQueue:       NewQueue(maxPoolSize),
		ConnSelect:           make([]reflect.SelectCase, maxPoolSize),
		PipelineDoneChan:     make(chan interface{}),
		Lock:                 sync.Mutex{},
		Log:                  grpclog.NewLoggerV2(os.Stdout, ioutil.Discard, ioutil.Discard),
	}
}

func (c *ConnPool) ConnectionPoolPipeline(conn *grpc.ClientConn, pipelineDoneChan chan interface{}) {

	// 1
	connReplicas := func(conn *grpc.ClientConn) <-chan *grpc.ClientConn {
		connInstanceCh := make(chan *grpc.ClientConn)
		go func() {
			c.Log.Infoln("1#connReplicas ...")
			defer close(connInstanceCh)
			for i := 0; uint64(i) < c.MaxPoolSize; i++ {
				select {
				case connInstanceCh <- conn:
				}
			}
		}()
		return connInstanceCh
	}

	// 2
	connBatch := func(connInstanceCh <-chan *grpc.ClientConn) chan []batch.BatchItems {

		go func() {
			c.Log.Infoln("2#connBatch ...")
			c.ConnInstanceBatch.StartBatchProcessing()
			for conn := range connInstanceCh {
				select {
				case c.ConnInstanceBatch.Item <- conn:
				}
			}
		}()
		return c.ConnInstanceBatch.Consumer.Supply.ClientSupplyCh
	}

	// 3
	connEnqueue := func(connSupplyCh <-chan []batch.BatchItems) <-chan batch.BatchItems {
		receiveBatchCh := make(chan batch.BatchItems)
		go func() {
			c.Log.Infoln("3#connEnqueue ...")
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
			if (c.MaxPoolSize - poolSize) == 2 {
				atomic.AddUint64(&ConnRecreateCount, 1)
				IsConnRecreate = true
			}
			return
		}
	}

	// Recreate connection pool
	if IsConnRecreate {
		c.Log.Infoln("Connection Recreate !!!")
		c.ConnSelect = make([]reflect.SelectCase, c.MaxPoolSize)
		c.ConnBatchQueue.enqueCh = make([]chan batch.BatchItems, 0, c.MaxPoolSize)
		c.PipelineDoneChan = make(chan interface{})
		c.ConnInstanceBatch.Unlock()
		IsConnRecreate = false
	}

	// Pipeline
	for s := range connEnqueue(connBatch(connReplicas(conn))) {
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
}

func (c *ConnPool) GetConnIndex() uint64 {
	return atomic.LoadUint64(&ConnIndex)
}

func (c *ConnPool) IncreaseConnIndex() {
	c.Log.Infoln("EnqueItem -- IncreaseConnIndex : ", c.GetConnIndex())
	atomic.AddUint64(&ConnIndex, 1)
}

func (c *ConnPool) EnqueConnBatch(connItems batch.BatchItems) {
	c.ConnBatchQueue.Enqueue(connItems)

	i := c.ConnBatchQueue.GetEnqueIndex()
	c.ConnSelect[i] = reflect.SelectCase{Dir: reflect.SelectRecv, Chan: reflect.ValueOf(c.ConnBatchQueue.GetEnqueCh(i))}

	c.Log.Infoln("Select Created Index --- ", i)
}

func (c *ConnPool) GetConnBatch() batch.BatchItems {

	batchItemCh := make(chan batch.BatchItems)
	defer close(batchItemCh)
	go c.ConnectionPoolPipeline(c.Conn, c.PipelineDoneChan)

	for {
		select {
		case <-c.PipelineDoneChan:
			c.Log.Infoln("Pipeline Done Channel !")
			defer c.recoverInvalidSelect()

			for {
				chosen, rcv, ok := reflect.Select(c.ConnSelect)
				if !ok {
					c.Log.Infoln("Conn Batch Instance Not Chosen = ", chosen)
					continue
				}
				c.Log.Infoln("SelectCase", "Batch Conn : chosen = ", chosen, " : channel = ", c.ConnSelect[chosen], " : received = ", rcv)
				poolSize := c.GetConnPoolSize() - 1
				atomic.StoreUint64(&ConnPoolPipeline, poolSize)

				return rcv.Interface().(batch.BatchItems)
			}
		}
	}
}

func (c *ConnPool) GetConnPoolSize() uint64 {
	return atomic.LoadUint64(&ConnPoolPipeline)
}

func (c *ConnPool) recoverInvalidSelect() {
	if r := recover(); r != nil {
		c.Log.Infoln("Recovered", r)
	}
}
