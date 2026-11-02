package api

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"task3/internal/model"
)

type locationPreviewResponse struct {
	Error *wikipediaAPIError `json:"error"`

	Query *struct {
		Pages map[string]wikipediaPageExtract `json:"pages"`
	} `json:"query"`
}

// LoadLocationPreview ищет страницу по точному названию.
// nil, nil означает: подходящей страницы нет.
func (c *WikipediaClient) LoadLocationPreview(
	ctx context.Context,
	articleTitle string,
) (*model.LocationPreview, error) {
	articleTitle = strings.TrimSpace(articleTitle)
	if articleTitle == "" {
		return nil, fmt.Errorf("Wikipedia article title is empty")
	}

	// titles поддерживает несколько названий через "|".
	// Здесь ожидаем ровно одно название.
	if strings.Contains(articleTitle, "|") {
		return nil, fmt.Errorf("expected one Wikipedia article title")
	}

	params := url.Values{}
	params.Set("action", "query")
	params.Set("titles", articleTitle)
	params.Set("redirects", "1")
	params.Set("prop", "pageimages")
	params.Set("piprop", "thumbnail")
	params.Set("pithumbsize", "640")
	params.Set("format", "json")

	var response locationPreviewResponse

	if err := c.getJSON(ctx, params, &response); err != nil {
		return nil, fmt.Errorf("load location preview: %w", err)
	}

	if response.Error != nil {
		return nil, fmt.Errorf(
			"Wikipedia API error %s: %s",
			response.Error.Code,
			response.Error.Info,
		)
	}

	if response.Query == nil {
		return nil, fmt.Errorf("location preview has no query object")
	}

	for _, page := range response.Query.Pages {
		if len(page.Missing) != 0 ||
			len(page.Invalid) != 0 ||
			page.PageID <= 0 {
			continue
		}

		result := &model.LocationPreview{
			PageID:       page.PageID,
			ArticleTitle: page.Title,
		}

		if page.Thumbnail == nil || page.Thumbnail.Source == "" {
			return result, nil
		}

		thumbnail, err := c.loadThumbnail(
			ctx,
			page.Thumbnail.Source,
		)
		if err != nil {
			if ctx.Err() != nil {
				return nil, ctx.Err()
			}

			result.ImageError = err.Error()
			return result, nil
		}

		result.Image = thumbnail
		return result, nil
	}

	return nil, nil
}
