package service

import (
	"context"
	"task3/internal/model"
	"task3/internal/rxutil"

	"github.com/reactivex/rxgo/v2"
)

func (s *Service) FindNearbyPlaces(ctx context.Context, coordinates model.Coordinates, radiusM int, limit int) rxgo.Observable {
	return rxutil.Defer(ctx, func(scope context.Context) rxgo.Observable {
		return s.places.FindNearbyPlaces(scope, coordinates, radiusM, limit)
	})
}
