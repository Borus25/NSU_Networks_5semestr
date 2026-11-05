package service

import (
	"context"
	"task3/internal/model"

	"github.com/reactivex/rxgo/v2"
)

type LocationSearcher interface {
	SearchLocations(context.Context, string) rxgo.Observable
}
type WeatherLoader interface {
	LoadWeather(context.Context, model.Coordinates) rxgo.Observable
}
type PlacesSearcher interface {
	FindNearbyPlaces(context.Context, model.Coordinates, int, int) rxgo.Observable
	LoadPlaceDescription(context.Context, model.PlaceRef) rxgo.Observable
}
type Service struct {
	locations LocationSearcher
	weather   WeatherLoader
	places    PlacesSearcher
}

func New(locations LocationSearcher, weather WeatherLoader, places PlacesSearcher) *Service {
	return &Service{locations: locations, weather: weather, places: places}
}
