package api

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"io"
	"net/http"
	"net/url"

	_ "image/jpeg"
	_ "image/png"
)

const (
	maxThumbnailBytes  = 4 * 1024 * 1024
	maxThumbnailPixels = 16 * 1024 * 1024
)

func (c *WikipediaClient) loadThumbnail(
	ctx context.Context,
	source string,
) (image.Image, error) {
	u, err := url.Parse(source)
	if err != nil {
		return nil, fmt.Errorf("invalid thumbnail URL")
	}

	// Не выполняем произвольные запросы по адресу из ответа API.
	allowedHost := u.Host == "upload.wikimedia.org" ||
		u.Host == "thumb.wikimedia.org"

	if u.Scheme != "https" || !allowedHost || u.User != nil {
		return nil, fmt.Errorf(
			"unsupported thumbnail URL: scheme=%q host=%q path=%q hasUserInfo=%t",
			u.Scheme,
			u.Host,
			u.Path,
			u.User != nil,
		)
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		u.String(),
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("create thumbnail request: %w", err)
	}

	req.Header.Set("User-Agent", c.cfg.UserAgent)
	req.Header.Set("Accept", "image/jpeg, image/png")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("download thumbnail: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf(
			"thumbnail returned HTTP %d",
			resp.StatusCode,
		)
	}

	data, err := io.ReadAll(
		io.LimitReader(resp.Body, maxThumbnailBytes+1),
	)
	if err != nil {
		return nil, fmt.Errorf("read thumbnail: %w", err)
	}

	if len(data) > maxThumbnailBytes {
		return nil, fmt.Errorf("thumbnail exceeds size limit")
	}

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

	return decoded, nil
}
