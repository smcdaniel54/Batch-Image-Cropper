package qualityreport

import "testing"

func TestClassifyStatus_PASS(t *testing.T) {
	if g := ClassifyStatus([]float64{0.65, 0.88, 0.7}); g != "PASS" {
		t.Fatalf("got %q want PASS", g)
	}
	if g := ClassifyStatus([]float64{0.7}); g != "PASS" {
		t.Fatalf("single high: got %q want PASS", g)
	}
}

func TestClassifyStatus_REVIEW(t *testing.T) {
	if g := ClassifyStatus([]float64{0.7, 0.5}); g != "REVIEW" {
		t.Fatalf("got %q want REVIEW", g)
	}
	if g := ClassifyStatus([]float64{0.64}); g != "REVIEW" {
		t.Fatalf("0.64: got %q want REVIEW", g)
	}
	if g := ClassifyStatus([]float64{0.22, 0.7}); g != "REVIEW" {
		t.Fatalf("0.22 with 0.7: got %q want REVIEW", g)
	}
}

func TestClassifyStatus_FAIL(t *testing.T) {
	if g := ClassifyStatus([]float64{0.2}); g != "FAIL" {
		t.Fatalf("0.2: got %q want FAIL", g)
	}
	if g := ClassifyStatus([]float64{0.15, 0.9}); g != "FAIL" {
		t.Fatalf("mixed: got %q want FAIL", g)
	}
}

func TestClassifyStatus_Empty(t *testing.T) {
	if g := ClassifyStatus([]float64{}); g != "PASS" {
		t.Fatalf("empty: got %q want PASS", g)
	}
}

func TestClassifyStatus_REVIEW_justAboveFailThreshold(t *testing.T) {
	// 0.21: not FAIL (not <= 0.2), not all PASS
	if g := ClassifyStatus([]float64{0.21}); g != "REVIEW" {
		t.Fatalf("got %q want REVIEW", g)
	}
}
