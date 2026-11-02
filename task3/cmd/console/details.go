package console

import (
	"context"
	"fmt"

	"task3/internal/model"
	"task3/internal/service"
)

func loadAndPrintDetails(ctx context.Context, svc *service.Service, location model.Location, options service.DetailsOptions) error {
	loadCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	items := svc.LoadLocationInfo(loadCtx, location, options).Observe()

	select {
	case <-loadCtx.Done():
		return fmt.Errorf("load location info canceled: %w", loadCtx.Err())

	case item, ok := <-items:
		if !ok {
			if err := loadCtx.Err(); err != nil {
				return fmt.Errorf("load location info canceled: %w", err)
			}

			return fmt.Errorf("location info completed without a result")
		}

		if item.Error() {
			return item.E
		}

		info, ok := item.V.(model.LocationInfo)
		if !ok {
			return fmt.Errorf("unexpected location info type: %T", item.V)
		}

		printLocationInfo(info)
		return nil
	}
}

func printLocationInfo(info model.LocationInfo) {
	fmt.Printf(
		"\nЛокация: %s\nГород: %s\nКоординаты: %.6f, %.6f\n",
		info.Location.Name,
		info.Location.City,
		info.Location.Coordinates.Lat,
		info.Location.Coordinates.Lon,
	)

	fmt.Printf(
		"\nПогода: %s\n"+
			"Температура: %.1f °C\n"+
			"Ощущается как: %.1f °C\n"+
			"Влажность: %d %%\n"+
			"Ветер: %.1f м/с\n",
		info.Weather.Description,
		info.Weather.TemperatureC,
		info.Weather.FeelsLikeC,
		info.Weather.HumidityPct,
		info.Weather.WindSpeedMPS,
	)

	fmt.Println("\nМеста поблизости:")

	if len(info.Places) == 0 {
		fmt.Println("Объекты не найдены")
		return
	}

	for i, description := range info.Places {
		fmt.Printf("\n%d. %s — %.0f м\n", i+1, description.Place.Title, description.Place.DistanceM)

		if description.Summary == "" {
			fmt.Println("Краткое описание отсутствует")
			continue
		}

		fmt.Println(description.Summary)
	}
}
