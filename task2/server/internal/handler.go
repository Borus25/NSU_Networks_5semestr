package internal

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"strings"
	"sync/atomic"
	"time"
	"unicode/utf8"
)

func handleConn(conn net.Conn) (retErr error) {
	client := conn.RemoteAddr().String()

	// Ответ отправляется после завершения остальных deferred-функций.
	defer func() {
		status := statusOK
		if retErr != nil {
			status = statusError
		}

		err := conn.SetWriteDeadline(time.Now().Add(replyTimeout))
		if err == nil {
			var n int
			n, err = conn.Write([]byte{status})
			if err == nil && n != 1 {
				err = io.ErrShortWrite
			}
		}

		if err != nil {
			if retErr == nil {
				retErr = fmt.Errorf(
					"file saved, but sending result failed: %w", err,
				)
			} else {
				fmt.Printf("client %s: send error status: %v", client, err)
			}
		}
	}()

	if err := conn.SetReadDeadline(
		time.Now().Add(headerTimeout),
	); err != nil {
		return fmt.Errorf("set header deadline: %w", err)
	}

	var nameLen uint16
	var expectedSize uint64

	if err := binary.Read(conn, binary.BigEndian, &nameLen); err != nil {
		return fmt.Errorf("read filename length: %w", err)
	}

	if err := binary.Read(conn, binary.BigEndian, &expectedSize); err != nil {
		return fmt.Errorf("read file size: %w", err)
	}

	if nameLen == 0 || nameLen > maxNameBytes {
		return fmt.Errorf("invalid filename length: %d", nameLen)
	}
	if expectedSize > maxFileBytes {
		return fmt.Errorf("file too large: %d bytes", expectedSize)
	}

	nameBytes := make([]byte, int(nameLen))
	if _, err := io.ReadFull(conn, nameBytes); err != nil {
		return fmt.Errorf("read filename: %w", err)
	}

	name := string(nameBytes)
	if !utf8.ValidString(name) || name == "." || name == ".." || strings.ContainsAny(name, "/\\\x00") {
		return fmt.Errorf("invalid filename: %q", name)
	}

	root, err := os.OpenRoot("uploads")
	if err != nil {
		return fmt.Errorf("open uploads directory: %w", err)
	}
	defer func() {
		if err := root.Close(); err != nil {
			fmt.Printf("client %s: close uploads directory: %v", client, err)
		}
	}()

	file, savedName, err := createUpload(root, name)
	if err != nil {
		return fmt.Errorf("create upload file: %w", err)
	}

	closed := false
	saved := false

	defer func() {
		if !closed {
			if err := file.Close(); err != nil {
				fmt.Printf("client %s: close incomplete file: %v", client, err)
			}
		}

		if !saved {
			if err := root.Remove(savedName); err != nil {
				fmt.Printf(
					"client %s: remove incomplete file %q: %v",
					client, savedName, err,
				)
			}
		}
	}()

	fmt.Printf(
		"client %s: receiving %q as %q, expected=%d bytes",
		client, name, savedName, expectedSize,
	)

	var received atomic.Uint64
	stopStats := startTransferStats(client, &received)
	defer stopStats()

	buf := make([]byte, 64*1024)
	var written uint64

	for written < expectedSize {
		remaining := expectedSize - written
		chunk := len(buf)
		if remaining < uint64(chunk) {
			chunk = int(remaining)
		}

		if err := conn.SetReadDeadline(
			time.Now().Add(readTimeout),
		); err != nil {
			return fmt.Errorf("set data deadline: %w", err)
		}

		n, readErr := conn.Read(buf[:chunk])
		received.Add(uint64(n))

		if n > 0 {
			nw, writeErr := file.Write(buf[:n])
			written += uint64(nw)

			if writeErr != nil {
				return fmt.Errorf("write file: %w", writeErr)
			}
			if nw != n {
				return fmt.Errorf("write file: %w", io.ErrShortWrite)
			}
		}

		if readErr != nil {
			// EOF после получения последнего байта допустим.
			if errors.Is(readErr, io.EOF) && written == expectedSize {
				break
			}

			return fmt.Errorf(
				"receive file: got %d of %d bytes: %w",
				received.Load(), expectedSize, readErr,
			)
		}

		if n == 0 {
			return fmt.Errorf("receive file: %w", io.ErrNoProgress)
		}
	}

	info, err := file.Stat()
	if err != nil {
		return fmt.Errorf("stat received file: %w", err)
	}

	if written != expectedSize || uint64(info.Size()) != expectedSize {
		return fmt.Errorf("size mismatch: expected=%d, written=%d, actual=%d", expectedSize, written, info.Size())
	}

	if err := file.Sync(); err != nil {
		return fmt.Errorf("sync file: %w", err)
	}

	err = file.Close()
	closed = true
	if err != nil {
		return fmt.Errorf("close file: %w", err)
	}

	saved = true

	fmt.Printf(
		"client %s: saved %q, size=%d bytes",
		client, savedName, written,
	)

	return nil
}
