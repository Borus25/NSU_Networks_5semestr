package service

import (
	"context"
	"fmt"

	"github.com/reactivex/rxgo/v2"

	"task3/internal/model"
	"task3/internal/rxutil"
)

func (s *Service) LoadPlaceDescription(ctx context.Context, place model.PlaceRef) rxgo.Observable {
	return rxutil.Defer(ctx, func(scope context.Context) rxgo.Observable {
		return s.places.LoadPlaceDescription(scope, place).OnErrorResumeNext(
			func(err error) rxgo.Observable {
				// Отмена всей операции не является
				// частичной ошибкой отдельного места.
				if scope.Err() != nil {
					return rxgo.Thrown(scope.Err())
				}

				return rxutil.Value(
					scope,
					model.PlaceDescription{
						Place: place,
						DescriptionError: fmt.Sprintf(
							"Не удалось загрузить описание: %v",
							err,
						),
					},
				)
			},
			rxgo.WithContext(scope),
		)
	},
	)
}
