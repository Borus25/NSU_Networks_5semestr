package ui

import (
	"context"
	"fmt"
	"task3/internal/service"

	"fyne.io/fyne/v2"

	"task3/internal/model"
)

// Вызывается из обработчика кнопки, то есть в UI-контексте.
func (a *App) onLocationSelected(location model.Location) {
	// Предыдущий выбранный объект больше не актуален.
	if a.detailsCancel != nil {
		a.detailsCancel()
	}

	a.detailsID++
	requestID := a.detailsID

	ctx, cancel := context.WithTimeout(
		a.ctx,
		a.operationTimeout,
	)
	a.detailsCancel = cancel

	a.updateBusy()

	title := locationTitle(location)
	options := a.detailsOptions

	a.status.SetText(
		fmt.Sprintf(
			"Загрузка: %s. Радиус: %d м, объектов не более %d.",
			title,
			options.RadiusM,
			options.Limit,
		),
	)

	a.weather.SetText("Загрузка погоды...")
	a.setPlacesMessage("Загрузка мест и описаний...")

	go func() {
		defer cancel()

		info, err := a.awaitLocationInfo(ctx, location, options)

		fyne.Do(func() {
			if a.closed || requestID != a.detailsID {
				return
			}

			a.detailsCancel = nil
			a.updateBusy()

			if err != nil {
				a.status.SetText(
					fmt.Sprintf(
						"Ошибка загрузки: %s",
						operationErrorText(err),
					),
				)

				a.weather.SetText(
					"Не удалось загрузить общий результат.",
				)
				a.setPlacesMessage(
					"Попробуйте выбрать локацию ещё раз.",
				)
				return
			}

			a.showLocationInfo(info)
		})
	}()
}

func (a *App) awaitLocationInfo(
	ctx context.Context,
	location model.Location,
	options service.DetailsOptions,
) (model.LocationInfo, error) {
	items := a.service.LoadLocationInfo(
		ctx,
		location,
		options,
	).Observe()

	select {
	case <-ctx.Done():
		return model.LocationInfo{}, ctx.Err()

	case item, ok := <-items:
		if !ok {
			if err := ctx.Err(); err != nil {
				return model.LocationInfo{}, err
			}

			return model.LocationInfo{}, fmt.Errorf(
				"location info completed without a result",
			)
		}

		if item.Error() {
			return model.LocationInfo{}, item.E
		}

		info, ok := item.V.(model.LocationInfo)
		if !ok {
			return model.LocationInfo{}, fmt.Errorf(
				"unexpected location info type: %T",
				item.V,
			)
		}

		return info, nil
	}
}

// Вызывается только в UI-контексте.
// Вызывается только в UI-контексте.
func (a *App) showLocationInfo(info model.LocationInfo) {
	hasErrors := info.WeatherError != "" ||
		info.PlacesError != ""

	for _, place := range info.Places {
		if place.DescriptionError != "" ||
			place.ImageError != "" {
			hasErrors = true
			break
		}
	}

	if hasErrors {
		a.status.SetText(
			fmt.Sprintf(
				"Загружено с частичными ошибками: %s.",
				locationTitle(info.Location),
			),
		)
	} else {
		a.status.SetText(
			fmt.Sprintf(
				"Загружено: %s.",
				locationTitle(info.Location),
			),
		)
	}

	if info.WeatherError != "" {
		a.weather.SetText(
			"Не удалось получить погоду: " +
				info.WeatherError,
		)
	} else {
		a.weather.SetText(
			fmt.Sprintf(
				"%s\n"+
					"Температура: %.1f °C\n"+
					"Ощущается как: %.1f °C\n"+
					"Влажность: %d %%\n"+
					"Ветер: %.1f м/с",
				info.Weather.Description,
				info.Weather.TemperatureC,
				info.Weather.FeelsLikeC,
				info.Weather.HumidityPct,
				info.Weather.WindSpeedMPS,
			),
		)
	}

	if info.PlacesError != "" {
		a.setPlacesMessage(
			"Не удалось получить места поблизости: " +
				info.PlacesError,
		)
		return
	}

	a.showPlaceCards(info.Places)
}
