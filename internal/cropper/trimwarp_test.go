package cropper

import (
	"image"
	"image/color"
	"testing"
)

// TestPostWarpTrimMarginsRemovesWhiteBorder builds a warped-like frame: large near-white margin
// and a darker card interior; postWarpTrimMargins must shrink bounds to the card plus padding.
func TestPostWarpTrimMarginsRemovesWhiteBorder(t *testing.T) {
	const W, H = 100, 100
	th := 245
	img := image.NewRGBA(image.Rect(0, 0, W, H))
	white := color.RGBA{R: 252, G: 252, B: 252, A: 255}
	dark := color.RGBA{R: 20, G: 20, B: 22, A: 255}
	for y := 0; y < H; y++ {
		for x := 0; x < W; x++ {
			img.SetRGBA(x, y, white)
		}
	}
	// Inset "photo" 30×40 at (35,30) — foreground under default threshold
	x0, y0, x1, y1 := 35, 30, 65, 70
	for y := y0; y < y1; y++ {
		for x := x0; x < x1; x++ {
			img.SetRGBA(x, y, dark)
		}
	}
	out := postWarpTrimMargins(img, th)
	b := out.Bounds()
	if b.Dx() >= W || b.Dy() >= H {
		t.Fatalf("expected trimmed image smaller than %dx%d, got bounds %v", W, H, b)
	}
	// inner 30×40 + 2*postWarpTrimPadPx on each side
	wantW := (x1 - x0) + 2*postWarpTrimPadPx
	wantH := (y1 - y0) + 2*postWarpTrimPadPx
	if b.Dx() != wantW || b.Dy() != wantH {
		t.Fatalf("got %dx%d want %dx%d (tight bbox + pad)", b.Dx(), b.Dy(), wantW, wantH)
	}
	// trimmed region should still be mostly non-white at center
	mid := out.RGBAAt(b.Min.X+b.Dx()/2, b.Min.Y+b.Dy()/2)
	if gray8(mid) >= uint8(th) {
		t.Fatalf("center pixel should remain foreground after trim: %+v", mid)
	}
}

func TestPostWarpTrimMarginsAllBackgroundUnchanged(t *testing.T) {
	const W, H = 40, 40
	img := image.NewRGBA(image.Rect(0, 0, W, H))
	c := color.RGBA{R: 250, G: 250, B: 250, A: 255}
	for y := 0; y < H; y++ {
		for x := 0; x < W; x++ {
			img.SetRGBA(x, y, c)
		}
	}
	out := postWarpTrimMargins(img, 245)
	if out.Bounds() != img.Bounds() {
		t.Fatalf("no foreground: bounds should match input, got %v", out.Bounds())
	}
}
