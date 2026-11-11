package service

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"task3/internal/model"
	"task3/internal/rxutil"

	"github.com/reactivex/rxgo/v2"
)

type mockLocations struct{ calls atomic.Int32 }

func (m *mockLocations) SearchLocations(ctx context.Context, query string) rxgo.Observable {
	m.calls.Add(1)
	return rxutil.Value(ctx, []model.Location{{Name: query}})
}

type mockWeather func(context.Context, model.Coordinates) rxgo.Observable

func (m mockWeather) LoadWeather(ctx context.Context, c model.Coordinates) rxgo.Observable {
	return m(ctx, c)
}

type mockPlaces struct {
	nearby   func(context.Context, model.Coordinates, int, int) rxgo.Observable
	describe func(context.Context, model.PlaceRef) rxgo.Observable
}

func (m mockPlaces) FindNearbyPlaces(ctx context.Context, c model.Coordinates, r, l int) rxgo.Observable {
	return m.nearby(ctx, c, r, l)
}
func (m mockPlaces) LoadPlaceDescription(ctx context.Context, p model.PlaceRef) rxgo.Observable {
	return m.describe(ctx, p)
}
func awaitInfo(t *testing.T, ctx context.Context, source rxgo.Observable) model.LocationInfo {
	t.Helper()
	items := source.Observe()
	select {
	case item, ok := <-items:
		if !ok {
			t.Fatal("no report")
		}
		if item.Error() {
			t.Fatal(item.E)
		}
		info, ok := item.V.(model.LocationInfo)
		if !ok {
			t.Fatalf("unexpected %T", item.V)
		}
		select {
		case _, ok := <-items:
			if ok {
				t.Fatal("more than one report")
			}
		case <-ctx.Done():
			t.Fatal(ctx.Err())
		}
		return info
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}
	return model.LocationInfo{}
}
func immediateWeather(ctx context.Context, _ model.Coordinates) rxgo.Observable {
	return rxutil.Value(ctx, model.Weather{TemperatureC: 12})
}
func options() DetailsOptions { return DetailsOptions{RadiusM: 1000, Limit: 10} }

func TestEveryIndependentDescriptionStartsBeforeAnyCompletes(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	const n = 7 // More than the former pool size of four.
	started := make(chan int64, n)
	weatherStarted, nearbyStarted, release := make(chan struct{}), make(chan struct{}), make(chan struct{})
	refs := make([]model.PlaceRef, n)
	for i := range refs {
		refs[i] = model.PlaceRef{PageID: int64(i + 1), DistanceM: float64(n - i)}
	}
	weather := mockWeather(func(ctx context.Context, _ model.Coordinates) rxgo.Observable {
		return rxgo.Defer([]rxgo.Producer{func(ctx context.Context, out chan<- rxgo.Item) {
			close(weatherStarted)
			select {
			case <-nearbyStarted:
			case <-ctx.Done():
				return
			}
			rxgo.Of(model.Weather{TemperatureC: 12}).SendContext(ctx, out)
		}}, rxgo.WithContext(ctx))
	})
	places := mockPlaces{
		nearby: func(ctx context.Context, _ model.Coordinates, _, _ int) rxgo.Observable {
			return rxgo.Defer([]rxgo.Producer{func(ctx context.Context, out chan<- rxgo.Item) {
				close(nearbyStarted)
				select {
				case <-weatherStarted:
				case <-ctx.Done():
					return
				}
				rxgo.Of(refs).SendContext(ctx, out)
			}}, rxgo.WithContext(ctx))
		},
		describe: func(ctx context.Context, p model.PlaceRef) rxgo.Observable {
			return rxgo.Defer([]rxgo.Producer{func(ctx context.Context, out chan<- rxgo.Item) {
				started <- p.PageID
				select {
				case <-release:
				case <-ctx.Done():
					return
				}
				rxgo.Of(model.PlaceDescription{Place: p, Summary: fmt.Sprint(p.PageID)}).SendContext(ctx, out)
			}}, rxgo.WithContext(ctx))
		},
	}
	svc := New(&mockLocations{}, weather, places)
	result := svc.LoadLocationInfo(ctx, model.Location{Name: "selected"}, options()).Observe()
	seen := map[int64]bool{}
	for i := 0; i < n; i++ {
		select {
		case id := <-started:
			seen[id] = true
		case <-ctx.Done():
			t.Fatal("independent calls were serialized")
		}
	}
	if len(seen) != n {
		t.Fatal("duplicate description calls")
	}
	select {
	case item := <-result:
		t.Fatalf("intermediate result emitted: %+v", item)
	default:
	}
	close(release)
	select {
	case item := <-result:
		if item.Error() {
			t.Fatal(item.E)
		}
		info := item.V.(model.LocationInfo)
		if info.Location.Name != "selected" || info.Weather.TemperatureC != 12 || len(info.Places) != n {
			t.Fatalf("bad report %+v", info)
		}
		for i, p := range info.Places {
			if p.Place.DistanceM != float64(i+1) {
				t.Fatal("sorting failed")
			}
		}
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}
}

func TestEmptyNearbyAndPartialFailures(t *testing.T) {
	for _, mode := range []string{"empty", "weather", "nearby", "description"} {
		t.Run(mode, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			weather := mockWeather(immediateWeather)
			if mode == "weather" {
				weather = func(context.Context, model.Coordinates) rxgo.Observable {
					return rxgo.Thrown(errors.New("weather failed"))
				}
			}
			places := mockPlaces{
				nearby: func(ctx context.Context, _ model.Coordinates, _, _ int) rxgo.Observable {
					if mode == "nearby" {
						return rxgo.Thrown(errors.New("nearby failed"))
					}
					if mode == "empty" {
						return rxutil.Value(ctx, []model.PlaceRef{})
					}
					return rxutil.Value(ctx, []model.PlaceRef{{PageID: 1}, {PageID: 2}})
				},
				describe: func(ctx context.Context, p model.PlaceRef) rxgo.Observable {
					if mode == "description" && p.PageID == 1 {
						return rxgo.Thrown(errors.New("description failed"))
					}
					return rxutil.Value(ctx, model.PlaceDescription{Place: p, Summary: "ok"})
				},
			}
			info := awaitInfo(t, ctx, New(nil, weather, places).LoadLocationInfo(ctx, model.Location{}, options()))
			switch mode {
			case "empty":
				if info.Places == nil || len(info.Places) != 0 || info.Weather.TemperatureC != 12 {
					t.Fatal(info)
				}
			case "weather":
				if info.WeatherError == "" || len(info.Places) != 2 {
					t.Fatal(info)
				}
			case "nearby":
				if info.PlacesError == "" || info.Weather.TemperatureC != 12 {
					t.Fatal(info)
				}
			case "description":
				if len(info.Places) != 2 || info.Places[0].DescriptionError == "" || info.Places[1].Summary != "ok" {
					t.Fatal(info)
				}
			}
		})
	}
}

func TestCancellationStopsBothBranches(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	started, stopped := make(chan struct{}, 2), make(chan struct{}, 2)
	blocking := func(ctx context.Context) rxgo.Observable {
		return rxgo.Defer([]rxgo.Producer{func(ctx context.Context, out chan<- rxgo.Item) {
			started <- struct{}{}
			<-ctx.Done()
			stopped <- struct{}{}
		}}, rxgo.WithContext(ctx))
	}
	weather := mockWeather(func(ctx context.Context, _ model.Coordinates) rxgo.Observable { return blocking(ctx) })
	places := mockPlaces{nearby: func(ctx context.Context, _ model.Coordinates, _, _ int) rxgo.Observable { return blocking(ctx) }}
	result := New(nil, weather, places).LoadLocationInfo(ctx, model.Location{}, options()).Observe()
	for i := 0; i < 2; i++ {
		select {
		case <-started:
		case <-time.After(2 * time.Second):
			t.Fatal("branch not started")
		}
	}
	cancel()
	for i := 0; i < 2; i++ {
		select {
		case <-stopped:
		case <-time.After(2 * time.Second):
			t.Fatal("branch not cancelled")
		}
	}
	select {
	case item, ok := <-result:
		if ok && !item.Error() {
			t.Fatal("cancellation became a successful report")
		}
	case <-time.After(time.Second):
		t.Fatal("stream did not close")
	}
}

func TestSearchIsLazyAndRepeatable(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	mock := &mockLocations{}
	source := New(mock, nil, nil).SearchLocations(ctx, "query")
	if mock.calls.Load() != 0 {
		t.Fatal("eager search")
	}
	for i := 0; i < 2; i++ {
		select {
		case item := <-source.Observe():
			if item.Error() || item.V.([]model.Location)[0].Name != "query" {
				t.Fatal(item)
			}
		case <-ctx.Done():
			t.Fatal(ctx.Err())
		}
	}
	if mock.calls.Load() != 2 {
		t.Fatal("search did not repeat")
	}
}

func TestInvalidOptionsFailBeforeAPI(t *testing.T) {
	for _, o := range []DetailsOptions{{RadiusM: 9, Limit: 1}, {RadiusM: 10001, Limit: 1}, {RadiusM: 10, Limit: 0}, {RadiusM: 10, Limit: 501}} {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		select {
		case item := <-New(nil, nil, nil).LoadLocationInfo(ctx, model.Location{}, o).Observe():
			if !item.Error() {
				t.Fatal("invalid options accepted")
			}
		case <-ctx.Done():
			t.Fatal(ctx.Err())
		}
		cancel()
	}
}

func TestLocationGraphIsLazyAndRepeatable(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	var weatherCalls, nearbyCalls, descriptionCalls atomic.Int32
	weather := mockWeather(func(ctx context.Context, _ model.Coordinates) rxgo.Observable {
		weatherCalls.Add(1)
		return rxutil.Value(ctx, model.Weather{})
	})
	places := mockPlaces{
		nearby: func(ctx context.Context, _ model.Coordinates, _, _ int) rxgo.Observable {
			nearbyCalls.Add(1)
			return rxutil.Value(ctx, []model.PlaceRef{{PageID: 1}})
		},
		describe: func(ctx context.Context, p model.PlaceRef) rxgo.Observable {
			descriptionCalls.Add(1)
			return rxutil.Value(ctx, model.PlaceDescription{Place: p})
		},
	}
	source := New(nil, weather, places).LoadLocationInfo(ctx, model.Location{}, options())
	if weatherCalls.Load() != 0 || nearbyCalls.Load() != 0 || descriptionCalls.Load() != 0 {
		t.Fatal("eager graph")
	}
	for i := 0; i < 2; i++ {
		info := awaitInfo(t, ctx, source)
		if len(info.Places) != 1 {
			t.Fatal(info)
		}
	}
	if weatherCalls.Load() != 2 || nearbyCalls.Load() != 2 || descriptionCalls.Load() != 2 {
		t.Fatalf("calls: weather=%d nearby=%d description=%d", weatherCalls.Load(), nearbyCalls.Load(), descriptionCalls.Load())
	}
}
