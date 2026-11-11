package asynchttp

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/reactivex/rxgo/v2"
)

func one(t *testing.T, ctx context.Context, source rxgo.Observable) rxgo.Item {
	t.Helper()
	select {
	case item, ok := <-source.Observe():
		if !ok {
			t.Fatal("stream completed without an item")
		}
		return item
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}
	return rxgo.Item{}
}

func TestGetIsLazyRepeatableAndPreservesBytes(t *testing.T) {
	var calls atomic.Int32
	body := []byte{0, 255, 128, 10, 13, 1}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if r.Header.Get("X-Test") != "original" {
			t.Error("headers not snapshotted")
		}
		w.Header().Set("Content-Type", "text/plain; charset=unavailable-charset")
		_, _ = w.Write(body)
	}))
	defer server.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	headers := http.Header{"X-Test": {"original"}}
	source := New(server.Client()).Get(ctx, server.URL, headers, 64)
	headers.Set("X-Test", "modified")
	if calls.Load() != 0 {
		t.Fatal("request started before subscription")
	}
	for i := 0; i < 2; i++ {
		item := one(t, ctx, source)
		if item.Error() {
			t.Fatal(item.E)
		}
		if !bytes.Equal(item.V.(Response).Body, body) {
			t.Fatal("binary response changed")
		}
	}
	if calls.Load() != 2 {
		t.Fatalf("calls=%d", calls.Load())
	}
}

func TestHTTPFailuresAndLimits(t *testing.T) {
	var redirected atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/status":
			http.Error(w, "remote secret", http.StatusUnauthorized)
		case "/redirect":
			http.Redirect(w, r, "/target", http.StatusFound)
		case "/target":
			redirected.Add(1)
		default:
			_, _ = w.Write([]byte("12345"))
		}
	}))
	defer server.Close()
	for _, test := range []struct {
		name, path string
		limit      int
		want       string
	}{
		{"status", "/status", 1024, "401"}, {"redirect", "/redirect", 1024, "302"},
		{"oversized", "/body", 4, "exceeds"}, {"invalid limit", "/body", 0, "size limit"},
	} {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			item := one(t, ctx, New(server.Client()).Get(ctx, server.URL+test.path+"?key=LOCAL_SECRET", nil, test.limit))
			if !item.Error() || !strings.Contains(item.E.Error(), test.want) {
				t.Fatalf("unexpected item %+v", item)
			}
			if strings.Contains(item.E.Error(), "LOCAL_SECRET") || strings.Contains(item.E.Error(), "remote secret") {
				t.Fatal("secret leaked")
			}
		})
	}
	if redirected.Load() != 0 {
		t.Fatal("redirect followed")
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	item := one(t, ctx, New(nil).Get(ctx, "file:///etc/passwd", nil, 10))
	if !item.Error() {
		t.Fatal("invalid scheme accepted")
	}
}

func TestCancellationReachesHTTPRequest(t *testing.T) {
	started, stopped := make(chan struct{}), make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		close(started)
		<-r.Context().Done()
		close(stopped)
	}))
	defer server.Close()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	source := New(server.Client()).Get(ctx, server.URL, nil, 1024)
	items := source.Observe()
	select {
	case <-started:
	case <-time.After(2 * time.Second):
		t.Fatal("request never started")
	}
	cancel()
	select {
	case <-stopped:
	case <-time.After(2 * time.Second):
		t.Fatal("HTTP request not cancelled")
	}
	select {
	case item, ok := <-items:
		if ok && (!item.Error() || !errors.Is(item.E, context.Canceled)) {
			t.Fatalf("unexpected result %+v", item)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("subscription did not close")
	}
}

func TestTransportErrorDoesNotExposeQueryKey(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	endpoint := server.URL + "?key=LOCAL_SECRET"
	server.Close()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	item := one(t, ctx, New(server.Client()).Get(ctx, endpoint, nil, 64))
	if !item.Error() {
		t.Fatal("expected transport error")
	}
	if strings.Contains(item.E.Error(), "LOCAL_SECRET") {
		t.Fatal("API key leaked")
	}
}
