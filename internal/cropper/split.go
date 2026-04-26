package cropper

import (
	"image"
	"math"

	"photo-cropper/internal/seg"
)

// Heuristics for splitting one connected component when a thin background gap
// suggests multiple photos inside the region bbox.
const (
	sepBgFrac        = 0.9
	sepBandMinPx     = 3
	sepBandMaxPx     = 10
	bboxEdgeFrac     = 0.15
	sepMinInnerSpan  = 5 // require inner [min,max] to span at least this many columns/rows for a valid separator search
)

// trySplitRegionClips returns two disjoint clip rectangles (image coords 0..w, 0..h, Max exclusive)
// when a vertical or horizontal background separator is found inside the region bbox and both
// sides retain at least minArea foreground pixels with labels[r.ID]. Otherwise returns nil.
func trySplitRegionClips(w, h int, mask []byte, labels []int, r seg.Region, minArea int) []image.Rectangle {
	rw := r.MaxX - r.MinX + 1
	rh := r.MaxY - r.MinY + 1
	if rw < sepMinInnerSpan*3 || rh < sepMinInnerSpan*3 {
		return nil
	}
	if v, ok := findVerticalSeparatorBand(mask, w, h, r); ok {
		left := image.Rect(r.MinX, r.MinY, v.runMinX, r.MaxY+1)
		right := image.Rect(v.runMaxX+1, r.MinY, r.MaxX+1, r.MaxY+1)
		if left.Empty() || right.Empty() {
			return nil
		}
		if countLabelInRect(w, h, labels, r.ID, left) >= minArea &&
			countLabelInRect(w, h, labels, r.ID, right) >= minArea {
			return []image.Rectangle{left, right}
		}
	}
	if hz, ok := findHorizontalSeparatorBand(mask, w, h, r); ok {
		top := image.Rect(r.MinX, r.MinY, r.MaxX+1, hz.runMinY)
		bot := image.Rect(r.MinX, hz.runMaxY+1, r.MaxX+1, r.MaxY+1)
		if top.Empty() || bot.Empty() {
			return nil
		}
		if countLabelInRect(w, h, labels, r.ID, top) >= minArea &&
			countLabelInRect(w, h, labels, r.ID, bot) >= minArea {
			return []image.Rectangle{top, bot}
		}
	}
	return nil
}

type sepBand1D struct {
	runMinX, runMaxX int // inclusive column range (vertical band)
	runMinY, runMaxY int // inclusive row range (horizontal band)
}

func findVerticalSeparatorBand(mask []byte, w, h int, r seg.Region) (sepBand1D, bool) {
	var zero sepBand1D
	rw := r.MaxX - r.MinX + 1
	rh := r.MaxY - r.MinY + 1
	margin := int(math.Floor(bboxEdgeFrac * float64(rw)))
	if margin < 1 {
		margin = 1
	}
	innerMinX := r.MinX + margin
	innerMaxX := r.MaxX - margin
	if innerMinX > innerMaxX {
		return zero, false
	}
	colSep := make([]bool, rw)
	for i := 0; i < rw; i++ {
		x := r.MinX + i
		bg := 0
		for y := r.MinY; y <= r.MaxY; y++ {
			if mask[y*w+x] == 0 {
				bg++
			}
		}
		colSep[i] = float64(bg)/float64(rh) >= sepBgFrac
	}
	return pickBestVerticalRun(colSep, r.MinX, innerMinX, innerMaxX)
}

func pickBestVerticalRun(colSep []bool, originX, innerMinX, innerMaxX int) (sepBand1D, bool) {
	var best sepBand1D
	bestW := -1
	bestDist := 1e9
	center := float64(originX) + float64(len(colSep)-1)/2
	i := 0
	for i < len(colSep) {
		if !colSep[i] {
			i++
			continue
		}
		j := i
		for j < len(colSep) && colSep[j] {
			j++
		}
		runMinX := originX + i
		runMaxX := originX + j - 1
		width := j - i
		if width < sepBandMinPx || width > sepBandMaxPx {
			i = j
			continue
		}
		if runMinX < innerMinX || runMaxX > innerMaxX {
			i = j
			continue
		}
		mid := float64(runMinX+runMaxX) / 2
		dist := math.Abs(mid - center)
		if width > bestW || (width == bestW && dist < bestDist) {
			bestW = width
			bestDist = dist
			best.runMinX = runMinX
			best.runMaxX = runMaxX
		}
		i = j
	}
	if bestW < sepBandMinPx {
		return sepBand1D{}, false
	}
	return best, true
}

func findHorizontalSeparatorBand(mask []byte, w, h int, r seg.Region) (sepBand1D, bool) {
	var zero sepBand1D
	rw := r.MaxX - r.MinX + 1
	rh := r.MaxY - r.MinY + 1
	margin := int(math.Floor(bboxEdgeFrac * float64(rh)))
	if margin < 1 {
		margin = 1
	}
	innerMinY := r.MinY + margin
	innerMaxY := r.MaxY - margin
	if innerMinY > innerMaxY {
		return zero, false
	}
	rowSep := make([]bool, rh)
	for j := 0; j < rh; j++ {
		y := r.MinY + j
		bg := 0
		for x := r.MinX; x <= r.MaxX; x++ {
			if mask[y*w+x] == 0 {
				bg++
			}
		}
		rowSep[j] = float64(bg)/float64(rw) >= sepBgFrac
	}
	return pickBestHorizontalRun(rowSep, r.MinY, innerMinY, innerMaxY)
}

func pickBestHorizontalRun(rowSep []bool, originY, innerMinY, innerMaxY int) (sepBand1D, bool) {
	var best sepBand1D
	bestH := -1
	bestDist := 1e9
	center := float64(originY) + float64(len(rowSep)-1)/2
	i := 0
	for i < len(rowSep) {
		if !rowSep[i] {
			i++
			continue
		}
		j := i
		for j < len(rowSep) && rowSep[j] {
			j++
		}
		runMinY := originY + i
		runMaxY := originY + j - 1
		height := j - i
		if height < sepBandMinPx || height > sepBandMaxPx {
			i = j
			continue
		}
		if runMinY < innerMinY || runMaxY > innerMaxY {
			i = j
			continue
		}
		mid := float64(runMinY+runMaxY) / 2
		dist := math.Abs(mid - center)
		if height > bestH || (height == bestH && dist < bestDist) {
			bestH = height
			bestDist = dist
			best.runMinY = runMinY
			best.runMaxY = runMaxY
		}
		i = j
	}
	if bestH < sepBandMinPx {
		return sepBand1D{}, false
	}
	return best, true
}

func countLabelInRect(w, h int, labels []int, id int, rect image.Rectangle) int {
	rect = rect.Intersect(image.Rect(0, 0, w, h))
	if rect.Empty() {
		return 0
	}
	n := 0
	for y := rect.Min.Y; y < rect.Max.Y; y++ {
		row := y * w
		for x := rect.Min.X; x < rect.Max.X; x++ {
			if labels[row+x] == id {
				n++
			}
		}
	}
	return n
}
