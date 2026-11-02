package console

import (
	"bufio"
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"task3/internal/api"
	"task3/internal/config"
	"task3/internal/model"
	"task3/internal/service"
)

func run(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load configuration: %w", err)
	}

	// 2. Создаём общий HTTP-клиент.
	httpClient := &http.Client{
		Timeout: cfg.HTTPTimeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	defer httpClient.CloseIdleConnections()

	//graphHopper := api.NewGraphHopperClient(httpClient, cfg.GraphHopper)
	//openWeather := api.NewOpenWeatherClient(httpClient, cfg.OpenWeather)
	wikipedia := api.NewWikipediaClient(httpClient, cfg.Wikipedia)

	preview, err := wikipedia.LoadLocationPreview(
		ctx,
		"Цветной проезд (Новосибирск)",
	)
	if err != nil {
		return err
	}

	switch {
	case preview == nil:
		fmt.Println("Статья не найдена")

	case preview.ImageError != "":
		fmt.Printf(
			"Статья: %s\nОшибка изображения: %s\n",
			preview.ArticleTitle,
			preview.ImageError,
		)

	case preview.Image == nil:
		fmt.Printf(
			"У статьи %q нет миниатюры\n",
			preview.ArticleTitle,
		)

	default:
		fmt.Printf(
			"Статья: %s\nИзображение загружено: %v\n",
			preview.ArticleTitle,
			preview.Image.Bounds(),
		)
	}
	return nil
}

func searchLocations(ctx context.Context, svc *service.Service, query string, cfg config.ConfigDetails) ([]model.Location, error) {
	searchCtx, cancel := context.WithTimeout(ctx, cfg.OperationTimeout)
	defer cancel()

	items := svc.SearchLocations(searchCtx, query).Observe()

	select {
	case <-searchCtx.Done():
		return nil, fmt.Errorf(
			"search locations canceled: %w",
			searchCtx.Err(),
		)

	case item, ok := <-items:
		if !ok {
			if err := searchCtx.Err(); err != nil {
				return nil, fmt.Errorf(
					"search locations canceled: %w",
					err,
				)
			}

			return nil, fmt.Errorf(
				"search locations completed without a result",
			)
		}

		if item.Error() {
			return nil, item.E
		}

		locations, ok := item.V.([]model.Location)
		if !ok {
			return nil, fmt.Errorf(
				"unexpected search result type: %T",
				item.V,
			)
		}

		return locations, nil
	}
}

func printLocations(locations []model.Location) {
	fmt.Println("\nНайденные локации:")

	for i, location := range locations {
		fmt.Printf(
			"\n%d.\n"+
				"  Название: %s\n"+
				"  Страна:  %s\n"+
				"  Регион:  %s\n"+
				"  Город:   %s\n"+
				"  Улица:   %s\n"+
				"  Дом:     %s\n"+
				"  Точка:   %.6f, %.6f\n",
			i+1,
			location.Name,
			location.Country,
			location.Region,
			location.City,
			location.Street,
			location.HouseNumber,
			location.Coordinates.Lat,
			location.Coordinates.Lon,
		)
	}
}

func selectLocation(scanner *bufio.Scanner, locations []model.Location) (model.Location, error) {
	if len(locations) == 0 {
		return model.Location{}, fmt.Errorf("no locations available")
	}

	for {
		fmt.Printf("\nВыберите локацию [1–%d]: ", len(locations))
		if !scanner.Scan() {
			if err := scanner.Err(); err != nil {
				return model.Location{}, fmt.Errorf("read location number: %w", err)
			}

			return model.Location{}, fmt.Errorf("input closed before a location was selected")
		}

		input := strings.TrimSpace(scanner.Text())

		number, err := strconv.Atoi(input)
		if err != nil {
			fmt.Println("Введите целое число.")
			continue
		}

		if number < 1 || number > len(locations) {
			fmt.Printf("Номер должен быть от 1 до %d.\n", len(locations))
			continue
		}

		return locations[number-1], nil
	}
}
