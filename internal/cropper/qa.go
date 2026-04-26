package cropper

import (
	"image"
	"image/color"
	"image/draw"
	"math"
	"strconv"
	"strings"

	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/math/fixed"
)

// QA edge stroke: number of parallel Bresenham lines drawn perpendicular to each segment.
// For segment direction (vx, vy), the unit normal is (-vy, vx)/len; integer offsets k*n
// shift each copy so the bundle reads as a ~3–5px stroke; bres clips each pixel to image bounds.
const qaEdgeParallelLines = 5

// qaCornerRadius is the disk radius (pixels) for corner highlights on the QA overlay.
const qaCornerRadius = 4

// QaScanOverlay returns a new RGBA copy of src with each crop’s quad, corner markers, and
// 1-based index labels, using the same per-photo geometry as the exported crops (metas[i].Corners).
func QaScanOverlay(src *image.RGBA, metas []Meta) *image.RGBA {
	dupe := toRGBA(src)
	if len(metas) == 0 {
		return dupe
	}
	face := basicfont.Face7x13
	for i, m := range metas {
		line := qaLineColorForMode(m.Mode)
		for j := 0; j < 4; j++ {
			k := (j + 1) % 4
			x0 := int(0.5 + m.Corners[j][0])
			y0 := int(0.5 + m.Corners[j][1])
			x1 := int(0.5 + m.Corners[k][0])
			y1 := int(0.5 + m.Corners[k][1])
			bresThick(dupe, x0, y0, x1, y1, line)
		}
		mark := line
		r2 := qaCornerRadius * qaCornerRadius
		for j := 0; j < 4; j++ {
			cx, cy := int(0.5+m.Corners[j][0]), int(0.5+m.Corners[j][1])
			for dy := -qaCornerRadius; dy <= qaCornerRadius; dy++ {
				for dx := -qaCornerRadius; dx <= qaCornerRadius; dx++ {
					if dx*dx+dy*dy > r2 {
						continue
					}
					x, y := cx+dx, cy+dy
					bb := dupe.Bounds()
					if x >= bb.Min.X && x < bb.Max.X && y >= bb.Min.Y && y < bb.Max.Y {
						dupe.SetRGBA(x, y, mark)
					}
				}
			}
		}
		var sx, sy float64
		for j := 0; j < 4; j++ {
			sx += m.Corners[j][0]
			sy += m.Corners[j][1]
		}
		cx := int(0.5 + sx/4.0)
		cy := int(0.5 + sy/4.0)
		label := strconv.Itoa(i + 1)
		drawQALabel(dupe, face, cx, cy, line, label)
	}
	return dupe
}

// bresThick draws a 1px Bresenham line plus parallel copies offset along the unit normal
// (-Δy, Δx)/‖Δ‖ so the stroke appears thicker without new dependencies. Each offset line is
// drawn with bres, which only paints in-bounds pixels (edges stay clipped to the image).
func bresThick(img *image.RGBA, x0, y0, x1, y1 int, c color.RGBA) {
	dx := float64(x1 - x0)
	dy := float64(y1 - y0)
	rlen := math.Hypot(dx, dy)
	if rlen < 1e-9 {
		return
	}
	nx := -dy / rlen
	ny := dx / rlen
	n := qaEdgeParallelLines
	half := n / 2
	for k := -half; k <= half; k++ {
		ox := int(math.Round(float64(k) * nx))
		oy := int(math.Round(float64(k) * ny))
		bres(img, x0+ox, y0+oy, x1+ox, y1+oy, c)
	}
}

// qaLineColorForMode maps detection mode to QA overlay stroke/fill (geometry unchanged).
func qaLineColorForMode(mode string) color.RGBA {
	switch mode {
	case "quad_hull":
		return color.RGBA{R: 0, G: 200, B: 40, A: 255}
	case "rotated_min_area_rect":
		return color.RGBA{R: 255, G: 230, B: 0, A: 255}
	default:
		if strings.HasPrefix(mode, "axis_aligned") {
			return color.RGBA{R: 255, G: 50, B: 50, A: 255}
		}
		return color.RGBA{R: 180, G: 180, B: 200, A: 255}
	}
}

func drawQALabel(dst *image.RGBA, face font.Face, cx, cy int, accent color.RGBA, text string) {
	if text == "" {
		return
	}
	fg := color.NRGBA{255, 255, 120, 255}
	d0 := &font.Drawer{Dst: dst, Face: face}
	adv := d0.MeasureString(text)
	w := adv.Ceil()
	if w < 1 {
		return
	}
	const h = 15
	padX, padY := 3, 2
	rect := image.Rect(cx-w/2-padX, cy-h/2-padY, cx+w/2+padX, cy+h/2+padY)
	rect = rect.Intersect(dst.Bounds())
	if rect.Empty() {
		return
	}
	border := color.NRGBA{accent.R, accent.G, accent.B, 255}
	drawBorder(dst, rect, border, 1)
	draw.Draw(dst, rect.Inset(1), &image.Uniform{color.NRGBA{8, 8, 24, 210}}, image.Point{}, draw.Over)
	d := &font.Drawer{Dst: dst, Src: &image.Uniform{fg}, Face: face, Dot: fixed.P(cx-w/2+1, cy+h/2+3)}
	d.DrawString(text)
}

func drawBorder(dst *image.RGBA, r image.Rectangle, c color.NRGBA, w int) {
	b := dst.Bounds()
	for t := 0; t < w; t++ {
		rr := r.Inset(-t)
		for x := rr.Min.X; x < rr.Max.X; x++ {
			if x < b.Min.X || x >= b.Max.X {
				continue
			}
			if rr.Min.Y >= b.Min.Y && rr.Min.Y < b.Max.Y {
				dst.Set(x, rr.Min.Y, c)
			}
			if rr.Max.Y-1 >= b.Min.Y && rr.Max.Y-1 < b.Max.Y {
				dst.Set(x, rr.Max.Y-1, c)
			}
		}
		for y := rr.Min.Y; y < rr.Max.Y; y++ {
			if y < b.Min.Y || y >= b.Max.Y {
				continue
			}
			if rr.Min.X >= b.Min.X && rr.Min.X < b.Max.X {
				dst.Set(rr.Min.X, y, c)
			}
			if rr.Max.X-1 >= b.Min.X && rr.Max.X-1 < b.Max.X {
				dst.Set(rr.Max.X-1, y, c)
			}
		}
	}
}
