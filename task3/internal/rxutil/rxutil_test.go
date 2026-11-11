package rxutil

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/reactivex/rxgo/v2"
)

func TestDeferredGraphIsFreshForEverySubscription(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	var calls atomic.Int32
	source := Defer(ctx, func(scope context.Context) rxgo.Observable { return Value(scope, calls.Add(1)) })
	if calls.Load() != 0 {
		t.Fatal("factory is not lazy")
	}
	for i := int32(1); i <= 2; i++ {
		select {
		case item := <-source.Observe():
			if item.Error() || item.V != i {
				t.Fatalf("item=%+v", item)
			}
		case <-ctx.Done():
			t.Fatal(ctx.Err())
		}
	}
}

func TestValueDoesNotFlattenSliceOrDropNilPointer(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	var pointer *int
	for _, value := range []any{[]int{}, []int{1, 2}, pointer} {
		n := 0
		for item := range Value(ctx, value).Observe() {
			n++
			if item.Error() {
				t.Fatal(item.E)
			}
		}
		if n != 1 {
			t.Fatalf("got %d items for %T", n, value)
		}
	}
}
