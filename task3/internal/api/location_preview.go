package api

import (
	"context"
	"fmt"
	"image"
	"net/url"
	"strings"
	"task3/internal/model"
	"task3/internal/rxutil"

	"github.com/reactivex/rxgo/v2"
)

type locationPreviewResponse struct {
	Error *wikipediaAPIError `json:"error"`

	Query *struct {
		Pages map[string]wikipediaPageExtract `json:"pages"`
	} `json:"query"`
}

// LoadLocationPreview emits ONE *LocationPreview; a typed nil means no article.
func (c *WikipediaClient) LoadLocationPreview(ctx context.Context, articleTitle string) rxgo.Observable {
	articleTitle = strings.TrimSpace(articleTitle)
	if articleTitle == "" || strings.Contains(articleTitle, "|") {
		return rxgo.Thrown(fmt.Errorf("expected one nonempty Wikipedia article title"))
	}

	params := url.Values{
		"action": {"query"}, "titles": {articleTitle}, "redirects": {"1"},
		"prop": {"pageimages"}, "piprop": {"thumbnail"}, "pithumbsize": {"640"}, "format": {"json"},
	}

	return rxutil.Defer(ctx, func(scope context.Context) rxgo.Observable {
		return decodeJSON[locationPreviewResponse](scope, c.getJSON(scope, params)).FlatMap(
			func(item rxgo.Item) rxgo.Observable {
				if item.Error() {
					return rxgo.Thrown(item.E)
				}

				response := item.V.(locationPreviewResponse)
				if response.Error != nil {
					return rxgo.Thrown(fmt.Errorf("wikipedia API error %s: %s", response.Error.Code, response.Error.Info))
				}
				if response.Query == nil {
					return rxgo.Thrown(fmt.Errorf("location preview has no query object"))
				}

				for _, page := range response.Query.Pages {
					if page.PageID <= 0 || len(page.Missing) != 0 || len(page.Invalid) != 0 {
						continue
					}

					result := &model.LocationPreview{
						PageID:       page.PageID,
						ArticleTitle: page.Title,
					}

					if page.Thumbnail == nil || page.Thumbnail.Source == "" {
						return rxutil.Value(scope, result)
					}

					return c.loadThumbnail(scope, page.Thumbnail.Source).Map(func(_ context.Context, value any) (any, error) {
						result.Image = value.(image.Image)
						return result, nil
					}, rxgo.WithContext(scope)).OnErrorResumeNext(func(err error) rxgo.Observable {
						if scope.Err() != nil {
							return rxgo.Thrown(scope.Err())
						}

						result.ImageError = err.Error()
						return rxutil.Value(scope, result)
					}, rxgo.WithContext(scope))
				}

				return rxutil.Value(scope, (*model.LocationPreview)(nil))
			}, rxgo.WithContext(scope))
	})
}
