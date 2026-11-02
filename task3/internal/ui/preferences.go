package ui

import (
	"fyne.io/fyne/v2"

	"task3/internal/service"
)

const (
	preferenceRadius = "search.radius_m"
	preferenceLimit  = "search.limit"
)

func restoreDetailsOptions(
	preferences fyne.Preferences,
	defaults service.DetailsOptions,
) service.DetailsOptions {
	options := defaults

	radius := preferences.IntWithFallback(
		preferenceRadius,
		defaults.RadiusM,
	)

	if radius >= 10 && radius <= 10000 {
		options.RadiusM = radius
	}

	limit := preferences.IntWithFallback(
		preferenceLimit,
		defaults.Limit,
	)

	if limit >= 1 && limit <= 500 {
		options.Limit = limit
	}

	// Workers остаётся из конфигурации приложения.
	return options
}
