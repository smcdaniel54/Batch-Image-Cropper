package fsutil

import (
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// IsImageName returns true for .jpg .jpeg .png (case-insensitive).
func IsImageName(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	return ext == ".jpg" || ext == ".jpeg" || ext == ".png"
}

// ListImageFiles returns sorted absolute paths of supported images in the top-level of dir only
// (not recursive). Unsupported files in that directory are ignored.
func ListImageFiles(dir string) ([]string, error) {
	return ListImageFilesWithWarnings(dir, nil)
}

// ListImageFilesWithWarnings is like [ListImageFiles] but for each top-level file with an
// unsupported extension it calls warnf once with a full message. Subdirectories are not
// entered. If warnf is nil, unsupported files are still skipped. Paths are returned sorted, absolute.
func ListImageFilesWithWarnings(dir string, warnf func(string)) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var list []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		p := filepath.Join(dir, name)
		if !IsImageName(name) {
			if warnf != nil {
				abs, aerr := filepath.Abs(p)
				if aerr != nil {
					return nil, aerr
				}
				warnf("warning: unsupported file (expected .jpg, .jpeg, or .png), skipping: " + abs)
			}
			continue
		}
		abs, aerr := filepath.Abs(p)
		if aerr != nil {
			return nil, aerr
		}
		list = append(list, abs)
	}
	sort.Strings(list)
	return list, nil
}

// WarnWriter adapts an [io.Writer] to the warnf callback used by [ListImageFilesWithWarnings] (e.g. [os.Stderr]).
func WarnWriter(w io.Writer) func(string) {
	return func(msg string) {
		_, _ = io.WriteString(w, msg+"\n")
	}
}
