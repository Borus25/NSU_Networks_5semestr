package api

import (
	"net/http"

	"task3/internal/config"
)

type GraphHopperClient struct {
	httpClient *http.Client
	cfg        config.GraphHopperConfig
}

func NewGraphHopperClient(httpClient *http.Client, cfg config.GraphHopperConfig) *GraphHopperClient {
	return &GraphHopperClient{
		httpClient: httpClient,
		cfg:        cfg,
	}
}
