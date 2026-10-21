package internal

import (
	"crypto/rand"
	"errors"
	"fmt"
	"os"
	"syscall"
)

// createUpload сначала пытается сохранить исходное имя.
// Случайное имя используется, если исходное уже занято
// или превышает ограничение файловой системы.
//
// Предполагается, что имя уже проверено обработчиком.
func createUpload(root *os.Root, name string) (*os.File, string, error) {
	const flags = os.O_WRONLY | os.O_CREATE | os.O_EXCL

	file, err := root.OpenFile(name, flags, 0600)
	if err == nil {
		return file, name, nil
	}

	// Остальные ошибки не скрываем: например, отсутствие
	// прав доступа или свободного места на диске.
	if !errors.Is(err, os.ErrExist) &&
		!errors.Is(err, syscall.ENAMETOOLONG) {
		return nil, "", fmt.Errorf(
			"create file with original name: %w",
			err,
		)
	}

	for range 10 {
		var id [8]byte
		if _, err := rand.Read(id[:]); err != nil {
			return nil, "", fmt.Errorf(
				"generate upload filename: %w",
				err,
			)
		}

		savedName := fmt.Sprintf("upload-%x.bin", id)

		file, err = root.OpenFile(savedName, flags, 0600)
		if err == nil {
			return file, savedName, nil
		}

		if !errors.Is(err, os.ErrExist) {
			return nil, "", fmt.Errorf(
				"create file with generated name: %w",
				err,
			)
		}
	}

	return nil, "", fmt.Errorf(
		"cannot choose an unused filename after 10 attempts",
	)
}
