package geom

import (
	"math"
	"testing"
)

func TestValidateQuadNearDegenerate(t *testing.T) {
	v := DefaultQuadValidation()
	// Tiny area
	tiny := [4]Point{{0, 0}, {2, 0}, {2, 2}, {0, 2}}
	if err := ValidateQuadOrdered(tiny, v); err == nil {
		t.Fatal("expected area rejection")
	}
	// One vertical edge length 2 < MinEdgeLength 3
	short := [4]Point{{0, 0}, {100, 0}, {100, 100}, {0, 2}}
	if err := ValidateQuadOrdered(short, v); err == nil {
		t.Fatal("expected short edge rejection")
	}
}

func TestValidateQuadExtremeAspect(t *testing.T) {
	v := DefaultQuadValidation()
	// Very wide thin strip
	wide := [4]Point{{0, 0}, {5000, 0}, {5000, 5}, {0, 5}}
	if err := ValidateQuadOrdered(wide, v); err == nil {
		t.Fatal("expected aspect rejection")
	}
}

func TestValidateQuadSelfIntersectingBowtie(t *testing.T) {
	v := DefaultQuadValidation()
	// Deliberate bowtie in declared TL,TR,BR,BL order (invalid for a photo)
	bow := [4]Point{
		{X: 0, Y: 0},
		{X: 100, Y: 100},
		{X: 0, Y: 100},
		{X: 100, Y: 0},
	}
	if !QuadSelfIntersecting(bow) {
		t.Fatal("expected bowtie self-intersection")
	}
	if err := ValidateQuadOrdered(bow, v); err == nil {
		t.Fatal("expected validation failure for self-intersecting quad")
	}
}

func TestValidateQuadNormalPasses(t *testing.T) {
	v := DefaultQuadValidation()
	ok := [4]Point{{0, 0}, {200, 0}, {200, 150}, {0, 150}}
	if err := ValidateQuadOrdered(ok, v); err != nil {
		t.Fatal(err)
	}
}

func TestSignedAreaReferenceDst(t *testing.T) {
	dst := [4]Point{{0, 0}, {1, 0}, {1, 1}, {0, 1}}
	s := signedQuadArea2(dst)
	// signedQuadArea2 returns the shoelace sum (twice signed area); unit square → +2
	if s <= 0 || math.Abs(s-2) > 1e-9 {
		t.Fatalf("reference dst shoelace sum want +2, got %v", s)
	}
}
