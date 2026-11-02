package ui

// updateBusy вызывается только в UI-контексте.
func (a *App) updateBusy() {
	busy := a.searchCancel != nil || a.detailsCancel != nil

	if busy {
		a.cancelBtn.Enable()
		a.progress.Show()
		a.progress.Start()
		return
	}

	a.cancelBtn.Disable()
	a.progress.Stop()
	a.progress.Hide()
}

// onCancel вызывается обработчиком кнопки в UI-контексте.
func (a *App) onCancel() {
	searchWasRunning := a.searchCancel != nil
	detailsWereRunning := a.detailsCancel != nil

	if !searchWasRunning && !detailsWereRunning {
		return
	}

	// Сначала делаем результаты отменённых операций неактуальными.
	// Уже поставленные в UI-очередь обновления будут проигнорированы.
	a.searchID++
	a.detailsID++

	if a.searchCancel != nil {
		a.searchCancel()
		a.searchCancel = nil
	}

	if a.detailsCancel != nil {
		a.detailsCancel()
		a.detailsCancel = nil
	}

	if searchWasRunning {
		a.foundLocations = nil

		a.locationsBox.Objects = nil
		a.locationsBox.Refresh()
	}

	if detailsWereRunning {
		a.weather.SetText("Загрузка погоды отменена.")
		a.setPlacesMessage("Загрузка мест и описаний отменена.")
	}

	a.status.SetText("Операция отменена.")
	a.updateBusy()
}
