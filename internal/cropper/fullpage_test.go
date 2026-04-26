package cropper

import (
	"image"
	"image/color"
	"image/jpeg"
	"os"
	"path/filepath"
	"testing"

	"photo-cropper/internal/geom"
	"photo-cropper/internal/seg"
)

func TestIsFullPageCandidateByArea(t *testing.T) {
	r := seg.Region{MinX: 0, MinY: 0, MaxX: 99, MaxY: 99} // 100×100 in 100×100 source
	if !isFullPageCandidate(r, 100, 100) {
		t.Fatal("full bbox should reject (area ratio > 0.85)")
	}
	r2 := seg.Region{MinX: 0, MinY: 0, MaxX: 84, MaxY: 99} // 85×100 = 8500/10000 = 0.85, not > 0.85
	if isFullPageCandidate(r2, 100, 100) {
		t.Fatal("exactly 0.85 area ratio should not reject (must be > 0.85)")
	}
	r3 := seg.Region{MinX: 0, MinY: 0, MaxX: 85, MaxY: 99} // 86×100 > 0.85
	if !isFullPageCandidate(r3, 100, 100) {
		t.Fatal("area fraction > 0.85 should reject")
	}
}

func TestIsFullPageCandidateBySides(t *testing.T) {
	// 91% × 91% of 200×200 without exceeding 0.85 area if thin? 182×182 / 40000 = 0.828 < 0.85
	// use 190×190 in 200×200: area 36100/40000=0.9025 > 0.85 triggers first rule anyway
	// narrow band: width 190 height 50 on 200×200 -> area 9500/40000=0.237 — need width>=180 height>=180
	r := seg.Region{MinX: 0, MinY: 0, MaxX: 180, MaxY: 180} // 181×181
	if !isFullPageCandidate(r, 200, 200) {
		t.Fatal("both sides >= 90% should reject")
	}
	r2 := seg.Region{MinX: 0, MinY: 0, MaxX: 170, MaxY: 180} // width 171/200=0.855, height 181/200=0.905 — width < 90%
	if isFullPageCandidate(r2, 200, 200) {
		t.Fatalf("only one side >= 90%% should not reject by side rule: %+v", r2)
	}
}

func TestQuadCornersCoverFullSource(t *testing.T) {
	b := image.Rect(0, 0, 100, 100)
	meta := Meta{
		Corners: [4][2]float64{{0, 0}, {99, 0}, {99, 99}, {0, 99}},
		Mode:    "quad_hull",
	}
	if !quadCornersCoverFullSource(meta, b) {
		t.Fatal("corners on full frame should reject")
	}
	meta2 := Meta{
		Corners: [4][2]float64{{10, 10}, {40, 8}, {42, 50}, {8, 48}},
		Mode:    "quad_hull",
	}
	if quadCornersCoverFullSource(meta2, b) {
		t.Fatal("small quad should not reject")
	}
}

func TestProcessScanRejectsMonolithicFullPage(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "fullfield.jpg")
	const size = 220
	img := image.NewRGBA(image.Rect(0, 0, size, size))
	d := color.RGBA{R: 15, G: 15, B: 15, A: 255}
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			img.SetRGBA(x, y, d)
		}
	}
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := jpeg.Encode(f, img, &jpeg.Options{Quality: 90}); err != nil {
		t.Fatal(err)
	}
	_ = f.Close()

	opts := Options{Threshold: 245, MinArea: 5000, Padding: 0, Aspect: 0, DebugDir: ""}
	_, out, metas, err := ProcessScan(path, opts)
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 0 || len(metas) != 0 {
		t.Fatalf("full-field foreground must yield zero crops, got %d", len(out))
	}
}

// TestProcessScanNoCropMatchesSourceDimensions ensures no extracted crop has the same pixel size as the source
// when the only component would have been a full-page reject (integration-style via ProcessScan).
func TestProcessScanNoCropMatchesSourceDimensions(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "page.jpg")
	const W, H = 240, 180
	img := image.NewRGBA(image.Rect(0, 0, W, H))
	fill := color.RGBA{R: 12, G: 12, B: 14, A: 255}
	for y := 0; y < H; y++ {
		for x := 0; x < W; x++ {
			img.SetRGBA(x, y, fill)
		}
	}
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := jpeg.Encode(f, img, &jpeg.Options{Quality: 90}); err != nil {
		t.Fatal(err)
	}
	_ = f.Close()

	opts := Options{Threshold: 245, MinArea: 1000, Padding: 0, Aspect: 0, DebugDir: ""}
	_, out, _, err := ProcessScan(path, opts)
	if err != nil {
		t.Fatal(err)
	}
	for i, sub := range out {
		b := sub.Bounds()
		if b.Dx() == W && b.Dy() == H {
			t.Fatalf("crop %d must not match full source size %dx%d", i, W, H)
		}
	}
}

func TestQuadCornersRejectUsesGeomQuad(t *testing.T) {
	// Region bbox small (not full-page) but corners span full image — rare but must filter after extractRegion.
	b := image.Rect(10, 10, 210, 210) // 200×200 logical size
	sw, sh := b.Dx(), b.Dy()
	q := [4]geom.Point{
		{X: float64(b.Min.X), Y: float64(b.Min.Y)},
		{X: float64(b.Max.X - 1), Y: float64(b.Min.Y)},
		{X: float64(b.Max.X - 1), Y: float64(b.Max.Y - 1)},
		{X: float64(b.Min.X), Y: float64(b.Max.Y - 1)},
	}
	meta := metaFromCorners(q, "quad_hull", 0.9)
	if !quadCornersCoverFullSource(meta, b) {
		t.Fatalf("quad spanning full %dx%d source should reject", sw, sh)
	}
}
