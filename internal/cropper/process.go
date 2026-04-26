package cropper

import (
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strings"

	"photo-cropper/internal/seg"
	"photo-cropper/internal/warp"
)

// ProcessScan loads a scan, segments it, and returns the full-decoded source (for QA overlays) and
// extracted photos in reading order. Returned crop images are always warped or sub-rectangle crops;
// the raw scan file bytes are not re-saved to callers, only a decoded *image.RGBA.
func ProcessScan(path string, o Options) (source *image.RGBA, out []*image.RGBA, metas []Meta, err error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, nil, nil, err
	}
	defer f.Close()
	img, _, err := image.Decode(f)
	if err != nil {
		return nil, nil, nil, err
	}
	src := toRGBA(img)
	b := src.Bounds()
	w, h := b.Dx(), b.Dy()
	if w < 1 || h < 1 {
		return nil, nil, nil, nil
	}
	mask := buildBinary(src, o.Threshold)
	labels, _, _ := seg.Label4Connect(w, h, mask)
	regions := seg.BuildRegions(w, h, labels, o.MinArea)
	seg.SortTopToBottomLeftToRight(regions)
	stride := 2
	if w*h > 8_000_000 {
		stride = 3
	}
	var allImg []*image.RGBA
	var allM []Meta
	for _, r := range regions {
		sub, meta := extractRegion(src, w, h, labels, r, o.Padding, stride)
		if o.Aspect > 0 {
			sub = warp.EnforceAspect(sub, o.Aspect)
		}
		allImg = append(allImg, sub)
		allM = append(allM, meta)
	}
	if o.DebugDir != "" {
		saveDebug(path, o.DebugDir, src, allM)
	}
	return src, allImg, allM, nil
}

// saveDebug writes one PNG with all detected quads and corner markers.
func saveDebug(path, dir string, src *image.RGBA, metas []Meta) {
	dupe := toRGBA(src)
	green := color.RGBA{0, 200, 0, 255}
	red := color.RGBA{255, 0, 0, 255}
	for _, m := range metas {
		for i := 0; i < 4; i++ {
			j := (i + 1) % 4
			x0 := int(0.5 + m.Corners[i][0])
			y0 := int(0.5 + m.Corners[i][1])
			x1 := int(0.5 + m.Corners[j][0])
			y1 := int(0.5 + m.Corners[j][1])
			bres(dupe, x0, y0, x1, y1, green)
		}
		for i := 0; i < 4; i++ {
			cx, cy := int(0.5+m.Corners[i][0]), int(0.5+m.Corners[i][1])
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
						dupe.SetRGBA(x, y, red)
					}
				}
			}
		}
	}
	stem := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	p := filepath.Join(dir, stem+"_debug.png")
	of, err := os.Create(p)
	if err != nil {
		return
	}
	_ = png.Encode(of, dupe)
	_ = of.Close()
}

func maxAbs(a, b int) int {
	if a < 0 {
		a = -a
	}
	if b < 0 {
		b = -b
	}
	if a > b {
		return a
	}
	return b
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

// bres Bresenham in image space (integer pixel coords, aligned with 0,0 of src).
func bres(img *image.RGBA, x0, y0, x1, y1 int, c color.RGBA) {
	bb := img.Bounds()
	dx := abs(x1 - x0)
	dy := abs(y1 - y0)
	sx, sy := 1, 1
	if x0 > x1 {
		sx = -1
	}
	if y0 > y1 {
		sy = -1
	}
	errL := dx - dy
	x, y := x0, y0
	maxSt := 4 * (abs(dx) + abs(dy) + 1)
	for k := 0; k < maxSt; k++ {
		if x >= bb.Min.X && x < bb.Max.X && y >= bb.Min.Y && y < bb.Max.Y {
			img.SetRGBA(x, y, c)
		}
		if x == x1 && y == y1 {
			break
		}
		e2 := 2 * errL
		if e2 > -dy {
			errL -= dy
			x += sx
		}
		if e2 < dx {
			errL += dx
			y += sy
		}
	}
}
