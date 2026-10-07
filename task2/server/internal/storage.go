package internal

import (
	"crypto/rand"
	"errors"
	"fmt"
	"os"
)

// Не перезаписываем существующие файлы.
// Для длинного или занятого имени выбираем короткое случайное.
func createUpload(root *os.Root, name string) (*os.File, string, error) {
	flags := os.O_WRONLY | os.O_CREATE | os.O_EXCL

	// 200 — консервативный порог для попытки сохранить исходное имя.
	if len(name) <= 200 {
		file, err := root.OpenFile(name, flags, 0600)
		if err == nil {
			return file, name, nil
		}
		if !errors.Is(err, os.ErrExist) {
			return nil, "", err
		}
	}

	for range 10 {
		var id [8]byte
		if _, err := rand.Read(id[:]); err != nil {
			return nil, "", err
		}

		savedName := fmt.Sprintf("upload-%x.bin", id)

		file, err := root.OpenFile(savedName, flags, 0600)
		if err == nil {
			return file, savedName, nil
		}
		if !errors.Is(err, os.ErrExist) {
			return nil, "", err
		}
	}

	return nil, "", fmt.Errorf("cannot choose an unused filename")
}
