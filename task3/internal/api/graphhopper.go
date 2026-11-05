package api

import (
	"task3/internal/asynchttp"

	"task3/internal/config"
)

type GraphHopperClient struct {
	httpClient *asynchttp.Client
	cfg        config.GraphHopperConfig
}

func NewGraphHopperClient(httpClient *asynchttp.Client, cfg config.GraphHopperConfig) *GraphHopperClient {
	return &GraphHopperClient{
		httpClient: httpClient,
		cfg:        cfg,
	}
}
