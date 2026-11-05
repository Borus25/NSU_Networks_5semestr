// Package rxutil adapts lazy callback-backed sources to RxGo.
package rxutil

import (
	"context"

	"github.com/reactivex/rxgo/v2"
)

// Value emits the value as ONE item, including slices and nil pointers.
func Value(ctx context.Context, value any) rxgo.Observable {
	return rxgo.Defer([]rxgo.Producer{func(ctx context.Context, out chan<- rxgo.Item) {
		rxgo.Of(value).SendContext(ctx, out)
	}}, rxgo.WithContext(ctx))
}

// Defer creates a fresh graph and cancellation scope for every subscription.
// It forwards stream events, never calls a blocking HTTP client.
func Defer(ctx context.Context, factory func(context.Context) rxgo.Observable) rxgo.Observable {
	return rxgo.Defer([]rxgo.Producer{func(parent context.Context, out chan<- rxgo.Item) {
		scope, cancel := context.WithCancel(parent)
		defer cancel()
		input := factory(scope).Observe(rxgo.WithContext(scope))
		for {
			select {
			case <-scope.Done():
				return
			case item, ok := <-input:
				if !ok {
					return
				}
				if !item.SendContext(scope, out) || item.Error() {
					return
				}
			}
		}
	}}, rxgo.WithContext(ctx))
}
