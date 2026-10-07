package internal

import (
	"time"
)

const (
	maxNameBytes = 4096
	// Здесь 1 ТБ трактуется как 10^12 байт.
	maxFileBytes uint64 = 1_000_000_000_000

	headerTimeout = 30 * time.Second
	readTimeout   = 2 * time.Minute
	replyTimeout  = 5 * time.Second

	statusOK    byte = 0
	statusError byte = 1
)
