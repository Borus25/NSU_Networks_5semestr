// Package asynchttp supplies a callback-backed asynchronous HTTP interface.
package asynchttp

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"sync"

	"task3/internal/rxutil"

	"github.com/gocolly/colly/v2"
	"github.com/reactivex/rxgo/v2"
)

const maxBodySize = 64 * 1024 * 1024

type Response struct {
	StatusCode int
	Body       []byte
}

type Client struct {
	httpClient *http.Client
}

func New(httpClient *http.Client) *Client {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	return &Client{httpClient: httpClient}
}

// Get returns immediately. Each subscription schedules one Colly async request.
func (c *Client) Get(ctx context.Context, endpoint string, headers http.Header, maxBodyBytes int) rxgo.Observable {
	headers = headers.Clone()
	return rxutil.Defer(ctx, func(scope context.Context) rxgo.Observable {
		return rxgo.FromChannel(c.startGET(scope, endpoint, headers, maxBodyBytes), rxgo.WithContext(scope))
	})
}

func (c *Client) startGET(ctx context.Context, endpoint string, headers http.Header, maxBodyBytes int) <-chan rxgo.Item {
	result := make(chan rxgo.Item, 1)
	var once sync.Once
	finish := func(item rxgo.Item) {
		once.Do(func() { result <- item; close(result) })
	}
	if err := ctx.Err(); err != nil {
		finish(rxgo.Error(err))
		return result
	}
	if maxBodyBytes <= 0 || maxBodyBytes > maxBodySize {
		finish(rxgo.Error(fmt.Errorf("invalid HTTP body size limit")))
		return result
	}
	u, err := url.Parse(endpoint)
	if err != nil || u == nil || (u.Scheme != "https" && u.Scheme != "http") || u.Hostname() == "" || u.User != nil {
		finish(rxgo.Error(fmt.Errorf("invalid HTTP endpoint")))
		return result
	}
	collector := colly.NewCollector(
		colly.Async(true), colly.StdlibContext(ctx),
		colly.IgnoreRobotsTxt(), colly.MaxBodySize(maxBodyBytes+1),
	)
	// Copy configuration, but retain the connection-pooling Transport.
	client := *c.httpClient
	client.CheckRedirect = func(_ *http.Request, _ []*http.Request) error { return http.ErrUseLastResponse }
	collector.SetClient(&client)
	// Handle any HTTP status ourselves; do not expose remote error bodies or URLs.
	collector.ParseHTTPErrorResponse = true
	// The adapter transports bytes, not text. Prevent Colly charset conversion
	// (including its early-return path on unsupported charset declarations).
	collector.OnResponseHeaders(func(response *colly.Response) {
		response.Headers.Set("Content-Type", "application/octet-stream")
	})
	collector.OnResponse(func(response *colly.Response) {
		if err = ctx.Err(); err != nil {
			finish(rxgo.Error(err))
			return
		}
		if response.StatusCode != http.StatusOK {
			finish(rxgo.Error(fmt.Errorf("HTTP request returned status %d", response.StatusCode)))
			return
		}
		if len(response.Body) > maxBodyBytes {
			finish(rxgo.Error(fmt.Errorf("HTTP response exceeds %d bytes", maxBodyBytes)))
			return
		}
		finish(rxgo.Of(Response{StatusCode: response.StatusCode, Body: append([]byte(nil), response.Body...)}))
	})
	collector.OnError(func(response *colly.Response, err error) {
		if ctxErr := ctx.Err(); ctxErr != nil {
			finish(rxgo.Error(ctxErr))
			return
		}
		if response != nil && response.StatusCode != 0 {
			finish(rxgo.Error(fmt.Errorf("HTTP request failed with status %d", response.StatusCode)))
			return
		}
		if urlErr, ok := errors.AsType[*url.Error](err); ok {
			err = urlErr.Err
		}
		finish(rxgo.Error(fmt.Errorf("asynchronous HTTP request failed: %w", err)))
	})
	if err = collector.Request(http.MethodGet, endpoint, nil, nil, headers.Clone()); err != nil {
		// A scheduling error can contain the endpoint. Preserve its type, not secrets.
		finish(rxgo.Error(fmt.Errorf("cannot schedule HTTP request: %T", err)))
	}
	return result
}
