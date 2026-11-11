package service

import (
	"context"

	"github.com/reactivex/rxgo/v2"

	"task3/internal/model"
	"task3/internal/rxutil"
)

type weatherOutcome struct {
	Value model.Weather
	Error string
}

type placesOutcome struct {
	Value []model.PlaceDescription
	Error string
}

func (s *Service) loadWeatherOutcome(ctx context.Context, coordinates model.Coordinates) rxgo.Observable {
	return rxutil.Defer(
		ctx,
		func(scope context.Context) rxgo.Observable {
			return s.LoadWeather(scope, coordinates).Map(
				func(_ context.Context, value any) (any, error) {
					return weatherOutcome{
						Value: value.(model.Weather),
					}, nil
				},
				rxgo.WithContext(scope),
			).OnErrorResumeNext(
				func(err error) rxgo.Observable {
					if scope.Err() != nil {
						return rxgo.Thrown(scope.Err())
					}

					return rxutil.Value(
						scope,
						weatherOutcome{
							Error: err.Error(),
						},
					)
				},
				rxgo.WithContext(scope),
			)
		},
	)
}

func (s *Service) loadPlacesOutcome(ctx context.Context, coordinates model.Coordinates, options DetailsOptions) rxgo.Observable {
	return rxutil.Defer(
		ctx,
		func(scope context.Context) rxgo.Observable {
			return s.loadNearbyDescriptions(scope, coordinates, options).Map(
				func(_ context.Context, value any) (any, error) {
					return placesOutcome{
						Value: value.([]model.PlaceDescription),
					}, nil
				},
				rxgo.WithContext(scope),
			).OnErrorResumeNext(
				func(err error) rxgo.Observable {
					if scope.Err() != nil {
						return rxgo.Thrown(scope.Err())
					}

					return rxutil.Value(
						scope,
						placesOutcome{
							Value: []model.PlaceDescription{},
							Error: err.Error(),
						},
					)
				},
				rxgo.WithContext(scope),
			)
		},
	)
}
