package service

import (
	"context"
	"task3/internal/model"
	"task3/internal/rxutil"

	"github.com/reactivex/rxgo/v2"
)

func (s *Service) LoadPlaceDescription(ctx context.Context, place model.PlaceRef) rxgo.Observable {
	return rxutil.Defer(ctx, func(scope context.Context) rxgo.Observable {
		return s.places.LoadPlaceDescription(scope, place)
	})
}
