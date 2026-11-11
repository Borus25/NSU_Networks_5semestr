package api

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"task3/internal/model"

	"github.com/reactivex/rxgo/v2"
)

type currentWeatherResponse struct {
	Main *struct {
		Temperature float64  `json:"temp"`
		FeelsLike   float64  `json:"feels_like"`
		Humidity    int      `json:"humidity"`
		Pressure    *float64 `json:"pressure"`
	} `json:"main"`

	Wind *struct {
		Speed float64 `json:"speed"`
	} `json:"wind"`

	Weather []struct {
		Description string `json:"description"`
	} `json:"weather"`
}

func (c *OpenWeatherClient) LoadWeather(ctx context.Context, coordinates model.Coordinates) rxgo.Observable {
	u, err := url.Parse(c.cfg.WeatherURL)
	if err != nil {
		return rxgo.Thrown(fmt.Errorf("invalid weather endpoint"))
	}

	params := u.Query()
	params.Set("lat", strconv.FormatFloat(coordinates.Lat, 'f', -1, 64))
	params.Set("lon", strconv.FormatFloat(coordinates.Lon, 'f', -1, 64))
	params.Set("appid", c.cfg.APIKey)
	params.Set("units", "metric")
	params.Set("lang", "ru")
	u.RawQuery = params.Encode()

	headers := http.Header{"Accept": {"application/json"}}

	return decodeJSON[currentWeatherResponse](ctx, c.httpClient.Get(ctx, u.String(), headers, 2*1024*1024)).Map(func(_ context.Context, value any) (any, error) {
		response := value.(currentWeatherResponse)
		if response.Main == nil || response.Wind == nil {
			return nil, fmt.Errorf("weather response has no main or wind data")
		}
		if len(response.Weather) == 0 {
			return nil, fmt.Errorf("weather response has no description")
		}

		return model.Weather{
			TemperatureC: response.Main.Temperature,
			FeelsLikeC:   response.Main.FeelsLike,
			HumidityPct:  response.Main.Humidity,
			WindSpeedMPS: response.Wind.Speed,
			Description:  response.Weather[0].Description,
			PressureHPa:  response.Main.Pressure,
		}, nil
	}, rxgo.WithContext(ctx))
}
