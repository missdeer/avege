package semaphore

type Semaphore struct {
	c chan int
}

func New(n int) *Semaphore {
	s := &Semaphore{
		c: make(chan int, n),
	}
	return s
}

func (s *Semaphore) Acquire() {
	s.c <- 0
}

func (s *Semaphore) Release() {
	<-s.c
}
