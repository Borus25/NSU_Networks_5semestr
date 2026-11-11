package ui

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"task3/internal/rxutil"

	"github.com/reactivex/rxgo/v2"
)

type delivery struct {
	value int
	err   error
}

func TestSubscribeResultReturnsWithoutWaiting(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	release := make(chan struct{})
	returned := make(chan struct{})
	result := make(chan delivery, 2)
	source := rxgo.Defer([]rxgo.Producer{func(ctx context.Context, out chan<- rxgo.Item) {
		select {
		case <-release:
		case <-ctx.Done():
			return
		}
		rxgo.Of(42).SendContext(ctx, out)
	}}, rxgo.WithContext(ctx))
	go func() {
		subscribeResult(ctx, source, func(value int, err error) { result <- delivery{value, err} })
		close(returned)
	}()
	select {
	case <-returned:
	case <-ctx.Done():
		t.Fatal("subscription blocked")
	}
	select {
	case <-result:
		t.Fatal("result delivered before completion")
	default:
	}
	close(release)
	select {
	case got := <-result:
		if got.err != nil || got.value != 42 {
			t.Fatal(got)
		}
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}
}
func TestSubscribeResultFailuresCompleteExactlyOnce(t *testing.T) {
	for _, mode := range []string{"error", "empty", "wrong type", "cancelled"} {
		t.Run(mode, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			var source rxgo.Observable
			switch mode {
			case "error":
				source = rxgo.Thrown(errors.New("failure"))
			case "empty":
				source = rxgo.Empty()
			case "wrong type":
				source = rxutil.Value(ctx, "not an int")
			case "cancelled":
				cancel()
				source = rxgo.Never()
			}
			var calls atomic.Int32
			results := make(chan delivery, 2)
			subscribeResult(ctx, source, func(value int, err error) { calls.Add(1); results <- delivery{value, err} })
			select {
			case got := <-results:
				if got.err == nil {
					t.Fatal("missing error")
				}
			case <-time.After(2 * time.Second):
				t.Fatal("callback missing")
			}
			if calls.Load() != 1 {
				t.Fatalf("calls=%d", calls.Load())
			}
		})
	}
}
