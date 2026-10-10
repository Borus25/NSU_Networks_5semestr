package internal

import (
	"fmt"
	"log"
	"sync"
	"sync/atomic"
	"time"
)

const (
	kilobyte = 1024
	megabyte = 1024 * kilobyte
	gigabyte = 1024 * megabyte
)

// formatSpeed автоматически выбирает подходящую единицу измерения.
func formatSpeed(bytesPerSecond float64) string {
	switch {
	case bytesPerSecond >= gigabyte:
		return fmt.Sprintf("%.4f ГБ/с", bytesPerSecond/gigabyte)
	case bytesPerSecond >= megabyte:
		return fmt.Sprintf("%.4f МБ/с", bytesPerSecond/megabyte)
	case bytesPerSecond >= kilobyte:
		return fmt.Sprintf("%.4f КБ/с", bytesPerSecond/kilobyte)
	default:
		return fmt.Sprintf("%.2f Б/с", bytesPerSecond)
	}
}

// startTransferStats запускает вывод текущей и средней скорости каждые 3 секунды.
// Возвращённую функцию необходимо вызвать ровно один раз.
func startTransferStats(client string, received *atomic.Uint64) func() {
	started := time.Now()
	stop := make(chan time.Time)
	finished := make(chan struct{})

	var stopOnce sync.Once

	go func() {
		defer close(finished)

		ticker := time.NewTicker(3 * time.Second)
		defer ticker.Stop()

		lastTime := started
		var lastBytes uint64

		report := func(now time.Time, final bool) {
			currentBytes := received.Load()

			var currentSpeed float64
			var averageSpeed float64

			interval := now.Sub(lastTime).Seconds()
			if interval > 0 {
				currentSpeed = float64(currentBytes-lastBytes) / interval
			}

			totalTime := now.Sub(started).Seconds()
			if totalTime > 0 {
				averageSpeed = float64(currentBytes) / totalTime
			}

			label := "progress"
			if final {
				label = "final"
			}

			log.Printf(
				"client %s: %s, received=%d bytes, current=%s, average=%s",
				client,
				label,
				currentBytes,
				formatSpeed(currentSpeed),
				formatSpeed(averageSpeed),
			)

			lastTime = now
			lastBytes = currentBytes
		}

		for {
			select {
			case now := <-ticker.C:
				report(now, false)
			case ended := <-stop:
				report(ended, true)
				return
			}
		}
	}()

	return func() {
		stopOnce.Do(func() {
			stop <- time.Now()
			<-finished
		})
	}
}
