package main

import "sync"

type SafeCounter struct {
	mu  sync.Mutex
	val int
}

var anoncounter = SafeCounter{val: 1}

func (counter *SafeCounter) Inc() {
	counter.mu.Lock()
	counter.val++
	counter.mu.Unlock()
}

func (counter *SafeCounter) Val() int {
	counter.mu.Lock()
	defer counter.mu.Unlock()
	return counter.val
}
