package service

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"testing"
	"time"

	"task3/internal/model"
)

// Тестовая реализация WeatherLoader.
type weatherStub struct {
	result model.Weather
	err    error
}

func (s weatherStub) LoadWeather(
	ctx context.Context,
	coordinates model.Coordinates,
) (model.Weather, error) {
	if err := ctx.Err(); err != nil {
		return model.Weather{}, err
	}

	return s.result, s.err
}

// Тестовая реализация PlacesClient.
// Поля после создания не изменяются, поэтому их можно
// читать из нескольких обработчиков Map.
type placesStub struct {
	places         []model.PlaceRef
	searchErr      error
	descriptions   map[int64]string
	descriptionErr error
}

func (s placesStub) FindNearbyPlaces(
	ctx context.Context,
	coordinates model.Coordinates,
	radiusM int,
	limit int,
) ([]model.PlaceRef, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	return s.places, s.searchErr
}

func (s placesStub) LoadPlaceDescription(
	ctx context.Context,
	place model.PlaceRef,
) (model.PlaceDescription, error) {
	if err := ctx.Err(); err != nil {
		return model.PlaceDescription{}, err
	}

	if s.descriptionErr != nil {
		return model.PlaceDescription{}, s.descriptionErr
	}

	summary, ok := s.descriptions[place.PageID]
	if !ok {
		return model.PlaceDescription{}, fmt.Errorf(
			"test description not found for page %d",
			place.PageID,
		)
	}

	return model.PlaceDescription{
		Place:   place,
		Summary: summary,
	}, nil
}

// Ждём единственный результат с ограничением времени.
func awaitDetailsForTest(
	t *testing.T,
	svc *Service,
	location model.Location,
) (model.LocationInfo, error) {
	t.Helper()

	ctx, cancel := context.WithTimeout(
		context.Background(),
		3*time.Second,
	)
	defer cancel()

	items := svc.LoadLocationInfo(
		ctx,
		location,
		DetailsOptions{
			RadiusM: 5000,
			Limit:   10,
			Workers: 2,
		},
	).Observe()

	select {
	case <-ctx.Done():
		t.Fatalf("pipeline did not return a result: %v", ctx.Err())
		return model.LocationInfo{}, ctx.Err()

	case item, ok := <-items:
		if !ok {
			t.Fatal("pipeline closed without a result")
			return model.LocationInfo{}, errors.New("no result")
		}

		if item.Error() {
			return model.LocationInfo{}, item.E
		}

		info, ok := item.V.(model.LocationInfo)
		if !ok {
			t.Fatalf("unexpected result type: %T", item.V)
			return model.LocationInfo{}, errors.New("invalid type")
		}

		return info, nil
	}
}

func TestLoadLocationInfoSuccess(t *testing.T) {
	location := model.Location{
		Name: "Тестовая локация",
		Coordinates: model.Coordinates{
			Lat: 55,
			Lon: 83,
		},
	}

	weather := model.Weather{
		TemperatureC: -5,
		Description:  "Облачно",
	}

	// Передаём в обратном порядке расстояний:
	// итоговый список должен быть отсортирован.
	far := model.PlaceRef{
		PageID:    2,
		Title:     "Дальний объект",
		DistanceM: 900,
	}

	near := model.PlaceRef{
		PageID:    1,
		Title:     "Ближний объект",
		DistanceM: 100,
	}

	svc := New(
		nil, // Поиск локаций эта операция не использует.
		weatherStub{
			result: weather,
		},
		placesStub{
			places: []model.PlaceRef{far, near},
			descriptions: map[int64]string{
				1: "Описание ближнего объекта",
				2: "Описание дальнего объекта",
			},
		},
	)

	got, err := awaitDetailsForTest(t, svc, location)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := model.LocationInfo{
		Location: location,
		Weather:  weather,
		Places: []model.PlaceDescription{
			{
				Place:   near,
				Summary: "Описание ближнего объекта",
			},
			{
				Place:   far,
				Summary: "Описание дальнего объекта",
			},
		},
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf(
			"unexpected result:\ngot:  %#v\nwant: %#v",
			got,
			want,
		)
	}
}

func TestLoadLocationInfoNoPlaces(t *testing.T) {
	weather := model.Weather{
		TemperatureC: 10,
		Description:  "Ясно",
	}

	svc := New(
		nil,
		weatherStub{
			result: weather,
		},
		placesStub{
			places: []model.PlaceRef{},
		},
	)

	got, err := awaitDetailsForTest(
		t,
		svc,
		model.Location{Name: "Локация без мест"},
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(got.Places) != 0 {
		t.Fatalf("expected no places, got %d", len(got.Places))
	}

	if !reflect.DeepEqual(got.Weather, weather) {
		t.Fatalf("weather was lost: %#v", got.Weather)
	}
}

func TestLoadLocationInfoDescriptionError(t *testing.T) {
	expectedErr := errors.New("description service unavailable")

	svc := New(
		nil,
		weatherStub{
			result: model.Weather{
				TemperatureC: 10,
			},
		},
		placesStub{
			places: []model.PlaceRef{
				{
					PageID: 1,
					Title:  "Объект",
				},
			},
			descriptionErr: expectedErr,
		},
	)

	_, err := awaitDetailsForTest(
		t,
		svc,
		model.Location{Name: "Тестовая локация"},
	)

	if err == nil {
		t.Fatal("expected an error, got a successful result")
	}
}
