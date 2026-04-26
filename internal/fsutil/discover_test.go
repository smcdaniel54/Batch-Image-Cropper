package fsutil

import (
	"os"
	"path/filepath"
	"testing"
)

func TestIsImageName(t *testing.T) {
	if !IsImageName("a.JPG") || !IsImageName("b.png") || !IsImageName("c.jpeg") {
		t.Fatal("expected image extensions to match")
	}
	if IsImageName("x.txt") || IsImageName("x.gif") {
		t.Fatal("expected non-matches")
	}
}

func TestListImageFiles(t *testing.T) {
	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "a.jpg"), []byte{1}, 0o644)
	_ = os.WriteFile(filepath.Join(dir, "b.PnG"), []byte{1}, 0o644)
	_ = os.MkdirAll(filepath.Join(dir, "sub"), 0o755)
	_ = os.WriteFile(filepath.Join(dir, "sub", "c.jpeg"), []byte{1}, 0o644)
	_ = os.WriteFile(filepath.Join(dir, "readme.txt"), []byte{1}, 0o644)
	list, err := ListImageFiles(dir)
	if err != nil {
		t.Fatal(err)
	}
	// Top-level only: a.jpg, b.PnG — not sub/c.jpeg
	if len(list) != 2 {
		t.Fatalf("want 2 top-level images, got %d", len(list))
	}
}

func TestListImageFilesWithWarnings(t *testing.T) {
	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "ok.jpg"), []byte{1}, 0o644)
	_ = os.WriteFile(filepath.Join(dir, "skip.txt"), []byte{1}, 0o644)
	var warns int
	warnf := func(msg string) { warns++ }
	list, err := ListImageFilesWithWarnings(dir, warnf)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 {
		t.Fatalf("want 1 image, got %d", len(list))
	}
	if warns != 1 {
		t.Fatalf("want 1 warning, got %d", warns)
	}
}

func TestListImageFiles_TopLevelOnly_SkipsSubdirImages(t *testing.T) {
	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "scan1.jpg"), []byte{1}, 0o644)
	_ = os.MkdirAll(filepath.Join(dir, "subdir"), 0o755)
	_ = os.WriteFile(filepath.Join(dir, "subdir", "scan2.jpg"), []byte{2}, 0o644)
	list, err := ListImageFilesWithWarnings(dir, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 {
		t.Fatalf("want 1 file, got %d: %v", len(list), list)
	}
	if filepath.Base(list[0]) != "scan1.jpg" {
		t.Fatalf("got %q want scan1.jpg", list[0])
	}
}
