package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"task3/internal/api"
	"task3/internal/asynchttp"
	"task3/internal/config"
	"task3/internal/service"
	"task3/internal/ui"
)

const (
	radius = 5000
	count  = 10
)

func main() {
	log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds)

	if err := run(); err != nil {
		log.Printf("application error: %v", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load configuration: %w", err)
	}

	httpClient := &http.Client{
		Timeout: cfg.HTTPTimeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	defer httpClient.CloseIdleConnections()

	asyncClient := asynchttp.New(httpClient)

	graphHopper := api.NewGraphHopperClient(
		asyncClient,
		cfg.GraphHopper,
	)

	openWeather := api.NewOpenWeatherClient(
		asyncClient,
		cfg.OpenWeather,
	)

	wikipedia := api.NewWikipediaClient(
		asyncClient,
		cfg.Wikipedia,
	)

	svc := service.New(graphHopper, openWeather, wikipedia)

	gui := ui.New(
		svc,
		cfg.OperationTimeout,
		service.DetailsOptions{
			RadiusM: radius,
			Limit:   count,
			Workers: cfg.DetailsWorkers,
		},
		cfg.Wikipedia.APIURL,
	)
	gui.Run()

	return nil
}
