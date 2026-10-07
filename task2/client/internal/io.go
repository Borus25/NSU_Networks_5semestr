package internal

import (
	"io"
)

// Учитывает возможность частичной записи заголовка.
func writeAll(w io.Writer, data []byte) error {
	for len(data) > 0 {
		n, err := w.Write(data)

		if n > 0 {
			data = data[n:]
		}

		if err != nil {
			return err
		}
		if n == 0 {
			return io.ErrNoProgress
		}
	}

	return nil
}
