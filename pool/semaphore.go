package pool

import "sync"

type empty struct{}

type Semaphore struct {
	blockch chan empty
	write   sync.WaitGroup
	waiter  uint64
}

func NewSemaphore(n uint64) *Semaphore {
	return &Semaphore{
		blockch: make(chan empty, n),
		waiter:  n,
	}
}

func (s Semaphore) Acquire(n uint64) {

	var e empty
	for i := 0; uint64(i) < n; i++ {
		s.blockch <- e
	}
}

func (s Semaphore) Release(n uint64) {

	for i := 0; uint64(i) < n; i++ {
		<-s.blockch
	}
}

func (s Semaphore) Lock() {

	//s.write.Add(1)
	s.Acquire(s.waiter)
}

func (s Semaphore) Unlock() {

	s.Release(s.waiter)
	//s.write.Done()
}

func (s Semaphore) RLock() {

	//s.write.Wait()
	s.Acquire(1)
}

func (s Semaphore) RUnlock() {

	s.Release(1)
}
