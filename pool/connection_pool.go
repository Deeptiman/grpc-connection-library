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

func (c *ConnPool) CreateConnectionPool(conn *grpc.ClientConn) {

	connReplicas := func(conn *grpc.ClientConn) <-chan *grpc.ClientConn {
		c.Log.Infoln("createConnectionReplicas ...")

		connInstanceCh := make(chan *grpc.ClientConn)

		go func() {
			defer close(connInstanceCh)
			for i := 0; uint64(i) < c.MaxPoolSize; i++ {

				select {
				case connInstanceCh <- conn:
				}
			}
		}()
		return connInstanceCh
	}

	connBatch := func(connInstanceCh <-chan *grpc.ClientConn) chan []batch.BatchItems {
		c.Log.Infoln("createConnectionBatch ...")

		go func() {

			c.ConnInstanceBatch.StartBatchProcessing()

			for conn := range connInstanceCh {
				select {
				case c.ConnInstanceBatch.Item <- conn:
				}
			}
		}()

		return c.ConnInstanceBatch.Consumer.Supply.ClientSupplyCh
	}

	connSupplyCh := connBatch(connReplicas(conn))

	for supply := range connSupplyCh {

		c.EnqueConnBatch(supply)
		for _, s := range supply {
			c.Log.Infoln("Conn", " Id - ", s.Id, " -- Batch.No : ", s.BatchNo, " - ConnState : ", s.Item.(*grpc.ClientConn).GetState().String())
		}
		c.Log.Infoln(" ------------------------------------------------- ")
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
