package cropper

import (
	"image"
	"image/color"
	"testing"
)

// TestBresThickDrawsMorePixelsThanThin checks the parallel-offset strategy: several shifted
// Bresenham copies along the segment normal produce a visibly thicker stroke than a single bres.
func TestBresThickDrawsMorePixelsThanThin(t *testing.T) {
	const w, h = 80, 80
	c := color.RGBA{0, 200, 0, 255}
	x0, y0, x1, y1 := 10, 15, 65, 55

	imgThin := image.NewRGBA(image.Rect(0, 0, w, h))
	bres(imgThin, x0, y0, x1, y1, c)
	nThin := countRGBA(imgThin, c)

	imgThick := image.NewRGBA(image.Rect(0, 0, w, h))
	bresThick(imgThick, x0, y0, x1, y1, c)
	nThick := countRGBA(imgThick, c)

	if nThick <= nThin {
		t.Fatalf("thick stroke should cover more pixels than thin: thin=%d thick=%d", nThin, nThick)
	}
	if nThick < nThin+qaEdgeParallelLines-1 {
		t.Fatalf("expected at least ~%d extra pixels from parallel lines, thin=%d thick=%d", qaEdgeParallelLines-1, nThin, nThick)
	}
}

func TestQaLineColorForMode(t *testing.T) {
	want := map[string]color.RGBA{
		"quad_hull":                    {R: 0, G: 200, B: 40, A: 255},
		"rotated_min_area_rect":        {R: 255, G: 230, B: 0, A: 255},
		"axis_aligned":                 {R: 255, G: 50, B: 50, A: 255},
		"axis_aligned_invalid_quad":    {R: 255, G: 50, B: 50, A: 255},
		"axis_aligned_homography_fail": {R: 255, G: 50, B: 50, A: 255},
	}
	for mode, w := range want {
		if got := qaLineColorForMode(mode); got != w {
			t.Fatalf("%s: got %+v want %+v", mode, got, w)
		}
	}
	unknown := qaLineColorForMode("future_mode")
	if unknown.R != 180 || unknown.G != 180 {
		t.Fatalf("unknown mode should use neutral fallback: %+v", unknown)
	}
}

func countRGBA(img *image.RGBA, want color.RGBA) int {
	n := 0
	b := img.Bounds()
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			if img.RGBAAt(x, y) == want {
				n++
			}
		}
	}
	return n
}
