package main

import (
	"encoding/json"
	"sync"
)

type EventBus struct {
	mu          sync.RWMutex
	subscribers map[chan []byte]struct{}
}

func NewEventBus() *EventBus {
	return &EventBus{subscribers: make(map[chan []byte]struct{})}
}

func (b *EventBus) Subscribe() chan []byte {
	ch := make(chan []byte, 64)
	b.mu.Lock()
	b.subscribers[ch] = struct{}{}
	b.mu.Unlock()
	return ch
}

func (b *EventBus) Unsubscribe(ch chan []byte) {
	b.mu.Lock()
	if _, ok := b.subscribers[ch]; ok {
		delete(b.subscribers, ch)
		close(ch)
	}
	b.mu.Unlock()
}

func (b *EventBus) Publish(v any) {
	payload, err := json.Marshal(v)
	if err != nil {
		return
	}

	b.mu.RLock()
	defer b.mu.RUnlock()
	for ch := range b.subscribers {
		select {
		case ch <- payload:
		default:
		}
	}
}
