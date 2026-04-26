// Package manifest describes JSON output for each extracted photo.
package manifest

// Size is the output image dimensions in pixels.
type Size struct {
	Width  int `json:"width"`
	Height int `json:"height"`
}

// File is the top-level manifest written to output folder.
type File struct {
	Version int     `json:"version"`
	Entries []Entry `json:"entries"`
}

// Entry is one source → output record with detection metadata.
// QaImage is the QA annotated scan (<stem>_000_qa.jpg); it is the same for every entry from a given source.
type Entry struct {
	Source     string      `json:"source"`
	Output     string      `json:"output"`
	QaImage    string      `json:"qa_image,omitempty"`
	Corners    [][]float64 `json:"corners"`
	OutputSize Size        `json:"output_size"`
	Mode       string      `json:"mode"`
	Confidence float64     `json:"confidence"`
}
