package pool

import (
	"container/list"
	"fmt"
	log "github.com/sirupsen/logrus"
	"sync/atomic"
	"time"
)

type Queue struct {
	items   *list.List
	size    uint64
	watcher chan chan interface{}
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
		sem:     NewSemaphore(size),
		log:     log.New(),
	}
	return q
}

func (q *Queue) Enqueue(item interface{}) {

	q.sem.Lock()
	defer q.sem.Unlock()

	q.items.PushBack(item)

	log.WithFields(log.Fields{"AddItem": item}).Info("Enqueue")

	atomic.AddInt32(&q.en, 1)

	time.Sleep(100 * time.Millisecond)
}

func (q *Queue) Dequeue() interface{} {

	q.sem.RLock()
	defer q.sem.RUnlock()

	if q.items.Len() == 0 {
		log.WithFields(log.Fields{"Empty Item": ""}).Warning("Dequeue")
		return nil
	}

	dequeueItem := q.items.Front().Value

	log.WithFields(log.Fields{"Get Item": dequeueItem}).Debugln("Dequeue")

	return dequeueItem
}

func (q *Queue) GetItems() interface{} {

	responseCh := make(chan interface{})
	defer close(responseCh)

	if q.getTotalDeQueue() <= 2 {
		q.log.WithFields(log.Fields{"Listen Item": ""}).Warning("Listener")
		go func() {
			q.listener()
		}()
	}

	q.watcher <- responseCh

	response := <-responseCh
	atomic.AddInt32(&q.qu, 1)
	return response
}

func (q *Queue) listener() {

	for {
		select {
		case responseCh := <-q.watcher:
			responseCh <- q.Dequeue()
			time.Sleep(100 * time.Millisecond)
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
