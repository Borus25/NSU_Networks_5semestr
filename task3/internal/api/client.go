package api

import (
	"context"
	"encoding/json"
	"fmt"
	"task3/internal/asynchttp"

	"github.com/reactivex/rxgo/v2"
)

func decodeJSON[T any](ctx context.Context, source rxgo.Observable) rxgo.Observable {
	return source.Map(func(_ context.Context, value any) (any, error) {
		response, ok := value.(asynchttp.Response)
		if !ok {
			return nil, fmt.Errorf("unexpected HTTP result: %T", value)
		}

		var decoded *T
		if err := json.Unmarshal(response.Body, &decoded); err != nil {
			return nil, fmt.Errorf("decode API response: %w", err)
		}

		if decoded == nil {
			return nil, fmt.Errorf("API response must not be null")
		}
		return *decoded, nil
	}, rxgo.WithContext(ctx))
}
