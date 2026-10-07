package internal

import (
	"fmt"
	"sync/atomic"
	"time"
)

// Возвращённую функцию нужно вызвать ровно один раз.
func startTransferStats(client string, received *atomic.Uint64) func() {
	started := time.Now()
	stop := make(chan time.Time)
	finished := make(chan struct{})

	go func() {
		defer close(finished)

		ticker := time.NewTicker(3 * time.Second)
		defer ticker.Stop()

		lastTime := started
		var lastBytes uint64

		report := func(now time.Time, final bool) {
			current := received.Load()

			var instant, average float64

			if seconds := now.Sub(lastTime).Seconds(); seconds > 0 {
				instant = float64(current-lastBytes) / seconds
			}
			if seconds := now.Sub(started).Seconds(); seconds > 0 {
				average = float64(current) / seconds
			}

			label := "progress"
			if final {
				label = "final"
			}

			fmt.Printf(
				"client %s: %s, received=%d bytes, "+
					"instant=%.2f B/s, average=%.2f B/s",
				client, label, current, instant, average,
			)

			lastTime = now
			lastBytes = current
		}

		for {
			select {
			case <-ticker.C:
				report(time.Now(), false)

			case ended := <-stop:
				report(ended, true)
				return
			}
		}
	}()

	return func() {
		stop <- time.Now()
		<-finished
	}
}
