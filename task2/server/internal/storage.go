package internal

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"unicode/utf8"
)

// createUpload preserves the original name whenever possible.
// Existing files are never overwritten, including concurrent uploads.
func createUpload(root *os.Root, name string) (*os.File, string, error) {
	const flags = os.O_WRONLY | os.O_CREATE | os.O_EXCL
	file, err := root.OpenFile(name, flags, 0600)
	if err == nil {
		return file, name, nil
	}
	if !errors.Is(err, os.ErrExist) && !errors.Is(err, syscall.ENAMETOOLONG) {
		return nil, "", fmt.Errorf("create original filename: %w", err)
	}

	ext := filepath.Ext(name)
	// Treat .gitignore and similar names as extensionless.
	if ext == name {
		ext = ""
	}
	stem := strings.TrimSuffix(name, ext)

	for copyNumber := 1; copyNumber <= 10000; copyNumber++ {
		suffix := " (copy)"
		if copyNumber > 1 {
			suffix = fmt.Sprintf(" (%d-copy)", copyNumber)
		}
		candidateStem, candidateExt := stem, ext
		for {
			savedName := candidateStem + suffix + candidateExt
			file, err := root.OpenFile(savedName, flags, 0600)
			if err == nil {
				return file, savedName, nil
			}
			if errors.Is(err, os.ErrExist) {
				break
			}
			if !errors.Is(err, syscall.ENAMETOOLONG) {
				return nil, "", fmt.Errorf("create copy filename: %w", err)
			}
			// Shorten only when the filesystem actually rejects the name.
			// Remove complete UTF-8 characters, preserving the extension
			// as long as possible.
			if candidateStem != "" {
				_, size := utf8.DecodeLastRuneInString(candidateStem)
				candidateStem = candidateStem[:len(candidateStem)-size]
			} else if candidateExt != "" {
				_, size := utf8.DecodeLastRuneInString(candidateExt)
				candidateExt = candidateExt[:len(candidateExt)-size]
			} else {
				return nil, "", fmt.Errorf("copy filename is too long: %w", err)
			}
		}
	}
	return nil, "", fmt.Errorf("cannot choose an unused filename after 10000 copy attempts")
}
