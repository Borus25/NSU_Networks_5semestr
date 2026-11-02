package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"task3/internal/config"
)

type wikipediaAPIError struct {
	Code string `json:"code"`
	Info string `json:"info"`
}

type WikipediaClient struct {
	httpClient *http.Client
	cfg        config.WikipediaConfig
}

func NewWikipediaClient(httpClient *http.Client, cfg config.WikipediaConfig) *WikipediaClient {
	return &WikipediaClient{
		httpClient: httpClient,
		cfg:        cfg,
	}
}

func (c *WikipediaClient) getJSON(ctx context.Context, params url.Values, dst any) error {
	u, err := url.Parse(c.cfg.APIURL)
	if err != nil {
		return fmt.Errorf("invalid Wikipedia endpoint")
	}

	u.RawQuery = params.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return fmt.Errorf("create Wikipedia request: %w", err)
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", c.cfg.UserAgent)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("send Wikipedia request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("wikipedia returned HTTP %d (%s)", resp.StatusCode, http.StatusText(resp.StatusCode))
	}

	if err = json.NewDecoder(resp.Body).Decode(dst); err != nil {
		return fmt.Errorf("decode Wikipedia response: %w", err)
	}

	return nil
}
