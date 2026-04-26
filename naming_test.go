package main

import (
	"path/filepath"
	"sort"
	"testing"
)

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
