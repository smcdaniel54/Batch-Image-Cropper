package cropper

import (
	"image"
	"image/color"
	"image/jpeg"
	"os"
	"path/filepath"
	"testing"

	"batch-image-cropper/internal/seg"
)

// writeMergedTwoPhotosScanJPEG draws two large dark cards on white, connected only along a thin
// top row so 4-connectivity yields one component with a vertical background gap through the bodies.
func writeMergedTwoPhotosScanJPEG(t *testing.T, path string) {
	t.Helper()
	const w, h = 420, 200
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	white := color.RGBA{R: 255, G: 255, B: 255, A: 255}
	dark := color.RGBA{R: 12, G: 12, B: 12, A: 255}
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.SetRGBA(x, y, white)
		}
	}
	fill := func(x0, y0, x1, y1 int) {
		for y := y0; y < y1; y++ {
			for x := x0; x < x1; x++ {
				img.SetRGBA(x, y, dark)
			}
		}
	}
	// Bodies + thin bridge row; interior columns stay white on the bridge so mask shows a vertical gap.
	fill(10, 25, 161, 176)  // left body
	fill(270, 25, 411, 176) // right body
	fill(10, 15, 161, 16)   // bridge segment over left (y=15)
	fill(270, 15, 411, 16)  // bridge segment over right (y=15); x=161..269 stays white (thin gap)
	fill(160, 15, 161, 26)  // connect left body to left bridge
	fill(270, 15, 271, 26)  // connect right body to right bridge
	// Bottom rim connects the two bodies around the side so labeling is one component.
	fill(10, 199, 411, 200)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if err := jpeg.Encode(f, img, &jpeg.Options{Quality: 92}); err != nil {
		t.Fatal(err)
	}
}

func TestProcessScanSplitsMergedComponentByVerticalGap(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "merged.jpg")
	writeMergedTwoPhotosScanJPEG(t, path)
	opts := Options{Threshold: 245, MinArea: 20000, Padding: 0, Aspect: 0}
	_, out, metas, err := ProcessScan(path, opts)
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 2 {
		t.Fatalf("merged component with interior vertical gap: want 2 crops, got %d (metas %d)", len(out), len(metas))
	}
	if len(metas) != 2 {
		t.Fatalf("want 2 metas, got %d", len(metas))
	}
}

func TestProcessScanSingleRectangleDoesNotSplit(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "onecard.jpg")
	const w, h = 220, 220
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	white := color.RGBA{R: 255, G: 255, B: 255, A: 255}
	dark := color.RGBA{R: 10, G: 10, B: 10, A: 255}
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.SetRGBA(x, y, white)
		}
	}
	for y := 30; y < 190; y++ {
		for x := 30; x < 190; x++ {
			img.SetRGBA(x, y, dark)
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

	opts := Options{Threshold: 245, MinArea: 20000, Padding: 0, Aspect: 0}
	_, out, _, err := ProcessScan(path, opts)
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 1 {
		t.Fatalf("single component: want 1 crop, got %d", len(out))
	}
}

func TestTrySplitRegionClipsDirectlyVertical(t *testing.T) {
	const w, h = 100, 40
	mask := make([]byte, w*h)
	labels := make([]int, w*h)
	const id = 1
	// Foreground: two blocks with full-height white gap columns 48–52 (5 px), connected by y=0 bridge.
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			fg := false
			if y == 0 && x >= 5 && x < 95 {
				fg = true
			}
			if y >= 5 && x >= 5 && x < 45 {
				fg = true
			}
			if y >= 5 && x >= 55 && x < 95 {
				fg = true
			}
			if fg {
				mask[y*w+x] = 1
				labels[y*w+x] = id
			}
		}
	}
	r := seg.Region{ID: id, MinX: 5, MinY: 0, MaxX: 94, MaxY: h - 1, Area: 1000}
	clips := trySplitRegionClips(w, h, mask, labels, r, 400)
	if len(clips) != 2 {
		t.Fatalf("want 2 clips from synthetic mask, got %v (n=%d)", clips, len(clips))
	}
	if clips[0].Max.X > 48 || clips[1].Min.X <= 52 {
		t.Fatalf("expected split around gap cols 48–52, got left %v right %v", clips[0], clips[1])
	}
}

func TestTrySplitRegionClipsSingleBlobNoVertical(t *testing.T) {
	const w, h = 60, 60
	mask := make([]byte, w*h)
	labels := make([]int, w*h)
	const id = 1
	for y := 10; y < 50; y++ {
		for x := 10; x < 50; x++ {
			mask[y*w+x] = 1
			labels[y*w+x] = id
		}
	}
	r := seg.Region{ID: id, MinX: 10, MinY: 10, MaxX: 49, MaxY: 49, Area: 40 * 40}
	clips := trySplitRegionClips(w, h, mask, labels, r, 100)
	if clips != nil {
		t.Fatalf("solid rectangle: want nil clips, got %v", clips)
	}
}
