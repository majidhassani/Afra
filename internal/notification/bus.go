// Package notification provides an in-process pub/sub bus used to push case
// events to SSE subscribers.
package notification

import (
	"sync"
	"time"

	"github.com/google/uuid"
)

type Event struct {
	ID        uuid.UUID      `json:"id"`
	CaseID    uuid.UUID      `json:"case_id"`
	Type      string         `json:"type"`
	Payload   map[string]any `json:"payload"`
	CreatedAt time.Time      `json:"created_at"`
}

type Bus struct {
	mu   sync.RWMutex
	subs map[uuid.UUID]map[chan Event]struct{}
}

func NewBus() *Bus {
	return &Bus{subs: map[uuid.UUID]map[chan Event]struct{}{}}
}

// Subscribe returns a buffered channel of events for one case and an
// unsubscribe function.
func (b *Bus) Subscribe(caseID uuid.UUID) (<-chan Event, func()) {
	ch := make(chan Event, 16)
	b.mu.Lock()
	if b.subs[caseID] == nil {
		b.subs[caseID] = map[chan Event]struct{}{}
	}
	b.subs[caseID][ch] = struct{}{}
	b.mu.Unlock()

	unsubscribe := func() {
		b.mu.Lock()
		if set, ok := b.subs[caseID]; ok {
			delete(set, ch)
			if len(set) == 0 {
				delete(b.subs, caseID)
			}
		}
		b.mu.Unlock()
	}
	return ch, unsubscribe
}

// Publish delivers the event to all subscribers of its case. Slow
// subscribers are skipped rather than blocking the publisher.
func (b *Bus) Publish(ev Event) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	for ch := range b.subs[ev.CaseID] {
		select {
		case ch <- ev:
		default:
		}
	}
}
