package service

import (
	"context"
	"fmt"

	"github.com/reactivex/rxgo/v2"

	"task3/internal/model"
)

func (s *Service) FindNearbyPlaces(ctx context.Context, coordinates model.Coordinates, radiusM int, limit int) rxgo.Observable {
	return rxgo.Defer([]rxgo.Producer{
		func(ctx context.Context, out chan<- rxgo.Item) {
			places, err := s.places.FindNearbyPlaces(ctx, coordinates, radiusM, limit)
			if err != nil {
				rxgo.Error(fmt.Errorf("find nearby places: %w", err)).SendContext(ctx, out)
				return
			}

			rxgo.Of(places).SendContext(ctx, out)
		},
	}, rxgo.WithContext(ctx))
}
