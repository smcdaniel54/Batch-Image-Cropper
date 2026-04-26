package qualityreport

import (
	"fmt"
	"math"
	"sort"
	"strings"

	"photo-cropper/internal/manifest"
)

// KnownModeOrder is the stable order for table rows; unknown modes in data are appended sorted.
var KnownModeOrder = []string{
	"quad_hull",
	"rotated_min_area_rect",
	"axis_aligned",
	"axis_aligned_invalid_quad",
	"axis_aligned_homography_fail",
}

// BuildMarkdown assembles the quality report document from manifest entries and source file count.
func BuildMarkdown(sourceImages int, entries []manifest.Entry) string {
	var b strings.Builder
	b.WriteString("# Quality report\n\n")
	b.WriteString("Generated from the same run as `manifest.json`.\n\n")

	byMode := make(map[string]int)
	for _, e := range entries {
		byMode[e.Mode]++
	}
	confidences := make([]float64, 0, len(entries))
	for _, e := range entries {
		confidences = append(confidences, e.Confidence)
	}
	avg, avgOK := meanConfidence(confidences)
	status := ClassifyStatus(confidences)

	b.WriteString("## Summary\n\n")
	b.WriteString("| Metric | Value |\n|--------|--------|\n")
	b.WriteString(fmt.Sprintf("| Source images processed | %d |\n", sourceImages))
	b.WriteString(fmt.Sprintf("| Photos extracted | %d |\n", len(entries)))
	if avgOK {
		b.WriteString(fmt.Sprintf("| Average confidence | %.3f |\n", avg))
	} else {
		b.WriteString("| Average confidence | — |\n")
	}
	b.WriteString(fmt.Sprintf("| **Status** | **%s** |\n\n", status))
	b.WriteString(
		"Status: **PASS** if all confidences are ≥ 0.65; **REVIEW** if any are below 0.65 but above 0.2; " +
			"**FAIL** if any confidence is ≤ 0.2.\n\n",
	)

	b.WriteString("## Counts by mode\n\n")
	b.WriteString("| Mode | Count |\n|------|--------|\n")
	seen := make(map[string]bool)
	for _, m := range KnownModeOrder {
		c := byMode[m]
		b.WriteString(fmt.Sprintf("| %s | %d |\n", m, c))
		seen[m] = true
	}
	var rest []string
	for m := range byMode {
		if !seen[m] {
			rest = append(rest, m)
		}
	}
	sort.Strings(rest)
	for _, m := range rest {
		b.WriteString(fmt.Sprintf("| %s | %d |\n", m, byMode[m]))
	}
	b.WriteString("\n")

	low := make([]manifest.Entry, 0)
	for _, e := range entries {
		if e.Confidence < 0.5 {
			low = append(low, e)
		}
	}
	sort.Slice(low, func(i, j int) bool { return low[i].Output < low[j].Output })

	b.WriteString("## Low confidence (< 0.5)\n\n")
	if len(low) == 0 {
		b.WriteString("None.\n\n")
	} else {
		b.WriteString("| Output | Mode | Confidence |\n|--------|------|------------|\n")
		for _, e := range low {
			b.WriteString(fmt.Sprintf("| %s | %s | %.3f |\n", e.Output, e.Mode, e.Confidence))
		}
		b.WriteString("\n")
	}

	return b.String()
}

func meanConfidence(xs []float64) (float64, bool) {
	if len(xs) == 0 {
		return 0, false
	}
	s := 0.0
	for _, x := range xs {
		s += x
	}
	avg := s / float64(len(xs))
	// avoid -0.000
	if math.Abs(avg) < 1e-10 {
		return 0, true
	}
	return avg, true
}
