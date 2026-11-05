package service

import (
	"context"
	"task3/internal/model"
	"task3/internal/rxutil"

	"github.com/reactivex/rxgo/v2"
)

func (s *Service) LoadWeather(ctx context.Context, coordinates model.Coordinates) rxgo.Observable {
	return rxutil.Defer(ctx, func(scope context.Context) rxgo.Observable {
		return s.weather.LoadWeather(scope, coordinates)
	})
}
