package pool

import (
	"github.com/Deeptiman/go-batch"
	log "github.com/Deeptiman/grpc-connection-library/logger"
	"reflect"
)

// Queue struct defines the constructors to enqueue/dequeue of batch processing items.
//
//  size: The total size of the connection queue is similar to the value of MaxPoolSize.
//
//  itemSelect: The list of enqueued cases that gets stored as channels and dequeued using pseudo-random selection.
//
//  enqueCh: The enqueCh will store the array of channel batchItems and get added to the reflect.SelectCase.
//
//  sem: The semaphore will establish the synchronization between the enqueue/dequeue process by creating Acquire/Release blocking channels.
//
//  log: The log uses the loggrus library to provide logging into the library.
type Queue struct {
	size       uint64
	itemSelect []reflect.SelectCase
	enqueCh    []chan batch.BatchItems
	sem        *Semaphore
	log        *log.Logger
}

// NewQueue will instantiate a new Queue with the given size and initialize the connection array of different
// configurations like []channel and reflect.SelectCase.
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

// enqueItemCh will send the batch items to the enCh channel and that's get picked using the reflect.SelectCase during the
// dequeuing process.
func (q *Queue) enqueItemCh(enCh chan<- batch.BatchItems, item batch.BatchItems) {
	enCh <- item
	close(enCh)
}

// Enqueue function will append the received batch item to the array of channel queue and also gets added to the
// []reflect.SelectCase as a channel case.
func (q *Queue) Enqueue(item batch.BatchItems) {

	q.sem.Lock()
	defer q.sem.Unlock()

	enCh := make(chan batch.BatchItems)
	q.enqueCh = append(q.enqueCh, enCh)
	go q.enqueItemCh(enCh, item)

	i := len(q.enqueCh) - 1
	q.itemSelect[i] = reflect.SelectCase{Dir: reflect.SelectRecv, Chan: reflect.ValueOf(q.GetEnqueCh(i))}

}

// Dequeue function will pick a batch item from the channel cases using the pseudo-random technique.
func (q *Queue) Dequeue() batch.BatchItems {

	defer q.recoverInvalidSelect()
	for {
		chosen, rcv, ok := reflect.Select(q.itemSelect)
		if !ok {
			q.log.Infoln("Conn Batch Instance Not Chosen = ", chosen)
			continue
		}
		q.log.Infoln("SelectCase", "Batch Conn : chosen = ", chosen)

		// Remove the selected case from the array to avoid the duplicate choosing of the cases.
		q.itemSelect = append(q.itemSelect[:chosen], q.itemSelect[chosen+1:]...)

		return rcv.Interface().(batch.BatchItems)
	}
}

// GetEnqueCh will get the enqueued batch item for an index.
func (q *Queue) GetEnqueCh(index int) chan batch.BatchItems {
	return q.enqueCh[index]
}

// recoverInvalidSelect will recover from an invalid select case that got panic during the selection process.
func (q *Queue) recoverInvalidSelect() {
	if r := recover(); r != nil {
		q.log.Infoln("Recovered", r)
	}
}
