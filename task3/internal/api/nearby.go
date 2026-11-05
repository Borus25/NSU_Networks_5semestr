package api

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"task3/internal/model"

	"github.com/reactivex/rxgo/v2"
)

type nearbyPlacesResponse struct {
	Error *wikipediaAPIError `json:"error"`

	Query *struct {
		Geosearch []wikipediaPlace `json:"geosearch"`
	} `json:"query"`
}

type wikipediaPlace struct {
	PageID   int64   `json:"pageid"`
	Title    string  `json:"title"`
	Lat      float64 `json:"lat"`
	Lon      float64 `json:"lon"`
	Distance float64 `json:"dist"`
}

func (c *WikipediaClient) FindNearbyPlaces(ctx context.Context, coordinates model.Coordinates, radiusM int, limit int) rxgo.Observable {
	if radiusM < 10 || radiusM > 10000 {
		return rxgo.Thrown(fmt.Errorf("search radius must be between 10 and 10000 meters"))
	}
	if limit < 1 || limit > 500 {
		return rxgo.Thrown(fmt.Errorf("search limit must be between 1 and 500"))
	}

	lat := strconv.FormatFloat(coordinates.Lat, 'f', -1, 64)
	lon := strconv.FormatFloat(coordinates.Lon, 'f', -1, 64)
	params := url.Values{
		"action": {"query"}, "list": {"geosearch"}, "gscoord": {lat + "|" + lon},
		"gsradius": {strconv.Itoa(radiusM)}, "gslimit": {strconv.Itoa(limit)},
		"gsnamespace": {"0"}, "format": {"json"},
	}
	return decodeJSON[nearbyPlacesResponse](ctx, c.getJSON(ctx, params)).Map(
		func(_ context.Context, value any) (any, error) {
			response := value.(nearbyPlacesResponse)
			if response.Error != nil {
				return nil, fmt.Errorf("wikipedia API error %s: %s", response.Error.Code, response.Error.Info)
			}
			if response.Query == nil {
				return nil, fmt.Errorf("wikipedia response has no query object")
			}

			places := make([]model.PlaceRef, 0, len(response.Query.Geosearch))
			for _, item := range response.Query.Geosearch {
				places = append(places, model.PlaceRef{PageID: item.PageID,
					Title: item.Title,
					Coordinates: model.Coordinates{
						Lat: item.Lat,
						Lon: item.Lon,
					},
					DistanceM: item.Distance,
				})
			}

			return places, nil
		}, rxgo.WithContext(ctx))
}
