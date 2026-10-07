package internal

import (
	"net"
	"time"
)

// Обновляет дедлайн перед каждой записью.
// Таким образом, нет общего ограничения в 2 минуты на весь файл.
type deadlineWriter struct {
	conn    net.Conn
	timeout time.Duration
}

func (w *deadlineWriter) Write(p []byte) (int, error) {
	if err := w.conn.SetWriteDeadline(
		time.Now().Add(w.timeout),
	); err != nil {
		return 0, err
	}

	return w.conn.Write(p)
}
