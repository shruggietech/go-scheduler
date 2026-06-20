// Package events provides a small in-process publish/subscribe broker used to
// stream run-state changes and new alerts to connected GUI clients over
// Server-Sent Events. Publishing is non-blocking: a slow subscriber drops
// events rather than stalling the engine.
package events

import (
	"sync"

	"github.com/shruggietech/go-scheduler/internal/domain"
)

// Kind classifies an event.
type Kind string

const (
	KindRun   Kind = "run"
	KindAlert Kind = "alert"
)

// Event is a single notification delivered to subscribers.
type Event struct {
	Kind  Kind          `json:"kind"`
	Run   *domain.Run   `json:"run,omitempty"`
	Alert *domain.Alert `json:"alert,omitempty"`
}

// Broker fans out events to all current subscribers.
type Broker struct {
	mu   sync.RWMutex
	subs map[chan Event]struct{}
}

// NewBroker creates an empty broker.
func NewBroker() *Broker {
	return &Broker{subs: make(map[chan Event]struct{})}
}

// Subscribe registers a new subscriber, returning its event channel and an
// unsubscribe function. The channel is buffered; the caller should drain it.
func (b *Broker) Subscribe() (<-chan Event, func()) {
	ch := make(chan Event, 64)
	b.mu.Lock()
	b.subs[ch] = struct{}{}
	b.mu.Unlock()

	var once sync.Once
	cancel := func() {
		once.Do(func() {
			b.mu.Lock()
			delete(b.subs, ch)
			b.mu.Unlock()
			close(ch)
		})
	}
	return ch, cancel
}

// Publish delivers e to all subscribers without blocking. Events for a
// subscriber whose buffer is full are dropped for that subscriber.
func (b *Broker) Publish(e Event) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	for ch := range b.subs {
		select {
		case ch <- e:
		default:
		}
	}
}

// PublishRun is a convenience for run events.
func (b *Broker) PublishRun(r domain.Run) { b.Publish(Event{Kind: KindRun, Run: &r}) }

// PublishAlert is a convenience for alert events.
func (b *Broker) PublishAlert(a domain.Alert) { b.Publish(Event{Kind: KindAlert, Alert: &a}) }

// SubscriberCount reports the number of active subscribers (for tests/metrics).
func (b *Broker) SubscriberCount() int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return len(b.subs)
}
