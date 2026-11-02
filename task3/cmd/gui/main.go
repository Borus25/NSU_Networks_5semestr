package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"task3/internal/api"
	"task3/internal/config"
	"task3/internal/service"
	"task3/internal/ui"
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

		CheckRedirect: func(
			req *http.Request,
			via []*http.Request,
		) error {
			return http.ErrUseLastResponse
		},
	}
	defer httpClient.CloseIdleConnections()

	graphHopper := api.NewGraphHopperClient(
		httpClient,
		cfg.GraphHopper,
	)

	openWeather := api.NewOpenWeatherClient(
		httpClient,
		cfg.OpenWeather,
	)

	wikipedia := api.NewWikipediaClient(
		httpClient,
		cfg.Wikipedia,
	)

	svc := service.New(
		graphHopper,
		openWeather,
		wikipedia,
	)

	gui := ui.New(
		svc,
		cfg.OperationTimeout,
		service.DetailsOptions{
			RadiusM: 5000,
			Limit:   10,
			Workers: cfg.DetailsWorkers,
		},
		cfg.Wikipedia.APIURL,
	)
	gui.Run()

	return nil
}
