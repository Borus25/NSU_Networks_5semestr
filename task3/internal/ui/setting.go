package ui

import (
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

func (a *App) buildSettings() fyne.CanvasObject {
	// Добавляем исходное значение из настроек,
	// даже если оно отличается от стандартных вариантов.
	radiusValues := []int{1000, 3000, 5000, 10000}
	limitValues := []int{5, 10, 20}

	radiusLabels, radiusByLabel, selectedRadius := makeOptions(
		radiusValues,
		a.detailsOptions.RadiusM,
		func(value int) string {
			return fmt.Sprintf("%d м", value)
		},
	)

	limitLabels, limitByLabel, selectedLimit := makeOptions(
		limitValues,
		a.detailsOptions.Limit,
		func(value int) string {
			return fmt.Sprintf("%d", value)
		},
	)

	radiusSelect := widget.NewSelect(
		radiusLabels,
		func(selected string) {
			if value, ok := radiusByLabel[selected]; ok {
				a.detailsOptions.RadiusM = value

				a.application.Preferences().SetInt(
					preferenceRadius,
					value,
				)
			}
		},
	)
	radiusSelect.SetSelected(selectedRadius)

	limitSelect := widget.NewSelect(
		limitLabels,
		func(selected string) {
			if value, ok := limitByLabel[selected]; ok {
				a.detailsOptions.Limit = value

				a.application.Preferences().SetInt(
					preferenceLimit,
					value,
				)
			}
		},
	)
	limitSelect.SetSelected(selectedLimit)

	return container.NewHBox(
		widget.NewLabel("Радиус:"),
		radiusSelect,
		widget.NewLabel("Объектов, не более:"),
		limitSelect,
	)
}

func makeOptions(
	values []int,
	initial int,
	format func(int) string,
) ([]string, map[string]int, string) {
	labels := make([]string, 0, len(values)+1)
	byLabel := make(map[string]int, len(values)+1)

	add := func(value int) {
		label := format(value)

		if _, exists := byLabel[label]; exists {
			return
		}

		labels = append(labels, label)
		byLabel[label] = value
	}

	for _, value := range values {
		add(value)
	}

	add(initial)

	return labels, byLabel, format(initial)
}
