package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"task3/internal/model"
)

const (
	limitCountResults = 5
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

func (c *GraphHopperClient) SearchLocations(ctx context.Context, query string) ([]model.Location, error) {
	if query == "" {
		return nil, fmt.Errorf("location query must not be empty")
	}

	u, err := url.Parse(c.cfg.GeocodingURL)
	if err != nil {
		return nil, fmt.Errorf("invalid GraphHopper geocoding endpoint")
	}

	params := u.Query()
	params.Set("q", query)
	params.Set("locale", "ru")
	params.Set("limit", strconv.Itoa(limitCountResults))
	params.Set("key", c.cfg.APIKey)

	u.RawQuery = params.Encode()

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("create GraphHopper request")
	}

	request.Header.Set("Accept", "application/json")

	responce, err := c.httpClient.Do(request)
	if err != nil {
		if urlErr, ok := errors.AsType[*url.Error](err); ok {
			return nil, fmt.Errorf(
				"send GraphHopper request: %w",
				urlErr.Err,
			)
		}

		return nil, fmt.Errorf("send GraphHopper request: %w", err)
	}
	defer responce.Body.Close()

	if responce.StatusCode != http.StatusOK {
		return nil, fmt.Errorf(
			"GraphHopper returned HTTP %d (%s)",
			responce.StatusCode,
			http.StatusText(responce.StatusCode),
		)
	}

	var response graphHopperGeocodingResponse

	if err := json.NewDecoder(responce.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf(
			"decode GraphHopper response: %w",
			err,
		)
	}

	locations := make([]model.Location, 0, len(response.Hits))

	for i, hit := range response.Hits {
		if hit.Point == nil {
			return nil, fmt.Errorf(
				"GraphHopper result %d has no coordinates",
				i,
			)
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
}
