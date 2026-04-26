package main

import (
	"encoding/json"
	"image"
	"image/color"
	"image/jpeg"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"photo-cropper/internal/manifest"
)

// TestEmptyInputDirDoesNotCreateOutput runs the app against an empty input directory and
// asserts the default output path is not created. Uses `go run` from the module root.
func TestEmptyInputDirDoesNotCreateOutput(t *testing.T) {
	if testing.Short() {
		t.Skip("subprocess: go run")
	}
	modRoot := moduleDir(t)
	work := t.TempDir()
	emptyIn := filepath.Join(work, "input")
	if err := os.MkdirAll(emptyIn, 0o755); err != nil {
		t.Fatal(err)
	}
	defaultOut := filepath.Join(work, "output")
	cmd := exec.Command("go", "run", ".", "-input-dir", emptyIn, "-out-dir", defaultOut)
	cmd.Dir = modRoot
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("expected error when no image files, output:\n%s", string(out))
	}
	t.Logf("go run (expected failure) output: %s", out)
	if _, err := os.Stat(defaultOut); !os.IsNotExist(err) {
		t.Fatalf("output directory must not be created, stat err=%v", err)
	}
}

func moduleDir(t *testing.T) string {
	t.Helper()
	_, f, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller")
	}
	return filepath.Dir(f)
}

// TestOutputExcludesOriginalScanFile runs a real extraction and ensures out-dir
// contains only derived crops and the QA overlay (never the unmodified input basename).
// The source is moved to processed/ only after successful writes (enforced in main; verified here
// by presence of all outputs and the moved file).
func TestOutputExcludesOriginalScanFile(t *testing.T) {
	if testing.Short() {
		t.Skip("subprocess: go run")
	}
	modRoot := moduleDir(t)
	work := t.TempDir()
	inPath := filepath.Join(work, "scan.jpg")
	outDir := filepath.Join(work, "out")
	writeSyntheticScanJPEG(t, inPath)
	cmd := exec.Command("go", "run", ".", "-input", inPath, "-out-dir", outDir)
	cmd.Dir = modRoot
	logOut, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("go run: %v\n%s", err, logOut)
	}
	derived := filepath.Join(outDir, "scan_001.jpg")
	if _, err := os.Stat(derived); err != nil {
		t.Fatalf("expected at least %s: %v\ngo run log:\n%s", derived, err, logOut)
	}
	qa := filepath.Join(outDir, "scan_000_qa.jpg")
	if _, err := os.Stat(qa); err != nil {
		t.Fatalf("expected QA overlay %s: %v\ngo run log:\n%s", qa, err, logOut)
	}
	originalName := filepath.Join(outDir, "scan.jpg")
	if _, err := os.Stat(originalName); !os.IsNotExist(err) {
		t.Fatalf("out-dir must not contain original filename scan.jpg, stat err=%v", err)
	}
	if !strings.Contains(string(logOut), "QA images written: 1") {
		t.Fatalf("batch summary should list QA count; log:\n%s", logOut)
	}
	done := filepath.Join(work, "processed", "scan.jpg")
	if _, err := os.Stat(done); err != nil {
		t.Fatalf("source should move to processed/ after success: %v", err)
	}
	if _, err := os.Stat(inPath); !os.IsNotExist(err) {
		t.Fatalf("original input path should be moved away: %s", inPath)
	}
}

// TestManifestQaImageOnAllEntries uses a synthetic scan with two separable dark regions
// (default min-area) and checks manifest.json: every entry lists the same qa_image.
func TestManifestQaImageOnAllEntries(t *testing.T) {
	if testing.Short() {
		t.Skip("subprocess: go run")
	}
	modRoot := moduleDir(t)
	work := t.TempDir()
	inPath := filepath.Join(work, "twoscan.jpg")
	outDir := filepath.Join(work, "out")
	writeSyntheticScanTwoCropsJPEG(t, inPath)
	cmd := exec.Command("go", "run", ".", "-input", inPath, "-out-dir", outDir)
	cmd.Dir = modRoot
	logOut, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("go run: %v\n%s", err, logOut)
	}
	manifestPath := filepath.Join(outDir, "manifest.json")
	raw, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatalf("read manifest: %v\ngo run log:\n%s", err, logOut)
	}
	var m manifest.File
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(m.Entries) < 2 {
		t.Fatalf("expected at least 2 entries for two-crop scan, got %d: %s", len(m.Entries), string(logOut))
	}
	const want = "twoscan_000_qa.jpg"
	for i, e := range m.Entries {
		if e.QaImage != want {
			t.Fatalf("entries[%d] output %q: qa_image=%q want %q for every entry from same source", i, e.Output, e.QaImage, want)
		}
	}
}

func writeSyntheticScanJPEG(t *testing.T, path string) {
	t.Helper()
	// ~220x220 near-white with a dark card region so one component meets default min-area.
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
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if err := jpeg.Encode(f, img, &jpeg.Options{Quality: 90}); err != nil {
		t.Fatal(err)
	}
}

// writeSyntheticScanTwoCropsJPEG is a near-white field with two 150×150 dark squares (> default min-area each),
// well separated so labeling yields two components.
func writeSyntheticScanTwoCropsJPEG(t *testing.T, path string) {
	t.Helper()
	const w, h = 520, 240
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	white := color.RGBA{R: 255, G: 255, B: 255, A: 255}
	dark := color.RGBA{R: 10, G: 10, B: 10, A: 255}
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.SetRGBA(x, y, white)
		}
	}
	fillRect := func(x0, y0, x1, y1 int) {
		for y := y0; y < y1; y++ {
			for x := x0; x < x1; x++ {
				img.SetRGBA(x, y, dark)
			}
		}
	}
	// 150*150 = 22500 > 20000
	fillRect(20, 45, 170, 195)
	fillRect(350, 45, 500, 195)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if err := jpeg.Encode(f, img, &jpeg.Options{Quality: 90}); err != nil {
		t.Fatal(err)
	}
}
