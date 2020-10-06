package worker_test

import (
	"context"
	"testing"
	"time"

	"app/worker"
)

func TestManager(t *testing.T) {
	h := newTestHelper(t)
	m := worker.NewManager()
	m.NewTicker = h.newTicker
	id := 42

	h.assertExecutionCountToBe(0)
	m.Start(id, time.Second, func(ctx context.Context) {
		h.markExecuted()
	})

	h.timeTick()
	h.assertExecutionCountToBe(1)

	h.timeTick()
	h.assertExecutionCountToBe(2)

	m.Stop(id)

	h.timeTick()
	h.assertExecutionCountToBe(2)
}

func newTestHelper(t *testing.T) *testHelper {
	return &testHelper{
		done:           make(chan struct{}),
		executionCount: 0,
		tickerC:        make(chan time.Time),
		t:              t,
	}
}

type testHelper struct {
	done           chan struct{}
	executionCount int
	tickerC        chan time.Time
	t              *testing.T
}

func (h *testHelper) markExecuted() {
	h.executionCount++
	h.done <- struct{}{}
}

func (h *testHelper) timeTick() {
	select {
	case h.tickerC <- time.Now():
	case <-time.After(100 * time.Millisecond):
	}
	select {
	case <-h.done:
	case <-time.After(100 * time.Millisecond):
	}
}

func (h *testHelper) newTicker(d time.Duration) *time.Ticker {
	return &time.Ticker{C: h.tickerC}
}

func (h *testHelper) assertExecutionCountToBe(expectedCount int) {
	if h.executionCount != expectedCount {
		h.t.Errorf("Expected execution count be '%d'. Got '%d'", expectedCount, h.executionCount)
	}
}
