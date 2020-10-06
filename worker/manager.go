package worker

import (
	"context"
	"sync"
	"time"
)

func NewManager() *Manager {
	return &Manager{
		cancels:   map[int]context.CancelFunc{},
		mutex:     &sync.Mutex{},
		NewTicker: time.NewTicker,
	}
}

type Manager struct {
	cancels   map[int]context.CancelFunc
	mutex     *sync.Mutex
	NewTicker func(d time.Duration) *time.Ticker
}

func (m Manager) Start(id int, interval time.Duration, f func(ctx context.Context)) {
	ctx, cancel := context.WithCancel(context.Background())

	m.mutex.Lock()
	m.cancels[id] = cancel
	m.mutex.Unlock()

	go func() {
		ticker := m.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				// prioritize finishing
				select {
				case <-ctx.Done():
					return
				default:
				}
			}
			f(ctx)
		}
	}()
}

func (m Manager) Stop(id int) {
	m.mutex.Lock()
	cancel, ok := m.cancels[id]
	if ok {
		delete(m.cancels, id)
	}
	m.mutex.Unlock()
	if ok {
		cancel()
	}
}
