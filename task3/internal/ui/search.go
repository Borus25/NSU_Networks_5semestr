package ui

import (
	"context"
	"fmt"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
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

	backgrounds := make([]*canvas.Rectangle, len(locations))
	labels := make([]*widget.Label, len(locations))
	buttons := make([]*widget.Button, len(locations))

	selectedIndex := -1

	selectRow := func(index int) {
		// Снимаем прежнее выделение.
		if selectedIndex >= 0 {
			backgrounds[selectedIndex].Hide()

			labels[selectedIndex].TextStyle.Bold = false
			labels[selectedIndex].Refresh()

			buttons[selectedIndex].SetText("Выбрать")
		}

		selectedIndex = index

		// Выделяем новую строку.
		backgrounds[index].FillColor = theme.Color(
			theme.ColorNameSelection,
		)
		backgrounds[index].Show()
		backgrounds[index].Refresh()

		labels[index].TextStyle.Bold = true
		labels[index].Refresh()

		buttons[index].SetText("Выбрано")
	}

	for i, location := range locations {
		index := i
		selected := location

		label := widget.NewLabel(
			fmt.Sprintf(
				"%d. %s\nТип объекта: %s\nКоординаты: %.6f, %.6f",
				i+1,
				locationTitle(selected),
				locationType(selected),
				selected.Coordinates.Lat,
				selected.Coordinates.Lon,
			),
		)
		label.Wrapping = fyne.TextWrapWord

		labels[index] = label

		background := canvas.NewRectangle(
			theme.Color(theme.ColorNameSelection),
		)
		background.Hide()
		backgrounds[index] = background

		button := widget.NewButton("Выбрать", func() {
			selectRow(index)
			a.onLocationSelected(selected)
		})
		buttons[index] = button

		content := container.NewBorder(
			nil,
			nil,
			nil,
			button,
			label,
		)

		row := container.NewStack(
			background,
			container.NewPadded(content),
		)

		// Добавляем строку только один раз.
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
