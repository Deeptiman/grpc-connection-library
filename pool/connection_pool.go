package pool

import (
	"container/list"
	"context"
	"fmt"
	batch "github.com/Deeptiman/go-batch"
	"golang.org/x/sync/errgroup"
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

	errs, _ := errgroup.WithContext(context.Background())

	replicaCh := make(chan bool)
	batchCh := make(chan bool)
	enqueCh := make(chan bool)

	errs.Go(func() error {

		c.createConnectionReplicas(conn)

		replicaCh <- true

		return nil
	})

	errs.Go(func() error {

		select {
		case <-replicaCh:
			c.createConnectionBatch()
			batchCh <- true
		}
		return nil
	})

	errs.Go(func() error {

		select {
		case <-batchCh:

			c.Log.Infoln("Batch Size ------ ", len(c.ConnInstanceBatch.Consumer.Supply.ClientSupplyCh))
			for bt := range c.ConnInstanceBatch.Consumer.Supply.ClientSupplyCh {
				c.Log.Infoln("Reading Conn Batches : Total Size = ", len(bt))
				c.EnqueConnBatch(bt)
			}
			c.Log.Infoln("Conn Batch Reading Complete!")
			enqueCh <- true			
		}

		return nil
	})

	errs.Go(func() error {
		select {
		case <-enqueCh:

			c.Log.Infoln(" ConnBatchQueue.GetLen() --- ", c.ConnBatchQueue.GetLen())
			for c.ConnBatchQueue.GetLen() > 0 {
				c.GetConnBatch()
			}
		}
		return nil
	})

	if err := errs.Wait(); err != nil {
		c.Log.Infoln("Error Conn Pool - ", err.Error())
		return
	}

}

func (c *ConnPool) createConnectionReplicas(conn *grpc.ClientConn) {
	c.Log.Infoln("createConnectionReplicas ...")
	for i := 0; uint64(i) < c.MaxPoolSize; i++ {
		c.ConnInstanceReplicas.PushFront(conn)
	}
}

func (c *ConnPool) createConnectionBatch() {

	c.Log.Infoln("createConnectionBatch ...")

	c.ConnInstanceBatch.StartBatchProcessing()

	for c.ConnInstanceReplicas.Len() > 0 {

		if conn, _ := c.getConnReplicas(); conn != nil {
			c.ConnInstanceBatch.Item <- conn
		}
	}
	c.ConnInstanceBatch.Close()
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

func (c *ConnPool) getConnReplicas() (*grpc.ClientConn, error) {

	c.Log.Infoln("getConnReplicas ...")

	if c.ConnInstanceReplicas.Len() > 0 {
		if conn, ok := c.ConnInstanceReplicas.Front().Value.(*grpc.ClientConn); ok {
			c.ConnInstanceReplicas.Remove(c.ConnInstanceReplicas.Front())
			return conn, nil
		}
		return nil, fmt.Errorf("Failed to retrieve conn instance from stack!")
	}

	return nil, fmt.Errorf("Conn Replica Stack is empty!")
}
