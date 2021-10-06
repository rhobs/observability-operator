package prometheus_operator

import (
	"time"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

// tickerSource is a source.Source that sends stub events every 30 seconds.
// tickerSource can be used as a source which controllers can watch
// to trigger periodic reconciliation loops.
type tickerSource struct {
	source.Channel
	ticker  *time.Ticker
	channel chan event.GenericEvent
}

// newTickerSource creates a new tickerSource
func newTickerSource() *tickerSource {
	channel := make(chan event.GenericEvent, 1)
	return &tickerSource{
		Channel: source.Channel{
			Source: channel,
		},
		ticker:  time.NewTicker(30 * time.Second),
		channel: channel,
	}
}

// tickerSource starts sending events to the source.
func (t *tickerSource) run() {
	t.tick()
	for range t.ticker.C {
		t.tick()
	}
}

// tickerSource sends a single event to the source.
func (t *tickerSource) tick() {
	t.channel <- event.GenericEvent{
		Object: newObjectStub(),
	}
}

func newObjectStub() client.Object {
	return &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{Namespace: "", Name: ""},
	}
}
