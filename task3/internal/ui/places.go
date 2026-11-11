package ui

import (
	"fmt"
	"net/url"
	"path"
	"strconv"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"task3/internal/model"
)

// setPlacesMessage очищает старые карточки
// и устанавливает сообщение над списком.
// Вызывается только в UI-контексте.
func (a *App) setPlacesMessage(message string) {
	a.places.SetText(message)

	a.placesBox.Objects = nil
	a.placesBox.Refresh()
}

// showPlaceCards отображает уже загруженные описания.
// Новых HTTP-запросов здесь нет.
func (a *App) showPlaceCards(descriptions []model.PlaceDescription) {
	if len(descriptions) == 0 {
		a.setPlacesMessage("Объекты поблизости не найдены.")
		return
	}

	a.setPlacesMessage(
		fmt.Sprintf("Найдено объектов: %d.", len(descriptions)),
	)

	for i, description := range descriptions {
		title := strings.TrimSpace(description.Place.Title)
		if title == "" {
			title = "Без названия"
		}

		summary := strings.TrimSpace(description.Summary)

		if description.DescriptionError != "" {
			summary = description.DescriptionError
		} else if summary == "" {
			summary = "Краткое описание отсутствует."
		}

		titleLabel := widget.NewLabel(fmt.Sprintf("%d. %s", i+1, title))
		titleLabel.TextStyle = fyne.TextStyle{Bold: true}
		titleLabel.Wrapping = fyne.TextWrapWord

		distanceLabel := widget.NewLabel(fmt.Sprintf("Расстояние: %.0f м", description.Place.DistanceM))

		summaryLabel := widget.NewLabel(summary)
		summaryLabel.Wrapping = fyne.TextWrapWord

		// Текстовая часть карточки.
		textContent := container.NewVBox(titleLabel, distanceLabel, summaryLabel)

		articleURL, err := wikipediaArticleURL(a.wikipediaAPIURL, description.Place.PageID)

		if err == nil {
			textContent.Add(widget.NewHyperlink("Открыть статью в Википедии", articleURL))
		}

		var photoContent fyne.CanvasObject

		if description.Image != nil {
			photo := canvas.NewImageFromImage(description.Image)
			photo.FillMode = canvas.ImageFillContain
			photo.SetMinSize(fyne.NewSize(440, 300))

			// Центрирование сохраняет заданный размер области фото,
			// даже если текстовая часть карточки значительно выше.
			photoContent = container.NewCenter(photo)
		} else if description.DescriptionError == "" {
			message := "У статьи нет фотографии."

			if description.ImageError != "" {
				message = "Не удалось загрузить фотографию: " + description.ImageError
			}

			imageStatus := widget.NewLabel(message)
			imageStatus.Wrapping = fyne.TextWrapWord

			textContent.Add(imageStatus)
		}

		content := container.NewBorder(nil, // Сверху.
			nil,          // Снизу.
			nil,          // Слева.
			photoContent, // Справа — фотография.
			textContent,  // В центре — текст.
		)

		a.placesBox.Add(widget.NewCard("", "", content))
	}
}

// Для используемого нами endpoint /w/api.php
// строим адрес /w/index.php?curid=PAGE_ID.
func wikipediaArticleURL(apiURL string, pageID int64) (*url.URL, error) {
	if pageID <= 0 {
		return nil, fmt.Errorf("invalid Wikipedia page ID")
	}

	u, err := url.Parse(apiURL)
	if err != nil {
		return nil, fmt.Errorf("invalid Wikipedia API URL")
	}

	if (u.Scheme != "http" && u.Scheme != "https") ||
		u.Hostname() == "" {
		return nil, fmt.Errorf("invalid Wikipedia API endpoint")
	}

	if path.Base(u.Path) != "api.php" {
		return nil, fmt.Errorf("unsupported Wikipedia API path")
	}

	u.Path = path.Join(path.Dir(u.Path), "index.php")
	u.RawPath = ""
	u.Fragment = ""
	u.RawFragment = ""
	u.User = nil
	u.ForceQuery = false

	params := url.Values{}
	params.Set("curid", strconv.FormatInt(pageID, 10))
	u.RawQuery = params.Encode()

	return u, nil
}
