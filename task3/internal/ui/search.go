package ui

import (
	"context"
	"fmt"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"task3/internal/model"
)

// onSearch вызывается из UI:
// по нажатию кнопки «Найти» или Enter в поле ввода.
func (a *App) onSearch() {
	query := strings.TrimSpace(a.queryEntry.Text)
	if query == "" {
		a.status.SetText("Ошибка: введите название места.")
		return
	}

	// Отменяем предыдущий поиск.
	if a.searchCancel != nil {
		a.searchCancel()
		a.searchCancel = nil
	}

	// Новый поиск делает предыдущую загрузку деталей неактуальной.
	a.detailsID++

	if a.detailsCancel != nil {
		a.detailsCancel()
		a.detailsCancel = nil
	}

	a.weather.SetText("Выберите локацию для загрузки погоды.")
	a.setPlacesMessage("Выберите локацию для поиска мест поблизости.")

	// Защита от отображения устаревшего результата поиска.
	a.searchID++
	requestID := a.searchID

	ctx, cancel := context.WithTimeout(
		a.ctx,
		a.operationTimeout,
	)
	a.searchCancel = cancel

	a.updateBusy()

	a.foundLocations = nil

	a.locationsBox.Objects = nil
	a.locationsBox.Add(widget.NewLabel("Поиск..."))

	a.status.SetText("Поиск локаций...")

	// Subscribe without waiting for a result in the GUI thread or a manual waiter.
	subscribeResult(ctx, a.service.SearchLocations(ctx, query), func(locations []model.Location, err error) {
		cancel()
		fyne.Do(func() {
			if a.closed || requestID != a.searchID {
				return
			}
			a.searchCancel = nil
			a.updateBusy()
			if err != nil {
				a.locationsBox.Objects = nil
				a.locationsBox.Add(widget.NewLabel("Не удалось получить локации."))
				a.status.SetText(fmt.Sprintf("Ошибка поиска: %s", operationErrorText(err)))
				return
			}
			a.showLocations(locations)
		})
	})
}

// showLocations отображает варианты с кнопками выбора.
// Вызывается только в UI-контексте.
func (a *App) showLocations(locations []model.Location) {
	a.foundLocations = locations
	a.locationsBox.Objects = nil

	if len(locations) == 0 {
		a.locationsBox.Add(
			widget.NewLabel("Локации не найдены."),
		)
		a.status.SetText("Попробуйте уточнить запрос.")
		return
	}

	for i, location := range locations {
		// Отдельное значение для обработчика этой кнопки.
		selected := location

		label := widget.NewLabel(
			fmt.Sprintf(
				"%d. %s\nКоординаты: %.6f, %.6f",
				i+1,
				locationTitle(selected),
				selected.Coordinates.Lat,
				selected.Coordinates.Lon,
			),
		)
		label.Wrapping = fyne.TextWrapWord

		button := widget.NewButton("Выбрать", func() {
			a.onLocationSelected(selected)
		})

		row := container.NewBorder(
			nil,
			nil,
			nil,
			button,
			label,
		)

		a.locationsBox.Add(row)
		a.locationsBox.Add(widget.NewSeparator())
	}

	a.status.SetText(
		fmt.Sprintf(
			"Найдено вариантов: %d. Выберите локацию.",
			len(locations),
		),
	)
}

// locationTitle собирает подпись из доступных адресных полей.
// Пустые значения и точные повторы пропускаются.
func locationTitle(location model.Location) string {
	var parts []string
	seen := make(map[string]bool)

	for _, value := range []string{
		location.Name,
		location.Street,
		location.HouseNumber,
		location.City,
		location.Region,
		location.Country,
	} {
		value = strings.TrimSpace(value)

		if value == "" || seen[value] {
			continue
		}

		seen[value] = true
		parts = append(parts, value)
	}

	if len(parts) == 0 {
		return "Без названия"
	}

	return strings.Join(parts, ", ")
}
