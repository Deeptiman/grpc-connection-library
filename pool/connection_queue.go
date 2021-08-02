package pool

import (
	batch "github.com/Deeptiman/go-batch"
	log "grpc-connection-library/logger"
	"reflect"
)

type Queue struct {
	size       uint64
	itemSelect []reflect.SelectCase
	enqueCh    []chan batch.BatchItems
	sem        *Semaphore
	log        *log.Logger
}

func NewQueue(size uint64) *Queue {
	q := &Queue{
		size:       size,
		itemSelect: make([]reflect.SelectCase, size),
		enqueCh:    make([]chan batch.BatchItems, 0, size),
		sem:        NewSemaphore(size),
		log:        log.NewLogger(),
	}
	return q
}

func (q *Queue) enqueItemCh(enCh chan<- batch.BatchItems, item batch.BatchItems) {
	enCh <- item
	close(enCh)
}

func (q *Queue) Enqueue(item batch.BatchItems) {

	q.sem.Lock()
	defer q.sem.Unlock()

	enCh := make(chan batch.BatchItems)
	q.enqueCh = append(q.enqueCh, enCh)
	go q.enqueItemCh(enCh, item)

	i := q.GetEnqueIndex()
	q.itemSelect[i] = reflect.SelectCase{Dir: reflect.SelectRecv, Chan: reflect.ValueOf(q.GetEnqueCh(i))}

}

func (q *Queue) Dequeue() batch.BatchItems {

	defer q.recoverInvalidSelect()

	for {
		chosen, rcv, ok := reflect.Select(q.itemSelect)
		if !ok {
			q.log.Infoln("Conn Batch Instance Not Chosen = ", chosen)
			continue
		}
		q.log.Infoln("SelectCase", "Batch Conn : chosen = ", chosen)
		q.itemSelect = append(q.itemSelect[:chosen], q.itemSelect[chosen+1:]...)
		return rcv.Interface().(batch.BatchItems)
	}
}

func (q *Queue) GetEnqueCh(index int) chan batch.BatchItems {
	return q.enqueCh[index]
}

func (q *Queue) GetEnqueIndex() int {
	return len(q.enqueCh) - 1
}

func (q *Queue) recoverInvalidSelect() {
	if r := recover(); r != nil {
		q.log.Infoln("Recovered", r)
	}
}
