package api

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"net/http"
	"net/url"
	"task3/internal/asynchttp"

	"github.com/reactivex/rxgo/v2"
)

const (
	maxThumbnailBytes  = 4 * 1024 * 1024
	maxThumbnailPixels = 16 * 1024 * 1024
)

func (c *WikipediaClient) loadThumbnail(ctx context.Context, source string) rxgo.Observable {
	u, err := url.Parse(source)
	if err != nil {
		return rxgo.Thrown(fmt.Errorf("invalid thumbnail URL"))
	}
	allowedHost := u.Host == "upload.wikimedia.org" || u.Host == "thumb.wikimedia.org"
	if u.Scheme != "https" || !allowedHost || u.User != nil {
		return rxgo.Thrown(fmt.Errorf("unsupported thumbnail URL: scheme=%q host=%q", u.Scheme, u.Host))
	}
	headers := http.Header{"User-Agent": {c.cfg.UserAgent}, "Accept": {"image/jpeg, image/png"}}
	return c.httpClient.Get(ctx, u.String(), headers, maxThumbnailBytes).Map(
		func(ctx context.Context, value any) (any, error) {
			if err := ctx.Err(); err != nil {
				return nil, err
			}
			data := value.(asynchttp.Response).Body
			config, _, err := image.DecodeConfig(bytes.NewReader(data))
			if err != nil {
				return nil, fmt.Errorf("read thumbnail dimensions: %w", err)
			}
			if config.Width <= 0 || config.Height <= 0 ||
				int64(config.Width)*int64(config.Height) > maxThumbnailPixels {
				return nil, fmt.Errorf("thumbnail dimensions exceed limit")
			}
			decoded, _, err := image.Decode(bytes.NewReader(data))
			if err != nil {
				return nil, fmt.Errorf("decode thumbnail: %w", err)
			}
			if err := ctx.Err(); err != nil {
				return nil, err
			}
			return decoded, nil
		}, rxgo.WithContext(ctx))
}
