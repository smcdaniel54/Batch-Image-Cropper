package warp

import (
	"image"
	"image/color"
	"math"

	"photo-cropper/internal/geom"
)

// DLT3x3 computes homography h from 4 point pairs (h22=1), src to dst. Returns false if singular.
func DLT3x3(src, dst [4]geom.Point) (h [3][3]float64, ok bool) {
	// 8x9 matrix A, solve Ah=0, h last component 1
	var a [8][8]float64
	var b [8]float64
	k := 0
	for i := 0; i < 4; i++ {
		sx, sy, dx, dy := src[i].X, src[i].Y, dst[i].X, dst[i].Y
		a[k][0] = sx
		a[k][1] = sy
		a[k][2] = 1
		a[k][3] = 0
		a[k][4] = 0
		a[k][5] = 0
		a[k][6] = -dx * sx
		a[k][7] = -dx * sy
		b[k] = dx
		k++
		a[k][0] = 0
		a[k][1] = 0
		a[k][2] = 0
		a[k][3] = sx
		a[k][4] = sy
		a[k][5] = 1
		a[k][6] = -dy * sx
		a[k][7] = -dy * sy
		b[k] = dy
		k++
	}
	x, ok2 := solve8(a, b)
	if !ok2 {
		return h, false
	}
	h[0][0] = x[0]
	h[0][1] = x[1]
	h[0][2] = x[2]
	h[1][0] = x[3]
	h[1][1] = x[4]
	h[1][2] = x[5]
	h[2][0] = x[6]
	h[2][1] = x[7]
	h[2][2] = 1
	return h, true
}

func solve8(a [8][8]float64, b [8]float64) ([]float64, bool) {
	// Gauss-Jordan
	n := 8
	m := n + 1
	aug := make([][]float64, n)
	for i := 0; i < n; i++ {
		aug[i] = make([]float64, m)
		for j := 0; j < n; j++ {
			aug[i][j] = a[i][j]
		}
		aug[i][n] = b[i]
	}
	for col := 0; col < n; col++ {
		// partial pivot
		piv := col
		maxA := math.Abs(aug[piv][col])
		for r := col + 1; r < n; r++ {
			if v := math.Abs(aug[r][col]); v > maxA {
				maxA = v
				piv = r
			}
		}
		if maxA < 1e-12 {
			return nil, false
		}
		aug[col], aug[piv] = aug[piv], aug[col]
		p := aug[col][col]
		for j := col; j <= n; j++ {
			aug[col][j] /= p
		}
		for r := 0; r < n; r++ {
			if r == col {
				continue
			}
			f := aug[r][col]
			for j := col; j <= n; j++ {
				aug[r][j] -= f * aug[col][j]
			}
		}
	}
	x := make([]float64, n)
	for i := 0; i < n; i++ {
		x[i] = aug[i][n]
	}
	return x, true
}

// Invert3x3 in place small matrix.
func Invert3x3(m [3][3]float64) (o [3][3]float64, ok bool) {
	det := m[0][0]*(m[1][1]*m[2][2]-m[1][2]*m[2][1]) -
		m[0][1]*(m[1][0]*m[2][2]-m[1][2]*m[2][0]) +
		m[0][2]*(m[1][0]*m[2][1]-m[1][1]*m[2][0])
	if math.Abs(det) < 1e-15 {
		return o, false
	}
	invf := 1.0 / det
	o[0][0] = (m[1][1]*m[2][2] - m[1][2]*m[2][1]) * invf
	o[0][1] = (m[0][2]*m[2][1] - m[0][1]*m[2][2]) * invf
	o[0][2] = (m[0][1]*m[1][2] - m[0][2]*m[1][1]) * invf
	o[1][0] = (m[1][2]*m[2][0] - m[1][0]*m[2][2]) * invf
	o[1][1] = (m[0][0]*m[2][2] - m[0][2]*m[2][0]) * invf
	o[1][2] = (m[0][2]*m[1][0] - m[0][0]*m[1][2]) * invf
	o[2][0] = (m[1][0]*m[2][1] - m[1][1]*m[2][0]) * invf
	o[2][1] = (m[0][1]*m[2][0] - m[0][0]*m[2][1]) * invf
	o[2][2] = (m[0][0]*m[1][1] - m[0][1]*m[1][0]) * invf
	return o, true
}

// MultH applies H to 2D point as homogeneous, returns w-scaled; divide by w for image coords
func multH(m [3][3]float64, p geom.Point) (x, y, w3 float64) {
	x3 := m[0][0]*p.X + m[0][1]*p.Y + m[0][2]
	y3 := m[1][0]*p.X + m[1][1]*p.Y + m[1][2]
	w3 = m[2][0]*p.X + m[2][1]*p.Y + m[2][2]
	return x3, y3, w3
}

// Bilinear at float coords in source image, bounds safe (white outside).
func BilinearAt(img *image.RGBA, x, y float64) color.RGBA {
	b := img.Bounds()
	xi, yi := int(math.Floor(x)), int(math.Floor(y))
	xf, yf := x-float64(xi), y-float64(yi)
	var c00, c10, c01, c11 color.RGBA
	inside := func(ix, iy int) bool { return ix >= b.Min.X && ix < b.Max.X && iy >= b.Min.Y && iy < b.Max.Y }
	white := color.RGBA{255, 255, 255, 255}
	if !inside(xi, yi) || !inside(xi+1, yi) || !inside(xi, yi+1) || !inside(xi+1, yi+1) {
		// partial outside: sample with white padding
		get := func(ix, iy int) color.RGBA {
			if !inside(ix, iy) {
				return white
			}
			return img.RGBAAt(ix, iy)
		}
		c00, c10 = get(xi, yi), get(xi+1, yi)
		c01, c11 = get(xi, yi+1), get(xi+1, yi+1)
	} else {
		c00, c10 = img.RGBAAt(xi, yi), img.RGBAAt(xi+1, yi)
		c01, c11 = img.RGBAAt(xi, yi+1), img.RGBAAt(xi+1, yi+1)
	}
	r := lerp(lerp(float64(c00.R), float64(c10.R), xf), lerp(float64(c01.R), float64(c11.R), xf), yf)
	g := lerp(lerp(float64(c00.G), float64(c10.G), xf), lerp(float64(c01.G), float64(c11.G), xf), yf)
	bl := lerp(lerp(float64(c00.B), float64(c10.B), xf), lerp(float64(c01.B), float64(c11.B), xf), yf)
	return color.RGBA{uint8(r + 0.5), uint8(g + 0.5), uint8(bl + 0.5), 255}
}

func lerp(a, b, t float64) float64 { return a + t*(b-a) }

// PerspectiveWarp into outW x outH, mapping dst pixels back through invH to src. invH: dst->src
func PerspectiveWarp(src *image.RGBA, invH [3][3]float64, outW, outH int) *image.RGBA {
	b := image.Rect(0, 0, outW, outH)
	out := image.NewRGBA(b)
	white := color.RGBA{255, 255, 255, 255}
	for y := 0; y < outH; y++ {
		for x := 0; x < outW; x++ {
			px, py, pw := multH(invH, geom.Point{X: float64(x), Y: float64(y)})
			if math.Abs(pw) < 1e-9 {
				out.SetRGBA(x, y, white)
				continue
			}
			sx, sy := px/pw, py/pw
			out.SetRGBA(x, y, BilinearAt(src, sx, sy))
		}
	}
	return out
}

// Quadbounds returns positive output size from corner distances.
func Quadbounds(s [4]geom.Point) (w, h int) {
	dist := func(a, b geom.Point) float64 {
		return math.Hypot(b.X-a.X, b.Y-a.Y)
	}
	// s is TL, TR, BR, BL
	dTop := dist(s[0], s[1])
	dBottom := dist(s[3], s[2])
	dLeft := dist(s[0], s[3])
	dRight := dist(s[1], s[2])
	wf := math.Max(dTop, dBottom)
	hf := math.Max(dLeft, dRight)
	if wf < 1 {
		wf = 1
	}
	if hf < 1 {
		hf = 1
	}
	return int(0.5 + wf), int(0.5 + hf)
}

// EnforceAspect center-crops img to w/h = aspect. If aspect <= 0, returns img and false.
func EnforceAspect(img *image.RGBA, aspect float64) *image.RGBA {
	if aspect <= 0 {
		return img
	}
	b := img.Bounds()
	iw, ih := b.Dx(), b.Dy()
	if iw < 1 || ih < 1 {
		return img
	}
	r := float64(iw) / float64(ih)
	var cw, ch int
	if r > aspect {
		cw = int(0.5 + float64(ih)*aspect)
		if cw < 1 {
			cw = 1
		}
		ch = ih
	} else {
		cw = iw
		ch = int(0.5 + float64(iw)/aspect)
		if ch < 1 {
			ch = 1
		}
	}
	x0 := b.Min.X + (iw-cw)/2
	y0 := b.Min.Y + (ih-ch)/2
	nb := image.Rect(0, 0, cw, ch)
	nc := image.NewRGBA(nb)
	for yy := 0; yy < ch; yy++ {
		for xx := 0; xx < cw; xx++ {
			nc.SetRGBA(xx, yy, img.RGBAAt(x0+xx, y0+yy))
		}
	}
	return nc
}

// ClampToImageBounds in place clamps corners to the given image rectangle (inclusive of Min, exclusive of Max for pixel access).
func ClampToImageBounds(pts *[4]geom.Point, b image.Rectangle) {
	mx0 := float64(b.Min.X)
	my0 := float64(b.Min.Y)
	mx1 := float64(b.Max.X) - 1
	my1 := float64(b.Max.Y) - 1
	if mx1 < mx0 {
		mx1 = mx0
	}
	if my1 < my0 {
		my1 = my0
	}
	for i := 0; i < 4; i++ {
		pts[i].X = math.Max(mx0, math.Min(mx1, pts[i].X))
		pts[i].Y = math.Max(my0, math.Min(my1, pts[i].Y))
	}
}
