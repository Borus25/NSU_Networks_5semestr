package service

import (
	"context"
	"fmt"
	"sort"
	"task3/internal/model"
	"task3/internal/rxutil"

	"github.com/reactivex/rxgo/v2"
)

type DetailsOptions struct {
	RadiusM int
	Limit   int
	Workers int
}

// Each lane subscribes to at most one description at a time.
// A slot stays occupied through BOTH the article request and thumbnail request.
func (s *Service) describePlaces(ctx context.Context, places []model.PlaceRef, workers int) rxgo.Observable {
	return rxutil.Defer(ctx, func(scope context.Context) rxgo.Observable {
		if len(places) == 0 {
			return rxutil.Value(scope, []model.PlaceDescription{})
		}
		if workers > len(places) {
			workers = len(places)
		}
		lanes := make([]rxgo.Observable, 0, workers)
		for lane := 0; lane < workers; lane++ {
			refs := make([]model.PlaceRef, 0)
			for index := lane; index < len(places); index += workers {
				refs = append(refs, places[index])
			}
			source := rxgo.Defer([]rxgo.Producer{func(ctx context.Context, out chan<- rxgo.Item) {
				for _, ref := range refs {
					if !rxgo.Of(ref).SendContext(ctx, out) {
						return
					}
				}
			}}, rxgo.WithContext(scope))
			// RxGo v2.5 FlatMap consumes one inner Observable to completion.
			lanes = append(lanes, source.FlatMap(func(item rxgo.Item) rxgo.Observable {
				if item.Error() {
					return rxgo.Thrown(item.E)
				}
				place, ok := item.V.(model.PlaceRef)
				if !ok {
					return rxgo.Thrown(fmt.Errorf("unexpected place type: %T", item.V))
				}
				return s.LoadPlaceDescription(scope, place)
			}, rxgo.WithContext(scope)))
		}
		// RxGo v2.5 Merge sends without a cancellation select. Bound its output
		// by the maximum number of events so early cancellation cannot strand senders.
		return rxgo.Merge(lanes, rxgo.WithContext(scope), rxgo.WithBufferedChannel(len(places)+workers)).
			BufferWithCount(len(places), rxgo.WithContext(scope)).Map(
			func(_ context.Context, value any) (any, error) {
				buffer, ok := value.([]any)
				if !ok || len(buffer) != len(places) {
					return nil, fmt.Errorf("incomplete descriptions")
				}
				result := make([]model.PlaceDescription, 0, len(buffer))
				for _, value := range buffer {
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
			}, rxgo.WithContext(scope))
	})
}

func (s *Service) loadNearbyDescriptions(ctx context.Context, coordinates model.Coordinates, options DetailsOptions) rxgo.Observable {
	return rxutil.Defer(ctx, func(scope context.Context) rxgo.Observable {
		return s.FindNearbyPlaces(scope, coordinates, options.RadiusM, options.Limit).FlatMap(
			func(item rxgo.Item) rxgo.Observable {
				if item.Error() {
					return rxgo.Thrown(item.E)
				}
				places, ok := item.V.([]model.PlaceRef)
				if !ok {
					return rxgo.Thrown(fmt.Errorf("unexpected nearby places type: %T", item.V))
				}
				return s.describePlaces(scope, places, options.Workers)
			}, rxgo.WithContext(scope))
	})
}

func (s *Service) LoadLocationInfo(ctx context.Context, location model.Location, options DetailsOptions) rxgo.Observable {
	if options.Workers <= 0 {
		return rxgo.Thrown(
			fmt.Errorf("description workers must be > 0"),
		)
	}

	if options.RadiusM < 10 || options.RadiusM > 10000 {
		return rxgo.Thrown(
			fmt.Errorf(
				"search radius must be between 10 and 10000 meters",
			),
		)
	}

	if options.Limit < 1 || options.Limit > 500 {
		return rxgo.Thrown(
			fmt.Errorf(
				"search limit must be between 1 and 500",
			),
		)
	}

	return rxutil.Defer(
		ctx,
		func(scope context.Context) rxgo.Observable {
			weather := s.loadWeatherOutcome(
				scope,
				location.Coordinates,
			)

			places := s.loadPlacesOutcome(
				scope,
				location.Coordinates,
				options,
			)

			return rxgo.Merge(
				[]rxgo.Observable{
					weather,
					places,
				},
				rxgo.WithContext(scope),
				rxgo.WithBufferedChannel(4),
			).BufferWithCount(
				2,
				rxgo.WithContext(scope),
			).Map(
				func(
					_ context.Context,
					value any,
				) (any, error) {
					parts, ok := value.([]any)
					if !ok || len(parts) != 2 {
						return nil, fmt.Errorf(
							"incomplete location report",
						)
					}

					result := model.LocationInfo{
						Location: location,
						Places:   []model.PlaceDescription{},
					}

					var weatherSeen, placesSeen bool

					for _, part := range parts {
						switch outcome := part.(type) {
						case weatherOutcome:
							if weatherSeen {
								return nil, fmt.Errorf(
									"duplicate weather outcome",
								)
							}

							weatherSeen = true
							result.Weather = outcome.Value
							result.WeatherError = outcome.Error

						case placesOutcome:
							if placesSeen {
								return nil, fmt.Errorf(
									"duplicate places outcome",
								)
							}

							placesSeen = true
							result.Places = outcome.Value
							result.PlacesError = outcome.Error

						default:
							return nil, fmt.Errorf(
								"unexpected report part: %T",
								part,
							)
						}
					}

					if !weatherSeen || !placesSeen {
						return nil, fmt.Errorf(
							"missing report outcome",
						)
					}

					return result, nil
				},
				rxgo.WithContext(scope),
			)
		},
	)
}
