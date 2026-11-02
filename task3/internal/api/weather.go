package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"task3/internal/model"
)

type currentWeatherResponse struct {
	Main *struct {
		Temperature float64 `json:"temp"`
		FeelsLike   float64 `json:"feels_like"`
		Humidity    int     `json:"humidity"`
	} `json:"main"`

	Wind *struct {
		Speed float64 `json:"speed"`
	} `json:"wind"`

	Weather []struct {
		Description string `json:"description"`
	} `json:"weather"`
}

func (c *OpenWeatherClient) LoadWeather(ctx context.Context, coordinates model.Coordinates) (model.Weather, error) {
	u, err := url.Parse(c.cfg.WeatherURL)
	if err != nil {
		return model.Weather{}, fmt.Errorf("invalid weather endpoint")
	}

	params := u.Query()
	params.Set("lat", strconv.FormatFloat(coordinates.Lat, 'f', -1, 64))
	params.Set("lon", strconv.FormatFloat(coordinates.Lon, 'f', -1, 64))
	params.Set("appid", c.cfg.APIKey)
	params.Set("units", "metric")
	params.Set("lang", "ru")

	u.RawQuery = params.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return model.Weather{}, fmt.Errorf("create weather request")
	}

	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		if urlErr, ok := errors.AsType[*url.Error](err); ok {
			return model.Weather{}, fmt.Errorf("send weather request: %w", urlErr.Err)
		}

		return model.Weather{}, fmt.Errorf("send weather request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return model.Weather{}, fmt.Errorf("OpenWeather returned HTTP %d (%s)", resp.StatusCode, http.StatusText(resp.StatusCode))
	}

	var response currentWeatherResponse

	if err = json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return model.Weather{}, fmt.Errorf("decode weather response: %w", err)
	}

	if response.Main == nil || response.Wind == nil {
		return model.Weather{}, fmt.Errorf("weather response has no main or wind data")
	}

	if len(response.Weather) == 0 {
		return model.Weather{}, fmt.Errorf("weather response has no description")
	}

	return model.Weather{
		TemperatureC: response.Main.Temperature,
		FeelsLikeC:   response.Main.FeelsLike,
		HumidityPct:  response.Main.Humidity,
		WindSpeedMPS: response.Wind.Speed,
		Description:  response.Weather[0].Description,
	}, nil
}
