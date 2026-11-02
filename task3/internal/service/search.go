package service

import (
	"context"
	"fmt"

	"github.com/reactivex/rxgo/v2"
)

func (s *Service) SearchLocations(ctx context.Context, query string) rxgo.Observable {
	return rxgo.Defer([]rxgo.Producer{
		func(ctx context.Context, out chan<- rxgo.Item) {
			if query == "" {
				rxgo.Error(fmt.Errorf("location query must not be empty")).SendContext(ctx, out)
				return
			}

			locations, err := s.locations.SearchLocations(ctx, query)
			if err != nil {
				rxgo.Error(fmt.Errorf("search locations: %w", err)).SendContext(ctx, out)
				return
			}

			rxgo.Of(locations).SendContext(ctx, out)
		},
	}, rxgo.WithContext(ctx))
}
