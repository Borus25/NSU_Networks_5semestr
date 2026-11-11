package ui

import (
	"context"
	"fmt"

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

	subscribeResult(ctx, a.service.LoadLocationInfo(ctx, location, options), func(info model.LocationInfo, err error) {
		cancel()
		fyne.Do(func() {
			if a.closed || requestID != a.detailsID {
				return
			}
			a.detailsCancel = nil
			a.updateBusy()
			if err != nil {
				a.status.SetText(fmt.Sprintf("Ошибка загрузки: %s", operationErrorText(err)))
				a.weather.SetText("Не удалось загрузить общий результат.")
				a.setPlacesMessage("Попробуйте выбрать локацию ещё раз.")
				return
			}
			a.showLocationInfo(info)
		})
	})
}

// Only the COMPLETE report is published; no intermediate network results reach widgets.
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
		const hPaToMmHg = 0.7500616

		pressureText := "нет данных"
		if info.Weather.PressureHPa != nil {
			pressureMmHg := *info.Weather.PressureHPa * hPaToMmHg

			pressureText = fmt.Sprintf(
				"%.0f мм рт. ст.",
				pressureMmHg,
			)
		}

		a.weather.SetText(
			fmt.Sprintf(
				"%s\n"+
					"Температура: %.1f °C\n"+
					"Ощущается как: %.1f °C\n"+
					"Влажность: %d %%\n"+
					"Ветер: %.1f м/с\n"+
					"Давление на уровне моря: %s",
				info.Weather.Description,
				info.Weather.TemperatureC,
				info.Weather.FeelsLikeC,
				info.Weather.HumidityPct,
				info.Weather.WindSpeedMPS,
				pressureText,
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
