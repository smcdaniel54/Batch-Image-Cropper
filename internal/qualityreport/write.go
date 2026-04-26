package qualityreport

import (
	"os"

	"batch-image-cropper/internal/manifest"
)

// WriteFile writes `quality_report.md` text built from the same inputs as the manifest.
func WriteFile(path string, sourceImages int, entries []manifest.Entry) error {
	return os.WriteFile(path, []byte(BuildMarkdown(sourceImages, entries)), 0o644)
}
