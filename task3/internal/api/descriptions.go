package api

import (
	"context"
	"encoding/json"
	"fmt"
	"image"
	"net/url"
	"strconv"
	"strings"
	"task3/internal/model"
	"task3/internal/rxutil"

	"github.com/reactivex/rxgo/v2"
)

type placeDescriptionResponse struct {
	Error *wikipediaAPIError `json:"error"`

	Query *struct {
		Pages map[string]wikipediaPageExtract `json:"pages"`
	} `json:"query"`
}

type wikipediaPageExtract struct {
	PageID  int64  `json:"pageid"`
	Title   string `json:"title"`
	Extract string `json:"extract"`

	Missing json.RawMessage `json:"missing"`
	Invalid json.RawMessage `json:"invalid"`

	Thumbnail *struct {
		Source string `json:"source"`
		Width  int    `json:"width"`
		Height int    `json:"height"`
	} `json:"thumbnail"`
}

func (c *WikipediaClient) LoadPlaceDescription(ctx context.Context, place model.PlaceRef) rxgo.Observable {
	if place.PageID <= 0 {
		return rxgo.Thrown(fmt.Errorf("invalid Wikipedia page ID: %d", place.PageID))
	}

	params := url.Values{
		"action": {"query"}, "pageids": {strconv.FormatInt(place.PageID, 10)},
		"exintro": {"1"}, "explaintext": {"1"}, "format": {"json"},
		"prop": {"extracts|pageimages"}, "piprop": {"thumbnail"}, "pithumbsize": {"480"},
	}

	return rxutil.Defer(ctx, func(scope context.Context) rxgo.Observable {
		return decodeJSON[placeDescriptionResponse](scope, c.getJSON(scope, params)).FlatMap(
			func(item rxgo.Item) rxgo.Observable {
				if item.Error() {
					return rxgo.Thrown(item.E)
				}

				response := item.V.(placeDescriptionResponse)
				if response.Error != nil {
					return rxgo.Thrown(fmt.Errorf("wikipedia API error %s: %s", response.Error.Code, response.Error.Info))
				}
				if response.Query == nil {
					return rxgo.Thrown(fmt.Errorf("wikipedia description response has no query object"))
				}

				page, ok := response.Query.Pages[strconv.FormatInt(place.PageID, 10)]
				if !ok || page.PageID != place.PageID || len(page.Missing) != 0 || len(page.Invalid) != 0 {
					return rxgo.Thrown(fmt.Errorf("wikipedia page %d is missing, invalid or was not returned", place.PageID))
				}

				currentPlace := place
				if page.Title != "" {
					currentPlace.Title = page.Title
				}

				result := model.PlaceDescription{Place: currentPlace, Summary: strings.TrimSpace(page.Extract)}
				if page.Thumbnail == nil || page.Thumbnail.Source == "" {
					return rxutil.Value(scope, result)
				}

				result.ImageURL = page.Thumbnail.Source
				return c.loadThumbnail(scope, result.ImageURL).Map(func(_ context.Context, value any) (any, error) {
					result.Image = value.(image.Image)
					return result, nil
				}, rxgo.WithContext(scope)).OnErrorResumeNext(func(err error) rxgo.Observable {
					if scope.Err() != nil {
						return rxgo.Thrown(scope.Err())
					}

					result.ImageError = err.Error()
					return rxutil.Value(scope, result)
				}, rxgo.WithContext(scope))
			}, rxgo.WithContext(scope))
	})
}
