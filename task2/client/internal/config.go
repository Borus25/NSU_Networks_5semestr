package internal

import (
	"time"
)

const (
	maxNameBytes       = 4096
	maxFileBytes int64 = 1_000_000_000_000

	connectTimeout = 5 * time.Second
	writeTimeout   = 2 * time.Minute
	resultTimeout  = 2 * time.Minute

	statusOK    byte = 0
	statusError byte = 1
)
