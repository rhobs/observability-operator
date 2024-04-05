package eventsource

import (
	"time"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

// TickerSource is a source.Source that sends stub events every 30 seconds.
// TickerSource can be used as a source which controllers can watch
// to trigger periodic reconciliation loops.
type TickerSource struct {
	source.Channel
	ticker  *time.Ticker
	channel chan event.GenericEvent
}

// NewTickerSource creates a new TickerSource
func NewTickerSource(interval time.Duration) *TickerSource {
	channel := make(chan event.GenericEvent, 1)
	return &TickerSource{
		Channel: source.Channel{
			Source: channel,
		},
		ticker:  time.NewTicker(interval),
		channel: channel,
	}
}

// Run starts sending events to the source.
func (t *TickerSource) Run() {
	t.tick()
	for range t.ticker.C {
		t.tick()
	}
}

// tick sends a single event to the source.
func (t *TickerSource) tick() {
	t.channel <- event.GenericEvent{
		Object: newObjectStub(),
	}
}

func newObjectStub() client.Object {
	return &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{Namespace: "", Name: ""},
	}
}
