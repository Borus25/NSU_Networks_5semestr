package service

import (
	"context"
	"fmt"
	"testing"
	"time"

	"task3/internal/model"
)

type cancelWeather struct {
	started chan<- string
	stopped chan<- string
}

func (w *cancelWeather) LoadWeather(
	ctx context.Context,
	coordinates model.Coordinates,
) (model.Weather, error) {
	const name = "weather"

	select {
	case w.started <- name:
	case <-ctx.Done():
		return model.Weather{}, ctx.Err()
	}

	// Имитируем операцию, которая ждёт отмены.
	<-ctx.Done()

	w.stopped <- name
	return model.Weather{}, ctx.Err()
}

type cancelPlaces struct {
	started chan<- string
	stopped chan<- string
}

func (p *cancelPlaces) FindNearbyPlaces(
	ctx context.Context,
	coordinates model.Coordinates,
	radiusM int,
	limit int,
) ([]model.PlaceRef, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	return []model.PlaceRef{
		{
			PageID: 1,
			Title:  "Первый объект",
		},
		{
			PageID: 2,
			Title:  "Второй объект",
		},
	}, nil
}

func (p *cancelPlaces) LoadPlaceDescription(
	ctx context.Context,
	place model.PlaceRef,
) (model.PlaceDescription, error) {
	name := fmt.Sprintf("description-%d", place.PageID)

	select {
	case p.started <- name:
	case <-ctx.Done():
		return model.PlaceDescription{}, ctx.Err()
	}

	<-ctx.Done()

	p.stopped <- name
	return model.PlaceDescription{}, ctx.Err()
}

func TestLoadLocationInfoCancellation(t *testing.T) {
	const operations = 3 // Погода и два описания.

	// Буферы позволяют методам сообщить о завершении,
	// даже если тест уже вышел по ошибке.
	started := make(chan string, operations)
	stopped := make(chan string, operations)

	// Общий предел времени теста.
	testCtx, testCancel := context.WithTimeout(
		context.Background(),
		5*time.Second,
	)
	defer testCancel()

	// Отдельная отмена проверяемой операции.
	ctx, cancel := context.WithCancel(testCtx)
	defer cancel()

	svc := New(
		nil,
		&cancelWeather{
			started: started,
			stopped: stopped,
		},
		&cancelPlaces{
			started: started,
			stopped: stopped,
		},
	)

	items := svc.LoadLocationInfo(
		ctx,
		model.Location{Name: "Тест отмены"},
		DetailsOptions{
			RadiusM: 5000,
			Limit:   10,
			Workers: 2,
		},
	).Observe()

	// Сначала убеждаемся, что отменяем именно начатые операции.
	startedNames := make(map[string]bool)

	for len(startedNames) < operations {
		select {
		case name := <-started:
			startedNames[name] = true
			t.Logf("started: %s", name)

		case <-testCtx.Done():
			t.Fatalf(
				"not all operations started: %v",
				startedNames,
			)
		}
	}

	t.Log("canceling operation")
	cancel()

	// Проверяем выход всех начатых методов.
	stoppedNames := make(map[string]bool)

	for len(stoppedNames) < operations {
		select {
		case name := <-stopped:
			if !startedNames[name] {
				t.Fatalf("unexpected stopped operation: %s", name)
			}

			stoppedNames[name] = true
			t.Logf("stopped: %s", name)

		case <-testCtx.Done():
			t.Fatalf(
				"some operations did not stop: started=%v stopped=%v",
				startedNames,
				stoppedNames,
			)
		}
	}

	// После отмены выходная последовательность должна закрыться.
	// Ошибка может быть выдана элементом, но успешного
	// LocationInfo в этом сценарии быть не должно.
	for {
		select {
		case item, ok := <-items:
			if !ok {
				return
			}

			if !item.Error() {
				t.Fatalf(
					"unexpected successful result after cancellation: %T",
					item.V,
				)
			}

		case <-testCtx.Done():
			t.Fatal("output channel did not close after cancellation")
		}
	}
}
