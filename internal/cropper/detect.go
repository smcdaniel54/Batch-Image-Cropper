package cropper

import (
	"batch-image-cropper/internal/contour"
	"batch-image-cropper/internal/geom"
	"batch-image-cropper/internal/seg"
	"batch-image-cropper/internal/warp"
	"image"
	"image/color"
	"image/draw"
)

func gray8(c color.RGBA) uint8 {
	return uint8(0.299*float64(c.R) + 0.587*float64(c.G) + 0.114*float64(c.B) + 0.5)
}

func toRGBA(img image.Image) *image.RGBA {
	if r, ok := img.(*image.RGBA); ok {
		return r
	}
	b := img.Bounds()
	rgba := image.NewRGBA(b)
	draw.Draw(rgba, b, img, b.Min, draw.Src)
	return rgba
}

// extractRegion warps a connected component. labels use local x,y 0..w-1, 0..h-1; output coords are in src.Bounds() space.
func extractRegion(
	src *image.RGBA,
	w, h int,
	labels []int,
	region seg.Region,
	pad int,
	stride int,
) (out *image.RGBA, meta Meta) {
	return extractRegionClip(src, w, h, labels, region, nil, pad, stride)
}

// extractRegionClip is like extractRegion but optionally restricts contour sampling to clip (image.Rect in 0..w, 0..h, Max exclusive).
// When clip is nil, behavior matches extractRegion.
func extractRegionClip(
	src *image.RGBA,
	w, h int,
	labels []int,
	region seg.Region,
	clip *image.Rectangle,
	pad int,
	stride int,
) (out *image.RGBA, meta Meta) {
	b := src.Bounds()
	var axisR seg.Region
	if clip == nil {
		axisR = region
	} else {
		var okTight bool
		axisR, okTight = tightRegionFromLabelInClip(w, h, labels, region.ID, clip)
		if !okTight {
			axisR = region
		}
	}
	pts := borderPointsSubsampled(w, h, labels, region.ID, stride, b, clip)
	if len(pts) < 3 {
		return toRGBAaxis(src, axisR, b), axisMeta(axisR, b)
	}
	hull := contour.ConvexHullMonotone(append([]geom.Point{}, pts...))
	if len(hull) < 3 {
		return toRGBAaxis(src, axisR, b), axisMeta(axisR, b)
	}

	val := geom.DefaultQuadValidation()
	var q [4]geom.Point
	var mode string
	var conf float64
	ok := false

	hullQuadAttempt := false
	if len(hull) == 4 {
		var h4 [4]geom.Point
		copy(h4[:], hull)
		if contour.IsConvex(h4) {
			hullQuadAttempt = true
			ord := geom.OrderCornersTopLeftCCW(h4)
			if geom.ValidateQuadOrdered(ord, val) == nil {
				q, mode, conf, ok = ord, "quad_hull", 0.88, true
			}
		}
	}
	if !ok {
		c4, _ := contour.MinAreaRectBrute(pts)
		ord := geom.OrderCornersTopLeftCCW(c4)
		if geom.ValidateQuadOrdered(ord, val) == nil {
			q = ord
			mode = "rotated_min_area_rect"
			if hullQuadAttempt {
				conf = 0.52
			} else {
				conf = 0.65
			}
			ok = true
		}
	}
	if !ok {
		return toRGBAaxis(src, axisR, b), axisMetaInvalidQuad(axisR, b)
	}

	c := geom.Centroid(q)
	if pad > 0 {
		q = geom.ExpandQuadFromCenter(q, c, float64(pad))
	}
	warp.ClampToImageBounds(&q, b)
	ow, oh := warp.Quadbounds(q)
	if ow < 2 || oh < 2 {
		return toRGBAaxis(src, axisR, b), axisMeta(axisR, b)
	}
	dst0 := [4]geom.Point{
		{X: 0, Y: 0}, {X: float64(ow - 1), Y: 0},
		{X: float64(ow - 1), Y: float64(oh - 1)}, {X: 0, Y: float64(oh - 1)},
	}
	hmat, okDLT := warp.DLT3x3(q, dst0)
	if !okDLT {
		return toRGBAaxis(src, axisR, b), axisMetaHomographyFail(axisR, b)
	}
	inv, okInv := warp.Invert3x3(hmat)
	if !okInv {
		return toRGBAaxis(src, axisR, b), axisMetaHomographyFail(axisR, b)
	}
	out = warp.PerspectiveWarp(src, inv, ow, oh)
	meta = metaFromCorners(q, mode, conf)
	return out, meta
}

func axisMeta(r seg.Region, b image.Rectangle) Meta {
	return Meta{
		Corners:    cornersAABB2(r, b),
		Mode:       "axis_aligned",
		Confidence: 0.2,
	}
}

func axisMetaInvalidQuad(r seg.Region, b image.Rectangle) Meta {
	return Meta{
		Corners:    cornersAABB2(r, b),
		Mode:       "axis_aligned_invalid_quad",
		Confidence: 0.12,
	}
}

func axisMetaHomographyFail(r seg.Region, b image.Rectangle) Meta {
	return Meta{
		Corners:    cornersAABB2(r, b),
		Mode:       "axis_aligned_homography_fail",
		Confidence: 0.22,
	}
}

func cornersAABB2(r seg.Region, b image.Rectangle) [4][2]float64 {
	ox, oy := float64(b.Min.X), float64(b.Min.Y)
	return [4][2]float64{
		{ox + float64(r.MinX), oy + float64(r.MinY)},
		{ox + float64(r.MaxX), oy + float64(r.MinY)},
		{ox + float64(r.MaxX), oy + float64(r.MaxY)},
		{ox + float64(r.MinX), oy + float64(r.MaxY)},
	}
}

func metaFromCorners(q [4]geom.Point, mode string, conf float64) Meta {
	return Meta{
		Corners: [4][2]float64{
			{q[0].X, q[0].Y}, {q[1].X, q[1].Y},
			{q[2].X, q[2].Y}, {q[3].X, q[3].Y},
		},
		Mode:       mode,
		Confidence: conf,
	}
}

func toRGBAaxis(src *image.RGBA, r seg.Region, b image.Rectangle) *image.RGBA {
	bb := image.Rect(b.Min.X+r.MinX, b.Min.Y+r.MinY, b.Min.X+r.MaxX+1, b.Min.Y+r.MaxY+1)
	bb = bb.Intersect(src.Bounds())
	if bb.Empty() {
		return image.NewRGBA(image.Rect(0, 0, 1, 1))
	}
	return toRGBA(src.SubImage(bb))
}

func borderPointsSubsampled(w, h int, labels []int, id int, stride int, b image.Rectangle, clip *image.Rectangle) []geom.Point {
	if stride < 1 {
		stride = 1
	}
	var pts []geom.Point
	c := 0
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			if labels[y*w+x] != id {
				continue
			}
			if clip != nil && !rectClipContains(clip, x, y) {
				continue
			}
			border := false
			for _, d := range [4]struct{ dx, dy int }{{-1, 0}, {1, 0}, {0, -1}, {0, 1}} {
				nx, ny := x+d.dx, y+d.dy
				if nx < 0 || ny < 0 || nx >= w || ny >= h {
					border = true
					break
				}
				if labels[ny*w+nx] != id {
					border = true
					break
				}
				if clip != nil && !rectClipContains(clip, nx, ny) {
					border = true
					break
				}
			}
			if !border {
				continue
			}
			c++
			if c%stride != 0 {
				continue
			}
			pts = append(pts, geom.Point{X: float64(b.Min.X) + float64(x), Y: float64(b.Min.Y) + float64(y)})
		}
	}
	if len(pts) > 4000 {
		s := 1 + len(pts)/3000
		var p2 []geom.Point
		for i := 0; i < len(pts); i += s {
			p2 = append(p2, pts[i])
		}
		pts = p2
	}
	return pts
}

// tightRegionFromLabelInClip returns the axis-aligned bbox of pixels with labels==id, optionally restricted to clip.
func tightRegionFromLabelInClip(w, h int, labels []int, id int, clip *image.Rectangle) (seg.Region, bool) {
	minX, minY := w, h
	maxX, maxY := -1, -1
	n := 0
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			if labels[y*w+x] != id {
				continue
			}
			if clip != nil && !rectClipContains(clip, x, y) {
				continue
			}
			n++
			if x < minX {
				minX = x
			}
			if y < minY {
				minY = y
			}
			if x > maxX {
				maxX = x
			}
			if y > maxY {
				maxY = y
			}
		}
	}
	if n == 0 || maxX < minX {
		return seg.Region{}, false
	}
	return seg.Region{
		ID: id, Area: n,
		Cx: float64(minX+maxX) / 2, Cy: float64(minY+maxY) / 2,
		MinX: minX, MinY: minY, MaxX: maxX, MaxY: maxY,
	}, true
}

func rectClipContains(clip *image.Rectangle, x, y int) bool {
	return x >= clip.Min.X && x < clip.Max.X && y >= clip.Min.Y && y < clip.Max.Y
}

// buildBinary: foreground=1, background=0 (foreground is darker than threshold).
func buildBinary(rgba *image.RGBA, th int) []byte {
	b := rgba.Bounds()
	w, h := b.Dx(), b.Dy()
	tu := uint8(th)
	out := make([]byte, w*h)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			c := rgba.RGBAAt(b.Min.X+x, b.Min.Y+y)
			if gray8(c) < tu {
				out[y*w+x] = 1
			}
		}
	}
	return out
}
