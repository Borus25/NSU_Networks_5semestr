package ui

import (
	"context"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"task3/internal/model"
	"task3/internal/service"
)

type App struct {
	application fyne.App
	window      fyne.Window

	service          *service.Service
	operationTimeout time.Duration

	ctx    context.Context
	cancel context.CancelFunc

	searchCancel context.CancelFunc
	searchID     uint64
	closed       bool

	foundLocations []model.Location

	queryEntry *widget.Entry
	searchBtn  *widget.Button
	status     *widget.Label
	cancelBtn  *widget.Button
	progress   *widget.ProgressBarInfinite

	locationsBox    *fyne.Container
	weather         *widget.Label
	places          *widget.Label
	detailsOptions  service.DetailsOptions
	placesBox       *fyne.Container
	wikipediaAPIURL string

	detailsCancel context.CancelFunc
	detailsID     uint64
}

func New(
	svc *service.Service,
	operationTimeout time.Duration,
	detailsOptions service.DetailsOptions,
	wikipediaAPIURL string,
) *App {
	application := app.NewWithID("networks.places")

	detailsOptions = restoreDetailsOptions(
		application.Preferences(),
		detailsOptions,
	)

	ctx, cancel := context.WithCancel(context.Background())

	gui := &App{
		application:      application,
		window:           application.NewWindow("Места"),
		service:          svc,
		operationTimeout: operationTimeout,
		detailsOptions:   detailsOptions,
		wikipediaAPIURL:  wikipediaAPIURL,
		ctx:              ctx,
		cancel:           cancel,
	}

	gui.build()

	gui.window.SetOnClosed(func() {
		gui.closed = true
		gui.cancel()
	})

	return gui
}

func (a *App) Run() {
	defer a.cancel()
	a.window.ShowAndRun()
}

func (a *App) build() {
	a.queryEntry = widget.NewEntry()
	a.queryEntry.SetPlaceHolder(
		"Например: Новосибирск, Цветной проезд",
	)

	a.searchBtn = widget.NewButton("Найти", a.onSearch)

	a.queryEntry.OnSubmitted = func(_ string) {
		a.onSearch()
	}

	a.status = widget.NewLabel("Введите название места.")

	a.cancelBtn = widget.NewButton("Отменить", a.onCancel)
	a.cancelBtn.Disable()

	a.progress = widget.NewProgressBarInfinite()
	a.progress.Stop()
	a.progress.Hide()

	a.locationsBox = container.NewVBox(
		widget.NewLabel("Здесь появятся варианты локаций."),
	)

	a.weather = widget.NewLabel(
		"Здесь появится погода для выбранной локации.",
	)
	a.weather.Wrapping = fyne.TextWrapWord

	a.places = widget.NewLabel(
		"Здесь появятся объекты поблизости и их описания.",
	)
	a.places.Wrapping = fyne.TextWrapWord

	a.placesBox = container.NewVBox()

	searchRow := container.NewBorder(
		nil,
		nil,
		nil,
		a.searchBtn,
		a.queryEntry,
	)

	statusRow := container.NewBorder(
		nil,
		nil,
		nil,
		a.cancelBtn,
		a.status,
	)

	settings := a.buildSettings()

	header := container.NewVBox(
		widget.NewLabelWithStyle(
			"Поиск мест",
			fyne.TextAlignLeading,
			fyne.TextStyle{Bold: true},
		),
		searchRow,
		settings,
		statusRow,
		a.progress,
		widget.NewSeparator(),
	)

	results := container.NewVBox(
		widget.NewCard(
			"Найденные локации",
			"",
			a.locationsBox,
		),
		widget.NewCard(
			"Погода",
			"",
			a.weather,
		),
		widget.NewCard(
			"Места поблизости",
			"",
			container.NewVBox(
				a.places,
				a.placesBox,
			),
		),
	)

	a.window.SetContent(
		container.NewBorder(
			header,
			nil,
			nil,
			nil,
			container.NewVScroll(results),
		),
	)

	a.window.Resize(fyne.NewSize(1200, 800))
}
