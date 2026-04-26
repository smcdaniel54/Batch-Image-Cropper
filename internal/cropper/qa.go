package cropper

import (
	"image"
	"image/color"
	"image/draw"
	"strconv"

	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/math/fixed"
)

// QaScanOverlay returns a new RGBA copy of src with each crop’s quad, corner markers, and
// 1-based index labels, using the same per-photo geometry as the exported crops (metas[i].Corners).
func QaScanOverlay(src *image.RGBA, metas []Meta) *image.RGBA {
	dupe := toRGBA(src)
	if len(metas) == 0 {
		return dupe
	}
	face := basicfont.Face7x13
	for i, m := range metas {
		line := hullColors[i%len(hullColors)]
		for j := 0; j < 4; j++ {
			k := (j + 1) % 4
			x0 := int(0.5 + m.Corners[j][0])
			y0 := int(0.5 + m.Corners[j][1])
			x1 := int(0.5 + m.Corners[k][0])
			y1 := int(0.5 + m.Corners[k][1])
			bres(dupe, x0, y0, x1, y1, line)
		}
		mark := color.RGBA{255, 50, 50, 255}
		for j := 0; j < 4; j++ {
			cx, cy := int(0.5+m.Corners[j][0]), int(0.5+m.Corners[j][1])
			for dy := -2; dy <= 2; dy++ {
				for dx := -2; dx <= 2; dx++ {
					if maxAbs(dx, dy) > 2 {
						continue
					}
					if abs(dx)+abs(dy) > 3 {
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

var hullColors = []color.RGBA{
	{0, 180, 0, 255},
	{0, 100, 255, 255},
	{200, 0, 200, 255},
	{255, 140, 0, 255},
	{0, 200, 200, 255},
	{100, 200, 0, 255},
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
