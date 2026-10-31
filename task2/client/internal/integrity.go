package internal

import (
	"crypto/sha256"
	"fmt"
	"io"
	"os"
)

// sourceChecksum читает файл через отдельный SectionReader,
// не изменяя текущую позицию чтения file.
func sourceChecksum(
	file *os.File,
	expectedSize int64,
) ([sha256.Size]byte, error) {
	var result [sha256.Size]byte

	before, err := file.Stat()
	if err != nil {
		return result, fmt.Errorf("stat source file: %w", err)
	}

	if before.Size() != expectedSize {
		return result, fmt.Errorf(
			"source file size changed: expected=%d, actual=%d",
			expectedSize,
			before.Size(),
		)
	}

	hash := sha256.New()
	reader := io.NewSectionReader(file, 0, expectedSize)

	n, err := io.Copy(hash, reader)
	if err != nil {
		return result, fmt.Errorf("hash source file: %w", err)
	}

	if n != expectedSize {
		return result, fmt.Errorf(
			"source file changed while hashing: read=%d, expected=%d",
			n,
			expectedSize,
		)
	}

	after, err := file.Stat()
	if err != nil {
		return result, fmt.Errorf("stat source after hashing: %w", err)
	}

	if after.Size() != before.Size() ||
		!after.ModTime().Equal(before.ModTime()) {
		return result, fmt.Errorf(
			"source file changed while computing checksum",
		)
	}

	copy(result[:], hash.Sum(nil))
	return result, nil
}
