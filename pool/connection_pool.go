package pool

import (
	"container/list"
	batch "github.com/Deeptiman/go-batch"
	"google.golang.org/grpc"
	"google.golang.org/grpc/grpclog"
	"io/ioutil"
	"os"
	"sync"
)

type ConnPool struct {
	MaxPoolSize          uint64
	ConnInstanceReplicas *list.List
	ConnInstanceBatch    *batch.Batch
	ConnBatchQueue       *Queue
	Lock                 sync.Mutex
	Log                  grpclog.LoggerV2
}

var (
	DefaultConnBatch uint64 = 20
)

func NewConnPool(maxPoolSize uint64) *ConnPool {

	queueSize := maxPoolSize / DefaultConnBatch
	return &ConnPool{
		MaxPoolSize:          maxPoolSize,
		ConnInstanceReplicas: list.New(),
		ConnInstanceBatch:    batch.NewBatch(batch.WithMaxItems(DefaultConnBatch)),
		ConnBatchQueue:       NewQueue(queueSize),
		Lock:                 sync.Mutex{},
		Log:                  grpclog.NewLoggerV2(os.Stdout, ioutil.Discard, ioutil.Discard),
	}
}

func (c *ConnPool) ConnectionPoolPipeline(conn *grpc.ClientConn) {

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
		go func(){
			c.Log.Infoln("3#connEnqueue ...")
			defer close(receiveBatchCh)
			for supply := range connSupplyCh {
	
				c.EnqueConnBatch(supply)
				for _, s := range supply {					
					select {
					case receiveBatchCh <- s:
					}
				}
				c.Log.Infoln(" ------------------------------------------------- ")
			}
		}()
		return receiveBatchCh
	}
	
	// Pipeline	
	for s := range connEnqueue(connBatch(connReplicas(conn))) {
		c.Log.Infoln("Batch Receive : Conn", " Id - ", s.Id, " -- Batch.No : ", s.BatchNo, " -- Queue Size : ", c.ConnBatchQueue.GetLen(), " --  MaxQueue Size : ", c.ConnBatchQueue.size," - ConnState : ", s.Item.(*grpc.ClientConn).GetState().String())
	}
}

func (c *ConnPool) EnqueConnBatch(connItems []batch.BatchItems) {
	c.Log.Infoln("EnqueConnBatch...")
	c.ConnBatchQueue.Enqueue(connItems)
}

func (c *ConnPool) GetConnBatch() []batch.BatchItems {
	c.Log.Infoln("GetConnBatch...", " -- Queue Size : ", c.ConnBatchQueue.GetLen())
	queueItems := c.ConnBatchQueue.Dequeue()
	return queueItems.([]batch.BatchItems)
}

func (c *ConnPool) IterateConnBatch() {
	c.Log.Infoln(" IterateConnBatch ... ")
	for c.ConnBatchQueue.GetLen() > 0 {
		items := c.GetConnBatch()
		for i, item := range items {
			c.Log.Infoln(" ConnBatchQueue", "Iterate--", i, " === BatchNo : ", item.BatchNo, " === BatchID : ", item.Id, " -- ConnState : ", item.Item.(*grpc.ClientConn).GetState().String())
		}
	}
}
