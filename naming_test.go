package main

import (
	"image"
	"image/color"
	"os"
	"path/filepath"
	"sort"
	"testing"
)

func TestCropLooksFullPageAgainstSource(t *testing.T) {
	src := image.NewRGBA(image.Rect(0, 0, 100, 100))
	cropOK := image.NewRGBA(image.Rect(0, 0, 89, 95))
	if cropLooksFullPageAgainstSource(cropOK, src) {
		t.Fatal("89×95 vs 100×100 should not reject (width < 90%)")
	}
	cropReject := image.NewRGBA(image.Rect(0, 0, 90, 90))
	if !cropLooksFullPageAgainstSource(cropReject, src) {
		t.Fatal("90×90 vs 100×100 should reject")
	}
}

func TestQANameSortsBeforeCrops(t *testing.T) {
	qa, err := outputQAName("scan.jpg")
	if err != nil {
		t.Fatal(err)
	}
	names := []string{
		outputPhotoNameString(t, "scan.jpg", 2),
		qa,
		outputPhotoNameString(t, "scan.jpg", 1),
	}
	sort.Strings(names)
	if names[0] != qa {
		t.Fatalf("QA should sort first: got %v", names)
	}
}

func outputPhotoNameString(t *testing.T, path string, n int) string {
	t.Helper()
	s, err := outputPhotoName(path, n)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func TestOutputNaming(t *testing.T) {
	got, err := outputPhotoName(`C:\data\My Scan\foo.jpeg`, 1)
	if err != nil {
		t.Fatal(err)
	}
	if got != "foo_001.jpg" {
		t.Fatalf("got %q", got)
	}
	got, err = outputPhotoName("scan.jpg", 12)
	if err != nil {
		t.Fatal(err)
	}
	if got != "scan_012.jpg" {
		t.Fatalf("got %q", got)
	}
}

func TestOutputNameNeverEqualsInputBase(t *testing.T) {
	for _, path := range []string{`C:\a\scan.jpg`, "scan.png", "x.JPEG", "p.png"} {
		for n := 1; n <= 3; n++ {
			out, err := outputPhotoName(path, n)
			if err != nil {
				t.Fatalf("%q n=%d: %v", path, n, err)
			}
			if out == filepath.Base(path) {
				t.Fatalf("output %q must not equal input base of %q", out, path)
			}
			if !isDerivedOutputPhotoName(out) {
				t.Fatalf("not derived: %q", out)
			}
		}
	}
}

func TestQAOutputName(t *testing.T) {
	got, err := outputQAName("scan.jpg")
	if err != nil {
		t.Fatal(err)
	}
	if got != "scan_000_qa.jpg" {
		t.Fatalf("got %q", got)
	}
	if !isQAOutputName(got) {
		t.Fatalf("isQA: %q", got)
	}
}

func TestValidateAllowlistedOutputJPEGRejectsOriginalScanName(t *testing.T) {
	reject := []string{
		"scan.jpg",
		"Photo_2026.jpg",
		"manifest.jpg",
		"scan_000.jpg",
		"scan_000_qa.png",
		"x.png",
		"",
		"only.jpg",
	}
	for _, b := range reject {
		if err := validateAllowlistedOutputJPEG(b); err == nil {
			t.Fatalf("expected error for disallowed name %q", b)
		}
	}
	for _, b := range []string{"scan_000_qa.jpg", "scan_001.jpg", "my-scan_012.jpg", "a_999.jpg"} {
		if err := validateAllowlistedOutputJPEG(b); err != nil {
			t.Fatalf("expected ok for %q: %v", b, err)
		}
	}
}

// TestEncodeJPEGFileDoesNotCreateDisallowedScanBasename ensures the output JPEG gate runs before os.Create.
func TestEncodeJPEGFileDoesNotCreateDisallowedScanBasename(t *testing.T) {
	dir := t.TempDir()
	bad := filepath.Join(dir, "scan.jpg")
	img := image.NewRGBA(image.Rect(0, 0, 2, 2))
	img.SetRGBA(0, 0, color.RGBA{1, 2, 3, 255})
	if err := encodeJPEGFile(bad, img, 90); err == nil {
		t.Fatal("expected encodeJPEGFile to reject scan.jpg as output basename")
	}
	if _, err := os.Stat(bad); !os.IsNotExist(err) {
		t.Fatalf("disallowed path must not be created: %v", err)
	}
}
