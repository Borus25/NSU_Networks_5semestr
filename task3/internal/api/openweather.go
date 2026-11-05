package api

import (
	"task3/internal/asynchttp"
	"task3/internal/config"
)

type OpenWeatherClient struct {
	httpClient *asynchttp.Client
	cfg        config.OpenWeatherConfig
}

func NewOpenWeatherClient(httpClient *asynchttp.Client, cfg config.OpenWeatherConfig) *OpenWeatherClient {
	return &OpenWeatherClient{
		httpClient: httpClient,
		cfg:        cfg,
	}
}
