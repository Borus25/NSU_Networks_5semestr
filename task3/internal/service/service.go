package service

import (
	"context"
	"task3/internal/model"
)

type LocationSearcher interface {
	SearchLocations(ctx context.Context, query string) ([]model.Location, error)
}

type WeatherLoader interface {
	LoadWeather(ctx context.Context, coordinates model.Coordinates) (model.Weather, error)
}

type PlacesSearcher interface {
	FindNearbyPlaces(ctx context.Context, coordinates model.Coordinates, radiusM int, limit int) ([]model.PlaceRef, error)
	LoadPlaceDescription(ctx context.Context, place model.PlaceRef) (model.PlaceDescription, error)
}

type Service struct {
	locations LocationSearcher
	weather   WeatherLoader
	places    PlacesSearcher
}

func New(locations LocationSearcher, weather WeatherLoader, places PlacesSearcher) *Service {
	return &Service{
		locations: locations,
		weather:   weather,
		places:    places,
	}
}
