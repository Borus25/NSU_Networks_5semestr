package console

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)

	err := run(ctx)

	// Освобождаем ресурсы подписки на сигналы до возможного os.Exit.
	stop()

	if err != nil {
		log.Printf("application error: %v", err)
		os.Exit(1)
	}
}
