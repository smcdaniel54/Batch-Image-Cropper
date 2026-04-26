// Package moveprocessed moves successfully extracted scan files into an input tree's "processed" folder.
package moveprocessed

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ProcessedDir returns <baseDir>/processed
func ProcessedDir(baseDir string) string {
	return filepath.Join(baseDir, "processed")
}

// NextDestPath returns the first path under processedDir with filename base that does not
// already exist: base, then stem_2+ext, stem_3+ext, ...
func NextDestPath(processedDir, base string) (string, error) {
	stem, ext, ok := splitName(base)
	if !ok {
		return "", fmt.Errorf("invalid file name: %q", base)
	}
	// 1-based suffix in the filename: _2, _3, ...
	for n := 0; n < 10000; n++ {
		var name string
		if n == 0 {
			name = base
		} else {
			name = fmt.Sprintf("%s_%d%s", stem, n+1, ext)
		}
		p := filepath.Join(processedDir, name)
		if _, err := os.Stat(p); err != nil {
			if os.IsNotExist(err) {
				return p, nil
			}
			return "", err
		}
	}
	return "", fmt.Errorf("could not find free name in %q", processedDir)
}

func splitName(base string) (stem, ext string, ok bool) {
	ext = filepath.Ext(base)
	if ext == "" {
		return base, "", true
	}
	stem = strings.TrimSuffix(base, ext)
	if stem == "" {
		return "", ext, true
	}
	return stem, ext, true
}

// MoveToProcessed creates processed/ under inputBaseDir, picks a free destination name, and
// renames sourcePath there. inputBaseDir is the input directory in batch mode, or the
// parent directory of the file in single -input mode.
func MoveToProcessed(sourcePath, inputBaseDir string) (dest string, err error) {
	processed := ProcessedDir(inputBaseDir)
	if err := os.MkdirAll(processed, 0o755); err != nil {
		return "", err
	}
	base := filepath.Base(sourcePath)
	dest, err = NextDestPath(processed, base)
	if err != nil {
		return "", err
	}
	if err := os.Rename(sourcePath, dest); err != nil {
		return "", err
	}
	return dest, nil
}

// MoveAfterExtraction reports whether the original scan should be moved to processed/
// (at least one photo was extracted from it).
func MoveAfterExtraction(photosExtracted int) bool {
	return photosExtracted > 0
}
