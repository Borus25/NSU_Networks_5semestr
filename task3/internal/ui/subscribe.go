package ui

import (
	"context"
	"fmt"
	"sync"

	"github.com/reactivex/rxgo/v2"
)

type resultEvent[T any] struct {
	value T
	err   error
}

// subscribeResult returns immediately. RxGo owns the subscription; the only
// state retained here is a final-delivery guard, not an intermediate API result.
func subscribeResult[T any](ctx context.Context, source rxgo.Observable, done func(T, error)) {
	var once sync.Once
	finish := func(value T, err error) {
		once.Do(func() { done(value, err) })
	}
	source.Map(func(_ context.Context, value any) (any, error) {
		result, ok := value.(T)
		if !ok {
			return nil, fmt.Errorf("unexpected final result type: %T", value)
		}
		return resultEvent[T]{value: result}, nil
	}, rxgo.WithContext(ctx)).OnErrorReturn(func(err error) any {
		return resultEvent[T]{err: err}
	}, rxgo.WithContext(ctx)).Take(1, rxgo.WithContext(ctx)).ForEach(
		func(value any) {
			if event, ok := value.(resultEvent[T]); ok {
				finish(event.value, event.err)
			}
		},
		func(err error) { var zero T; finish(zero, err) },
		func() {
			var zero T
			err := ctx.Err()
			if err == nil {
				err = fmt.Errorf("operation completed without a result")
			}
			finish(zero, err)
		},
		rxgo.WithContext(ctx),
	)
}
