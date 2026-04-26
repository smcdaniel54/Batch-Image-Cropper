package moveprocessed

import (
	"os"
	"path/filepath"
	"testing"
)

func TestMoveToProcessed_success(t *testing.T) {
	root := t.TempDir()
	indir := filepath.Join(root, "in")
	_ = os.MkdirAll(indir, 0o755)
	src := filepath.Join(indir, "scan.jpg")
	if err := os.WriteFile(src, []byte{1, 2, 3}, 0o644); err != nil {
		t.Fatal(err)
	}
	dest, err := MoveToProcessed(src, indir)
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(indir, "processed", "scan.jpg")
	if dest != want {
		t.Fatalf("dest %q want %q", dest, want)
	}
	if _, e := os.Stat(src); !os.IsNotExist(e) {
		t.Fatalf("source should be gone: %v", e)
	}
	if b, _ := os.ReadFile(dest); len(b) != 3 {
		t.Fatal("data lost")
	}
}

func TestNextDestPath_collision(t *testing.T) {
	processed := t.TempDir()
	_ = os.WriteFile(filepath.Join(processed, "a.jpg"), []byte{1}, 0o644)
	p, err := NextDestPath(processed, "a.jpg")
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(p) != "a_2.jpg" {
		t.Fatalf("got %q want a_2.jpg", p)
	}
	_ = os.WriteFile(p, []byte{2}, 0o644)
	p2, err := NextDestPath(processed, "a.jpg")
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(p2) != "a_3.jpg" {
		t.Fatalf("got %q want a_3.jpg", p2)
	}
}

func TestMoveAfterExtraction(t *testing.T) {
	if MoveAfterExtraction(0) {
		t.Fatal("zero photos must not move")
	}
	if !MoveAfterExtraction(1) {
		t.Fatal("one or more photos should allow move")
	}
}

func TestNextDestPath_firstFree(t *testing.T) {
	processed := t.TempDir()
	p, err := NextDestPath(processed, "x.png")
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(p) != "x.png" {
		t.Fatalf("got %q", p)
	}
}
