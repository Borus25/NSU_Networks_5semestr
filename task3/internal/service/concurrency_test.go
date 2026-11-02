package service

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"task3/internal/model"
)

type controlledWeather struct {
	started chan<- struct{}
	release <-chan struct{}
}

func (w *controlledWeather) LoadWeather(
	ctx context.Context,
	coordinates model.Coordinates,
) (model.Weather, error) {
	select {
	case w.started <- struct{}{}:
	case <-ctx.Done():
		return model.Weather{}, ctx.Err()
	}

	select {
	case <-w.release:
		return model.Weather{
			TemperatureC: 10,
			Description:  "Тестовая погода",
		}, nil

	case <-ctx.Done():
		return model.Weather{}, ctx.Err()
	}
}

type controlledPlaces struct {
	items   []model.PlaceRef
	started chan<- int64
	release <-chan struct{}

	active atomic.Int32
	peak   atomic.Int32
}

func (p *controlledPlaces) FindNearbyPlaces(
	ctx context.Context,
	coordinates model.Coordinates,
	radiusM int,
	limit int,
) ([]model.PlaceRef, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	return p.items, nil
}

func (p *controlledPlaces) LoadPlaceDescription(
	ctx context.Context,
	place model.PlaceRef,
) (model.PlaceDescription, error) {
	active := p.active.Add(1)
	defer p.active.Add(-1)

	// Обновляем максимальное число одновременно работающих вызовов.
	for {
		previous := p.peak.Load()

		if active <= previous {
			break
		}

		if p.peak.CompareAndSwap(previous, active) {
			break
		}
	}

	select {
	case p.started <- place.PageID:
	case <-ctx.Done():
		return model.PlaceDescription{}, ctx.Err()
	}

	select {
	case <-p.release:
		return model.PlaceDescription{
			Place: place,
			Summary: fmt.Sprintf(
				"Описание страницы %d",
				place.PageID,
			),
		}, nil

	case <-ctx.Done():
		return model.PlaceDescription{}, ctx.Err()
	}
}

func TestLoadLocationInfoConcurrency(t *testing.T) {
	const workers = 2
	const placesCount = 6

	ctx, cancel := context.WithTimeout(
		context.Background(),
		5*time.Second,
	)
	defer cancel()

	weatherStarted := make(chan struct{}, 1)
	descriptionStarted := make(chan int64, placesCount)

	// Пока канал открыт, тестовые запросы не завершаются.
	release := make(chan struct{})

	places := &controlledPlaces{
		started: descriptionStarted,
		release: release,
	}

	for i := 0; i < placesCount; i++ {
		places.items = append(places.items, model.PlaceRef{
			PageID:    int64(i + 1),
			Title:     fmt.Sprintf("Объект %d", i+1),
			DistanceM: float64(i * 100),
		})
	}

	weather := &controlledWeather{
		started: weatherStarted,
		release: release,
	}

	svc := New(nil, weather, places)

	items := svc.LoadLocationInfo(
		ctx,
		model.Location{
			Name: "Тестовая локация",
		},
		DetailsOptions{
			RadiusM: 5000,
			Limit:   placesCount,
			Workers: workers,
		},
	).Observe()

	// Погода должна начать загружаться,
	// несмотря на заблокированные описания.
	select {
	case <-weatherStarted:
		t.Log("weather request started")

	case <-ctx.Done():
		t.Fatal("weather request did not start")
	}

	// До завершения хотя бы одного описания должны
	// начать работу два обработчика.
	for i := 0; i < workers; i++ {
		select {
		case pageID := <-descriptionStarted:
			t.Logf("description request started: pageID=%d", pageID)

		case <-ctx.Done():
			t.Fatalf(
				"only %d of %d description workers started",
				i,
				workers,
			)
		}
	}

	// В этой точке погода и два описания одновременно
	// находятся внутри своих методов и ждут release.
	if got := places.active.Load(); got != int32(workers) {
		t.Fatalf(
			"expected %d active descriptions, got %d",
			workers,
			got,
		)
	}

	// Разрешаем текущим и последующим запросам завершаться.
	close(release)

	select {
	case <-ctx.Done():
		t.Fatalf("pipeline did not finish: %v", ctx.Err())

	case item, ok := <-items:
		if !ok {
			t.Fatal("pipeline closed without a result")
		}

		if item.Error() {
			t.Fatalf("pipeline error: %v", item.E)
		}

		info, ok := item.V.(model.LocationInfo)
		if !ok {
			t.Fatalf("unexpected result type: %T", item.V)
		}

		if len(info.Places) != placesCount {
			t.Fatalf(
				"expected %d descriptions, got %d",
				placesCount,
				len(info.Places),
			)
		}
	}

	if got := places.peak.Load(); got != int32(workers) {
		t.Fatalf(
			"expected maximum concurrency %d, got %d",
			workers,
			got,
		)
	}
}
