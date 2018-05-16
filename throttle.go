package main

import "sync"

// Throttle ...
type Throttle struct {
	limit  int
	cond   *sync.Cond
	curCnt int
}

// NewThrottle ...
func NewThrottle(limit int) *Throttle {
	return &Throttle{
		cond:   sync.NewCond(&sync.Mutex{}),
		limit:  limit,
		curCnt: limit,
	}
}

// Reset ..
func (t *Throttle) Reset() {
	t.cond.L.Lock()
	t.curCnt = t.limit
	t.cond.L.Unlock()
	t.cond.Broadcast()
}

// CheckLimit ..
func (t *Throttle) CheckLimit() bool {
	t.cond.L.Lock()
	defer t.cond.L.Unlock()

	if t.curCnt == 0 {
		t.cond.Wait()
		return false
	}
	t.curCnt--
	return true
}
