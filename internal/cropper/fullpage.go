package cropper

import (
	"image"
	"math"

	"photo-cropper/internal/seg"
)

const (
	fullPageBBoxAreaFracMax   = 0.85 // reject when bbox area / source area > this
	fullPageMinSideFrac       = 0.90 // reject when both bbox sides >= this fraction of source
	fullPageQuadSideFrac      = 0.90 // same for quad corner AABB
	fullPageQuadAreaFracMax   = 0.85
)

// isFullPageCandidate reports whether a segmented region is almost the entire scan (page border
// or full-sheet foreground), and must not be warped as a numbered crop.
func isFullPageCandidate(r seg.Region, srcW, srcH int) bool {
	if srcW < 1 || srcH < 1 {
		return false
	}
	rw := r.MaxX - r.MinX + 1
	rh := r.MaxY - r.MinY + 1
	if rw < 1 || rh < 1 {
		return false
	}
	bboxArea := float64(rw * rh)
	srcArea := float64(srcW * srcH)
	if bboxArea/srcArea > fullPageBBoxAreaFracMax {
		return true
	}
	if float64(rw) >= fullPageMinSideFrac*float64(srcW) && float64(rh) >= fullPageMinSideFrac*float64(srcH) {
		return true
	}
	return false
}

// quadCornersCoverFullSource reports when the quad’s axis-aligned bounds nearly match the whole source
// (corners effectively the full image frame), so the crop should be dropped even if the region bbox was smaller.
func quadCornersCoverFullSource(meta Meta, srcBounds image.Rectangle) bool {
	sw, sh := srcBounds.Dx(), srcBounds.Dy()
	if sw < 1 || sh < 1 {
		return false
	}
	var minx, miny = math.Inf(1), math.Inf(1)
	var maxx, maxy = math.Inf(-1), math.Inf(-1)
	for i := 0; i < 4; i++ {
		x, y := meta.Corners[i][0], meta.Corners[i][1]
		if x < minx {
			minx = x
		}
		if y < miny {
			miny = y
		}
		if x > maxx {
			maxx = x
		}
		if y > maxy {
			maxy = y
		}
	}
	cw := maxx - minx
	ch := maxy - miny
	if cw < 1 || ch < 1 {
		return false
	}
	if cw/float64(sw) >= fullPageQuadSideFrac && ch/float64(sh) >= fullPageQuadSideFrac {
		return true
	}
	if (cw*ch)/(float64(sw)*float64(sh)) > fullPageQuadAreaFracMax {
		return true
	}
	return false
}
