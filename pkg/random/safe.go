package random

import (
	"math/rand"
	"sync"
	"time"
)

type Safe struct {
	mu  sync.Mutex
	rnd *rand.Rand
}

func NewSafe(seed int64) *Safe {
	return &Safe{rnd: rand.New(rand.NewSource(seed))}
}

func New() *Safe {
	return NewSafe(time.Now().UnixNano())
}

func (s *Safe) Intn(n int) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.rnd.Intn(n)
}
