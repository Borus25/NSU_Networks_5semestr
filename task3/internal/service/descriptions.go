package service

import (
	"context"
	"fmt"

	"github.com/reactivex/rxgo/v2"

	"task3/internal/model"
)

func (s *Service) LoadPlaceDescription(ctx context.Context, place model.PlaceRef) rxgo.Observable {
	return rxgo.Defer([]rxgo.Producer{
		func(ctx context.Context, out chan<- rxgo.Item) {
			description, err := s.places.LoadPlaceDescription(ctx, place)
			if err != nil {
				rxgo.Error(fmt.Errorf("load description for page %d: %w", place.PageID, err)).SendContext(ctx, out)
				return
			}

			rxgo.Of(description).SendContext(ctx, out)
		},
	}, rxgo.WithContext(ctx))
}
