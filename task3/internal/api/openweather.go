package api

import (
	"net/http"
	"task3/internal/config"
)

type OpenWeatherClient struct {
	httpClient *http.Client
	cfg        config.OpenWeatherConfig
}

func NewOpenWeatherClient(httpClient *http.Client, cfg config.OpenWeatherConfig) *OpenWeatherClient {
	return &OpenWeatherClient{
		httpClient: httpClient,
		cfg:        cfg,
	}
}
