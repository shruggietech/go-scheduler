package events

import (
	"testing"
	"time"

	"github.com/shruggietech/go-scheduler/internal/domain"
)

func TestBroker_PublishToSubscribers(t *testing.T) {
	b := NewBroker()
	ch1, cancel1 := b.Subscribe()
	ch2, cancel2 := b.Subscribe()
	defer cancel1()
	defer cancel2()

	if b.SubscriberCount() != 2 {
		t.Fatalf("want 2 subscribers, got %d", b.SubscriberCount())
	}

	b.PublishRun(domain.Run{TaskID: "t1", Outcome: domain.OutcomeSuccess})
	for i, ch := range []<-chan Event{ch1, ch2} {
		select {
		case e := <-ch:
			if e.Kind != KindRun || e.Run == nil || e.Run.TaskID != "t1" {
				t.Fatalf("sub %d got unexpected event: %+v", i, e)
			}
		case <-time.After(time.Second):
			t.Fatalf("sub %d did not receive event", i)
		}
	}
}

func TestBroker_UnsubscribeStopsDelivery(t *testing.T) {
	b := NewBroker()
	ch, cancel := b.Subscribe()
	cancel()
	if b.SubscriberCount() != 0 {
		t.Fatal("unsubscribe should remove the subscriber")
	}
	// Channel is closed; a publish must not panic.
	b.PublishAlert(domain.Alert{Kind: domain.AlertRunFailed})
	if _, ok := <-ch; ok {
		t.Fatal("channel should be closed after unsubscribe")
	}
	// Double cancel is safe.
	cancel()
}

func TestBroker_SlowSubscriberDropsRatherThanBlocks(t *testing.T) {
	b := NewBroker()
	_, cancel := b.Subscribe() // never drained
	defer cancel()

	done := make(chan struct{})
	go func() {
		for i := 0; i < 1000; i++ { // far exceeds the 64 buffer
			b.PublishRun(domain.Run{TaskID: "flood"})
		}
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Publish blocked on a slow subscriber")
	}
}
