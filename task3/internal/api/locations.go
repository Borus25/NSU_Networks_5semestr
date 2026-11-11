package api

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"task3/internal/model"

	"github.com/reactivex/rxgo/v2"
)

const (
	limitCountResults = 5
	maxBodySize       = 2 * 1024 * 1024
)

type graphHopperGeocodingResponse struct {
	Hits []graphHopperLocation `json:"hits"`
}

type graphHopperPoint struct {
	Lat float64 `json:"lat"`
	Lng float64 `json:"lng"`
}

type graphHopperLocation struct {
	Name        string            `json:"name"`
	Country     string            `json:"country"`
	State       string            `json:"state"`
	City        string            `json:"city"`
	Street      string            `json:"street"`
	HouseNumber string            `json:"housenumber"`
	Point       *graphHopperPoint `json:"point"`
}

func (c *GraphHopperClient) SearchLocations(ctx context.Context, query string) rxgo.Observable {
	query = strings.TrimSpace(query)
	if query == "" {
		return rxgo.Thrown(fmt.Errorf("location query must not be empty"))
	}

	u, err := url.Parse(c.cfg.GeocodingURL)
	if err != nil {
		return rxgo.Thrown(fmt.Errorf("invalid GraphHopper geocoding endpoint"))
	}

	params := u.Query()
	params.Set("q", query)
	params.Set("locale", "ru")
	params.Set("limit", strconv.Itoa(limitCountResults))
	params.Set("key", c.cfg.APIKey)
	u.RawQuery = params.Encode()
	headers := http.Header{"Accept": {"application/json"}}

	return decodeJSON[graphHopperGeocodingResponse](ctx, c.httpClient.Get(ctx, u.String(), headers, maxBodySize)).Map(func(_ context.Context, value any) (any, error) {
		response := value.(graphHopperGeocodingResponse)
		locations := make([]model.Location, 0, len(response.Hits))
		for i, hit := range response.Hits {
			if hit.Point == nil {
				return nil, fmt.Errorf("GraphHopper result %d has no coordinates", i)
			}

			locations = append(locations, model.Location{
				Name:        hit.Name,
				Country:     hit.Country,
				Region:      hit.State,
				City:        hit.City,
				Street:      hit.Street,
				HouseNumber: hit.HouseNumber,
				Coordinates: model.Coordinates{
					Lat: hit.Point.Lat,
					Lon: hit.Point.Lng,
				},
			})
		}
		return locations, nil
	}, rxgo.WithContext(ctx))
}
