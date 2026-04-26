package warp

import (
	"image"
	"image/color"
	"testing"

	"photo-cropper/internal/geom"
)

func TestEnforceAspectWider(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 100, 100))
	// 1:1 into 1.5:1 -> crop top/bottom
	out := EnforceAspect(img, 1.5)
	if out.Bounds().Dx() != 100 {
		t.Fatalf("width: %d", out.Bounds().Dx())
	}
	// 100/1.5 = 66.66 -> 67
	if out.Bounds().Dy() != 67 {
		t.Fatalf("height: %d", out.Bounds().Dy())
	}
}

func TestEnforceAspectTaller(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 200, 100))
	// 2:1, aspect=1.0 square -> cut width
	out := EnforceAspect(img, 1.0)
	if out.Bounds().Dx() != 100 {
		t.Fatalf("width: %d", out.Bounds().Dx())
	}
	if out.Bounds().Dy() != 100 {
		t.Fatalf("height: %d", out.Bounds().Dy())
	}
}

func TestEnforceAspectZeroPassthrough(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 10, 5))
	if EnforceAspect(img, 0) != img {
		t.Fatal("zero aspect should return same")
	}
}

func TestClampToImageBounds(t *testing.T) {
	pts := [4]geom.Point{
		{X: -5, Y: 10}, {X: 20, Y: 20}, {X: 30, Y: 300}, {X: 10, Y: 10},
	}
	b := image.Rect(0, 0, 50, 50)
	ClampToImageBounds(&pts, b)
	if pts[0].X < 0 || pts[0].X > 49 || pts[0].Y < 0 {
		t.Fatalf("clamped: %+v", pts[0])
	}
	if pts[2].Y > 49 {
		t.Fatalf("y not clamped: %v", pts[2].Y)
	}
}

func TestBilinearAtOutsideWhite(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 2, 2))
	img.SetRGBA(0, 0, color.RGBA{A: 255})
	c := BilinearAt(img, 10, 10)
	if c.R != 255 {
		t.Fatalf("far outside sample R got %d want 255", c.R)
	}
}
