package service

import (
	"context"
	"fmt"
	"sort"
	"task3/internal/model"

	"github.com/reactivex/rxgo/v2"
)

type DetailsOptions struct {
	RadiusM int
	Limit   int
	Workers int
}

func (s *Service) describePlaces(ctx context.Context, places []model.PlaceRef, workers int) rxgo.Observable {
	source := rxgo.Defer([]rxgo.Producer{
		func(ctx context.Context, out chan<- rxgo.Item) {
			for _, place := range places {
				if !rxgo.Of(place).SendContext(ctx, out) {
					return
				}
			}
		},
	}, rxgo.WithContext(ctx))

	descriptions := source.Map(
		func(ctx context.Context, value any) (any, error) {
			place, ok := value.(model.PlaceRef)
			if !ok {
				return nil, fmt.Errorf("unexpected place type: %T", value)
			}

			description, err := s.places.LoadPlaceDescription(ctx, place)
			if err != nil {
				return nil, fmt.Errorf("load description for page %d: %w", place.PageID, err)
			}

			return description, nil
		}, rxgo.WithPool(workers), rxgo.WithContext(ctx))

	return descriptions.
		BufferWithCount(len(places), rxgo.WithContext(ctx)).Map(
		func(_ context.Context, value any) (any, error) {
			buffer, ok := value.([]any)
			if !ok {
				return nil, fmt.Errorf("unexpected descriptions buffer type: %T", value)
			}

			if len(buffer) != len(places) {
				return nil, fmt.Errorf("incomplete descriptions: got %d, expected %d", len(buffer), len(places))
			}

			result := make([]model.PlaceDescription, 0, len(buffer))

			for _, value = range buffer {
				description, ok := value.(model.PlaceDescription)
				if !ok {
					return nil, fmt.Errorf("unexpected description type: %T", value)
				}

				result = append(result, description)
			}

			sort.SliceStable(result, func(i, j int) bool {
				if result[i].Place.DistanceM == result[j].Place.DistanceM {
					return result[i].Place.PageID < result[j].Place.PageID
				}

				return result[i].Place.DistanceM < result[j].Place.DistanceM
			})

			return result, nil
		}, rxgo.WithContext(ctx))
}

func (s *Service) loadNearbyDescriptions(
	ctx context.Context,
	coordinates model.Coordinates,
	options DetailsOptions,
) rxgo.Observable {
	return s.FindNearbyPlaces(
		ctx,
		coordinates,
		options.RadiusM,
		options.Limit,
	).FlatMap(
		func(item rxgo.Item) rxgo.Observable {
			if item.Error() {
				return rxgo.Thrown(item.E)
			}

			places, ok := item.V.([]model.PlaceRef)
			if !ok {
				return rxgo.Thrown(fmt.Errorf(
					"unexpected nearby places type: %T",
					item.V,
				))
			}

			// Zip должен получить один элемент даже при отсутствии мест.
			if len(places) == 0 {
				return rxgo.Defer(
					[]rxgo.Producer{
						func(ctx context.Context, out chan<- rxgo.Item) {
							rxgo.Of(
								[]model.PlaceDescription{},
							).SendContext(ctx, out)
						},
					},
					rxgo.WithContext(ctx),
				)
			}

			return s.describePlaces(ctx, places, options.Workers)
		},
		rxgo.WithContext(ctx),
	)
}

func (s *Service) LoadLocationInfo(ctx context.Context, location model.Location, options DetailsOptions) rxgo.Observable {
	if options.Workers <= 0 {
		return rxgo.Thrown(fmt.Errorf("description workers must be > 0"))
	}

	if options.RadiusM < 10 || options.RadiusM > 10000 {
		return rxgo.Thrown(fmt.Errorf(
			"search radius must be between 10 and 10000 meters",
		))
	}

	if options.Limit < 1 || options.Limit > 500 {
		return rxgo.Thrown(fmt.Errorf(
			"search limit must be between 1 and 500",
		))
	}

	weather := s.LoadWeather(ctx, location.Coordinates)
	places := s.loadNearbyDescriptions(ctx, location.Coordinates, options)

	return weather.ZipFromIterable(places,
		func(_ context.Context, weatherValue, placesValue any) (any, error) {
			weatherLocal, ok := weatherValue.(model.Weather)
			if !ok {
				return nil, fmt.Errorf("unexpected weather type: %T", weatherValue)
			}

			descriptions, ok := placesValue.([]model.PlaceDescription)
			if !ok {
				return nil, fmt.Errorf("unexpected descriptions type: %T", placesValue)
			}

			return model.LocationInfo{
				Location: location,
				Weather:  weatherLocal,
				Places:   descriptions,
			}, nil
		}, rxgo.WithContext(ctx))
}
