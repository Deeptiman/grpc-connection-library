package pool

import (
	"container/list"
	"fmt"
	batch "github.com/Deeptiman/go-batch"
	log "github.com/sirupsen/logrus"
	"sync/atomic"
)

type Queue struct {
	items   *list.List
	size    uint64
	watcher chan chan interface{}
	enqueCh []chan batch.BatchItems
	sem     *Semaphore
	log     *log.Logger
	en      int32
	qu      int32
}

func NewQueue(size uint64) *Queue {
	q := &Queue{
		items:   list.New(),
		size:    size,
		watcher: make(chan chan interface{}, 100),
		enqueCh: make([]chan batch.BatchItems, 0, size),
		sem:     NewSemaphore(size),
		log:     log.New(),
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

	q.items.PushBack(item)

	enCh := make(chan batch.BatchItems)
	q.enqueCh = append(q.enqueCh, enCh)
	go q.enqueItemCh(enCh, item)

}

func (q *Queue) Dequeue() interface{} {

	q.sem.RLock()
	defer q.sem.RUnlock()

	dequeueItem := q.items.Front().Value

	log.Infoln("Dequeue", "Pop Item ---- ", dequeueItem.(batch.BatchItems).BatchNo)

	return dequeueItem
}

func (q *Queue) GetItems() chan interface{} {

	go q.dispatcher(q.watcher)

	requestCh := make(chan interface{})

	q.watcher <- requestCh

	return <-q.watcher
}

func (q *Queue) GetEnqueCh(index int) chan batch.BatchItems {
	return q.enqueCh[index]
}

func (q *Queue) GetEnqueIndex() int {
	return len(q.enqueCh) - 1
}

func (q *Queue) dispatcher(requestCh <-chan chan interface{}) {

	for {
		select {
		case requestChan := <-requestCh:
			q.log.Infoln("Dispatch Request from Watcher")
			requestChan <- q.Dequeue()
		}
	}
}

func (q *Queue) getTotalEnqueue() int32 {
	return atomic.LoadInt32(&q.en)
}

func (q *Queue) getTotalDeQueue() int32 {
	return atomic.LoadInt32(&q.qu)
}

func (q *Queue) Front() interface{} {

	q.sem.RLock()
	defer q.sem.RUnlock()

	return q.items.Front().Value
}

func (q *Queue) IsEmpty() bool {
	return q.items.Len() == 0
}

func (q *Queue) GetCapacity() int {

	q.sem.RLock()
	defer q.sem.RUnlock()

	return q.items.Len()
}

func (q *Queue) GetLen() int {

	q.sem.RLock()
	defer q.sem.RUnlock()

	return q.items.Len()
}

func (q *Queue) RemoveElement(index int) error {

	q.sem.Lock()
	defer q.sem.Unlock()

	if q.items.Len() <= index {
		return fmt.Errorf("index out of range")
	}

	q.items.Remove(q.items.Front())

	return nil
}
