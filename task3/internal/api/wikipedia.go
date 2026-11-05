package api

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"task3/internal/asynchttp"
	"task3/internal/config"

	"github.com/reactivex/rxgo/v2"
)

type wikipediaAPIError struct {
	Code string `json:"code"`
	Info string `json:"info"`
}
type WikipediaClient struct {
	httpClient *asynchttp.Client
	cfg        config.WikipediaConfig
}

func NewWikipediaClient(httpClient *asynchttp.Client, cfg config.WikipediaConfig) *WikipediaClient {
	return &WikipediaClient{httpClient: httpClient, cfg: cfg}
}

// getJSON starts no request until observation; the result is a raw HTTP response.
func (c *WikipediaClient) getJSON(ctx context.Context, params url.Values) rxgo.Observable {
	u, err := url.Parse(c.cfg.APIURL)
	if err != nil {
		return rxgo.Thrown(fmt.Errorf("invalid Wikipedia endpoint"))
	}

	u.RawQuery = params.Encode()

	headers := http.Header{"Accept": {"application/json"}, "User-Agent": {c.cfg.UserAgent}}

	return c.httpClient.Get(ctx, u.String(), headers, 2*1024*1024)
}
