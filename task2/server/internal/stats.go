package internal

import (
	"fmt"
	"log"
	"sync"
	"sync/atomic"
	"time"
)

func formatSpeed(bytesPerSecond float64) string {
	units := [...]string{"Б/с", "КБ/с", "МБ/с", "ГБ/с"}
	i := 0
	for bytesPerSecond >= 1024 && i < len(units)-1 {
		bytesPerSecond /= 1024
		i++
	}
	return fmt.Sprintf("%.2f %s", bytesPerSecond, units[i])
}

func startTransferStats(client string, received *atomic.Uint64) func() {
	started := time.Now()
	baseline := received.Load()
	stop := make(chan struct{})
	finished := make(chan struct{})
	var once sync.Once

	go func() {
		defer close(finished)
		ticker := time.NewTicker(IntervalReplyTimeout)
		defer ticker.Stop()
		lastTime := started
		lastBytes := baseline

		report := func(final bool) {
			current := received.Load()
			now := time.Now()
			var instant, average float64
			if seconds := now.Sub(lastTime).Seconds(); seconds > 0 {
				instant = float64(current-lastBytes) / seconds
			}
			if seconds := now.Sub(started).Seconds(); seconds > 0 {
				average = float64(current-baseline) / seconds
			}
			label := "progress"
			if final {
				label = "final"
			}
			log.Printf("client %s: %s, received=%s, instant=%s, average=%s",
				client, label, formatSize(current-baseline), formatSpeed(instant), formatSpeed(average))
			lastTime = now
			lastBytes = current
		}

		for {
			select {
			case <-stop:
				report(true)
				return
			case <-ticker.C:
				select {
				case <-stop:
					report(true)
					return
				default:
					report(false)
				}
			}
		}
	}()

	return func() {
		once.Do(func() { close(stop) })
		<-finished
	}
}
