package qualityreport

import (
	"strings"
	"testing"

	"photo-cropper/internal/manifest"
)

func TestBuildMarkdown_includesAllModes(t *testing.T) {
	entries := []manifest.Entry{
		{Source: "a.jpg", Output: "a_001.jpg", Mode: "quad_hull", Confidence: 0.9},
		{Source: "a.jpg", Output: "a_002.jpg", Mode: "rotated_min_area_rect", Confidence: 0.65},
		{Source: "b.jpg", Output: "b_001.jpg", Mode: "axis_aligned", Confidence: 0.2},
	}
	s := BuildMarkdown(2, entries)
	if !strings.Contains(s, "quad_hull") || !strings.Contains(s, "rotated_min_area_rect") {
		t.Fatal("missing mode rows")
	}
	if !strings.Contains(s, "Average confidence") {
		t.Fatal("missing average")
	}
	// 0.2 triggers FAIL
	if !strings.Contains(s, "**FAIL**") {
		t.Fatal("expected FAIL in summary")
	}
}
