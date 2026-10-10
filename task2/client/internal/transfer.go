package internal

import (
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

func SendFile(host string, port int, filePath string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("open file %q: %w", filePath, err)
	}
	defer func() {
		if err := file.Close(); err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "close source file: %v\n", err)
		}
	}()

	info, err := file.Stat()
	if err != nil {
		return fmt.Errorf("stat file: %w", err)
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("%q is not a regular file", filePath)
	}

	fileSize := info.Size()
	if fileSize < 0 || fileSize > maxFileBytes {
		return fmt.Errorf(
			"invalid file size: %d bytes, maximum=%d",
			fileSize,
			maxFileBytes,
		)
	}

	name := filepath.Base(filePath)
	if len(name) == 0 || len(name) > maxNameBytes {
		return fmt.Errorf("invalid filename length: %d bytes", len(name))
	}
	if !utf8.ValidString(name) ||
		name == "." ||
		name == ".." ||
		strings.ContainsAny(name, "/\\\x00") {
		return fmt.Errorf("invalid filename: %q", name)
	}

	addr := net.JoinHostPort(host, strconv.Itoa(port))
	conn, err := net.DialTimeout("tcp", addr, connectTimeout)
	if err != nil {
		return fmt.Errorf("connect to %s: %w", addr, err)
	}
	defer func() {
		if err := conn.Close(); err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "close connection: %v\n", err)
		}
	}()

	log.Printf("connected to %s\n", conn.RemoteAddr())

	header := make([]byte, 10+len(name))
	binary.BigEndian.PutUint16(header[0:2], uint16(len(name)))
	binary.BigEndian.PutUint64(header[2:10], uint64(fileSize))
	copy(header[10:], name)

	writer := &deadlineWriter{
		conn:    conn,
		timeout: writeTimeout,
	}

	if err := writeAll(writer, header); err != nil {
		return fmt.Errorf("send header: %w", err)
	}

	log.Printf("sending %q, size=%d bytes\n", name, fileSize)

	sent, err := io.CopyN(writer, file, fileSize)
	if err != nil {
		return fmt.Errorf(
			"send file: sent %d of %d bytes: %w",
			sent,
			fileSize,
			err,
		)
	}

	fmt.Printf("sent %d bytes; waiting for server confirmation\n", sent)

	if err := conn.SetReadDeadline(time.Now().Add(resultTimeout)); err != nil {
		return fmt.Errorf("set result deadline: %w", err)
	}

	var response [1]byte
	if _, err := io.ReadFull(conn, response[:]); err != nil {
		return fmt.Errorf(
			"no server confirmation; file may or may not have been saved: %w",
			err,
		)
	}

	switch response[0] {
	case statusOK:
		log.Printf(
			"Передача успешна: сервер подтвердил сохранение %q (%d байт).\n",
			name,
			sent,
		)
		return nil
	case statusError:
		return fmt.Errorf("server reported a receive or save error for %q", name)
	default:
		return fmt.Errorf("unknown server status: %d", response[0])
	}
}
