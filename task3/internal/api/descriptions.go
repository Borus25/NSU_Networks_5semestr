package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"task3/internal/model"
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

func (c *WikipediaClient) LoadPlaceDescription(ctx context.Context, place model.PlaceRef) (model.PlaceDescription, error) {
	if place.PageID <= 0 {
		return model.PlaceDescription{}, fmt.Errorf(
			"invalid Wikipedia page ID: %d",
			place.PageID,
		)
	}

	pageID := strconv.FormatInt(place.PageID, 10)

	params := url.Values{}
	params.Set("action", "query")
	params.Set("pageids", pageID)
	params.Set("exintro", "1")
	params.Set("explaintext", "1")
	params.Set("format", "json")
	params.Set("prop", "extracts|pageimages")
	params.Set("piprop", "thumbnail")
	params.Set("pithumbsize", "480")

	var response placeDescriptionResponse
	if err := c.getJSON(ctx, params, &response); err != nil {
		return model.PlaceDescription{}, fmt.Errorf("load Wikipedia page %d: %w", place.PageID, err)
	}

	if response.Query == nil {
		return model.PlaceDescription{}, fmt.Errorf("wikipedia description response has no query object")
	}

	for _, page := range response.Query.Pages {
		if page.PageID != place.PageID {
			continue
		}

		if len(page.Missing) != 0 || len(page.Invalid) != 0 {
			return model.PlaceDescription{}, fmt.Errorf("wikipedia page %d is missing or invalid", place.PageID)
		}

		if page.Title != "" {
			place.Title = page.Title
		}

		result := model.PlaceDescription{
			Place:   place,
			Summary: strings.TrimSpace(page.Extract),
		}

		if page.Thumbnail != nil && page.Thumbnail.Source != "" {
			result.ImageURL = page.Thumbnail.Source

			thumbnail, err := c.loadThumbnail(
				ctx,
				result.ImageURL,
			)

			if err != nil {
				// Отмена операции не должна превращаться
				// в успешный результат с отсутствующей картинкой.
				if ctx.Err() != nil {
					return model.PlaceDescription{}, ctx.Err()
				}

				result.ImageError = err.Error()
			} else {
				result.Image = thumbnail
			}
		}

		if err := ctx.Err(); err != nil {
			return model.PlaceDescription{}, err
		}

		return result, nil
	}

	return model.PlaceDescription{}, fmt.Errorf("wikipedia page %d was not returned", place.PageID)
}
