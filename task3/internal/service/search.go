package service

import (
	"context"
	"task3/internal/rxutil"

	"github.com/reactivex/rxgo/v2"
)

func (s *Service) SearchLocations(ctx context.Context, query string) rxgo.Observable {
	return rxutil.Defer(ctx, func(scope context.Context) rxgo.Observable {
		return s.locations.SearchLocations(scope, query)
	})
}
