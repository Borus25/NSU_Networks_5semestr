package service

import (
	"context"
	"fmt"

	"github.com/reactivex/rxgo/v2"

	"task3/internal/model"
)

func (s *Service) LoadWeather(ctx context.Context, coordinates model.Coordinates) rxgo.Observable {
	return rxgo.Defer([]rxgo.Producer{
		func(ctx context.Context, out chan<- rxgo.Item) {
			weather, err := s.weather.LoadWeather(ctx, coordinates)
			if err != nil {
				rxgo.Error(
					fmt.Errorf("load weather: %w", err),
				).SendContext(ctx, out)
				return
			}

			rxgo.Of(weather).SendContext(ctx, out)
		},
	}, rxgo.WithContext(ctx))
}
