package eventrouter

import (
	"reflect"
	"strings"
	"sync"
	"testing"
)

type mock struct {
	HandleStub func(Event)
}

func (m mock) Handle(e Event) {
	if m.HandleStub != nil {
		m.HandleStub(e)
	}
}

func TestHandlers(t *testing.T) {
	tests := []struct {
		desc           string
		subscribeRts   []string
		publishRt      string
		expectedCalled int
	}{
		{
			"TopLevelEvent",
			[]string{"event"},
			"event",
			1,
		},
		{
			"TopLevelWildcardEvent",
			[]string{"*"},
			"event",
			1,
		},
		{
			"SecondLevelEvent",
			[]string{"first.second"},
			"first.second",
			1,
		},
		{
			"TopLevelEventTwoHandlers",
			[]string{"event", "event"},
			"event",
			2,
		},
		{
			"TopLevelEventAndWildcardTwoHandlers",
			[]string{"event", "*"},
			"event",
			2,
		},
		{
			"PartialSubscribe",
			[]string{"first"},
			"first.second",
			1,
		},
		{
			"PartialPublish",
			[]string{"first.second"},
			"first",
			0,
		},
		{
			"NoMatchingHandlers",
			[]string{"first"},
			"none",
			0,
		},
		{
			"NoHandlers",
			[]string{},
			"none",
			0,
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			r := New()
			p := "payload"

			var wg sync.WaitGroup
			wg.Add(test.expectedCalled)

			var called int
			h := mock{
				HandleStub: func(e Event) {
					defer wg.Done()
					called++

					// TODO(Erik): when these failures happen, it's almost impossible to debug
					// the output just complains goroutine deadlock
					expectedRt := strings.Split(test.publishRt, ".")
					if !reflect.DeepEqual(e.Route.parts, expectedRt) {
						t.Fatalf("incorrect route; expected: %v, actual: %v", expectedRt, e.Route)
					}

					if e.Payload != p {
						t.Fatalf("incorrect payload; expected: %s, actual: %s", p, e.Payload)
					}
				},
			}

			for _, rt := range test.subscribeRts {
				r.Subscribe(rt, h)
			}

			r.Publish(test.publishRt, p)

			wg.Wait()
			if called != test.expectedCalled {
				t.Fatalf("handler called count incorrect; expected: %d, actual: %d", test.expectedCalled, called)
			}
		})
	}
}

func TestUnsubscribe(t *testing.T) {
	tests := []struct {
		desc           string
		subscribeRts   []string
		unsubscribeRts []string
		publishRt      string
		expectedCalled []int
	}{
		{
			"SingleHandler",
			[]string{"first"},
			[]string{"first"},
			"first",
			[]int{1, 0},
		},
		{
			"MultipleHandlers",
			[]string{"first", "first"},
			[]string{"first"},
			"first",
			[]int{2, 1},
		},
		{
			"NestedHandler",
			[]string{"first.second"},
			[]string{"first.second"},
			"first.second",
			[]int{1, 0},
		},
		{
			"NoMatchingHandlers",
			[]string{"first"},
			[]string{"second"},
			"second",
			[]int{0, 0},
		},
		{
			"NoHandlers",
			[]string{},
			[]string{"first"},
			"first",
			[]int{0, 0},
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			var wg sync.WaitGroup
			wg.Add(test.expectedCalled[0])

			var called int
			handlers := make(map[string][]Handler)

			for _, rt := range test.subscribeRts {
				handlers[rt] = append(handlers[rt], &mock{
					HandleStub: func(e Event) {
						defer wg.Done()
						called++
					},
				})
			}

			r := New(handlers)
			r.Publish(test.publishRt, nil)

			wg.Wait()
			if called != test.expectedCalled[0] {
				t.Fatalf("handler called incorrect times pre-unsubscribe; expected: %d, actual: %d", test.expectedCalled[0], called)
			}

			wg.Add(test.expectedCalled[1])

			for i, rt := range test.unsubscribeRts {
				var h Handler
				hs, ok := handlers[rt]
				if ok {
					h = hs[i]
				}

				r.Unsubscribe(rt, h)
			}

			called = 0
			r.Publish(test.publishRt, nil)

			wg.Wait()
			if called != test.expectedCalled[1] {
				t.Fatalf("handler called incorrect times post-unsubscribe; expected: %d, actual: %d", test.expectedCalled[1], called)
			}
		})
	}
}
